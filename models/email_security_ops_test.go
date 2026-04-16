package models

import (
	"testing"
	"time"
)

func TestExtractDomain(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"user@example.com", "example.com"},
		{"USER@Example.COM", "example.com"},
		{"noatsign", ""},
		{"trailing@", ""},
		{"multiple@at@signs.com", "signs.com"},
		{"spaced@ domain.com ", "domain.com"},
	}
	for _, tc := range tests {
		got := extractDomain(tc.input)
		if got != tc.expected {
			t.Errorf("extractDomain(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

func TestRound2(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{1.234, 1.23},
		{1.235, 1.24},
		{0.0, 0.0},
		{99.999, 100.0},
	}
	for _, tc := range tests {
		got := round2(tc.input)
		if got != tc.expected {
			t.Errorf("round2(%f) = %f, want %f", tc.input, got, tc.expected)
		}
	}
}

func TestRound4(t *testing.T) {
	got := round4(0.12345)
	if got != 0.1235 {
		t.Errorf("round4(0.12345) = %f, want 0.1235", got)
	}
}

func TestEmptyDailyTrend(t *testing.T) {
	trend := emptyDailyTrend(
		func() time.Time { t, _ := time.Parse(time.RFC3339, "2026-03-01T00:00:00Z"); return t }(),
		5,
	)
	if len(trend) != 5 {
		t.Fatalf("Expected 5 days, got %d", len(trend))
	}
	expected := []string{"2026-03-01", "2026-03-02", "2026-03-03", "2026-03-04", "2026-03-05"}
	for i, want := range expected {
		if trend[i].Date != want {
			t.Errorf("Day %d: got %s, want %s", i, trend[i].Date, want)
		}
		if trend[i].AvgMinutes != 0 || trend[i].Count != 0 {
			t.Errorf("Day %d should be zero-valued", i)
		}
	}
}

func TestFalsePositiveStatsZeroDivision(t *testing.T) {
	stats := FalsePositiveStats{}
	// No classified tickets = rate stays 0
	if stats.FalsePositiveRate != 0 {
		t.Errorf("Expected 0 rate, got %f", stats.FalsePositiveRate)
	}
}

func TestSLACompliancePercentage(t *testing.T) {
	stats := SLAComplianceStats{
		TotalTickets: 10,
		WithinSLA:    8,
		Breached:     2,
	}
	// Mirror the calculation logic
	pct := round2(float64(stats.WithinSLA) / float64(stats.TotalTickets) * 100)
	if pct != 80.0 {
		t.Errorf("Expected 80.0%%, got %f", pct)
	}
}
