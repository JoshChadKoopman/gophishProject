package models

import (
	"testing"

	"github.com/gophish/gophish/config"
)

func setupPraiseMessageTest(t *testing.T) func() {
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

func TestPraiseEventConstants(t *testing.T) {
	if PraiseEventCourseComplete != "course_complete" {
		t.Fatalf("expected 'course_complete', got %q", PraiseEventCourseComplete)
	}
	if PraiseEventQuizPassed != "quiz_passed" {
		t.Fatalf("expected 'quiz_passed', got %q", PraiseEventQuizPassed)
	}
	if PraiseEventCertEarned != "cert_earned" {
		t.Fatalf("expected 'cert_earned', got %q", PraiseEventCertEarned)
	}
	if PraiseEventTierComplete != "tier_complete" {
		t.Fatalf("expected 'tier_complete', got %q", PraiseEventTierComplete)
	}
}

func TestDefaultPraiseMessages(t *testing.T) {
	defaults := DefaultPraiseMessages()
	if len(defaults) != 4 {
		t.Fatalf("expected 4 default praise messages, got %d", len(defaults))
	}

	eventTypes := map[string]bool{}
	for _, m := range defaults {
		eventTypes[m.EventType] = true
		if m.Heading == "" {
			t.Fatalf("expected non-empty heading for event type %q", m.EventType)
		}
		if m.Body == "" {
			t.Fatalf("expected non-empty body for event type %q", m.EventType)
		}
		if m.Icon == "" {
			t.Fatalf("expected non-empty icon for event type %q", m.EventType)
		}
		if !m.IsActive {
			t.Fatalf("expected default message to be active for %q", m.EventType)
		}
	}

	for _, et := range []string{PraiseEventCourseComplete, PraiseEventQuizPassed, PraiseEventCertEarned, PraiseEventTierComplete} {
		if !eventTypes[et] {
			t.Fatalf("missing default praise message for event type %q", et)
		}
	}
}

func TestGetPraiseMessages(t *testing.T) {
	teardown := setupPraiseMessageTest(t)
	defer teardown()

	msgs, err := GetPraiseMessages(1)
	if err != nil {
		t.Fatalf("GetPraiseMessages: %v", err)
	}
	// Should return defaults or seeded messages for org 1
	_ = msgs
}

func TestGetPraiseMessagesNonExistentOrg(t *testing.T) {
	teardown := setupPraiseMessageTest(t)
	defer teardown()

	msgs, err := GetPraiseMessages(9999)
	if err != nil {
		t.Fatalf("GetPraiseMessages for non-existent org: %v", err)
	}
	// Should return defaults or empty — no panic
	_ = msgs
}
