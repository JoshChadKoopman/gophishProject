package models

import (
	"testing"
)

func TestBuiltInContentLibraryNotEmpty(t *testing.T) {
	if len(builtInContentLibrary) == 0 {
		t.Fatal("builtInContentLibrary should not be empty")
	}
}

func TestContentLibraryUniqueSlugs(t *testing.T) {
	seen := make(map[string]bool)
	for _, c := range builtInContentLibrary {
		if seen[c.Slug] {
			t.Errorf("duplicate content slug: %s", c.Slug)
		}
		seen[c.Slug] = true
	}
}

func TestContentLibraryRequiredFields(t *testing.T) {
	for _, c := range builtInContentLibrary {
		if c.Slug == "" {
			t.Error("content has empty slug")
		}
		if c.Title == "" {
			t.Errorf("content %s has empty title", c.Slug)
		}
		if c.Category == "" {
			t.Errorf("content %s has empty category", c.Slug)
		}
		if c.Description == "" {
			t.Errorf("content %s has empty description", c.Slug)
		}
		if c.DifficultyLevel < 1 || c.DifficultyLevel > 4 {
			t.Errorf("content %s has invalid difficulty: %d", c.Slug, c.DifficultyLevel)
		}
		if c.EstimatedMinutes <= 0 {
			t.Errorf("content %s has invalid estimated minutes: %d", c.Slug, c.EstimatedMinutes)
		}
		if len(c.Pages) == 0 {
			t.Errorf("content %s has no pages", c.Slug)
		}
	}
}

func TestContentLibraryPagesHaveContent(t *testing.T) {
	for _, c := range builtInContentLibrary {
		for i, p := range c.Pages {
			if p.Title == "" {
				t.Errorf("content %s page %d has empty title", c.Slug, i)
			}
			if p.Body == "" {
				t.Errorf("content %s page %d has empty body", c.Slug, i)
			}
		}
	}
}

func TestContentLibraryQuizzes(t *testing.T) {
	quizCount := 0
	for _, c := range builtInContentLibrary {
		if c.Quiz == nil {
			continue
		}
		quizCount++
		if c.Quiz.PassPercentage <= 0 || c.Quiz.PassPercentage > 100 {
			t.Errorf("content %s has invalid pass percentage: %d", c.Slug, c.Quiz.PassPercentage)
		}
		if len(c.Quiz.Questions) == 0 {
			t.Errorf("content %s has quiz with no questions", c.Slug)
		}
		for j, q := range c.Quiz.Questions {
			if q.QuestionText == "" {
				t.Errorf("content %s question %d has empty text", c.Slug, j)
			}
			if len(q.Options) < 2 {
				t.Errorf("content %s question %d has fewer than 2 options", c.Slug, j)
			}
			if q.CorrectOption < 0 || q.CorrectOption >= len(q.Options) {
				t.Errorf("content %s question %d has out-of-range correct option: %d (options: %d)", c.Slug, j, q.CorrectOption, len(q.Options))
			}
		}
	}
	if quizCount == 0 {
		t.Error("expected at least one content entry with a quiz")
	}
}

func TestContentLibraryHasTags(t *testing.T) {
	for _, c := range builtInContentLibrary {
		if len(c.Tags) == 0 {
			t.Errorf("content %s has no tags", c.Slug)
		}
	}
}

func TestContentLibraryDifficultyDistribution(t *testing.T) {
	levels := make(map[int]int)
	for _, c := range builtInContentLibrary {
		levels[c.DifficultyLevel]++
	}
	// Should have content at multiple difficulty levels
	if len(levels) < 2 {
		t.Errorf("expected content at multiple difficulty levels, found %d", len(levels))
	}
}

func TestContentLibraryCategoryDistribution(t *testing.T) {
	cats := make(map[string]int)
	for _, c := range builtInContentLibrary {
		cats[c.Category]++
	}
	// Should have content across multiple categories
	if len(cats) < 2 {
		t.Errorf("expected content in multiple categories, found %d", len(cats))
	}
}
