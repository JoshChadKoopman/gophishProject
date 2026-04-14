package models

import (
	"testing"
	"time"

	"github.com/gophish/gophish/config"
)

func setupSandboxTest(t *testing.T) func() {
	t.Helper()
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	if err := Setup(conf); err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	db.Exec("DELETE FROM sandbox_tests")
	return func() {
		db.Exec("DELETE FROM sandbox_tests")
	}
}

func TestSandboxTestValidation(t *testing.T) {
	s := &SandboxTest{}
	if err := s.Validate(); err == nil {
		t.Fatal("expected error for empty template_id")
	}
	s.TemplateId = 1
	if err := s.Validate(); err == nil {
		t.Fatal("expected error for empty smtp_id")
	}
	s.SmtpId = 1
	if err := s.Validate(); err == nil {
		t.Fatal("expected error for empty to_email")
	}
	s.ToEmail = "test@example.com"
	if err := s.Validate(); err != nil {
		t.Fatalf("valid sandbox test should pass validation: %v", err)
	}
}

func TestPostAndGetSandboxTest(t *testing.T) {
	teardown := setupSandboxTest(t)
	defer teardown()

	s := &SandboxTest{
		OrgId:      1,
		CreatedBy:  1,
		TemplateId: 99,
		SmtpId:     88,
		ToEmail:    "sandbox@test.com",
		Subject:    "Test Subject",
	}
	if err := PostSandboxTest(s); err != nil {
		t.Fatalf("PostSandboxTest: %v", err)
	}
	if s.Id == 0 {
		t.Fatal("expected non-zero ID")
	}
	if s.Status != SandboxStatusPending {
		t.Fatalf("expected status %q, got %q", SandboxStatusPending, s.Status)
	}

	got, err := GetSandboxTest(s.Id, 1)
	if err != nil {
		t.Fatalf("GetSandboxTest: %v", err)
	}
	if got.ToEmail != "sandbox@test.com" {
		t.Fatalf("expected to_email 'sandbox@test.com', got %q", got.ToEmail)
	}
}

func TestGetSandboxTestOrgIsolation(t *testing.T) {
	teardown := setupSandboxTest(t)
	defer teardown()

	s := &SandboxTest{OrgId: 1, CreatedBy: 1, TemplateId: 1, SmtpId: 1, ToEmail: "a@b.com"}
	PostSandboxTest(s)

	// Org 2 should not be able to access org 1's test.
	_, err := GetSandboxTest(s.Id, 2)
	if err == nil {
		t.Fatal("expected org isolation to prevent cross-org access")
	}
}

func TestGetSandboxTests(t *testing.T) {
	teardown := setupSandboxTest(t)
	defer teardown()

	for i := 0; i < 3; i++ {
		PostSandboxTest(&SandboxTest{
			OrgId: 1, CreatedBy: 1, TemplateId: int64(i + 1),
			SmtpId: 1, ToEmail: "test@example.com",
		})
	}

	tests, err := GetSandboxTests(1)
	if err != nil {
		t.Fatalf("GetSandboxTests: %v", err)
	}
	if len(tests) != 3 {
		t.Fatalf("expected 3 tests, got %d", len(tests))
	}
}

func TestUpdateSandboxTestStatus(t *testing.T) {
	teardown := setupSandboxTest(t)
	defer teardown()

	s := &SandboxTest{OrgId: 1, CreatedBy: 1, TemplateId: 1, SmtpId: 1, ToEmail: "a@b.com"}
	PostSandboxTest(s)

	now := time.Now().UTC()
	if err := UpdateSandboxTestStatus(s.Id, SandboxStatusDelivered, "<p>rendered</p>", "", now); err != nil {
		t.Fatalf("UpdateSandboxTestStatus: %v", err)
	}

	got, _ := GetSandboxTest(s.Id, 1)
	if got.Status != SandboxStatusDelivered {
		t.Fatalf("expected status %q, got %q", SandboxStatusDelivered, got.Status)
	}
	if got.RenderedHTML != "<p>rendered</p>" {
		t.Fatalf("expected rendered HTML to be set")
	}
}

func TestUpdateSandboxTestStatusFailed(t *testing.T) {
	teardown := setupSandboxTest(t)
	defer teardown()

	s := &SandboxTest{OrgId: 1, CreatedBy: 1, TemplateId: 1, SmtpId: 1, ToEmail: "a@b.com"}
	PostSandboxTest(s)

	if err := UpdateSandboxTestStatus(s.Id, SandboxStatusFailed, "", "SMTP connection refused", time.Time{}); err != nil {
		t.Fatalf("UpdateSandboxTestStatus: %v", err)
	}
	got, _ := GetSandboxTest(s.Id, 1)
	if got.Status != SandboxStatusFailed {
		t.Fatalf("expected failed status")
	}
	if got.ErrorMsg != "SMTP connection refused" {
		t.Fatalf("expected error message to be set, got %q", got.ErrorMsg)
	}
}

func TestReviewSandboxTest(t *testing.T) {
	teardown := setupSandboxTest(t)
	defer teardown()

	s := &SandboxTest{OrgId: 1, CreatedBy: 1, TemplateId: 1, SmtpId: 1, ToEmail: "a@b.com"}
	PostSandboxTest(s)

	if err := ReviewSandboxTest(s.Id, 42, SandboxStatusApproved, "Looks good"); err != nil {
		t.Fatalf("ReviewSandboxTest: %v", err)
	}
	got, _ := GetSandboxTest(s.Id, 1)
	if got.Status != SandboxStatusApproved {
		t.Fatalf("expected approved status, got %q", got.Status)
	}
	if got.ReviewedBy != 42 {
		t.Fatalf("expected reviewed_by 42, got %d", got.ReviewedBy)
	}
	if got.Notes != "Looks good" {
		t.Fatalf("expected notes 'Looks good', got %q", got.Notes)
	}
}

func TestReviewSandboxTestRejected(t *testing.T) {
	teardown := setupSandboxTest(t)
	defer teardown()

	s := &SandboxTest{OrgId: 1, CreatedBy: 1, TemplateId: 1, SmtpId: 1, ToEmail: "a@b.com"}
	PostSandboxTest(s)

	ReviewSandboxTest(s.Id, 99, SandboxStatusRejected, "Too aggressive")
	got, _ := GetSandboxTest(s.Id, 1)
	if got.Status != SandboxStatusRejected {
		t.Fatalf("expected rejected status, got %q", got.Status)
	}
}

func TestDeleteSandboxTest(t *testing.T) {
	teardown := setupSandboxTest(t)
	defer teardown()

	s := &SandboxTest{OrgId: 1, CreatedBy: 1, TemplateId: 1, SmtpId: 1, ToEmail: "a@b.com"}
	PostSandboxTest(s)

	if err := DeleteSandboxTest(s.Id, 1); err != nil {
		t.Fatalf("DeleteSandboxTest: %v", err)
	}
	_, err := GetSandboxTest(s.Id, 1)
	if err == nil {
		t.Fatal("expected sandbox test to be deleted")
	}
}

func TestDeleteSandboxTestOrgIsolation(t *testing.T) {
	teardown := setupSandboxTest(t)
	defer teardown()

	s := &SandboxTest{OrgId: 1, CreatedBy: 1, TemplateId: 1, SmtpId: 1, ToEmail: "a@b.com"}
	PostSandboxTest(s)

	// Org 2 should not be able to delete org 1's test.
	DeleteSandboxTest(s.Id, 2)
	got, err := GetSandboxTest(s.Id, 1)
	if err != nil {
		t.Fatal("org 2's delete should not affect org 1's test")
	}
	if got.Id != s.Id {
		t.Fatal("test should still exist")
	}
}

func TestSandboxStatusConstants(t *testing.T) {
	statuses := []string{
		SandboxStatusPending,
		SandboxStatusSending,
		SandboxStatusDelivered,
		SandboxStatusFailed,
		SandboxStatusApproved,
		SandboxStatusRejected,
	}
	seen := map[string]bool{}
	for _, s := range statuses {
		if s == "" {
			t.Fatal("sandbox status constant should not be empty")
		}
		if seen[s] {
			t.Fatalf("duplicate sandbox status: %q", s)
		}
		seen[s] = true
	}
}
