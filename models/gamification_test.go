package models

import (
	"testing"
	"time"

	"github.com/gophish/gophish/config"
)

// setupGamificationTest initialises an in-memory DB for gamification tests.
func setupGamificationTest(t *testing.T) func() {
	t.Helper()
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	if err := Setup(conf); err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	db.Exec("DELETE FROM leaderboard_cache")
	db.Exec("DELETE FROM user_streaks")
	db.Exec("DELETE FROM user_badges")
	db.Exec("DELETE FROM badges")
	return func() {
		db.Exec("DELETE FROM leaderboard_cache")
		db.Exec("DELETE FROM user_streaks")
		db.Exec("DELETE FROM user_badges")
		db.Exec("DELETE FROM badges")
	}
}

// ---------- Badge CRUD ----------

func TestGetAllBadges(t *testing.T) {
	teardown := setupGamificationTest(t)
	defer teardown()

	db.Create(&Badge{Slug: "first_course", Name: "First Course", Category: "course", CriteriaType: "courses_completed", CriteriaValue: 1, CreatedDate: time.Now()})
	db.Create(&Badge{Slug: "five_courses", Name: "Five Courses", Category: "course", CriteriaType: "courses_completed", CriteriaValue: 5, CreatedDate: time.Now()})

	badges, err := GetAllBadges()
	if err != nil {
		t.Fatalf("GetAllBadges failed: %v", err)
	}
	if len(badges) < 2 {
		t.Fatalf("expected at least 2 badges, got %d", len(badges))
	}
}

func TestGetUserBadgesEmpty(t *testing.T) {
	teardown := setupGamificationTest(t)
	defer teardown()

	ubs, err := GetUserBadges(999)
	if err != nil {
		t.Fatalf("GetUserBadges failed: %v", err)
	}
	if len(ubs) != 0 {
		t.Fatalf("expected 0 badges, got %d", len(ubs))
	}
}

// ---------- AwardBadge ----------

func TestAwardBadge(t *testing.T) {
	teardown := setupGamificationTest(t)
	defer teardown()

	db.Create(&Badge{Slug: "test_badge", Name: "Test Badge", Category: "test", CreatedDate: time.Now()})

	ub, awarded := AwardBadge(1, "test_badge")
	if !awarded {
		t.Fatal("expected badge to be awarded")
	}
	if ub == nil {
		t.Fatal("expected non-nil UserBadge")
	}
	if ub.BadgeName != "Test Badge" {
		t.Fatalf("expected BadgeName 'Test Badge', got %q", ub.BadgeName)
	}
}

func TestAwardBadgeDuplicate(t *testing.T) {
	teardown := setupGamificationTest(t)
	defer teardown()

	db.Create(&Badge{Slug: "unique_badge", Name: "Unique", Category: "test", CreatedDate: time.Now()})

	AwardBadge(1, "unique_badge")
	_, awarded := AwardBadge(1, "unique_badge")
	if awarded {
		t.Fatal("expected badge NOT to be awarded twice")
	}
}

func TestAwardBadgeNonExistent(t *testing.T) {
	teardown := setupGamificationTest(t)
	defer teardown()

	_, awarded := AwardBadge(1, "nonexistent")
	if awarded {
		t.Fatal("expected badge NOT to be awarded when slug doesn't exist")
	}
}

// ---------- HasBadge ----------

func TestHasBadge(t *testing.T) {
	teardown := setupGamificationTest(t)
	defer teardown()

	db.Create(&Badge{Slug: "check_badge", Name: "Check Badge", Category: "test", CreatedDate: time.Now()})

	if HasBadge(1, "check_badge") {
		t.Fatal("expected HasBadge to be false before awarding")
	}

	AwardBadge(1, "check_badge")

	if !HasBadge(1, "check_badge") {
		t.Fatal("expected HasBadge to be true after awarding")
	}
}

// ---------- GetUserBadgeCount ----------

func TestGetUserBadgeCount(t *testing.T) {
	teardown := setupGamificationTest(t)
	defer teardown()

	db.Create(&Badge{Slug: "b1", Name: "B1", Category: "test", CreatedDate: time.Now()})
	db.Create(&Badge{Slug: "b2", Name: "B2", Category: "test", CreatedDate: time.Now()})
	db.Create(&Badge{Slug: "b3", Name: "B3", Category: "test", CreatedDate: time.Now()})

	AwardBadge(1, "b1")
	AwardBadge(1, "b2")

	count := GetUserBadgeCount(1)
	if count != 2 {
		t.Fatalf("expected 2 badges, got %d", count)
	}
}

// ---------- GetUserBadges (with enrichment) ----------

func TestGetUserBadgesEnriched(t *testing.T) {
	teardown := setupGamificationTest(t)
	defer teardown()

	db.Create(&Badge{Slug: "enriched", Name: "Enriched Badge", Description: "Has details", IconURL: "/icon.png", Category: "test", CreatedDate: time.Now()})
	AwardBadge(1, "enriched")

	ubs, err := GetUserBadges(1)
	if err != nil {
		t.Fatalf("GetUserBadges failed: %v", err)
	}
	if len(ubs) != 1 {
		t.Fatalf("expected 1 badge, got %d", len(ubs))
	}
	if ubs[0].BadgeName != "Enriched Badge" {
		t.Fatalf("expected BadgeName 'Enriched Badge', got %q", ubs[0].BadgeName)
	}
	if ubs[0].BadgeDescription != "Has details" {
		t.Fatalf("expected BadgeDescription 'Has details', got %q", ubs[0].BadgeDescription)
	}
	if ubs[0].BadgeIconURL != "/icon.png" {
		t.Fatalf("expected BadgeIconURL '/icon.png', got %q", ubs[0].BadgeIconURL)
	}
}

// ---------- Streak functions ----------

func TestGetUserStreakNotFound(t *testing.T) {
	teardown := setupGamificationTest(t)
	defer teardown()

	_, err := GetUserStreak(999, "weekly")
	if err == nil {
		t.Fatal("expected error for non-existent streak")
	}
}

func TestRecordTrainingActivityCreatesStreak(t *testing.T) {
	teardown := setupGamificationTest(t)
	defer teardown()

	RecordTrainingActivity(1)

	streak, err := GetUserStreak(1, "weekly")
	if err != nil {
		t.Fatalf("GetUserStreak failed: %v", err)
	}
	if streak.CurrentStreak != 1 {
		t.Fatalf("expected CurrentStreak 1, got %d", streak.CurrentStreak)
	}
	if streak.LongestStreak != 1 {
		t.Fatalf("expected LongestStreak 1, got %d", streak.LongestStreak)
	}
}

func TestRecordTrainingActivitySameWeekNoIncrement(t *testing.T) {
	teardown := setupGamificationTest(t)
	defer teardown()

	RecordTrainingActivity(1)
	RecordTrainingActivity(1) // same week

	streak, _ := GetUserStreak(1, "weekly")
	if streak.CurrentStreak != 1 {
		t.Fatalf("expected CurrentStreak 1 (same week), got %d", streak.CurrentStreak)
	}
}

func TestGetUserStreaks(t *testing.T) {
	teardown := setupGamificationTest(t)
	defer teardown()

	RecordTrainingActivity(1) // Creates a "weekly" streak

	streaks, err := GetUserStreaks(1)
	if err != nil {
		t.Fatalf("GetUserStreaks failed: %v", err)
	}
	if len(streaks) != 1 {
		t.Fatalf("expected 1 streak, got %d", len(streaks))
	}
	if streaks[0].StreakType != "weekly" {
		t.Fatalf("expected 'weekly', got %q", streaks[0].StreakType)
	}
}

func TestExpireStaleStreaks(t *testing.T) {
	teardown := setupGamificationTest(t)
	defer teardown()

	// Create a streak with old activity date
	s := UserStreak{
		UserId:           1,
		StreakType:       "weekly",
		CurrentStreak:    5,
		LongestStreak:    5,
		LastActivityDate: time.Now().UTC().AddDate(0, 0, -20), // 20 days ago
		CreatedDate:      time.Now().UTC(),
	}
	db.Save(&s)

	ExpireStaleStreaks()

	updated, _ := GetUserStreak(1, "weekly")
	if updated.CurrentStreak != 0 {
		t.Fatalf("expected CurrentStreak 0 after expiry, got %d", updated.CurrentStreak)
	}
	// LongestStreak should be preserved
	if updated.LongestStreak != 5 {
		t.Fatalf("expected LongestStreak 5 preserved, got %d", updated.LongestStreak)
	}
}

// ---------- Leaderboard functions ----------

func TestGetLeaderboardEmpty(t *testing.T) {
	teardown := setupGamificationTest(t)
	defer teardown()

	entries, err := GetLeaderboard(1, "all_time", "", 0)
	if err != nil {
		t.Fatalf("GetLeaderboard failed: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("expected 0 entries, got %d", len(entries))
	}
}

func TestGetLeaderboardWithEntries(t *testing.T) {
	teardown := setupGamificationTest(t)
	defer teardown()

	now := time.Now().UTC()
	db.Save(&LeaderboardEntry{OrgId: 1, UserId: 1, Score: 100, Rank: 1, Period: "all_time", CalculatedDate: now})
	db.Save(&LeaderboardEntry{OrgId: 1, UserId: 2, Score: 80, Rank: 2, Period: "all_time", CalculatedDate: now})

	entries, err := GetLeaderboard(1, "all_time", "", 0)
	if err != nil {
		t.Fatalf("GetLeaderboard failed: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].Rank > entries[1].Rank {
		t.Fatal("expected entries ordered by rank ascending")
	}
}

func TestGetLeaderboardLimit(t *testing.T) {
	teardown := setupGamificationTest(t)
	defer teardown()

	now := time.Now().UTC()
	for i := 1; i <= 5; i++ {
		db.Save(&LeaderboardEntry{OrgId: 1, UserId: int64(i), Score: 100 - i, Rank: i, Period: "all_time", CalculatedDate: now})
	}

	entries, _ := GetLeaderboard(1, "all_time", "", 3)
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries (limit), got %d", len(entries))
	}
}

func TestGetLeaderboardByDepartment(t *testing.T) {
	teardown := setupGamificationTest(t)
	defer teardown()

	now := time.Now().UTC()
	db.Save(&LeaderboardEntry{OrgId: 1, Department: "Engineering", UserId: 1, Score: 90, Rank: 1, Period: "all_time", CalculatedDate: now})
	db.Save(&LeaderboardEntry{OrgId: 1, Department: "Marketing", UserId: 2, Score: 80, Rank: 2, Period: "all_time", CalculatedDate: now})

	entries, _ := GetLeaderboard(1, "all_time", "Engineering", 0)
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry for Engineering, got %d", len(entries))
	}
}

func TestGetUserLeaderboardPosition(t *testing.T) {
	teardown := setupGamificationTest(t)
	defer teardown()

	now := time.Now().UTC()
	db.Save(&LeaderboardEntry{OrgId: 1, UserId: 1, Score: 100, Rank: 1, Period: "all_time", CalculatedDate: now})

	entry, err := GetUserLeaderboardPosition(1, 1, "all_time")
	if err != nil {
		t.Fatalf("GetUserLeaderboardPosition failed: %v", err)
	}
	if entry.Rank != 1 {
		t.Fatalf("expected Rank 1, got %d", entry.Rank)
	}
	if entry.Score != 100 {
		t.Fatalf("expected Score 100, got %d", entry.Score)
	}
}

func TestGetUserLeaderboardPositionNotFound(t *testing.T) {
	teardown := setupGamificationTest(t)
	defer teardown()

	_, err := GetUserLeaderboardPosition(999, 1, "all_time")
	if err == nil {
		t.Fatal("expected error for user not in leaderboard")
	}
}
