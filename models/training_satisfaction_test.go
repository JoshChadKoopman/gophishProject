package models

import (
	"testing"

	"github.com/gophish/gophish/config"
)

// setupSatisfactionTest initialises an in-memory DB for satisfaction tests.
func setupSatisfactionTest(t *testing.T) func() {
	t.Helper()
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	if err := Setup(conf); err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	// Clear migration-seeded data
	db.Exec("DELETE FROM training_satisfaction_ratings")
	db.Exec("DELETE FROM course_progress")
	db.Exec("DELETE FROM quiz_attempts")
	db.Exec("DELETE FROM quiz_questions")
	db.Exec("DELETE FROM quizzes")
	db.Exec("DELETE FROM training_presentations")
	return func() {
		db.Exec("DELETE FROM training_satisfaction_ratings")
		db.Exec("DELETE FROM course_progress")
		db.Exec("DELETE FROM quiz_attempts")
		db.Exec("DELETE FROM quiz_questions")
		db.Exec("DELETE FROM quizzes")
		db.Exec("DELETE FROM training_presentations")
	}
}

// ---------- Satisfaction Ratings ----------

func TestPostSatisfactionRatingBasic(t *testing.T) {
	teardown := setupSatisfactionTest(t)
	defer teardown()

	r := &TrainingSatisfactionRating{
		UserId:         1,
		PresentationId: 100,
		Rating:         4,
		Feedback:       "Great course!",
	}
	if err := PostTrainingSatisfactionRating(r); err != nil {
		t.Fatalf("failed to post rating: %v", err)
	}
	if r.Id == 0 {
		t.Fatal("expected non-zero ID after save")
	}
	if r.CreatedDate.IsZero() {
		t.Fatal("expected CreatedDate to be set")
	}
}

func TestPostSatisfactionRatingClampsLow(t *testing.T) {
	teardown := setupSatisfactionTest(t)
	defer teardown()

	r := &TrainingSatisfactionRating{UserId: 1, PresentationId: 100, Rating: -5}
	PostTrainingSatisfactionRating(r)

	found, err := GetSatisfactionRating(1, 100)
	if err != nil {
		t.Fatalf("failed to get rating: %v", err)
	}
	if found.Rating != 1 {
		t.Fatalf("expected rating clamped to 1, got %d", found.Rating)
	}
}

func TestPostSatisfactionRatingClampsHigh(t *testing.T) {
	teardown := setupSatisfactionTest(t)
	defer teardown()

	r := &TrainingSatisfactionRating{UserId: 1, PresentationId: 100, Rating: 99}
	PostTrainingSatisfactionRating(r)

	found, _ := GetSatisfactionRating(1, 100)
	if found.Rating != 5 {
		t.Fatalf("expected rating clamped to 5, got %d", found.Rating)
	}
}

func TestPostSatisfactionRatingUpsert(t *testing.T) {
	teardown := setupSatisfactionTest(t)
	defer teardown()

	// First rating
	r := &TrainingSatisfactionRating{UserId: 1, PresentationId: 100, Rating: 3, Feedback: "OK"}
	PostTrainingSatisfactionRating(r)

	// Update same user/presentation — should upsert, not duplicate
	r2 := &TrainingSatisfactionRating{UserId: 1, PresentationId: 100, Rating: 5, Feedback: "Actually amazing!"}
	PostTrainingSatisfactionRating(r2)

	found, _ := GetSatisfactionRating(1, 100)
	if found.Rating != 5 {
		t.Fatalf("expected updated rating 5, got %d", found.Rating)
	}
	if found.Feedback != "Actually amazing!" {
		t.Fatalf("expected updated feedback, got %q", found.Feedback)
	}

	// Count — should be exactly one record
	var count int
	db.Table("training_satisfaction_ratings").
		Where("user_id = ? AND presentation_id = ?", 1, 100).
		Count(&count)
	if count != 1 {
		t.Fatalf("expected 1 rating record (upsert), got %d", count)
	}
}

func TestGetSatisfactionRatingNotFound(t *testing.T) {
	teardown := setupSatisfactionTest(t)
	defer teardown()

	_, err := GetSatisfactionRating(999, 999)
	if err == nil {
		t.Fatal("expected error for non-existent rating")
	}
}

// ---------- Satisfaction Stats ----------

func TestGetPresentationSatisfactionStats(t *testing.T) {
	teardown := setupSatisfactionTest(t)
	defer teardown()

	// Create several ratings for the same presentation
	ratings := []int{5, 5, 4, 3, 5}
	for i, r := range ratings {
		rating := &TrainingSatisfactionRating{
			UserId:         int64(i + 1),
			PresentationId: 100,
			Rating:         r,
		}
		PostTrainingSatisfactionRating(rating)
	}

	stats := GetPresentationSatisfactionStats(100)
	if stats.TotalRatings != 5 {
		t.Fatalf("expected 5 total ratings, got %d", stats.TotalRatings)
	}
	// Average: (5+5+4+3+5)/5 = 4.4
	if stats.AverageScore < 4.3 || stats.AverageScore > 4.5 {
		t.Fatalf("expected average ~4.4, got %.2f", stats.AverageScore)
	}
	if stats.Star5Count != 3 {
		t.Fatalf("expected 3 five-star ratings, got %d", stats.Star5Count)
	}
	if stats.Star4Count != 1 {
		t.Fatalf("expected 1 four-star rating, got %d", stats.Star4Count)
	}
	if stats.Star3Count != 1 {
		t.Fatalf("expected 1 three-star rating, got %d", stats.Star3Count)
	}
	if stats.Star2Count != 0 {
		t.Fatalf("expected 0 two-star ratings, got %d", stats.Star2Count)
	}
	if stats.Star1Count != 0 {
		t.Fatalf("expected 0 one-star ratings, got %d", stats.Star1Count)
	}
}

func TestGetPresentationSatisfactionStatsEmpty(t *testing.T) {
	teardown := setupSatisfactionTest(t)
	defer teardown()

	stats := GetPresentationSatisfactionStats(999)
	if stats.TotalRatings != 0 {
		t.Fatalf("expected 0 ratings for non-existent presentation, got %d", stats.TotalRatings)
	}
	if stats.AverageScore != 0 {
		t.Fatalf("expected 0 average for no ratings, got %.2f", stats.AverageScore)
	}
}

func TestGetOrgSatisfactionStats(t *testing.T) {
	teardown := setupSatisfactionTest(t)
	defer teardown()

	// Create a presentation in org 1
	tp := &TrainingPresentation{OrgId: 1, Name: "Rated Course", FileName: "r.pdf", FilePath: "/f"}
	PostTrainingPresentation(tp)

	// Add ratings
	for i, r := range []int{5, 4, 4, 3} {
		PostTrainingSatisfactionRating(&TrainingSatisfactionRating{
			UserId:         int64(i + 1),
			PresentationId: tp.Id,
			Rating:         r,
		})
	}

	stats := GetOrgSatisfactionStats(1)
	if stats.TotalRatings != 4 {
		t.Fatalf("expected 4 ratings, got %d", stats.TotalRatings)
	}
	// Average: (5+4+4+3)/4 = 4.0
	if stats.AverageScore < 3.9 || stats.AverageScore > 4.1 {
		t.Fatalf("expected average ~4.0, got %.2f", stats.AverageScore)
	}
}

func TestGetOrgSatisfactionStatsIsolation(t *testing.T) {
	teardown := setupSatisfactionTest(t)
	defer teardown()

	// Org 1 presentation
	tp1 := &TrainingPresentation{OrgId: 1, Name: "Org1 Course", FileName: "o1.pdf", FilePath: "/f"}
	PostTrainingPresentation(tp1)
	PostTrainingSatisfactionRating(&TrainingSatisfactionRating{UserId: 1, PresentationId: tp1.Id, Rating: 5})

	// Org 2 presentation
	tp2 := &TrainingPresentation{OrgId: 2, Name: "Org2 Course", FileName: "o2.pdf", FilePath: "/f"}
	PostTrainingPresentation(tp2)
	PostTrainingSatisfactionRating(&TrainingSatisfactionRating{UserId: 2, PresentationId: tp2.Id, Rating: 1})

	// Org 1 stats should not include Org 2 ratings
	stats := GetOrgSatisfactionStats(1)
	if stats.TotalRatings != 1 {
		t.Fatalf("expected 1 rating for org 1, got %d", stats.TotalRatings)
	}
	if stats.AverageScore < 4.9 {
		t.Fatalf("expected average ~5.0 for org 1, got %.2f", stats.AverageScore)
	}
}

// ---------- Training Analytics ----------

func TestGetTrainingAnalyticsBasic(t *testing.T) {
	teardown := setupSatisfactionTest(t)
	defer teardown()

	// Create course
	tp := &TrainingPresentation{OrgId: 1, Name: "Analytics Course", FileName: "a.pdf", FilePath: "/f"}
	PostTrainingPresentation(tp)

	// Create enrollments (progress)
	for _, uid := range []int64{1, 2, 3} {
		status := "in_progress"
		if uid == 1 {
			status = "complete"
		}
		SaveCourseProgress(&CourseProgress{
			UserId:         uid,
			PresentationId: tp.Id,
			Status:         status,
			TotalPages:     5,
		})
	}

	// Add a satisfaction rating
	PostTrainingSatisfactionRating(&TrainingSatisfactionRating{UserId: 1, PresentationId: tp.Id, Rating: 5})

	analytics := GetTrainingAnalytics(1)
	if analytics.TotalCourses != 1 {
		t.Fatalf("expected 1 course, got %d", analytics.TotalCourses)
	}
	if analytics.TotalEnrollments != 3 {
		t.Fatalf("expected 3 enrollments, got %d", analytics.TotalEnrollments)
	}
	// 1 of 3 completed = 33.3%
	if analytics.CompletionRate < 33.0 || analytics.CompletionRate > 34.0 {
		t.Fatalf("expected completion rate ~33.3%%, got %.1f%%", analytics.CompletionRate)
	}
	if analytics.Satisfaction.TotalRatings != 1 {
		t.Fatalf("expected 1 satisfaction rating, got %d", analytics.Satisfaction.TotalRatings)
	}
	if len(analytics.TopCourses) != 1 {
		t.Fatalf("expected 1 top course, got %d", len(analytics.TopCourses))
	}
	if analytics.TopCourses[0].Name != "Analytics Course" {
		t.Fatalf("unexpected course name: %q", analytics.TopCourses[0].Name)
	}
	if analytics.TopCourses[0].Enrollments != 3 {
		t.Fatalf("expected 3 enrollments in top course, got %d", analytics.TopCourses[0].Enrollments)
	}
	if analytics.TopCourses[0].Completions != 1 {
		t.Fatalf("expected 1 completion in top course, got %d", analytics.TopCourses[0].Completions)
	}
}

func TestGetTrainingAnalyticsEmpty(t *testing.T) {
	teardown := setupSatisfactionTest(t)
	defer teardown()

	analytics := GetTrainingAnalytics(999)
	if analytics.TotalCourses != 0 {
		t.Fatalf("expected 0 courses, got %d", analytics.TotalCourses)
	}
	if analytics.TotalEnrollments != 0 {
		t.Fatalf("expected 0 enrollments, got %d", analytics.TotalEnrollments)
	}
	if analytics.CompletionRate != 0 {
		t.Fatalf("expected 0 completion rate, got %.1f", analytics.CompletionRate)
	}
}

func TestGetTrainingAnalyticsQuizPassRate(t *testing.T) {
	teardown := setupSatisfactionTest(t)
	defer teardown()

	tp := &TrainingPresentation{OrgId: 1, Name: "Quiz Course", FileName: "qc.pdf", FilePath: "/f"}
	PostTrainingPresentation(tp)

	quiz := &Quiz{PresentationId: tp.Id, PassPercentage: 70, CreatedBy: 1}
	PostQuiz(quiz)

	// 2 passed, 1 failed
	for _, passed := range []bool{true, true, false} {
		attempt := &QuizAttempt{
			QuizId:         quiz.Id,
			UserId:         1,
			Score:          3,
			TotalQuestions: 5,
			Passed:         passed,
		}
		db.Save(attempt)
	}

	analytics := GetTrainingAnalytics(1)
	// 2/3 = 66.7%
	if analytics.QuizPassRate < 66.0 || analytics.QuizPassRate > 67.0 {
		t.Fatalf("expected quiz pass rate ~66.7%%, got %.1f%%", analytics.QuizPassRate)
	}
}
