package models

import (
	"time"

	log "github.com/gophish/gophish/logger"
)

// ContentUpdateConfig controls auto-update behaviour for the training content
// library at the organization level. When enabled, the background worker
// automatically seeds new built-in content into the org's academy.
type ContentUpdateConfig struct {
	Id                int64     `json:"id" gorm:"primary_key"`
	OrgId             int64     `json:"org_id" gorm:"unique_index"`
	Enabled           bool      `json:"enabled"`                             // Auto-update enabled
	AutoAssignNew     bool      `json:"auto_assign_new"`                     // Auto-assign new content to groups
	NotifyAdmins      bool      `json:"notify_admins"`                       // Notify admins when new content is seeded
	ContentCategories string    `json:"content_categories" gorm:"type:text"` // Comma-separated categories to include (empty = all)
	MinDifficulty     int       `json:"min_difficulty"`                      // Minimum difficulty to auto-seed (0 = all)
	MaxDifficulty     int       `json:"max_difficulty"`                      // Maximum difficulty to auto-seed (0 = all)
	ModifiedDate      time.Time `json:"modified_date" gorm:"column:modified_date"`
}

func (ContentUpdateConfig) TableName() string { return "content_update_configs" }

// Default content update settings
const (
	DefaultContentUpdateEnabled = true
	DefaultAutoAssignNew        = false
	DefaultNotifyAdmins         = true
)

// GetContentUpdateConfig retrieves the content update config for an org, or returns defaults.
func GetContentUpdateConfig(orgId int64) ContentUpdateConfig {
	cfg := ContentUpdateConfig{}
	err := db.Where(queryWhereOrgID, orgId).First(&cfg).Error
	if err != nil {
		return ContentUpdateConfig{
			OrgId:         orgId,
			Enabled:       DefaultContentUpdateEnabled,
			AutoAssignNew: DefaultAutoAssignNew,
			NotifyAdmins:  DefaultNotifyAdmins,
			MinDifficulty: 0,
			MaxDifficulty: 0,
		}
	}
	return cfg
}

// SaveContentUpdateConfig creates or updates the content update config for an org.
func SaveContentUpdateConfig(cfg *ContentUpdateConfig) error {
	existing := ContentUpdateConfig{}
	err := db.Where(queryWhereOrgID, cfg.OrgId).First(&existing).Error
	if err == nil {
		cfg.Id = existing.Id
	}
	cfg.ModifiedDate = time.Now().UTC()
	return db.Save(cfg).Error
}

// ContentUpdateLog records each automatic content library update cycle for
// audit and operational visibility.
type ContentUpdateLog struct {
	Id            int64     `json:"id" gorm:"primary_key"`
	OrgId         int64     `json:"org_id" gorm:"column:org_id"`
	OrgName       string    `json:"org_name" gorm:"column:org_name"`
	CoursesAdded  int       `json:"courses_added" gorm:"column:courses_added"`
	SessionsAdded int       `json:"sessions_added" gorm:"column:sessions_added"`
	QuizzesAdded  int       `json:"quizzes_added" gorm:"column:quizzes_added"`
	Skipped       int       `json:"skipped" gorm:"column:skipped"`
	Status        string    `json:"status" gorm:"column:status"` // "success", "partial", "error", "skipped"
	ErrorMessage  string    `json:"error_message,omitempty" gorm:"column:error_message;type:text"`
	RunDate       time.Time `json:"run_date" gorm:"column:run_date"`
}

func (ContentUpdateLog) TableName() string { return "content_update_logs" }

// RecordContentUpdate saves a content update log entry.
func RecordContentUpdate(entry *ContentUpdateLog) {
	entry.RunDate = time.Now().UTC()
	if err := db.Save(entry).Error; err != nil {
		log.Errorf("content_update: failed to save update log: %v", err)
	}
}

// GetContentUpdateHistory returns the last N content update logs for an org.
func GetContentUpdateHistory(orgId int64, limit int) ([]ContentUpdateLog, error) {
	var logs []ContentUpdateLog
	if limit <= 0 {
		limit = 20
	}
	err := db.Where(queryWhereOrgID, orgId).
		Order("run_date desc").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

// GetGlobalContentUpdateHistory returns recent content update logs across all orgs.
func GetGlobalContentUpdateHistory(limit int) ([]ContentUpdateLog, error) {
	var logs []ContentUpdateLog
	if limit <= 0 {
		limit = 50
	}
	err := db.Order("run_date desc").Limit(limit).Find(&logs).Error
	return logs, err
}

// ContentUpdateSummary provides a high-level overview of the content update system.
type ContentUpdateSummary struct {
	Config          ContentUpdateConfig `json:"config"`
	LastRunDate     *time.Time          `json:"last_run_date"`
	TotalUpdates    int                 `json:"total_updates"`
	TotalNewCourses int                 `json:"total_new_courses"`
	LibrarySize     int                 `json:"library_size"`
	SeededCount     int                 `json:"seeded_count"`
	RecentRuns      []ContentUpdateLog  `json:"recent_runs"`
}

// GetContentUpdateSummary returns a summary of the content update system for an org.
func GetContentUpdateSummary(orgId int64) ContentUpdateSummary {
	summary := ContentUpdateSummary{
		Config:      GetContentUpdateConfig(orgId),
		LibrarySize: len(GetBuiltInContentLibrary()),
	}

	// Count seeded presentations for this org
	var seeded int
	db.Table("training_presentations").
		Where("org_id = ? AND content_type = ?", orgId, "application/nivoxis-builtin").
		Count(&seeded)
	summary.SeededCount = seeded

	runs, _ := GetContentUpdateHistory(orgId, 10)
	summary.RecentRuns = runs

	if len(runs) > 0 {
		summary.LastRunDate = &runs[0].RunDate
	}

	for _, r := range runs {
		summary.TotalUpdates++
		summary.TotalNewCourses += r.CoursesAdded
	}

	return summary
}
