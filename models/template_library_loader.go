package models

import (
	"encoding/json"
	"os"
	"path/filepath"

	log "github.com/gophish/gophish/logger"
)

// TemplateLibraryDir is the directory containing JSON template library files.
const TemplateLibraryDir = "./static/db/templates"

// LoadTemplateLibrary scans the templates directory for JSON files and loads
// all templates into the in-memory TemplateLibrary slice. Each JSON file should
// contain an array of LibraryTemplate objects.
//
// If the directory does not exist or contains no valid files, the hardcoded
// fallback templates in TemplateLibrary remain unchanged.
func LoadTemplateLibrary() {
	loaded, err := loadTemplatesFromDir(TemplateLibraryDir)
	if err != nil {
		log.Infof("Template library: using %d built-in templates (no JSON dir: %v)", len(TemplateLibrary), err)
		return
	}
	if len(loaded) == 0 {
		log.Infof("Template library: no JSON templates found, using %d built-in templates", len(TemplateLibrary))
		return
	}

	// Merge: JSON templates first, then built-in templates that don't conflict
	merged := deduplicateTemplates(loaded, TemplateLibrary)
	TemplateLibrary = merged
	log.Infof("Template library: loaded %d templates (%d from JSON, %d built-in)",
		len(merged), len(loaded), len(merged)-len(loaded))
}

// loadTemplatesFromDir reads all .json files in the directory and parses them
// as arrays of LibraryTemplate.
func loadTemplatesFromDir(dir string) ([]LibraryTemplate, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	var all []LibraryTemplate
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(dir, entry.Name())
		templates, err := loadTemplateFile(path)
		if err != nil {
			log.Errorf("Template library: failed to load %s: %v", entry.Name(), err)
			continue
		}
		all = append(all, templates...)
		log.Infof("Template library: loaded %d templates from %s", len(templates), entry.Name())
	}
	return all, nil
}

func loadTemplateFile(path string) ([]LibraryTemplate, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var templates []LibraryTemplate
	if err := json.Unmarshal(data, &templates); err != nil {
		return nil, err
	}
	return templates, nil
}

// deduplicateTemplates merges primary and fallback template lists. If a slug
// exists in both, the primary (JSON) version wins.
func deduplicateTemplates(primary, fallback []LibraryTemplate) []LibraryTemplate {
	seen := make(map[string]bool, len(primary))
	for _, t := range primary {
		seen[t.Slug] = true
	}

	merged := make([]LibraryTemplate, len(primary))
	copy(merged, primary)

	for _, t := range fallback {
		if !seen[t.Slug] {
			merged = append(merged, t)
			seen[t.Slug] = true
		}
	}
	return merged
}
