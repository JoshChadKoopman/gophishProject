package models

import (
	"testing"
)

func TestValidateDifficultyLevel(t *testing.T) {
	tests := []struct {
		name    string
		level   int
		wantErr bool
	}{
		{"level 0 invalid", 0, true},
		{"level 1 valid", 1, false},
		{"level 2 valid", 2, false},
		{"level 3 valid", 3, false},
		{"level 4 valid", 4, false},
		{"level 5 invalid", 5, true},
		{"negative invalid", -1, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDifficultyLevel(tt.level)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDifficultyLevel(%d) error = %v, wantErr %v", tt.level, err, tt.wantErr)
			}
		})
	}
}

func TestValidateDifficultyMode(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		wantErr bool
	}{
		{"adaptive valid", DifficultyModeAdaptive, false},
		{"manual valid", DifficultyModeManual, false},
		{"empty invalid", "", true},
		{"unknown invalid", "auto", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDifficultyMode(tt.mode)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDifficultyMode(%q) error = %v, wantErr %v", tt.mode, err, tt.wantErr)
			}
		})
	}
}

func TestDifficultyLevelLabels(t *testing.T) {
	if len(DifficultyLevelLabels) != 4 {
		t.Errorf("Expected 4 difficulty labels, got %d", len(DifficultyLevelLabels))
	}
	for level := DifficultyEasy; level <= DifficultySophisticated; level++ {
		if DifficultyLevelLabels[level] == "" {
			t.Errorf("Missing label for difficulty level %d", level)
		}
	}
}

func TestDifficultyConstants(t *testing.T) {
	if DifficultyModeAdaptive != "adaptive" {
		t.Errorf("Expected DifficultyModeAdaptive to be 'adaptive', got %q", DifficultyModeAdaptive)
	}
	if DifficultyModeManual != "manual" {
		t.Errorf("Expected DifficultyModeManual to be 'manual', got %q", DifficultyModeManual)
	}
	if DifficultyEasy != 1 {
		t.Errorf("Expected DifficultyEasy to be 1, got %d", DifficultyEasy)
	}
	if DifficultySophisticated != 4 {
		t.Errorf("Expected DifficultySophisticated to be 4, got %d", DifficultySophisticated)
	}
}

func TestBuildAdaptiveReason(t *testing.T) {
	profile := &UserTargetingProfile{
		BRSComposite:     65.0,
		OverallClickRate: 0.15,
		TrendDirection:   "improving",
		TotalSimulations: 20,
	}

	reason := buildAdaptiveReason(profile, 2, 3)
	if reason == "" {
		t.Error("Expected non-empty reason string")
	}
	// Should mention "increased"
	if !containsSubstring(reason, "increased") {
		t.Errorf("Expected reason to mention 'increased', got: %s", reason)
	}

	reason2 := buildAdaptiveReason(profile, 3, 2)
	if !containsSubstring(reason2, "decreased") {
		t.Errorf("Expected reason to mention 'decreased', got: %s", reason2)
	}
}

func TestFilterByDifficultyRange(t *testing.T) {
	pool := []BuiltInTrainingContent{
		{Slug: "a", DifficultyLevel: 1},
		{Slug: "b", DifficultyLevel: 2},
		{Slug: "c", DifficultyLevel: 3},
		{Slug: "d", DifficultyLevel: 4},
	}

	// Exact match for level 2
	exact := filterByDifficultyRange(pool, 2, 0)
	if len(exact) != 1 || exact[0].Slug != "b" {
		t.Errorf("Expected 1 result for exact match level 2, got %d", len(exact))
	}

	// ±1 from level 2 should include 1,2,3
	fuzzy := filterByDifficultyRange(pool, 2, 1)
	if len(fuzzy) != 3 {
		t.Errorf("Expected 3 results for ±1 from level 2, got %d", len(fuzzy))
	}
}

func TestFilterByDifficultyWithFallback(t *testing.T) {
	pool := []BuiltInTrainingContent{
		{Slug: "a", DifficultyLevel: 1},
		{Slug: "b", DifficultyLevel: 3},
	}

	// Level 2 has no exact match, should fall back to ±1 (includes 1 and 3)
	result := filterByDifficultyWithFallback(pool, 2)
	if len(result) != 2 {
		t.Errorf("Expected 2 results from fallback, got %d", len(result))
	}

	// Level 1 has exact match
	result2 := filterByDifficultyWithFallback(pool, 1)
	if len(result2) != 1 || result2[0].Slug != "a" {
		t.Errorf("Expected 1 exact result for level 1, got %d", len(result2))
	}
}

func containsSubstring(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
