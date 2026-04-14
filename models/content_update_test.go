package models

import (
	"testing"
	"time"

	"github.com/gophish/gophish/config"
)

func setupContentUpdateTest(t *testing.T) func() {
	t.Helper()
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	if err := Setup(conf); err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	return func() { /* in-memory DB cleaned up automatically */ }
}

func TestContentUpdateConfigTableName(t *testing.T) {
	c := ContentUpdateConfig{}
	if c.TableName() != "content_update_configs" {
		t.Fatalf("expected 'content_update_configs', got %q", c.TableName())
	}
}

func TestContentUpdateLogTableName(t *testing.T) {
	l := ContentUpdateLog{}
	if l.TableName() != "content_update_logs" {
		t.Fatalf("expected 'content_update_logs', got %q", l.TableName())
	}
}

func TestGetContentUpdateConfigDefaults(t *testing.T) {
	teardown := setupContentUpdateTest(t)
	defer teardown()

	cfg := GetContentUpdateConfig(9999)
	if !cfg.Enabled {
		t.Fatal("expected default config to be enabled")
	}
	if cfg.AutoAssignNew != DefaultAutoAssignNew {
		t.Fatalf("expected AutoAssignNew=%v, got %v", DefaultAutoAssignNew, cfg.AutoAssignNew)
	}
	if cfg.NotifyAdmins != DefaultNotifyAdmins {
		t.Fatalf("expected NotifyAdmins=%v, got %v", DefaultNotifyAdmins, cfg.NotifyAdmins)
	}
}

func TestSaveAndGetContentUpdateConfig(t *testing.T) {
	teardown := setupContentUpdateTest(t)
	defer teardown()

	cfg := &ContentUpdateConfig{
		OrgId:             1,
		Enabled:           false,
		AutoAssignNew:     true,
		NotifyAdmins:      false,
		ContentCategories: "phishing,compliance",
		MinDifficulty:     1,
		MaxDifficulty:     3,
	}
	if err := SaveContentUpdateConfig(cfg); err != nil {
		t.Fatalf("SaveContentUpdateConfig: %v", err)
	}

	got := GetContentUpdateConfig(1)
	if got.Enabled {
		t.Fatal("expected disabled")
	}
	if !got.AutoAssignNew {
		t.Fatal("expected AutoAssignNew true")
	}
	if got.ContentCategories != "phishing,compliance" {
		t.Fatalf("expected 'phishing,compliance', got %q", got.ContentCategories)
	}
}

func TestRecordContentUpdate(t *testing.T) {
	teardown := setupContentUpdateTest(t)
	defer teardown()

	entry := &ContentUpdateLog{
		OrgId:        1,
		OrgName:      "Test Org",
		CoursesAdded: 3,
		Status:       "success",
	}
	RecordContentUpdate(entry)

	if entry.RunDate.IsZero() {
		t.Fatal("expected RunDate to be set")
	}
}

func TestGetContentUpdateHistory(t *testing.T) {
	teardown := setupContentUpdateTest(t)
	defer teardown()

	RecordContentUpdate(&ContentUpdateLog{OrgId: 1, OrgName: "Org1", Status: "success", CoursesAdded: 2})
	RecordContentUpdate(&ContentUpdateLog{OrgId: 1, OrgName: "Org1", Status: "partial", CoursesAdded: 1})

	logs, err := GetContentUpdateHistory(1, 10)
	if err != nil {
		t.Fatalf("GetContentUpdateHistory: %v", err)
	}
	if len(logs) < 2 {
		t.Fatalf("expected at least 2 logs, got %d", len(logs))
	}
	// Should be ordered by run_date desc
	if logs[0].RunDate.Before(logs[1].RunDate) {
		t.Fatal("expected descending order by run_date")
	}
}

func TestGetGlobalContentUpdateHistory(t *testing.T) {
	teardown := setupContentUpdateTest(t)
	defer teardown()

	RecordContentUpdate(&ContentUpdateLog{OrgId: 1, OrgName: "O1", Status: "success"})
	RecordContentUpdate(&ContentUpdateLog{OrgId: 2, OrgName: "O2", Status: "error", ErrorMessage: "test"})

	logs, err := GetGlobalContentUpdateHistory(50)
	if err != nil {
		t.Fatalf("GetGlobalContentUpdateHistory: %v", err)
	}
	if len(logs) < 2 {
		t.Fatalf("expected at least 2 logs, got %d", len(logs))
	}
}

func TestGetContentUpdateSummary(t *testing.T) {
	teardown := setupContentUpdateTest(t)
	defer teardown()

	summary := GetContentUpdateSummary(1)
	if summary.Config.OrgId != 1 {
		t.Fatalf("expected OrgId=1, got %d", summary.Config.OrgId)
	}
	_ = summary.LibrarySize // just ensure no panic
}

func TestContentUpdateLogStatusValues(t *testing.T) {
	teardown := setupContentUpdateTest(t)
	defer teardown()

	for _, status := range []string{"success", "partial", "error", "skipped"} {
		entry := &ContentUpdateLog{
			OrgId:   1,
			OrgName: "Test",
			Status:  status,
		}
		RecordContentUpdate(entry)
		if entry.RunDate.Before(time.Now().Add(-1 * time.Minute)) {
			t.Fatalf("expected recent RunDate for status %s", status)
		}
	}
}
