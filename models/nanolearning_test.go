package models

import (
	"testing"
)

func TestNanolearningEventTableName(t *testing.T) {
	e := NanolearningEvent{}
	if e.TableName() != "nanolearning_events" {
		t.Fatalf("expected 'nanolearning_events', got %q", e.TableName())
	}
}

func TestGetNanolearningTipForCategory(t *testing.T) {
	// Should not panic even with empty library
	tip := GetNanolearningTipForCategory("phishing")
	// May be nil if the built-in library isn't loaded; that's OK
	_ = tip
}

func TestGetNanolearningTipForCategoryFallback(t *testing.T) {
	tip := GetNanolearningTipForCategory("nonexistent_category_xyz")
	// Should either return nil or a fallback tip — no panic
	_ = tip
}

func TestNanolearningTipStruct(t *testing.T) {
	tip := NanolearningTip{
		Slug:     "test-slug",
		Tip:      "Always verify sender addresses",
		Category: "phishing",
		Title:    "Phishing Tip",
	}
	if tip.Slug != "test-slug" {
		t.Fatalf("expected slug 'test-slug', got %q", tip.Slug)
	}
	if tip.Category != "phishing" {
		t.Fatalf("expected category 'phishing', got %q", tip.Category)
	}
}
