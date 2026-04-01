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
	CreatedDate    time.Time `json:"created_date" gorm:"column:created_date"`
	ModifiedDate   time.Time `json:"modified_date" gorm:"column:modified_date"`
}

// Assignment status constants
const (
	AssignmentStatusPending    = "pending"
	AssignmentStatusInProgress = "in_progress"
	AssignmentStatusCompleted  = "completed"
)

var ErrAssignmentExists = errors.New("Course already assigned to this user")

// GetAssignment returns the assignment for a user on a specific presentation.
func GetAssignment(userId, presentationId int64) (CourseAssignment, error) {
	a := CourseAssignment{}
	err := db.Where("user_id=? AND presentation_id=?", userId, presentationId).First(&a).Error
	return a, err
}

// GetAssignmentsForUser returns all assignments for a user, ordered by creation date.
func GetAssignmentsForUser(userId int64) ([]CourseAssignment, error) {
	assignments := []CourseAssignment{}
	err := db.Where("user_id=?", userId).Order("created_date desc").Find(&assignments).Error
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
	err := db.Order("created_date desc").Find(&assignments).Error
	return assignments, err
}

// PostAssignment creates a new course assignment. Returns ErrAssignmentExists
// if the user already has an assignment for this presentation.
func PostAssignment(a *CourseAssignment) error {
	// Check for existing assignment (unique index will catch this too, but we want a clean error)
	existing := CourseAssignment{}
	err := db.Where("user_id=? AND presentation_id=?", a.UserId, a.PresentationId).First(&existing).Error
	if err == nil {
		return ErrAssignmentExists
	}
	a.Status = AssignmentStatusPending
	a.CreatedDate = time.Now().UTC()
	a.ModifiedDate = time.Now().UTC()
	return db.Save(a).Error
}

// UpdateAssignmentStatus updates the status of an assignment.
func UpdateAssignmentStatus(userId, presentationId int64, status string) error {
	return db.Model(&CourseAssignment{}).
		Where("user_id=? AND presentation_id=?", userId, presentationId).
		Updates(map[string]interface{}{
			"status":        status,
			"modified_date": time.Now().UTC(),
		}).Error
}

// DeleteAssignment removes an assignment by ID.
func DeleteAssignment(id int64) error {
	return db.Where("id=?", id).Delete(&CourseAssignment{}).Error
}

// GroupAssignResult contains the summary of a group assignment operation.
type GroupAssignResult struct {
	TotalTargets         int `json:"total_targets"`
	Assigned             int `json:"assigned"`
	SkippedNoAccount     int `json:"skipped_no_account"`
	SkippedAlreadyAssigned int `json:"skipped_already_assigned"`
}

// AssignCourseToGroup assigns a course to all platform users that match
// the email addresses in the given group's targets.
func AssignCourseToGroup(presentationId, groupId, assignedBy int64, dueDate time.Time) (GroupAssignResult, error) {
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
