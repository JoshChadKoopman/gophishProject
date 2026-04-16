package models

import (
	"crypto/sha256"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"sort"
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
	Source          string    `json:"source"`                // "builtin", "user", "ai_generated", "community"
	OrgId           int64     `json:"org_id"`                // 0 = global, >0 = org-specific
	CreatedBy       int64     `json:"created_by"`
	IsPublished     bool      `json:"is_published" gorm:"default:true"`
	UsageCount      int64     `json:"usage_count"`    // How many times imported into a campaign
	AvgClickRate    float64   `json:"avg_click_rate"` // Average click rate when used
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
	Query      string `json:"query"` // Full-text search term
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

// ── Community Marketplace ───────────────────────────────────────
// Allows orgs to submit templates back to the global library (opt-in).
// Admin approval workflow before publication.

const (
	CommunityStatusDraft    = "draft"
	CommunityStatusPending  = "pending_review"
	CommunityStatusApproved = "approved"
	CommunityStatusRejected = "rejected"
)

// CommunitySubmission tracks a template submitted to the global marketplace.
type CommunitySubmission struct {
	Id           int64     `json:"id" gorm:"primary_key"`
	TemplateId   int64     `json:"template_id"` // FK to library_templates
	OrgId        int64     `json:"org_id"`
	SubmittedBy  int64     `json:"submitted_by"`
	Status       string    `json:"status" gorm:"default:'pending_review'"`
	ReviewedBy   int64     `json:"reviewed_by"`
	ReviewNotes  string    `json:"review_notes" gorm:"type:text"`
	AnonymizeOrg bool      `json:"anonymize_org" gorm:"default:true"`
	CreatedDate  time.Time `json:"created_date"`
	ReviewedDate time.Time `json:"reviewed_date"`
}

func (CommunitySubmission) TableName() string { return "community_submissions" }

// SubmitToCommunity creates a community submission for an existing template.
func SubmitToCommunity(templateId, orgId, userId int64, anonymize bool) (*CommunitySubmission, error) {
	t, err := GetDBLibraryTemplate(templateId)
	if err != nil {
		return nil, fmt.Errorf("template not found")
	}
	if t.OrgId != orgId {
		return nil, fmt.Errorf("template does not belong to your organization")
	}
	// Check for existing pending submission
	var existing CommunitySubmission
	if err := db.Where("template_id = ? AND status IN (?)", templateId,
		[]string{CommunityStatusPending, CommunityStatusApproved}).First(&existing).Error; err == nil {
		return nil, fmt.Errorf("template already submitted or approved")
	}

	sub := &CommunitySubmission{
		TemplateId:   templateId,
		OrgId:        orgId,
		SubmittedBy:  userId,
		Status:       CommunityStatusPending,
		AnonymizeOrg: anonymize,
		CreatedDate:  time.Now().UTC(),
	}
	return sub, db.Save(sub).Error
}

// ReviewCommunitySubmission approves or rejects a community submission.
func ReviewCommunitySubmission(submissionId int64, approve bool, reviewerId int64, notes string) error {
	var sub CommunitySubmission
	if err := db.Where("id = ?", submissionId).First(&sub).Error; err != nil {
		return fmt.Errorf("submission not found")
	}
	if sub.Status != CommunityStatusPending {
		return fmt.Errorf("submission is not pending review")
	}

	sub.ReviewedBy = reviewerId
	sub.ReviewNotes = notes
	sub.ReviewedDate = time.Now().UTC()

	if approve {
		sub.Status = CommunityStatusApproved
		// Publish a copy as a global template
		original, err := GetDBLibraryTemplate(sub.TemplateId)
		if err == nil {
			global := original
			global.Id = 0
			global.OrgId = 0 // global
			global.Source = "community"
			global.Slug = original.Slug + "-community"
			global.IsPublished = true
			if sub.AnonymizeOrg {
				global.CreatedBy = 0
			}
			CreateDBLibraryTemplate(&global)
		}
	} else {
		sub.Status = CommunityStatusRejected
	}

	return db.Save(&sub).Error
}

// GetPendingCommunitySubmissions lists all submissions awaiting review.
func GetPendingCommunitySubmissions() ([]CommunitySubmission, error) {
	var subs []CommunitySubmission
	err := db.Where("status = ?", CommunityStatusPending).Order("created_date ASC").Find(&subs).Error
	return subs, err
}

// GetCommunitySubmissions lists submissions for an org.
func GetCommunitySubmissions(orgId int64) ([]CommunitySubmission, error) {
	var subs []CommunitySubmission
	err := db.Where("org_id = ?", orgId).Order("created_date DESC").Find(&subs).Error
	return subs, err
}

// ── Multilingual Bulk Generation ────────────────────────────────
// Generates translations of existing templates for NL, DE, FR, ES.

// MultilingualLanguages are the target languages for bulk translation.
var MultilingualLanguages = []string{"nl", "de", "fr", "es"}

// MultilingualVariant represents a translation request for a template.
type MultilingualVariant struct {
	OriginalSlug string `json:"original_slug"`
	Language     string `json:"language"`
	Name         string `json:"name"`
	Subject      string `json:"subject"`
	Text         string `json:"text"`
	HTML         string `json:"html"`
	Description  string `json:"description"`
}

// GenerateMultilingualSeeds creates DB template stubs for all 18 × 4 language variants.
// The actual translation content should be populated by the AI translation engine
// and then human-reviewed. This creates the skeleton records.
func GenerateMultilingualSeeds() (created, skipped int) {
	for _, lt := range TemplateLibrary {
		for _, lang := range MultilingualLanguages {
			slug := lt.Slug + "-" + lang
			if _, err := GetDBLibraryTemplateBySlug(slug); err == nil {
				skipped++
				continue // Already exists
			}

			dbt := &DBLibraryTemplate{
				Slug:            slug,
				Name:            lt.Name + " (" + strings.ToUpper(lang) + ")",
				Category:        lt.Category,
				DifficultyLevel: lt.DifficultyLevel,
				Description:     lt.Description + " [" + strings.ToUpper(lang) + " translation — needs review]",
				Subject:         lt.Subject, // Placeholder: to be translated
				Text:            lt.Text,    // Placeholder: to be translated
				HTML:            lt.HTML,    // Placeholder: to be translated
				EnvelopeSender:  lt.EnvelopeSender,
				Language:        lang,
				TargetRole:      lt.TargetRole,
				Source:          "builtin",
				OrgId:           0,
				IsPublished:     false, // Not published until translated & reviewed
			}
			dbt.SetTags([]string{lt.Category, lt.TargetRole, "needs-translation", lang})
			if err := CreateDBLibraryTemplate(dbt); err != nil {
				skipped++
			} else {
				created++
			}
		}
	}
	return
}

// ── New Category Template Seeds ─────────────────────────────────
// Templates for the expanded categories (Supply Chain, Legal, Cloud/SaaS, etc.)

// NewCategoryTemplates contains pre-built templates for the expanded categories.
var NewCategoryTemplates = []LibraryTemplate{
	// ─── SUPPLY CHAIN ────────────────────────────────────────────
	{
		Slug:            "vendor-credential-reset",
		Name:            "Vendor Portal Credential Reset",
		Category:        "Supply Chain",
		DifficultyLevel: 2,
		Description:     "Spoofed vendor portal password reset targeting procurement staff.",
		Subject:         "Action Required: Reset Your Vendor Portal Password",
		Language:        "en",
		TargetRole:      "procurement",
		Text: `Dear {{.FirstName}},

Your credentials for the {{.OrgName}} Vendor Management Portal have been flagged for a mandatory security reset as part of our annual compliance review.

Please reset your password within 48 hours: {{.URL}}

Failure to comply will result in temporary suspension of your vendor access.

Regards,
Vendor Management Team`,
		HTML: `<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto">
<div style="background:#1a237e;padding:20px;text-align:center"><h2 style="color:#fff;margin:0">Vendor Management Portal</h2></div>
<div style="padding:30px;background:#fff;border:1px solid #e0e0e0">
<p>Dear {{.FirstName}},</p>
<p>Your credentials for the <strong>{{.OrgName}} Vendor Management Portal</strong> have been flagged for a mandatory security reset as part of our annual compliance review.</p>
<p style="text-align:center;margin:30px 0"><a href="{{.URL}}" style="background:#1a237e;color:#fff;padding:12px 30px;text-decoration:none;border-radius:4px">Reset Password</a></p>
<p style="color:#888;font-size:12px">Failure to comply within 48 hours will result in temporary suspension of your vendor access.</p>
</div></div>{{.Tracker}}`,
	},
	{
		Slug:            "supplier-payment-redirect",
		Name:            "Supplier Payment Details Update",
		Category:        "Supply Chain",
		DifficultyLevel: 3,
		Description:     "Fraudulent supplier requesting updated payment details — BEC supply chain variant.",
		Subject:         "Updated Banking Details – Urgent Action Required",
		Language:        "en",
		TargetRole:      "finance",
		Text: `Dear {{.FirstName}},

I hope this message finds you well. Due to a recent change in our banking provider, we need to update our payment details on file with {{.OrgName}}.

Please review and confirm the updated details at your earliest convenience: {{.URL}}

Please treat this as urgent — our next invoice is due shortly.

Kind regards,
James Miller
Accounts Receivable, Trusted Supplies Ltd`,
		HTML: `<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto">
<div style="padding:30px;background:#fff">
<p>Dear {{.FirstName}},</p>
<p>I hope this message finds you well. Due to a recent change in our banking provider, we need to update our payment details on file with <strong>{{.OrgName}}</strong>.</p>
<p>Please review and confirm the updated details at your earliest convenience:</p>
<p style="text-align:center;margin:25px 0"><a href="{{.URL}}" style="background:#2196f3;color:#fff;padding:12px 30px;text-decoration:none;border-radius:4px">Review Payment Details</a></p>
<p>Please treat this as urgent — our next invoice is due shortly.</p>
<p>Kind regards,<br><strong>James Miller</strong><br>Accounts Receivable, Trusted Supplies Ltd</p>
</div></div>{{.Tracker}}`,
	},
	// ─── LEGAL / COMPLIANCE ──────────────────────────────────────
	{
		Slug:            "gdpr-audit-request",
		Name:            "GDPR Data Audit Request",
		Category:        "Legal / Compliance",
		DifficultyLevel: 2,
		Description:     "Fake GDPR audit notification from the Data Protection Officer.",
		Subject:         "GDPR Compliance Audit – Your Response Required by {{.Date}}",
		Language:        "en",
		TargetRole:      "all",
		Text: `Dear {{.FirstName}},

As part of our ongoing GDPR compliance program, your department has been selected for a data handling audit. Please complete the self-assessment form by {{.Date}}.

Access the audit form: {{.URL}}

Non-completion may result in escalation to the Data Protection Officer.

Regards,
Compliance Team, {{.OrgName}}`,
		HTML: `<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto">
<div style="background:#b71c1c;padding:20px;text-align:center"><h2 style="color:#fff;margin:0">⚖️ GDPR Compliance Audit</h2></div>
<div style="padding:30px;background:#fff;border:1px solid #e0e0e0">
<p>Dear {{.FirstName}},</p>
<p>As part of our ongoing GDPR compliance program, your department has been selected for a <strong>data handling audit</strong>.</p>
<p>Please complete the self-assessment form by <strong>{{.Date}}</strong>.</p>
<p style="text-align:center;margin:30px 0"><a href="{{.URL}}" style="background:#b71c1c;color:#fff;padding:12px 30px;text-decoration:none;border-radius:4px">Complete Audit Form</a></p>
<p style="color:#888;font-size:12px">Non-completion may result in escalation to the Data Protection Officer.</p>
</div></div>{{.Tracker}}`,
	},
	{
		Slug:            "legal-subpoena-notice",
		Name:            "Legal Subpoena Document Review",
		Category:        "Legal / Compliance",
		DifficultyLevel: 3,
		Description:     "Fake legal notice requiring urgent document review — high urgency social engineering.",
		Subject:         "CONFIDENTIAL: Legal Subpoena – Immediate Action Required",
		Language:        "en",
		TargetRole:      "management",
		Text: `Dear {{.FirstName}},

Our legal department has received a subpoena that may require documents from your division. This is a time-sensitive matter and we need your immediate cooperation.

Please review the subpoena details and identify relevant documents: {{.URL}}

This communication is privileged and confidential. Do not forward this email.

Regards,
Legal Department, {{.OrgName}}`,
		HTML: `<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto">
<div style="background:#263238;padding:20px;text-align:center"><h2 style="color:#fff;margin:0">🔒 CONFIDENTIAL – Legal Notice</h2></div>
<div style="padding:30px;background:#fff;border:2px solid #b71c1c">
<p>Dear {{.FirstName}},</p>
<p>Our legal department has received a <strong>subpoena</strong> that may require documents from your division. This is a <strong>time-sensitive matter</strong> and we need your immediate cooperation.</p>
<p style="text-align:center;margin:30px 0"><a href="{{.URL}}" style="background:#263238;color:#fff;padding:12px 30px;text-decoration:none;border-radius:4px">Review Subpoena Details</a></p>
<p style="color:#b71c1c;font-size:12px;font-style:italic">This communication is privileged and confidential. Do not forward this email.</p>
</div></div>{{.Tracker}}`,
	},
	// ─── CLOUD / SAAS ────────────────────────────────────────────
	{
		Slug:            "aws-security-alert",
		Name:            "AWS Security Alert – Unauthorized Access",
		Category:        "Cloud / SaaS",
		DifficultyLevel: 2,
		Description:     "Spoofed AWS alert about unauthorized API access from an unknown region.",
		Subject:         "[AWS] Security Alert: Unauthorized API Access Detected",
		Language:        "en",
		TargetRole:      "technical",
		Text: `Amazon Web Services

We detected unauthorized API calls from an unrecognized IP address on your AWS account.

Details:
  Region: ap-southeast-1 (Singapore)
  Service: IAM, S3
  Time: {{.Date}}

If this wasn't you, secure your account immediately: {{.URL}}

Amazon Web Services`,
		HTML: `<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto;background:#fff">
<div style="background:#232f3e;padding:15px 30px"><img src="https://a0.awsstatic.com/libra-css/images/logos/aws_logo_smile_1200x630.png" alt="AWS" style="height:30px" onerror="this.outerHTML='<span style=color:#fff;font-size:20px>Amazon Web Services</span>'"></div>
<div style="padding:30px;border:1px solid #e0e0e0">
<h3 style="color:#d13212">⚠ Security Alert: Unauthorized API Access</h3>
<p>We detected unauthorized API calls from an unrecognized IP address on your AWS account.</p>
<table style="width:100%;margin:20px 0;border-collapse:collapse"><tr><td style="padding:8px;color:#666">Region:</td><td style="padding:8px"><strong>ap-southeast-1 (Singapore)</strong></td></tr><tr><td style="padding:8px;color:#666">Services:</td><td style="padding:8px">IAM, S3</td></tr></table>
<p><a href="{{.URL}}" style="background:#ff9900;color:#111;padding:10px 24px;text-decoration:none;border-radius:3px;display:inline-block;font-weight:bold">Review Account Activity</a></p>
</div></div>{{.Tracker}}`,
	},
	{
		Slug:            "slack-workspace-invite",
		Name:            "Slack Workspace Shared Channel Invite",
		Category:        "Cloud / SaaS",
		DifficultyLevel: 2,
		Description:     "Spoofed Slack notification about a shared channel invite from a partner org.",
		Subject:         "You've been invited to a shared channel in Slack",
		Language:        "en",
		TargetRole:      "all",
		Text: `Slack

Hi {{.FirstName}},

You've been invited to join the shared channel #project-acme in the {{.OrgName}} workspace by an external partner.

Accept the invitation to start collaborating: {{.URL}}

– Slack`,
		HTML: `<div style="font-family:Lato,Arial,sans-serif;max-width:600px;margin:0 auto;background:#fff">
<div style="background:#4a154b;padding:20px;text-align:center"><h2 style="color:#fff;margin:0">slack</h2></div>
<div style="padding:30px">
<p>Hi {{.FirstName}},</p>
<p>You've been invited to join the shared channel <strong>#project-acme</strong> in the <strong>{{.OrgName}}</strong> workspace by an external partner.</p>
<p style="text-align:center;margin:30px 0"><a href="{{.URL}}" style="background:#4a154b;color:#fff;padding:12px 30px;text-decoration:none;border-radius:4px">Accept Invitation</a></p>
</div></div>{{.Tracker}}`,
	},
	// ─── TRAVEL / EXPENSE ────────────────────────────────────────
	{
		Slug:            "expense-report-rejected",
		Name:            "Expense Report Rejected – Resubmit Required",
		Category:        "Travel / Expense",
		DifficultyLevel: 1,
		Description:     "Fake expense management system notification about a rejected expense report.",
		Subject:         "Expense Report #{{.RId}} Rejected – Action Required",
		Language:        "en",
		TargetRole:      "all",
		Text: `Dear {{.FirstName}},

Your recent expense report has been rejected by your approver due to missing receipts. Please review and resubmit within 5 business days.

Review your expense report: {{.URL}}

If you have questions, contact the Finance team.

{{.OrgName}} Expense Management`,
		HTML: `<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto">
<div style="background:#f57c00;padding:20px;text-align:center"><h2 style="color:#fff;margin:0">💳 Expense Report Rejected</h2></div>
<div style="padding:30px;background:#fff;border:1px solid #e0e0e0">
<p>Dear {{.FirstName}},</p>
<p>Your recent expense report has been <strong style="color:#d32f2f">rejected</strong> by your approver due to missing receipts.</p>
<p>Please review and resubmit within 5 business days.</p>
<p style="text-align:center;margin:30px 0"><a href="{{.URL}}" style="background:#f57c00;color:#fff;padding:12px 30px;text-decoration:none;border-radius:4px">Review Expense Report</a></p>
</div></div>{{.Tracker}}`,
	},
	{
		Slug:            "travel-booking-confirmation",
		Name:            "Travel Booking Confirmation – Verify Itinerary",
		Category:        "Travel / Expense",
		DifficultyLevel: 2,
		Description:     "Fake corporate travel booking confirmation requiring itinerary verification.",
		Subject:         "Your Travel Booking Confirmation – Verify Details",
		Language:        "en",
		TargetRole:      "all",
		Text: `Dear {{.FirstName}},

A travel booking has been made on your behalf for an upcoming business trip. Please verify the itinerary details and confirm your attendance.

View booking details: {{.URL}}

If you did not request this booking, please contact the Travel desk immediately.

{{.OrgName}} Corporate Travel`,
		HTML: `<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto">
<div style="background:#0277bd;padding:20px;text-align:center"><h2 style="color:#fff;margin:0">✈️ Travel Booking Confirmation</h2></div>
<div style="padding:30px;background:#fff;border:1px solid #e0e0e0">
<p>Dear {{.FirstName}},</p>
<p>A travel booking has been made on your behalf for an upcoming business trip. Please verify the itinerary details and confirm your attendance.</p>
<p style="text-align:center;margin:30px 0"><a href="{{.URL}}" style="background:#0277bd;color:#fff;padding:12px 30px;text-decoration:none;border-radius:4px">View Booking Details</a></p>
<p style="color:#888;font-size:12px">If you did not request this booking, please contact the Travel desk immediately.</p>
</div></div>{{.Tracker}}`,
	},
	// ─── SEASONAL ────────────────────────────────────────────────
	{
		Slug:            "tax-season-refund",
		Name:            "Tax Refund Notification",
		Category:        "Seasonal / Timely",
		DifficultyLevel: 2,
		Description:     "Spoofed tax authority notification about a pending refund — seasonal Q1 template.",
		Subject:         "Tax Refund Pending – Verify Your Details",
		Language:        "en",
		TargetRole:      "all",
		Text: `Dear {{.FirstName}},

After the latest assessment of your tax records, we have determined that you are eligible for a tax refund of €1,247.50.

To process your refund, please verify your details: {{.URL}}

Please submit within 10 business days to avoid delays.

Tax Administration`,
		HTML: `<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto">
<div style="background:#1b5e20;padding:20px;text-align:center"><h2 style="color:#fff;margin:0">🏛️ Tax Administration</h2></div>
<div style="padding:30px;background:#fff;border:1px solid #e0e0e0">
<p>Dear {{.FirstName}},</p>
<p>After the latest assessment of your tax records, we have determined that you are eligible for a tax refund of <strong>€1,247.50</strong>.</p>
<p>To process your refund, please verify your details:</p>
<p style="text-align:center;margin:30px 0"><a href="{{.URL}}" style="background:#1b5e20;color:#fff;padding:12px 30px;text-decoration:none;border-radius:4px">Verify & Claim Refund</a></p>
<p style="color:#888;font-size:12px">Please submit within 10 business days to avoid delays.</p>
</div></div>{{.Tracker}}`,
	},
	{
		Slug:            "black-friday-deal",
		Name:            "Black Friday Employee Perks Portal",
		Category:        "Seasonal / Timely",
		DifficultyLevel: 1,
		Description:     "Fake internal employee perks portal with exclusive Black Friday deals — seasonal Q4.",
		Subject:         "🎉 Exclusive Black Friday Deals for {{.OrgName}} Employees!",
		Language:        "en",
		TargetRole:      "all",
		Text: `Hi {{.FirstName}},

Great news! As a valued {{.OrgName}} employee, you have exclusive early access to our Black Friday Employee Perks Portal.

Save up to 70% on top brands. Browse deals now: {{.URL}}

Offer ends Friday at midnight. Don't miss out!

{{.OrgName}} Employee Benefits`,
		HTML: `<div style="font-family:Arial,sans-serif;max-width:600px;margin:0 auto">
<div style="background:#212121;padding:20px;text-align:center"><h2 style="color:#ffab00;margin:0">🎉 BLACK FRIDAY — Employee Perks</h2></div>
<div style="padding:30px;background:#fff;border:1px solid #e0e0e0">
<p>Hi {{.FirstName}},</p>
<p>Great news! As a valued <strong>{{.OrgName}}</strong> employee, you have exclusive early access to our Black Friday Employee Perks Portal.</p>
<p style="font-size:24px;text-align:center;color:#d32f2f;font-weight:bold">Save up to 70%!</p>
<p style="text-align:center;margin:30px 0"><a href="{{.URL}}" style="background:#ffab00;color:#212121;padding:14px 35px;text-decoration:none;border-radius:4px;font-weight:bold;font-size:16px">Browse Deals Now</a></p>
<p style="color:#888;font-size:12px;text-align:center">Offer ends Friday at midnight.</p>
</div></div>{{.Tracker}}`,
	},
	// ─── EXECUTIVE WHALING ───────────────────────────────────────
	{
		Slug:            "board-meeting-agenda",
		Name:            "Board Meeting Agenda – Confidential Review",
		Category:        "Executive Whaling",
		DifficultyLevel: 3,
		Description:     "Targeted whaling email pretending to share confidential board meeting documents.",
		Subject:         "Board Meeting Agenda – Confidential – {{.Date}}",
		Language:        "en",
		TargetRole:      "executive",
		Text: `Dear {{.FirstName}},

Please find attached the agenda and pre-read materials for the upcoming board meeting. The materials contain sensitive financial projections and M&A discussions.

Access the secure board portal: {{.URL}}

Please review before Thursday's session. This material is strictly confidential.

Kind regards,
Office of the CEO`,
		HTML: `<div style="font-family:Georgia,serif;max-width:600px;margin:0 auto">
<div style="border-bottom:3px solid #1a237e;padding:20px"><h2 style="margin:0;color:#1a237e">Office of the CEO</h2></div>
<div style="padding:30px">
<p>Dear {{.FirstName}},</p>
<p>Please find the agenda and pre-read materials for the upcoming board meeting. The materials contain <strong>sensitive financial projections</strong> and M&A discussions.</p>
<p style="text-align:center;margin:30px 0"><a href="{{.URL}}" style="background:#1a237e;color:#fff;padding:12px 30px;text-decoration:none;border-radius:4px">Access Secure Board Portal</a></p>
<p style="color:#888;font-size:12px;font-style:italic">This material is strictly confidential and intended for board members only.</p>
</div></div>{{.Tracker}}`,
	},
	// ─── VOICEMAIL / TEAMS ───────────────────────────────────────
	{
		Slug:            "teams-missed-call",
		Name:            "Microsoft Teams Missed Call & Voicemail",
		Category:        "Voicemail / Teams",
		DifficultyLevel: 2,
		Description:     "Spoofed Microsoft Teams missed call notification with a voicemail link.",
		Subject:         "Missed call from Unknown Caller – Voicemail Available",
		Language:        "en",
		TargetRole:      "all",
		Text: `Microsoft Teams

You missed a call from Unknown Caller at {{.Date}}.

A voicemail was left (0:47).

Listen to voicemail: {{.URL}}

Microsoft Teams notifications`,
		HTML: `<div style="font-family:Segoe UI,Arial,sans-serif;max-width:600px;margin:0 auto;background:#fff">
<div style="background:#464775;padding:15px 30px"><span style="color:#fff;font-size:18px">Microsoft Teams</span></div>
<div style="padding:30px">
<p>You missed a call</p>
<div style="background:#f5f5f5;padding:20px;border-radius:8px;margin:20px 0">
<p style="margin:0"><strong>Unknown Caller</strong></p>
<p style="margin:5px 0;color:#666">{{.Date}} • Duration: 0:47</p>
<p style="margin:10px 0 0"><a href="{{.URL}}" style="color:#6264a7;text-decoration:none">▶ Play voicemail</a></p>
</div>
<p><a href="{{.URL}}" style="background:#6264a7;color:#fff;padding:10px 24px;text-decoration:none;border-radius:4px;display:inline-block">Open in Teams</a></p>
</div></div>{{.Tracker}}`,
	},
}

// SeedNewCategoryTemplates inserts the expanded category templates into the DB.
func SeedNewCategoryTemplates() (created, skipped int) {
	for _, lt := range NewCategoryTemplates {
		if _, err := GetDBLibraryTemplateBySlug(lt.Slug); err == nil {
			skipped++
			continue
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
			OrgId:           0,
			IsPublished:     true,
		}
		dbt.SetTags([]string{lt.Category, lt.TargetRole})
		if err := CreateDBLibraryTemplate(dbt); err != nil {
			skipped++
		} else {
			created++
		}
	}
	return
}

// SeedAllTemplates seeds built-in + new categories + multilingual skeletons.
func SeedAllTemplates() (total int, err error) {
	if err = SeedBuiltinTemplates(); err != nil {
		return 0, err
	}
	c1, _ := SeedNewCategoryTemplates()
	c2, _ := GenerateMultilingualSeeds()
	return c1 + c2 + len(TemplateLibrary), nil
}

// ── Full-Text Search ────────────────────────────────────────────

// FTSSearchLibraryTemplates performs a fast full-text search on the template library.
// Uses indexed LIKE queries across name, description, subject, and tags columns.
func FTSSearchLibraryTemplates(query string, orgId int64, page, pageSize int) (*LibrarySearchResult, error) {
	if pageSize <= 0 {
		pageSize = 25
	}
	if page <= 0 {
		page = 1
	}

	pattern := "%" + query + "%"
	q := db.Model(&DBLibraryTemplate{}).Where("is_published = ? AND (org_id = 0 OR org_id = ?)", true, orgId).
		Where("(name LIKE ? OR description LIKE ? OR subject LIKE ? OR tags LIKE ?)", pattern, pattern, pattern, pattern)

	var total int64
	q.Count(&total)

	var templates []DBLibraryTemplate
	offset := (page - 1) * pageSize
	q.Order("usage_count DESC, name ASC").Offset(offset).Limit(pageSize).Find(&templates)

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}
	return &LibrarySearchResult{
		Templates:  templates,
		Total:      total,
		Page:       page,
		PageSize:   pageSize,
		TotalPages: totalPages,
	}, nil
}

// ── TF-IDF Similarity Scoring ───────────────────────────────────

// tfidfTokenize splits text into lowercase word tokens.
func tfidfTokenize(text string) []string {
	text = strings.ToLower(text)
	var tokens []string
	var word strings.Builder
	for _, r := range text {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			word.WriteRune(r)
		} else {
			if word.Len() > 2 {
				tokens = append(tokens, word.String())
			}
			word.Reset()
		}
	}
	if word.Len() > 2 {
		tokens = append(tokens, word.String())
	}
	return tokens
}

// tfidfVector computes a term frequency vector from tokens.
func tfidfVector(tokens []string) map[string]float64 {
	tf := make(map[string]float64)
	for _, t := range tokens {
		tf[t]++
	}
	total := float64(len(tokens))
	if total == 0 {
		return tf
	}
	for k := range tf {
		tf[k] /= total
	}
	return tf
}

// cosineSimilarity computes cosine similarity between two TF vectors.
func cosineSimilarity(a, b map[string]float64) float64 {
	var dot, magA, magB float64
	for k, v := range a {
		if bv, ok := b[k]; ok {
			dot += v * bv
		}
		magA += v * v
	}
	for _, v := range b {
		magB += v * v
	}
	if magA == 0 || magB == 0 {
		return 0
	}
	return dot / (math.Sqrt(magA) * math.Sqrt(magB))
}

// SimilarityResult represents a template and its similarity score.
type SimilarityResult struct {
	Template DBLibraryTemplate `json:"template"`
	Score    float64           `json:"score"`
}

// FindSimilarTemplates computes TF-IDF cosine similarity of the given text
// against all published templates and returns those above the threshold.
func FindSimilarTemplates(subject, text string, threshold float64, maxResults int) ([]SimilarityResult, error) {
	if threshold <= 0 {
		threshold = 0.3
	}
	if maxResults <= 0 {
		maxResults = 10
	}

	inputTokens := tfidfTokenize(subject + " " + text)
	inputVec := tfidfVector(inputTokens)

	var templates []DBLibraryTemplate
	if err := db.Where("is_published = ?", true).Find(&templates).Error; err != nil {
		return nil, err
	}

	var results []SimilarityResult
	for _, t := range templates {
		tTokens := tfidfTokenize(t.Subject + " " + t.Text)
		tVec := tfidfVector(tTokens)
		score := cosineSimilarity(inputVec, tVec)
		if score >= threshold {
			results = append(results, SimilarityResult{Template: t, Score: score})
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	if len(results) > maxResults {
		results = results[:maxResults]
	}
	return results, nil
}

// ── Human Review Workflow ───────────────────────────────────────

const (
	ReviewStatusPending  = "pending"
	ReviewStatusApproved = "approved"
	ReviewStatusRejected = "rejected"
	ReviewStatusRevision = "needs_revision"

	ReviewTypeTranslation = "translation"
	ReviewTypeAIGenerated = "ai_generated"
	ReviewTypeCommunity   = "community"
)

// TemplateReview represents an admin review record for a template.
type TemplateReview struct {
	Id           int64     `json:"id" gorm:"primary_key"`
	TemplateId   int64     `json:"template_id"`
	ReviewerId   int64     `json:"reviewer_id"`
	Status       string    `json:"status" gorm:"default:'pending'"`
	ReviewType   string    `json:"review_type" gorm:"default:'translation'"`
	Notes        string    `json:"notes" gorm:"type:text"`
	QualityScore int       `json:"quality_score"`
	CreatedDate  time.Time `json:"created_date"`
	ReviewedDate time.Time `json:"reviewed_date"`
}

func (TemplateReview) TableName() string { return "template_reviews" }

// CreateTemplateReview creates a new review record.
func CreateTemplateReview(r *TemplateReview) error {
	r.CreatedDate = time.Now().UTC()
	return db.Save(r).Error
}

// GetPendingReviews returns all pending reviews, optionally filtered by type.
func GetPendingReviews(reviewType string) ([]TemplateReview, error) {
	var reviews []TemplateReview
	q := db.Where("status = ?", ReviewStatusPending)
	if reviewType != "" {
		q = q.Where("review_type = ?", reviewType)
	}
	err := q.Order("created_date ASC").Find(&reviews).Error
	return reviews, err
}

// GetPendingReviewsWithTemplates returns pending reviews with their associated template data.
func GetPendingReviewsWithTemplates(reviewType string) ([]map[string]interface{}, error) {
	reviews, err := GetPendingReviews(reviewType)
	if err != nil {
		return nil, err
	}
	var result []map[string]interface{}
	for _, r := range reviews {
		t, tErr := GetDBLibraryTemplate(r.TemplateId)
		entry := map[string]interface{}{
			"review":   r,
			"template": nil,
		}
		if tErr == nil {
			entry["template"] = t
		}
		result = append(result, entry)
	}
	return result, nil
}

// CompleteTemplateReview approves or rejects a review and optionally publishes the template.
func CompleteTemplateReview(reviewId int64, approved bool, reviewerId int64, notes string, qualityScore int) error {
	var review TemplateReview
	if err := db.Where("id = ?", reviewId).First(&review).Error; err != nil {
		return fmt.Errorf("review not found")
	}
	if review.Status != ReviewStatusPending {
		return fmt.Errorf("review is not pending")
	}

	review.ReviewerId = reviewerId
	review.Notes = notes
	review.QualityScore = qualityScore
	review.ReviewedDate = time.Now().UTC()

	if approved {
		review.Status = ReviewStatusApproved
		// Publish the template
		db.Model(&DBLibraryTemplate{}).Where("id = ?", review.TemplateId).
			Updates(map[string]interface{}{
				"is_published":  true,
				"modified_date": time.Now().UTC(),
			})
	} else {
		review.Status = ReviewStatusRejected
	}

	return db.Save(&review).Error
}

// RequestRevision marks a review as needing revision.
func RequestRevision(reviewId int64, reviewerId int64, notes string) error {
	var review TemplateReview
	if err := db.Where("id = ?", reviewId).First(&review).Error; err != nil {
		return fmt.Errorf("review not found")
	}
	review.Status = ReviewStatusRevision
	review.ReviewerId = reviewerId
	review.Notes = notes
	review.ReviewedDate = time.Now().UTC()
	return db.Save(&review).Error
}

// GetReviewsByTemplate returns all reviews for a template.
func GetReviewsByTemplate(templateId int64) ([]TemplateReview, error) {
	var reviews []TemplateReview
	err := db.Where("template_id = ?", templateId).Order("created_date DESC").Find(&reviews).Error
	return reviews, err
}

// GetReviewStats returns review statistics.
type ReviewStats struct {
	Pending       int64 `json:"pending"`
	Approved      int64 `json:"approved"`
	Rejected      int64 `json:"rejected"`
	NeedsRevision int64 `json:"needs_revision"`
}

func GetReviewStats() ReviewStats {
	var stats ReviewStats
	db.Model(&TemplateReview{}).Where("status = ?", ReviewStatusPending).Count(&stats.Pending)
	db.Model(&TemplateReview{}).Where("status = ?", ReviewStatusApproved).Count(&stats.Approved)
	db.Model(&TemplateReview{}).Where("status = ?", ReviewStatusRejected).Count(&stats.Rejected)
	db.Model(&TemplateReview{}).Where("status = ?", ReviewStatusRevision).Count(&stats.NeedsRevision)
	return stats
}

// ── Auto-create reviews for unpublished templates ───────────────

// CreateReviewsForUnreviewedTemplates creates review records for templates that
// are unpublished and don't have a pending review yet (e.g., newly generated translations).
func CreateReviewsForUnreviewedTemplates() (int, error) {
	var templates []DBLibraryTemplate
	if err := db.Where("is_published = ? AND source IN (?)", false,
		[]string{"builtin", "ai_generated"}).Find(&templates).Error; err != nil {
		return 0, err
	}

	created := 0
	for _, t := range templates {
		// Check if a pending review already exists
		var count int64
		db.Model(&TemplateReview{}).Where("template_id = ? AND status = ?", t.Id, ReviewStatusPending).Count(&count)
		if count > 0 {
			continue
		}

		reviewType := ReviewTypeTranslation
		if t.Source == "ai_generated" {
			reviewType = ReviewTypeAIGenerated
		}

		review := &TemplateReview{
			TemplateId: t.Id,
			Status:     ReviewStatusPending,
			ReviewType: reviewType,
		}
		if err := CreateTemplateReview(review); err == nil {
			created++
		}
	}
	return created, nil
}

// ── CSV Export ───────────────────────────────────────────────────

// ExportLibraryTemplatesCSV writes templates in CSV format.
func ExportLibraryTemplatesCSV(orgId int64) ([]byte, error) {
	templates, err := ExportLibraryTemplates(orgId)
	if err != nil {
		return nil, err
	}

	var buf strings.Builder
	w := csv.NewWriter(&buf)
	// Header
	w.Write([]string{
		"slug", "name", "category", "difficulty_level", "description",
		"subject", "text", "html", "envelope_sender", "language",
		"target_role", "tags", "source", "usage_count",
	})
	for _, t := range templates {
		w.Write([]string{
			t.Slug, t.Name, t.Category, fmt.Sprintf("%d", t.DifficultyLevel),
			t.Description, t.Subject, t.Text, t.HTML, t.EnvelopeSender,
			t.Language, t.TargetRole, t.Tags, t.Source,
			fmt.Sprintf("%d", t.UsageCount),
		})
	}
	w.Flush()
	return []byte(buf.String()), w.Error()
}

// ImportLibraryTemplatesCSV parses CSV data and imports templates.
func ImportLibraryTemplatesCSV(data []byte, orgId, userId int64) (imported, skipped int, err error) {
	r := csv.NewReader(strings.NewReader(string(data)))
	records, err := r.ReadAll()
	if err != nil {
		return 0, 0, fmt.Errorf("invalid CSV: %v", err)
	}
	if len(records) < 2 {
		return 0, 0, fmt.Errorf("CSV must have a header row and at least one data row")
	}

	// Find column indices from header
	header := records[0]
	cols := map[string]int{}
	for i, h := range header {
		cols[strings.TrimSpace(strings.ToLower(h))] = i
	}

	for _, row := range records[1:] {
		t := &DBLibraryTemplate{
			OrgId:     orgId,
			CreatedBy: userId,
			Source:    "user",
		}
		if idx, ok := cols["slug"]; ok && idx < len(row) {
			t.Slug = row[idx]
		}
		if idx, ok := cols["name"]; ok && idx < len(row) {
			t.Name = row[idx]
		}
		if idx, ok := cols["category"]; ok && idx < len(row) {
			t.Category = row[idx]
		}
		if idx, ok := cols["difficulty_level"]; ok && idx < len(row) {
			fmt.Sscanf(row[idx], "%d", &t.DifficultyLevel)
		}
		if idx, ok := cols["description"]; ok && idx < len(row) {
			t.Description = row[idx]
		}
		if idx, ok := cols["subject"]; ok && idx < len(row) {
			t.Subject = row[idx]
		}
		if idx, ok := cols["text"]; ok && idx < len(row) {
			t.Text = row[idx]
		}
		if idx, ok := cols["html"]; ok && idx < len(row) {
			t.HTML = row[idx]
		}
		if idx, ok := cols["envelope_sender"]; ok && idx < len(row) {
			t.EnvelopeSender = row[idx]
		}
		if idx, ok := cols["language"]; ok && idx < len(row) {
			t.Language = row[idx]
		}
		if idx, ok := cols["target_role"]; ok && idx < len(row) {
			t.TargetRole = row[idx]
		}
		if idx, ok := cols["tags"]; ok && idx < len(row) {
			t.Tags = row[idx]
		}

		if t.Name == "" {
			skipped++
			continue
		}
		if err := CreateDBLibraryTemplate(t); err != nil {
			skipped++
		} else {
			imported++
		}
	}
	return imported, skipped, nil
}

// ── Template Moderation (Admin Controls) ────────────────────────

// BulkPublishTemplates publishes multiple templates by IDs.
func BulkPublishTemplates(ids []int64) (int, error) {
	result := db.Model(&DBLibraryTemplate{}).Where("id IN (?)", ids).
		Updates(map[string]interface{}{
			"is_published":  true,
			"modified_date": time.Now().UTC(),
		})
	return int(result.RowsAffected), result.Error
}

// BulkUnpublishTemplates unpublishes multiple templates by IDs.
func BulkUnpublishTemplates(ids []int64) (int, error) {
	result := db.Model(&DBLibraryTemplate{}).Where("id IN (?)", ids).
		Updates(map[string]interface{}{
			"is_published":  false,
			"modified_date": time.Now().UTC(),
		})
	return int(result.RowsAffected), result.Error
}

// BulkDeleteTemplates deletes multiple templates by IDs.
func BulkDeleteTemplates(ids []int64) (int, error) {
	result := db.Where("id IN (?)", ids).Delete(&DBLibraryTemplate{})
	return int(result.RowsAffected), result.Error
}

// BulkTagTemplates adds tags to multiple templates.
func BulkTagTemplates(ids []int64, newTags []string) (int, error) {
	var templates []DBLibraryTemplate
	if err := db.Where("id IN (?)", ids).Find(&templates).Error; err != nil {
		return 0, err
	}
	updated := 0
	for _, t := range templates {
		existing := t.GetTags()
		merged := mergeStringSlices(existing, newTags)
		t.SetTags(merged)
		t.ModifiedDate = time.Now().UTC()
		if err := db.Save(&t).Error; err == nil {
			updated++
		}
	}
	return updated, nil
}

func mergeStringSlices(a, b []string) []string {
	seen := map[string]bool{}
	for _, s := range a {
		seen[s] = true
	}
	merged := append([]string{}, a...)
	for _, s := range b {
		if !seen[s] {
			merged = append(merged, s)
			seen[s] = true
		}
	}
	return merged
}
