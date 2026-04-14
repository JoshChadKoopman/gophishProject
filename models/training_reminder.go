package models

import (
	"time"

	log "github.com/gophish/gophish/logger"
)

// TrainingReminder records a reminder sent for a training assignment.
type TrainingReminder struct {
	Id             int64     `json:"id" gorm:"primary_key"`
	UserId         int64     `json:"user_id" gorm:"column:user_id"`
	AssignmentId   int64     `json:"assignment_id" gorm:"column:assignment_id"`
	PresentationId int64     `json:"presentation_id" gorm:"column:presentation_id"`
	CourseName     string    `json:"course_name" gorm:"column:course_name"`
	DueDate        time.Time `json:"due_date" gorm:"column:due_date"`
	ReminderType   string    `json:"reminder_type" gorm:"column:reminder_type"` // "standard", "final", "urgent"
	Message        string    `json:"message" gorm:"column:message;type:text"`
	SentDate       time.Time `json:"sent_date" gorm:"column:sent_date"`
	EmailSent      bool      `json:"email_sent" gorm:"column:email_sent"`
}

func (TrainingReminder) TableName() string { return "training_reminders" }

// ReminderConfig holds org-level settings for automated reminders.
type ReminderConfig struct {
	Id                  int64  `json:"id" gorm:"primary_key"`
	OrgId               int64  `json:"org_id" gorm:"unique_index"`
	Enabled             bool   `json:"enabled"`
	FirstReminderHours  int    `json:"first_reminder_hours"`                                  // Hours before due (default 48)
	SecondReminderHours int    `json:"second_reminder_hours"`                                 // Hours before due for final (default 24)
	UrgentReminderHours int    `json:"urgent_reminder_hours"`                                 // Hours before due for urgent (default 4)
	EscalateOverdueDays int    `json:"escalate_overdue_days"`                                 // Days overdue before escalation (default 2)
	EmailTemplate       string `json:"email_template" gorm:"column:email_template;type:text"` // Custom email template
	SendingProfileId    int64  `json:"sending_profile_id" gorm:"column:sending_profile_id"`   // SMTP profile to use
}

func (ReminderConfig) TableName() string { return "reminder_configs" }

// Default reminder settings
const (
	DefaultFirstReminderHours  = 48
	DefaultSecondReminderHours = 24
	DefaultUrgentReminderHours = 4
	DefaultEscalateOverdueDays = 2
)

// GetReminderConfig retrieves the reminder config for an org, or returns defaults.
func GetReminderConfig(orgId int64) ReminderConfig {
	cfg := ReminderConfig{}
	err := db.Where(queryWhereOrgID, orgId).First(&cfg).Error
	if err != nil {
		return ReminderConfig{
			OrgId:               orgId,
			Enabled:             true,
			FirstReminderHours:  DefaultFirstReminderHours,
			SecondReminderHours: DefaultSecondReminderHours,
			UrgentReminderHours: DefaultUrgentReminderHours,
			EscalateOverdueDays: DefaultEscalateOverdueDays,
		}
	}
	return cfg
}

// SaveReminderConfig creates or updates the reminder config.
func SaveReminderConfig(cfg *ReminderConfig) error {
	existing := ReminderConfig{}
	err := db.Where(queryWhereOrgID, cfg.OrgId).First(&existing).Error
	if err == nil {
		cfg.Id = existing.Id
	}
	return db.Save(cfg).Error
}

// CreateTrainingReminder inserts a new training reminder record.
func CreateTrainingReminder(r *TrainingReminder) error {
	r.SentDate = time.Now().UTC()
	return db.Save(r).Error
}

// GetUserReminders returns reminders for a specific user, ordered by sent date.
func GetUserReminders(userId int64, limit int) ([]TrainingReminder, error) {
	reminders := []TrainingReminder{}
	if limit <= 0 {
		limit = 20
	}
	err := db.Where(queryWhereUserID, userId).
		Order("sent_date desc").
		Limit(limit).
		Find(&reminders).Error
	return reminders, err
}

// GetReminderStats returns aggregate stats about reminders sent.
type ReminderStats struct {
	TotalSent     int `json:"total_sent"`
	StandardCount int `json:"standard_count"`
	FinalCount    int `json:"final_count"`
	UrgentCount   int `json:"urgent_count"`
	Last24Hours   int `json:"last_24_hours"`
	Escalated     int `json:"escalated"`
}

// GetReminderStats returns reminder statistics for an org.
func GetReminderStatsForOrg(orgId int64) ReminderStats {
	stats := ReminderStats{}

	const joinUsers = "JOIN users ON training_reminders.user_id = users.id"
	const whereOrgAndType = "users.org_id = ? AND training_reminders.reminder_type = ?"

	db.Table("training_reminders").Joins(joinUsers).
		Where("users.org_id = ?", orgId).Count(&stats.TotalSent)

	db.Table("training_reminders").Joins(joinUsers).
		Where(whereOrgAndType, orgId, "standard").Count(&stats.StandardCount)

	db.Table("training_reminders").Joins(joinUsers).
		Where(whereOrgAndType, orgId, "final").Count(&stats.FinalCount)

	db.Table("training_reminders").Joins(joinUsers).
		Where(whereOrgAndType, orgId, "urgent").Count(&stats.UrgentCount)

	oneDayAgo := time.Now().UTC().Add(-24 * time.Hour)
	db.Table("training_reminders").Joins(joinUsers).
		Where("users.org_id = ? AND training_reminders.sent_date > ?", orgId, oneDayAgo).
		Count(&stats.Last24Hours)

	// Count escalated assignments
	db.Table("course_assignments").
		Joins("JOIN users ON course_assignments.user_id = users.id").
		Where("users.org_id = ? AND course_assignments.escalated_to > 0", orgId).
		Count(&stats.Escalated)

	_ = log.Logger // ensure import
	return stats
}
