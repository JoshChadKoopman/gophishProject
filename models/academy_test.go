package models

import (
	"testing"

	"github.com/gophish/gophish/config"
)

// setupAcademyTest initialises an in-memory DB for academy tests.
// It clears migration-seeded tiers so tests start with a clean slate.
func setupAcademyTest(t *testing.T) func() {
	t.Helper()
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	if err := Setup(conf); err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	// Clear migration-seeded data so each test starts from scratch
	db.Exec("DELETE FROM academy_sessions")
	db.Exec("DELETE FROM academy_user_progress")
	db.Exec("DELETE FROM academy_tiers")
	db.Exec("DELETE FROM training_satisfaction_ratings")
	db.Exec("DELETE FROM course_progress")
	db.Exec("DELETE FROM quiz_attempts")
	db.Exec("DELETE FROM quiz_questions")
	db.Exec("DELETE FROM quizzes")
	db.Exec("DELETE FROM training_presentations")
	return func() {
		db.Exec("DELETE FROM academy_sessions")
		db.Exec("DELETE FROM academy_user_progress")
		db.Exec("DELETE FROM academy_tiers")
		db.Exec("DELETE FROM training_satisfaction_ratings")
		db.Exec("DELETE FROM course_progress")
		db.Exec("DELETE FROM quiz_attempts")
		db.Exec("DELETE FROM quiz_questions")
		db.Exec("DELETE FROM quizzes")
		db.Exec("DELETE FROM training_presentations")
	}
}

// ---------- Academy Tier CRUD ----------

func TestCreateAcademyTier(t *testing.T) {
	teardown := setupAcademyTest(t)
	defer teardown()

	tier := &AcademyTier{
		OrgId:       1,
		Slug:        "bronze",
		Name:        "Bronze — Fundamentals",
		Description: "Build your foundation.",
		SortOrder:   1,
		IsActive:    true,
	}
	if err := CreateAcademyTier(tier); err != nil {
		t.Fatalf("failed to create tier: %v", err)
	}
	if tier.Id == 0 {
		t.Fatal("expected non-zero tier ID")
	}
	if tier.CreatedDate.IsZero() {
		t.Fatal("expected CreatedDate to be set")
	}
}

func TestGetAcademyTierBySlug(t *testing.T) {
	teardown := setupAcademyTest(t)
	defer teardown()

	tier := &AcademyTier{OrgId: 1, Slug: "silver", Name: "Silver", SortOrder: 2, IsActive: true}
	CreateAcademyTier(tier)

	found, err := GetAcademyTierBySlug(1, "silver")
	if err != nil {
		t.Fatalf("failed to find tier: %v", err)
	}
	if found.Name != "Silver" {
		t.Fatalf("expected name 'Silver', got %q", found.Name)
	}
}

func TestGetAcademyTierBySlugNotFound(t *testing.T) {
	teardown := setupAcademyTest(t)
	defer teardown()

	_, err := GetAcademyTierBySlug(1, "nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent tier slug")
	}
}

func TestGetAcademyTiers(t *testing.T) {
	teardown := setupAcademyTest(t)
	defer teardown()

	for i, slug := range []string{"bronze", "silver", "gold"} {
		CreateAcademyTier(&AcademyTier{OrgId: 1, Slug: slug, Name: slug, SortOrder: i + 1, IsActive: true})
	}

	tiers, err := GetAcademyTiers(1)
	if err != nil {
		t.Fatalf("failed to get tiers: %v", err)
	}
	if len(tiers) != 3 {
		t.Fatalf("expected 3 tiers, got %d", len(tiers))
	}
	// Should be sorted by sort_order
	if tiers[0].Slug != "bronze" || tiers[1].Slug != "silver" || tiers[2].Slug != "gold" {
		t.Fatal("tiers not in expected sort order")
	}
}

// ---------- Academy Sessions ----------

func TestCreateAcademySession(t *testing.T) {
	teardown := setupAcademyTest(t)
	defer teardown()

	tier := &AcademyTier{OrgId: 1, Slug: "bronze", Name: "Bronze", SortOrder: 1, IsActive: true}
	CreateAcademyTier(tier)

	tp := &TrainingPresentation{OrgId: 1, Name: "Course", FileName: "c.pdf", FilePath: "/f"}
	PostTrainingPresentation(tp)

	session := &AcademySession{
		TierId:           tier.Id,
		PresentationId:   tp.Id,
		SortOrder:        1,
		EstimatedMinutes: 15,
		IsRequired:       true,
	}
	if err := CreateAcademySession(session); err != nil {
		t.Fatalf("failed to create session: %v", err)
	}
	if session.Id == 0 {
		t.Fatal("expected non-zero session ID")
	}
}

func TestGetAcademySessions(t *testing.T) {
	teardown := setupAcademyTest(t)
	defer teardown()

	tier := &AcademyTier{OrgId: 1, Slug: "bronze", Name: "Bronze", SortOrder: 1, IsActive: true}
	CreateAcademyTier(tier)

	for i, name := range []string{"Course A", "Course B"} {
		tp := &TrainingPresentation{OrgId: 1, Name: name, FileName: name + ".pdf", FilePath: "/f"}
		PostTrainingPresentation(tp)
		CreateAcademySession(&AcademySession{
			TierId:         tier.Id,
			PresentationId: tp.Id,
			SortOrder:      i + 1,
			IsRequired:     true,
		})
	}

	sessions, err := GetAcademySessions(tier.Id)
	if err != nil {
		t.Fatalf("failed to get sessions: %v", err)
	}
	if len(sessions) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(sessions))
	}
	// Should have presentation names populated
	if sessions[0].PresentationName != "Course A" {
		t.Fatalf("expected presentation name 'Course A', got %q", sessions[0].PresentationName)
	}
}

func TestDeleteAcademySession(t *testing.T) {
	teardown := setupAcademyTest(t)
	defer teardown()

	tier := &AcademyTier{OrgId: 1, Slug: "bronze", Name: "Bronze", SortOrder: 1, IsActive: true}
	CreateAcademyTier(tier)

	tp := &TrainingPresentation{OrgId: 1, Name: "Del", FileName: "d.pdf", FilePath: "/f"}
	PostTrainingPresentation(tp)

	session := &AcademySession{TierId: tier.Id, PresentationId: tp.Id, SortOrder: 1, IsRequired: true}
	CreateAcademySession(session)

	if err := DeleteAcademySession(session.Id); err != nil {
		t.Fatalf("failed to delete session: %v", err)
	}

	sessions, _ := GetAcademySessions(tier.Id)
	if len(sessions) != 0 {
		t.Fatalf("expected 0 sessions after deletion, got %d", len(sessions))
	}
}

// ---------- Academy Tier Progression ----------

func TestTierProgressionSingleTier(t *testing.T) {
	teardown := setupAcademyTest(t)
	defer teardown()

	tier := &AcademyTier{OrgId: 1, Slug: "bronze", Name: "Bronze", SortOrder: 1, IsActive: true}
	CreateAcademyTier(tier)

	tp := &TrainingPresentation{OrgId: 1, Name: "Course", FileName: "c.pdf", FilePath: "/f"}
	PostTrainingPresentation(tp)
	CreateAcademySession(&AcademySession{TierId: tier.Id, PresentationId: tp.Id, SortOrder: 1, IsRequired: true})

	// Complete the course
	SaveCourseProgress(&CourseProgress{UserId: 1, PresentationId: tp.Id, Status: "complete", TotalPages: 3})

	// Update academy progress
	if err := UpdateAcademyProgress(1, tier.Id); err != nil {
		t.Fatalf("failed to update progress: %v", err)
	}

	progress, err := GetAcademyUserProgress(1, tier.Id)
	if err != nil {
		t.Fatalf("failed to get progress: %v", err)
	}
	if progress.SessionsCompleted != 1 {
		t.Fatalf("expected 1 session completed, got %d", progress.SessionsCompleted)
	}
	if !progress.TierCompleted {
		t.Fatal("expected tier to be completed")
	}
}

func TestTierProgressionUnlocksNextTier(t *testing.T) {
	teardown := setupAcademyTest(t)
	defer teardown()

	bronze := &AcademyTier{OrgId: 1, Slug: "bronze", Name: "Bronze", SortOrder: 1, IsActive: true}
	CreateAcademyTier(bronze)
	silver := &AcademyTier{OrgId: 1, Slug: "silver", Name: "Silver", SortOrder: 2, IsActive: true}
	CreateAcademyTier(silver)

	tp := &TrainingPresentation{OrgId: 1, Name: "Bronze Course", FileName: "b.pdf", FilePath: "/f"}
	PostTrainingPresentation(tp)
	CreateAcademySession(&AcademySession{TierId: bronze.Id, PresentationId: tp.Id, SortOrder: 1, IsRequired: true})

	// Complete bronze
	SaveCourseProgress(&CourseProgress{UserId: 1, PresentationId: tp.Id, Status: "complete", TotalPages: 3})
	UpdateAcademyProgress(1, bronze.Id)

	// Silver should now be unlocked
	silverProgress, err := GetAcademyUserProgress(1, silver.Id)
	if err != nil {
		t.Fatalf("failed to get silver progress: %v", err)
	}
	if !silverProgress.TierUnlocked {
		t.Fatal("expected silver tier to be unlocked after completing bronze")
	}
}

func TestTiersWithUserProgress(t *testing.T) {
	teardown := setupAcademyTest(t)
	defer teardown()

	bronze := &AcademyTier{OrgId: 1, Slug: "bronze", Name: "Bronze", SortOrder: 1, IsActive: true}
	CreateAcademyTier(bronze)
	silver := &AcademyTier{OrgId: 1, Slug: "silver", Name: "Silver", SortOrder: 2, IsActive: true}
	CreateAcademyTier(silver)

	tiers, err := GetAcademyTiersWithProgress(1, 1)
	if err != nil {
		t.Fatalf("failed to get tiers with progress: %v", err)
	}
	if len(tiers) != 2 {
		t.Fatalf("expected 2 tiers, got %d", len(tiers))
	}
	// First tier should auto-unlock
	if tiers[0].UserProgress == nil || !tiers[0].UserProgress.TierUnlocked {
		t.Fatal("expected first tier to be auto-unlocked")
	}
}

// ---------- Content Library Seeding ----------

func TestSeedBuiltInContent(t *testing.T) {
	teardown := setupAcademyTest(t)
	defer teardown()

	result, err := SeedBuiltInContent(1, 1)
	if err != nil {
		t.Fatalf("failed to seed content: %v", err)
	}
	if result.TiersCreated != 4 {
		t.Fatalf("expected 4 tiers created, got %d", result.TiersCreated)
	}
	if result.CoursesCreated == 0 {
		t.Fatal("expected some courses to be created")
	}
	if result.SessionsCreated == 0 {
		t.Fatal("expected some sessions to be created")
	}
	if result.QuizzesCreated == 0 {
		t.Fatal("expected some quizzes to be created")
	}

	// Verify tiers were created
	tiers, _ := GetAcademyTiers(1)
	if len(tiers) != 4 {
		t.Fatalf("expected 4 tiers in DB, got %d", len(tiers))
	}

	// Verify presentations match library
	scope := OrgScope{OrgId: 1, UserId: 1, IsSuperAdmin: true}
	tps, _ := GetTrainingPresentations(scope)
	libSize := len(GetBuiltInContentLibrary())
	if len(tps) != libSize {
		t.Fatalf("expected %d presentations, got %d", libSize, len(tps))
	}
}

func TestSeedBuiltInContentIdempotent(t *testing.T) {
	teardown := setupAcademyTest(t)
	defer teardown()

	// Seed twice
	result1, _ := SeedBuiltInContent(1, 1)
	result2, err := SeedBuiltInContent(1, 1)
	if err != nil {
		t.Fatalf("failed second seed: %v", err)
	}

	// Second run should skip everything
	if result2.CoursesCreated != 0 {
		t.Fatalf("expected 0 courses on second seed, got %d", result2.CoursesCreated)
	}
	if result2.Skipped != result1.CoursesCreated {
		t.Fatalf("expected %d skipped on second seed, got %d", result1.CoursesCreated, result2.Skipped)
	}
}

func TestSeedBuiltInContentOrgIsolation(t *testing.T) {
	teardown := setupAcademyTest(t)
	defer teardown()

	// Seed org 1 and org 2
	SeedBuiltInContent(1, 1)
	SeedBuiltInContent(2, 2)

	scope1 := OrgScope{OrgId: 1, UserId: 1, IsSuperAdmin: true}
	scope2 := OrgScope{OrgId: 2, UserId: 2, IsSuperAdmin: true}

	tps1, _ := GetTrainingPresentations(scope1)
	tps2, _ := GetTrainingPresentations(scope2)

	if len(tps1) != len(tps2) {
		t.Fatalf("expected same count for both orgs: %d vs %d", len(tps1), len(tps2))
	}
	if len(tps1) == 0 {
		t.Fatal("expected presentations to be created")
	}
}

// ---------- End-to-end Academy flow ----------

func TestAcademyEndToEndFlow(t *testing.T) {
	teardown := setupAcademyTest(t)
	defer teardown()

	// 1. Seed content
	SeedBuiltInContent(1, 1)

	// 2. Get tiers with progress for user 1
	tiers, err := GetAcademyTiersWithProgress(1, 1)
	if err != nil {
		t.Fatalf("failed to get tiers: %v", err)
	}
	if len(tiers) != 4 {
		t.Fatalf("expected 4 tiers, got %d", len(tiers))
	}

	// 3. First tier should be unlocked
	if tiers[0].UserProgress == nil || !tiers[0].UserProgress.TierUnlocked {
		t.Fatal("expected bronze tier to be auto-unlocked")
	}

	// 4. Get sessions for bronze tier
	sessions, err := GetAcademySessions(tiers[0].Id)
	if err != nil {
		t.Fatalf("failed to get bronze sessions: %v", err)
	}
	if len(sessions) == 0 {
		t.Fatal("expected bronze tier to have sessions")
	}

	// 5. Complete all required sessions
	for _, s := range sessions {
		if s.IsRequired {
			SaveCourseProgress(&CourseProgress{
				UserId:         1,
				PresentationId: s.PresentationId,
				Status:         "complete",
				TotalPages:     5,
			})
		}
	}

	// 6. Update progress and verify completion
	UpdateAcademyProgress(1, tiers[0].Id)
	progress, _ := GetAcademyUserProgress(1, tiers[0].Id)
	if !progress.TierCompleted {
		t.Fatal("expected bronze tier to be completed after finishing all required sessions")
	}

	// 7. Verify silver is now unlocked
	silverProgress, err := GetAcademyUserProgress(1, tiers[1].Id)
	if err != nil {
		t.Fatalf("failed to get silver progress: %v", err)
	}
	if !silverProgress.TierUnlocked {
		t.Fatal("expected silver tier to be unlocked")
	}
}
