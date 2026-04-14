package models

import (
	"errors"
	"time"

	log "github.com/gophish/gophish/logger"
)

// ErrModifyingOnlyAdmin occurs when there is an attempt to modify the only
// user account with the Admin role in such a way that there will be no user
// accounts left in Gophish with that role.
var ErrModifyingOnlyAdmin = errors.New("Cannot remove the only administrator")

// User represents the user model for gophish.
type User struct {
	Id                       int64        `json:"id"`
	Username                 string       `json:"username" sql:"not null;unique"`
	Hash                     string       `json:"-"`
	ApiKey                   string       `json:"api_key" sql:"not null;unique"`
	FirstName                string       `json:"first_name"`
	LastName                 string       `json:"last_name"`
	Email                    string       `json:"email"`
	Position                 string       `json:"position"`
	Role                     Role         `json:"role" gorm:"association_autoupdate:false;association_autocreate:false"`
	RoleID                   int64        `json:"-"`
	OrgId                    int64        `json:"org_id" gorm:"column:org_id"`
	Org                      Organization `json:"org,omitempty" gorm:"association_autoupdate:false;association_autocreate:false"`
	PasswordChangeRequired   bool         `json:"password_change_required"`
	AccountLocked            bool         `json:"account_locked"`
	LastLogin                time.Time    `json:"last_login"`
	Department               string       `json:"department" gorm:"column:department"`
	JobTitle                 string       `json:"job_title" gorm:"column:job_title"`
	FailedLogins             int          `json:"failed_logins" gorm:"column:failed_logins;default:0"`
	LastFailedLogin          time.Time    `json:"last_failed_login" gorm:"column:last_failed_login"`
	PreferredLanguage        string       `json:"preferred_language" gorm:"column:preferred_language;default:'en'"`
	TrainingDifficultyMode   string       `json:"training_difficulty_mode" gorm:"column:training_difficulty_mode;default:'adaptive'"`
	TrainingDifficultyManual int          `json:"training_difficulty_manual" gorm:"column:training_difficulty_manual;default:0"`
}

// GetUser returns the user that the given id corresponds to. If no user is found, an
// error is thrown.
func GetUser(id int64) (User, error) {
	u := User{}
	err := db.Preload("Role").Where("id=?", id).First(&u).Error
	return u, err
}

// GetUsers returns the users registered in Gophish
func GetUsers() ([]User, error) {
	us := []User{}
	err := db.Preload("Role").Find(&us).Error
	return us, err
}

// GetUserByAPIKey returns the user that the given API Key corresponds to. If no user is found, an
// error is thrown.
func GetUserByAPIKey(key string) (User, error) {
	u := User{}
	err := db.Preload("Role").Where("api_key = ?", key).First(&u).Error
	return u, err
}

// GetUserByUsername returns the user that the given username corresponds to. If no user is found, an
// error is thrown.
func GetUserByUsername(username string) (User, error) {
	u := User{}
	err := db.Preload("Role").Where("username = ?", username).First(&u).Error
	return u, err
}

// GetUserByEmail returns the user that the given email corresponds to. If no user is found, an
// error is thrown.
func GetUserByEmail(email string) (User, error) {
	u := User{}
	err := db.Preload("Role").Where("email = ?", email).First(&u).Error
	return u, err
}

// PutUser updates the given user
func PutUser(u *User) error {
	err := db.Save(u).Error
	return err
}

// EnsureEnoughAdmins ensures that there is more than one user account in
// Gophish with the Admin role. This function is meant to be called before
// modifying a user account with the Admin role in a non-revokable way.
func EnsureEnoughAdmins() error {
	role, err := GetRoleBySlug(RoleAdmin)
	if err != nil {
		return err
	}
	var adminCount int
	err = db.Model(&User{}).Where("role_id=?", role.ID).Count(&adminCount).Error
	if err != nil {
		return err
	}
	if adminCount == 1 {
		return ErrModifyingOnlyAdmin
	}
	return nil
}

// GetUsersByOrg returns the users belonging to the given organization.
func GetUsersByOrg(scope OrgScope) ([]User, error) {
	us := []User{}
	query := db.Preload("Role")
	query = scopeQuery(query, scope)
	err := query.Find(&us).Error
	return us, err
}

// DeleteUser deletes the given user. To ensure that there is always at least
// one user account with the Admin role, this function will refuse to delete
// the last Admin.
//
// In multi-tenant mode, org-level resources (campaigns, templates, groups,
// pages, sending profiles) are NOT cascade-deleted because they belong to the
// org rather than the individual user.
func DeleteUser(id int64) error {
	existing, err := GetUser(id)
	if err != nil {
		return err
	}
	// If the user is an admin, we need to verify that it's not the last one.
	if existing.Role.Slug == RoleAdmin {
		err = EnsureEnoughAdmins()
		if err != nil {
			return err
		}
	}
	// Delete the user record. Org-level resources (campaigns, groups,
	// templates, pages, smtp) persist as they belong to the organization.
	log.Infof("Deleting user account ID %d (%s)", id, existing.Username)
	err = db.Where("id=?", id).Delete(&User{}).Error
	return err
}

// RecordFailedLogin increments the failed login counter and records the
// timestamp. Called after a password validation failure.
func RecordFailedLogin(uid int64) error {
	return db.Model(&User{}).Where("id=?", uid).
		UpdateColumns(map[string]interface{}{
			"failed_logins":     db.Raw("failed_logins + 1"),
			"last_failed_login": time.Now().UTC(),
		}).Error
}

// ResetFailedLogins clears the failed login counter after a successful login.
func ResetFailedLogins(uid int64) error {
	return db.Model(&User{}).Where("id=?", uid).
		UpdateColumns(map[string]interface{}{
			"failed_logins":     0,
			"last_failed_login": time.Time{},
		}).Error
}

// IsLoginLockedOut returns true if the user has exceeded the maximum failed
// login attempts within the lockout duration window.
func IsLoginLockedOut(u *User) bool {
	if u.FailedLogins < 5 {
		return false
	}
	return time.Since(u.LastFailedLogin) < 15*time.Minute
}
