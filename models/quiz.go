package models

import (
	"errors"
	"time"

	log "github.com/gophish/gophish/logger"
)

// Quiz represents a quiz attached to a training presentation.
// Each presentation may have at most one quiz (presentation_id is UNIQUE).
type Quiz struct {
	Id             int64          `json:"id" gorm:"column:id; primary_key:yes"`
	PresentationId int64          `json:"presentation_id" gorm:"column:presentation_id"`
	PassPercentage int            `json:"pass_percentage" gorm:"column:pass_percentage"`
	CreatedBy      int64          `json:"created_by" gorm:"column:created_by"`
	CreatedDate    time.Time      `json:"created_date" gorm:"column:created_date"`
	ModifiedDate   time.Time      `json:"modified_date" gorm:"column:modified_date"`
	Questions      []QuizQuestion `json:"questions,omitempty" gorm:"-"`
}

// QuizQuestion represents a single multiple-choice question in a quiz.
// Options is a JSON-encoded array of strings; CorrectOption is the 0-based index.
type QuizQuestion struct {
	Id            int64     `json:"id" gorm:"column:id; primary_key:yes"`
	QuizId        int64     `json:"quiz_id" gorm:"column:quiz_id"`
	QuestionText  string    `json:"question_text" gorm:"column:question_text"`
	Options       string    `json:"options" gorm:"column:options;type:text"`
	CorrectOption int       `json:"correct_option" gorm:"column:correct_option"`
	SortOrder     int       `json:"sort_order" gorm:"column:sort_order"`
	CreatedDate   time.Time `json:"created_date" gorm:"column:created_date"`
}

// QuizAttempt records a user's quiz submission and score.
type QuizAttempt struct {
	Id             int64     `json:"id" gorm:"column:id; primary_key:yes"`
	QuizId         int64     `json:"quiz_id" gorm:"column:quiz_id"`
	UserId         int64     `json:"user_id" gorm:"column:user_id"`
	Score          int       `json:"score" gorm:"column:score"`
	TotalQuestions int       `json:"total_questions" gorm:"column:total_questions"`
	Passed         bool      `json:"passed" gorm:"column:passed"`
	Answers        string    `json:"answers" gorm:"column:answers;type:text"`
	CompletedDate  time.Time `json:"completed_date" gorm:"column:completed_date"`
}

var ErrQuizNotFound = errors.New("Quiz not found")
var ErrNoQuestions = errors.New("Quiz must have at least one question")

// GetQuizByPresentationId returns the quiz (with questions) for a presentation.
func GetQuizByPresentationId(presentationId int64) (Quiz, error) {
	q := Quiz{}
	err := db.Where("presentation_id=?", presentationId).First(&q).Error
	if err != nil {
		return q, err
	}
	questions := []QuizQuestion{}
	err = db.Where("quiz_id=?", q.Id).Order("sort_order asc").Find(&questions).Error
	q.Questions = questions
	return q, err
}

// PostQuiz creates a new quiz for a presentation.
func PostQuiz(q *Quiz) error {
	q.CreatedDate = time.Now().UTC()
	q.ModifiedDate = time.Now().UTC()
	return db.Save(q).Error
}

// PutQuiz updates an existing quiz's metadata (e.g. pass_percentage).
func PutQuiz(q *Quiz) error {
	q.ModifiedDate = time.Now().UTC()
	return db.Save(q).Error
}

// DeleteQuiz deletes a quiz and its questions/attempts by presentation ID.
func DeleteQuiz(presentationId int64) error {
	q := Quiz{}
	err := db.Where("presentation_id=?", presentationId).First(&q).Error
	if err != nil {
		return err
	}
	// Delete attempts first, then questions, then the quiz itself
	if err := db.Where("quiz_id=?", q.Id).Delete(&QuizAttempt{}).Error; err != nil {
		log.Error(err)
	}
	if err := db.Where("quiz_id=?", q.Id).Delete(&QuizQuestion{}).Error; err != nil {
		log.Error(err)
	}
	return db.Where("id=?", q.Id).Delete(&Quiz{}).Error
}

// SaveQuizQuestions replaces all questions for a quiz in a transaction.
// Follows the same delete-all-then-insert pattern used for group targets.
func SaveQuizQuestions(quizId int64, questions []QuizQuestion) error {
	tx := db.Begin()
	// Delete existing questions
	if err := tx.Where("quiz_id=?", quizId).Delete(&QuizQuestion{}).Error; err != nil {
		tx.Rollback()
		return err
	}
	// Insert new questions with sequential sort order
	for i, q := range questions {
		q.QuizId = quizId
		q.SortOrder = i
		q.CreatedDate = time.Now().UTC()
		if err := tx.Save(&q).Error; err != nil {
			tx.Rollback()
			return err
		}
	}
	return tx.Commit().Error
}

// PostQuizAttempt saves a quiz attempt record.
func PostQuizAttempt(a *QuizAttempt) error {
	a.CompletedDate = time.Now().UTC()
	return db.Save(a).Error
}

// GetQuizAttempts returns all attempts for a user on a specific quiz.
func GetQuizAttempts(userId, quizId int64) ([]QuizAttempt, error) {
	attempts := []QuizAttempt{}
	err := db.Where("user_id=? AND quiz_id=?", userId, quizId).
		Order("completed_date desc").Find(&attempts).Error
	return attempts, err
}

// GetLatestPassedAttempt returns the most recent passing attempt for a user on a quiz.
func GetLatestPassedAttempt(userId, quizId int64) (QuizAttempt, error) {
	a := QuizAttempt{}
	err := db.Where("user_id=? AND quiz_id=? AND passed=?", userId, quizId, true).
		Order("completed_date desc").First(&a).Error
	return a, err
}

// QuizExistsForPresentation returns true if a quiz is attached to the presentation.
func QuizExistsForPresentation(presentationId int64) bool {
	q := Quiz{}
	err := db.Where("presentation_id=?", presentationId).First(&q).Error
	return err == nil
}
