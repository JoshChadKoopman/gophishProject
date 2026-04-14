package models

import (
	"testing"
	"time"

	"github.com/gophish/gophish/config"
)

// setupRemediationTest initialises an in-memory DB for remediation tests.
func setupRemediationTest(t *testing.T) func() {
	t.Helper()
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	if err := Setup(conf); err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	db.Exec("DELETE FROM remediation_steps")
	db.Exec("DELETE FROM remediation_paths")
	db.Exec("DELETE FROM course_assignments")
	return func() {
		db.Exec("DELETE FROM remediation_steps")
		db.Exec("DELETE FROM remediation_paths")
		db.Exec("DELETE FROM course_assignments")
	}
}

func createRemTestPresentation(t *testing.T, name string) *TrainingPresentation {
	t.Helper()
	tp := &TrainingPresentation{
		OrgId: 1, Name: name, FileName: name + ".pdf",
		FilePath: "/uploads/" + name + ".pdf", FileSize: 1024,
	}
	if err := PostTrainingPresentation(tp); err != nil {
		t.Fatalf("create presentation: %v", err)
	}
	return tp
}

// ---- DetermineRiskLevel ----

func TestDetermineRiskLevel_Low(t *testing.T) {
	if got := DetermineRiskLevel(1); got != RiskLevelLow {
		t.Fatalf("expected low, got %s", got)
	}
}

func TestDetermineRiskLevel_Medium(t *testing.T) {
	if got := DetermineRiskLevel(3); got != RiskLevelMedium {
		t.Fatalf("expected medium, got %s", got)
	}
}

func TestDetermineRiskLevel_High(t *testing.T) {
	if got := DetermineRiskLevel(5); got != RiskLevelHigh {
		t.Fatalf("expected high, got %s", got)
	}
}

func TestDetermineRiskLevel_Critical(t *testing.T) {
	if got := DetermineRiskLevel(10); got != RiskLevelCritical {
		t.Fatalf("expected critical, got %s", got)
	}
}

// ---- PostRemediationPath ----

func TestPostRemediationPath(t *testing.T) {
	teardown := setupRemediationTest(t)
	defer teardown()

	tp := createRemTestPresentation(t, "Remediation Course 1")
	path := &RemediationPath{
		OrgId: 1, UserId: 1, UserEmail: "offender@test.com",
		Name: "Test Remediation", FailCount: 5,
	}
	err := PostRemediationPath(path, []int64{tp.Id})
	if err != nil {
		t.Fatalf("PostRemediationPath: %v", err)
	}
	if path.Id == 0 {
		t.Fatal("expected non-zero ID")
	}
	if path.Status != RemediationStatusActive {
		t.Fatalf("expected active status, got %s", path.Status)
	}
	if path.RiskLevel != RiskLevelHigh {
		t.Fatalf("expected high risk (5 fails), got %s", path.RiskLevel)
	}
	if path.TotalCourses != 1 {
		t.Fatalf("expected 1 total course, got %d", path.TotalCourses)
	}
}

func TestPostRemediationPath_NoCourses(t *testing.T) {
	teardown := setupRemediationTest(t)
	defer teardown()

	path := &RemediationPath{
		OrgId: 1, UserId: 1, Name: "Empty Path",
	}
	err := PostRemediationPath(path, []int64{})
	if err == nil {
		t.Fatal("expected error for empty course list")
	}
}

func TestPostRemediationPath_NoName(t *testing.T) {
	teardown := setupRemediationTest(t)
	defer teardown()

	path := &RemediationPath{OrgId: 1, UserId: 1}
	err := PostRemediationPath(path, []int64{1})
	if err == nil {
		t.Fatal("expected error for missing name")
	}
}

func TestPostRemediationPath_MultipleCourses(t *testing.T) {
	teardown := setupRemediationTest(t)
	defer teardown()

	tp1 := createRemTestPresentation(t, "Course A")
	tp2 := createRemTestPresentation(t, "Course B")
	tp3 := createRemTestPresentation(t, "Course C")

	path := &RemediationPath{
		OrgId: 1, UserId: 1, UserEmail: "multi@test.com",
		Name: "Multi-Course Path", FailCount: 9,
	}
	err := PostRemediationPath(path, []int64{tp1.Id, tp2.Id, tp3.Id})
	if err != nil {
		t.Fatalf("PostRemediationPath: %v", err)
	}
	if path.TotalCourses != 3 {
		t.Fatalf("expected 3 courses, got %d", path.TotalCourses)
	}
	if path.RiskLevel != RiskLevelCritical {
		t.Fatalf("expected critical risk (9 fails), got %s", path.RiskLevel)
	}
}

func TestPostRemediationPath_CreatesAssignments(t *testing.T) {
	teardown := setupRemediationTest(t)
	defer teardown()

	tp := createRemTestPresentation(t, "Assignment Course")
	path := &RemediationPath{
		OrgId: 1, UserId: 1, Name: "With Assignment", FailCount: 6,
	}
	err := PostRemediationPath(path, []int64{tp.Id})
	if err != nil {
		t.Fatalf("PostRemediationPath: %v", err)
	}

	a, err := GetAssignment(1, tp.Id)
	if err != nil {
		t.Fatalf("expected assignment to be created: %v", err)
	}
	if a.Priority != AssignmentPriorityHigh {
		t.Fatalf("expected high priority for high-risk path, got %s", a.Priority)
	}
}

// ---- GetRemediationPath ----

func TestGetRemediationPath(t *testing.T) {
	teardown := setupRemediationTest(t)
	defer teardown()

	tp := createRemTestPresentation(t, "Get Test")
	path := &RemediationPath{
		OrgId: 1, UserId: 1, Name: "Fetch Me",
	}
	PostRemediationPath(path, []int64{tp.Id})

	fetched, err := GetRemediationPath(path.Id, 1)
	if err != nil {
		t.Fatalf("GetRemediationPath: %v", err)
	}
	if fetched.Name != "Fetch Me" {
		t.Fatalf("wrong name: %s", fetched.Name)
	}
	if len(fetched.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(fetched.Steps))
	}
}

func TestGetRemediationPath_NotFound(t *testing.T) {
	teardown := setupRemediationTest(t)
	defer teardown()

	_, err := GetRemediationPath(999, 1)
	if err != ErrRemediationNotFound {
		t.Fatalf("expected ErrRemediationNotFound, got %v", err)
	}
}

// ---- GetRemediationPathsForUser ----

func TestGetRemediationPathsForUser(t *testing.T) {
	teardown := setupRemediationTest(t)
	defer teardown()

	tp := createRemTestPresentation(t, "User Path")
	p1 := &RemediationPath{OrgId: 1, UserId: 1, Name: "Path 1"}
	PostRemediationPath(p1, []int64{tp.Id})

	paths, err := GetRemediationPathsForUser(1, 1)
	if err != nil {
		t.Fatalf("GetRemediationPathsForUser: %v", err)
	}
	if len(paths) != 1 {
		t.Fatalf("expected 1 path, got %d", len(paths))
	}
}

// ---- CompleteRemediationStep ----

func TestCompleteRemediationStep(t *testing.T) {
	teardown := setupRemediationTest(t)
	defer teardown()

	tp := createRemTestPresentation(t, "Complete Step")
	path := &RemediationPath{
		OrgId: 1, UserId: 1, Name: "Step Test",
	}
	PostRemediationPath(path, []int64{tp.Id})

	err := CompleteRemediationStep(path.Id, tp.Id)
	if err != nil {
		t.Fatalf("CompleteRemediationStep: %v", err)
	}

	updated, _ := GetRemediationPath(path.Id, 1)
	if updated.CompletedCount != 1 {
		t.Fatalf("expected completed_count 1, got %d", updated.CompletedCount)
	}
	if updated.Status != RemediationStatusCompleted {
		t.Fatalf("expected path completed (single course), got %s", updated.Status)
	}
}

func TestCompleteRemediationStep_Partial(t *testing.T) {
	teardown := setupRemediationTest(t)
	defer teardown()

	tp1 := createRemTestPresentation(t, "Step 1")
	tp2 := createRemTestPresentation(t, "Step 2")
	path := &RemediationPath{
		OrgId: 1, UserId: 1, Name: "Partial Test",
	}
	PostRemediationPath(path, []int64{tp1.Id, tp2.Id})

	CompleteRemediationStep(path.Id, tp1.Id)

	updated, _ := GetRemediationPath(path.Id, 1)
	if updated.CompletedCount != 1 {
		t.Fatalf("expected 1 completed, got %d", updated.CompletedCount)
	}
	if updated.Status != RemediationStatusActive {
		t.Fatalf("expected still active, got %s", updated.Status)
	}

	// Complete second
	CompleteRemediationStep(path.Id, tp2.Id)
	updated, _ = GetRemediationPath(path.Id, 1)
	if updated.Status != RemediationStatusCompleted {
		t.Fatalf("expected completed, got %s", updated.Status)
	}
}

func TestCompleteRemediationStep_NotFound(t *testing.T) {
	teardown := setupRemediationTest(t)
	defer teardown()

	err := CompleteRemediationStep(999, 888)
	if err != ErrRemediationStepMissing {
		t.Fatalf("expected ErrRemediationStepMissing, got %v", err)
	}
}

func TestCompleteRemediationStep_Idempotent(t *testing.T) {
	teardown := setupRemediationTest(t)
	defer teardown()

	tp := createRemTestPresentation(t, "Idempotent")
	path := &RemediationPath{OrgId: 1, UserId: 1, Name: "Idempotent Test"}
	PostRemediationPath(path, []int64{tp.Id})

	CompleteRemediationStep(path.Id, tp.Id)
	err := CompleteRemediationStep(path.Id, tp.Id)
	if err != nil {
		t.Fatalf("second completion should be no-op, got %v", err)
	}
}

// ---- CancelRemediationPath ----

func TestCancelRemediationPath(t *testing.T) {
	teardown := setupRemediationTest(t)
	defer teardown()

	tp := createRemTestPresentation(t, "Cancel Test")
	path := &RemediationPath{OrgId: 1, UserId: 1, Name: "Cancel Me"}
	PostRemediationPath(path, []int64{tp.Id})

	err := CancelRemediationPath(path.Id, 1)
	if err != nil {
		t.Fatalf("CancelRemediationPath: %v", err)
	}

	cancelled, _ := GetRemediationPath(path.Id, 1)
	if cancelled.Status != RemediationStatusCancelled {
		t.Fatalf("expected cancelled, got %s", cancelled.Status)
	}
	for _, s := range cancelled.Steps {
		if s.Status != StepStatusSkipped {
			t.Fatalf("expected step skipped, got %s", s.Status)
		}
	}
}

func TestCancelRemediationPath_NotFound(t *testing.T) {
	teardown := setupRemediationTest(t)
	defer teardown()

	err := CancelRemediationPath(999, 1)
	if err != ErrRemediationNotFound {
		t.Fatalf("expected not found, got %v", err)
	}
}

// ---- GetRemediationSummary ----

func TestGetRemediationSummary(t *testing.T) {
	teardown := setupRemediationTest(t)
	defer teardown()

	tp := createRemTestPresentation(t, "Summary")
	p1 := &RemediationPath{OrgId: 1, UserId: 1, Name: "Active", FailCount: 3}
	PostRemediationPath(p1, []int64{tp.Id})

	summary, err := GetRemediationSummary(1)
	if err != nil {
		t.Fatalf("GetRemediationSummary: %v", err)
	}
	if summary.TotalPaths != 1 {
		t.Fatalf("expected 1 total, got %d", summary.TotalPaths)
	}
	if summary.ActivePaths != 1 {
		t.Fatalf("expected 1 active, got %d", summary.ActivePaths)
	}
}

func TestGetRemediationSummary_Empty(t *testing.T) {
	teardown := setupRemediationTest(t)
	defer teardown()

	summary, err := GetRemediationSummary(1)
	if err != nil {
		t.Fatalf("GetRemediationSummary: %v", err)
	}
	if summary.TotalPaths != 0 {
		t.Fatalf("expected 0 paths, got %d", summary.TotalPaths)
	}
}

// ---- MarkExpiredRemediationPaths ----

func TestMarkExpiredRemediationPaths(t *testing.T) {
	teardown := setupRemediationTest(t)
	defer teardown()

	tp := createRemTestPresentation(t, "Expired")
	path := &RemediationPath{
		OrgId: 1, UserId: 1, Name: "Expired Path",
		DueDate: time.Now().UTC().Add(-24 * time.Hour),
	}
	PostRemediationPath(path, []int64{tp.Id})

	count, err := MarkExpiredRemediationPaths(1)
	if err != nil {
		t.Fatalf("MarkExpiredRemediationPaths: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 expired, got %d", count)
	}

	updated, _ := GetRemediationPath(path.Id, 1)
	if updated.Status != RemediationStatusExpired {
		t.Fatalf("expected expired, got %s", updated.Status)
	}
}

func TestMarkExpiredRemediationPaths_NoEffect(t *testing.T) {
	teardown := setupRemediationTest(t)
	defer teardown()

	tp := createRemTestPresentation(t, "Future")
	path := &RemediationPath{
		OrgId: 1, UserId: 1, Name: "Future Path",
		DueDate: time.Now().UTC().Add(7 * 24 * time.Hour),
	}
	PostRemediationPath(path, []int64{tp.Id})

	count, _ := MarkExpiredRemediationPaths(1)
	if count != 0 {
		t.Fatalf("expected 0 expired for future path, got %d", count)
	}
}

// ---- mapRiskToPriority ----

func TestMapRiskToPriority(t *testing.T) {
	cases := []struct {
		risk     string
		expected string
	}{
		{RiskLevelCritical, AssignmentPriorityCritical},
		{RiskLevelHigh, AssignmentPriorityHigh},
		{RiskLevelMedium, AssignmentPriorityNormal},
		{RiskLevelLow, AssignmentPriorityLow},
		{"unknown", AssignmentPriorityLow},
	}
	for _, c := range cases {
		got := mapRiskToPriority(c.risk)
		if got != c.expected {
			t.Errorf("mapRiskToPriority(%s) = %s, want %s", c.risk, got, c.expected)
		}
	}
}

// ---- GetRemediationPaths (list) ----

func TestGetRemediationPaths_OrgScoped(t *testing.T) {
	teardown := setupRemediationTest(t)
	defer teardown()

	tp := createRemTestPresentation(t, "Scoped")
	p1 := &RemediationPath{OrgId: 1, UserId: 1, Name: "Org 1 Path"}
	PostRemediationPath(p1, []int64{tp.Id})

	paths, err := GetRemediationPaths(1)
	if err != nil {
		t.Fatalf("GetRemediationPaths: %v", err)
	}
	if len(paths) != 1 {
		t.Fatalf("expected 1 path for org 1, got %d", len(paths))
	}

	paths2, _ := GetRemediationPaths(999)
	if len(paths2) != 0 {
		t.Fatalf("expected 0 paths for org 999, got %d", len(paths2))
	}
}

// ---- Escalation integration ----

func TestCreateRemediationFromEscalation(t *testing.T) {
	teardown := setupRemediationTest(t)
	defer teardown()

	tp := createRemTestPresentation(t, "Escalation Course")
	event := EscalationEvent{
		Id: 1, OrgId: 1, UserId: 1, UserEmail: "repeat@test.com",
		Level: 2, FailCount: 6, Action: EscalationActionTraining,
	}
	path, err := CreateRemediationFromEscalation(1, event, []int64{tp.Id})
	if err != nil {
		t.Fatalf("CreateRemediationFromEscalation: %v", err)
	}
	if path.RiskLevel != RiskLevelHigh {
		t.Fatalf("expected high risk for 6 fails, got %s", path.RiskLevel)
	}
	if path.EscalationEvent != 1 {
		t.Fatalf("expected escalation_event_id = 1, got %d", path.EscalationEvent)
	}
}

func TestCreateRemediationFromEscalation_NoCourses(t *testing.T) {
	teardown := setupRemediationTest(t)
	defer teardown()

	event := EscalationEvent{Id: 1, FailCount: 3}
	_, err := CreateRemediationFromEscalation(1, event, []int64{})
	if err == nil {
		t.Fatal("expected error for empty course list")
	}
}

func TestCreateRemediationFromEscalation_Critical(t *testing.T) {
	teardown := setupRemediationTest(t)
	defer teardown()

	tp := createRemTestPresentation(t, "Critical Course")
	event := EscalationEvent{
		Id: 2, OrgId: 1, UserId: 1, UserEmail: "critical@test.com",
		Level: 3, FailCount: 10,
	}
	path, err := CreateRemediationFromEscalation(1, event, []int64{tp.Id})
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if path.RiskLevel != RiskLevelCritical {
		t.Fatalf("expected critical risk, got %s", path.RiskLevel)
	}
	// Critical due date should be ~7 days, not 14
	diff := path.DueDate.Sub(time.Now().UTC())
	if diff > 8*24*time.Hour || diff < 6*24*time.Hour {
		t.Fatalf("expected ~7 day due date for critical, got %v", diff)
	}
}

// ---- Hygiene steps hydration ----

func TestRemediationStepCourseName(t *testing.T) {
	teardown := setupRemediationTest(t)
	defer teardown()

	tp := createRemTestPresentation(t, "Named Course")
	path := &RemediationPath{OrgId: 1, UserId: 1, Name: "Hydration Test"}
	PostRemediationPath(path, []int64{tp.Id})

	fetched, _ := GetRemediationPath(path.Id, 1)
	if len(fetched.Steps) != 1 {
		t.Fatalf("expected 1 step, got %d", len(fetched.Steps))
	}
	if fetched.Steps[0].CourseName != "Named Course" {
		t.Fatalf("expected course name 'Named Course', got %s", fetched.Steps[0].CourseName)
	}
}
