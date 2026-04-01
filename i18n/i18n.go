package i18n

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"sync"

	log "github.com/gophish/gophish/logger"
)

// SupportedLanguages lists all available locale codes.
var SupportedLanguages = []string{"en", "nl", "fr", "de", "es"}

// DefaultLocale is the fallback locale when no match is found.
const DefaultLocale = "en"

// translations holds the loaded locale data: map[locale]map[key]string
var translations = make(map[string]map[string]string)
var mu sync.RWMutex

// LoadTranslations reads all JSON locale files from the given directory.
// Each file must be named {locale}.json (e.g. en.json, nl.json).
func LoadTranslations(localesDir string) error {
	mu.Lock()
	defer mu.Unlock()

	for _, lang := range SupportedLanguages {
		path := filepath.Join(localesDir, lang+".json")
		data, err := ioutil.ReadFile(path)
		if err != nil {
			log.Warnf("i18n: could not load locale file %s: %v", path, err)
			continue
		}
		var msgs map[string]string
		if err := json.Unmarshal(data, &msgs); err != nil {
			log.Warnf("i18n: invalid JSON in %s: %v", path, err)
			continue
		}
		translations[lang] = msgs
		log.Infof("i18n: loaded %d keys for locale %s", len(msgs), lang)
	}

	if _, ok := translations[DefaultLocale]; !ok {
		return fmt.Errorf("i18n: default locale %q not loaded", DefaultLocale)
	}
	return nil
}

// T translates a key for the given locale. If the key is missing in the
// requested locale, it falls back to the default locale. If still missing,
// the key itself is returned. Optional args are interpolated via fmt.Sprintf
// if the translated string contains %s/%d/etc.
func T(locale, key string, args ...interface{}) string {
	mu.RLock()
	defer mu.RUnlock()

	if msg, ok := translations[locale][key]; ok {
		if len(args) > 0 {
			return fmt.Sprintf(msg, args...)
		}
		return msg
	}
	// Fallback to default locale
	if locale != DefaultLocale {
		if msg, ok := translations[DefaultLocale][key]; ok {
			if len(args) > 0 {
				return fmt.Sprintf(msg, args...)
			}
			return msg
		}
	}
	return key
}

// GetTranslations returns the full translation map for a locale (for the
// frontend /api/i18n/:locale endpoint). Falls back to the default locale
// for any missing keys.
func GetTranslations(locale string) map[string]string {
	mu.RLock()
	defer mu.RUnlock()

	result := make(map[string]string)
	// Start with default locale as base
	if base, ok := translations[DefaultLocale]; ok {
		for k, v := range base {
			result[k] = v
		}
	}
	// Override with requested locale
	if locale != DefaultLocale {
		if loc, ok := translations[locale]; ok {
			for k, v := range loc {
				result[k] = v
			}
		}
	}
	return result
}

// IsSupported returns true if the locale code is in SupportedLanguages.
func IsSupported(locale string) bool {
	for _, l := range SupportedLanguages {
		if l == locale {
			return true
		}
	}
	return false
}

// DetectLocale determines the best locale from an Accept-Language header,
// user preference, and org default (in priority order).
func DetectLocale(userPref, orgDefault, acceptLang string) string {
	// 1. User preference takes priority
	if userPref != "" && IsSupported(userPref) {
		return userPref
	}
	// 2. Organization default
	if orgDefault != "" && IsSupported(orgDefault) {
		return orgDefault
	}
	// 3. Parse Accept-Language header
	if acceptLang != "" {
		if best := parseAcceptLanguage(acceptLang); best != "" {
			return best
		}
	}
	return DefaultLocale
}

// LanguageInfo represents a language option for the frontend selector.
type LanguageInfo struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// GetLanguages returns the list of available languages with display names.
func GetLanguages() []LanguageInfo {
	names := map[string]string{
		"en": "English",
		"nl": "Nederlands",
		"fr": "Français",
		"de": "Deutsch",
		"es": "Español",
	}
	var langs []LanguageInfo
	for _, code := range SupportedLanguages {
		langs = append(langs, LanguageInfo{
			Code: code,
			Name: names[code],
		})
	}
	return langs
}

// parseAcceptLanguage extracts the best supported language from an
// Accept-Language header value (simplified, ignores q-values).
func parseAcceptLanguage(header string) string {
	parts := strings.Split(header, ",")
	for _, part := range parts {
		lang := strings.TrimSpace(strings.SplitN(part, ";", 2)[0])
		// Try exact match first (e.g. "nl")
		if IsSupported(lang) {
			return lang
		}
		// Try base language (e.g. "nl-BE" → "nl")
		if idx := strings.IndexAny(lang, "-_"); idx > 0 {
			base := lang[:idx]
			if IsSupported(base) {
				return base
			}
		}
	}
	return ""
}
