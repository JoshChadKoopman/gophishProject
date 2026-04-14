package models

import (
	"testing"
)

// setupQuizTest initialises an in-memory database and returns a teardown func.
func setupQuizTest(t *testing.T) func() {
	t.Helper()
	teardown := setupTrainingTest(t)
	return teardown
}

// helperCreatePresentation creates a minimal TrainingPresentation for quiz attachment.
func helperCreatePresentation(t *testing.T) *TrainingPresentation {
	t.Helper()
	tp := &TrainingPresentation{
		OrgId:    1,
		Name:     "Quiz Test",
		FileName: "test.pdf",
		FilePath: "/test.pdf",
		FileSize: 100,
	}
	if err := PostTrainingPresentation(tp); err != nil {
		t.Fatalf("failed to create presentation: %v", err)
	}
	if tp.Id == 0 {
		t.Fatalf("expected presentation id to be set")
	}
	return tp
}

// ---------- PostQuiz + GetQuizByPresentationId ----------

func TestPostQuizAndGetByPresentationId(t *testing.T) {
	teardown := setupQuizTest(t)
	defer teardown()

	tp := helperCreatePresentation(t)

	q := &Quiz{
		PresentationId: tp.Id,
		PassPercentage: 80,
		CreatedBy:      1,
	}
	if err := PostQuiz(q); err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
	if q.Id == 0 {
		t.Fatalf("expected quiz id to be set after PostQuiz")
	}

	// Add questions
	questions := []QuizQuestion{
		{
			QuestionType: QuestionTypeMultipleChoice,
			QuestionText: "What is phishing?",
			Options:      `["A","B","C"]`,
			CorrectOption: 0,
			Explanation:  "Phishing is a social engineering attack.",
		},
		{
			QuestionType: QuestionTypeTrueFalse,
			QuestionText: "Phishing only happens via email.",
			Options:      `["True","False"]`,
			CorrectOption: 1,
			Explanation:  "Phishing can happen via SMS, voice, and more.",
		},
	}
	if err := SaveQuizQuestions(q.Id, questions); err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}

	// Retrieve
	got, err := GetQuizByPresentationId(tp.Id)
	if err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
	if got.Id != q.Id {
		t.Fatalf("expected quiz id %d, got %d", q.Id, got.Id)
	}
	if got.PassPercentage != 80 {
		t.Fatalf("expected pass_percentage 80, got %d", got.PassPercentage)
	}
	if len(got.Questions) != 2 {
		t.Fatalf("expected 2 questions, got %d", len(got.Questions))
	}
	// Verify sort order
	if got.Questions[0].QuestionText != "What is phishing?" {
		t.Fatalf("expected first question text 'What is phishing?', got '%s'", got.Questions[0].QuestionText)
	}
}

// ---------- SaveQuizQuestions ----------

func TestSaveQuizQuestionsReplacesExisting(t *testing.T) {
	teardown := setupQuizTest(t)
	defer teardown()

	tp := helperCreatePresentation(t)

	q := &Quiz{PresentationId: tp.Id, PassPercentage: 70, CreatedBy: 1}
	if err := PostQuiz(q); err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}

	// Save initial questions
	initial := []QuizQuestion{
		{QuestionType: QuestionTypeMultipleChoice, QuestionText: "Q1", Options: `["A","B"]`, CorrectOption: 0},
		{QuestionType: QuestionTypeMultipleChoice, QuestionText: "Q2", Options: `["A","B"]`, CorrectOption: 1},
	}
	if err := SaveQuizQuestions(q.Id, initial); err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}

	// Replace with a single new question
	replacement := []QuizQuestion{
		{QuestionType: QuestionTypeTrueFalse, QuestionText: "New Q1", Options: `["True","False"]`, CorrectOption: 0},
	}
	if err := SaveQuizQuestions(q.Id, replacement); err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}

	got, err := GetQuizByPresentationId(tp.Id)
	if err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
	if len(got.Questions) != 1 {
		t.Fatalf("expected 1 question after replacement, got %d", len(got.Questions))
	}
	if got.Questions[0].QuestionText != "New Q1" {
		t.Fatalf("expected question text 'New Q1', got '%s'", got.Questions[0].QuestionText)
	}
}

// ---------- PutQuiz ----------

func TestPutQuizUpdatesPassPercentage(t *testing.T) {
	teardown := setupQuizTest(t)
	defer teardown()

	tp := helperCreatePresentation(t)

	q := &Quiz{PresentationId: tp.Id, PassPercentage: 70, CreatedBy: 1}
	if err := PostQuiz(q); err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}

	q.PassPercentage = 90
	if err := PutQuiz(q); err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}

	got, err := GetQuizByPresentationId(tp.Id)
	if err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
	if got.PassPercentage != 90 {
		t.Fatalf("expected pass_percentage 90 after update, got %d", got.PassPercentage)
	}
}

// ---------- DeleteQuiz ----------

func TestDeleteQuizCascades(t *testing.T) {
	teardown := setupQuizTest(t)
	defer teardown()

	tp := helperCreatePresentation(t)

	q := &Quiz{PresentationId: tp.Id, PassPercentage: 80, CreatedBy: 1}
	if err := PostQuiz(q); err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}

	// Add questions
	questions := []QuizQuestion{
		{QuestionType: QuestionTypeMultipleChoice, QuestionText: "Q1", Options: `["A","B"]`, CorrectOption: 0},
	}
	if err := SaveQuizQuestions(q.Id, questions); err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}

	// Add an attempt
	attempt := &QuizAttempt{
		QuizId:         q.Id,
		UserId:         1,
		Score:          1,
		TotalQuestions: 1,
		Passed:         true,
		Answers:        `[{"question_id":1,"answer":[0]}]`,
	}
	if err := PostQuizAttempt(attempt); err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}

	// Delete the quiz by presentation ID
	if err := DeleteQuiz(tp.Id); err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}

	// Quiz should be gone
	_, err := GetQuizByPresentationId(tp.Id)
	if err == nil {
		t.Fatalf("expected error when getting deleted quiz, got nil")
	}

	// Questions should be gone
	var qCount int
	db.Model(&QuizQuestion{}).Where("quiz_id=?", q.Id).Count(&qCount)
	if qCount != 0 {
		t.Fatalf("expected 0 questions after delete, got %d", qCount)
	}

	// Attempts should be gone
	var aCount int
	db.Model(&QuizAttempt{}).Where("quiz_id=?", q.Id).Count(&aCount)
	if aCount != 0 {
		t.Fatalf("expected 0 attempts after delete, got %d", aCount)
	}
}

// ---------- PostQuizAttempt + GetQuizAttempts ----------

func TestPostQuizAttemptAndGetQuizAttempts(t *testing.T) {
	teardown := setupQuizTest(t)
	defer teardown()

	tp := helperCreatePresentation(t)

	q := &Quiz{PresentationId: tp.Id, PassPercentage: 80, CreatedBy: 1}
	if err := PostQuiz(q); err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}

	a1 := &QuizAttempt{QuizId: q.Id, UserId: 10, Score: 3, TotalQuestions: 5, Passed: false}
	if err := PostQuizAttempt(a1); err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}

	a2 := &QuizAttempt{QuizId: q.Id, UserId: 10, Score: 5, TotalQuestions: 5, Passed: true}
	if err := PostQuizAttempt(a2); err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}

	attempts, err := GetQuizAttempts(10, q.Id)
	if err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
	if len(attempts) != 2 {
		t.Fatalf("expected 2 attempts, got %d", len(attempts))
	}

	// Different user should get no attempts
	otherAttempts, err := GetQuizAttempts(99, q.Id)
	if err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
	if len(otherAttempts) != 0 {
		t.Fatalf("expected 0 attempts for other user, got %d", len(otherAttempts))
	}
}

// ---------- GetLatestPassedAttempt ----------

func TestGetLatestPassedAttempt(t *testing.T) {
	teardown := setupQuizTest(t)
	defer teardown()

	tp := helperCreatePresentation(t)

	q := &Quiz{PresentationId: tp.Id, PassPercentage: 80, CreatedBy: 1}
	if err := PostQuiz(q); err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}

	// Failing attempt
	fail := &QuizAttempt{QuizId: q.Id, UserId: 10, Score: 1, TotalQuestions: 5, Passed: false}
	if err := PostQuizAttempt(fail); err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}

	// Passing attempt
	pass := &QuizAttempt{QuizId: q.Id, UserId: 10, Score: 5, TotalQuestions: 5, Passed: true}
	if err := PostQuizAttempt(pass); err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}

	got, err := GetLatestPassedAttempt(10, q.Id)
	if err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}
	if !got.Passed {
		t.Fatalf("expected latest passed attempt to have Passed=true")
	}
	if got.Score != 5 {
		t.Fatalf("expected score 5, got %d", got.Score)
	}

	// User with no passing attempt should get error
	_, err = GetLatestPassedAttempt(99, q.Id)
	if err == nil {
		t.Fatalf("expected error for user with no passing attempt, got nil")
	}
}

// ---------- QuizExistsForPresentation ----------

func TestQuizExistsForPresentation(t *testing.T) {
	teardown := setupQuizTest(t)
	defer teardown()

	tp := helperCreatePresentation(t)

	// Before creating a quiz
	if QuizExistsForPresentation(tp.Id) {
		t.Fatalf("expected QuizExistsForPresentation to return false before quiz creation")
	}

	q := &Quiz{PresentationId: tp.Id, PassPercentage: 80, CreatedBy: 1}
	if err := PostQuiz(q); err != nil {
		t.Fatalf(fmtUnexpectedErr, err)
	}

	// After creating a quiz
	if !QuizExistsForPresentation(tp.Id) {
		t.Fatalf("expected QuizExistsForPresentation to return true after quiz creation")
	}

	// Non-existent presentation
	if QuizExistsForPresentation(99999) {
		t.Fatalf("expected QuizExistsForPresentation to return false for non-existent presentation")
	}
}

// ---------- GetCorrectOptionsSet ----------

func TestGetCorrectOptionsSetWithJSON(t *testing.T) {
	q := &QuizQuestion{
		QuestionType:   QuestionTypeMultiSelect,
		CorrectOptions: `[0,2]`,
		CorrectOption:  0,
	}
	set := q.GetCorrectOptionsSet()
	if len(set) != 2 {
		t.Fatalf("expected 2 entries in correct options set, got %d", len(set))
	}
	if !set[0] {
		t.Fatalf("expected index 0 in correct options set")
	}
	if !set[2] {
		t.Fatalf("expected index 2 in correct options set")
	}
	if set[1] {
		t.Fatalf("did not expect index 1 in correct options set")
	}
}

func TestGetCorrectOptionsSetFallback(t *testing.T) {
	q := &QuizQuestion{
		QuestionType:   QuestionTypeMultipleChoice,
		CorrectOptions: "",
		CorrectOption:  1,
	}
	set := q.GetCorrectOptionsSet()
	if len(set) != 1 {
		t.Fatalf("expected 1 entry in fallback correct options set, got %d", len(set))
	}
	if !set[1] {
		t.Fatalf("expected index 1 in fallback correct options set")
	}
}
