package models

import (
	"math/rand"
	"strings"
	"time"

	log "github.com/gophish/gophish/logger"
)

// NanolearningEvent records a nanolearning tip shown to a user after a
// phishing simulation click. This allows tracking which tips were shown,
// how often, and to whom, enabling analytics on micro-intervention effectiveness.
type NanolearningEvent struct {
	Id           int64     `json:"id" gorm:"primary_key"`
	UserId       int64     `json:"user_id"`
	Email        string    `json:"email"`
	CampaignId   int64     `json:"campaign_id"`
	ResultId     string    `json:"result_id"`
	ContentSlug  string    `json:"content_slug"` // The built-in content slug the tip came from
	TipText      string    `json:"tip_text"`     // The actual nanolearning tip shown
	Category     string    `json:"category"`     // Content category (phishing, passwords, etc.)
	Acknowledged bool      `json:"acknowledged"` // Whether the user clicked "I understand"
	CreatedDate  time.Time `json:"created_date"`
}

func (NanolearningEvent) TableName() string { return "nanolearning_events" }

// NanolearningTip is a lightweight struct passed to the phishing response
// renderer to inject a tip into the page.
type NanolearningTip struct {
	Slug     string `json:"slug"`
	Tip      string `json:"tip"`
	Category string `json:"category"`
	Title    string `json:"title"`
}

// GetNanolearningTipForCategory returns a relevant nanolearning tip from the
// built-in content library, matched by category. If no match is found for the
// given category, a random tip from any category is returned.
func GetNanolearningTipForCategory(category string) *NanolearningTip {
	library := GetBuiltInContentLibrary()
	if len(library) == 0 {
		return nil
	}

	// First try exact category match
	var matches []BuiltInTrainingContent
	for _, c := range library {
		if c.NanolearningTip != "" && c.Category == category {
			matches = append(matches, c)
		}
	}

	// Fall back to all content with nanolearning tips
	if len(matches) == 0 {
		for _, c := range library {
			if c.NanolearningTip != "" {
				matches = append(matches, c)
			}
		}
	}

	if len(matches) == 0 {
		return nil
	}

	// Pick a random tip from the matches
	pick := matches[rand.Intn(len(matches))]
	return &NanolearningTip{
		Slug:     pick.Slug,
		Tip:      pick.NanolearningTip,
		Category: pick.Category,
		Title:    pick.Title,
	}
}

// GetNanolearningTipForCampaign selects a contextually relevant nanolearning
// tip based on the campaign's template category. Falls back to phishing category.
func GetNanolearningTipForCampaign(campaignId int64) *NanolearningTip {
	// Try to determine the campaign's template category
	c := Campaign{}
	if err := db.Where("id=?", campaignId).First(&c).Error; err != nil {
		return GetNanolearningTipForCategory(ContentCategoryPhishing)
	}

	// Look up the template to determine category/tags
	t := Template{}
	if err := db.Where("id=?", c.TemplateId).First(&t).Error; err != nil {
		return GetNanolearningTipForCategory(ContentCategoryPhishing)
	}

	// Map common template categories to content categories
	category := mapTemplateCategoryToContent(t.Name)
	return GetNanolearningTipForCategory(category)
}

// mapTemplateCategoryToContent maps a template name/category to a built-in content category.
// Uses simple keyword matching for flexibility.
func mapTemplateCategoryToContent(templateName string) string {
	name := templateName
	switch {
	case containsAny(name, "password", "credential", "login", "sign-in"):
		return ContentCategoryPasswords
	case containsAny(name, "ceo", "executive", "wire", "invoice", "payment", "bec"):
		return ContentCategorySocialEng
	case containsAny(name, "malware", "ransomware", "macro", "attachment"):
		return ContentCategoryMalware
	case containsAny(name, "gdpr", "compliance", "privacy", "data protection"):
		return ContentCategoryDataProtection
	case containsAny(name, "mobile", "sms", "smishing", "device"):
		return ContentCategoryMobileSec
	case containsAny(name, "wifi", "vpn", "remote", "work from home"):
		return ContentCategoryRemoteWork
	case containsAny(name, "qr", "qrcode"):
		return ContentCategoryPhishing
	case containsAny(name, "cloud", "saas", "share", "drive"):
		return ContentCategoryCloudSec
	case containsAny(name, "ai", "deepfake", "chatgpt"):
		return ContentCategoryAISec
	default:
		return ContentCategoryPhishing
	}
}

// containsAny returns true if the string s contains any of the substrings (case-insensitive).
func containsAny(s string, subs ...string) bool {
	lower := strings.ToLower(s)
	for _, sub := range subs {
		if strings.Contains(lower, sub) {
			return true
		}
	}
	return false
}

// RecordNanolearningEvent saves a nanolearning event to the database.
func RecordNanolearningEvent(event *NanolearningEvent) error {
	event.CreatedDate = time.Now().UTC()
	return db.Save(event).Error
}

// AcknowledgeNanolearning marks a nanolearning event as acknowledged by the user.
func AcknowledgeNanolearning(eventId int64) error {
	return db.Model(&NanolearningEvent{}).Where("id=?", eventId).Updates(map[string]interface{}{
		"acknowledged": true,
	}).Error
}

// GetNanolearningEventsForUser returns all nanolearning events for a user,
// ordered by creation date (most recent first).
func GetNanolearningEventsForUser(userId int64) ([]NanolearningEvent, error) {
	var events []NanolearningEvent
	err := db.Where("user_id=?", userId).Order("created_date desc").Find(&events).Error
	return events, err
}

// GetNanolearningStats returns aggregate nanolearning statistics for an org.
type NanolearningStats struct {
	TotalShown        int64   `json:"total_shown"`
	TotalAcknowledged int64   `json:"total_acknowledged"`
	AckRate           float64 `json:"ack_rate"`
	UniqueUsers       int64   `json:"unique_users"`
}

// GetNanolearningStatsForOrg returns aggregate nanolearning stats for all
// users in an org based on their click events.
func GetNanolearningStatsForOrg(orgId int64) NanolearningStats {
	stats := NanolearningStats{}

	// Get user IDs for the org
	var userIds []int64
	db.Table("users").Where(queryWhereOrgID, orgId).Pluck("id", &userIds)
	if len(userIds) == 0 {
		return stats
	}

	db.Table("nanolearning_events").Where("user_id IN (?)", userIds).Count(&stats.TotalShown)
	db.Table("nanolearning_events").Where("user_id IN (?) AND acknowledged = ?", userIds, true).Count(&stats.TotalAcknowledged)

	var uniqueUsers int64
	db.Table("nanolearning_events").Where("user_id IN (?)", userIds).
		Select("COUNT(DISTINCT user_id)").Row().Scan(&uniqueUsers)
	stats.UniqueUsers = uniqueUsers

	if stats.TotalShown > 0 {
		stats.AckRate = float64(stats.TotalAcknowledged) / float64(stats.TotalShown) * 100
	}

	return stats
}

// TriggerNanolearningOnClick is the main entry point called when a user clicks
// a phishing link. It selects a relevant tip and records the event.
// Returns the tip to be displayed, or nil if no tip is available.
// Deduplicates: only one nanolearning event per user per campaign.
func TriggerNanolearningOnClick(campaignId int64, email, resultId string) *NanolearningTip {
	// Dedup: don't create another event if one already exists for this email+campaign
	var existing int
	db.Table("nanolearning_events").
		Where("email = ? AND campaign_id = ?", email, campaignId).
		Count(&existing)
	if existing > 0 {
		return nil
	}

	tip := GetNanolearningTipForCampaign(campaignId)
	if tip == nil {
		return nil
	}

	// Look up the user (may not be a platform user)
	userId := int64(0)
	user, err := GetUserByUsername(email)
	if err != nil {
		user, err = GetUserByEmail(email)
	}
	if err == nil {
		userId = user.Id
	}

	// Record the event
	event := &NanolearningEvent{
		UserId:      userId,
		Email:       email,
		CampaignId:  campaignId,
		ResultId:    resultId,
		ContentSlug: tip.Slug,
		TipText:     tip.Tip,
		Category:    tip.Category,
	}

	if err := RecordNanolearningEvent(event); err != nil {
		log.Errorf("Failed to record nanolearning event: %v", err)
	}

	return tip
}
