package models

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ── User-Facing AI Feedback ─────────────────────────────────────
// When users report emails or the inbox AI assistant scans messages,
// provide structured feedback back to the end user so they understand
// *why* an email was flagged as safe, suspicious, or phishing.

// Shared query constants for this file.
const (
	orderCreatedDateDescFB = "created_date DESC"
	queryOrgFeedbackType   = "org_id = ? AND feedback_type = ?"
)

// UserEmailFeedback is the AI analysis result surfaced to end users
// through the report button, Outlook add-in, or Gmail add-on.
type UserEmailFeedback struct {
	Id               int64     `json:"id" gorm:"primary_key"`
	OrgId            int64     `json:"org_id"`
	UserId           int64     `json:"user_id"`
	Email            string    `json:"email"`
	MessageId        string    `json:"message_id"`
	Subject          string    `json:"subject"`
	SenderEmail      string    `json:"sender_email"`
	ThreatLevel      string    `json:"threat_level"`      // safe, suspicious, likely_phishing, confirmed_phishing
	ConfidenceScore  float64   `json:"confidence_score"`  // 0-1
	Summary          string    `json:"summary"`           // 2-3 sentence plain-language explanation
	Indicators       string    `json:"indicators"`        // JSON array of detected indicators
	Recommendation   string    `json:"recommendation"`    // what the user should do
	WasSimulation    bool      `json:"was_simulation"`    // true if this was a Nivoxis phishing simulation
	SimulationResult string    `json:"simulation_result"` // "correctly_reported", "missed", ""
	LearningTip      string    `json:"learning_tip"`      // educational tip based on the email type
	FeedbackRead     bool      `json:"feedback_read"`     // has the user seen this feedback
	UserAcknowledged bool      `json:"user_acknowledged"` // has the user dismissed/acknowledged
	CreatedDate      time.Time `json:"created_date"`
	ModifiedDate     time.Time `json:"modified_date"`
}

func (UserEmailFeedback) TableName() string { return "user_email_feedback" }

// EmailIndicator is defined in inbox_security.go — reused here.
// (no redeclaration needed)

// CreateUserEmailFeedback saves feedback for a user.
func CreateUserEmailFeedback(f *UserEmailFeedback) error {
	f.CreatedDate = time.Now().UTC()
	f.ModifiedDate = time.Now().UTC()
	return db.Save(f).Error
}

// GetUserEmailFeedback returns feedback items for a user, newest first.
func GetUserEmailFeedback(userId int64, limit int) ([]UserEmailFeedback, error) {
	if limit <= 0 {
		limit = 20
	}
	var items []UserEmailFeedback
	err := db.Where(queryWhereUserID, userId).
		Order(orderCreatedDateDescFB).
		Limit(limit).
		Find(&items).Error
	return items, err
}

// GetUnreadFeedbackCount returns how many unread feedback items a user has.
func GetUnreadFeedbackCount(userId int64) int64 {
	var count int64
	db.Model(&UserEmailFeedback{}).
		Where("user_id = ? AND feedback_read = ?", userId, false).
		Count(&count)
	return count
}

// MarkFeedbackRead marks a feedback item as read.
func MarkFeedbackRead(feedbackId, userId int64) error {
	return db.Model(&UserEmailFeedback{}).
		Where("id = ? AND user_id = ?", feedbackId, userId).
		Updates(map[string]interface{}{
			"feedback_read": true,
			"modified_date": time.Now().UTC(),
		}).Error
}

// AcknowledgeFeedback marks feedback as acknowledged by the user.
func AcknowledgeFeedback(feedbackId, userId int64) error {
	return db.Model(&UserEmailFeedback{}).
		Where("id = ? AND user_id = ?", feedbackId, userId).
		Updates(map[string]interface{}{
			"user_acknowledged": true,
			"modified_date":     time.Now().UTC(),
		}).Error
}

// BuildUserFeedbackFromAnalysis converts an AI scan result into user-facing feedback.
func BuildUserFeedbackFromAnalysis(orgId int64, userId int64, email string, scanResult *InboxScanResult) *UserEmailFeedback {
	recommendation := buildFeedbackRecommendation(scanResult.ThreatLevel)
	learningTip := buildLearningTip(scanResult.Classification, scanResult.ThreatLevel)

	return &UserEmailFeedback{
		OrgId:           orgId,
		UserId:          userId,
		Email:           email,
		MessageId:       scanResult.MessageId,
		Subject:         scanResult.Subject,
		SenderEmail:     scanResult.SenderEmail,
		ThreatLevel:     scanResult.ThreatLevel,
		ConfidenceScore: scanResult.ConfidenceScore,
		Summary:         scanResult.Summary,
		Indicators:      scanResult.Indicators,
		Recommendation:  recommendation,
		LearningTip:     learningTip,
	}
}

func buildFeedbackRecommendation(threatLevel string) string {
	switch threatLevel {
	case ThreatLevelConfirmedPhishing:
		return "🚨 Do NOT click any links or download attachments in this email. It has been flagged as confirmed phishing. If you already interacted with it, contact your IT security team immediately."
	case ThreatLevelLikelyPhishing:
		return "⚠️ This email shows strong indicators of phishing. Do not click links or reply. It has been escalated for review by your security team."
	case ThreatLevelSuspicious:
		return "⚠️ This email has some suspicious elements. Verify the sender through a separate channel (phone, internal chat) before taking any action."
	default:
		return "✅ This email appears to be legitimate. No immediate action needed. Stay vigilant — if anything seems off, report it."
	}
}

func buildLearningTip(classification, threatLevel string) string {
	switch classification {
	case ClassificationBEC:
		return "💡 Business Email Compromise (BEC): Attackers impersonate executives or vendors to request urgent payments or sensitive data. Always verify unusual requests through a separate channel."
	case "credential_harvesting":
		return "💡 Credential Harvesting: These emails try to steal your login credentials by mimicking trusted services. Always check the URL before entering passwords — hover over links to verify they go to legitimate domains."
	case "graymail":
		return "💡 Graymail: This email isn't malicious but is likely unwanted marketing or newsletters. Consider unsubscribing to reduce inbox clutter."
	case "malware_delivery":
		return "💡 Malware Delivery: These emails contain malicious attachments or links that install malware. Never open unexpected attachments, especially .exe, .zip, or macro-enabled documents."
	default:
		if threatLevel == ThreatLevelSafe {
			return "💡 Tip: Even legitimate emails can sometimes be crafted by attackers. Always verify before sharing sensitive information."
		}
		return "💡 When in doubt, don't click. Verify the sender through a separate communication channel before taking action."
	}
}

// ── False Positive Feedback Loop ────────────────────────────────
// Allows admins to mark AI classifications as incorrect, building a
// feedback dataset that can improve prompt engineering over time.

// AIClassificationFeedback records admin corrections to AI email analysis.
type AIClassificationFeedback struct {
	Id                      int64     `json:"id" gorm:"primary_key"`
	OrgId                   int64     `json:"org_id"`
	ScanResultId            int64     `json:"scan_result_id"`
	ReportedEmailId         int64     `json:"reported_email_id"`
	OriginalThreatLevel     string    `json:"original_threat_level"`
	CorrectedThreatLevel    string    `json:"corrected_threat_level"`
	OriginalClassification  string    `json:"original_classification"`
	CorrectedClassification string    `json:"corrected_classification"`
	FeedbackType            string    `json:"feedback_type"` // "false_positive", "false_negative", "misclassification"
	AdminNotes              string    `json:"admin_notes"`
	AdminUserId             int64     `json:"admin_user_id"`
	CreatedDate             time.Time `json:"created_date"`
}

func (AIClassificationFeedback) TableName() string { return "ai_classification_feedback" }

// SubmitClassificationFeedback records an admin's correction of an AI classification.
func SubmitClassificationFeedback(fb *AIClassificationFeedback) error {
	fb.CreatedDate = time.Now().UTC()

	// Determine feedback type
	if fb.CorrectedThreatLevel == ThreatLevelSafe && fb.OriginalThreatLevel != ThreatLevelSafe {
		fb.FeedbackType = "false_positive"
	} else if fb.OriginalThreatLevel == ThreatLevelSafe && fb.CorrectedThreatLevel != ThreatLevelSafe {
		fb.FeedbackType = "false_negative"
	} else {
		fb.FeedbackType = "misclassification"
	}

	if err := db.Save(fb).Error; err != nil {
		return fmt.Errorf("save classification feedback: %w", err)
	}

	// Update the scan result with the corrected classification
	if fb.ScanResultId > 0 {
		db.Model(&InboxScanResult{}).Where("id = ?", fb.ScanResultId).Updates(map[string]interface{}{
			"threat_level":      fb.CorrectedThreatLevel,
			"classification":    fb.CorrectedClassification,
			"admin_override":    true,
			"admin_override_by": fb.AdminUserId,
		})
	}

	return nil
}

// GetClassificationFeedbackStats returns aggregated stats about AI accuracy.
type ClassificationAccuracyStats struct {
	TotalAnalysed      int64   `json:"total_analysed"`
	TotalCorrected     int64   `json:"total_corrected"`
	FalsePositives     int64   `json:"false_positives"`
	FalseNegatives     int64   `json:"false_negatives"`
	Misclassifications int64   `json:"misclassifications"`
	AccuracyRate       float64 `json:"accuracy_rate"`       // (total - corrected) / total
	FalsePositiveRate  float64 `json:"false_positive_rate"` // FP / total
}

// GetClassificationAccuracy returns AI accuracy metrics for an org.
func GetClassificationAccuracy(orgId int64) ClassificationAccuracyStats {
	stats := ClassificationAccuracyStats{}

	// Total scan results
	db.Model(&InboxScanResult{}).Where(queryWhereOrgID, orgId).Count(&stats.TotalAnalysed)

	// Corrections
	db.Model(&AIClassificationFeedback{}).Where(queryWhereOrgID, orgId).Count(&stats.TotalCorrected)
	db.Model(&AIClassificationFeedback{}).Where(queryOrgFeedbackType, orgId, "false_positive").Count(&stats.FalsePositives)
	db.Model(&AIClassificationFeedback{}).Where(queryOrgFeedbackType, orgId, "false_negative").Count(&stats.FalseNegatives)
	db.Model(&AIClassificationFeedback{}).Where(queryOrgFeedbackType, orgId, "misclassification").Count(&stats.Misclassifications)

	if stats.TotalAnalysed > 0 {
		stats.AccuracyRate = float64(stats.TotalAnalysed-stats.TotalCorrected) / float64(stats.TotalAnalysed) * 100
		stats.FalsePositiveRate = float64(stats.FalsePositives) / float64(stats.TotalAnalysed) * 100
	}

	return stats
}

// GetRecentFeedback returns recent AI corrections for prompt improvement analysis.
func GetRecentFeedback(orgId int64, limit int) ([]AIClassificationFeedback, error) {
	if limit <= 0 {
		limit = 50
	}
	var items []AIClassificationFeedback
	err := db.Where(queryWhereOrgID, orgId).
		Order(orderCreatedDateDescFB).
		Limit(limit).
		Find(&items).Error
	return items, err
}

// BuildFalsePositivePromptContext generates a context block that can be injected
// into the AI email analysis prompt to reduce repeat false positives.
func BuildFalsePositivePromptContext(orgId int64) string {
	var recentFPs []AIClassificationFeedback
	db.Where(queryOrgFeedbackType, orgId, "false_positive").
		Order(orderCreatedDateDescFB).
		Limit(10).
		Find(&recentFPs)

	if len(recentFPs) == 0 {
		return ""
	}

	context := "\n--- False Positive Avoidance Context ---\n"
	context += "The following types of emails were previously flagged incorrectly as threats. Be more conservative when analysing similar patterns:\n"
	for i, fp := range recentFPs {
		if i >= 5 {
			break
		}
		notes := fp.AdminNotes
		if notes == "" {
			notes = fmt.Sprintf("Originally classified as %s, actually %s", fp.OriginalThreatLevel, fp.CorrectedThreatLevel)
		}
		context += fmt.Sprintf("- %s\n", notes)
	}
	context += "--- End Context ---\n"
	return context
}

// ── Outlook/Gmail Add-in Support ────────────────────────────────
// API models for native Outlook Add-in and Gmail Add-on integration.

// AddInAnalysisRequest is the payload sent by the Outlook/Gmail add-in
// when a user requests real-time analysis of the email they are viewing.
type AddInAnalysisRequest struct {
	OrgId       int64  `json:"org_id"`
	UserEmail   string `json:"user_email"`
	MessageId   string `json:"message_id"`
	Subject     string `json:"subject"`
	SenderEmail string `json:"sender_email"`
	SenderName  string `json:"sender_name"`
	Headers     string `json:"headers"`
	Body        string `json:"body"`
	Provider    string `json:"provider"` // "outlook", "gmail"
}

// AddInAnalysisResponse is the real-time analysis result returned to the add-in.
type AddInAnalysisResponse struct {
	ThreatLevel     string           `json:"threat_level"`
	ConfidenceScore float64          `json:"confidence_score"`
	Summary         string           `json:"summary"`
	Indicators      []EmailIndicator `json:"indicators"`
	Recommendation  string           `json:"recommendation"`
	LearningTip     string           `json:"learning_tip"`
	WasSimulation   bool             `json:"was_simulation"`
	AnalysisTimeMs  int64            `json:"analysis_time_ms"`
}

// AnalyzeEmailForAddIn performs real-time AI analysis of an email from the
// Outlook/Gmail add-in and returns structured results for display in the sidebar.
func AnalyzeEmailForAddIn(req *AddInAnalysisRequest, aiAnalysis *InboxScanResult) *AddInAnalysisResponse {
	// Check if this is one of our own simulation emails
	wasSimulation := isSimulationEmail(req.OrgId, req.MessageId)

	var indicators []EmailIndicator
	if aiAnalysis.Indicators != "" {
		json.Unmarshal([]byte(aiAnalysis.Indicators), &indicators)
	}

	return &AddInAnalysisResponse{
		ThreatLevel:     aiAnalysis.ThreatLevel,
		ConfidenceScore: aiAnalysis.ConfidenceScore,
		Summary:         aiAnalysis.Summary,
		Indicators:      indicators,
		Recommendation:  buildFeedbackRecommendation(aiAnalysis.ThreatLevel),
		LearningTip:     buildLearningTip(aiAnalysis.Classification, aiAnalysis.ThreatLevel),
		WasSimulation:   wasSimulation,
		AnalysisTimeMs:  int64(aiAnalysis.ScanDurationMs),
	}
}

// isSimulationEmail checks if a message ID corresponds to one of our phishing simulations.
func isSimulationEmail(orgId int64, messageId string) bool {
	if messageId == "" {
		return false
	}
	var count int64
	db.Model(&Result{}).
		Joins("JOIN campaigns c ON results.campaign_id = c.id").
		Where("c.org_id = ? AND results.r_id = ?", orgId, messageId).
		Count(&count)
	return count > 0
}

// ── Webhook/Push Notification Registration ──────────────────────
// Supports Microsoft Graph subscriptions and Gmail Pub/Sub for near-real-time
// email threat detection (< 30 seconds).

// InboxWebhookConfig stores webhook/push notification settings per org.
type InboxWebhookConfig struct {
	Id       int64  `json:"id" gorm:"primary_key"`
	OrgId    int64  `json:"org_id"`
	Provider string `json:"provider"` // "microsoft_graph", "gmail"
	Enabled  bool   `json:"enabled"`
	// Microsoft Graph webhook fields
	SubscriptionId string    `json:"subscription_id,omitempty"`
	WebhookURL     string    `json:"webhook_url,omitempty"`
	ExpirationDate time.Time `json:"expiration_date,omitempty"`
	// Gmail Pub/Sub fields
	PubSubTopicName    string `json:"pubsub_topic,omitempty"`
	PubSubSubscription string `json:"pubsub_subscription,omitempty"`
	HistoryId          string `json:"history_id,omitempty"`
	// Common
	LastNotification time.Time `json:"last_notification,omitempty"`
	CreatedDate      time.Time `json:"created_date"`
	ModifiedDate     time.Time `json:"modified_date"`
}

func (InboxWebhookConfig) TableName() string { return "inbox_webhook_configs" }

// GetInboxWebhookConfig returns the webhook config for an org+provider.
func GetInboxWebhookConfig(orgId int64, provider string) (InboxWebhookConfig, error) {
	var cfg InboxWebhookConfig
	err := db.Where("org_id = ? AND provider = ?", orgId, provider).First(&cfg).Error
	return cfg, err
}

// SaveInboxWebhookConfig creates or updates a webhook config.
func SaveInboxWebhookConfig(cfg *InboxWebhookConfig) error {
	cfg.ModifiedDate = time.Now().UTC()
	if cfg.CreatedDate.IsZero() {
		cfg.CreatedDate = time.Now().UTC()
	}
	return db.Save(cfg).Error
}

// GetActiveWebhookConfigs returns all enabled webhook configurations.
// Used by the webhook subscription lifecycle worker.
func GetActiveWebhookConfigs() ([]InboxWebhookConfig, error) {
	var configs []InboxWebhookConfig
	err := db.Where("enabled = ?", true).Find(&configs).Error
	return configs, err
}

// GetWebhookConfigsByOrg returns all webhook configs for an org.
func GetWebhookConfigsByOrg(orgId int64) ([]InboxWebhookConfig, error) {
	var configs []InboxWebhookConfig
	err := db.Where("org_id = ?", orgId).Find(&configs).Error
	return configs, err
}

// ── User-Specific Threat Intelligence for Inbox AI ──────────────
// Correlates the user's targeting profile (BRS, weak categories, department
// threats) with real email characteristics to prioritize threat detection.

// EnrichScanWithUserThreatIntel checks the user's targeting profile and
// adjusts the scan result's threat level / confidence if the email pattern
// matches the user's known vulnerabilities.
//
// For example, if a user is weak against BEC and the email looks like a
// payment/wire request, bump the confidence score and add a context note.
//
// Returns a supplementary context string for the response summary.
func EnrichScanWithUserThreatIntel(userId int64, scan *InboxScanResult) string {
	profile, err := GetUserTargetingProfile(userId)
	if err != nil || profile == nil || profile.TotalSimulations < 3 {
		return ""
	}

	var notes []string

	// Check if the email classification matches user's weak categories
	for _, wc := range profile.WeakCategories {
		if matchesCategoryPattern(scan, wc.Category) {
			// This email matches a pattern the user is vulnerable to
			// Bump confidence by 10-20% based on weakness severity
			boost := (1.0 - wc.Score/100.0) * 0.2
			scan.ConfidenceScore = clampFloat(scan.ConfidenceScore+boost, 0, 1)

			// If the scan was "safe" but matches a weak category, flag as suspicious
			if scan.ThreatLevel == ThreatLevelSafe && wc.Score < 40 {
				scan.ThreatLevel = ThreatLevelSuspicious
				notes = append(notes, "⚡ This email matches a pattern you've been vulnerable to in training ("+wc.Category+"). Extra caution advised.")
			}
			break
		}
	}

	// Department-specific risk escalation
	if profile.Department != "" {
		deptTP := GetDepartmentThreatProfile(profile.Department)
		if deptTP.RiskMultiplier > 1.2 {
			for _, threat := range deptTP.PrimaryThreats {
				if matchesCategoryPattern(scan, threat.Category) && threat.Relevance >= 0.8 {
					notes = append(notes, "🏢 As a "+profile.Department+" team member, you may be a target for "+threat.Category+" attacks.")
					break
				}
			}
		}
	}

	if len(notes) == 0 {
		return ""
	}

	result := ""
	for _, n := range notes {
		result += n + " "
	}
	return result
}

// matchesCategoryPattern checks if an email's characteristics match a known
// phishing category pattern. This is a heuristic check based on subject/sender keywords.
func matchesCategoryPattern(scan *InboxScanResult, category string) bool {
	subject := strings.ToLower(scan.Subject)
	sender := strings.ToLower(scan.SenderEmail)

	switch category {
	case CategoryBEC:
		return containsAnySubstr(subject, "wire transfer", "payment", "invoice", "urgent request",
			"confidential", "executive", "ceo", "cfo") ||
			containsAnySubstr(sender, "ceo", "cfo", "president", "director")

	case CategoryCredentialHarvesting:
		return containsAnySubstr(subject, "password", "verify", "confirm your", "reset",
			"account suspended", "sign in", "login", "credentials")

	case CategoryHRPayroll:
		return containsAnySubstr(subject, "payroll", "direct deposit", "w-2", "benefits",
			"open enrollment", "salary", "compensation")

	case CategoryITHelpdesk:
		return containsAnySubstr(subject, "helpdesk", "ticket", "password reset",
			"it support", "security update", "patch")

	case CategoryDeliveryNotification:
		return containsAnySubstr(subject, "delivery", "package", "tracking",
			"shipment", "fedex", "ups", "dhl", "usps")

	case CategorySocialEngineering:
		return containsAnySubstr(subject, "congratulations", "you've won", "limited time",
			"act now", "special offer")

	case CategorySupplyChain:
		return containsAnySubstr(subject, "software update", "vendor", "partner portal",
			"security advisory", "dependency", "npm", "github")

	default:
		return false
	}
}

// containsAnySubstr returns true if s contains any of the given substrings.
func containsAnySubstr(s string, subs ...string) bool {
	for _, sub := range subs {
		if strings.Contains(s, sub) {
			return true
		}
	}
	return false
}

// clampFloat constrains a float64 to [min, max].
func clampFloat(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
