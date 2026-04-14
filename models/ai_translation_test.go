package models

import (
	"testing"
)

func TestTranslationStatusConstants(t *testing.T) {
	if TranslationStatusPending != "pending" {
		t.Errorf("expected 'pending', got %q", TranslationStatusPending)
	}
	if TranslationStatusCompleted != "completed" {
		t.Errorf("expected 'completed', got %q", TranslationStatusCompleted)
	}
	if TranslationStatusFailed != "failed" {
		t.Errorf("expected 'failed', got %q", TranslationStatusFailed)
	}
}

func TestTranslationContentTypeConstants(t *testing.T) {
	types := []string{
		TranslationContentTemplate,
		TranslationContentTraining,
		TranslationContentPage,
		TranslationContentEmail,
		TranslationContentQuiz,
	}
	expected := []string{"template", "training", "page", "email", "quiz"}
	for i, ct := range types {
		if ct != expected[i] {
			t.Errorf("content type %d: expected %q, got %q", i, expected[i], ct)
		}
	}
}

func TestIsValidTranslationLang(t *testing.T) {
	validLangs := []string{"en", "nl", "de", "fr", "es", "ja", "ko", "zh", "ar"}
	for _, code := range validLangs {
		if !IsValidTranslationLang(code) {
			t.Errorf("expected %q to be a valid language", code)
		}
	}
}

func TestIsValidTranslationLangInvalid(t *testing.T) {
	invalidLangs := []string{"xx", "zz", "klingon", "", "123"}
	for _, code := range invalidLangs {
		if IsValidTranslationLang(code) {
			t.Errorf("expected %q to be invalid", code)
		}
	}
}

func TestAITranslationLanguagesCount(t *testing.T) {
	if len(AITranslationLanguages) < 20 {
		t.Errorf("expected at least 20 supported languages, got %d", len(AITranslationLanguages))
	}
}

func TestAITranslationLanguagesContainsDutch(t *testing.T) {
	name, ok := AITranslationLanguages["nl"]
	if !ok {
		t.Fatal("expected Dutch (nl) to be in supported languages")
	}
	if name != "Dutch" {
		t.Errorf("expected language name 'Dutch', got %q", name)
	}
}

func TestTranslationRequestTableName(t *testing.T) {
	r := TranslationRequest{}
	if r.TableName() != "translation_requests" {
		t.Errorf("expected 'translation_requests', got %q", r.TableName())
	}
}

func TestTranslatedContentTableName(t *testing.T) {
	c := TranslatedContent{}
	if c.TableName() != "translated_contents" {
		t.Errorf("expected 'translated_contents', got %q", c.TableName())
	}
}

func TestTranslationConfigTableName(t *testing.T) {
	c := TranslationConfig{}
	if c.TableName() != "translation_configs" {
		t.Errorf("expected 'translation_configs', got %q", c.TableName())
	}
}

func TestTranslationRequestStruct(t *testing.T) {
	r := TranslationRequest{
		OrgId:       1,
		ContentType: TranslationContentTemplate,
		ContentId:   42,
		SourceLang:  "en",
		TargetLang:  "nl",
		Status:      TranslationStatusPending,
	}
	if r.OrgId != 1 {
		t.Errorf("expected OrgId 1, got %d", r.OrgId)
	}
	if r.ContentType != "template" {
		t.Errorf("expected content type 'template', got %q", r.ContentType)
	}
	if r.TargetLang != "nl" {
		t.Errorf("expected target lang 'nl', got %q", r.TargetLang)
	}
}

func TestTranslatedContentStruct(t *testing.T) {
	tc := TranslatedContent{
		ContentType:     TranslationContentTraining,
		TranslatedTitle: "Sicherheitsbewusstsein",
		TranslatedBody:  "Translated body content",
		TargetLang:      "de",
		Quality:         92.5,
		IsApproved:      true,
	}
	if tc.Quality != 92.5 {
		t.Errorf("expected quality 92.5, got %f", tc.Quality)
	}
	if !tc.IsApproved {
		t.Error("expected IsApproved to be true")
	}
}

func TestTranslationConfigDefaults(t *testing.T) {
	cfg := TranslationConfig{
		Enabled:          true,
		AutoTranslate:    false,
		ReviewRequired:   true,
		MaxMonthlyTokens: 500000,
	}
	if !cfg.Enabled {
		t.Error("expected Enabled to be true")
	}
	if cfg.AutoTranslate {
		t.Error("expected AutoTranslate to be false by default")
	}
	if !cfg.ReviewRequired {
		t.Error("expected ReviewRequired to be true by default")
	}
	if cfg.MaxMonthlyTokens != 500000 {
		t.Errorf("expected 500000 max tokens, got %d", cfg.MaxMonthlyTokens)
	}
}

func TestTranslationUsageSummaryStruct(t *testing.T) {
	s := TranslationUsageSummary{
		TotalRequests:     100,
		CompletedCount:    90,
		FailedCount:       10,
		TotalInputTokens:  50000,
		TotalOutputTokens: 60000,
		TotalTokens:       110000,
		LanguagesUsed:     5,
	}
	if s.TotalRequests != 100 {
		t.Errorf("expected 100 total requests, got %d", s.TotalRequests)
	}
	if s.TotalTokens != 110000 {
		t.Errorf("expected 110000 total tokens, got %d", s.TotalTokens)
	}
	if s.LanguagesUsed != 5 {
		t.Errorf("expected 5 languages, got %d", s.LanguagesUsed)
	}
}

func TestQueryConstantsTranslation(t *testing.T) {
	if queryWhereOrgIDTranslation != "org_id = ?" {
		t.Errorf("unexpected query constant: %q", queryWhereOrgIDTranslation)
	}
	if queryWhereIDTranslation != "id = ?" {
		t.Errorf("unexpected query constant: %q", queryWhereIDTranslation)
	}
}
