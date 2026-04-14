package models

import (
	"time"

	log "github.com/gophish/gophish/logger"
)

// ── Enhanced Training Reminder Models ───────────────────────────
// Additional model functions for manual nudges, templates, and
// assignment-level reminder tracking.

// ReminderTemplate stores a custom email template for training reminders.
type ReminderTemplate struct {
	Id          int64  `json:"id" gorm:"primary_key"`
	OrgId       int64  `json:"org_id" gorm:"column:org_id;unique_index"`
	Subject     string `json:"subject" gorm:"column:subject;type:text"`
	BodyHTML    string `json:"body_html" gorm:"column:body_html;type:text"`
	BodyText    string `json:"body_text" gorm:"column:body_text;type:text"`
	IsCustom    bool   `json:"is_custom" gorm:"column:is_custom;default:false"`
}

func (ReminderTemplate) TableName() string { return "reminder_templates" }

// DefaultReminderSubject is the fallback subject line.
const DefaultReminderSubject = "Training Reminder: {{.CourseName}} — Due {{.DueDate}}"

// DefaultReminderBodyHTML is the fallback HTML body.
const DefaultReminderBodyHTML = `<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto">
<h2>Training Reminder</h2>
<p>Hello {{.FirstName}},</p>
<p>This is a reminder that your training course <strong>{{.CourseName}}</strong> is due on <strong>{{.DueDate}}</strong>.</p>
<p>Time remaining: <strong>{{.TimeLeft}}</strong></p>
<p><a href="{{.TrainingURL}}" style="background:#3498db;color:#fff;padding:12px 24px;text-decoration:none;border-radius:4px">Complete Training</a></p>
<p style="color:#888;font-size:12px">This is an automated reminder from your security awareness programme.</p>
</div>`

// GetReminderTemplate returns the custom template for an org, or defaults.
func GetReminderTemplate(orgId int64) ReminderTemplate {
	tpl := ReminderTemplate{}
	err := db.Where(queryWhereOrgID, orgId).First(&tpl).Error
	if err != nil {
		return ReminderTemplate{
			OrgId:    orgId,
			Subject:  DefaultReminderSubject,
			BodyHTML: DefaultReminderBodyHTML,
			IsCustom: false,
		}
	}
	return tpl
}

// SaveReminderTemplate upserts the custom reminder template.
func SaveReminderTemplate(tpl *ReminderTemplate) error {
	existing := ReminderTemplate{}
	err := db.Where(queryWhereOrgID, tpl.OrgId).First(&existing).Error
	if err == nil {
		tpl.Id = existing.Id
	}
	tpl.IsCustom = true
	return db.Save(tpl).Error
}

// RecordReminderSent creates a reminder record for manual/bulk nudges.
func RecordReminderSent(userId, assignmentId int64, reminderType string, sentBy int64) error {
	// Get assignment details for the record
	a, err := GetAssignmentById(assignmentId)
	if err != nil {
		return err
	}

	reminder := &TrainingReminder{
		UserId:         userId,
		AssignmentId:   assignmentId,
		PresentationId: a.PresentationId,
		DueDate:        a.DueDate,
		ReminderType:   reminderType,
		Message:        "Manual nudge sent by admin",
		EmailSent:      false, // manual nudge is recorded, actual email sending handled separately
		SentDate:       time.Now().UTC(),
	}
	if err := db.Save(reminder).Error; err != nil {
		log.Errorf("RecordReminderSent: failed for assignment %d: %v", assignmentId, err)
		return err
	}
	return nil
}

// GetPendingAssignmentsForUser returns all non-completed, non-cancelled assignments.
func GetPendingAssignmentsForUser(userId int64) ([]CourseAssignment, error) {
	assignments := []CourseAssignment{}
	err := db.Where("user_id = ? AND status NOT IN (?)", userId,
		[]string{AssignmentStatusCompleted, AssignmentStatusCancelled}).
		Order(orderDueDateAsc).
		Find(&assignments).Error
	return assignments, err
}

// GetOverdueAssignmentsForOrg returns all overdue assignments for an org.
func GetOverdueAssignmentsForOrg(orgId int64) ([]CourseAssignment, error) {
	assignments := []CourseAssignment{}
	err := db.Table("course_assignments").
		Joins("JOIN users ON course_assignments.user_id = users.id").
		Where("users.org_id = ? AND course_assignments.status = ?", orgId, AssignmentStatusOverdue).
		Find(&assignments).Error
	return assignments, err
}

// GetRemindersForAssignment returns all reminders sent for a specific assignment.
func GetRemindersForAssignment(assignmentId int64) ([]TrainingReminder, error) {
	reminders := []TrainingReminder{}
	err := db.Where("assignment_id = ?", assignmentId).
		Order("sent_date desc").
		Find(&reminders).Error
	return reminders, err
}
