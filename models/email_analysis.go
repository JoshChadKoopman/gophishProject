package models

import (
	"errors"
	"time"

	"github.com/gophish/gophish/ai"
	log "github.com/gophish/gophish/logger"
	"github.com/jinzhu/gorm"
)

// Analysis status constants.
const (
	AnalysisStatusPending   = "pending"
	AnalysisStatusAnalyzing = "analyzing"
	AnalysisStatusCompleted = "completed"
	AnalysisStatusFailed    = "failed"
)

// Threat level constants.
const (
	ThreatLevelSafe              = "safe"
	ThreatLevelSuspicious        = "suspicious"
	ThreatLevelLikelyPhishing    = "likely_phishing"
	ThreatLevelConfirmedPhishing = "confirmed_phishing"
)

// Classification constants.
const (
	ClassificationPhishing      = "phishing"
	ClassificationSpearPhishing = "spear_phishing"
	ClassificationBEC           = "bec"
	ClassificationSpam          = "spam"
	ClassificationLegitimate    = "legitimate"
	ClassificationUnknown       = "unknown"
)

// Indicator type constants.
const (
	IndicatorTypeURL             = "url"
	IndicatorTypeDomain          = "domain"
	IndicatorTypeIP              = "ip"
	IndicatorTypeEmailAddress    = "email_address"
	IndicatorTypeAttachment      = "attachment"
	IndicatorTypeHeaderAnomaly   = "header_anomaly"
	IndicatorTypeLanguagePattern = "language_pattern"
	IndicatorTypeUrgencyCue      = "urgency_cue"
	IndicatorTypeImpersonation   = "impersonation"
)

// Indicator severity constants.
const (
	SeverityInfo     = "info"
	SeverityLow      = "low"
	SeverityMedium   = "medium"
	SeverityHigh     = "high"
	SeverityCritical = "critical"
)

// Error variables for email analysis operations.
var (
	ErrAnalysisNotFound   = errors.New("Email analysis not found")
	ErrAnalysisInProgress = errors.New("Analysis already exists for this reported email")
)

// EmailAnalysis stores the results of an AI-powered NLP analysis of a
// reported email. Each analysis is scoped to an organization and linked
// to the ReportedEmail that triggered it.
type EmailAnalysis struct {
	Id               int64            `json:"id" gorm:"primary_key;auto_increment"`
	OrgId            int64            `json:"org_id"`
	ReportedEmailId  int64            `json:"reported_email_id"`
	Status           string           `json:"status" gorm:"default:'pending'"`
	ThreatLevel      string           `json:"threat_level"`
	ConfidenceScore  float64          `json:"confidence_score"`
	Classification   string           `json:"classification"`
	Summary          string           `json:"summary" gorm:"type:text"`
	AIProvider       string           `json:"ai_provider"`
	TokensUsed       int              `json:"tokens_used"`
	AnalysisDuration int              `json:"analysis_duration"`
	Indicators       []EmailIndicator `json:"indicators" gorm:"-"`
	CreatedDate      time.Time        `json:"created_date"`
	CompletedDate    time.Time        `json:"completed_date"`
}

// TableName overrides the default table name.
func (EmailAnalysis) TableName() string {
	return "email_analyses"
}

// EmailIndicator represents a single threat indicator extracted during
// the NLP analysis of a reported email (e.g., a suspicious URL, an
// urgency cue, or a header anomaly).
type EmailIndicator struct {
	Id          int64  `json:"id" gorm:"primary_key;auto_increment"`
	AnalysisId  int64  `json:"analysis_id"`
	Type        string `json:"type"`
	Value       string `json:"value"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	SortOrder   int    `json:"sort_order"`
}

// TableName overrides the default table name.
func (EmailIndicator) TableName() string {
	return "email_indicators"
}

// EmailAnalysisSummary holds aggregate statistics for email analyses
// within an organization.
type EmailAnalysisSummary struct {
	TotalAnalyzed     int     `json:"total_analyzed"`
	Pending           int     `json:"pending"`
	PhishingDetected  int     `json:"phishing_detected"`
	SpamDetected      int     `json:"spam_detected"`
	LegitimateDetected int   `json:"legitimate_detected"`
	AvgConfidence     float64 `json:"avg_confidence"`
	HighThreatCount   int     `json:"high_threat_count"`
}

// AnalyzeReportedEmail is the main entry point for running an AI-powered
// analysis on a reported email. It creates an analysis record, calls the
// AI provider, parses the response into indicators, and updates the record
// with the results. If the AI call fails the analysis status is set to
// "failed" and the error is returned.
func AnalyzeReportedEmail(orgId int64, reportedEmailId int64, emailHeaders, emailBody, senderEmail, subject string, aiClient ai.Client) (*EmailAnalysis, error) {
	// Check for an existing analysis for this reported email
	var existing EmailAnalysis
	err := db.Where("reported_email_id = ? AND org_id = ?", reportedEmailId, orgId).First(&existing).Error
	if err == nil {
		// An analysis already exists
		if existing.Status == AnalysisStatusAnalyzing || existing.Status == AnalysisStatusPending {
			return nil, ErrAnalysisInProgress
		}
		// If a previous analysis completed or failed, allow re-analysis by
		// removing the old record and its indicators.
		db.Where("analysis_id = ?", existing.Id).Delete(&EmailIndicator{})
		db.Delete(&existing)
	} else if err != gorm.ErrRecordNotFound {
		return nil, err
	}

	// Create analysis record in "analyzing" state
	analysis := EmailAnalysis{
		OrgId:           orgId,
		ReportedEmailId: reportedEmailId,
		Status:          AnalysisStatusAnalyzing,
		AIProvider:      aiClient.Provider(),
		CreatedDate:     time.Now().UTC(),
	}
	if err := db.Create(&analysis).Error; err != nil {
		return nil, err
	}

	// Build prompts and call the AI provider
	startTime := time.Now()
	systemPrompt := ai.EmailAnalysisSystemPrompt
	userPrompt := ai.BuildEmailAnalysisPrompt(emailHeaders, emailBody, senderEmail, subject)

	resp, err := aiClient.Generate(systemPrompt, userPrompt)
	duration := time.Since(startTime).Milliseconds()

	if err != nil {
		log.Errorf("email analysis AI call failed for reported_email %d: %v", reportedEmailId, err)
		analysis.Status = AnalysisStatusFailed
		analysis.AnalysisDuration = int(duration)
		analysis.CompletedDate = time.Now().UTC()
		db.Save(&analysis)
		return &analysis, err
	}

	// Parse the AI response
	result, err := ai.ParseEmailAnalysisResponse(resp.Content)
	if err != nil {
		log.Errorf("email analysis parse failed for reported_email %d: %v", reportedEmailId, err)
		analysis.Status = AnalysisStatusFailed
		analysis.AnalysisDuration = int(duration)
		analysis.CompletedDate = time.Now().UTC()
		db.Save(&analysis)
		return &analysis, err
	}

	// Update analysis record with results
	analysis.Status = AnalysisStatusCompleted
	analysis.ThreatLevel = result.ThreatLevel
	analysis.ConfidenceScore = result.Confidence
	analysis.Classification = result.Classification
	analysis.Summary = result.Summary
	analysis.TokensUsed = resp.InputTokens + resp.OutputTokens
	analysis.AnalysisDuration = int(duration)
	analysis.CompletedDate = time.Now().UTC()

	if err := db.Save(&analysis).Error; err != nil {
		return nil, err
	}

	// Create indicator records
	for i, ind := range result.Indicators {
		indicator := EmailIndicator{
			AnalysisId:  analysis.Id,
			Type:        ind.Type,
			Value:       ind.Value,
			Severity:    ind.Severity,
			Description: ind.Description,
			SortOrder:   i,
		}
		if err := db.Create(&indicator).Error; err != nil {
			log.Errorf("failed to create indicator for analysis %d: %v", analysis.Id, err)
			continue
		}
		analysis.Indicators = append(analysis.Indicators, indicator)
	}

	log.Infof("email analysis completed for reported_email %d: threat_level=%s confidence=%.2f classification=%s (%d indicators)",
		reportedEmailId, analysis.ThreatLevel, analysis.ConfidenceScore, analysis.Classification, len(analysis.Indicators))

	return &analysis, nil
}

// GetEmailAnalysis returns a single email analysis by ID, scoped to the
// given organization. Indicators are hydrated automatically.
func GetEmailAnalysis(id, orgId int64) (EmailAnalysis, error) {
	var analysis EmailAnalysis
	err := db.Where("id = ? AND org_id = ?", id, orgId).First(&analysis).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return analysis, ErrAnalysisNotFound
		}
		return analysis, err
	}
	hydrateEmailAnalysis(&analysis)
	return analysis, nil
}

// GetEmailAnalysisByReportedEmail returns the analysis associated with a
// specific reported email, scoped to the given organization.
func GetEmailAnalysisByReportedEmail(reportedEmailId, orgId int64) (EmailAnalysis, error) {
	var analysis EmailAnalysis
	err := db.Where("reported_email_id = ? AND org_id = ?", reportedEmailId, orgId).First(&analysis).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return analysis, ErrAnalysisNotFound
		}
		return analysis, err
	}
	hydrateEmailAnalysis(&analysis)
	return analysis, nil
}

// GetEmailAnalyses returns all email analyses for an organization, newest
// first. Each analysis includes its indicators.
func GetEmailAnalyses(orgId int64) ([]EmailAnalysis, error) {
	var analyses []EmailAnalysis
	err := db.Where("org_id = ?", orgId).Order("created_date DESC").Find(&analyses).Error
	if err != nil {
		return nil, err
	}
	for i := range analyses {
		hydrateEmailAnalysis(&analyses[i])
	}
	return analyses, nil
}

// GetEmailAnalysisSummary returns aggregate statistics for all email
// analyses within an organization.
func GetEmailAnalysisSummary(orgId int64) (EmailAnalysisSummary, error) {
	var summary EmailAnalysisSummary

	// Total completed analyses
	db.Model(&EmailAnalysis{}).
		Where("org_id = ? AND status = ?", orgId, AnalysisStatusCompleted).
		Count(&summary.TotalAnalyzed)

	// Pending analyses
	db.Model(&EmailAnalysis{}).
		Where("org_id = ? AND (status = ? OR status = ?)", orgId, AnalysisStatusPending, AnalysisStatusAnalyzing).
		Count(&summary.Pending)

	// Phishing detected (phishing + spear_phishing + bec)
	db.Model(&EmailAnalysis{}).
		Where("org_id = ? AND status = ? AND classification IN (?)",
			orgId, AnalysisStatusCompleted,
			[]string{ClassificationPhishing, ClassificationSpearPhishing, ClassificationBEC}).
		Count(&summary.PhishingDetected)

	// Spam detected
	db.Model(&EmailAnalysis{}).
		Where("org_id = ? AND status = ? AND classification = ?",
			orgId, AnalysisStatusCompleted, ClassificationSpam).
		Count(&summary.SpamDetected)

	// Legitimate detected
	db.Model(&EmailAnalysis{}).
		Where("org_id = ? AND status = ? AND classification = ?",
			orgId, AnalysisStatusCompleted, ClassificationLegitimate).
		Count(&summary.LegitimateDetected)

	// Average confidence score
	row := db.Model(&EmailAnalysis{}).
		Where("org_id = ? AND status = ?", orgId, AnalysisStatusCompleted).
		Select("COALESCE(AVG(confidence_score), 0)").Row()
	row.Scan(&summary.AvgConfidence)

	// High threat count (likely_phishing + confirmed_phishing)
	db.Model(&EmailAnalysis{}).
		Where("org_id = ? AND status = ? AND threat_level IN (?)",
			orgId, AnalysisStatusCompleted,
			[]string{ThreatLevelLikelyPhishing, ThreatLevelConfirmedPhishing}).
		Count(&summary.HighThreatCount)

	return summary, nil
}

// hydrateEmailAnalysis loads the associated indicators for an analysis
// record from the database.
func hydrateEmailAnalysis(a *EmailAnalysis) {
	var indicators []EmailIndicator
	err := db.Where("analysis_id = ?", a.Id).Order("sort_order ASC").Find(&indicators).Error
	if err != nil {
		log.Errorf("failed to hydrate indicators for analysis %d: %v", a.Id, err)
		a.Indicators = []EmailIndicator{}
		return
	}
	a.Indicators = indicators
}
