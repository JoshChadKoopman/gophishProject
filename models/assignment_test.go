package models

import (
	"testing"
	"time"

	"github.com/gophish/gophish/config"
)

// setupAssignmentTest initialises an in-memory DB for assignment tests.
func setupAssignmentTest(t *testing.T) func() {
	t.Helper()
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	if err := Setup(conf); err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	db.Exec("DELETE FROM course_assignments")
	db.Exec("DELETE FROM certificates")
	return func() {
		db.Exec("DELETE FROM course_assignments")
		db.Exec("DELETE FROM certificates")
	}
}

// createTestPresentation creates a minimal training presentation for tests.
func createTestPresentation(t *testing.T, name string) *TrainingPresentation {
	t.Helper()
	tp := &TrainingPresentation{
		OrgId:    1,
		Name:     name,
		FileName: name + ".pdf",
		FilePath: "/uploads/" + name + ".pdf",
		FileSize: 1024,
	}
	if err := PostTrainingPresentation(tp); err != nil {
		t.Fatalf("failed to create presentation %s: %v", name, err)
	}
	return tp
}

// =====================================================================
// PostAssignment
// =====================================================================

func TestPostAssignment(t *testing.T) {
	teardown := setupAssignmentTest(t)
	defer teardown()

	a := &CourseAssignment{
		UserId:         1,
		PresentationId: 100,
		AssignedBy:     2,
		DueDate:        time.Now().UTC().Add(72 * time.Hour),
	}
	if err := PostAssignment(a); err != nil {
		t.Fatalf("PostAssignment failed: %v", err)
	}
	if a.Id == 0 {
		t.Fatal("expected non-zero ID after save")
	}
	if a.Status != AssignmentStatusPending {
		t.Fatalf("expected status 'pending', got %q", a.Status)
	}
	if a.Priority != AssignmentPriorityNormal {
		t.Fatalf("expected default priority 'normal', got %q", a.Priority)
	}
	if a.CreatedDate.IsZero() {
		t.Fatal("expected CreatedDate to be set")
	}
}

func TestPostAssignmentWithPriority(t *testing.T) {
	teardown := setupAssignmentTest(t)
	defer teardown()

	a := &CourseAssignment{
		UserId:         1,
		PresentationId: 100,
		AssignedBy:     2,
		Priority:       AssignmentPriorityCritical,
	}
	if err := PostAssignment(a); err != nil {
		t.Fatalf("PostAssignment failed: %v", err)
	}
	if a.Priority != AssignmentPriorityCritical {
		t.Fatalf("expected priority 'critical', got %q", a.Priority)
	}
}

func TestPostAssignmentDuplicate(t *testing.T) {
	teardown := setupAssignmentTest(t)
	defer teardown()

	a1 := &CourseAssignment{UserId: 1, PresentationId: 100, AssignedBy: 2}
	PostAssignment(a1)

	a2 := &CourseAssignment{UserId: 1, PresentationId: 100, AssignedBy: 3}
	err := PostAssignment(a2)
	if err != ErrAssignmentExists {
		t.Fatalf("expected ErrAssignmentExists, got %v", err)
	}
}

// =====================================================================
// GetAssignment variants
// =====================================================================

func TestGetAssignment(t *testing.T) {
	teardown := setupAssignmentTest(t)
	defer teardown()

	PostAssignment(&CourseAssignment{UserId: 1, PresentationId: 100, AssignedBy: 2})

	a, err := GetAssignment(1, 100)
	if err != nil {
		t.Fatalf("GetAssignment failed: %v", err)
	}
	if a.UserId != 1 || a.PresentationId != 100 {
		t.Fatal("returned wrong assignment")
	}
}

func TestGetAssignmentById(t *testing.T) {
	teardown := setupAssignmentTest(t)
	defer teardown()

	a := &CourseAssignment{UserId: 1, PresentationId: 100, AssignedBy: 2}
	PostAssignment(a)

	found, err := GetAssignmentById(a.Id)
	if err != nil {
		t.Fatalf("GetAssignmentById failed: %v", err)
	}
	if found.Id != a.Id {
		t.Fatal("wrong assignment returned")
	}
}

func TestGetAssignmentByIdNotFound(t *testing.T) {
	teardown := setupAssignmentTest(t)
	defer teardown()

	_, err := GetAssignmentById(999)
	if err != ErrAssignmentNotFound {
		t.Fatalf("expected ErrAssignmentNotFound, got %v", err)
	}
}

func TestGetAssignmentsForUser(t *testing.T) {
	teardown := setupAssignmentTest(t)
	defer teardown()

	PostAssignment(&CourseAssignment{UserId: 1, PresentationId: 100, AssignedBy: 2})
	PostAssignment(&CourseAssignment{UserId: 1, PresentationId: 200, AssignedBy: 2})
	PostAssignment(&CourseAssignment{UserId: 2, PresentationId: 100, AssignedBy: 2})

	assignments, err := GetAssignmentsForUser(1)
	if err != nil {
		t.Fatalf("GetAssignmentsForUser failed: %v", err)
	}
	if len(assignments) != 2 {
		t.Fatalf("expected 2 assignments for user 1, got %d", len(assignments))
	}
}

func TestGetAssignmentsForPresentation(t *testing.T) {
	teardown := setupAssignmentTest(t)
	defer teardown()

	PostAssignment(&CourseAssignment{UserId: 1, PresentationId: 100, AssignedBy: 2})
	PostAssignment(&CourseAssignment{UserId: 2, PresentationId: 100, AssignedBy: 2})

	assignments, err := GetAssignmentsForPresentation(100)
	if err != nil {
		t.Fatalf("GetAssignmentsForPresentation failed: %v", err)
	}
	if len(assignments) != 2 {
		t.Fatalf("expected 2, got %d", len(assignments))
	}
}

func TestGetAllAssignments(t *testing.T) {
	teardown := setupAssignmentTest(t)
	defer teardown()

	PostAssignment(&CourseAssignment{UserId: 1, PresentationId: 100, AssignedBy: 2})
	PostAssignment(&CourseAssignment{UserId: 2, PresentationId: 200, AssignedBy: 2})

	all, err := GetAllAssignments()
	if err != nil {
		t.Fatalf("GetAllAssignments failed: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2, got %d", len(all))
	}
}

// =====================================================================
// Status transitions
// =====================================================================

func TestUpdateAssignmentStatus(t *testing.T) {
	teardown := setupAssignmentTest(t)
	defer teardown()

	PostAssignment(&CourseAssignment{UserId: 1, PresentationId: 100, AssignedBy: 2})

	if err := UpdateAssignmentStatus(1, 100, AssignmentStatusInProgress); err != nil {
		t.Fatalf("UpdateAssignmentStatus failed: %v", err)
	}
	a, _ := GetAssignment(1, 100)
	if a.Status != AssignmentStatusInProgress {
		t.Fatalf("expected 'in_progress', got %q", a.Status)
	}
}

func TestUpdateAssignmentStatusCompleted(t *testing.T) {
	teardown := setupAssignmentTest(t)
	defer teardown()

	PostAssignment(&CourseAssignment{UserId: 1, PresentationId: 100, AssignedBy: 2})

	if err := UpdateAssignmentStatus(1, 100, AssignmentStatusCompleted); err != nil {
		t.Fatalf("UpdateAssignmentStatus failed: %v", err)
	}
	a, _ := GetAssignment(1, 100)
	if a.Status != AssignmentStatusCompleted {
		t.Fatalf("expected 'completed', got %q", a.Status)
	}
	if a.CompletedDate.IsZero() {
		t.Fatal("expected CompletedDate to be set")
	}
}

func TestUpdateAssignmentStatusInvalid(t *testing.T) {
	teardown := setupAssignmentTest(t)
	defer teardown()

	PostAssignment(&CourseAssignment{UserId: 1, PresentationId: 100, AssignedBy: 2})

	err := UpdateAssignmentStatus(1, 100, "invalid_status")
	if err != ErrInvalidAssignmentStatus {
		t.Fatalf("expected ErrInvalidAssignmentStatus, got %v", err)
	}
}

func TestCancelAssignment(t *testing.T) {
	teardown := setupAssignmentTest(t)
	defer teardown()

	a := &CourseAssignment{UserId: 1, PresentationId: 100, AssignedBy: 2}
	PostAssignment(a)

	if err := CancelAssignment(a.Id); err != nil {
		t.Fatalf("CancelAssignment failed: %v", err)
	}
	found, _ := GetAssignment(1, 100)
	if found.Status != AssignmentStatusCancelled {
		t.Fatalf("expected 'cancelled', got %q", found.Status)
	}
}

// =====================================================================
// Priority management
// =====================================================================

func TestUpdateAssignmentPriority(t *testing.T) {
	teardown := setupAssignmentTest(t)
	defer teardown()

	a := &CourseAssignment{UserId: 1, PresentationId: 100, AssignedBy: 2}
	PostAssignment(a)

	if err := UpdateAssignmentPriority(a.Id, AssignmentPriorityHigh); err != nil {
		t.Fatalf("UpdateAssignmentPriority failed: %v", err)
	}
	found, _ := GetAssignment(1, 100)
	if found.Priority != AssignmentPriorityHigh {
		t.Fatalf("expected 'high', got %q", found.Priority)
	}
}

func TestUpdateAssignmentPriorityInvalid(t *testing.T) {
	teardown := setupAssignmentTest(t)
	defer teardown()

	a := &CourseAssignment{UserId: 1, PresentationId: 100, AssignedBy: 2}
	PostAssignment(a)

	err := UpdateAssignmentPriority(a.Id, "urgent")
	if err == nil {
		t.Fatal("expected error for invalid priority")
	}
}

func TestGetAssignmentsByPriority(t *testing.T) {
	teardown := setupAssignmentTest(t)
	defer teardown()

	PostAssignment(&CourseAssignment{UserId: 1, PresentationId: 100, AssignedBy: 2, Priority: AssignmentPriorityHigh})
	PostAssignment(&CourseAssignment{UserId: 2, PresentationId: 200, AssignedBy: 2, Priority: AssignmentPriorityNormal})

	high, err := GetAssignmentsByPriority(AssignmentPriorityHigh)
	if err != nil {
		t.Fatalf("GetAssignmentsByPriority failed: %v", err)
	}
	if len(high) != 1 {
		t.Fatalf("expected 1 high priority, got %d", len(high))
	}
}

// =====================================================================
// Reminder tracking
// =====================================================================

func TestMarkReminderSent(t *testing.T) {
	teardown := setupAssignmentTest(t)
	defer teardown()

	a := &CourseAssignment{UserId: 1, PresentationId: 100, AssignedBy: 2}
	PostAssignment(a)

	if err := MarkReminderSent(a.Id); err != nil {
		t.Fatalf("MarkReminderSent failed: %v", err)
	}
	found, _ := GetAssignment(1, 100)
	if !found.ReminderSent {
		t.Fatal("expected ReminderSent to be true")
	}
	if found.ReminderDate.IsZero() {
		t.Fatal("expected ReminderDate to be set")
	}
}

func TestGetPendingReminderAssignments(t *testing.T) {
	teardown := setupAssignmentTest(t)
	defer teardown()

	// Due in 24 hours — should be in the 48h window
	PostAssignment(&CourseAssignment{
		UserId: 1, PresentationId: 100, AssignedBy: 2,
		DueDate: time.Now().UTC().Add(24 * time.Hour),
	})
	// Due in 96 hours — outside the 48h window
	PostAssignment(&CourseAssignment{
		UserId: 2, PresentationId: 200, AssignedBy: 2,
		DueDate: time.Now().UTC().Add(96 * time.Hour),
	})

	pending, err := GetPendingReminderAssignments(48)
	if err != nil {
		t.Fatalf("GetPendingReminderAssignments failed: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending reminder, got %d", len(pending))
	}
}

// =====================================================================
// Escalation
// =====================================================================

func TestEscalateAssignment(t *testing.T) {
	teardown := setupAssignmentTest(t)
	defer teardown()

	a := &CourseAssignment{UserId: 1, PresentationId: 100, AssignedBy: 2}
	PostAssignment(a)

	if err := EscalateAssignment(a.Id, 99); err != nil {
		t.Fatalf("EscalateAssignment failed: %v", err)
	}
	found, _ := GetAssignment(1, 100)
	if found.EscalatedTo != 99 {
		t.Fatalf("expected EscalatedTo 99, got %d", found.EscalatedTo)
	}
	if found.EscalatedDate.IsZero() {
		t.Fatal("expected EscalatedDate to be set")
	}
}

// =====================================================================
// Notes
// =====================================================================

func TestUpdateAssignmentNotes(t *testing.T) {
	teardown := setupAssignmentTest(t)
	defer teardown()

	a := &CourseAssignment{UserId: 1, PresentationId: 100, AssignedBy: 2}
	PostAssignment(a)

	if err := UpdateAssignmentNotes(a.Id, "Needs follow-up"); err != nil {
		t.Fatalf("UpdateAssignmentNotes failed: %v", err)
	}
	found, _ := GetAssignment(1, 100)
	if found.Notes != "Needs follow-up" {
		t.Fatalf("expected notes 'Needs follow-up', got %q", found.Notes)
	}
}

// =====================================================================
// Overdue detection
// =====================================================================

func TestGetOverdueAssignments(t *testing.T) {
	teardown := setupAssignmentTest(t)
	defer teardown()

	// Past due
	PostAssignment(&CourseAssignment{
		UserId: 1, PresentationId: 100, AssignedBy: 2,
		DueDate: time.Now().UTC().Add(-48 * time.Hour),
	})
	// Future due
	PostAssignment(&CourseAssignment{
		UserId: 2, PresentationId: 200, AssignedBy: 2,
		DueDate: time.Now().UTC().Add(48 * time.Hour),
	})

	overdue, err := GetOverdueAssignments()
	if err != nil {
		t.Fatalf("GetOverdueAssignments failed: %v", err)
	}
	if len(overdue) != 1 {
		t.Fatalf("expected 1 overdue, got %d", len(overdue))
	}
}

func TestMarkOverdueAssignments(t *testing.T) {
	teardown := setupAssignmentTest(t)
	defer teardown()

	PostAssignment(&CourseAssignment{
		UserId: 1, PresentationId: 100, AssignedBy: 2,
		DueDate: time.Now().UTC().Add(-24 * time.Hour),
	})
	PostAssignment(&CourseAssignment{
		UserId: 2, PresentationId: 200, AssignedBy: 2,
		DueDate: time.Now().UTC().Add(24 * time.Hour),
	})

	count, err := MarkOverdueAssignments()
	if err != nil {
		t.Fatalf("MarkOverdueAssignments failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 marked overdue, got %d", count)
	}

	a, _ := GetAssignment(1, 100)
	if a.Status != AssignmentStatusOverdue {
		t.Fatalf("expected status 'overdue', got %q", a.Status)
	}
}

// =====================================================================
// Bulk operations
// =====================================================================

func TestBulkUpdateAssignmentStatus(t *testing.T) {
	teardown := setupAssignmentTest(t)
	defer teardown()

	a1 := &CourseAssignment{UserId: 1, PresentationId: 100, AssignedBy: 2}
	a2 := &CourseAssignment{UserId: 2, PresentationId: 200, AssignedBy: 2}
	PostAssignment(a1)
	PostAssignment(a2)

	count, err := BulkUpdateAssignmentStatus([]int64{a1.Id, a2.Id}, AssignmentStatusCompleted)
	if err != nil {
		t.Fatalf("BulkUpdateAssignmentStatus failed: %v", err)
	}
	if count != 2 {
		t.Fatalf("expected 2 updated, got %d", count)
	}

	found1, _ := GetAssignment(1, 100)
	found2, _ := GetAssignment(2, 200)
	if found1.Status != AssignmentStatusCompleted || found2.Status != AssignmentStatusCompleted {
		t.Fatal("not all assignments were completed")
	}
}

func TestBulkUpdateAssignmentStatusInvalid(t *testing.T) {
	teardown := setupAssignmentTest(t)
	defer teardown()

	_, err := BulkUpdateAssignmentStatus([]int64{1}, "bogus")
	if err != ErrInvalidAssignmentStatus {
		t.Fatalf("expected ErrInvalidAssignmentStatus, got %v", err)
	}
}

// =====================================================================
// Counts and summary
// =====================================================================

func TestGetActiveAssignmentCount(t *testing.T) {
	teardown := setupAssignmentTest(t)
	defer teardown()

	PostAssignment(&CourseAssignment{UserId: 1, PresentationId: 100, AssignedBy: 2})
	PostAssignment(&CourseAssignment{UserId: 1, PresentationId: 200, AssignedBy: 2})
	a3 := &CourseAssignment{UserId: 1, PresentationId: 300, AssignedBy: 2}
	PostAssignment(a3)
	UpdateAssignmentStatus(1, 300, AssignmentStatusCompleted)

	count := GetActiveAssignmentCount(1)
	if count != 2 {
		t.Fatalf("expected 2 active, got %d", count)
	}
}

func TestGetAssignmentSummary(t *testing.T) {
	teardown := setupAssignmentTest(t)
	defer teardown()

	PostAssignment(&CourseAssignment{UserId: 1, PresentationId: 100, AssignedBy: 2, Priority: AssignmentPriorityHigh})
	PostAssignment(&CourseAssignment{UserId: 2, PresentationId: 200, AssignedBy: 2, Priority: AssignmentPriorityCritical})
	a3 := &CourseAssignment{UserId: 3, PresentationId: 300, AssignedBy: 2}
	PostAssignment(a3)
	UpdateAssignmentStatus(3, 300, AssignmentStatusCompleted)

	summary, err := GetAssignmentSummary()
	if err != nil {
		t.Fatalf("GetAssignmentSummary failed: %v", err)
	}
	if summary.TotalAssignments != 3 {
		t.Fatalf("expected total 3, got %d", summary.TotalAssignments)
	}
	if summary.Pending != 2 {
		t.Fatalf("expected 2 pending, got %d", summary.Pending)
	}
	if summary.Completed != 1 {
		t.Fatalf("expected 1 completed, got %d", summary.Completed)
	}
	if summary.HighPriority != 1 {
		t.Fatalf("expected 1 high priority, got %d", summary.HighPriority)
	}
	if summary.CriticalPriority != 1 {
		t.Fatalf("expected 1 critical priority, got %d", summary.CriticalPriority)
	}
}

// =====================================================================
// GetAssignmentsByStatus
// =====================================================================

func TestGetAssignmentsByStatus(t *testing.T) {
	teardown := setupAssignmentTest(t)
	defer teardown()

	PostAssignment(&CourseAssignment{UserId: 1, PresentationId: 100, AssignedBy: 2})
	a2 := &CourseAssignment{UserId: 2, PresentationId: 200, AssignedBy: 2}
	PostAssignment(a2)
	UpdateAssignmentStatus(2, 200, AssignmentStatusInProgress)

	pending, err := GetAssignmentsByStatus(AssignmentStatusPending)
	if err != nil {
		t.Fatalf("GetAssignmentsByStatus failed: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending, got %d", len(pending))
	}
}

// =====================================================================
// Delete
// =====================================================================

func TestDeleteAssignment(t *testing.T) {
	teardown := setupAssignmentTest(t)
	defer teardown()

	a := &CourseAssignment{UserId: 1, PresentationId: 100, AssignedBy: 2}
	PostAssignment(a)

	if err := DeleteAssignment(a.Id); err != nil {
		t.Fatalf("DeleteAssignment failed: %v", err)
	}

	_, err := GetAssignment(1, 100)
	if err == nil {
		t.Fatal("expected error getting deleted assignment")
	}
}

// =====================================================================
// CompleteAssignmentOnCourseFinish
// =====================================================================

func TestCompleteAssignmentOnCourseFinish(t *testing.T) {
	teardown := setupAssignmentTest(t)
	defer teardown()

	PostAssignment(&CourseAssignment{UserId: 1, PresentationId: 100, AssignedBy: 2})

	if err := CompleteAssignmentOnCourseFinish(1, 100); err != nil {
		t.Fatalf("CompleteAssignmentOnCourseFinish failed: %v", err)
	}
	a, _ := GetAssignment(1, 100)
	if a.Status != AssignmentStatusCompleted {
		t.Fatalf("expected 'completed', got %q", a.Status)
	}
}

func TestCompleteAssignmentOnCourseFinishNoAssignment(t *testing.T) {
	teardown := setupAssignmentTest(t)
	defer teardown()

	// Should not error if no assignment exists
	if err := CompleteAssignmentOnCourseFinish(999, 999); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCompleteAssignmentOnCourseFinishAlreadyCompleted(t *testing.T) {
	teardown := setupAssignmentTest(t)
	defer teardown()

	PostAssignment(&CourseAssignment{UserId: 1, PresentationId: 100, AssignedBy: 2})
	UpdateAssignmentStatus(1, 100, AssignmentStatusCompleted)

	// Should not error
	if err := CompleteAssignmentOnCourseFinish(1, 100); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

// =====================================================================
// ValidAssignmentStatuses and ValidAssignmentPriorities
// =====================================================================

func TestValidAssignmentStatuses(t *testing.T) {
	expected := []string{"pending", "in_progress", "completed", "overdue", "cancelled"}
	for _, s := range expected {
		if !ValidAssignmentStatuses[s] {
			t.Fatalf("expected %q to be valid", s)
		}
	}
	if ValidAssignmentStatuses["bogus"] {
		t.Fatal("'bogus' should not be valid")
	}
}

func TestValidAssignmentPriorities(t *testing.T) {
	expected := []string{"low", "normal", "high", "critical"}
	for _, p := range expected {
		if !ValidAssignmentPriorities[p] {
			t.Fatalf("expected %q to be valid", p)
		}
	}
	if ValidAssignmentPriorities["urgent"] {
		t.Fatal("'urgent' should not be valid")
	}
}
