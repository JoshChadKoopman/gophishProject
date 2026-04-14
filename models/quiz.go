package models

import (
	"encoding/json"
	"errors"
	"time"

	log "github.com/gophish/gophish/logger"
)

// Question type constants for the quiz engine.
const (
	// QuestionTypeMultipleChoice is a single-answer multiple-choice question.
	// The correct answer is stored in CorrectOption (0-based index).
	QuestionTypeMultipleChoice = "multiple_choice"
	// QuestionTypeTrueFalse is a true/false question. Options must be ["True","False"]
	// or similar. CorrectOption is 0 or 1.
	QuestionTypeTrueFalse = "true_false"
	// QuestionTypeMultiSelect allows multiple correct answers. The correct answers
	// are stored in CorrectOptions (JSON array of 0-based indices). Full credit
	// requires the submitted set to match exactly.
	QuestionTypeMultiSelect = "multi_select"
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

// QuizQuestion represents a single quiz question. The QuestionType field
// determines how the question is rendered and graded:
//   - multiple_choice: single correct answer, CorrectOption holds the index
//   - true_false: CorrectOption is 0 or 1
//   - multi_select: multiple correct answers, CorrectOptions is a JSON-encoded
//     array of 0-based indices that must all match for credit
//
// Explanation is shown to the user after the quiz is submitted and provides
// educational reinforcement for right or wrong answers.
type QuizQuestion struct {
	Id             int64     `json:"id" gorm:"column:id; primary_key:yes"`
	QuizId         int64     `json:"quiz_id" gorm:"column:quiz_id"`
	QuestionType   string    `json:"question_type" gorm:"column:question_type"`
	QuestionText   string    `json:"question_text" gorm:"column:question_text"`
	Options        string    `json:"options" gorm:"column:options;type:text"`
	CorrectOption  int       `json:"correct_option" gorm:"column:correct_option"`
	CorrectOptions string    `json:"correct_options" gorm:"column:correct_options;type:text"`
	Explanation    string    `json:"explanation" gorm:"column:explanation;type:text"`
	SortOrder      int       `json:"sort_order" gorm:"column:sort_order"`
	CreatedDate    time.Time `json:"created_date" gorm:"column:created_date"`
}

// GetCorrectOptionsSet parses CorrectOptions JSON into a set of indices.
// For multi_select questions this returns the expected correct answer set.
// For other question types it falls back to a set containing CorrectOption.
func (q *QuizQuestion) GetCorrectOptionsSet() map[int]bool {
	set := make(map[int]bool)
	if q.CorrectOptions != "" {
		var indices []int
		if err := json.Unmarshal([]byte(q.CorrectOptions), &indices); err == nil {
			for _, idx := range indices {
				set[idx] = true
			}
			return set
		}
	}
	set[q.CorrectOption] = true
	return set
}

// GradeAnswer compares a user's answer against the question's correct answer
// and returns true if the answer is fully correct. Answer is a slice to
// support multi_select; for single-answer questions it should contain one
// element (or be empty for no-answer).
func (q *QuizQuestion) GradeAnswer(answer []int) bool {
	switch q.QuestionType {
	case QuestionTypeMultiSelect:
		expected := q.GetCorrectOptionsSet()
		if len(answer) != len(expected) {
			return false
		}
		for _, idx := range answer {
			if !expected[idx] {
				return false
			}
		}
		return true
	default:
		// multiple_choice, true_false, and unset legacy questions
		if len(answer) != 1 {
			return false
		}
		return answer[0] == q.CorrectOption
	}
}

// NormalizeQuestionType returns a valid question type, defaulting to
// multiple_choice for empty or unknown values (preserves backward compatibility).
func NormalizeQuestionType(t string) string {
	switch t {
	case QuestionTypeMultipleChoice, QuestionTypeTrueFalse, QuestionTypeMultiSelect:
		return t
	default:
		return QuestionTypeMultipleChoice
	}
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

// queryWhereQuizID is the shared WHERE clause for quiz_id lookups.
const queryWhereQuizID = "quiz_id=?"

// GetQuizByPresentationId returns the quiz (with questions) for a presentation.
func GetQuizByPresentationId(presentationId int64) (Quiz, error) {
	q := Quiz{}
	err := db.Where(queryWherePresentationID, presentationId).First(&q).Error
	if err != nil {
		return q, err
	}
	questions := []QuizQuestion{}
	err = db.Where(queryWhereQuizID, q.Id).Order("sort_order asc").Find(&questions).Error
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
	err := db.Where(queryWherePresentationID, presentationId).First(&q).Error
	if err != nil {
		return err
	}
	// Delete attempts first, then questions, then the quiz itself
	if err := db.Where(queryWhereQuizID, q.Id).Delete(&QuizAttempt{}).Error; err != nil {
		log.Error(err)
	}
	if err := db.Where(queryWhereQuizID, q.Id).Delete(&QuizQuestion{}).Error; err != nil {
		log.Error(err)
	}
	return db.Where("id=?", q.Id).Delete(&Quiz{}).Error
}

// SaveQuizQuestions replaces all questions for a quiz in a transaction.
// Follows the same delete-all-then-insert pattern used for group targets.
func SaveQuizQuestions(quizId int64, questions []QuizQuestion) error {
	tx := db.Begin()
	// Delete existing questions
	if err := tx.Where(queryWhereQuizID, quizId).Delete(&QuizQuestion{}).Error; err != nil {
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
	err := db.Where(queryWherePresentationID, presentationId).First(&q).Error
	return err == nil
}
