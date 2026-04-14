package models

import (
	"testing"

	"github.com/gophish/gophish/config"
)

// Test constants to avoid duplicate literal warnings (SonarLint S1192).
const (
	testEscEmail = "test@example.com"
	testEmailA   = "a@test.com"
	testEmailB   = "b@test.com"
)

// setupEscalationTest initialises an in-memory DB for escalation tests.
func setupEscalationTest(t *testing.T) func() {
	t.Helper()
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	if err := Setup(conf); err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	db.Exec("DELETE FROM escalation_events")
	db.Exec("DELETE FROM escalation_policies")
	return func() {
		db.Exec("DELETE FROM escalation_events")
		db.Exec("DELETE FROM escalation_policies")
	}
}

// ---------- PostEscalationPolicy ----------

func TestPostEscalationPolicy(t *testing.T) {
	teardown := setupEscalationTest(t)
	defer teardown()

	p := &EscalationPolicy{
		OrgId:         1,
		Name:          "Level 1 — Notify",
		Level:         1,
		FailThreshold: 3,
		LookbackDays:  90,
		Action:        EscalationActionNotify,
		NotifyManager: true,
		IsActive:      true,
	}
	if err := PostEscalationPolicy(p); err != nil {
		t.Fatalf("PostEscalationPolicy failed: %v", err)
	}
	if p.Id == 0 {
		t.Fatal("expected non-zero policy ID")
	}
	if p.CreatedDate.IsZero() {
		t.Fatal("expected CreatedDate to be set")
	}
}

// ---------- GetEscalationPolicies ----------

func TestGetEscalationPolicies(t *testing.T) {
	teardown := setupEscalationTest(t)
	defer teardown()

	PostEscalationPolicy(&EscalationPolicy{OrgId: 1, Name: "L1", Level: 1, FailThreshold: 2, LookbackDays: 90, Action: EscalationActionNotify, IsActive: true})
	PostEscalationPolicy(&EscalationPolicy{OrgId: 1, Name: "L2", Level: 2, FailThreshold: 5, LookbackDays: 90, Action: EscalationActionTraining, IsActive: true})
	PostEscalationPolicy(&EscalationPolicy{OrgId: 2, Name: "Other", Level: 1, FailThreshold: 3, LookbackDays: 60, Action: EscalationActionNotify, IsActive: true})

	policies, err := GetEscalationPolicies(1)
	if err != nil {
		t.Fatalf("GetEscalationPolicies failed: %v", err)
	}
	if len(policies) != 2 {
		t.Fatalf("expected 2 policies for org 1, got %d", len(policies))
	}
	// Should be ordered by level
	if policies[0].Level > policies[1].Level {
		t.Fatal("expected policies ordered by level ascending")
	}
}

// ---------- GetEscalationPolicy ----------

func TestGetEscalationPolicy(t *testing.T) {
	teardown := setupEscalationTest(t)
	defer teardown()

	p := &EscalationPolicy{OrgId: 1, Name: "Find Me", Level: 1, FailThreshold: 3, LookbackDays: 90, Action: EscalationActionNotify, IsActive: true}
	PostEscalationPolicy(p)

	fetched, err := GetEscalationPolicy(p.Id, 1)
	if err != nil {
		t.Fatalf("GetEscalationPolicy failed: %v", err)
	}
	if fetched.Name != "Find Me" {
		t.Fatalf("expected name 'Find Me', got %q", fetched.Name)
	}
}

func TestGetEscalationPolicyWrongOrg(t *testing.T) {
	teardown := setupEscalationTest(t)
	defer teardown()

	p := &EscalationPolicy{OrgId: 1, Name: "P1", Level: 1, FailThreshold: 3, LookbackDays: 90, Action: EscalationActionNotify, IsActive: true}
	PostEscalationPolicy(p)

	_, err := GetEscalationPolicy(p.Id, 999)
	if err == nil {
		t.Fatal("expected error when fetching policy from wrong org")
	}
}

// ---------- PutEscalationPolicy ----------

func TestPutEscalationPolicy(t *testing.T) {
	teardown := setupEscalationTest(t)
	defer teardown()

	p := &EscalationPolicy{OrgId: 1, Name: "Original", Level: 1, FailThreshold: 3, LookbackDays: 90, Action: EscalationActionNotify, IsActive: true}
	PostEscalationPolicy(p)

	p.Name = "Updated"
	p.FailThreshold = 5
	p.Action = EscalationActionTraining
	if err := PutEscalationPolicy(p); err != nil {
		t.Fatalf("PutEscalationPolicy failed: %v", err)
	}

	fetched, _ := GetEscalationPolicy(p.Id, 1)
	if fetched.Name != "Updated" {
		t.Fatalf("expected name 'Updated', got %q", fetched.Name)
	}
	if fetched.FailThreshold != 5 {
		t.Fatalf("expected FailThreshold 5, got %d", fetched.FailThreshold)
	}
	if fetched.Action != EscalationActionTraining {
		t.Fatalf("expected action %q, got %q", EscalationActionTraining, fetched.Action)
	}
}

// ---------- DeleteEscalationPolicy ----------

func TestDeleteEscalationPolicy(t *testing.T) {
	teardown := setupEscalationTest(t)
	defer teardown()

	p := &EscalationPolicy{OrgId: 1, Name: "To Delete", Level: 1, FailThreshold: 3, LookbackDays: 90, Action: EscalationActionNotify, IsActive: true}
	PostEscalationPolicy(p)

	if err := DeleteEscalationPolicy(p.Id, 1); err != nil {
		t.Fatalf("DeleteEscalationPolicy failed: %v", err)
	}

	_, err := GetEscalationPolicy(p.Id, 1)
	if err == nil {
		t.Fatal("expected error after deleting policy")
	}
}

func TestDeleteEscalationPolicyWrongOrg(t *testing.T) {
	teardown := setupEscalationTest(t)
	defer teardown()

	p := &EscalationPolicy{OrgId: 1, Name: "P1", Level: 1, FailThreshold: 3, LookbackDays: 90, Action: EscalationActionNotify, IsActive: true}
	PostEscalationPolicy(p)

	// Should not error but also shouldn't delete (wrong org)
	DeleteEscalationPolicy(p.Id, 999)

	// Policy should still exist
	_, err := GetEscalationPolicy(p.Id, 1)
	if err != nil {
		t.Fatal("policy should not have been deleted from wrong org")
	}
}

// ---------- Escalation Action Constants ----------

func TestEscalationActionConstants(t *testing.T) {
	actions := map[string]string{
		"notify":             EscalationActionNotify,
		"mandatory_training": EscalationActionTraining,
		"restrict_access":    EscalationActionRestrictAccess,
		"manager_escalate":   EscalationActionManagerEscalate,
	}
	for expected, got := range actions {
		if got != expected {
			t.Fatalf("expected %q, got %q", expected, got)
		}
	}
}

func TestEscalationStatusConstants(t *testing.T) {
	if EscalationStatusOpen != "open" {
		t.Fatalf("expected 'open', got %q", EscalationStatusOpen)
	}
	if EscalationStatusResolved != "resolved" {
		t.Fatalf("expected 'resolved', got %q", EscalationStatusResolved)
	}
	if EscalationStatusExpired != "expired" {
		t.Fatalf("expected 'expired', got %q", EscalationStatusExpired)
	}
}

// ---------- determineEscalationLevel ----------

func TestDetermineEscalationLevel(t *testing.T) {
	policies := []EscalationPolicy{
		{Level: 1, FailThreshold: 2, IsActive: true},
		{Level: 2, FailThreshold: 5, IsActive: true},
		{Level: 3, FailThreshold: 10, IsActive: true},
		{Level: 4, FailThreshold: 3, IsActive: false}, // inactive
	}

	tests := []struct {
		failCount int
		expected  int
	}{
		{0, 0},
		{1, 0},
		{2, 1},
		{4, 1},
		{5, 2},
		{9, 2},
		{10, 3},
		{15, 3},
	}

	for _, tc := range tests {
		got := determineEscalationLevel(tc.failCount, policies)
		if got != tc.expected {
			t.Errorf("failCount=%d: expected level %d, got %d", tc.failCount, tc.expected, got)
		}
	}
}

// ---------- hasOpenEscalation ----------

func TestHasOpenEscalation(t *testing.T) {
	teardown := setupEscalationTest(t)
	defer teardown()

	// No events — should be false
	if hasOpenEscalation(1, testEscEmail, 1) {
		t.Fatal("expected no open escalation")
	}

	// Create an open event
	event := &EscalationEvent{
		OrgId:     1,
		PolicyId:  1,
		UserEmail: testEscEmail,
		Level:     1,
		Action:    EscalationActionNotify,
		Status:    EscalationStatusOpen,
	}
	db.Save(event)

	if !hasOpenEscalation(1, testEscEmail, 1) {
		t.Fatal("expected open escalation to be found")
	}

	// Different level should not match
	if hasOpenEscalation(1, testEscEmail, 2) {
		t.Fatal("expected no open escalation at different level")
	}
}

// ---------- GetEscalationEvents ----------

func TestGetEscalationEvents(t *testing.T) {
	teardown := setupEscalationTest(t)
	defer teardown()

	db.Save(&EscalationEvent{OrgId: 1, PolicyId: 1, UserEmail: testEmailA, Level: 1, Status: EscalationStatusOpen})
	db.Save(&EscalationEvent{OrgId: 1, PolicyId: 1, UserEmail: testEmailB, Level: 1, Status: EscalationStatusResolved})
	db.Save(&EscalationEvent{OrgId: 2, PolicyId: 2, UserEmail: "c@test.com", Level: 1, Status: EscalationStatusOpen})

	events, err := GetEscalationEvents(1, "", 0)
	if err != nil {
		t.Fatalf("GetEscalationEvents failed: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events for org 1, got %d", len(events))
	}
}

func TestGetEscalationEventsFilterByStatus(t *testing.T) {
	teardown := setupEscalationTest(t)
	defer teardown()

	db.Save(&EscalationEvent{OrgId: 1, PolicyId: 1, UserEmail: testEmailA, Level: 1, Status: EscalationStatusOpen})
	db.Save(&EscalationEvent{OrgId: 1, PolicyId: 1, UserEmail: testEmailB, Level: 1, Status: EscalationStatusResolved})

	events, _ := GetEscalationEvents(1, EscalationStatusOpen, 0)
	if len(events) != 1 {
		t.Fatalf("expected 1 open event, got %d", len(events))
	}
}

// ---------- ResolveEscalation ----------

func TestResolveEscalation(t *testing.T) {
	teardown := setupEscalationTest(t)
	defer teardown()

	event := &EscalationEvent{
		OrgId:     1,
		PolicyId:  1,
		UserEmail: "test@test.com",
		Level:     1,
		Action:    EscalationActionNotify,
		Status:    EscalationStatusOpen,
	}
	db.Save(event)

	err := ResolveEscalation(event.Id, 1, 42)
	if err != nil {
		t.Fatalf("ResolveEscalation failed: %v", err)
	}

	var resolved EscalationEvent
	db.Where("id = ?", event.Id).First(&resolved)
	if resolved.Status != EscalationStatusResolved {
		t.Fatalf("expected status 'resolved', got %q", resolved.Status)
	}
	if resolved.ResolvedBy != 42 {
		t.Fatalf("expected resolved_by 42, got %d", resolved.ResolvedBy)
	}
}

// ---------- GetEscalationSummary ----------

func TestGetEscalationSummary(t *testing.T) {
	teardown := setupEscalationTest(t)
	defer teardown()

	db.Save(&EscalationEvent{OrgId: 1, PolicyId: 1, UserEmail: testEmailA, Level: 1, FailCount: 3, Status: EscalationStatusOpen})
	db.Save(&EscalationEvent{OrgId: 1, PolicyId: 1, UserEmail: testEmailB, Level: 1, FailCount: 5, Status: EscalationStatusOpen})
	db.Save(&EscalationEvent{OrgId: 1, PolicyId: 1, UserEmail: "c@test.com", Level: 2, FailCount: 7, Status: EscalationStatusResolved})

	summary, err := GetEscalationSummary(1)
	if err != nil {
		t.Fatalf("GetEscalationSummary failed: %v", err)
	}
	if summary.OpenCount != 2 {
		t.Fatalf("expected OpenCount 2, got %d", summary.OpenCount)
	}
	if summary.ResolvedCount != 1 {
		t.Fatalf("expected ResolvedCount 1, got %d", summary.ResolvedCount)
	}
	if summary.TotalOffenders != 2 {
		t.Fatalf("expected TotalOffenders 2, got %d", summary.TotalOffenders)
	}
	if summary.AvgFailCount != 4.0 { // (3+5)/2
		t.Fatalf("expected AvgFailCount 4.0, got %f", summary.AvgFailCount)
	}
}

func TestGetEscalationSummaryEmpty(t *testing.T) {
	teardown := setupEscalationTest(t)
	defer teardown()

	summary, err := GetEscalationSummary(1)
	if err != nil {
		t.Fatalf("GetEscalationSummary failed: %v", err)
	}
	if summary.OpenCount != 0 {
		t.Fatalf("expected 0 open, got %d", summary.OpenCount)
	}
	if summary.AvgFailCount != 0 {
		t.Fatalf("expected 0 avg fail count, got %f", summary.AvgFailCount)
	}
}
