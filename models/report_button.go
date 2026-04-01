package models

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// ReportButtonConfig stores the per-org configuration for the email report
// button plugin (Outlook add-in / Gmail add-on).
type ReportButtonConfig struct {
	Id                 int64     `json:"id" gorm:"primary_key;auto_increment"`
	OrgId              int64     `json:"org_id"`
	PluginApiKey       string    `json:"plugin_api_key"`
	FeedbackSimulation string    `json:"feedback_simulation"`
	FeedbackReal       string    `json:"feedback_real"`
	Enabled            bool      `json:"enabled" gorm:"default:true"`
	CreatedDate        time.Time `json:"created_date"`
	ModifiedDate       time.Time `json:"modified_date"`
}

// TableName overrides the default table name.
func (ReportButtonConfig) TableName() string {
	return "report_button_configs"
}

// ReportedEmail represents an email reported via the report button plugin.
type ReportedEmail struct {
	Id             int64     `json:"id" gorm:"primary_key;auto_increment"`
	OrgId          int64     `json:"org_id"`
	ReporterEmail  string    `json:"reporter_email"`
	SenderEmail    string    `json:"sender_email"`
	Subject        string    `json:"subject"`
	HeadersHash    string    `json:"headers_hash"`
	IsSimulation   bool      `json:"is_simulation"`
	CampaignId     int64     `json:"campaign_id"`
	ResultId       int64     `json:"result_id"`
	Classification string    `json:"classification" gorm:"default:'pending'"`
	AdminNotes     string    `json:"admin_notes"`
	CreatedDate    time.Time `json:"created_date"`
}

// TableName overrides the default table name.
func (ReportedEmail) TableName() string {
	return "reported_emails"
}

// GeneratePluginAPIKey creates a random 32-byte hex API key for the report button plugin.
func GeneratePluginAPIKey() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// GetReportButtonConfig returns the report button config for an org, creating
// one with a fresh API key if none exists.
func GetReportButtonConfig(orgId int64) (ReportButtonConfig, error) {
	var config ReportButtonConfig
	err := db.Where("org_id = ?", orgId).First(&config).Error
	if err != nil {
		return config, err
	}
	return config, nil
}

// GetReportButtonConfigByAPIKey looks up a config by its plugin API key.
// Used by the plugin auth middleware.
func GetReportButtonConfigByAPIKey(apiKey string) (ReportButtonConfig, error) {
	var config ReportButtonConfig
	err := db.Where("plugin_api_key = ? AND enabled = ?", apiKey, true).First(&config).Error
	return config, err
}

// CreateReportButtonConfig creates or updates the report button config for an org.
func CreateReportButtonConfig(config *ReportButtonConfig) error {
	config.ModifiedDate = time.Now().UTC()
	if config.Id == 0 {
		config.CreatedDate = time.Now().UTC()
		if config.PluginApiKey == "" {
			key, err := GeneratePluginAPIKey()
			if err != nil {
				return err
			}
			config.PluginApiKey = key
		}
		return db.Create(config).Error
	}
	return db.Save(config).Error
}

// UpdateReportButtonConfig updates an existing config.
func UpdateReportButtonConfig(config *ReportButtonConfig) error {
	config.ModifiedDate = time.Now().UTC()
	return db.Save(config).Error
}

// RegeneratePluginAPIKey generates a new API key for the org's report button config.
func RegeneratePluginAPIKey(orgId int64) (ReportButtonConfig, error) {
	config, err := GetReportButtonConfig(orgId)
	if err != nil {
		return config, err
	}
	key, err := GeneratePluginAPIKey()
	if err != nil {
		return config, err
	}
	config.PluginApiKey = key
	config.ModifiedDate = time.Now().UTC()
	return config, db.Save(&config).Error
}

// CreateReportedEmail saves a new reported email record.
func CreateReportedEmail(re *ReportedEmail) error {
	re.CreatedDate = time.Now().UTC()
	return db.Create(re).Error
}

// GetReportedEmails returns all reported emails for an org, newest first.
func GetReportedEmails(orgId int64) ([]ReportedEmail, error) {
	var emails []ReportedEmail
	err := db.Where("org_id = ?", orgId).Order("created_date DESC").Find(&emails).Error
	return emails, err
}

// GetReportedEmail returns a single reported email by ID within an org scope.
func GetReportedEmail(id int64, orgId int64) (ReportedEmail, error) {
	var email ReportedEmail
	err := db.Where("id = ? AND org_id = ?", id, orgId).First(&email).Error
	return email, err
}

// ClassifyReportedEmail updates the classification and admin notes of a reported email.
func ClassifyReportedEmail(id int64, orgId int64, classification string, notes string) error {
	return db.Model(&ReportedEmail{}).
		Where("id = ? AND org_id = ?", id, orgId).
		Updates(map[string]interface{}{
			"classification": classification,
			"admin_notes":    notes,
		}).Error
}

// ClassifyEmailBySimulation checks if the reported email matches a campaign
// simulation by looking up the reporter's email in campaign results. Returns
// true if a matching simulation was found.
func ClassifyEmailBySimulation(re *ReportedEmail) bool {
	var result Result
	err := db.Where("email = ?", re.ReporterEmail).
		Order("send_date DESC").
		First(&result).Error
	if err != nil {
		return false
	}
	// Check if this result belongs to a campaign in the same org
	var campaign Campaign
	err = db.Where("id = ?", result.CampaignId).First(&campaign).Error
	if err != nil || campaign.UserId == 0 {
		return false
	}
	// Verify org match via the campaign owner
	var owner User
	err = db.Where("id = ?", campaign.UserId).First(&owner).Error
	if err != nil || owner.OrgId != re.OrgId {
		return false
	}
	re.IsSimulation = true
	re.CampaignId = result.CampaignId
	re.ResultId = result.Id
	re.Classification = "simulation"
	// Mark the result as reported
	result.Reported = true
	result.ModifiedDate = time.Now().UTC()
	db.Save(&result)
	return true
}
