package models

import (
	"testing"
	"time"

	"github.com/gophish/gophish/config"
)

// Shared test constants for autopilot tests.
const (
	autopilotFmtUnexpectedErr = "unexpected error: %v"
	autopilotExpectedNonZero  = "expected non-zero ID"
	autopilotTestDate         = "2026-12-25"
	autopilotTestEmail        = "alice@example.com"
)

// setupAutopilotTest initialises an in-memory DB for autopilot tests.
func setupAutopilotTest(t *testing.T) func() {
	t.Helper()
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	if err := Setup(conf); err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	db.Exec("DELETE FROM autopilot_blackout_dates")
	db.Exec("DELETE FROM autopilot_schedules")
	db.Exec("DELETE FROM autopilot_configs")
	return func() {
		db.Exec("DELETE FROM autopilot_blackout_dates")
		db.Exec("DELETE FROM autopilot_schedules")
		db.Exec("DELETE FROM autopilot_configs")
	}
}

// ---------- AutopilotConfig CRUD ----------

func TestSaveAutopilotConfigNew(t *testing.T) {
	teardown := setupAutopilotTest(t)
	defer teardown()

	ac := &AutopilotConfig{
		OrgId:            1,
		Enabled:          false,
		CadenceDays:      15,
		ActiveHoursStart: 9,
		ActiveHoursEnd:   17,
		Timezone:         "Europe/Amsterdam",
		SendingProfileId: 1,
		LandingPageId:    1,
		PhishURL:         "https://phish.example.com",
	}
	ac.SetGroupIds([]int64{1, 2, 3})

	if err := SaveAutopilotConfig(ac); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}
	if ac.Id == 0 {
		t.Fatal(autopilotExpectedNonZero)
	}
	if ac.CreatedDate.IsZero() {
		t.Fatal("expected CreatedDate to be set")
	}
	if ac.ModifiedDate.IsZero() {
		t.Fatal("expected ModifiedDate to be set")
	}
}

func TestSaveAutopilotConfigDefaults(t *testing.T) {
	teardown := setupAutopilotTest(t)
	defer teardown()

	// Save with invalid values — should apply defaults
	ac := &AutopilotConfig{
		OrgId:            1,
		CadenceDays:      0,  // should default to 15
		ActiveHoursStart: -1, // should default to 9
		ActiveHoursEnd:   25, // should default to 17
		Timezone:         "", // should default to "UTC"
	}
	SaveAutopilotConfig(ac)

	found, _ := GetAutopilotConfig(1)
	if found.CadenceDays != 15 {
		t.Fatalf("expected cadence_days 15, got %d", found.CadenceDays)
	}
	if found.ActiveHoursStart != 9 {
		t.Fatalf("expected active_hours_start 9, got %d", found.ActiveHoursStart)
	}
	if found.ActiveHoursEnd != 17 {
		t.Fatalf("expected active_hours_end 17, got %d", found.ActiveHoursEnd)
	}
	if found.Timezone != "UTC" {
		t.Fatalf("expected timezone 'UTC', got %q", found.Timezone)
	}
}

func TestSaveAutopilotConfigUpsert(t *testing.T) {
	teardown := setupAutopilotTest(t)
	defer teardown()

	// Create
	ac := &AutopilotConfig{OrgId: 1, CadenceDays: 10, Timezone: "UTC"}
	SaveAutopilotConfig(ac)
	origId := ac.Id

	// Update
	ac2 := &AutopilotConfig{OrgId: 1, CadenceDays: 30, Timezone: "US/Eastern"}
	SaveAutopilotConfig(ac2)

	found, _ := GetAutopilotConfig(1)
	if found.Id != origId {
		t.Fatalf("expected same ID after upsert: %d vs %d", origId, found.Id)
	}
	if found.CadenceDays != 30 {
		t.Fatalf("expected updated cadence_days 30, got %d", found.CadenceDays)
	}
}

func TestGetAutopilotConfigNotFound(t *testing.T) {
	teardown := setupAutopilotTest(t)
	defer teardown()

	_, err := GetAutopilotConfig(999)
	if err == nil {
		t.Fatal("expected error for non-existent config")
	}
}

// ---------- Enable / Disable ----------

func TestEnableAutopilot(t *testing.T) {
	teardown := setupAutopilotTest(t)
	defer teardown()

	ac := &AutopilotConfig{OrgId: 1, CadenceDays: 15, Timezone: "UTC"}
	SaveAutopilotConfig(ac)

	if err := EnableAutopilot(1); err != nil {
		t.Fatalf("failed to enable autopilot: %v", err)
	}

	found, _ := GetAutopilotConfig(1)
	if !found.Enabled {
		t.Fatal("expected autopilot to be enabled")
	}
	if found.NextRun.IsZero() {
		t.Fatal("expected NextRun to be set after enabling")
	}
}

func TestEnableAutopilotNotConfigured(t *testing.T) {
	teardown := setupAutopilotTest(t)
	defer teardown()

	err := EnableAutopilot(999)
	if err != ErrAutopilotNotConfigured {
		t.Fatalf("expected ErrAutopilotNotConfigured, got %v", err)
	}
}

func TestDisableAutopilot(t *testing.T) {
	teardown := setupAutopilotTest(t)
	defer teardown()

	ac := &AutopilotConfig{OrgId: 1, Enabled: true, CadenceDays: 15, Timezone: "UTC"}
	SaveAutopilotConfig(ac)
	EnableAutopilot(1)

	if err := DisableAutopilot(1); err != nil {
		t.Fatalf("failed to disable autopilot: %v", err)
	}

	found, _ := GetAutopilotConfig(1)
	if found.Enabled {
		t.Fatal("expected autopilot to be disabled")
	}
}

// ---------- Group IDs ----------

func TestAutopilotGroupIds(t *testing.T) {
	ac := AutopilotConfig{}

	// Empty
	ids := ac.GetGroupIds()
	if len(ids) != 0 {
		t.Fatalf("expected 0 group IDs, got %d", len(ids))
	}

	// Set and get
	ac.SetGroupIds([]int64{10, 20, 30})
	ids = ac.GetGroupIds()
	if len(ids) != 3 {
		t.Fatalf("expected 3 group IDs, got %d", len(ids))
	}
	if ids[0] != 10 || ids[1] != 20 || ids[2] != 30 {
		t.Fatalf("unexpected group IDs: %v", ids)
	}
}

func TestAutopilotGroupIdsNull(t *testing.T) {
	ac := AutopilotConfig{TargetGroupIds: "null"}
	ids := ac.GetGroupIds()
	if len(ids) != 0 {
		t.Fatalf("expected 0 group IDs for 'null', got %d", len(ids))
	}
}

// ---------- Autopilot Schedules ----------

func TestCreateAutopilotSchedule(t *testing.T) {
	teardown := setupAutopilotTest(t)
	defer teardown()

	s := &AutopilotSchedule{
		OrgId:           1,
		UserEmail:       "user@example.com",
		CampaignId:      42,
		DifficultyLevel: 2,
		ScheduledDate:   time.Now().UTC().Add(24 * time.Hour),
	}
	if err := CreateAutopilotSchedule(s); err != nil {
		t.Fatalf("failed to create schedule: %v", err)
	}
	if s.Id == 0 {
		t.Fatal(autopilotExpectedNonZero)
	}
	if s.CreatedDate.IsZero() {
		t.Fatal("expected CreatedDate to be set")
	}
}

func TestGetAutopilotSchedule(t *testing.T) {
	teardown := setupAutopilotTest(t)
	defer teardown()

	for i := 0; i < 3; i++ {
		CreateAutopilotSchedule(&AutopilotSchedule{
			OrgId:         1,
			UserEmail:     "user@example.com",
			ScheduledDate: time.Now().UTC().Add(time.Duration(i) * time.Hour),
		})
	}
	CreateAutopilotSchedule(&AutopilotSchedule{
		OrgId:         2,
		UserEmail:     "other@example.com",
		ScheduledDate: time.Now().UTC(),
	})

	entries, err := GetAutopilotSchedule(1, 0)
	if err != nil {
		t.Fatalf(autopilotFmtUnexpectedErr, err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 schedule entries for org 1, got %d", len(entries))
	}
}

// ---------- Blackout Dates ----------

func TestCreateAutopilotBlackoutDate(t *testing.T) {
	teardown := setupAutopilotTest(t)
	defer teardown()

	d := &AutopilotBlackoutDate{
		OrgId:  1,
		Date:   autopilotTestDate,
		Reason: "Christmas",
	}
	if err := CreateAutopilotBlackoutDate(d); err != nil {
		t.Fatalf("failed to create blackout date: %v", err)
	}
	if d.Id == 0 {
		t.Fatal(autopilotExpectedNonZero)
	}
}

func TestGetAutopilotBlackoutDates(t *testing.T) {
	teardown := setupAutopilotTest(t)
	defer teardown()

	for _, date := range []string{autopilotTestDate, "2026-01-01", "2026-04-27"} {
		CreateAutopilotBlackoutDate(&AutopilotBlackoutDate{
			OrgId:  1,
			Date:   date,
			Reason: "Holiday",
		})
	}

	dates, err := GetAutopilotBlackoutDates(1)
	if err != nil {
		t.Fatalf(autopilotFmtUnexpectedErr, err)
	}
	if len(dates) != 3 {
		t.Fatalf("expected 3 blackout dates, got %d", len(dates))
	}
	// Should be sorted by date asc
	if dates[0].Date != "2026-01-01" {
		t.Fatalf("expected first date '2026-01-01', got %q", dates[0].Date)
	}
}

func TestDeleteAutopilotBlackoutDate(t *testing.T) {
	teardown := setupAutopilotTest(t)
	defer teardown()

	d := &AutopilotBlackoutDate{OrgId: 1, Date: autopilotTestDate, Reason: "Christmas"}
	CreateAutopilotBlackoutDate(d)

	if err := DeleteAutopilotBlackoutDate(d.Id, 1); err != nil {
		t.Fatalf("failed to delete blackout date: %v", err)
	}

	dates, _ := GetAutopilotBlackoutDates(1)
	if len(dates) != 0 {
		t.Fatalf("expected 0 blackout dates after deletion, got %d", len(dates))
	}
}

func TestDeleteAutopilotBlackoutDateOrgIsolation(t *testing.T) {
	teardown := setupAutopilotTest(t)
	defer teardown()

	// Create for org 1
	d := &AutopilotBlackoutDate{OrgId: 1, Date: autopilotTestDate, Reason: "Christmas"}
	CreateAutopilotBlackoutDate(d)

	// Try to delete from org 2 — should not delete org 1's record
	DeleteAutopilotBlackoutDate(d.Id, 2)

	dates, _ := GetAutopilotBlackoutDates(1)
	if len(dates) != 1 {
		t.Fatalf("expected 1 blackout date still present for org 1, got %d", len(dates))
	}
}

func TestIsBlackoutDate(t *testing.T) {
	teardown := setupAutopilotTest(t)
	defer teardown()

	CreateAutopilotBlackoutDate(&AutopilotBlackoutDate{OrgId: 1, Date: autopilotTestDate})

	xmas := time.Date(2026, 12, 25, 10, 0, 0, 0, time.UTC)
	if !IsBlackoutDate(1, xmas) {
		t.Fatal("expected 2026-12-25 to be a blackout date")
	}

	notBlackout := time.Date(2026, 12, 26, 10, 0, 0, 0, time.UTC)
	if IsBlackoutDate(1, notBlackout) {
		t.Fatal("expected 2026-12-26 to NOT be a blackout date")
	}
}

// ---------- Enabled Autopilots ----------

func TestGetEnabledAutopilots(t *testing.T) {
	teardown := setupAutopilotTest(t)
	defer teardown()

	// Create 2 enabled, 1 disabled
	for i, enabled := range []bool{true, true, false} {
		ac := &AutopilotConfig{
			OrgId:       int64(i + 1),
			Enabled:     enabled,
			CadenceDays: 15,
			Timezone:    "UTC",
		}
		SaveAutopilotConfig(ac)
		if enabled {
			// Set next_run to the past so it shows up
			db.Model(ac).Update("next_run", time.Now().UTC().Add(-1*time.Hour))
		}
	}

	enabled, err := GetEnabledAutopilots(time.Now().UTC())
	if err != nil {
		t.Fatalf(autopilotFmtUnexpectedErr, err)
	}
	if len(enabled) != 2 {
		t.Fatalf("expected 2 enabled autopilots, got %d", len(enabled))
	}
}

func TestGetEnabledAutopilotsIgnoresFuture(t *testing.T) {
	teardown := setupAutopilotTest(t)
	defer teardown()

	ac := &AutopilotConfig{
		OrgId:       1,
		Enabled:     true,
		CadenceDays: 15,
		Timezone:    "UTC",
	}
	SaveAutopilotConfig(ac)
	// Set next_run to the future
	db.Model(ac).Update("next_run", time.Now().UTC().Add(24*time.Hour))

	enabled, _ := GetEnabledAutopilots(time.Now().UTC())
	if len(enabled) != 0 {
		t.Fatalf("expected 0 enabled autopilots (next_run is future), got %d", len(enabled))
	}
}

// ---------- Update Autopilot Run ----------

func TestUpdateAutopilotRun(t *testing.T) {
	teardown := setupAutopilotTest(t)
	defer teardown()

	ac := &AutopilotConfig{OrgId: 1, CadenceDays: 15, Timezone: "UTC", ActiveHoursStart: 9, ActiveHoursEnd: 17}
	SaveAutopilotConfig(ac)
	EnableAutopilot(1)

	found, _ := GetAutopilotConfig(1)
	if err := UpdateAutopilotRun(&found); err != nil {
		t.Fatalf("failed to update run: %v", err)
	}

	updated, _ := GetAutopilotConfig(1)
	if updated.LastRun.IsZero() {
		t.Fatal("expected LastRun to be set")
	}
	if updated.NextRun.IsZero() {
		t.Fatal("expected NextRun to be set")
	}
	if !updated.NextRun.After(updated.LastRun) {
		t.Fatal("expected NextRun to be after LastRun")
	}
}

// ---------- Users Last Sent Date ----------

func TestGetUsersLastSentDate(t *testing.T) {
	teardown := setupAutopilotTest(t)
	defer teardown()

	// Create schedule entries
	past := time.Now().UTC().Add(-48 * time.Hour)
	recent := time.Now().UTC().Add(-1 * time.Hour)

	CreateAutopilotSchedule(&AutopilotSchedule{
		OrgId: 1, UserEmail: autopilotTestEmail, Sent: true,
		ScheduledDate: past,
	})
	CreateAutopilotSchedule(&AutopilotSchedule{
		OrgId: 1, UserEmail: autopilotTestEmail, Sent: true,
		ScheduledDate: recent,
	})
	CreateAutopilotSchedule(&AutopilotSchedule{
		OrgId: 1, UserEmail: "bob@example.com", Sent: true,
		ScheduledDate: past,
	})
	// Unsent entry should be ignored
	CreateAutopilotSchedule(&AutopilotSchedule{
		OrgId: 1, UserEmail: "carol@example.com", Sent: false,
		ScheduledDate: recent,
	})

	result, err := GetUsersLastSentDate(1)
	if err != nil {
		t.Fatalf(autopilotFmtUnexpectedErr, err)
	}
	if len(result) != 2 {
		t.Fatalf("expected 2 users with sent dates, got %d", len(result))
	}
	aliceDate, ok := result[autopilotTestEmail]
	if !ok {
		t.Fatal("expected alice in results")
	}
	// Alice's last date should be the more recent one
	if aliceDate.Before(past.Add(1 * time.Hour)) {
		t.Fatalf("expected alice's last date to be recent, got %v", aliceDate)
	}
}

// ---------- End-to-end Autopilot Flow ----------

func TestAutopilotEndToEndFlow(t *testing.T) {
	teardown := setupAutopilotTest(t)
	defer teardown()

	// 1. Create config
	ac := &AutopilotConfig{
		OrgId:            1,
		CadenceDays:      7,
		ActiveHoursStart: 8,
		ActiveHoursEnd:   18,
		Timezone:         "Europe/Amsterdam",
		SendingProfileId: 1,
		LandingPageId:    1,
		PhishURL:         "https://phish.test.com",
	}
	ac.SetGroupIds([]int64{1, 2})
	if err := SaveAutopilotConfig(ac); err != nil {
		t.Fatalf("failed to save config: %v", err)
	}

	// 2. Enable
	if err := EnableAutopilot(1); err != nil {
		t.Fatalf("failed to enable: %v", err)
	}

	// 3. Verify enabled and next_run set
	found, _ := GetAutopilotConfig(1)
	if !found.Enabled {
		t.Fatal("expected autopilot to be enabled")
	}
	if found.NextRun.IsZero() {
		t.Fatal("expected NextRun to be set")
	}

	// 4. Add blackout date
	CreateAutopilotBlackoutDate(&AutopilotBlackoutDate{
		OrgId: 1, Date: autopilotTestDate, Reason: "Christmas",
	})

	// 5. Check blackout
	xmas := time.Date(2026, 12, 25, 10, 0, 0, 0, time.UTC)
	if !IsBlackoutDate(1, xmas) {
		t.Fatal("expected Christmas to be blackout")
	}

	// 6. Create a schedule entry
	CreateAutopilotSchedule(&AutopilotSchedule{
		OrgId:           1,
		UserEmail:       "target@example.com",
		CampaignId:      100,
		DifficultyLevel: 2,
		ScheduledDate:   time.Now().UTC(),
		Sent:            true,
	})

	// 7. Check last sent
	lastSent, _ := GetUsersLastSentDate(1)
	if _, ok := lastSent["target@example.com"]; !ok {
		t.Fatal("expected target user in last sent dates")
	}

	// 8. Update run
	UpdateAutopilotRun(&found)
	updated, _ := GetAutopilotConfig(1)
	if updated.LastRun.IsZero() {
		t.Fatal("expected LastRun to be updated")
	}

	// 9. Disable
	DisableAutopilot(1)
	disabled, _ := GetAutopilotConfig(1)
	if disabled.Enabled {
		t.Fatal("expected autopilot to be disabled")
	}
}
