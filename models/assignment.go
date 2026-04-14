package models

import (
	"errors"
	"time"

	log "github.com/gophish/gophish/logger"
	"github.com/jinzhu/gorm"
)

// CourseAssignment represents a training course assigned to a user.
type CourseAssignment struct {
	Id             int64     `json:"id" gorm:"column:id; primary_key:yes"`
	UserId         int64     `json:"user_id" gorm:"column:user_id"`
	PresentationId int64     `json:"presentation_id" gorm:"column:presentation_id"`
	AssignedBy     int64     `json:"assigned_by" gorm:"column:assigned_by"`
	GroupId        int64     `json:"group_id,omitempty" gorm:"column:group_id"`
	CampaignId     int64     `json:"campaign_id,omitempty" gorm:"column:campaign_id"`
	DueDate        time.Time `json:"due_date" gorm:"column:due_date"`
	Status         string    `json:"status" gorm:"column:status"`
	Priority       string    `json:"priority" gorm:"column:priority"`
	ReminderSent   bool      `json:"reminder_sent" gorm:"column:reminder_sent"`
	ReminderDate   time.Time `json:"reminder_date,omitempty" gorm:"column:reminder_date"`
	EscalatedTo    int64     `json:"escalated_to,omitempty" gorm:"column:escalated_to"`
	EscalatedDate  time.Time `json:"escalated_date,omitempty" gorm:"column:escalated_date"`
	CompletedDate  time.Time `json:"completed_date,omitempty" gorm:"column:completed_date"`
	Notes          string    `json:"notes,omitempty" gorm:"column:notes"`
	CreatedDate    time.Time `json:"created_date" gorm:"column:created_date"`
	ModifiedDate   time.Time `json:"modified_date" gorm:"column:modified_date"`
}

// Assignment status constants
const (
	AssignmentStatusPending    = "pending"
	AssignmentStatusInProgress = "in_progress"
	AssignmentStatusCompleted  = "completed"
	AssignmentStatusOverdue    = "overdue"
	AssignmentStatusCancelled  = "cancelled"
)

// Assignment priority constants
const (
	AssignmentPriorityLow      = "low"
	AssignmentPriorityNormal   = "normal"
	AssignmentPriorityHigh     = "high"
	AssignmentPriorityCritical = "critical"
)

var ErrAssignmentExists = errors.New("Course already assigned to this user")
var ErrAssignmentNotFound = errors.New("Assignment not found")
var ErrInvalidAssignmentStatus = errors.New("Invalid assignment status")

// ValidAssignmentStatuses contains the set of valid status values.
var ValidAssignmentStatuses = map[string]bool{
	AssignmentStatusPending:    true,
	AssignmentStatusInProgress: true,
	AssignmentStatusCompleted:  true,
	AssignmentStatusOverdue:    true,
	AssignmentStatusCancelled:  true,
}

// ValidAssignmentPriorities contains the set of valid priority values.
var ValidAssignmentPriorities = map[string]bool{
	AssignmentPriorityLow:      true,
	AssignmentPriorityNormal:   true,
	AssignmentPriorityHigh:     true,
	AssignmentPriorityCritical: true,
}

// Shared query constants for assignments.
const (
	queryWhereUserAndPresentation = "user_id=? AND presentation_id=?"
	orderCreatedDateDesc          = "created_date desc"
	orderDueDateAsc               = "due_date asc"
	queryWhereStatus              = "status=?"
	queryWherePriorityAndActive   = "priority=? AND status NOT IN (?)"
)

// terminatedStatuses is the list of assignment statuses that are considered final.
var terminatedStatuses = []string{AssignmentStatusCompleted, AssignmentStatusCancelled}

// GetAssignment returns the assignment for a user on a specific presentation.
func GetAssignment(userId, presentationId int64) (CourseAssignment, error) {
	a := CourseAssignment{}
	err := db.Where(queryWhereUserAndPresentation, userId, presentationId).First(&a).Error
	return a, err
}

// GetAssignmentById returns an assignment by its ID.
func GetAssignmentById(id int64) (CourseAssignment, error) {
	a := CourseAssignment{}
	err := db.Where("id=?", id).First(&a).Error
	if err != nil {
		return a, ErrAssignmentNotFound
	}
	return a, nil
}

// GetAssignmentsForUser returns all assignments for a user, ordered by creation date.
func GetAssignmentsForUser(userId int64) ([]CourseAssignment, error) {
	assignments := []CourseAssignment{}
	err := db.Where("user_id=?", userId).Order(orderCreatedDateDesc).Find(&assignments).Error
	return assignments, err
}

// GetAssignmentsForPresentation returns all assignments for a presentation.
func GetAssignmentsForPresentation(presentationId int64) ([]CourseAssignment, error) {
	assignments := []CourseAssignment{}
	err := db.Where("presentation_id=?", presentationId).Find(&assignments).Error
	return assignments, err
}

// GetAllAssignments returns all assignments, ordered by creation date.
func GetAllAssignments() ([]CourseAssignment, error) {
	assignments := []CourseAssignment{}
	err := db.Order(orderCreatedDateDesc).Find(&assignments).Error
	return assignments, err
}

// GetOverdueAssignments returns assignments past their due date that are not completed or cancelled.
func GetOverdueAssignments() ([]CourseAssignment, error) {
	assignments := []CourseAssignment{}
	now := time.Now().UTC()
	err := db.Where("due_date < ? AND due_date != ? AND status NOT IN (?)",
		now, time.Time{}, terminatedStatuses).
		Order(orderDueDateAsc).Find(&assignments).Error
	return assignments, err
}

// GetPendingReminderAssignments returns assignments approaching their due date
// that haven't had a reminder sent yet. The threshold is the number of hours
// before due date to start sending reminders.
func GetPendingReminderAssignments(hoursBeforeDue int) ([]CourseAssignment, error) {
	assignments := []CourseAssignment{}
	now := time.Now().UTC()
	threshold := now.Add(time.Duration(hoursBeforeDue) * time.Hour)
	err := db.Where(
		"due_date > ? AND due_date <= ? AND due_date != ? AND reminder_sent = 0 AND status NOT IN (?)",
		now, threshold, time.Time{},
		[]string{AssignmentStatusCompleted, AssignmentStatusCancelled},
	).Order(orderDueDateAsc).Find(&assignments).Error
	return assignments, err
}

// GetAssignmentsByStatus returns all assignments with a given status.
func GetAssignmentsByStatus(status string) ([]CourseAssignment, error) {
	assignments := []CourseAssignment{}
	err := db.Where(queryWhereStatus, status).Order(orderCreatedDateDesc).Find(&assignments).Error
	return assignments, err
}

// GetAssignmentsByPriority returns all non-completed assignments with a given priority.
func GetAssignmentsByPriority(priority string) ([]CourseAssignment, error) {
	assignments := []CourseAssignment{}
	err := db.Where(queryWherePriorityAndActive, priority,
		terminatedStatuses).
		Order(orderDueDateAsc).Find(&assignments).Error
	return assignments, err
}

// GetActiveAssignmentCount returns the number of active (non-completed, non-cancelled) assignments for a user.
func GetActiveAssignmentCount(userId int64) int {
	var count int
	db.Table("course_assignments").Where(
		"user_id=? AND status NOT IN (?)", userId,
		terminatedStatuses,
	).Count(&count)
	return count
}

// PostAssignment creates a new course assignment. Returns ErrAssignmentExists
// if the user already has an assignment for this presentation.
func PostAssignment(a *CourseAssignment) error {
	// Check for existing assignment (unique index will catch this too, but we want a clean error)
	existing := CourseAssignment{}
	err := db.Where(queryWhereUserAndPresentation, a.UserId, a.PresentationId).First(&existing).Error
	if err == nil {
		return ErrAssignmentExists
	}
	a.Status = AssignmentStatusPending
	if a.Priority == "" {
		a.Priority = AssignmentPriorityNormal
	}
	a.CreatedDate = time.Now().UTC()
	a.ModifiedDate = time.Now().UTC()
	return db.Save(a).Error
}

// UpdateAssignmentStatus updates the status of an assignment.
func UpdateAssignmentStatus(userId, presentationId int64, status string) error {
	if !ValidAssignmentStatuses[status] {
		return ErrInvalidAssignmentStatus
	}
	updates := map[string]interface{}{
		"status":        status,
		"modified_date": time.Now().UTC(),
	}
	if status == AssignmentStatusCompleted {
		updates["completed_date"] = time.Now().UTC()
	}
	return db.Model(&CourseAssignment{}).
		Where(queryWhereUserAndPresentation, userId, presentationId).
		Updates(updates).Error
}

// MarkReminderSent marks the reminder as sent for an assignment.
func MarkReminderSent(id int64) error {
	return db.Model(&CourseAssignment{}).Where("id=?", id).Updates(map[string]interface{}{
		"reminder_sent": true,
		"reminder_date": time.Now().UTC(),
		"modified_date": time.Now().UTC(),
	}).Error
}

// EscalateAssignment marks an assignment as escalated to a manager.
func EscalateAssignment(id, escalatedTo int64) error {
	return db.Model(&CourseAssignment{}).Where("id=?", id).Updates(map[string]interface{}{
		"escalated_to":   escalatedTo,
		"escalated_date": time.Now().UTC(),
		"modified_date":  time.Now().UTC(),
	}).Error
}

// UpdateAssignmentPriority updates the priority of an assignment.
func UpdateAssignmentPriority(id int64, priority string) error {
	if !ValidAssignmentPriorities[priority] {
		return errors.New("invalid assignment priority")
	}
	return db.Model(&CourseAssignment{}).Where("id=?", id).Updates(map[string]interface{}{
		"priority":      priority,
		"modified_date": time.Now().UTC(),
	}).Error
}

// UpdateAssignmentNotes updates the notes field of an assignment.
func UpdateAssignmentNotes(id int64, notes string) error {
	return db.Model(&CourseAssignment{}).Where("id=?", id).Updates(map[string]interface{}{
		"notes":         notes,
		"modified_date": time.Now().UTC(),
	}).Error
}

// BulkUpdateAssignmentStatus updates the status of multiple assignments by ID.
func BulkUpdateAssignmentStatus(ids []int64, status string) (int, error) {
	if !ValidAssignmentStatuses[status] {
		return 0, ErrInvalidAssignmentStatus
	}
	updates := map[string]interface{}{
		"status":        status,
		"modified_date": time.Now().UTC(),
	}
	if status == AssignmentStatusCompleted {
		updates["completed_date"] = time.Now().UTC()
	}
	result := db.Model(&CourseAssignment{}).Where("id IN (?)", ids).Updates(updates)
	return int(result.RowsAffected), result.Error
}

// MarkOverdueAssignments transitions pending/in_progress assignments past their due date to overdue.
// Returns the number of assignments updated.
func MarkOverdueAssignments() (int, error) {
	now := time.Now().UTC()
	result := db.Model(&CourseAssignment{}).
		Where("due_date < ? AND due_date != ? AND status IN (?)",
			now, time.Time{}, []string{AssignmentStatusPending, AssignmentStatusInProgress}).
		Updates(map[string]interface{}{
			"status":        AssignmentStatusOverdue,
			"modified_date": now,
		})
	return int(result.RowsAffected), result.Error
}

// DeleteAssignment removes an assignment by ID.
func DeleteAssignment(id int64) error {
	return db.Where("id=?", id).Delete(&CourseAssignment{}).Error
}

// CancelAssignment marks an assignment as cancelled instead of hard-deleting.
func CancelAssignment(id int64) error {
	return db.Model(&CourseAssignment{}).Where("id=?", id).Updates(map[string]interface{}{
		"status":        AssignmentStatusCancelled,
		"modified_date": time.Now().UTC(),
	}).Error
}

// AssignmentSummary provides aggregate statistics for assignment management dashboards.
type AssignmentSummary struct {
	TotalAssignments int `json:"total_assignments"`
	Pending          int `json:"pending"`
	InProgress       int `json:"in_progress"`
	Completed        int `json:"completed"`
	Overdue          int `json:"overdue"`
	Cancelled        int `json:"cancelled"`
	HighPriority     int `json:"high_priority"`
	CriticalPriority int `json:"critical_priority"`
	RemindersSent    int `json:"reminders_sent"`
}

// GetAssignmentSummary returns aggregate assignment statistics.
func GetAssignmentSummary() (AssignmentSummary, error) {
	summary := AssignmentSummary{}
	db.Table("course_assignments").Count(&summary.TotalAssignments)
	db.Table("course_assignments").Where(queryWhereStatus, AssignmentStatusPending).Count(&summary.Pending)
	db.Table("course_assignments").Where(queryWhereStatus, AssignmentStatusInProgress).Count(&summary.InProgress)
	db.Table("course_assignments").Where(queryWhereStatus, AssignmentStatusCompleted).Count(&summary.Completed)
	db.Table("course_assignments").Where(queryWhereStatus, AssignmentStatusOverdue).Count(&summary.Overdue)
	db.Table("course_assignments").Where(queryWhereStatus, AssignmentStatusCancelled).Count(&summary.Cancelled)
	db.Table("course_assignments").Where(queryWherePriorityAndActive,
		AssignmentPriorityHigh, terminatedStatuses).Count(&summary.HighPriority)
	db.Table("course_assignments").Where(queryWherePriorityAndActive,
		AssignmentPriorityCritical, terminatedStatuses).Count(&summary.CriticalPriority)
	db.Table("course_assignments").Where("reminder_sent=1").Count(&summary.RemindersSent)
	return summary, nil
}

// GroupAssignResult contains the summary of a group assignment operation.
type GroupAssignResult struct {
	TotalTargets           int `json:"total_targets"`
	Assigned               int `json:"assigned"`
	SkippedNoAccount       int `json:"skipped_no_account"`
	SkippedAlreadyAssigned int `json:"skipped_already_assigned"`
}

// AssignCourseToGroup assigns a course to all platform users that match
// the email addresses in the given group's targets.
func AssignCourseToGroup(presentationId, groupId, assignedBy int64, dueDate time.Time) (GroupAssignResult, error) {
	return AssignCourseToGroupWithPriority(presentationId, groupId, assignedBy, dueDate, AssignmentPriorityNormal)
}

// AssignCourseToGroupWithPriority assigns a course to a group with a specified priority.
func AssignCourseToGroupWithPriority(presentationId, groupId, assignedBy int64, dueDate time.Time, priority string) (GroupAssignResult, error) {
	result := GroupAssignResult{}

	// Load group targets
	targets, err := GetTargets(groupId)
	if err != nil {
		return result, err
	}
	result.TotalTargets = len(targets)

	for _, target := range targets {
		// Look up the platform user by email
		user, err := GetUserByUsername(target.Email)
		if err != nil {
			// Also try the email field
			user, err = GetUserByEmail(target.Email)
			if err != nil {
				result.SkippedNoAccount++
				continue
			}
		}
		// Try to create the assignment
		a := &CourseAssignment{
			UserId:         user.Id,
			PresentationId: presentationId,
			AssignedBy:     assignedBy,
			GroupId:        groupId,
			DueDate:        dueDate,
			Priority:       priority,
		}
		err = PostAssignment(a)
		if err == ErrAssignmentExists {
			result.SkippedAlreadyAssigned++
			continue
		}
		if err != nil {
			log.Errorf("Error assigning course to user %d: %v", user.Id, err)
			continue
		}
		result.Assigned++
	}
	return result, nil
}

// AutoAssignOnClick is called asynchronously from HandleClickedLink to
// auto-assign a training course when a phishing target clicks a link.
// It is a no-op if the target is not a platform user or already has the assignment.
func AutoAssignOnClick(email string, presentationId, campaignId int64) error {
	// Look up platform user by username (which is their email in GoPhish)
	user, err := GetUserByUsername(email)
	if err != nil {
		// Also try the email field
		user, err = GetUserByEmail(email)
		if err != nil {
			// Target is not a platform user — expected, not an error
			return nil
		}
	}

	// Verify the presentation still exists
	scope := OrgScope{OrgId: user.OrgId, UserId: user.Id}
	_, err = GetTrainingPresentation(presentationId, scope)
	if err != nil {
		return nil
	}

	// Check if assignment already exists (dedup)
	_, err = GetAssignment(user.Id, presentationId)
	if err == nil {
		return nil // already assigned
	}

	// Create the assignment
	a := &CourseAssignment{
		UserId:         user.Id,
		PresentationId: presentationId,
		AssignedBy:     0, // system auto-assignment
		CampaignId:     campaignId,
		Priority:       AssignmentPriorityHigh, // auto-assigned from phishing = high priority
	}
	a.Status = AssignmentStatusPending
	a.CreatedDate = time.Now().UTC()
	a.ModifiedDate = time.Now().UTC()
	err = db.Save(a).Error
	if err != nil {
		// Unique index violation is expected on race condition — not an error
		if gorm.IsRecordNotFoundError(err) {
			return nil
		}
		return err
	}
	return nil
}

// CompleteAssignmentOnCourseFinish should be called when a user completes a course.
// It marks the assignment as completed and sets the completion date.
func CompleteAssignmentOnCourseFinish(userId, presentationId int64) error {
	a := CourseAssignment{}
	err := db.Where(queryWhereUserAndPresentation, userId, presentationId).First(&a).Error
	if err != nil {
		// No assignment for this user/course — not an error
		return nil
	}
	if a.Status == AssignmentStatusCompleted || a.Status == AssignmentStatusCancelled {
		return nil
	}
	return db.Model(&CourseAssignment{}).Where("id=?", a.Id).Updates(map[string]interface{}{
		"status":         AssignmentStatusCompleted,
		"completed_date": time.Now().UTC(),
		"modified_date":  time.Now().UTC(),
	}).Error
}
