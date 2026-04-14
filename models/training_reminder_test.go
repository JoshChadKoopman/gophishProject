package models

import (
	"testing"

	"github.com/gophish/gophish/config"
)

func setupTrainingReminderTest(t *testing.T) func() {
	t.Helper()
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	if err := Setup(conf); err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	return func() {}
}

func TestTrainingReminderTableName(t *testing.T) {
	r := TrainingReminder{}
	if r.TableName() != "training_reminders" {
		t.Fatalf("expected 'training_reminders', got %q", r.TableName())
	}
}

func TestReminderConfigTableName(t *testing.T) {
	c := ReminderConfig{}
	if c.TableName() != "reminder_configs" {
		t.Fatalf("expected 'reminder_configs', got %q", c.TableName())
	}
}

func TestGetReminderConfigDefaults(t *testing.T) {
	teardown := setupTrainingReminderTest(t)
	defer teardown()

	cfg := GetReminderConfig(9999) // non-existent org
	if !cfg.Enabled {
		t.Fatal("expected default config to be enabled")
	}
	if cfg.FirstReminderHours != DefaultFirstReminderHours {
		t.Fatalf("expected %d first reminder hours, got %d", DefaultFirstReminderHours, cfg.FirstReminderHours)
	}
	if cfg.SecondReminderHours != DefaultSecondReminderHours {
		t.Fatalf("expected %d second reminder hours, got %d", DefaultSecondReminderHours, cfg.SecondReminderHours)
	}
	if cfg.UrgentReminderHours != DefaultUrgentReminderHours {
		t.Fatalf("expected %d urgent reminder hours, got %d", DefaultUrgentReminderHours, cfg.UrgentReminderHours)
	}
}

func TestSaveAndGetReminderConfig(t *testing.T) {
	teardown := setupTrainingReminderTest(t)
	defer teardown()

	cfg := &ReminderConfig{
		OrgId:               1,
		Enabled:             false,
		FirstReminderHours:  72,
		SecondReminderHours: 36,
		UrgentReminderHours: 6,
		EscalateOverdueDays: 5,
		SendingProfileId:    0,
	}
	if err := SaveReminderConfig(cfg); err != nil {
		t.Fatalf("SaveReminderConfig: %v", err)
	}

	got := GetReminderConfig(1)
	if got.Enabled {
		t.Fatal("expected config to be disabled")
	}
	if got.FirstReminderHours != 72 {
		t.Fatalf("expected 72 first reminder hours, got %d", got.FirstReminderHours)
	}
}

func TestCreateTrainingReminder(t *testing.T) {
	teardown := setupTrainingReminderTest(t)
	defer teardown()

	r := &TrainingReminder{
		UserId:       1,
		AssignmentId: 10,
		CourseName:   "Test Course",
		ReminderType: "standard",
		Message:      "Please complete",
		EmailSent:    false,
	}
	if err := CreateTrainingReminder(r); err != nil {
		t.Fatalf("CreateTrainingReminder: %v", err)
	}
	if r.Id == 0 {
		t.Fatal("expected reminder to have an ID after creation")
	}
}

func TestGetUserReminders(t *testing.T) {
	teardown := setupTrainingReminderTest(t)
	defer teardown()

	// Create a reminder
	r := &TrainingReminder{
		UserId:       1,
		AssignmentId: 10,
		CourseName:   "Test",
		ReminderType: "standard",
		Message:      "Test",
	}
	CreateTrainingReminder(r)

	reminders, err := GetUserReminders(1, 10)
	if err != nil {
		t.Fatalf("GetUserReminders: %v", err)
	}
	if len(reminders) < 1 {
		t.Fatal("expected at least 1 reminder")
	}
}

func TestGetUserRemindersEmpty(t *testing.T) {
	teardown := setupTrainingReminderTest(t)
	defer teardown()

	reminders, err := GetUserReminders(9999, 10)
	if err != nil {
		t.Fatalf("GetUserReminders: %v", err)
	}
	if len(reminders) != 0 {
		t.Fatalf("expected 0 reminders, got %d", len(reminders))
	}
}

func TestGetReminderStatsForOrg(t *testing.T) {
	teardown := setupTrainingReminderTest(t)
	defer teardown()

	stats := GetReminderStatsForOrg(1)
	// Default org with no reminders
	if stats.TotalSent != 0 {
		t.Fatalf("expected 0 total sent, got %d", stats.TotalSent)
	}
}

func TestDefaultReminderConstants(t *testing.T) {
	if DefaultFirstReminderHours != 48 {
		t.Fatalf("expected 48, got %d", DefaultFirstReminderHours)
	}
	if DefaultSecondReminderHours != 24 {
		t.Fatalf("expected 24, got %d", DefaultSecondReminderHours)
	}
	if DefaultUrgentReminderHours != 4 {
		t.Fatalf("expected 4, got %d", DefaultUrgentReminderHours)
	}
	if DefaultEscalateOverdueDays != 2 {
		t.Fatalf("expected 2, got %d", DefaultEscalateOverdueDays)
	}
}
