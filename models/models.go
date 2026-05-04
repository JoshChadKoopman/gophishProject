package models

import (
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"os"
	"time"

	"bitbucket.org/liamstask/goose/lib/goose"

	mysql "github.com/go-sql-driver/mysql"
	"github.com/gophish/gophish/auth"
	"github.com/gophish/gophish/config"

	log "github.com/gophish/gophish/logger"
	"github.com/jinzhu/gorm"
	_ "github.com/lib/pq"              // Blank import needed to register postgres driver
	_ "github.com/mattn/go-sqlite3"    // Blank import needed to import sqlite3
)

var db *gorm.DB
var conf *config.Config

// GetDB returns the global database handle. Use sparingly — prefer
// package-level model functions for data access.
func GetDB() *gorm.DB {
	return db
}

const MaxDatabaseConnectionAttempts int = 10

// DefaultAdminUsername is the default username for the administrative user
const DefaultAdminUsername = "admin"

// InitialAdminPassword is the environment variable that specifies which
// password to use for the initial root login instead of generating one
// randomly
const InitialAdminPassword = "GOPHISH_INITIAL_ADMIN_PASSWORD"

// InitialAdminApiToken is the environment variable that specifies the
// API token to seed the initial root login instead of generating one
// randomly
const InitialAdminApiToken = "GOPHISH_INITIAL_ADMIN_API_TOKEN"

const (
	CampaignInProgress  string = "In progress"
	CampaignQueued      string = "Queued"
	CampaignCreated     string = "Created"
	CampaignEmailsSent  string = "Emails Sent"
	CampaignComplete    string = "Completed"
	EventSent           string = "Email Sent"
	EventSendingError   string = "Error Sending Email"
	EventOpened         string = "Email Opened"
	EventClicked        string = "Clicked Link"
	EventDataSubmit     string = "Submitted Data"
	EventReported       string = "Email Reported"
	EventFeedbackViewed string = "Feedback Viewed"
	EventProxyRequest   string = "Proxied request"
	StatusSuccess       string = "Success"
	StatusQueued        string = "Queued"
	StatusSending       string = "Sending"
	StatusUnknown       string = "Unknown"
	StatusScheduled     string = "Scheduled"
	StatusRetry         string = "Retrying"
	Error               string = "Error"
)

// Flash is used to hold flash information for use in templates.
type Flash struct {
	Type    string
	Message string
}

// Response contains the attributes found in an API response
type Response struct {
	Message string      `json:"message"`
	Success bool        `json:"success"`
	Data    interface{} `json:"data"`
}

// Copy of auth.GenerateSecureKey to prevent cyclic import with auth library.
// Panics if the system CSPRNG is unavailable.
func generateSecureKey() string {
	k := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, k); err != nil {
		panic("models: crypto/rand is unavailable: " + err.Error())
	}
	return fmt.Sprintf("%x", k)
}

func chooseDBDriver(name, openStr string) goose.DBDriver {
	d := goose.DBDriver{Name: name, OpenStr: openStr}

	switch name {
	case "mysql":
		d.Import = "github.com/go-sql-driver/mysql"
		d.Dialect = &goose.MySqlDialect{}
	case "postgres":
		d.Import = "github.com/lib/pq"
		d.Dialect = &goose.PostgresDialect{}
	// Default database is sqlite3
	default:
		d.Import = "github.com/mattn/go-sqlite3"
		d.Dialect = &goose.Sqlite3Dialect{}
	}

	return d
}

func createTemporaryPassword(u *User) error {
	var temporaryPassword string
	if envPassword := os.Getenv(InitialAdminPassword); envPassword != "" {
		temporaryPassword = envPassword
	} else {
		// This will result in a 16 character password which could be viewed as an
		// inconvenience, but it should be ok for now.
		temporaryPassword = auth.GenerateSecureKey(auth.MinPasswordLength)
	}
	hash, err := auth.GeneratePasswordHash(temporaryPassword)
	if err != nil {
		return err
	}
	u.Hash = hash
	// Anytime a temporary password is created, we will force the user
	// to change their password
	u.PasswordChangeRequired = true
	err = db.Save(u).Error
	if err != nil {
		return err
	}
	log.Infof("[SETUP] Please login with the username admin and the password: %s", temporaryPassword)
	return nil
}

// Setup initializes the database and runs any needed migrations.
//
// First, it establishes a connection to the database, then runs any migrations
// newer than the version the database is on.
//
// Once the database is up-to-date, we create an admin user (if needed) that
// has a randomly generated API key and password.
func Setup(c *config.Config) error {
	conf = c
	migrateConf := &goose.DBConf{
		MigrationsDir: conf.MigrationsPath,
		Env:           "production",
		Driver:        chooseDBDriver(conf.DBName, conf.DBPath),
	}
	latest, err := goose.GetMostRecentDBVersion(migrateConf.MigrationsDir)
	if err != nil {
		log.Error(err)
		return err
	}
	if err := registerTLSIfNeeded(); err != nil {
		return err
	}
	if err := openDatabaseConnection(); err != nil {
		return err
	}
	if err := goose.RunMigrationsOnDb(migrateConf, migrateConf.MigrationsDir, latest, db.DB()); err != nil {
		log.Error(err)
		return err
	}
	// Load template library from JSON files (falls back to built-in templates)
	LoadTemplateLibrary()
	// Seed built-in templates into the DB-backed library
	if err := SeedBuiltinTemplates(); err != nil {
		log.Warn("template library DB seed: ", err)
	}
	return ensureAdminUser()
}

// registerTLSIfNeeded registers TLS certificates for encrypted DB connections.
func registerTLSIfNeeded() error {
	if conf.DBSSLCaPath == "" || conf.DBName != "mysql" {
		return nil
	}
	rootCertPool := x509.NewCertPool()
	pem, err := os.ReadFile(conf.DBSSLCaPath)
	if err != nil {
		log.Error(err)
		return err
	}
	if ok := rootCertPool.AppendCertsFromPEM(pem); !ok {
		log.Error("Failed to append PEM.")
		return err
	}
	mysql.RegisterTLSConfig("ssl_ca", &tls.Config{
		RootCAs: rootCertPool,
	})
	return nil
}

// openDatabaseConnection opens the gorm database connection with retries.
func openDatabaseConnection() error {
	var err error
	for i := 0; ; i++ {
		db, err = gorm.Open(conf.DBName, conf.DBPath)
		if err == nil {
			break
		}
		if i >= MaxDatabaseConnectionAttempts {
			log.Error(err)
			return err
		}
		log.Warn("waiting for database to be up...")
		time.Sleep(5 * time.Second)
	}
	db.LogMode(false)
	db.SetLogger(log.Logger)
	db.DB().SetMaxOpenConns(1)
	return nil
}

// ensureAdminUser creates an admin user if none exists and handles temporary passwords.
func ensureAdminUser() error {
	var userCount int64
	var adminUser User
	db.Model(&User{}).Count(&userCount)
	adminRole, err := GetRoleBySlug(RoleAdmin)
	if err != nil {
		log.Error(err)
		return err
	}
	if userCount == 0 {
		adminUser = User{
			Username:               DefaultAdminUsername,
			OrgId:                  1,
			Role:                   adminRole,
			RoleID:                 adminRole.ID,
			PasswordChangeRequired: true,
		}
		if envToken := os.Getenv(InitialAdminApiToken); envToken != "" {
			adminUser.ApiKey = envToken
		} else {
			adminUser.ApiKey = auth.GenerateSecureKey(auth.APIKeyLength)
		}
		if err := db.Save(&adminUser).Error; err != nil {
			log.Error(err)
			return err
		}
	}
	if adminUser.Username == "" {
		adminUser, err = GetUserByUsername(DefaultAdminUsername)
		if err != nil {
			log.Error(err)
			return err
		}
	}
	if adminUser.PasswordChangeRequired {
		if err := createTemporaryPassword(&adminUser); err != nil {
			log.Error(err)
			return err
		}
	}
	return nil
}
