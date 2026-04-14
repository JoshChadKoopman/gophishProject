package ai

import (
	"strings"
	"testing"
)

func TestBuildUserPromptWithUserContext(t *testing.T) {
	req := GenerateRequest{
		Prompt:          "Password reset",
		DifficultyLevel: DifficultyHard,
		UserContext: &UserContext{
			WeakCategories:  []string{"HR / Payroll", "Credential Harvesting"},
			ClickRate:       0.45,
			BRSScore:        35,
			TrendDirection:  "declining",
			AvoidCategories: []string{"IT Helpdesk"},
		},
	}
	prompt := BuildUserPrompt(req)

	checks := []string{
		"Adaptive Targeting Context",
		"HR / Payroll",
		"Credential Harvesting",
		"45%",
		"clicks frequently",
		"declining",
		"IT Helpdesk",
		"Avoid these recently-used categories",
	}
	for _, expected := range checks {
		if !strings.Contains(prompt, expected) {
			t.Fatalf("prompt missing %q\nGot:\n%s", expected, prompt)
		}
	}
}

func TestBuildUserPromptLowClickRate(t *testing.T) {
	req := GenerateRequest{
		Prompt: "Test",
		UserContext: &UserContext{
			ClickRate: 0.05,
		},
	}
	prompt := BuildUserPrompt(req)
	if !strings.Contains(prompt, "rarely clicks") {
		t.Fatalf("expected 'rarely clicks' guidance for 5%% click rate")
	}
}

func TestBuildUserPromptImprovingTrend(t *testing.T) {
	req := GenerateRequest{
		Prompt: "Test",
		UserContext: &UserContext{
			TrendDirection: "improving",
		},
	}
	prompt := BuildUserPrompt(req)
	if !strings.Contains(prompt, "improving") {
		t.Fatal("expected improving trend guidance")
	}
	if !strings.Contains(prompt, "sophistication") {
		t.Fatal("expected sophistication increase guidance for improving users")
	}
}

func TestBuildUserPromptNilContext(t *testing.T) {
	req := GenerateRequest{
		Prompt:      "Test",
		UserContext: nil,
	}
	prompt := BuildUserPrompt(req)
	if strings.Contains(prompt, "Adaptive Targeting") {
		t.Fatal("nil UserContext should not produce targeting block")
	}
}

func TestBuildUserPromptStableTrend(t *testing.T) {
	req := GenerateRequest{
		Prompt: "Test",
		UserContext: &UserContext{
			TrendDirection: "stable",
		},
	}
	prompt := BuildUserPrompt(req)
	if strings.Contains(prompt, "improving") || strings.Contains(prompt, "declining") {
		t.Fatal("stable trend should not produce trend guidance")
	}
}

func TestBuildUserPromptModerateClickRate(t *testing.T) {
	req := GenerateRequest{
		Prompt: "Test",
		UserContext: &UserContext{
			ClickRate: 0.25,
		},
	}
	prompt := BuildUserPrompt(req)
	if strings.Contains(prompt, "rarely clicks") || strings.Contains(prompt, "clicks frequently") {
		t.Fatal("moderate click rate should not produce specific guidance")
	}
	if !strings.Contains(prompt, "25%") {
		t.Fatal("expected click rate percentage in prompt")
	}
}
