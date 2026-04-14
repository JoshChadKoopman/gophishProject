package models

import (
	"errors"
	"time"

	log "github.com/gophish/gophish/logger"
)

// TrainingPresentation represents an uploaded cybersecurity training presentation
type TrainingPresentation struct {
	Id            int64     `json:"id" gorm:"column:id; primary_key:yes"`
	OrgId         int64     `json:"-" gorm:"column:org_id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	FileName      string    `json:"file_name" gorm:"column:file_name"`
	FilePath      string    `json:"-" gorm:"column:file_path"`
	FileSize      int64     `json:"file_size" gorm:"column:file_size"`
	ContentType   string    `json:"content_type" gorm:"column:content_type"`
	ThumbnailPath string    `json:"thumbnail_path" gorm:"column:thumbnail_path"`
	YouTubeURL    string    `json:"youtube_url" gorm:"column:youtube_url"`
	ContentPages  string    `json:"content_pages" gorm:"column:content_pages;type:text"`
	UploadedBy    int64     `json:"uploaded_by" gorm:"column:uploaded_by"`
	CreatedDate   time.Time `json:"created_date" gorm:"column:created_date"`
	ModifiedDate  time.Time `json:"modified_date" gorm:"column:modified_date"`
}

// CourseProgress tracks a user's progress through a training course
type CourseProgress struct {
	Id               int64     `json:"id" gorm:"column:id; primary_key:yes"`
	UserId           int64     `json:"user_id" gorm:"column:user_id"`
	PresentationId   int64     `json:"presentation_id" gorm:"column:presentation_id"`
	CurrentPage      int       `json:"current_page" gorm:"column:current_page"`
	TotalPages       int       `json:"total_pages" gorm:"column:total_pages"`
	Status           string    `json:"status" gorm:"column:status"` // "no_progress", "in_progress", "complete"
	CompletedDate    time.Time `json:"completed_date" gorm:"column:completed_date"`
	LastAccessedDate time.Time `json:"last_accessed_date" gorm:"column:last_accessed_date"`
	CreatedDate      time.Time `json:"created_date" gorm:"column:created_date"`
}

// TableName overrides the default GORM table name.
func (CourseProgress) TableName() string {
	return "course_progress"
}

// ErrTrainingNameNotSpecified indicates there was no name specified
var ErrTrainingNameNotSpecified = errors.New("Training presentation name can't be empty")

// ErrTrainingFileNotSpecified indicates there was no file specified
var ErrTrainingFileNotSpecified = errors.New("Training presentation file is required")

// queryWherePresentationID is the shared WHERE clause for presentation_id lookups.
const queryWherePresentationID = "presentation_id=?"

// GetTrainingPresentations returns training presentations for the given org scope.
func GetTrainingPresentations(scope OrgScope) ([]TrainingPresentation, error) {
	tps := []TrainingPresentation{}
	err := scopeQuery(db.Table("training_presentations"), scope).Order("created_date desc").Find(&tps).Error
	return tps, err
}

// GetTrainingPresentation returns the training presentation with the given id and org scope.
func GetTrainingPresentation(id int64, scope OrgScope) (TrainingPresentation, error) {
	tp := TrainingPresentation{}
	err := scopeQuery(db.Where("id=?", id), scope).First(&tp).Error
	return tp, err
}

// PostTrainingPresentation creates a new training presentation in the database
func PostTrainingPresentation(tp *TrainingPresentation) error {
	if err := tp.Validate(); err != nil {
		log.Error(err)
		return err
	}
	tp.CreatedDate = time.Now().UTC()
	tp.ModifiedDate = time.Now().UTC()
	err := db.Save(tp).Error
	if err != nil {
		log.Error(err)
	}
	return err
}

// PutTrainingPresentation edits an existing training presentation
func PutTrainingPresentation(tp *TrainingPresentation) error {
	if err := tp.Validate(); err != nil {
		log.Error(err)
		return err
	}
	tp.ModifiedDate = time.Now().UTC()
	err := db.Save(tp).Error
	return err
}

// DeleteTrainingPresentation deletes a training presentation by id and all associated records.
func DeleteTrainingPresentation(id int64) error {
	// Delete associated quiz (cascades to questions and attempts)
	if err := DeleteQuiz(id); err != nil {
		log.Infof("No quiz to delete for presentation %d (or error: %v)", id, err)
	}
	// Delete associated certificates
	if err := db.Where(queryWherePresentationID, id).Delete(&Certificate{}).Error; err != nil {
		log.Error(err)
	}
	// Delete associated assignments
	if err := db.Where(queryWherePresentationID, id).Delete(&CourseAssignment{}).Error; err != nil {
		log.Error(err)
	}
	// Delete associated course progress records
	if err := db.Where(queryWherePresentationID, id).Delete(&CourseProgress{}).Error; err != nil {
		log.Error(err)
	}
	// Delete associated custom-builder assets. File blobs on disk are the
	// controller layer's responsibility; the controller walks the returned
	// slice and unlinks each file before calling this function.
	if _, err := DeleteTrainingAssetsByPresentation(id); err != nil {
		log.Error(err)
	}
	return db.Where("id=?", id).Delete(&TrainingPresentation{}).Error
}

// Validate checks that the required fields are set
func (tp *TrainingPresentation) Validate() error {
	if tp.Name == "" {
		return ErrTrainingNameNotSpecified
	}
	if tp.FileName == "" {
		return ErrTrainingFileNotSpecified
	}
	return nil
}

// GetCourseProgress returns progress for a user on a specific course
func GetCourseProgress(userId int64, presentationId int64) (CourseProgress, error) {
	cp := CourseProgress{}
	err := db.Where("user_id=? AND presentation_id=?", userId, presentationId).First(&cp).Error
	return cp, err
}

// GetUserCourseProgress returns all course progress records for a user
func GetUserCourseProgress(userId int64) ([]CourseProgress, error) {
	cps := []CourseProgress{}
	err := db.Where("user_id=?", userId).Order("last_accessed_date desc").Find(&cps).Error
	return cps, err
}

// SaveCourseProgress creates or updates a course progress record
func SaveCourseProgress(cp *CourseProgress) error {
	cp.LastAccessedDate = time.Now().UTC()
	if cp.Id == 0 {
		cp.CreatedDate = time.Now().UTC()
	}
	err := db.Save(cp).Error
	if err != nil {
		log.Error(err)
	}
	return err
}
