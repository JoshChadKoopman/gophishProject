package models

import (
	"testing"
	"time"

	"github.com/gophish/gophish/config"
)

// Shared test constant for AI config tests.
const aiConfigFmtUnexpectedErr = "unexpected error: %v"

// setupAIConfigTest initialises an in-memory DB for AI config tests.
func setupAIConfigTest(t *testing.T) func() {
	t.Helper()
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	if err := Setup(conf); err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	db.Exec("DELETE FROM ai_generation_logs")
	return func() {
		db.Exec("DELETE FROM ai_generation_logs")
	}
}

// ---------- AIGenerationLog CRUD ----------

func TestCreateAIGenerationLog(t *testing.T) {
	teardown := setupAIConfigTest(t)
	defer teardown()

	entry := &AIGenerationLog{
		OrgId:        1,
		UserId:       1,
		Provider:     "claude",
		ModelUsed:    "claude-sonnet-4-20250514",
		InputTokens:  100,
		OutputTokens: 200,
		TemplateId:   10,
	}
	if err := CreateAIGenerationLog(entry); err != nil {
		t.Fatalf("failed to create log: %v", err)
	}
	if entry.Id == 0 {
		t.Fatal("expected non-zero ID")
	}
	if entry.CreatedDate.IsZero() {
		t.Fatal("expected CreatedDate to be set")
	}
}

func TestGetAIGenerationLogs(t *testing.T) {
	teardown := setupAIConfigTest(t)
	defer teardown()

	// Create 3 entries for org 1
	for i := 0; i < 3; i++ {
		CreateAIGenerationLog(&AIGenerationLog{
			OrgId:        1,
			UserId:       1,
			Provider:     "openai",
			ModelUsed:    "gpt-4o",
			InputTokens:  50 * (i + 1),
			OutputTokens: 100 * (i + 1),
		})
	}
	// Create 1 entry for org 2
	CreateAIGenerationLog(&AIGenerationLog{
		OrgId:    2,
		UserId:   2,
		Provider: "claude",
	})

	// Fetch org 1 logs
	logs, err := GetAIGenerationLogs(1, 0)
	if err != nil {
		t.Fatalf(aiConfigFmtUnexpectedErr, err)
	}
	if len(logs) != 3 {
		t.Fatalf("expected 3 logs for org 1, got %d", len(logs))
	}

	// Should be most recent first
	if logs[0].InputTokens < logs[2].InputTokens {
		t.Fatal("expected logs to be ordered by created_date desc")
	}

	// Fetch org 2 logs
	logs2, err := GetAIGenerationLogs(2, 0)
	if err != nil {
		t.Fatalf(aiConfigFmtUnexpectedErr, err)
	}
	if len(logs2) != 1 {
		t.Fatalf("expected 1 log for org 2, got %d", len(logs2))
	}
}

func TestGetAIGenerationLogsLimit(t *testing.T) {
	teardown := setupAIConfigTest(t)
	defer teardown()

	for i := 0; i < 10; i++ {
		CreateAIGenerationLog(&AIGenerationLog{
			OrgId:    1,
			UserId:   1,
			Provider: "openai",
		})
	}

	logs, err := GetAIGenerationLogs(1, 3)
	if err != nil {
		t.Fatalf(aiConfigFmtUnexpectedErr, err)
	}
	if len(logs) != 3 {
		t.Fatalf("expected 3 logs with limit 3, got %d", len(logs))
	}
}

func TestGetAIGenerationLogsEmpty(t *testing.T) {
	teardown := setupAIConfigTest(t)
	defer teardown()

	logs, err := GetAIGenerationLogs(999, 0)
	if err != nil {
		t.Fatalf(aiConfigFmtUnexpectedErr, err)
	}
	if len(logs) != 0 {
		t.Fatalf("expected 0 logs for non-existent org, got %d", len(logs))
	}
}

// ---------- AI Usage Summary ----------

func TestGetAIUsageSummaryBasic(t *testing.T) {
	teardown := setupAIConfigTest(t)
	defer teardown()

	// Create logs with known token counts
	for _, tokens := range []struct{ input, output int }{
		{100, 200},
		{150, 300},
		{50, 100},
	} {
		CreateAIGenerationLog(&AIGenerationLog{
			OrgId:        1,
			UserId:       1,
			Provider:     "claude",
			InputTokens:  tokens.input,
			OutputTokens: tokens.output,
		})
	}

	since := time.Now().UTC().Add(-1 * time.Hour)
	summary, err := GetAIUsageSummary(1, since)
	if err != nil {
		t.Fatalf(aiConfigFmtUnexpectedErr, err)
	}
	if summary.TotalGenerations != 3 {
		t.Fatalf("expected 3 generations, got %d", summary.TotalGenerations)
	}
	if summary.TotalInputTokens != 300 {
		t.Fatalf("expected 300 input tokens, got %d", summary.TotalInputTokens)
	}
	if summary.TotalOutputTokens != 600 {
		t.Fatalf("expected 600 output tokens, got %d", summary.TotalOutputTokens)
	}
	if summary.TotalTokens != 900 {
		t.Fatalf("expected 900 total tokens, got %d", summary.TotalTokens)
	}
}

func TestGetAIUsageSummaryOrgIsolation(t *testing.T) {
	teardown := setupAIConfigTest(t)
	defer teardown()

	CreateAIGenerationLog(&AIGenerationLog{OrgId: 1, UserId: 1, InputTokens: 100, OutputTokens: 200})
	CreateAIGenerationLog(&AIGenerationLog{OrgId: 2, UserId: 2, InputTokens: 500, OutputTokens: 500})

	since := time.Now().UTC().Add(-1 * time.Hour)
	summary, _ := GetAIUsageSummary(1, since)
	if summary.TotalGenerations != 1 {
		t.Fatalf("expected 1 generation for org 1, got %d", summary.TotalGenerations)
	}
	if summary.TotalInputTokens != 100 {
		t.Fatalf("expected 100 input tokens for org 1, got %d", summary.TotalInputTokens)
	}
}

func TestGetAIUsageSummaryTimeFilter(t *testing.T) {
	teardown := setupAIConfigTest(t)
	defer teardown()

	CreateAIGenerationLog(&AIGenerationLog{OrgId: 1, UserId: 1, InputTokens: 100, OutputTokens: 200})

	// Use a future "since" date — should return 0
	futureDate := time.Now().UTC().Add(24 * time.Hour)
	summary, _ := GetAIUsageSummary(1, futureDate)
	if summary.TotalGenerations != 0 {
		t.Fatalf("expected 0 generations after future date, got %d", summary.TotalGenerations)
	}
}

func TestGetAIUsageSummaryEmpty(t *testing.T) {
	teardown := setupAIConfigTest(t)
	defer teardown()

	since := time.Now().UTC().Add(-1 * time.Hour)
	summary, err := GetAIUsageSummary(999, since)
	if err != nil {
		t.Fatalf(aiConfigFmtUnexpectedErr, err)
	}
	if summary.TotalGenerations != 0 {
		t.Fatalf("expected 0 generations for non-existent org, got %d", summary.TotalGenerations)
	}
	if summary.TotalTokens != 0 {
		t.Fatalf("expected 0 total tokens, got %d", summary.TotalTokens)
	}
}
