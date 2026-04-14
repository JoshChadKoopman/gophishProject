package models

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ── Database-Backed Template Library ────────────────────────────
// Extends the hardcoded template library with a persistent DB-backed store
// that supports CRUD, import/export, tagging, deduplication, and full-text search.

// DBLibraryTemplate is the database-persisted version of LibraryTemplate.
type DBLibraryTemplate struct {
	Id              int64     `json:"id" gorm:"primary_key"`
	Slug            string    `json:"slug" gorm:"unique_index"`
	Name            string    `json:"name"`
	Category        string    `json:"category"`
	DifficultyLevel int       `json:"difficulty_level"`
	Description     string    `json:"description"`
	Subject         string    `json:"subject"`
	Text            string    `json:"text" gorm:"type:text"`
	HTML            string    `json:"html" gorm:"type:text"`
	EnvelopeSender  string    `json:"envelope_sender"`
	Language        string    `json:"language"`
	TargetRole      string    `json:"target_role"`
	Tags            string    `json:"tags" gorm:"type:text"` // JSON array of tags
	SimilarityHash  string    `json:"similarity_hash"`       // SHA-256 of normalised subject+text for dedup
	Source          string    `json:"source"`                 // "builtin", "user", "ai_generated", "community"
	OrgId           int64     `json:"org_id"`                 // 0 = global, >0 = org-specific
	CreatedBy       int64     `json:"created_by"`
	IsPublished     bool      `json:"is_published" gorm:"default:true"`
	UsageCount      int64     `json:"usage_count"`            // How many times imported into a campaign
	AvgClickRate    float64   `json:"avg_click_rate"`         // Average click rate when used
	CreatedDate     time.Time `json:"created_date"`
	ModifiedDate    time.Time `json:"modified_date"`
}

func (DBLibraryTemplate) TableName() string { return "library_templates" }

// GetTags returns the tags as a string slice.
func (t *DBLibraryTemplate) GetTags() []string {
	var tags []string
	if t.Tags != "" {
		json.Unmarshal([]byte(t.Tags), &tags)
	}
	return tags
}

// SetTags serialises tags to JSON.
func (t *DBLibraryTemplate) SetTags(tags []string) {
	if len(tags) == 0 {
		t.Tags = "[]"
		return
	}
	b, _ := json.Marshal(tags)
	t.Tags = string(b)
}

// computeSimilarityHash generates a deduplication hash from the template content.
func computeSimilarityHash(subject, text string) string {
	normalised := strings.ToLower(strings.TrimSpace(subject)) + "|" + strings.ToLower(strings.TrimSpace(text))
	h := sha256.Sum256([]byte(normalised))
	return fmt.Sprintf("%x", h[:16]) // First 16 bytes = 32 hex chars
}

// ── Query & Filter ──────────────────────────────────────────────

// LibrarySearchParams holds all filter parameters for querying the template library.
type LibrarySearchParams struct {
	Query      string `json:"query"`       // Full-text search term
	Category   string `json:"category"`
	Difficulty int    `json:"difficulty"`
	Language   string `json:"language"`
	Tag        string `json:"tag"`
	Source     string `json:"source"`
	OrgId      int64  `json:"org_id"`
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
}

// LibrarySearchResult contains paginated search results.
type LibrarySearchResult struct {
	Templates  []DBLibraryTemplate `json:"templates"`
	Total      int64               `json:"total"`
	Page       int                 `json:"page"`
	PageSize   int                 `json:"page_size"`
	TotalPages int                 `json:"total_pages"`
}

// SearchLibraryTemplates performs a filtered, paginated search of the DB template library.
func SearchLibraryTemplates(params LibrarySearchParams) (*LibrarySearchResult, error) {
	if params.PageSize <= 0 {
		params.PageSize = 25
	}
	if params.Page <= 0 {
		params.Page = 1
	}

	query := db.Model(&DBLibraryTemplate{}).Where("is_published = ?", true)

	// Scope: global + org-specific
	if params.OrgId > 0 {
		query = query.Where("org_id = 0 OR org_id = ?", params.OrgId)
	} else {
		query = query.Where("org_id = 0")
	}

	if params.Category != "" {
		query = query.Where("category = ?", params.Category)
	}
	if params.Difficulty > 0 {
		query = query.Where("difficulty_level = ?", params.Difficulty)
	}
	if params.Language != "" {
		query = query.Where("language = ?", params.Language)
	}
	if params.Source != "" {
		query = query.Where("source = ?", params.Source)
	}
	if params.Tag != "" {
		// JSON contains for tags
		query = query.Where("tags LIKE ?", "%\""+params.Tag+"\"%")
	}
	if params.Query != "" {
		pattern := "%" + params.Query + "%"
		query = query.Where("(name LIKE ? OR description LIKE ? OR subject LIKE ? OR tags LIKE ?)",
			pattern, pattern, pattern, pattern)
	}

	var total int64
	query.Count(&total)

	offset := (params.Page - 1) * params.PageSize
	var templates []DBLibraryTemplate
	err := query.Order("usage_count DESC, name ASC").
		Offset(offset).Limit(params.PageSize).
		Find(&templates).Error

	totalPages := int(total) / params.PageSize
	if int(total)%params.PageSize > 0 {
		totalPages++
	}

	return &LibrarySearchResult{
		Templates:  templates,
		Total:      total,
		Page:       params.Page,
		PageSize:   params.PageSize,
		TotalPages: totalPages,
	}, err
}

// GetDBLibraryTemplate returns a single template by ID.
func GetDBLibraryTemplate(id int64) (DBLibraryTemplate, error) {
	var t DBLibraryTemplate
	err := db.Where("id = ?", id).First(&t).Error
	return t, err
}

// GetDBLibraryTemplateBySlug returns a template by slug.
func GetDBLibraryTemplateBySlug(slug string) (DBLibraryTemplate, error) {
	var t DBLibraryTemplate
	err := db.Where("slug = ?", slug).First(&t).Error
	return t, err
}

// ── CRUD ────────────────────────────────────────────────────────

// CreateDBLibraryTemplate adds a new template to the DB library.
func CreateDBLibraryTemplate(t *DBLibraryTemplate) error {
	if t.Name == "" {
		return fmt.Errorf("template name is required")
	}
	if t.Slug == "" {
		t.Slug = generateSlug(t.Name, t.Language)
	}
	t.SimilarityHash = computeSimilarityHash(t.Subject, t.Text)
	t.CreatedDate = time.Now().UTC()
	t.ModifiedDate = time.Now().UTC()

	// Check for near-duplicates
	var existing DBLibraryTemplate
	if err := db.Where("similarity_hash = ?", t.SimilarityHash).First(&existing).Error; err == nil {
		return fmt.Errorf("a similar template already exists: %q (slug: %s)", existing.Name, existing.Slug)
	}

	return db.Save(t).Error
}

// UpdateDBLibraryTemplate updates an existing template.
func UpdateDBLibraryTemplate(t *DBLibraryTemplate) error {
	t.SimilarityHash = computeSimilarityHash(t.Subject, t.Text)
	t.ModifiedDate = time.Now().UTC()
	return db.Save(t).Error
}

// DeleteDBLibraryTemplate removes a template.
func DeleteDBLibraryTemplate(id int64) error {
	return db.Where("id = ?", id).Delete(&DBLibraryTemplate{}).Error
}

// IncrementTemplateUsage bumps the usage count when a template is imported.
func IncrementTemplateUsage(slug string) {
	db.Model(&DBLibraryTemplate{}).Where("slug = ?", slug).
		UpdateColumn("usage_count", db.Raw("usage_count + 1"))
}

// ── Import / Export ─────────────────────────────────────────────

// TemplateImportPayload is the JSON format for bulk template import.
type TemplateImportPayload struct {
	Templates []DBLibraryTemplate `json:"templates"`
}

// BulkImportLibraryTemplates imports multiple templates, skipping duplicates.
func BulkImportLibraryTemplates(templates []DBLibraryTemplate, orgId, userId int64) (imported, skipped int, err error) {
	for i := range templates {
		t := &templates[i]
		t.OrgId = orgId
		t.CreatedBy = userId
		if t.Source == "" {
			t.Source = "user"
		}

		if createErr := CreateDBLibraryTemplate(t); createErr != nil {
			skipped++
		} else {
			imported++
		}
	}
	return imported, skipped, nil
}

// ExportLibraryTemplates exports all templates for an org as a JSON payload.
func ExportLibraryTemplates(orgId int64) ([]DBLibraryTemplate, error) {
	var templates []DBLibraryTemplate
	err := db.Where("org_id = 0 OR org_id = ?", orgId).
		Order("category, difficulty_level, name").
		Find(&templates).Error
	return templates, err
}

// ── Seed Built-in Templates ─────────────────────────────────────

// SeedBuiltinTemplates populates the DB library with the hardcoded templates
// from TemplateLibrary. Should be called during migrations.
func SeedBuiltinTemplates() error {
	for _, lt := range TemplateLibrary {
		existing, err := GetDBLibraryTemplateBySlug(lt.Slug)
		if err == nil && existing.Id > 0 {
			continue // Already exists
		}

		dbt := &DBLibraryTemplate{
			Slug:            lt.Slug,
			Name:            lt.Name,
			Category:        lt.Category,
			DifficultyLevel: lt.DifficultyLevel,
			Description:     lt.Description,
			Subject:         lt.Subject,
			Text:            lt.Text,
			HTML:            lt.HTML,
			EnvelopeSender:  lt.EnvelopeSender,
			Language:        lt.Language,
			TargetRole:      lt.TargetRole,
			Source:          "builtin",
			OrgId:           0, // global
			IsPublished:     true,
		}
		dbt.SetTags([]string{lt.Category, lt.TargetRole})
		if err := CreateDBLibraryTemplate(dbt); err != nil {
			continue
		}
	}
	return nil
}

// ── Library Stats ───────────────────────────────────────────────

// DBTemplateLibraryStats returns stats about the DB-backed library.
type DBTemplateLibraryStats struct {
	TotalTemplates int            `json:"total_templates"`
	Categories     int            `json:"categories"`
	Languages      int            `json:"languages"`
	ByCategory     map[string]int `json:"by_category"`
	ByDifficulty   map[int]int    `json:"by_difficulty"`
	ByLanguage     map[string]int `json:"by_language"`
	BySource       map[string]int `json:"by_source"`
}

// GetDBTemplateLibraryStats returns stats for the DB template library.
func GetDBTemplateLibraryStats(orgId int64) DBTemplateLibraryStats {
	stats := DBTemplateLibraryStats{
		ByCategory:   make(map[string]int),
		ByDifficulty: make(map[int]int),
		ByLanguage:   make(map[string]int),
		BySource:     make(map[string]int),
	}

	type row struct {
		Category        string
		DifficultyLevel int
		Language        string
		Source          string
	}
	var rows []row
	db.Model(&DBLibraryTemplate{}).
		Where("is_published = ? AND (org_id = 0 OR org_id = ?)", true, orgId).
		Select("category, difficulty_level, language, source").
		Scan(&rows)

	cats := map[string]bool{}
	langs := map[string]bool{}
	for _, r := range rows {
		stats.ByCategory[r.Category]++
		stats.ByDifficulty[r.DifficultyLevel]++
		stats.ByLanguage[r.Language]++
		stats.BySource[r.Source]++
		cats[r.Category] = true
		langs[r.Language] = true
	}
	stats.TotalTemplates = len(rows)
	stats.Categories = len(cats)
	stats.Languages = len(langs)
	return stats
}

// GetDBLibraryCategories returns all distinct categories.
func GetDBLibraryCategories(orgId int64) []string {
	type row struct {
		Category string
	}
	var rows []row
	db.Model(&DBLibraryTemplate{}).
		Where("is_published = ? AND (org_id = 0 OR org_id = ?)", true, orgId).
		Select("DISTINCT category").
		Scan(&rows)

	cats := make([]string, 0, len(rows))
	for _, r := range rows {
		if r.Category != "" {
			cats = append(cats, r.Category)
		}
	}
	return cats
}

// ── Helpers ─────────────────────────────────────────────────────

func generateSlug(name, language string) string {
	slug := strings.ToLower(name)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "/", "-")
	slug = strings.ReplaceAll(slug, "&", "and")
	// Remove non-alphanumeric chars except hyphens
	var clean strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			clean.WriteRune(r)
		}
	}
	result := clean.String()
	if language != "" && language != "en" {
		result = result + "-" + language
	}
	return result
}

// ── New Template Categories (Scale-Up) ──────────────────────────
// These are the new categories added to reach the 10x target.

const (
	CategorySupplyChainPhishing = "Supply Chain"
	CategoryLegalCompliance     = "Legal / Compliance"
	CategoryCloudSaaS           = "Cloud / SaaS"
	CategoryTravelExpense       = "Travel / Expense"
	CategorySeasonal            = "Seasonal / Timely"
	CategoryExecutiveWhaling    = "Executive Whaling"
	CategoryVoicemailPhishing   = "Voicemail / Teams"
)

// ExpandedCategoryList includes all original + new categories.
var ExpandedCategoryList = []string{
	CategoryCredentialHarvesting,
	CategoryBEC,
	CategoryDeliveryNotification,
	CategoryITHelpdesk,
	CategoryHRPayroll,
	CategorySocialEngineering,
	CategoryQRCodePhishing,
	CategorySMSPhishing,
	// New categories for scale-up
	CategorySupplyChainPhishing,
	CategoryLegalCompliance,
	CategoryCloudSaaS,
	CategoryTravelExpense,
	CategorySeasonal,
	CategoryExecutiveWhaling,
	CategoryVoicemailPhishing,
}
