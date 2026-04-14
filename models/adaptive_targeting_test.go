package models

import "testing"

// ---------- recommendDifficulty tests ----------

func TestRecommendDifficultyNewUser(t *testing.T) {
	p := &UserTargetingProfile{TotalSimulations: 0}
	if d := recommendDifficulty(p); d != 1 {
		t.Fatalf("new user (0 sims) expected difficulty 1, got %d", d)
	}
	p.TotalSimulations = 2
	if d := recommendDifficulty(p); d != 1 {
		t.Fatalf("new user (2 sims) expected difficulty 1, got %d", d)
	}
}

func TestRecommendDifficultyByBRS(t *testing.T) {
	tests := []struct {
		name     string
		brs      float64
		expected int
	}{
		{"very low BRS", 15, 1},
		{"low BRS", 40, 2},
		{"medium BRS", 65, 3},
		{"high BRS", 90, 4},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &UserTargetingProfile{
				TotalSimulations: 10,
				BRSComposite:     tt.brs,
				OverallClickRate: 0.15, // moderate — no click-rate override
				TrendDirection:   "stable",
			}
			if d := recommendDifficulty(p); d != tt.expected {
				t.Fatalf("BRS %.0f: expected difficulty %d, got %d", tt.brs, tt.expected, d)
			}
		})
	}
}

func TestRecommendDifficultyHighClickRateOverrides(t *testing.T) {
	p := &UserTargetingProfile{
		TotalSimulations: 20,
		BRSComposite:     70,          // would normally be difficulty 3
		OverallClickRate: 0.55,        // but >50% click rate forces level 1
		TrendDirection:   "stable",
	}
	if d := recommendDifficulty(p); d != 1 {
		t.Fatalf("high click rate should force difficulty 1, got %d", d)
	}
}

func TestRecommendDifficultyModerateClickRateCaps(t *testing.T) {
	p := &UserTargetingProfile{
		TotalSimulations: 20,
		BRSComposite:     85,          // would normally be difficulty 4
		OverallClickRate: 0.35,        // but 30-50% caps at level 2
		TrendDirection:   "stable",
	}
	if d := recommendDifficulty(p); d != 2 {
		t.Fatalf("moderate click rate should cap at difficulty 2, got %d", d)
	}
}

func TestRecommendDifficultyVeryLowClickRateBumps(t *testing.T) {
	p := &UserTargetingProfile{
		TotalSimulations: 15,
		BRSComposite:     65,          // difficulty 3 from BRS
		OverallClickRate: 0.03,        // very low click rate bumps +1
		TrendDirection:   "stable",
	}
	if d := recommendDifficulty(p); d != 4 {
		t.Fatalf("very low click rate should bump to difficulty 4, got %d", d)
	}
}

func TestRecommendDifficultyTrendImproving(t *testing.T) {
	p := &UserTargetingProfile{
		TotalSimulations: 10,
		BRSComposite:     50,          // difficulty 2 from BRS
		OverallClickRate: 0.15,        // moderate — no click-rate override
		TrendDirection:   "improving",
	}
	if d := recommendDifficulty(p); d != 3 {
		t.Fatalf("improving trend should bump 2→3, got %d", d)
	}
}

func TestRecommendDifficultyTrendDeclining(t *testing.T) {
	p := &UserTargetingProfile{
		TotalSimulations: 10,
		BRSComposite:     65,          // difficulty 3 from BRS
		OverallClickRate: 0.15,        // moderate — no click-rate override
		TrendDirection:   "declining",
	}
	if d := recommendDifficulty(p); d != 2 {
		t.Fatalf("declining trend should drop 3→2, got %d", d)
	}
}

func TestRecommendDifficultyFloorAndCeiling(t *testing.T) {
	// Declining can't go below 1
	p := &UserTargetingProfile{
		TotalSimulations: 10,
		BRSComposite:     20,
		OverallClickRate: 0.15,
		TrendDirection:   "declining",
	}
	if d := recommendDifficulty(p); d != 1 {
		t.Fatalf("difficulty should not go below 1, got %d", d)
	}

	// Improving can't go above 4
	p = &UserTargetingProfile{
		TotalSimulations: 10,
		BRSComposite:     95,
		OverallClickRate: 0.15,
		TrendDirection:   "improving",
	}
	if d := recommendDifficulty(p); d != 4 {
		t.Fatalf("difficulty should not exceed 4, got %d", d)
	}
}

// ---------- SelectTemplate tests ----------

func TestSelectTemplateNilProfile(t *testing.T) {
	templates := []Template{
		{Id: 1, Name: "T1", DifficultyLevel: 1},
		{Id: 2, Name: "T2", DifficultyLevel: 2},
	}
	result := SelectTemplate(nil, templates)
	if result.Id == 0 {
		t.Fatal("expected a template to be selected with nil profile")
	}
}

func TestSelectTemplateEmptyList(t *testing.T) {
	profile := &UserTargetingProfile{TotalSimulations: 10}
	result := SelectTemplate(profile, nil)
	if result.Id != 0 {
		t.Fatalf("expected zero template for empty list, got ID %d", result.Id)
	}
}

func TestSelectTemplatePrefersMatchingDifficulty(t *testing.T) {
	profile := &UserTargetingProfile{
		TotalSimulations:      20,
		BRSComposite:          70,
		RecommendedDifficulty: 3,
		TrendDirection:        "stable",
	}
	templates := []Template{
		{Id: 1, Name: "Easy", DifficultyLevel: 1, Category: "IT Helpdesk"},
		{Id: 2, Name: "Hard", DifficultyLevel: 3, Category: "IT Helpdesk"},
	}

	// Run multiple times — the hard template should be selected most of the time
	hardCount := 0
	for i := 0; i < 50; i++ {
		result := SelectTemplate(profile, templates)
		if result.DifficultyLevel == 3 {
			hardCount++
		}
	}
	if hardCount < 35 {
		t.Fatalf("expected hard template to be selected majority of time, got %d/50", hardCount)
	}
}

func TestSelectTemplatePrefersWeakCategory(t *testing.T) {
	profile := &UserTargetingProfile{
		TotalSimulations:      20,
		RecommendedDifficulty: 2,
		WeakCategories: []CategoryScore{
			{Category: "HR / Payroll", ClickRate: 0.6},
		},
		TrendDirection: "stable",
	}
	templates := []Template{
		{Id: 1, Name: "IT", DifficultyLevel: 2, Category: "IT Helpdesk"},
		{Id: 2, Name: "HR", DifficultyLevel: 2, Category: "HR / Payroll"},
	}

	hrCount := 0
	for i := 0; i < 50; i++ {
		result := SelectTemplate(profile, templates)
		if result.Category == "HR / Payroll" {
			hrCount++
		}
	}
	if hrCount < 35 {
		t.Fatalf("expected weak category (HR) to be preferred, got %d/50", hrCount)
	}
}

func TestSelectTemplateAvoidsRecentCategories(t *testing.T) {
	profile := &UserTargetingProfile{
		TotalSimulations:      20,
		RecommendedDifficulty: 2,
		RecentCategories:      []string{"IT Helpdesk"},
		TrendDirection:        "stable",
	}
	templates := []Template{
		{Id: 1, Name: "IT", DifficultyLevel: 2, Category: "IT Helpdesk"},
		{Id: 2, Name: "HR", DifficultyLevel: 2, Category: "HR / Payroll"},
	}

	hrCount := 0
	for i := 0; i < 50; i++ {
		result := SelectTemplate(profile, templates)
		if result.Category == "HR / Payroll" {
			hrCount++
		}
	}
	if hrCount < 35 {
		t.Fatalf("expected recently-used category to be deprioritized, HR selected %d/50", hrCount)
	}
}

// ---------- abs helper ----------

func TestAbsHelper(t *testing.T) {
	if abs(-5) != 5 {
		t.Fatal("abs(-5) should be 5")
	}
	if abs(3) != 3 {
		t.Fatal("abs(3) should be 3")
	}
	if abs(0) != 0 {
		t.Fatal("abs(0) should be 0")
	}
}

// ---------- integration: adaptive targeting + real template library ----------

const (
	testTemplateLibraryDir = "../static/db/templates"
	testTemplateLibrarySkipMsg = "template library dir not available: %v"
)

// TestIntegrationAdaptiveLibrarySelection wires the JSON template library
// together with the adaptive targeting engine and verifies a realistic
// end-to-end flow: load templates -> build profile -> pick template.
func TestIntegrationAdaptiveLibrarySelection(t *testing.T) {
	// Load the real JSON library.
	loaded, err := loadTemplatesFromDir(testTemplateLibraryDir)
	if err != nil {
		t.Skipf(testTemplateLibrarySkipMsg, err)
	}
	if len(loaded) < 50 {
		t.Fatalf("expected at least 50 templates in JSON library, got %d", len(loaded))
	}

	// Swap in loaded templates as the active library for this test.
	original := TemplateLibrary
	TemplateLibrary = loaded
	defer func() { TemplateLibrary = original }()

	cases := []struct {
		name          string
		profile       *UserTargetingProfile
		wantDifficMin int
		wantDifficMax int
	}{
		{
			name: "experienced user with high BRS",
			profile: &UserTargetingProfile{
				UserId:                42,
				TotalSimulations:      25,
				OverallClickRate:      0.05,
				BRSComposite:          88,
				RecommendedDifficulty: 4,
				WeakCategories:        []CategoryScore{{Category: CategoryBEC, ClickRate: 0.2}},
				TrendDirection:        "improving",
			},
			wantDifficMin: 3,
			wantDifficMax: 4,
		},
		{
			name: "vulnerable user with low BRS",
			profile: &UserTargetingProfile{
				UserId:                43,
				TotalSimulations:      15,
				OverallClickRate:      0.45,
				BRSComposite:          20,
				RecommendedDifficulty: 1,
				WeakCategories:        []CategoryScore{{Category: CategoryCredentialHarvesting, ClickRate: 0.6}},
				TrendDirection:        "stable",
			},
			wantDifficMin: 1,
			wantDifficMax: 2,
		},
		{
			name: "mid-tier user with recent IT helpdesk campaign",
			profile: &UserTargetingProfile{
				UserId:                44,
				TotalSimulations:      10,
				OverallClickRate:      0.20,
				BRSComposite:          55,
				RecommendedDifficulty: 2,
				RecentCategories:      []string{CategoryITHelpdesk},
				WeakCategories:        []CategoryScore{{Category: CategoryHRPayroll, ClickRate: 0.3}},
				TrendDirection:        "stable",
			},
			wantDifficMin: 1,
			wantDifficMax: 3,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tmpl := SelectLibraryTemplate(tc.profile)
			if tmpl == nil {
				t.Fatal("SelectLibraryTemplate returned nil")
			}
			if tmpl.Slug == "" || tmpl.Subject == "" {
				t.Fatalf("selected template missing data: %+v", tmpl)
			}
			if tmpl.DifficultyLevel < tc.wantDifficMin || tmpl.DifficultyLevel > tc.wantDifficMax {
				t.Errorf("selected template difficulty %d outside expected range [%d, %d] for %s",
					tmpl.DifficultyLevel, tc.wantDifficMin, tc.wantDifficMax, tc.name)
			}
		})
	}
}

// TestIntegrationAdaptiveWeakCategoryBias runs many selections and verifies
// the adaptive engine biases toward the user's weak category when templates
// are available in that category.
func TestIntegrationAdaptiveWeakCategoryBias(t *testing.T) {
	loaded, err := loadTemplatesFromDir(testTemplateLibraryDir)
	if err != nil {
		t.Skipf(testTemplateLibrarySkipMsg, err)
	}

	original := TemplateLibrary
	TemplateLibrary = loaded
	defer func() { TemplateLibrary = original }()

	profile := &UserTargetingProfile{
		UserId:                99,
		TotalSimulations:      20,
		OverallClickRate:      0.25,
		BRSComposite:          50,
		RecommendedDifficulty: 2,
		WeakCategories: []CategoryScore{
			{Category: CategoryBEC, ClickRate: 0.7},
		},
		TrendDirection: "stable",
	}

	const runs = 100
	becCount := 0
	for i := 0; i < runs; i++ {
		tmpl := SelectLibraryTemplate(profile)
		if tmpl == nil {
			t.Fatal("SelectLibraryTemplate returned nil mid-run")
		}
		if tmpl.Category == CategoryBEC {
			becCount++
		}
	}
	// With weak-category scoring, BEC should dominate selections even though
	// it is only one of ~14 categories (baseline ~7%).
	if becCount < 40 {
		t.Errorf("expected weak category (BEC) to be selected at least 40/100, got %d", becCount)
	}
}

// TestIntegrationAutopilotReadinessCheck validates the library contains
// enough variety across difficulties to support the autopilot adaptive flow
// (which needs templates at each difficulty level).
func TestIntegrationAutopilotReadinessCheck(t *testing.T) {
	loaded, err := loadTemplatesFromDir(testTemplateLibraryDir)
	if err != nil {
		t.Skipf(testTemplateLibrarySkipMsg, err)
	}

	byDifficulty := make(map[int]int)
	byCategory := make(map[string]int)
	for _, tmpl := range loaded {
		byDifficulty[tmpl.DifficultyLevel]++
		byCategory[tmpl.Category]++
	}

	// All four difficulty levels should have at least 10 templates.
	for level := 1; level <= 4; level++ {
		if byDifficulty[level] < 10 {
			t.Errorf("difficulty level %d has only %d templates (want >= 10)", level, byDifficulty[level])
		}
	}

	// At least 10 distinct categories to support adaptive weak-category selection.
	if len(byCategory) < 10 {
		t.Errorf("expected at least 10 categories for adaptive targeting, got %d", len(byCategory))
	}
}
