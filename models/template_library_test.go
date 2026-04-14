package models

import (
	"testing"
)

func TestTemplateLibraryNotEmpty(t *testing.T) {
	if len(TemplateLibrary) == 0 {
		t.Fatal("TemplateLibrary should not be empty")
	}
}

func TestTemplateLibraryUniqueSlugs(t *testing.T) {
	seen := make(map[string]bool)
	for _, tmpl := range TemplateLibrary {
		if seen[tmpl.Slug] {
			t.Errorf("duplicate slug: %s", tmpl.Slug)
		}
		seen[tmpl.Slug] = true
	}
}

func TestTemplateLibraryRequiredFields(t *testing.T) {
	for _, tmpl := range TemplateLibrary {
		if tmpl.Slug == "" {
			t.Error("template has empty slug")
		}
		if tmpl.Name == "" {
			t.Errorf("template %s has empty name", tmpl.Slug)
		}
		if tmpl.Category == "" {
			t.Errorf("template %s has empty category", tmpl.Slug)
		}
		// SMS templates don't have subjects
		if tmpl.Subject == "" && tmpl.Category != "SMS Phishing" {
			t.Errorf("template %s has empty subject", tmpl.Slug)
		}
		if tmpl.DifficultyLevel < 1 || tmpl.DifficultyLevel > 5 {
			t.Errorf("template %s has invalid difficulty: %d", tmpl.Slug, tmpl.DifficultyLevel)
		}
		if tmpl.Language == "" {
			t.Errorf("template %s has empty language", tmpl.Slug)
		}
		// Must have at least text or HTML
		if tmpl.Text == "" && tmpl.HTML == "" {
			t.Errorf("template %s has neither text nor HTML content", tmpl.Slug)
		}
	}
}

func TestGetTemplateLibraryNoFilter(t *testing.T) {
	result := GetTemplateLibrary("", 0)
	if len(result) != len(TemplateLibrary) {
		t.Errorf("expected %d templates, got %d", len(TemplateLibrary), len(result))
	}
}

func TestGetTemplateLibraryByCategory(t *testing.T) {
	// Use the category of the first template
	cat := TemplateLibrary[0].Category
	result := GetTemplateLibrary(cat, 0)
	if len(result) == 0 {
		t.Errorf("expected templates for category %s", cat)
	}
	for _, tmpl := range result {
		if tmpl.Category != cat {
			t.Errorf("expected category %s, got %s", cat, tmpl.Category)
		}
	}
}

func TestGetTemplateLibraryByDifficulty(t *testing.T) {
	result := GetTemplateLibrary("", 1)
	if len(result) == 0 {
		t.Error("expected templates for difficulty 1")
	}
	for _, tmpl := range result {
		if tmpl.DifficultyLevel != 1 {
			t.Errorf("expected difficulty 1, got %d", tmpl.DifficultyLevel)
		}
	}
}

func TestGetTemplateLibraryFilteredByLanguage(t *testing.T) {
	result := GetTemplateLibraryFiltered("", 0, "en")
	if len(result) == 0 {
		t.Error("expected English templates")
	}
	for _, tmpl := range result {
		if tmpl.Language != "en" {
			t.Errorf("expected language 'en', got '%s'", tmpl.Language)
		}
	}
}

func TestGetTemplateLibraryFilteredNoMatch(t *testing.T) {
	result := GetTemplateLibraryFiltered("", 0, "xx_nonexistent")
	if len(result) != 0 {
		t.Errorf("expected 0 templates for nonexistent language, got %d", len(result))
	}
}

func TestGetLibraryTemplateFound(t *testing.T) {
	slug := TemplateLibrary[0].Slug
	tmpl, ok := GetLibraryTemplate(slug)
	if !ok {
		t.Errorf("expected to find template %s", slug)
	}
	if tmpl.Slug != slug {
		t.Errorf("expected slug %s, got %s", slug, tmpl.Slug)
	}
}

func TestGetLibraryTemplateNotFound(t *testing.T) {
	_, ok := GetLibraryTemplate("nonexistent-slug-xyz")
	if ok {
		t.Error("expected not to find nonexistent template")
	}
}

func TestGetTemplateLibraryStats(t *testing.T) {
	stats := GetTemplateLibraryStats()
	if stats.TotalTemplates != len(TemplateLibrary) {
		t.Errorf("expected total %d, got %d", len(TemplateLibrary), stats.TotalTemplates)
	}
}

func TestGetTemplateLibraryCategories(t *testing.T) {
	cats := GetTemplateLibraryCategories()
	if len(cats) == 0 {
		t.Error("expected at least one category")
	}
	// Categories should be unique
	seen := make(map[string]bool)
	for _, c := range cats {
		if seen[c] {
			t.Errorf("duplicate category: %s", c)
		}
		seen[c] = true
	}
}
