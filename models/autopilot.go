package models

import (
	"encoding/json"
	"errors"
	"time"

	log "github.com/gophish/gophish/logger"
	"github.com/jinzhu/gorm"
)

// AutopilotConfig stores per-org autopilot settings.
type AutopilotConfig struct {
	Id               int64     `json:"id" gorm:"primary_key"`
	OrgId            int64     `json:"org_id" gorm:"unique_index"`
	Enabled          bool      `json:"enabled"`
	CadenceDays      int       `json:"cadence_days"`                                    // Days between simulations per user (default 15)
	ActiveHoursStart int       `json:"active_hours_start"`                              // 0-23, hour to start sending
	ActiveHoursEnd   int       `json:"active_hours_end"`                                // 0-23, hour to stop sending
	Timezone         string    `json:"timezone"`                                        // IANA timezone (e.g. "Europe/Amsterdam")
	TargetGroupIds   string    `json:"target_group_ids" gorm:"column:target_group_ids"` // JSON array of group IDs
	SendingProfileId int64     `json:"sending_profile_id"`
	LandingPageId    int64     `json:"landing_page_id"`
	PhishURL         string    `json:"phish_url"` // Base phishing URL
	LastRun          time.Time `json:"last_run"`
	NextRun          time.Time `json:"next_run"`
	CreatedDate      time.Time `json:"created_date"`
	ModifiedDate     time.Time `json:"modified_date"`
}

// AutopilotSchedule records each scheduled autopilot send.
type AutopilotSchedule struct {
	Id              int64     `json:"id" gorm:"primary_key"`
	OrgId           int64     `json:"org_id"`
	UserEmail       string    `json:"user_email"`
	CampaignId      int64     `json:"campaign_id"`
	DifficultyLevel int       `json:"difficulty_level"`
	VariantId       string    `json:"variant_id" gorm:"column:variant_id;default:''"` // A/B test variant: "A" or "B"
	ScheduledDate   time.Time `json:"scheduled_date"`
	Sent            bool      `json:"sent"`
	CreatedDate     time.Time `json:"created_date"`
}

// AutopilotBlackoutDate represents a date when autopilot should not send.
type AutopilotBlackoutDate struct {
	Id          int64     `json:"id" gorm:"primary_key"`
	OrgId       int64     `json:"org_id"`
	Date        string    `json:"date"` // YYYY-MM-DD format
	Reason      string    `json:"reason"`
	CreatedDate time.Time `json:"created_date"`
}

var ErrAutopilotNotConfigured = errors.New("Autopilot is not configured for this organization")

// queryWhereOrgID is the shared WHERE clause fragment for org_id lookups.
const queryWhereOrgID = "org_id = ?"

// GetAutopilotConfig returns the autopilot config for the given org.
func GetAutopilotConfig(orgId int64) (AutopilotConfig, error) {
	ac := AutopilotConfig{}
	err := db.Where(queryWhereOrgID, orgId).First(&ac).Error
	if err != nil {
		return ac, err
	}
	return ac, nil
}

// SaveAutopilotConfig creates or updates the autopilot config for an org.
func SaveAutopilotConfig(ac *AutopilotConfig) error {
	if ac.CadenceDays <= 0 {
		ac.CadenceDays = 15
	}
	if ac.ActiveHoursStart < 0 || ac.ActiveHoursStart > 23 {
		ac.ActiveHoursStart = 9
	}
	if ac.ActiveHoursEnd < 0 || ac.ActiveHoursEnd > 23 {
		ac.ActiveHoursEnd = 17
	}
	if ac.Timezone == "" {
		ac.Timezone = "UTC"
	}
	ac.ModifiedDate = time.Now().UTC()

	existing := AutopilotConfig{}
	err := db.Where(queryWhereOrgID, ac.OrgId).First(&existing).Error
	if err == gorm.ErrRecordNotFound {
		ac.CreatedDate = time.Now().UTC()
		return db.Save(ac).Error
	}
	if err != nil {
		return err
	}
	ac.Id = existing.Id
	ac.CreatedDate = existing.CreatedDate
	return db.Save(ac).Error
}

// EnableAutopilot enables autopilot for an org and sets the next run.
func EnableAutopilot(orgId int64) error {
	ac, err := GetAutopilotConfig(orgId)
	if err != nil {
		return ErrAutopilotNotConfigured
	}
	ac.Enabled = true
	ac.NextRun = calculateNextRun(ac)
	return db.Save(&ac).Error
}

// DisableAutopilot disables autopilot for an org.
func DisableAutopilot(orgId int64) error {
	return db.Model(&AutopilotConfig{}).Where(queryWhereOrgID, orgId).
		Updates(map[string]interface{}{"enabled": false}).Error
}

// GetGroupIds parses the JSON target group IDs from the config.
func (ac *AutopilotConfig) GetGroupIds() []int64 {
	var ids []int64
	if ac.TargetGroupIds == "" || ac.TargetGroupIds == "null" {
		return ids
	}
	if err := json.Unmarshal([]byte(ac.TargetGroupIds), &ids); err != nil {
		log.Errorf("AutopilotConfig.GetGroupIds: invalid JSON %q: %v", ac.TargetGroupIds, err)
	}
	return ids
}

// SetGroupIds serializes group IDs to JSON for storage.
func (ac *AutopilotConfig) SetGroupIds(ids []int64) {
	b, _ := json.Marshal(ids)
	ac.TargetGroupIds = string(b)
}

// GetEnabledAutopilots returns all enabled autopilot configs where next_run <= now.
func GetEnabledAutopilots(t time.Time) ([]AutopilotConfig, error) {
	acs := []AutopilotConfig{}
	err := db.Where("enabled = ? AND next_run <= ?", true, t).Find(&acs).Error
	if err != nil {
		log.Error(err)
	}
	return acs, err
}

// UpdateAutopilotRun updates last_run and next_run after an autopilot cycle.
func UpdateAutopilotRun(ac *AutopilotConfig) error {
	ac.LastRun = time.Now().UTC()
	ac.NextRun = calculateNextRun(*ac)
	return db.Model(ac).Updates(map[string]interface{}{
		"last_run": ac.LastRun,
		"next_run": ac.NextRun,
	}).Error
}

// calculateNextRun determines the next run time based on cadence and active hours.
func calculateNextRun(ac AutopilotConfig) time.Time {
	loc, err := time.LoadLocation(ac.Timezone)
	if err != nil {
		loc = time.UTC
	}
	// Next run is tomorrow at the start of active hours
	now := time.Now().In(loc)
	next := time.Date(now.Year(), now.Month(), now.Day()+1, ac.ActiveHoursStart, 0, 0, 0, loc)
	return next.UTC()
}

// GetAutopilotSchedule returns upcoming autopilot schedule entries for an org.
func GetAutopilotSchedule(orgId int64, limit int) ([]AutopilotSchedule, error) {
	entries := []AutopilotSchedule{}
	if limit <= 0 {
		limit = 50
	}
	err := db.Where(queryWhereOrgID, orgId).
		Order("scheduled_date desc").
		Limit(limit).
		Find(&entries).Error
	return entries, err
}

// CreateAutopilotSchedule inserts a schedule entry.
func CreateAutopilotSchedule(s *AutopilotSchedule) error {
	s.CreatedDate = time.Now().UTC()
	return db.Save(s).Error
}

// GetAutopilotBlackoutDates returns blackout dates for an org.
func GetAutopilotBlackoutDates(orgId int64) ([]AutopilotBlackoutDate, error) {
	dates := []AutopilotBlackoutDate{}
	err := db.Where(queryWhereOrgID, orgId).Order("date asc").Find(&dates).Error
	return dates, err
}

// CreateAutopilotBlackoutDate adds a blackout date.
func CreateAutopilotBlackoutDate(d *AutopilotBlackoutDate) error {
	d.CreatedDate = time.Now().UTC()
	return db.Save(d).Error
}

// DeleteAutopilotBlackoutDate removes a blackout date.
func DeleteAutopilotBlackoutDate(id int64, orgId int64) error {
	return db.Where("id = ? AND org_id = ?", id, orgId).Delete(&AutopilotBlackoutDate{}).Error
}

// IsBlackoutDate checks if the given date is a blackout date for the org.
func IsBlackoutDate(orgId int64, date time.Time) bool {
	dateStr := date.Format("2006-01-02")
	var count int
	db.Model(&AutopilotBlackoutDate{}).Where("org_id = ? AND date = ?", orgId, dateStr).Count(&count)
	return count > 0
}

// GetUsersLastSentDate returns a map of user emails to their last autopilot send date.
// Used to determine which users are due for their next simulation.
func GetUsersLastSentDate(orgId int64) (map[string]time.Time, error) {
	result := make(map[string]time.Time)
	rows, err := db.Table("autopilot_schedules").
		Select("user_email, MAX(scheduled_date) as last_date").
		Where("org_id = ? AND sent = ?", orgId, true).
		Group("user_email").
		Rows()
	if err != nil {
		return result, err
	}
	defer rows.Close()
	for rows.Next() {
		var email string
		var lastDateStr string
		if err := rows.Scan(&email, &lastDateStr); err == nil {
			// Parse the date string; try common formats
			for _, layout := range []string{
				time.RFC3339Nano,
				time.RFC3339,
				"2006-01-02T15:04:05Z07:00",
				"2006-01-02 15:04:05.999999999-07:00",
				"2006-01-02 15:04:05.999999999+00:00",
				"2006-01-02 15:04:05+00:00",
				"2006-01-02 15:04:05",
			} {
				if parsed, err := time.Parse(layout, lastDateStr); err == nil {
					result[email] = parsed
					break
				}
			}
		}
	}
	return result, nil
}
