package i18n

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// Shared test constants for i18n tests.
const (
	fmtFailedTempDir = "failed to create temp dir: %v"
	testLogOutLabel  = "Log Out"
)

// setupTestLocales creates a temporary directory with locale JSON files for
// testing. It returns the directory path and a cleanup function.
func setupTestLocales(t *testing.T) (string, func()) {
	t.Helper()
	dir, err := os.MkdirTemp("", "i18n-test-*")
	if err != nil {
		t.Fatalf(fmtFailedTempDir, err)
	}

	enData := map[string]string{
		"greeting":      "Hello, %s!",
		"logout":        testLogOutLabel,
		"dashboard":     "Dashboard",
		"save":          "Save",
		"campaign.name": "Campaign Name",
	}
	nlData := map[string]string{
		"greeting":      "Hallo, %s!",
		"logout":        "Uitloggen",
		"dashboard":     "Dashboard",
		"campaign.name": "Campagnenaam",
	}
	frData := map[string]string{
		"greeting": "Bonjour, %s!",
		"logout":   "Déconnexion",
	}

	writeLocale(t, dir, "en", enData)
	writeLocale(t, dir, "nl", nlData)
	writeLocale(t, dir, "fr", frData)

	cleanup := func() {
		os.RemoveAll(dir)
		// Reset global translations state
		mu.Lock()
		translations = make(map[string]map[string]string)
		mu.Unlock()
	}
	return dir, cleanup
}

func writeLocale(t *testing.T, dir, lang string, data map[string]string) {
	t.Helper()
	raw, err := json.Marshal(data)
	if err != nil {
		t.Fatalf("failed to marshal %s locale: %v", lang, err)
	}
	path := filepath.Join(dir, lang+".json")
	if err := os.WriteFile(path, raw, 0644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
}

// ---------- LoadTranslations tests ----------

func TestLoadTranslationsSuccess(t *testing.T) {
	dir, cleanup := setupTestLocales(t)
	defer cleanup()

	err := LoadTranslations(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadTranslationsMissingDefaultLocale(t *testing.T) {
	dir, err := os.MkdirTemp("", "i18n-no-en-*")
	if err != nil {
		t.Fatalf(fmtFailedTempDir, err)
	}
	defer os.RemoveAll(dir)

	// Reset translations
	mu.Lock()
	translations = make(map[string]map[string]string)
	mu.Unlock()

	// Write only a non-default locale
	writeLocale(t, dir, "nl", map[string]string{"key": "val"})

	err = LoadTranslations(dir)
	if err == nil {
		t.Fatal("expected error when default locale is missing")
	}
}

func TestLoadTranslationsInvalidJSON(t *testing.T) {
	dir, err := os.MkdirTemp("", "i18n-bad-json-*")
	if err != nil {
		t.Fatalf(fmtFailedTempDir, err)
	}
	defer os.RemoveAll(dir)

	// Reset translations
	mu.Lock()
	translations = make(map[string]map[string]string)
	mu.Unlock()

	// Write a valid en.json and an invalid nl.json
	writeLocale(t, dir, "en", map[string]string{"key": "val"})
	os.WriteFile(filepath.Join(dir, "nl.json"), []byte("not valid json{"), 0644)

	err = LoadTranslations(dir)
	if err != nil {
		t.Fatalf("should not error — invalid locale files are skipped: %v", err)
	}

	// nl should not be loaded
	mu.RLock()
	_, nlLoaded := translations["nl"]
	mu.RUnlock()
	if nlLoaded {
		t.Fatal("nl should not have been loaded due to invalid JSON")
	}
}

// ---------- T (translate) tests ----------

func TestTExactMatch(t *testing.T) {
	dir, cleanup := setupTestLocales(t)
	defer cleanup()
	LoadTranslations(dir)

	got := T("nl", "logout")
	if got != "Uitloggen" {
		t.Fatalf("expected 'Uitloggen', got %q", got)
	}
}

func TestTWithInterpolation(t *testing.T) {
	dir, cleanup := setupTestLocales(t)
	defer cleanup()
	LoadTranslations(dir)

	got := T("en", "greeting", "World")
	if got != "Hello, World!" {
		t.Fatalf("expected 'Hello, World!', got %q", got)
	}

	got = T("nl", "greeting", "Wereld")
	if got != "Hallo, Wereld!" {
		t.Fatalf("expected 'Hallo, Wereld!', got %q", got)
	}
}

func TestTFallbackToDefault(t *testing.T) {
	dir, cleanup := setupTestLocales(t)
	defer cleanup()
	LoadTranslations(dir)

	// "save" exists only in en, not in nl
	got := T("nl", "save")
	if got != "Save" {
		t.Fatalf("expected fallback 'Save', got %q", got)
	}
}

func TestTMissingKeyReturnsKey(t *testing.T) {
	dir, cleanup := setupTestLocales(t)
	defer cleanup()
	LoadTranslations(dir)

	got := T("en", "nonexistent.key")
	if got != "nonexistent.key" {
		t.Fatalf("expected key itself as fallback, got %q", got)
	}
}

func TestTDefaultLocaleNoDoubleLookup(t *testing.T) {
	dir, cleanup := setupTestLocales(t)
	defer cleanup()
	LoadTranslations(dir)

	got := T("en", "dashboard")
	if got != "Dashboard" {
		t.Fatalf("expected 'Dashboard', got %q", got)
	}
}

func TestTUnknownLocale(t *testing.T) {
	dir, cleanup := setupTestLocales(t)
	defer cleanup()
	LoadTranslations(dir)

	// Unknown locale should fall back to default
	got := T("xx", "logout")
	if got != testLogOutLabel {
		t.Fatalf("expected fallback 'Log Out', got %q", got)
	}
}

// ---------- GetTranslations tests ----------

func TestGetTranslationsDefault(t *testing.T) {
	dir, cleanup := setupTestLocales(t)
	defer cleanup()
	LoadTranslations(dir)

	result := GetTranslations("en")
	if result["logout"] != testLogOutLabel {
		t.Fatalf("expected 'Log Out', got %q", result["logout"])
	}
	if result["save"] != "Save" {
		t.Fatalf("expected 'Save', got %q", result["save"])
	}
}

func TestGetTranslationsOverride(t *testing.T) {
	dir, cleanup := setupTestLocales(t)
	defer cleanup()
	LoadTranslations(dir)

	result := GetTranslations("nl")
	// Overridden key
	if result["logout"] != "Uitloggen" {
		t.Fatalf("expected 'Uitloggen', got %q", result["logout"])
	}
	// Fallback key from English base
	if result["save"] != "Save" {
		t.Fatalf("expected 'Save' from en base, got %q", result["save"])
	}
}

func TestGetTranslationsUnknownLocale(t *testing.T) {
	dir, cleanup := setupTestLocales(t)
	defer cleanup()
	LoadTranslations(dir)

	result := GetTranslations("xx")
	// Should return only the default locale translations
	if result["logout"] != testLogOutLabel {
		t.Fatalf("expected 'Log Out', got %q", result["logout"])
	}
}

// ---------- IsSupported tests ----------

func TestIsSupportedTrue(t *testing.T) {
	for _, lang := range []string{"en", "nl", "fr", "de", "ja", "ar"} {
		if !IsSupported(lang) {
			t.Fatalf("expected %q to be supported", lang)
		}
	}
}

func TestIsSupportedFalse(t *testing.T) {
	for _, lang := range []string{"xx", "klingon", "EN", "NL", ""} {
		if IsSupported(lang) {
			t.Fatalf("expected %q to NOT be supported", lang)
		}
	}
}

// ---------- DetectLocale tests ----------

func TestDetectLocaleUserPref(t *testing.T) {
	got := DetectLocale("nl", "fr", "de")
	if got != "nl" {
		t.Fatalf("expected 'nl' (user pref), got %q", got)
	}
}

func TestDetectLocaleOrgDefault(t *testing.T) {
	got := DetectLocale("", "fr", "de")
	if got != "fr" {
		t.Fatalf("expected 'fr' (org default), got %q", got)
	}
}

func TestDetectLocaleAcceptLanguage(t *testing.T) {
	got := DetectLocale("", "", "de,en;q=0.9")
	if got != "de" {
		t.Fatalf("expected 'de' from Accept-Language, got %q", got)
	}
}

func TestDetectLocaleFallback(t *testing.T) {
	got := DetectLocale("", "", "")
	if got != DefaultLocale {
		t.Fatalf("expected default locale %q, got %q", DefaultLocale, got)
	}
}

func TestDetectLocaleInvalidUserPref(t *testing.T) {
	got := DetectLocale("xx", "", "nl")
	if got != "nl" {
		t.Fatalf("expected 'nl' from Accept-Language when user pref is invalid, got %q", got)
	}
}

// ---------- parseAcceptLanguage tests ----------

func TestParseAcceptLanguageExact(t *testing.T) {
	got := parseAcceptLanguage("nl")
	if got != "nl" {
		t.Fatalf("expected 'nl', got %q", got)
	}
}

func TestParseAcceptLanguageWithRegion(t *testing.T) {
	got := parseAcceptLanguage("nl-BE,en;q=0.9")
	if got != "nl" {
		t.Fatalf("expected 'nl' from 'nl-BE', got %q", got)
	}
}

func TestParseAcceptLanguageMultiple(t *testing.T) {
	got := parseAcceptLanguage("xx,fr,en;q=0.5")
	if got != "fr" {
		t.Fatalf("expected 'fr' (first supported), got %q", got)
	}
}

func TestParseAcceptLanguageNoneSupported(t *testing.T) {
	got := parseAcceptLanguage("xx,yy")
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestParseAcceptLanguageEmpty(t *testing.T) {
	got := parseAcceptLanguage("")
	if got != "" {
		t.Fatalf("expected empty string, got %q", got)
	}
}

func TestParseAcceptLanguageWithUnderscore(t *testing.T) {
	got := parseAcceptLanguage("pt_BR")
	if got != "pt" {
		t.Fatalf("expected 'pt' from 'pt_BR', got %q", got)
	}
}

// ---------- GetLanguages tests ----------

func TestGetLanguages(t *testing.T) {
	langs := GetLanguages()
	if len(langs) != len(SupportedLanguages) {
		t.Fatalf("expected %d languages, got %d", len(SupportedLanguages), len(langs))
	}

	// Verify first is English
	if langs[0].Code != "en" || langs[0].Name != "English" {
		t.Fatalf("expected first language to be English, got %+v", langs[0])
	}

	// Verify Dutch is present
	found := false
	for _, l := range langs {
		if l.Code == "nl" && l.Name == "Nederlands" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected Nederlands in language list")
	}

	// Every entry should have non-empty Code and Name
	for _, l := range langs {
		if l.Code == "" || l.Name == "" {
			t.Fatalf("language entry with empty code or name: %+v", l)
		}
	}
}
