package models

import (
	"testing"

	"github.com/gophish/gophish/config"
)

// Test constants to avoid duplicate literal warnings (SonarLint S1192).
const (
	testReporterEmail = "test@example.com"
)

// setupReportButtonTest initialises an in-memory DB for report button tests.
func setupReportButtonTest(t *testing.T) func() {
	t.Helper()
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	if err := Setup(conf); err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	db.Exec("DELETE FROM reported_emails")
	db.Exec("DELETE FROM report_button_configs")
	return func() {
		db.Exec("DELETE FROM reported_emails")
		db.Exec("DELETE FROM report_button_configs")
	}
}

// ---------- GeneratePluginAPIKey ----------

func TestGeneratePluginAPIKey(t *testing.T) {
	key, err := GeneratePluginAPIKey()
	if err != nil {
		t.Fatalf("GeneratePluginAPIKey failed: %v", err)
	}
	if len(key) != 64 { // 32 bytes = 64 hex chars
		t.Fatalf("expected 64-char key, got %d", len(key))
	}
}

func TestGeneratePluginAPIKeyUniqueness(t *testing.T) {
	keys := make(map[string]bool)
	for i := 0; i < 10; i++ {
		key, _ := GeneratePluginAPIKey()
		if keys[key] {
			t.Fatalf("duplicate API key generated: %s", key)
		}
		keys[key] = true
	}
}

// ---------- CreateReportButtonConfig ----------

func TestCreateReportButtonConfig(t *testing.T) {
	teardown := setupReportButtonTest(t)
	defer teardown()

	cfg := &ReportButtonConfig{
		OrgId:              1,
		FeedbackSimulation: "Good catch! This was a simulation.",
		FeedbackReal:       "Thanks for reporting this suspicious email.",
		Enabled:            true,
	}
	if err := CreateReportButtonConfig(cfg); err != nil {
		t.Fatalf("CreateReportButtonConfig failed: %v", err)
	}
	if cfg.Id == 0 {
		t.Fatal("expected non-zero config ID")
	}
	if cfg.PluginApiKey == "" {
		t.Fatal("expected API key to be auto-generated")
	}
	if cfg.CreatedDate.IsZero() {
		t.Fatal("expected CreatedDate to be set")
	}
}

func TestCreateReportButtonConfigWithCustomKey(t *testing.T) {
	teardown := setupReportButtonTest(t)
	defer teardown()

	cfg := &ReportButtonConfig{
		OrgId:        1,
		PluginApiKey: "custom-api-key-12345",
		Enabled:      true,
	}
	if err := CreateReportButtonConfig(cfg); err != nil {
		t.Fatalf("CreateReportButtonConfig failed: %v", err)
	}
	if cfg.PluginApiKey != "custom-api-key-12345" {
		t.Fatalf("expected custom API key, got %q", cfg.PluginApiKey)
	}
}

// ---------- GetReportButtonConfig ----------

func TestGetReportButtonConfig(t *testing.T) {
	teardown := setupReportButtonTest(t)
	defer teardown()

	cfg := &ReportButtonConfig{OrgId: 1, Enabled: true}
	CreateReportButtonConfig(cfg)

	fetched, err := GetReportButtonConfig(1)
	if err != nil {
		t.Fatalf("GetReportButtonConfig failed: %v", err)
	}
	if fetched.Id != cfg.Id {
		t.Fatalf("expected ID %d, got %d", cfg.Id, fetched.Id)
	}
}

func TestGetReportButtonConfigNotFound(t *testing.T) {
	teardown := setupReportButtonTest(t)
	defer teardown()

	_, err := GetReportButtonConfig(999)
	if err == nil {
		t.Fatal("expected error for non-existent config")
	}
}

// ---------- GetReportButtonConfigByAPIKey ----------

func TestGetReportButtonConfigByAPIKey(t *testing.T) {
	teardown := setupReportButtonTest(t)
	defer teardown()

	cfg := &ReportButtonConfig{OrgId: 1, Enabled: true}
	CreateReportButtonConfig(cfg)

	fetched, err := GetReportButtonConfigByAPIKey(cfg.PluginApiKey)
	if err != nil {
		t.Fatalf("GetReportButtonConfigByAPIKey failed: %v", err)
	}
	if fetched.OrgId != 1 {
		t.Fatalf("expected OrgId 1, got %d", fetched.OrgId)
	}
}

func TestGetReportButtonConfigByAPIKeyDisabled(t *testing.T) {
	teardown := setupReportButtonTest(t)
	defer teardown()

	cfg := &ReportButtonConfig{OrgId: 1, Enabled: true}
	CreateReportButtonConfig(cfg)

	// Now disable it
	cfg.Enabled = false
	UpdateReportButtonConfig(cfg)

	_, err := GetReportButtonConfigByAPIKey(cfg.PluginApiKey)
	if err == nil {
		t.Fatal("expected error when config is disabled")
	}
}

// ---------- UpdateReportButtonConfig ----------

func TestUpdateReportButtonConfig(t *testing.T) {
	teardown := setupReportButtonTest(t)
	defer teardown()

	cfg := &ReportButtonConfig{OrgId: 1, FeedbackSimulation: "Old feedback", Enabled: true}
	CreateReportButtonConfig(cfg)

	cfg.FeedbackSimulation = "New feedback"
	if err := UpdateReportButtonConfig(cfg); err != nil {
		t.Fatalf("UpdateReportButtonConfig failed: %v", err)
	}

	fetched, _ := GetReportButtonConfig(1)
	if fetched.FeedbackSimulation != "New feedback" {
		t.Fatalf("expected 'New feedback', got %q", fetched.FeedbackSimulation)
	}
}

// ---------- RegeneratePluginAPIKey ----------

func TestRegeneratePluginAPIKey(t *testing.T) {
	teardown := setupReportButtonTest(t)
	defer teardown()

	cfg := &ReportButtonConfig{OrgId: 1, Enabled: true}
	CreateReportButtonConfig(cfg)
	originalKey := cfg.PluginApiKey

	updated, err := RegeneratePluginAPIKey(1)
	if err != nil {
		t.Fatalf("RegeneratePluginAPIKey failed: %v", err)
	}
	if updated.PluginApiKey == originalKey {
		t.Fatal("expected a new API key after regeneration")
	}
	if len(updated.PluginApiKey) != 64 {
		t.Fatalf("expected 64-char key, got %d", len(updated.PluginApiKey))
	}
}

// ---------- CreateReportedEmail ----------

func TestCreateReportedEmail(t *testing.T) {
	teardown := setupReportButtonTest(t)
	defer teardown()

	re := &ReportedEmail{
		OrgId:         1,
		ReporterEmail: "employee@example.com",
		SenderEmail:   "phisher@evil.com",
		Subject:       "You've won a prize!",
	}
	if err := CreateReportedEmail(re); err != nil {
		t.Fatalf("CreateReportedEmail failed: %v", err)
	}
	if re.Id == 0 {
		t.Fatal("expected non-zero ID")
	}
	if re.CreatedDate.IsZero() {
		t.Fatal("expected CreatedDate to be set")
	}
}

// ---------- GetReportedEmails ----------

func TestGetReportedEmails(t *testing.T) {
	teardown := setupReportButtonTest(t)
	defer teardown()

	CreateReportedEmail(&ReportedEmail{OrgId: 1, ReporterEmail: "a@example.com", Subject: "S1"})
	CreateReportedEmail(&ReportedEmail{OrgId: 1, ReporterEmail: "b@example.com", Subject: "S2"})
	CreateReportedEmail(&ReportedEmail{OrgId: 2, ReporterEmail: "c@example.com", Subject: "S3"})

	emails, err := GetReportedEmails(1)
	if err != nil {
		t.Fatalf("GetReportedEmails failed: %v", err)
	}
	if len(emails) != 2 {
		t.Fatalf("expected 2 reported emails for org 1, got %d", len(emails))
	}
}

// ---------- GetReportedEmail ----------

func TestGetReportedEmail(t *testing.T) {
	teardown := setupReportButtonTest(t)
	defer teardown()

	re := &ReportedEmail{OrgId: 1, ReporterEmail: testReporterEmail, Subject: "Test"}
	CreateReportedEmail(re)

	fetched, err := GetReportedEmail(re.Id, 1)
	if err != nil {
		t.Fatalf("GetReportedEmail failed: %v", err)
	}
	if fetched.Subject != "Test" {
		t.Fatalf("expected subject 'Test', got %q", fetched.Subject)
	}
}

func TestGetReportedEmailWrongOrg(t *testing.T) {
	teardown := setupReportButtonTest(t)
	defer teardown()

	re := &ReportedEmail{OrgId: 1, ReporterEmail: testReporterEmail, Subject: "Test"}
	CreateReportedEmail(re)

	_, err := GetReportedEmail(re.Id, 999)
	if err == nil {
		t.Fatal("expected error when fetching from wrong org")
	}
}

// ---------- ClassifyReportedEmail ----------

func TestClassifyReportedEmail(t *testing.T) {
	teardown := setupReportButtonTest(t)
	defer teardown()

	re := &ReportedEmail{OrgId: 1, ReporterEmail: testReporterEmail, Subject: "Test"}
	CreateReportedEmail(re)

	err := ClassifyReportedEmail(re.Id, 1, "phishing", "Confirmed phishing attempt")
	if err != nil {
		t.Fatalf("ClassifyReportedEmail failed: %v", err)
	}

	fetched, _ := GetReportedEmail(re.Id, 1)
	if fetched.Classification != "phishing" {
		t.Fatalf("expected classification 'phishing', got %q", fetched.Classification)
	}
	if fetched.AdminNotes != "Confirmed phishing attempt" {
		t.Fatalf("expected admin notes, got %q", fetched.AdminNotes)
	}
}
