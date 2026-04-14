package models

import (
	"testing"

	"github.com/gophish/gophish/config"
)

// fmtUnexpectedErr is the shared format string for unexpected errors in tests.
const fmtUnexpectedErr = "unexpected error: %v"

// setupTrainingTest initialises an in-memory database and returns a teardown func.
func setupTrainingTest(t *testing.T) func() {
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
	db.Exec("DELETE FROM page_engagement")
	db.Exec("DELETE FROM anti_skip_policy")
	db.Exec("DELETE FROM academy_sessions")
	db.Exec("DELETE FROM academy_user_progress")
	db.Exec("DELETE FROM academy_tiers")
	db.Exec("DELETE FROM training_satisfaction_ratings")
	db.Exec("DELETE FROM course_progress")
	db.Exec("DELETE FROM quiz_attempts")
	db.Exec("DELETE FROM quiz_questions")
	db.Exec("DELETE FROM quizzes")
	db.Exec("DELETE FROM course_assignments")
	db.Exec("DELETE FROM certificates")
	db.Exec("DELETE FROM training_presentations")
	return func() {
		db.Exec("DELETE FROM page_engagement")
		db.Exec("DELETE FROM anti_skip_policy")
		db.Exec("DELETE FROM academy_sessions")
		db.Exec("DELETE FROM academy_user_progress")
		db.Exec("DELETE FROM academy_tiers")
		db.Exec("DELETE FROM training_satisfaction_ratings")
		db.Exec("DELETE FROM course_progress")
		db.Exec("DELETE FROM quiz_attempts")
		db.Exec("DELETE FROM quiz_questions")
		db.Exec("DELETE FROM quizzes")
		db.Exec("DELETE FROM course_assignments")
		db.Exec("DELETE FROM certificates")
		db.Exec("DELETE FROM training_presentations")
	}
}

// ---------- TrainingPresentation CRUD ----------

func TestPostTrainingPresentationSuccess(t *testing.T) {
	teardown := setupTrainingTest(t)
	defer teardown()

	tp := &TrainingPresentation{
		OrgId:    1,
		Name:     "Phishing 101",
		FileName: "phishing.pdf",
		FilePath: "/uploads/phishing.pdf",
		FileSize: 1024,
	}
	if err := PostTrainingPresentation(tp); err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
	if tp.Id == 0 {
		t.Fatal("expected non-zero ID after save")
	}
	if tp.CreatedDate.IsZero() {
		t.Fatal("expected CreatedDate to be set")
	}
}

func TestPostTrainingPresentationValidationNoName(t *testing.T) {
	teardown := setupTrainingTest(t)
	defer teardown()

	tp := &TrainingPresentation{
		FileName: "file.pdf",
	}
	err := PostTrainingPresentation(tp)
	if err != ErrTrainingNameNotSpecified {
		t.Fatalf("expected ErrTrainingNameNotSpecified, got %v", err)
	}
}

func TestPostTrainingPresentationValidationNoFile(t *testing.T) {
	teardown := setupTrainingTest(t)
	defer teardown()

	tp := &TrainingPresentation{
		Name: "Test",
	}
	err := PostTrainingPresentation(tp)
	if err != ErrTrainingFileNotSpecified {
		t.Fatalf("expected ErrTrainingFileNotSpecified, got %v", err)
	}
}

func TestGetTrainingPresentations(t *testing.T) {
	teardown := setupTrainingTest(t)
	defer teardown()

	scope := OrgScope{OrgId: 1, UserId: 1, IsSuperAdmin: true}

	// Initially empty
	tps, err := GetTrainingPresentations(scope)
	if err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
	if len(tps) != 0 {
		t.Fatalf("expected 0 presentations, got %d", len(tps))
	}

	// Create two presentations
	for _, name := range []string{"Course A", "Course B"} {
		tp := &TrainingPresentation{OrgId: 1, Name: name, FileName: name + ".pdf", FilePath: "/f"}
		if err := PostTrainingPresentation(tp); err != nil {
			t.Fatalf("failed to create %s: %v", name, err)
		}
	}

	tps, err = GetTrainingPresentations(scope)
	if err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
	if len(tps) != 2 {
		t.Fatalf("expected 2 presentations, got %d", len(tps))
	}
}

func TestGetTrainingPresentationById(t *testing.T) {
	teardown := setupTrainingTest(t)
	defer teardown()

	scope := OrgScope{OrgId: 1, UserId: 1, IsSuperAdmin: true}
	tp := &TrainingPresentation{OrgId: 1, Name: "Test", FileName: "t.pdf", FilePath: "/f"}
	PostTrainingPresentation(tp)

	found, err := GetTrainingPresentation(tp.Id, scope)
	if err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
	if found.Name != "Test" {
		t.Fatalf("expected name 'Test', got %q", found.Name)
	}
}

func TestPutTrainingPresentation(t *testing.T) {
	teardown := setupTrainingTest(t)
	defer teardown()

	tp := &TrainingPresentation{OrgId: 1, Name: "Original", FileName: "o.pdf", FilePath: "/f"}
	PostTrainingPresentation(tp)

	tp.Name = "Updated"
	if err := PutTrainingPresentation(tp); err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}

	scope := OrgScope{OrgId: 1, UserId: 1, IsSuperAdmin: true}
	found, _ := GetTrainingPresentation(tp.Id, scope)
	if found.Name != "Updated" {
		t.Fatalf("expected name 'Updated', got %q", found.Name)
	}
}

func TestDeleteTrainingPresentation(t *testing.T) {
	teardown := setupTrainingTest(t)
	defer teardown()

	scope := OrgScope{OrgId: 1, UserId: 1, IsSuperAdmin: true}
	tp := &TrainingPresentation{OrgId: 1, Name: "Delete Me", FileName: "d.pdf", FilePath: "/f"}
	PostTrainingPresentation(tp)

	if err := DeleteTrainingPresentation(tp.Id); err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}

	_, err := GetTrainingPresentation(tp.Id, scope)
	if err == nil {
		t.Fatal("expected error fetching deleted presentation")
	}
}

// ---------- CourseProgress ----------

func TestSaveCourseProgressNew(t *testing.T) {
	teardown := setupTrainingTest(t)
	defer teardown()

	cp := &CourseProgress{
		UserId:         1,
		PresentationId: 100,
		CurrentPage:    0,
		TotalPages:     5,
		Status:         "in_progress",
	}
	if err := SaveCourseProgress(cp); err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
	if cp.Id == 0 {
		t.Fatal("expected non-zero ID")
	}
	if cp.CreatedDate.IsZero() {
		t.Fatal("expected CreatedDate to be set")
	}
}

func TestGetCourseProgress(t *testing.T) {
	teardown := setupTrainingTest(t)
	defer teardown()

	cp := &CourseProgress{UserId: 1, PresentationId: 100, Status: "in_progress", TotalPages: 5}
	SaveCourseProgress(cp)

	found, err := GetCourseProgress(1, 100)
	if err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
	if found.Status != "in_progress" {
		t.Fatalf("expected status 'in_progress', got %q", found.Status)
	}
}

func TestGetCourseProgressNotFound(t *testing.T) {
	teardown := setupTrainingTest(t)
	defer teardown()

	_, err := GetCourseProgress(999, 999)
	if err == nil {
		t.Fatal("expected error for non-existent progress")
	}
}

func TestSaveCourseProgressUpdate(t *testing.T) {
	teardown := setupTrainingTest(t)
	defer teardown()

	cp := &CourseProgress{UserId: 1, PresentationId: 100, Status: "in_progress", CurrentPage: 1, TotalPages: 5}
	SaveCourseProgress(cp)

	// Update progress
	cp.CurrentPage = 3
	cp.Status = "in_progress"
	if err := SaveCourseProgress(cp); err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}

	found, _ := GetCourseProgress(1, 100)
	if found.CurrentPage != 3 {
		t.Fatalf("expected current_page 3, got %d", found.CurrentPage)
	}
}

func TestGetUserCourseProgress(t *testing.T) {
	teardown := setupTrainingTest(t)
	defer teardown()

	// Create progress for two courses
	for _, pid := range []int64{100, 200} {
		cp := &CourseProgress{UserId: 1, PresentationId: pid, Status: "in_progress", TotalPages: 5}
		SaveCourseProgress(cp)
	}

	all, err := GetUserCourseProgress(1)
	if err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 progress records, got %d", len(all))
	}
}

func TestCourseCompletionFlow(t *testing.T) {
	teardown := setupTrainingTest(t)
	defer teardown()

	tp := &TrainingPresentation{OrgId: 1, Name: "Complete Me", FileName: "c.pdf", FilePath: "/f"}
	PostTrainingPresentation(tp)

	// Start course
	cp := &CourseProgress{UserId: 1, PresentationId: tp.Id, Status: "in_progress", CurrentPage: 0, TotalPages: 3}
	SaveCourseProgress(cp)

	// Navigate through pages
	cp.CurrentPage = 1
	SaveCourseProgress(cp)
	cp.CurrentPage = 2
	SaveCourseProgress(cp)

	// Complete
	cp.CurrentPage = 3
	cp.Status = "complete"
	SaveCourseProgress(cp)

	found, _ := GetCourseProgress(1, tp.Id)
	if found.Status != "complete" {
		t.Fatalf("expected status 'complete', got %q", found.Status)
	}
	if found.CurrentPage != 3 {
		t.Fatalf("expected current_page 3, got %d", found.CurrentPage)
	}
}

// ---------- Quiz CRUD ----------

func TestQuizCRUD(t *testing.T) {
	teardown := setupTrainingTest(t)
	defer teardown()

	tp := &TrainingPresentation{OrgId: 1, Name: "Quiz Course", FileName: "q.pdf", FilePath: "/f"}
	PostTrainingPresentation(tp)

	// Create quiz
	quiz := &Quiz{PresentationId: tp.Id, PassPercentage: 70, CreatedBy: 1}
	if err := PostQuiz(quiz); err != nil {
		t.Fatalf("failed to create quiz: %v", err)
	}
	if quiz.Id == 0 {
		t.Fatal("expected non-zero quiz ID")
	}

	// Add questions
	questions := []QuizQuestion{
		{QuestionText: "Q1?", Options: `["A","B","C","D"]`, CorrectOption: 0},
		{QuestionText: "Q2?", Options: `["X","Y","Z"]`, CorrectOption: 2},
	}
	if err := SaveQuizQuestions(quiz.Id, questions); err != nil {
		t.Fatalf("failed to save questions: %v", err)
	}

	// Retrieve
	found, err := GetQuizByPresentationId(tp.Id)
	if err != nil {
		t.Fatalf("failed to get quiz: %v", err)
	}
	if found.PassPercentage != 70 {
		t.Fatalf("expected pass_percentage 70, got %d", found.PassPercentage)
	}
	if len(found.Questions) != 2 {
		t.Fatalf("expected 2 questions, got %d", len(found.Questions))
	}

	// Delete quiz
	if err := DeleteQuiz(tp.Id); err != nil {
		t.Fatalf("failed to delete quiz: %v", err)
	}
	_, err = GetQuizByPresentationId(tp.Id)
	if err == nil {
		t.Fatal("expected error fetching deleted quiz")
	}
}

// ---------- Delete cascade ----------

func TestDeleteTrainingPresentationCascade(t *testing.T) {
	teardown := setupTrainingTest(t)
	defer teardown()

	tp := &TrainingPresentation{OrgId: 1, Name: "Cascade", FileName: "c.pdf", FilePath: "/f"}
	PostTrainingPresentation(tp)

	// Create quiz
	quiz := &Quiz{PresentationId: tp.Id, PassPercentage: 80, CreatedBy: 1}
	PostQuiz(quiz)
	SaveQuizQuestions(quiz.Id, []QuizQuestion{
		{QuestionText: "Q?", Options: `["A","B"]`, CorrectOption: 0},
	})

	// Create progress
	cp := &CourseProgress{UserId: 1, PresentationId: tp.Id, Status: "complete", TotalPages: 1}
	SaveCourseProgress(cp)

	// Delete presentation — should cascade
	if err := DeleteTrainingPresentation(tp.Id); err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}

	// Verify quiz is gone
	_, err := GetQuizByPresentationId(tp.Id)
	if err == nil {
		t.Fatal("quiz should have been deleted")
	}

	// Verify progress is gone
	_, err = GetCourseProgress(1, tp.Id)
	if err == nil {
		t.Fatal("progress should have been deleted")
	}
}
