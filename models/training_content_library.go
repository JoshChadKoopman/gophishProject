package models

import (
	"encoding/json"
	"time"

	log "github.com/gophish/gophish/logger"
)

// TrainingContentCategory groups built-in training content by topic.
const (
	ContentCategoryPhishing       = "phishing"
	ContentCategoryPasswords      = "passwords"
	ContentCategorySocialEng      = "social_engineering"
	ContentCategoryDataProtection = "data_protection"
	ContentCategoryMalware        = "malware"
	ContentCategoryPhysicalSec    = "physical_security"
	ContentCategoryMobileSec      = "mobile_security"
	ContentCategoryRemoteWork     = "remote_work"
	ContentCategoryCompliance     = "compliance"
	ContentCategoryIncident       = "incident_response"
	ContentCategoryCloudSec       = "cloud_security"
	ContentCategoryAISec          = "ai_security"
)

// TrainingContentDifficulty maps to academy tiers.
const (
	ContentDiffBronze   = 1 // Fundamentals
	ContentDiffSilver   = 2 // Intermediate
	ContentDiffGold     = 3 // Advanced
	ContentDiffPlatinum = 4 // Expert
)

// BuiltInTrainingContent represents a single micro-learning session from the
// Nivoxis content library. These sessions ship with the platform and can be
// seeded into any organization's academy with a single API call.
type BuiltInTrainingContent struct {
	Slug             string         `json:"slug"`
	Title            string         `json:"title"`
	Category         string         `json:"category"`
	DifficultyLevel  int            `json:"difficulty_level"` // 1-4, maps to tier
	Description      string         `json:"description"`
	EstimatedMinutes int            `json:"estimated_minutes"`
	Tags             []string       `json:"tags"`
	ComplianceMapped []string       `json:"compliance_mapped"` // e.g. ["NIS2","ISO27001"]
	Pages            []TrainingPage `json:"pages"`
	Quiz             *BuiltInQuiz   `json:"quiz,omitempty"`
	NanolearningTip  string         `json:"nanolearning_tip"` // Short tip shown after phishing fails
}

// TrainingPage is a single slide/screen in a micro-learning session.
type TrainingPage struct {
	Title     string `json:"title"`
	Body      string `json:"body"`                 // Markdown-formatted content
	MediaType string `json:"media_type,omitempty"` // "image","video","interactive"
	MediaURL  string `json:"media_url,omitempty"`
	TipBox    string `json:"tip_box,omitempty"` // Highlighted best-practice tip
}

// BuiltInQuiz is the quiz definition for a built-in content piece.
type BuiltInQuiz struct {
	PassPercentage int               `json:"pass_percentage"`
	Questions      []BuiltInQuestion `json:"questions"`
}

// BuiltInQuestion is a single quiz question in the content library.
type BuiltInQuestion struct {
	QuestionText  string   `json:"question_text"`
	Options       []string `json:"options"`
	CorrectOption int      `json:"correct_option"` // 0-based index
}

// GetBuiltInContentLibrary returns the full content library.
func GetBuiltInContentLibrary() []BuiltInTrainingContent {
	return builtInContentLibrary
}

// GetBuiltInContentByCategory returns content filtered by category.
func GetBuiltInContentByCategory(category string) []BuiltInTrainingContent {
	result := []BuiltInTrainingContent{}
	for _, c := range builtInContentLibrary {
		if c.Category == category {
			result = append(result, c)
		}
	}
	return result
}

// GetBuiltInContentByDifficulty returns content filtered by difficulty/tier.
func GetBuiltInContentByDifficulty(level int) []BuiltInTrainingContent {
	result := []BuiltInTrainingContent{}
	for _, c := range builtInContentLibrary {
		if c.DifficultyLevel == level {
			result = append(result, c)
		}
	}
	return result
}

// GetBuiltInContentBySlug returns a single content piece by slug.
func GetBuiltInContentBySlug(slug string) *BuiltInTrainingContent {
	for _, c := range builtInContentLibrary {
		if c.Slug == slug {
			return &c
		}
	}
	return nil
}

// GetContentCategories returns all available content categories with counts.
func GetContentCategories() []ContentCategorySummary {
	counts := map[string]int{}
	for _, c := range builtInContentLibrary {
		counts[c.Category]++
	}
	labels := map[string]string{
		ContentCategoryPhishing:       "Phishing & Email Security",
		ContentCategoryPasswords:      "Passwords & Authentication",
		ContentCategorySocialEng:      "Social Engineering",
		ContentCategoryDataProtection: "Data Protection & Privacy",
		ContentCategoryMalware:        "Malware & Ransomware",
		ContentCategoryPhysicalSec:    "Physical Security",
		ContentCategoryMobileSec:      "Mobile & Device Security",
		ContentCategoryRemoteWork:     "Remote Work Security",
		ContentCategoryCompliance:     "Compliance & Regulations",
		ContentCategoryIncident:       "Incident Response",
		ContentCategoryCloudSec:       "Cloud & SaaS Security",
		ContentCategoryAISec:          "AI & Deepfake Threats",
	}
	result := []ContentCategorySummary{}
	for cat, count := range counts {
		result = append(result, ContentCategorySummary{
			Slug:  cat,
			Label: labels[cat],
			Count: count,
		})
	}
	return result
}

// ContentCategorySummary holds a category with its label and content count.
type ContentCategorySummary struct {
	Slug  string `json:"slug"`
	Label string `json:"label"`
	Count int    `json:"count"`
}

// SeedResult captures the outcome of seeding content into an org.
type SeedResult struct {
	TiersCreated    int `json:"tiers_created"`
	SessionsCreated int `json:"sessions_created"`
	CoursesCreated  int `json:"courses_created"`
	QuizzesCreated  int `json:"quizzes_created"`
	Skipped         int `json:"skipped"`
}

// tierSpec defines a tier to be created during content seeding.
type tierSpec struct {
	slug        string
	name        string
	description string
	sortOrder   int
}

// defaultTierSpecs returns the standard academy tier definitions.
func defaultTierSpecs() []tierSpec {
	return []tierSpec{
		{"bronze", "Bronze — Fundamentals", "Build your cybersecurity foundation. Learn to recognize common threats and protect yourself online.", 1},
		{"silver", "Silver — Intermediate", "Deepen your knowledge with more advanced attack techniques, data protection, and security best practices.", 2},
		{"gold", "Gold — Advanced", "Master sophisticated threats including targeted attacks, compliance requirements, and incident response.", 3},
		{"platinum", "Platinum — Expert", "Become a cybersecurity champion. Tackle AI-driven threats, zero-trust architecture, and security leadership.", 4},
	}
}

// diffToTierSlug maps a difficulty level to the corresponding tier slug.
var diffToTierSlug = map[int]string{
	ContentDiffBronze:   "bronze",
	ContentDiffSilver:   "silver",
	ContentDiffGold:     "gold",
	ContentDiffPlatinum: "platinum",
}

// ensureDefaultTiers ensures all default academy tiers exist for an org and
// returns a slug→ID mapping along with the count of newly created tiers.
func ensureDefaultTiers(orgId int64) (map[string]int64, int) {
	tierMap := map[string]int64{}
	created := 0
	for _, ts := range defaultTierSpecs() {
		existing, err := GetAcademyTierBySlug(orgId, ts.slug)
		if err == nil {
			tierMap[ts.slug] = existing.Id
			continue
		}
		tier := &AcademyTier{
			OrgId:       orgId,
			Slug:        ts.slug,
			Name:        ts.name,
			Description: ts.description,
			SortOrder:   ts.sortOrder,
			IsActive:    true,
		}
		if err := CreateAcademyTier(tier); err != nil {
			log.Errorf("SeedBuiltInContent: failed to create tier %s: %v", ts.slug, err)
			continue
		}
		tierMap[ts.slug] = tier.Id
		created++
	}
	return tierMap, created
}

// seedPresentation creates a TrainingPresentation from built-in content.
// Returns the saved presentation or nil on error.
func seedPresentation(orgId, userId int64, content BuiltInTrainingContent) *TrainingPresentation {
	pagesJSON := "[]"
	if len(content.Pages) > 0 {
		pj, _ := json.Marshal(content.Pages)
		pagesJSON = string(pj)
	}

	tp := &TrainingPresentation{
		OrgId:        orgId,
		Name:         "[Nivoxis] " + content.Title,
		Description:  content.Description,
		FileName:     content.Slug + ".builtin",
		FilePath:     "builtin://" + content.Slug,
		FileSize:     0,
		ContentType:  "application/nivoxis-builtin",
		ContentPages: pagesJSON,
		UploadedBy:   userId,
		CreatedDate:  time.Now().UTC(),
		ModifiedDate: time.Now().UTC(),
	}
	if err := db.Save(tp).Error; err != nil {
		log.Errorf("SeedBuiltInContent: failed to create presentation %s: %v", content.Slug, err)
		return nil
	}
	return tp
}

// seedAcademySession creates an academy session linking a presentation to a tier.
// Returns true if the session was created successfully.
func seedAcademySession(tierId, presentationId int64, content BuiltInTrainingContent) bool {
	var maxSort int
	db.Table("academy_sessions").Where("tier_id = ?", tierId).
		Select("COALESCE(MAX(sort_order), 0)").Row().Scan(&maxSort)

	session := &AcademySession{
		TierId:           tierId,
		PresentationId:   presentationId,
		SortOrder:        maxSort + 1,
		EstimatedMinutes: content.EstimatedMinutes,
		IsRequired:       true,
	}
	if err := CreateAcademySession(session); err != nil {
		log.Errorf("SeedBuiltInContent: failed to create session for %s: %v", content.Slug, err)
		return false
	}
	return true
}

// seedQuiz creates a quiz with questions for a presentation.
// Returns true if the quiz was created successfully.
func seedQuiz(presentationId, userId int64, builtInQuiz *BuiltInQuiz, slug string) bool {
	if builtInQuiz == nil || len(builtInQuiz.Questions) == 0 {
		return false
	}
	quiz := &Quiz{
		PresentationId: presentationId,
		PassPercentage: builtInQuiz.PassPercentage,
		CreatedBy:      userId,
	}
	if err := PostQuiz(quiz); err != nil {
		log.Errorf("SeedBuiltInContent: failed to create quiz for %s: %v", slug, err)
		return false
	}
	questions := make([]QuizQuestion, len(builtInQuiz.Questions))
	for i, q := range builtInQuiz.Questions {
		optJSON, _ := json.Marshal(q.Options)
		questions[i] = QuizQuestion{
			QuestionText:  q.QuestionText,
			Options:       string(optJSON),
			CorrectOption: q.CorrectOption,
		}
	}
	if err := SaveQuizQuestions(quiz.Id, questions); err != nil {
		log.Errorf("SeedBuiltInContent: failed to save quiz questions for %s: %v", slug, err)
		return false
	}
	return true
}

// SeedBuiltInContent seeds the built-in content library into an organization.
// It creates training presentations, academy tiers, sessions, and quizzes.
// Existing content (matched by slug-based naming) is skipped.
func SeedBuiltInContent(orgId, userId int64) (*SeedResult, error) {
	result := &SeedResult{}

	// 1. Ensure default academy tiers exist for this org
	tierMap, tiersCreated := ensureDefaultTiers(orgId)
	result.TiersCreated = tiersCreated

	// 2. For each content item, create a TrainingPresentation + academy session + quiz
	for _, content := range builtInContentLibrary {
		presentationName := "[Nivoxis] " + content.Title
		existing := TrainingPresentation{}
		if err := db.Where("org_id = ? AND name = ?", orgId, presentationName).First(&existing).Error; err == nil {
			result.Skipped++
			continue
		}

		tp := seedPresentation(orgId, userId, content)
		if tp == nil {
			continue
		}
		result.CoursesCreated++

		if tierId, ok := tierMap[diffToTierSlug[content.DifficultyLevel]]; ok {
			if seedAcademySession(tierId, tp.Id, content) {
				result.SessionsCreated++
			}
		}

		if seedQuiz(tp.Id, userId, content.Quiz, content.Slug) {
			result.QuizzesCreated++
		}
	}

	return result, nil
}

// OrgHasBuiltInContent returns true if the organization has at least one
// Nivoxis built-in training presentation. This indicates the org was
// previously initialized with the content library and should receive updates.
func OrgHasBuiltInContent(orgId int64) bool {
	var count int
	db.Table("training_presentations").
		Where("org_id = ? AND content_type = ?", orgId, "application/nivoxis-builtin").
		Count(&count)
	return count > 0
}

// GetOrgSystemUser returns the ID of an admin user for the given org to use
// as the system user for automated operations (e.g. content seeding).
// Returns 0 if no suitable user is found.
func GetOrgSystemUser(orgId int64) int64 {
	var user User
	// Look for the first admin user in this org
	err := db.Where("org_id = ? AND role_id IN (SELECT id FROM roles WHERE slug IN (?, ?))",
		orgId, "admin", "org_admin").
		Order("id asc").
		First(&user).Error
	if err != nil {
		// Fall back to any user in the org
		err = db.Where(queryWhereOrgID, orgId).Order("id asc").First(&user).Error
		if err != nil {
			return 0
		}
	}
	return user.Id
}
