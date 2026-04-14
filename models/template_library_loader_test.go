package models

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDeduplicateTemplates(t *testing.T) {
	primary := []LibraryTemplate{
		{Slug: "a", Name: "Primary A"},
		{Slug: "b", Name: "Primary B"},
	}
	fallback := []LibraryTemplate{
		{Slug: "b", Name: "Fallback B"}, // duplicate — should be skipped
		{Slug: "c", Name: "Fallback C"},
	}
	merged := deduplicateTemplates(primary, fallback)
	if len(merged) != 3 {
		t.Fatalf("expected 3 merged templates, got %d", len(merged))
	}
	// Primary B should win over Fallback B
	for _, m := range merged {
		if m.Slug == "b" && m.Name != "Primary B" {
			t.Fatalf("expected primary to win for slug 'b', got %q", m.Name)
		}
	}
}

func TestDeduplicateTemplatesEmptyPrimary(t *testing.T) {
	fallback := []LibraryTemplate{
		{Slug: "x", Name: "X"},
	}
	merged := deduplicateTemplates(nil, fallback)
	if len(merged) != 1 {
		t.Fatalf("expected 1 template, got %d", len(merged))
	}
}

func TestDeduplicateTemplatesEmptyFallback(t *testing.T) {
	primary := []LibraryTemplate{
		{Slug: "x", Name: "X"},
	}
	merged := deduplicateTemplates(primary, nil)
	if len(merged) != 1 {
		t.Fatalf("expected 1 template, got %d", len(merged))
	}
}

func TestLoadTemplateFileValid(t *testing.T) {
	dir := t.TempDir()
	templates := []LibraryTemplate{
		{Slug: "test-1", Name: "Test 1", Category: "Test", DifficultyLevel: 1},
		{Slug: "test-2", Name: "Test 2", Category: "Test", DifficultyLevel: 2},
	}
	data, _ := json.Marshal(templates)
	path := filepath.Join(dir, "test.json")
	os.WriteFile(path, data, 0644)

	loaded, err := loadTemplateFile(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(loaded) != 2 {
		t.Fatalf("expected 2 templates, got %d", len(loaded))
	}
	if loaded[0].Slug != "test-1" {
		t.Fatalf("expected slug 'test-1', got %q", loaded[0].Slug)
	}
}

func TestLoadTemplateFileInvalid(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	os.WriteFile(path, []byte("not json"), 0644)

	_, err := loadTemplateFile(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestLoadTemplateFileMissing(t *testing.T) {
	_, err := loadTemplateFile("/nonexistent/file.json")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadTemplatesFromDir(t *testing.T) {
	dir := t.TempDir()

	// Create two valid JSON files
	t1 := []LibraryTemplate{{Slug: "a", Name: "A"}}
	t2 := []LibraryTemplate{{Slug: "b", Name: "B"}, {Slug: "c", Name: "C"}}
	d1, _ := json.Marshal(t1)
	d2, _ := json.Marshal(t2)
	os.WriteFile(filepath.Join(dir, "cat1.json"), d1, 0644)
	os.WriteFile(filepath.Join(dir, "cat2.json"), d2, 0644)
	// Create a non-JSON file that should be ignored
	os.WriteFile(filepath.Join(dir, "readme.txt"), []byte("ignore me"), 0644)

	loaded, err := loadTemplatesFromDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(loaded) != 3 {
		t.Fatalf("expected 3 templates from 2 files, got %d", len(loaded))
	}
}

func TestLoadTemplatesFromDirMissing(t *testing.T) {
	_, err := loadTemplatesFromDir("/nonexistent/dir")
	if err == nil {
		t.Fatal("expected error for missing directory")
	}
}

func TestLoadTemplatesFromDirSkipsBadFiles(t *testing.T) {
	dir := t.TempDir()
	// One valid, one invalid
	good := []LibraryTemplate{{Slug: "good", Name: "Good"}}
	d, _ := json.Marshal(good)
	os.WriteFile(filepath.Join(dir, "good.json"), d, 0644)
	os.WriteFile(filepath.Join(dir, "bad.json"), []byte("broken"), 0644)

	loaded, err := loadTemplatesFromDir(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(loaded) != 1 {
		t.Fatalf("expected 1 template (skipping bad file), got %d", len(loaded))
	}
}

// TestLoadTemplatesFromRealDir verifies all JSON files shipped in
// static/db/templates parse correctly and every template has required fields.
// This catches hand-authored mistakes in the template data files.
func TestLoadTemplatesFromRealDir(t *testing.T) {
	dir := "../static/db/templates"
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Skip("template library dir not present")
	}
	loaded, err := loadTemplatesFromDir(dir)
	if err != nil {
		t.Fatalf("failed to load real template dir: %v", err)
	}
	if len(loaded) < 50 {
		t.Fatalf("expected at least 50 templates in library, got %d", len(loaded))
	}

	seen := make(map[string]bool)
	for i, tmpl := range loaded {
		validateLibraryTemplate(t, i, tmpl)
		if seen[tmpl.Slug] {
			t.Errorf("duplicate slug: %q", tmpl.Slug)
		}
		seen[tmpl.Slug] = true
	}
}

func validateLibraryTemplate(t *testing.T, idx int, tmpl LibraryTemplate) {
	t.Helper()
	if tmpl.Slug == "" {
		t.Errorf("template %d: slug is empty", idx)
	}
	if tmpl.Name == "" {
		t.Errorf("template %q: name is empty", tmpl.Slug)
	}
	if tmpl.Subject == "" {
		t.Errorf("template %q: subject is empty", tmpl.Slug)
	}
	if tmpl.HTML == "" && tmpl.Text == "" {
		t.Errorf("template %q: both HTML and text are empty", tmpl.Slug)
	}
	if tmpl.Category == "" {
		t.Errorf("template %q: category is empty", tmpl.Slug)
	}
	if tmpl.DifficultyLevel < 1 || tmpl.DifficultyLevel > 4 {
		t.Errorf("template %q: difficulty level %d out of range 1-4", tmpl.Slug, tmpl.DifficultyLevel)
	}
}
