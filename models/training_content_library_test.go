package models

import (
	"testing"
)

func TestGetBuiltInContentLibraryNotEmpty(t *testing.T) {
	lib := GetBuiltInContentLibrary()
	if len(lib) == 0 {
		t.Fatal("content library is empty")
	}
	// Should have content across all 4 tiers
	tierCounts := map[int]int{}
	for _, c := range lib {
		tierCounts[c.DifficultyLevel]++
	}
	for tier := 1; tier <= 4; tier++ {
		if tierCounts[tier] == 0 {
			t.Fatalf("no content for tier/difficulty %d", tier)
		}
	}
}

// validateContentFields checks that a single content item has all required fields.
func validateContentFields(t *testing.T, c BuiltInTrainingContent) {
	t.Helper()
	if c.Slug == "" {
		t.Fatal("content item has empty slug")
	}
	if c.Title == "" {
		t.Fatalf("content %q has empty title", c.Slug)
	}
	if c.Category == "" {
		t.Fatalf("content %q has empty category", c.Slug)
	}
	if c.Description == "" {
		t.Fatalf("content %q has empty description", c.Slug)
	}
	if c.DifficultyLevel < 1 || c.DifficultyLevel > 4 {
		t.Fatalf("content %q has invalid difficulty %d", c.Slug, c.DifficultyLevel)
	}
	if c.EstimatedMinutes <= 0 {
		t.Fatalf("content %q has invalid estimated minutes %d", c.Slug, c.EstimatedMinutes)
	}
	validateContentFieldCollections(t, c)
}

func validateContentFieldCollections(t *testing.T, c BuiltInTrainingContent) {
	t.Helper()
	if len(c.Pages) == 0 {
		t.Fatalf("content %q has no pages", c.Slug)
	}
	if len(c.Tags) == 0 {
		t.Fatalf("content %q has no tags", c.Slug)
	}
	if len(c.ComplianceMapped) == 0 {
		t.Fatalf("content %q has no compliance mappings", c.Slug)
	}
	if c.NanolearningTip == "" {
		t.Fatalf("content %q has no nanolearning tip", c.Slug)
	}
}

func TestGetBuiltInContentLibraryAllHaveRequiredFields(t *testing.T) {
	for _, c := range GetBuiltInContentLibrary() {
		validateContentFields(t, c)
	}
}

func TestGetBuiltInContentLibraryUniqueSlugs(t *testing.T) {
	seen := map[string]bool{}
	for _, c := range GetBuiltInContentLibrary() {
		if seen[c.Slug] {
			t.Fatalf("duplicate slug: %q", c.Slug)
		}
		seen[c.Slug] = true
	}
}

func TestGetBuiltInContentLibraryPagesHaveContent(t *testing.T) {
	for _, c := range GetBuiltInContentLibrary() {
		for i, page := range c.Pages {
			if page.Title == "" {
				t.Fatalf("content %q page %d has empty title", c.Slug, i)
			}
			if page.Body == "" {
				t.Fatalf("content %q page %d has empty body", c.Slug, i)
			}
		}
	}
}

// validateQuizQuestion checks a single quiz question for validity.
func validateQuizQuestion(t *testing.T, slug string, i int, q BuiltInQuestion) {
	t.Helper()
	if q.QuestionText == "" {
		t.Fatalf("content %q quiz question %d has empty text", slug, i)
	}
	if len(q.Options) < 2 {
		t.Fatalf("content %q quiz question %d has fewer than 2 options", slug, i)
	}
	if q.CorrectOption < 0 || q.CorrectOption >= len(q.Options) {
		t.Fatalf("content %q quiz question %d has invalid correct option %d (out of %d)",
			slug, i, q.CorrectOption, len(q.Options))
	}
}

// validateQuiz checks a content item's quiz for validity.
func validateQuiz(t *testing.T, c BuiltInTrainingContent) {
	t.Helper()
	if c.Quiz.PassPercentage <= 0 || c.Quiz.PassPercentage > 100 {
		t.Fatalf("content %q quiz has invalid pass percentage %d", c.Slug, c.Quiz.PassPercentage)
	}
	if len(c.Quiz.Questions) == 0 {
		t.Fatalf("content %q quiz has no questions", c.Slug)
	}
	for i, q := range c.Quiz.Questions {
		validateQuizQuestion(t, c.Slug, i, q)
	}
}

func TestGetBuiltInContentLibraryQuizzesValid(t *testing.T) {
	for _, c := range GetBuiltInContentLibrary() {
		if c.Quiz == nil {
			continue
		}
		validateQuiz(t, c)
	}
}

func TestGetBuiltInContentByCategory(t *testing.T) {
	phishing := GetBuiltInContentByCategory(ContentCategoryPhishing)
	if len(phishing) == 0 {
		t.Fatal("expected phishing category content")
	}
	for _, c := range phishing {
		if c.Category != ContentCategoryPhishing {
			t.Fatalf("expected category %q, got %q", ContentCategoryPhishing, c.Category)
		}
	}

	// Non-existent category
	empty := GetBuiltInContentByCategory("nonexistent")
	if len(empty) != 0 {
		t.Fatalf("expected no results for nonexistent category, got %d", len(empty))
	}
}

func TestGetBuiltInContentByDifficulty(t *testing.T) {
	bronze := GetBuiltInContentByDifficulty(ContentDiffBronze)
	if len(bronze) == 0 {
		t.Fatal("expected bronze difficulty content")
	}
	for _, c := range bronze {
		if c.DifficultyLevel != ContentDiffBronze {
			t.Fatalf("expected difficulty %d, got %d", ContentDiffBronze, c.DifficultyLevel)
		}
	}
}

func TestGetBuiltInContentBySlug(t *testing.T) {
	c := GetBuiltInContentBySlug("phishing-101")
	if c == nil {
		t.Fatal("expected phishing-101 content")
	}
	if c.Title != "Phishing 101 — Recognizing Phishing Emails" {
		t.Fatalf("unexpected title: %q", c.Title)
	}

	// Non-existent slug
	if GetBuiltInContentBySlug("nonexistent") != nil {
		t.Fatal("expected nil for nonexistent slug")
	}
}

func TestGetContentCategories(t *testing.T) {
	cats := GetContentCategories()
	if len(cats) == 0 {
		t.Fatal("expected at least one category")
	}
	for _, cat := range cats {
		if cat.Slug == "" {
			t.Fatal("category has empty slug")
		}
		if cat.Label == "" {
			t.Fatalf("category %q has empty label", cat.Slug)
		}
		if cat.Count <= 0 {
			t.Fatalf("category %q has count %d", cat.Slug, cat.Count)
		}
	}
}

func TestContentLibraryMinimumCoverage(t *testing.T) {
	lib := GetBuiltInContentLibrary()

	// We should have at least 12 content items
	if len(lib) < 12 {
		t.Fatalf("expected at least 12 content items, got %d", len(lib))
	}

	// Must cover at least 6 different categories
	cats := map[string]bool{}
	for _, c := range lib {
		cats[c.Category] = true
	}
	if len(cats) < 6 {
		t.Fatalf("expected at least 6 categories, got %d", len(cats))
	}

	// Every item should have at least 2 pages
	for _, c := range lib {
		if len(c.Pages) < 2 {
			t.Fatalf("content %q has only %d page(s), minimum is 2", c.Slug, len(c.Pages))
		}
	}

	// Most items should have a quiz (at least 80%)
	quizCount := 0
	for _, c := range lib {
		if c.Quiz != nil {
			quizCount++
		}
	}
	pct := float64(quizCount) / float64(len(lib)) * 100
	if pct < 80 {
		t.Fatalf("only %.0f%% of content has quizzes, expected at least 80%%", pct)
	}
}

func TestNanolearningTipsAreActionable(t *testing.T) {
	for _, c := range GetBuiltInContentLibrary() {
		tip := c.NanolearningTip
		// Tips should be substantial (at least 30 characters)
		if len(tip) < 30 {
			t.Fatalf("content %q nanolearning tip is too short (%d chars): %q", c.Slug, len(tip), tip)
		}
		// Tips should be a single paragraph (no newlines)
		for _, ch := range tip {
			if ch == '\n' {
				t.Fatalf("content %q nanolearning tip contains newline — should be a single sentence/paragraph", c.Slug)
			}
		}
	}
}
