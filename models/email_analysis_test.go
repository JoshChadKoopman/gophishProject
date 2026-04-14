package models

import (
	"strings"
	"testing"

	"github.com/gophish/gophish/ai"
)

// ---- Struct and constant sanity checks ----

func TestEmailAnalysisTableName(t *testing.T) {
	a := EmailAnalysis{}
	if a.TableName() != "email_analyses" {
		t.Errorf("expected table name 'email_analyses', got '%s'", a.TableName())
	}
}

func TestEmailIndicatorTableName(t *testing.T) {
	i := EmailIndicator{}
	if i.TableName() != "email_indicators" {
		t.Errorf("expected table name 'email_indicators', got '%s'", i.TableName())
	}
}

func TestEmailAnalysisConstants(t *testing.T) {
	// Status values
	if AnalysisStatusPending != "pending" {
		t.Error("unexpected AnalysisStatusPending")
	}
	if AnalysisStatusAnalyzing != "analyzing" {
		t.Error("unexpected AnalysisStatusAnalyzing")
	}
	if AnalysisStatusCompleted != "completed" {
		t.Error("unexpected AnalysisStatusCompleted")
	}
	if AnalysisStatusFailed != "failed" {
		t.Error("unexpected AnalysisStatusFailed")
	}

	// Threat levels
	if ThreatLevelSafe != "safe" {
		t.Error("unexpected ThreatLevelSafe")
	}
	if ThreatLevelLikelyPhishing != "likely_phishing" {
		t.Error("unexpected ThreatLevelLikelyPhishing")
	}

	// Classifications
	if ClassificationBEC != "bec" {
		t.Error("unexpected ClassificationBEC")
	}
	if ClassificationSpearPhishing != "spear_phishing" {
		t.Error("unexpected ClassificationSpearPhishing")
	}

	// Indicator types
	if IndicatorTypeURL != "url" {
		t.Error("unexpected IndicatorTypeURL")
	}
	if IndicatorTypeImpersonation != "impersonation" {
		t.Error("unexpected IndicatorTypeImpersonation")
	}
}

func TestEmailAnalysisSummaryDefaults(t *testing.T) {
	s := EmailAnalysisSummary{}
	if s.TotalAnalyzed != 0 || s.Pending != 0 || s.PhishingDetected != 0 {
		t.Error("default summary should have zero values")
	}
	if s.AvgConfidence != 0 || s.HighThreatCount != 0 {
		t.Error("default summary should have zero confidence and threat count")
	}
}

// ---- AI prompt and parsing tests ----

func TestBuildEmailAnalysisPrompt(t *testing.T) {
	prompt := ai.BuildEmailAnalysisPrompt(
		"From: attacker@evil.com\nTo: victim@company.com",
		"Click here to reset your password",
		"attacker@evil.com",
		"Urgent: Password Reset Required",
	)
	if prompt == "" {
		t.Fatal("expected non-empty prompt")
	}
	// Should contain all sections
	for _, section := range []string{"SENDER", "SUBJECT", "EMAIL HEADERS", "EMAIL BODY"} {
		if !strings.Contains(prompt, section) {
			t.Errorf("prompt missing section: %s", section)
		}
	}
}

func TestBuildEmailAnalysisPromptMissingHeaders(t *testing.T) {
	prompt := ai.BuildEmailAnalysisPrompt("", "body", "sender", "subject")
	if !strings.Contains(prompt, "[Not available]") {
		t.Error("expected '[Not available]' when headers are empty")
	}
}

func TestBuildEmailAnalysisPromptMissingBody(t *testing.T) {
	prompt := ai.BuildEmailAnalysisPrompt("headers", "", "sender", "subject")
	if !strings.Contains(prompt, "[Not available]") {
		t.Error("expected '[Not available]' when body is empty")
	}
}

func TestParseEmailAnalysisResponseValid(t *testing.T) {
	raw := `{
		"threat_level": "likely_phishing",
		"confidence": 0.87,
		"classification": "spear_phishing",
		"summary": "This email impersonates an IT administrator.",
		"indicators": [
			{
				"type": "url",
				"value": "https://evil-login.com/reset",
				"severity": "high",
				"description": "Suspicious URL that mimics a password reset page"
			},
			{
				"type": "urgency_cue",
				"value": "Your account will be suspended within 24 hours",
				"severity": "medium",
				"description": "Urgency language designed to pressure immediate action"
			}
		]
	}`
	result, err := ai.ParseEmailAnalysisResponse(raw)
	if err != nil {
		t.Fatalf("parse failed: %v", err)
	}
	if result.ThreatLevel != "likely_phishing" {
		t.Errorf("expected threat_level 'likely_phishing', got '%s'", result.ThreatLevel)
	}
	if result.Confidence != 0.87 {
		t.Errorf("expected confidence 0.87, got %f", result.Confidence)
	}
	if result.Classification != "spear_phishing" {
		t.Errorf("expected classification 'spear_phishing', got '%s'", result.Classification)
	}
	if len(result.Indicators) != 2 {
		t.Fatalf("expected 2 indicators, got %d", len(result.Indicators))
	}
	if result.Indicators[0].Type != "url" {
		t.Errorf("expected first indicator type 'url', got '%s'", result.Indicators[0].Type)
	}
}

func TestParseEmailAnalysisResponseWithCodeFences(t *testing.T) {
	raw := "```json\n" + `{
		"threat_level": "safe",
		"confidence": 0.95,
		"classification": "legitimate",
		"summary": "This is a legitimate email.",
		"indicators": [
			{"type": "domain", "value": "company.com", "severity": "info", "description": "Known legitimate domain"}
		]
	}` + "\n```"
	result, err := ai.ParseEmailAnalysisResponse(raw)
	if err != nil {
		t.Fatalf("parse with code fences failed: %v", err)
	}
	if result.ThreatLevel != "safe" {
		t.Errorf("expected 'safe', got '%s'", result.ThreatLevel)
	}
}

func TestParseEmailAnalysisResponseMissingThreatLevel(t *testing.T) {
	raw := `{"confidence": 0.5, "classification": "phishing", "summary": "test"}`
	_, err := ai.ParseEmailAnalysisResponse(raw)
	if err == nil {
		t.Fatal("expected error for missing threat_level")
	}
}

func TestParseEmailAnalysisResponseMissingClassification(t *testing.T) {
	raw := `{"threat_level": "safe", "confidence": 0.5, "summary": "test"}`
	_, err := ai.ParseEmailAnalysisResponse(raw)
	if err == nil {
		t.Fatal("expected error for missing classification")
	}
}

func TestParseEmailAnalysisResponseMissingSummary(t *testing.T) {
	raw := `{"threat_level": "safe", "confidence": 0.5, "classification": "legitimate"}`
	_, err := ai.ParseEmailAnalysisResponse(raw)
	if err == nil {
		t.Fatal("expected error for missing summary")
	}
}

func TestParseEmailAnalysisResponseConfidenceOutOfRange(t *testing.T) {
	raw := `{"threat_level": "safe", "confidence": 1.5, "classification": "legitimate", "summary": "test"}`
	_, err := ai.ParseEmailAnalysisResponse(raw)
	if err == nil {
		t.Fatal("expected error for confidence > 1.0")
	}
}

func TestParseEmailAnalysisResponseInvalidJSON(t *testing.T) {
	_, err := ai.ParseEmailAnalysisResponse("not json at all")
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestEmailAnalysisSystemPrompt(t *testing.T) {
	if ai.EmailAnalysisSystemPrompt == "" {
		t.Fatal("system prompt should not be empty")
	}
	// Should mention key analysis dimensions
	for _, keyword := range []string{"Header Analysis", "Social Engineering", "URL", "BEC", "Impersonation", "Language"} {
		if !strings.Contains(ai.EmailAnalysisSystemPrompt, keyword) {
			t.Errorf("system prompt missing keyword: %s", keyword)
		}
	}
}
