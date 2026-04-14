package models

import (
	"time"
)

// Feature slug constants used for feature gating across the platform.
const (
	FeatureBasicBRS           = "basic_brs"
	FeatureAdvancedBRS        = "advanced_brs"
	FeatureAITemplates        = "ai_templates"
	FeatureAutopilot          = "autopilot"
	FeatureAcademyAdvanced    = "academy_advanced"
	FeatureGamification       = "gamification"
	FeatureReportButton       = "report_button"
	FeatureThreatAlertsRead   = "threat_alerts_read"
	FeatureThreatAlertsCreate = "threat_alerts_create"
	FeatureBoardReports       = "board_reports"
	FeatureI18NFull           = "i18n_full"
	FeatureSCIM               = "scim"
	FeatureZIM                = "zim"
	FeatureAIAssistant        = "ai_assistant"
	FeaturePowerBI            = "power_bi"
	FeatureComplianceMapping  = "compliance_mapping"
	FeatureMSPWhitelabel      = "msp_whitelabel"
	FeatureCyberHygiene       = "cyber_hygiene"
	// FeatureCustomTrainingBuilder gates the multi-module custom training
	// builder (upload multiple PDF/PPTX/video assets per course). Sits in
	// the All-in-One tier.
	FeatureCustomTrainingBuilder = "custom_training_builder"
	// FeatureInboxMonitor gates real-time in-inbox AI threat detection.
	FeatureInboxMonitor = "inbox_monitor"
	// FeatureBECDetection gates automated BEC detection.
	FeatureBECDetection = "bec_detection"
	// FeatureGraymailClassification gates automated graymail categorization.
	FeatureGraymailClassification = "graymail_classification"
	// FeatureOneClickRemediation gates one-click threat removal from inboxes.
	FeatureOneClickRemediation = "one_click_remediation"
	// FeaturePhishingTickets gates automated phishing ticket management.
	FeaturePhishingTickets = "phishing_tickets"
	// FeatureGoogleWorkspaceReportButton gates Google Workspace support for the report button.
	FeatureGoogleWorkspaceReportButton = "google_workspace_report_button"
	// FeatureNLPEmailAnalysis gates AI-powered NLP analysis of reported emails.
	FeatureNLPEmailAnalysis = "nlp_email_analysis"
	// FeatureNetworkEvents gates the network events security dashboard.
	FeatureNetworkEvents = "network_events"
	// FeatureVishing gates voice phishing (vishing) simulation campaigns.
	FeatureVishing = "vishing"
	// FeatureTemplateLibraryDB gates the DB-backed template library management.
	FeatureTemplateLibraryDB = "template_library_db"
	// FeatureInboxAIFeedback gates AI-powered inbox feedback for users (Outlook/Gmail add-ins).
	FeatureInboxAIFeedback = "inbox_ai_feedback"
	// FeatureEnhancedBoardReports gates enhanced board reports with AI narrative and ROI.
	FeatureEnhancedBoardReports = "enhanced_board_reports"
	// FeatureROIDashboard gates the executive ROI reporting dashboard.
	FeatureROIDashboard = "roi_dashboard"
	// FeatureAITranslation gates AI-powered dynamic content translation.
	FeatureAITranslation = "ai_translation"
	// FeatureDomainPool gates the sending domain pool management.
	FeatureDomainPool = "domain_pool"
	// FeatureRealtimeDashboard gates the WebSocket-powered real-time dashboard.
	FeatureRealtimeDashboard = "realtime_dashboard"
)

// SubscriptionTier represents a pricing tier with associated limits.
type SubscriptionTier struct {
	Id           int64     `json:"id" gorm:"primary_key"`
	Slug         string    `json:"slug"`
	Name         string    `json:"name"`
	Description  string    `json:"description"`
	MaxUsers     int       `json:"max_users"`
	MaxCampaigns int       `json:"max_campaigns"`
	IsActive     bool      `json:"is_active"`
	SortOrder    int       `json:"sort_order"`
	CreatedDate  time.Time `json:"created_date"`
	Features     []string  `json:"features" gorm:"-"`
}

// TierFeature maps a feature slug to a subscription tier.
type TierFeature struct {
	Id          int64  `json:"id" gorm:"primary_key"`
	TierId      int64  `json:"tier_id"`
	FeatureSlug string `json:"feature_slug"`
}

// GetSubscriptionTiers returns all active tiers ordered by sort_order.
func GetSubscriptionTiers() ([]SubscriptionTier, error) {
	tiers := []SubscriptionTier{}
	err := db.Where("is_active = ?", true).Order("sort_order asc").Find(&tiers).Error
	if err != nil {
		return tiers, err
	}
	for i := range tiers {
		features, fErr := GetTierFeatures(tiers[i].Id)
		if fErr == nil {
			tiers[i].Features = features
		}
	}
	return tiers, nil
}

// GetSubscriptionTier returns a single tier by id.
func GetSubscriptionTier(id int64) (SubscriptionTier, error) {
	t := SubscriptionTier{}
	err := db.Where("id = ?", id).First(&t).Error
	if err != nil {
		return t, err
	}
	features, fErr := GetTierFeatures(t.Id)
	if fErr == nil {
		t.Features = features
	}
	return t, nil
}

// GetSubscriptionTierBySlug returns a tier by its slug.
func GetSubscriptionTierBySlug(slug string) (SubscriptionTier, error) {
	t := SubscriptionTier{}
	err := db.Where("slug = ?", slug).First(&t).Error
	if err != nil {
		return t, err
	}
	features, fErr := GetTierFeatures(t.Id)
	if fErr == nil {
		t.Features = features
	}
	return t, nil
}

// GetTierFeatures returns the feature slugs for a given tier.
func GetTierFeatures(tierId int64) ([]string, error) {
	var tfs []TierFeature
	err := db.Where("tier_id = ?", tierId).Find(&tfs).Error
	if err != nil {
		return nil, err
	}
	slugs := make([]string, len(tfs))
	for i, tf := range tfs {
		slugs[i] = tf.FeatureSlug
	}
	return slugs, nil
}

// OrgHasFeature checks whether the organization has a specific feature
// enabled via its subscription tier. Returns true if the org's tier includes
// the given feature slug.
func OrgHasFeature(orgId int64, featureSlug string) bool {
	org, err := GetOrganization(orgId)
	if err != nil {
		return false
	}
	var count int
	err = db.Model(&TierFeature{}).
		Where("tier_id = ? AND feature_slug = ?", org.TierId, featureSlug).
		Count(&count).Error
	if err != nil {
		return false
	}
	return count > 0
}

// GetOrgFeatures returns a map of feature slugs to booleans for the org's
// current subscription tier. Useful for passing to templates.
func GetOrgFeatures(orgId int64) map[string]bool {
	features := make(map[string]bool)
	org, err := GetOrganization(orgId)
	if err != nil {
		return features
	}
	slugs, err := GetTierFeatures(org.TierId)
	if err != nil {
		return features
	}
	for _, s := range slugs {
		features[s] = true
	}
	return features
}
