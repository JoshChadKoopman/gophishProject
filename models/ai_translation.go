package models

import (
	"time"

	log "github.com/gophish/gophish/logger"
)

// ── AI-Powered Content Translation ──────────────────────────────
// Dynamic translation of training content, templates, and phishing
// simulations using AI, going beyond static locale files to support
// any language on demand.

// TranslationRequest represents a request to translate content.
type TranslationRequest struct {
	Id           int64     `json:"id" gorm:"primary_key"`
	OrgId        int64     `json:"org_id" gorm:"column:org_id"`
	UserId       int64     `json:"user_id" gorm:"column:user_id"`
	ContentType  string    `json:"content_type" gorm:"column:content_type"`    // "template", "training", "page", "email", "quiz"
	ContentId    int64     `json:"content_id" gorm:"column:content_id"`
	SourceLang   string    `json:"source_lang" gorm:"column:source_lang"`
	TargetLang   string    `json:"target_lang" gorm:"column:target_lang"`
	Status       string    `json:"status" gorm:"column:status"` // "pending", "completed", "failed"
	InputTokens  int       `json:"input_tokens" gorm:"column:input_tokens"`
	OutputTokens int       `json:"output_tokens" gorm:"column:output_tokens"`
	CreatedDate  time.Time `json:"created_date" gorm:"column:created_date"`
	CompletedAt  time.Time `json:"completed_at,omitempty" gorm:"column:completed_at"`
}

// TranslatedContent stores the AI-translated version of content.
type TranslatedContent struct {
	Id              int64     `json:"id" gorm:"primary_key"`
	OrgId           int64     `json:"org_id" gorm:"column:org_id"`
	ContentType     string    `json:"content_type" gorm:"column:content_type"`
	ContentId       int64     `json:"content_id" gorm:"column:content_id"`
	SourceLang      string    `json:"source_lang" gorm:"column:source_lang"`
	TargetLang      string    `json:"target_lang" gorm:"column:target_lang"`
	TranslatedTitle string    `json:"translated_title" gorm:"column:translated_title;type:text"`
	TranslatedBody  string    `json:"translated_body" gorm:"column:translated_body;type:text"`
	TranslatedHTML  string    `json:"translated_html,omitempty" gorm:"column:translated_html;type:text"`
	Quality         float64   `json:"quality" gorm:"column:quality"` // 0-100 confidence score
	ReviewedBy      int64     `json:"reviewed_by,omitempty" gorm:"column:reviewed_by"`
	ReviewedAt      time.Time `json:"reviewed_at,omitempty" gorm:"column:reviewed_at"`
	IsApproved      bool      `json:"is_approved" gorm:"column:is_approved"`
	CreatedDate     time.Time `json:"created_date" gorm:"column:created_date"`
	ModifiedDate    time.Time `json:"modified_date" gorm:"column:modified_date"`
}

// TranslationConfig stores org-level AI translation preferences.
type TranslationConfig struct {
	Id            int64    `json:"id" gorm:"primary_key"`
	OrgId         int64    `json:"org_id" gorm:"column:org_id;unique_index"`
	Enabled       bool     `json:"enabled" gorm:"column:enabled;default:true"`
	AutoTranslate bool     `json:"auto_translate" gorm:"column:auto_translate;default:false"`
	DefaultLangs  string   `json:"default_langs" gorm:"column:default_langs;type:text"` // JSON array ["nl","de","fr"]
	ReviewRequired bool    `json:"review_required" gorm:"column:review_required;default:true"`
	MaxMonthlyTokens int  `json:"max_monthly_tokens" gorm:"column:max_monthly_tokens;default:500000"`
}

// Translation status constants
const (
	TranslationStatusPending   = "pending"
	TranslationStatusCompleted = "completed"
	TranslationStatusFailed    = "failed"
)

// Content type constants for translation
const (
	TranslationContentTemplate = "template"
	TranslationContentTraining = "training"
	TranslationContentPage     = "page"
	TranslationContentEmail    = "email"
	TranslationContentQuiz     = "quiz"
)

// Supported languages for AI translation (beyond static locale files).
var AITranslationLanguages = map[string]string{
	"en": "English",
	"nl": "Dutch",
	"de": "German",
	"fr": "French",
	"es": "Spanish",
	"it": "Italian",
	"pt": "Portuguese",
	"pl": "Polish",
	"cs": "Czech",
	"da": "Danish",
	"sv": "Swedish",
	"no": "Norwegian",
	"fi": "Finnish",
	"ja": "Japanese",
	"ko": "Korean",
	"zh": "Chinese (Simplified)",
	"ar": "Arabic",
	"hi": "Hindi",
	"tr": "Turkish",
	"ro": "Romanian",
	"hu": "Hungarian",
	"el": "Greek",
	"bg": "Bulgarian",
	"hr": "Croatian",
	"sk": "Slovak",
	"sl": "Slovenian",
	"et": "Estonian",
	"lv": "Latvian",
	"lt": "Lithuanian",
	"uk": "Ukrainian",
	"ru": "Russian",
}

// TranslationUsageSummary is returned for the usage endpoint.
type TranslationUsageSummary struct {
	TotalRequests  int64 `json:"total_requests"`
	CompletedCount int64 `json:"completed_count"`
	FailedCount    int64 `json:"failed_count"`
	TotalInputTokens  int `json:"total_input_tokens"`
	TotalOutputTokens int `json:"total_output_tokens"`
	TotalTokens       int `json:"total_tokens"`
	LanguagesUsed     int `json:"languages_used"`
}

// Shared query constants for translations.
const (
	queryWhereOrgIDTranslation = "org_id = ?"
	queryWhereIDTranslation    = "id = ?"
)

// Table names for translation models.
func (TranslationRequest) TableName() string   { return "translation_requests" }
func (TranslatedContent) TableName() string    { return "translated_contents" }
func (TranslationConfig) TableName() string    { return "translation_configs" }

// IsValidTranslationLang checks if a language code is supported.
func IsValidTranslationLang(code string) bool {
	_, ok := AITranslationLanguages[code]
	return ok
}

// GetTranslationConfig returns the translation configuration for an org.
func GetTranslationConfig(orgId int64) TranslationConfig {
	cfg := TranslationConfig{}
	err := db.Where(queryWhereOrgIDTranslation, orgId).First(&cfg).Error
	if err != nil {
		cfg = TranslationConfig{
			OrgId:            orgId,
			Enabled:          true,
			AutoTranslate:    false,
			ReviewRequired:   true,
			MaxMonthlyTokens: 500000,
		}
	}
	return cfg
}

// SaveTranslationConfig upserts the translation config.
func SaveTranslationConfig(cfg *TranslationConfig) error {
	existing := TranslationConfig{}
	err := db.Where(queryWhereOrgIDTranslation, cfg.OrgId).First(&existing).Error
	if err != nil {
		return db.Save(cfg).Error
	}
	cfg.Id = existing.Id
	return db.Save(cfg).Error
}

// CreateTranslationRequest stores a new translation request.
func CreateTranslationRequest(req *TranslationRequest) error {
	req.CreatedDate = time.Now().UTC()
	if req.Status == "" {
		req.Status = TranslationStatusPending
	}
	return db.Save(req).Error
}

// CompleteTranslationRequest marks a request as completed.
func CompleteTranslationRequest(id int64, inputTokens, outputTokens int) error {
	return db.Model(&TranslationRequest{}).
		Where(queryWhereIDTranslation, id).
		Updates(map[string]interface{}{
			"status":        TranslationStatusCompleted,
			"input_tokens":  inputTokens,
			"output_tokens": outputTokens,
			"completed_at":  time.Now().UTC(),
		}).Error
}

// FailTranslationRequest marks a request as failed.
func FailTranslationRequest(id int64) error {
	return db.Model(&TranslationRequest{}).
		Where(queryWhereIDTranslation, id).
		Update("status", TranslationStatusFailed).Error
}

// SaveTranslatedContent stores a translated content entry.
func SaveTranslatedContent(tc *TranslatedContent) error {
	tc.CreatedDate = time.Now().UTC()
	tc.ModifiedDate = tc.CreatedDate
	return db.Save(tc).Error
}

// GetTranslatedContent retrieves a cached translation for the given content and language.
func GetTranslatedContent(contentType string, contentId int64, targetLang string) (*TranslatedContent, error) {
	tc := &TranslatedContent{}
	err := db.Where("content_type = ? AND content_id = ? AND target_lang = ?",
		contentType, contentId, targetLang).
		Order("created_date desc").
		First(tc).Error
	if err != nil {
		return nil, err
	}
	return tc, nil
}

// GetTranslationsForContent returns all translations for a piece of content.
func GetTranslationsForContent(contentType string, contentId int64) ([]TranslatedContent, error) {
	translations := []TranslatedContent{}
	err := db.Where("content_type = ? AND content_id = ?", contentType, contentId).
		Order("target_lang asc").
		Find(&translations).Error
	return translations, err
}

// ApproveTranslation marks a translation as reviewed and approved.
func ApproveTranslation(id int64, reviewerUserId int64) error {
	return db.Model(&TranslatedContent{}).
		Where(queryWhereIDTranslation, id).
		Updates(map[string]interface{}{
			"is_approved":   true,
			"reviewed_by":   reviewerUserId,
			"reviewed_at":   time.Now().UTC(),
			"modified_date": time.Now().UTC(),
		}).Error
}

// GetTranslationRequests returns recent translation requests for an org.
func GetTranslationRequests(orgId int64, limit int) ([]TranslationRequest, error) {
	if limit <= 0 {
		limit = 50
	}
	requests := []TranslationRequest{}
	err := db.Where(queryWhereOrgIDTranslation, orgId).
		Order("created_date desc").
		Limit(limit).
		Find(&requests).Error
	return requests, err
}

// GetTranslationUsageSummary returns aggregate translation usage for an org.
func GetTranslationUsageSummary(orgId int64, since time.Time) (*TranslationUsageSummary, error) {
	summary := &TranslationUsageSummary{}

	// Totals
	row := db.Table("translation_requests").
		Select("COUNT(*) as total_requests, "+
			"SUM(CASE WHEN status='completed' THEN 1 ELSE 0 END) as completed_count, "+
			"SUM(CASE WHEN status='failed' THEN 1 ELSE 0 END) as failed_count, "+
			"COALESCE(SUM(input_tokens),0) as total_input_tokens, "+
			"COALESCE(SUM(output_tokens),0) as total_output_tokens").
		Where(queryWhereOrgIDTranslation+" AND created_date >= ?", orgId, since).
		Row()
	err := row.Scan(&summary.TotalRequests, &summary.CompletedCount, &summary.FailedCount,
		&summary.TotalInputTokens, &summary.TotalOutputTokens)
	if err != nil {
		log.Error(err)
		return summary, err
	}
	summary.TotalTokens = summary.TotalInputTokens + summary.TotalOutputTokens

	// Distinct languages
	var langCount int64
	db.Table("translation_requests").
		Where(queryWhereOrgIDTranslation+" AND created_date >= ?", orgId, since).
		Select("COUNT(DISTINCT target_lang)").
		Row().Scan(&langCount)
	summary.LanguagesUsed = int(langCount)

	return summary, nil
}
