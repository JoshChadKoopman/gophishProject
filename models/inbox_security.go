package models

import (
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"time"

	log "github.com/gophish/gophish/logger"
)

// Shared query-clause constants to avoid literal duplication.
const (
	qWhereOrg       = "org_id = ?"
	qWhereIdOrg     = "id = ? AND org_id = ?"
	qOrderCreated   = "created_date DESC"
	qWhereOrgStatus = "org_id = ? AND status = ?"
)

// ────────────────────────────────────────────────────────────────
// Feature flags
// ────────────────────────────────────────────────────────────────

// (registered in tier.go alongside the other feature constants)

// ────────────────────────────────────────────────────────────────
// Inbox Monitor Configuration
// ────────────────────────────────────────────────────────────────

// InboxMonitorConfig stores per-org settings for real-time inbox monitoring.
type InboxMonitorConfig struct {
	Id                     int64     `json:"id" gorm:"primary_key;auto_increment"`
	OrgId                  int64     `json:"org_id" gorm:"unique"`
	Enabled                bool      `json:"enabled"`
	ScanIntervalSeconds    int       `json:"scan_interval_seconds" gorm:"default:300"`
	MonitoredMailboxes     string    `json:"monitored_mailboxes" gorm:"type:text"` // JSON array
	ThreatThreshold        string    `json:"threat_threshold" gorm:"default:'suspicious'"`
	AutoQuarantine         bool      `json:"auto_quarantine"`
	AutoDelete             bool      `json:"auto_delete"`
	NotifyAdmin            bool      `json:"notify_admin" gorm:"default:true"`
	NotifyUser             bool      `json:"notify_user"`
	IMAPHost               string    `json:"imap_host"`
	IMAPPort               int       `json:"imap_port" gorm:"default:993"`
	IMAPUsername           string    `json:"imap_username"`
	IMAPPassword           string    `json:"imap_password"`
	IMAPTLS                bool      `json:"imap_tls" gorm:"default:true"`
	GoogleWorkspaceEnabled bool      `json:"google_workspace_enabled"`
	GoogleAdminEmail       string    `json:"google_admin_email"`
	MS365Enabled           bool      `json:"ms365_enabled"`
	MS365TenantId          string    `json:"ms365_tenant_id"`
	MS365ClientId          string    `json:"ms365_client_id"`
	MS365ClientSecret      string    `json:"ms365_client_secret"`
	LastScanDate           time.Time `json:"last_scan_date"`
	CreatedDate            time.Time `json:"created_date"`
	ModifiedDate           time.Time `json:"modified_date"`
}

func (InboxMonitorConfig) TableName() string { return "inbox_monitor_configs" }

// GetMonitoredMailboxList parses the JSON array of monitored mailboxes.
func (c *InboxMonitorConfig) GetMonitoredMailboxList() []string {
	var list []string
	if c.MonitoredMailboxes != "" {
		json.Unmarshal([]byte(c.MonitoredMailboxes), &list)
	}
	return list
}

// SetMonitoredMailboxList serializes the list of mailboxes into JSON.
func (c *InboxMonitorConfig) SetMonitoredMailboxList(list []string) {
	data, _ := json.Marshal(list)
	c.MonitoredMailboxes = string(data)
}

func GetInboxMonitorConfig(orgId int64) (InboxMonitorConfig, error) {
	var config InboxMonitorConfig
	err := db.Where("org_id = ?", orgId).First(&config).Error
	return config, err
}

func SaveInboxMonitorConfig(config *InboxMonitorConfig) error {
	config.ModifiedDate = time.Now().UTC()
	if config.Id == 0 {
		config.CreatedDate = time.Now().UTC()
		return db.Create(config).Error
	}
	return db.Save(config).Error
}

func GetAllEnabledMonitorConfigs() ([]InboxMonitorConfig, error) {
	var configs []InboxMonitorConfig
	err := db.Where("enabled = ?", true).Find(&configs).Error
	return configs, err
}

func UpdateMonitorLastScan(orgId int64) error {
	return db.Model(&InboxMonitorConfig{}).
		Where("org_id = ?", orgId).
		Update("last_scan_date", time.Now().UTC()).Error
}

// ────────────────────────────────────────────────────────────────
// Inbox Scan Results
// ────────────────────────────────────────────────────────────────

// Scan action constants
const (
	ScanActionNone        = "none"
	ScanActionFlagged     = "flagged"
	ScanActionQuarantined = "quarantined"
	ScanActionDeleted     = "deleted"
	ScanActionNotified    = "notified"
)

// InboxScanResult represents a single email scanned by the inbox monitor.
type InboxScanResult struct {
	Id               int64     `json:"id" gorm:"primary_key;auto_increment"`
	OrgId            int64     `json:"org_id"`
	ConfigId         int64     `json:"config_id"`
	MailboxEmail     string    `json:"mailbox_email"`
	MessageId        string    `json:"message_id"`
	SenderEmail      string    `json:"sender_email"`
	Subject          string    `json:"subject"`
	ReceivedDate     time.Time `json:"received_date"`
	ThreatLevel      string    `json:"threat_level" gorm:"default:'safe'"`
	Classification   string    `json:"classification" gorm:"default:'unknown'"`
	ConfidenceScore  float64   `json:"confidence_score"`
	IsBEC            bool      `json:"is_bec"`
	IsGraymail       bool      `json:"is_graymail"`
	GraymailCategory string    `json:"graymail_category"`
	Summary          string    `json:"summary" gorm:"type:text"`
	Indicators       string    `json:"indicators" gorm:"type:text"` // JSON
	ActionTaken      string    `json:"action_taken" gorm:"default:'none'"`
	ScanDurationMs   int       `json:"scan_duration_ms"`
	CreatedDate      time.Time `json:"created_date"`
}

func (InboxScanResult) TableName() string { return "inbox_scan_results" }

// GetIndicatorsList parses the JSON indicators string into a slice.
func (r *InboxScanResult) GetIndicatorsList() []EmailIndicator {
	var indicators []EmailIndicator
	if r.Indicators != "" {
		json.Unmarshal([]byte(r.Indicators), &indicators)
	}
	return indicators
}

func CreateInboxScanResult(r *InboxScanResult) error {
	r.CreatedDate = time.Now().UTC()
	return db.Create(r).Error
}

func GetInboxScanResults(orgId int64, limit int) ([]InboxScanResult, error) {
	var results []InboxScanResult
	q := db.Where("org_id = ?", orgId).Order("created_date DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&results).Error
	return results, err
}

func GetInboxScanResult(id, orgId int64) (InboxScanResult, error) {
	var result InboxScanResult
	err := db.Where("id = ? AND org_id = ?", id, orgId).First(&result).Error
	return result, err
}

func GetInboxScanResultsByThreat(orgId int64, threatLevel string) ([]InboxScanResult, error) {
	var results []InboxScanResult
	err := db.Where("org_id = ? AND threat_level = ?", orgId, threatLevel).
		Order("created_date DESC").Find(&results).Error
	return results, err
}

// InboxScanSummary holds aggregate stats for the inbox monitor dashboard.
type InboxScanSummary struct {
	TotalScanned     int     `json:"total_scanned"`
	ThreatsDetected  int     `json:"threats_detected"`
	BECDetected      int     `json:"bec_detected"`
	GraymailDetected int     `json:"graymail_detected"`
	Quarantined      int     `json:"quarantined"`
	Deleted          int     `json:"deleted"`
	SafeEmails       int     `json:"safe_emails"`
	AvgConfidence    float64 `json:"avg_confidence"`
	LastScanDate     string  `json:"last_scan_date"`
}

func GetInboxScanSummary(orgId int64) (InboxScanSummary, error) {
	var summary InboxScanSummary

	db.Model(&InboxScanResult{}).Where("org_id = ?", orgId).Count(&summary.TotalScanned)

	db.Model(&InboxScanResult{}).
		Where("org_id = ? AND threat_level IN (?)", orgId,
			[]string{ThreatLevelSuspicious, ThreatLevelLikelyPhishing, ThreatLevelConfirmedPhishing}).
		Count(&summary.ThreatsDetected)

	db.Model(&InboxScanResult{}).Where("org_id = ? AND is_bec = ?", orgId, true).Count(&summary.BECDetected)
	db.Model(&InboxScanResult{}).Where("org_id = ? AND is_graymail = ?", orgId, true).Count(&summary.GraymailDetected)
	db.Model(&InboxScanResult{}).Where("org_id = ? AND action_taken = ?", orgId, ScanActionQuarantined).Count(&summary.Quarantined)
	db.Model(&InboxScanResult{}).Where("org_id = ? AND action_taken = ?", orgId, ScanActionDeleted).Count(&summary.Deleted)
	db.Model(&InboxScanResult{}).Where("org_id = ? AND threat_level = ?", orgId, ThreatLevelSafe).Count(&summary.SafeEmails)

	row := db.Model(&InboxScanResult{}).
		Where("org_id = ? AND confidence_score > 0", orgId).
		Select("COALESCE(AVG(confidence_score), 0)").Row()
	row.Scan(&summary.AvgConfidence)

	config, err := GetInboxMonitorConfig(orgId)
	if err == nil && !config.LastScanDate.IsZero() {
		summary.LastScanDate = config.LastScanDate.Format(time.RFC3339)
	}

	return summary, nil
}

// ────────────────────────────────────────────────────────────────
// BEC Detection
// ────────────────────────────────────────────────────────────────

// BEC attack type constants
const (
	BECAttackCEOFraud            = "ceo_fraud"
	BECAttackInvoiceFraud        = "invoice_fraud"
	BECAttackAccountTakeover     = "account_takeover"
	BECAttackVendorImpersonation = "vendor_impersonation"
	BECAttackPayrollDiversion    = "payroll_diversion"
	BECAttackDataTheft           = "data_theft"
)

// BECProfile represents an executive/key person whose identity could be
// impersonated in a BEC attack.
type BECProfile struct {
	Id             int64     `json:"id" gorm:"primary_key;auto_increment"`
	OrgId          int64     `json:"org_id"`
	ExecutiveEmail string    `json:"executive_email"`
	ExecutiveName  string    `json:"executive_name"`
	Title          string    `json:"title"`
	Department     string    `json:"department"`
	KnownDomains   string    `json:"known_domains" gorm:"type:text"` // JSON array
	KnownSenders   string    `json:"known_senders" gorm:"type:text"` // JSON array
	IsActive       bool      `json:"is_active" gorm:"default:true"`
	CreatedDate    time.Time `json:"created_date"`
	ModifiedDate   time.Time `json:"modified_date"`
}

func (BECProfile) TableName() string { return "bec_profiles" }

func GetBECProfiles(orgId int64) ([]BECProfile, error) {
	var profiles []BECProfile
	err := db.Where("org_id = ? AND is_active = ?", orgId, true).Find(&profiles).Error
	return profiles, err
}

func GetBECProfile(id, orgId int64) (BECProfile, error) {
	var profile BECProfile
	err := db.Where("id = ? AND org_id = ?", id, orgId).First(&profile).Error
	return profile, err
}

func SaveBECProfile(p *BECProfile) error {
	p.ModifiedDate = time.Now().UTC()
	if p.Id == 0 {
		p.CreatedDate = time.Now().UTC()
		return db.Create(p).Error
	}
	return db.Save(p).Error
}

func DeleteBECProfile(id, orgId int64) error {
	return db.Where("id = ? AND org_id = ?", id, orgId).Delete(&BECProfile{}).Error
}

// BECDetection records a detected BEC attempt.
type BECDetection struct {
	Id                    int64     `json:"id" gorm:"primary_key;auto_increment"`
	OrgId                 int64     `json:"org_id"`
	ScanResultId          int64     `json:"scan_result_id"`
	ReportedEmailId       int64     `json:"reported_email_id"`
	ImpersonatedEmail     string    `json:"impersonated_email"`
	ImpersonatedName      string    `json:"impersonated_name"`
	ActualSender          string    `json:"actual_sender"`
	AttackType            string    `json:"attack_type"`
	UrgencyLevel          string    `json:"urgency_level" gorm:"default:'medium'"`
	FinancialRequest      bool      `json:"financial_request"`
	WireTransferMentioned bool      `json:"wire_transfer_mentioned"`
	GiftCardMentioned     bool      `json:"gift_card_mentioned"`
	ConfidenceScore       float64   `json:"confidence_score"`
	Summary               string    `json:"summary" gorm:"type:text"`
	ActionTaken           string    `json:"action_taken" gorm:"default:'flagged'"`
	Resolved              bool      `json:"resolved"`
	ResolvedBy            int64     `json:"resolved_by"`
	ResolvedDate          time.Time `json:"resolved_date"`
	CreatedDate           time.Time `json:"created_date"`
}

func (BECDetection) TableName() string { return "bec_detections" }

func CreateBECDetection(d *BECDetection) error {
	d.CreatedDate = time.Now().UTC()
	return db.Create(d).Error
}

func GetBECDetections(orgId int64, includeResolved bool) ([]BECDetection, error) {
	var detections []BECDetection
	q := db.Where("org_id = ?", orgId)
	if !includeResolved {
		q = q.Where("resolved = ?", false)
	}
	err := q.Order("created_date DESC").Find(&detections).Error
	return detections, err
}

func GetBECDetection(id, orgId int64) (BECDetection, error) {
	var d BECDetection
	err := db.Where("id = ? AND org_id = ?", id, orgId).First(&d).Error
	return d, err
}

func ResolveBECDetection(id, orgId, resolvedBy int64, action string) error {
	return db.Model(&BECDetection{}).
		Where("id = ? AND org_id = ?", id, orgId).
		Updates(map[string]interface{}{
			"resolved":      true,
			"resolved_by":   resolvedBy,
			"resolved_date": time.Now().UTC(),
			"action_taken":  action,
		}).Error
}

// BECDetectionSummary holds aggregate BEC stats.
type BECDetectionSummary struct {
	TotalDetections   int     `json:"total_detections"`
	UnresolvedCount   int     `json:"unresolved_count"`
	FinancialRequests int     `json:"financial_requests"`
	CEOFraudCount     int     `json:"ceo_fraud_count"`
	InvoiceFraudCount int     `json:"invoice_fraud_count"`
	AvgConfidence     float64 `json:"avg_confidence"`
}

func GetBECDetectionSummary(orgId int64) (BECDetectionSummary, error) {
	var s BECDetectionSummary
	db.Model(&BECDetection{}).Where("org_id = ?", orgId).Count(&s.TotalDetections)
	db.Model(&BECDetection{}).Where("org_id = ? AND resolved = ?", orgId, false).Count(&s.UnresolvedCount)
	db.Model(&BECDetection{}).Where("org_id = ? AND financial_request = ?", orgId, true).Count(&s.FinancialRequests)
	db.Model(&BECDetection{}).Where("org_id = ? AND attack_type = ?", orgId, BECAttackCEOFraud).Count(&s.CEOFraudCount)
	db.Model(&BECDetection{}).Where("org_id = ? AND attack_type = ?", orgId, BECAttackInvoiceFraud).Count(&s.InvoiceFraudCount)
	row := db.Model(&BECDetection{}).Where("org_id = ?", orgId).
		Select("COALESCE(AVG(confidence_score), 0)").Row()
	row.Scan(&s.AvgConfidence)
	return s, nil
}

// ────────────────────────────────────────────────────────────────
// Graymail Classification
// ────────────────────────────────────────────────────────────────

// Graymail category constants
const (
	GraymailNewsletter    = "newsletter"
	GraymailMarketing     = "marketing"
	GraymailNotification  = "notification"
	GraymailSocialMedia   = "social_media"
	GraymailBulkPromo     = "bulk_promo"
	GraymailAutoGenerated = "auto_generated"
)

// Graymail rule type constants
const (
	GraymailRuleSender  = "sender_pattern"
	GraymailRuleSubject = "subject_pattern"
	GraymailRuleHeader  = "header_pattern"
	GraymailRuleAI      = "ai_classification"
)

// GraymailRule is an org-defined rule for classifying graymail.
type GraymailRule struct {
	Id           int64     `json:"id" gorm:"primary_key;auto_increment"`
	OrgId        int64     `json:"org_id"`
	RuleType     string    `json:"rule_type"`
	Pattern      string    `json:"pattern"`
	Category     string    `json:"category"`
	Action       string    `json:"action" gorm:"default:'label'"`
	IsActive     bool      `json:"is_active" gorm:"default:true"`
	MatchCount   int       `json:"match_count"`
	CreatedDate  time.Time `json:"created_date"`
	ModifiedDate time.Time `json:"modified_date"`
}

func (GraymailRule) TableName() string { return "graymail_rules" }

func GetGraymailRules(orgId int64) ([]GraymailRule, error) {
	var rules []GraymailRule
	err := db.Where("org_id = ?", orgId).Order("created_date DESC").Find(&rules).Error
	return rules, err
}

func SaveGraymailRule(r *GraymailRule) error {
	r.ModifiedDate = time.Now().UTC()
	if r.Id == 0 {
		r.CreatedDate = time.Now().UTC()
		return db.Create(r).Error
	}
	return db.Save(r).Error
}

func DeleteGraymailRule(id, orgId int64) error {
	return db.Where("id = ? AND org_id = ?", id, orgId).Delete(&GraymailRule{}).Error
}

// GraymailClassification records a graymail classification result.
type GraymailClassification struct {
	Id              int64     `json:"id" gorm:"primary_key;auto_increment"`
	OrgId           int64     `json:"org_id"`
	ScanResultId    int64     `json:"scan_result_id"`
	EmailSubject    string    `json:"email_subject"`
	SenderEmail     string    `json:"sender_email"`
	Category        string    `json:"category"`
	Subcategory     string    `json:"subcategory"`
	ConfidenceScore float64   `json:"confidence_score"`
	ActionTaken     string    `json:"action_taken" gorm:"default:'labeled'"`
	CreatedDate     time.Time `json:"created_date"`
}

func (GraymailClassification) TableName() string { return "graymail_classifications" }

func CreateGraymailClassification(c *GraymailClassification) error {
	c.CreatedDate = time.Now().UTC()
	return db.Create(c).Error
}

func GetGraymailClassifications(orgId int64, limit int) ([]GraymailClassification, error) {
	var results []GraymailClassification
	q := db.Where("org_id = ?", orgId).Order("created_date DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&results).Error
	return results, err
}

// GraymailSummary holds aggregate graymail stats.
type GraymailSummary struct {
	TotalClassified int            `json:"total_classified"`
	ByCategory      map[string]int `json:"by_category"`
}

func GetGraymailSummary(orgId int64) (GraymailSummary, error) {
	s := GraymailSummary{ByCategory: make(map[string]int)}
	db.Model(&GraymailClassification{}).Where("org_id = ?", orgId).Count(&s.TotalClassified)

	type catCount struct {
		Category string
		Count    int
	}
	var counts []catCount
	db.Model(&GraymailClassification{}).
		Where("org_id = ?", orgId).
		Select("category, COUNT(*) as count").
		Group("category").
		Scan(&counts)
	for _, c := range counts {
		s.ByCategory[c.Category] = c.Count
	}
	return s, nil
}

// ────────────────────────────────────────────────────────────────
// One-Click Remediation
// ────────────────────────────────────────────────────────────────

// Remediation action types
const (
	RemediationActionDelete     = "delete"
	RemediationActionQuarantine = "quarantine"
	RemediationActionBlock      = "block_sender"
	RemediationActionPurge      = "purge_org_wide"
	RemediationActionRestore    = "restore"
)

// Remediation target types
const (
	RemediationTargetScanResult    = "scan_result"
	RemediationTargetReportedEmail = "reported_email"
	RemediationTargetBECDetection  = "bec_detection"
	RemediationTargetManual        = "manual"
)

// Inbox remediation status constants (prefixed to avoid collisions with
// remediation_path.go constants which use the same RemediationStatus* names).
const (
	InboxRemStatusPending   = "pending"
	InboxRemStatusApproved  = "approved"
	InboxRemStatusExecuting = "executing"
	InboxRemStatusCompleted = "completed"
	InboxRemStatusFailed    = "failed"
	InboxRemStatusRejected  = "rejected"
)

// Remediation scope constants
const (
	RemediationScopeSingle  = "single"
	RemediationScopeUser    = "user"
	RemediationScopeOrgWide = "org_wide"
)

// RemediationAction records an action taken to remove a threat from inbox(es).
type RemediationAction struct {
	Id                int64     `json:"id" gorm:"primary_key;auto_increment"`
	OrgId             int64     `json:"org_id"`
	ActionType        string    `json:"action_type"`
	TargetType        string    `json:"target_type"`
	TargetId          int64     `json:"target_id"`
	TargetEmail       string    `json:"target_email"`
	MessageId         string    `json:"message_id"`
	Subject           string    `json:"subject"`
	SenderEmail       string    `json:"sender_email"`
	Status            string    `json:"status" gorm:"default:'pending'"`
	ResultMessage     string    `json:"result_message" gorm:"type:text"`
	InitiatedBy       int64     `json:"initiated_by"`
	ApprovedBy        int64     `json:"approved_by"`
	RequiresApproval  bool      `json:"requires_approval"`
	Scope             string    `json:"scope" gorm:"default:'single'"`
	AffectedMailboxes int       `json:"affected_mailboxes"`
	CreatedDate       time.Time `json:"created_date"`
	CompletedDate     time.Time `json:"completed_date"`
}

func (RemediationAction) TableName() string { return "remediation_actions" }

var ErrInboxRemediationNotFound = errors.New("inbox remediation action not found")

func CreateRemediationAction(a *RemediationAction) error {
	a.CreatedDate = time.Now().UTC()
	if a.Status == "" {
		a.Status = InboxRemStatusPending
	}
	return db.Create(a).Error
}

func GetRemediationActions(orgId int64, limit int) ([]RemediationAction, error) {
	var actions []RemediationAction
	q := db.Where("org_id = ?", orgId).Order("created_date DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	err := q.Find(&actions).Error
	return actions, err
}

func GetRemediationAction(id, orgId int64) (RemediationAction, error) {
	var a RemediationAction
	err := db.Where("id = ? AND org_id = ?", id, orgId).First(&a).Error
	if err != nil {
		return a, ErrInboxRemediationNotFound
	}
	return a, nil
}

func GetPendingRemediationActions() ([]RemediationAction, error) {
	var actions []RemediationAction
	err := db.Where("status = ?", InboxRemStatusPending).Find(&actions).Error
	return actions, err
}

func UpdateRemediationStatus(id int64, status, resultMsg string, affected int) error {
	updates := map[string]interface{}{
		"status":             status,
		"result_message":     resultMsg,
		"affected_mailboxes": affected,
	}
	if status == InboxRemStatusCompleted || status == InboxRemStatusFailed {
		updates["completed_date"] = time.Now().UTC()
	}
	return db.Model(&RemediationAction{}).Where("id = ?", id).Updates(updates).Error
}

func ApproveRemediation(id, orgId, approvedBy int64) error {
	return db.Model(&RemediationAction{}).
		Where("id = ? AND org_id = ? AND status = ?", id, orgId, InboxRemStatusPending).
		Updates(map[string]interface{}{
			"status":      InboxRemStatusApproved,
			"approved_by": approvedBy,
		}).Error
}

func RejectRemediation(id, orgId int64, reason string) error {
	return db.Model(&RemediationAction{}).
		Where("id = ? AND org_id = ? AND status = ?", id, orgId, InboxRemStatusPending).
		Updates(map[string]interface{}{
			"status":         InboxRemStatusRejected,
			"result_message": reason,
			"completed_date": time.Now().UTC(),
		}).Error
}

// RemediationSummaryStats holds remediation statistics.
type RemediationSummaryStats struct {
	TotalActions  int `json:"total_actions"`
	Completed     int `json:"completed"`
	Pending       int `json:"pending"`
	Failed        int `json:"failed"`
	TotalAffected int `json:"total_affected_mailboxes"`
}

func GetRemediationSummaryStats(orgId int64) (RemediationSummaryStats, error) {
	var s RemediationSummaryStats
	db.Model(&RemediationAction{}).Where("org_id = ?", orgId).Count(&s.TotalActions)
	db.Model(&RemediationAction{}).Where("org_id = ? AND status = ?", orgId, InboxRemStatusCompleted).Count(&s.Completed)
	db.Model(&RemediationAction{}).Where("org_id = ? AND status = ?", orgId, InboxRemStatusPending).Count(&s.Pending)
	db.Model(&RemediationAction{}).Where("org_id = ? AND status = ?", orgId, InboxRemStatusFailed).Count(&s.Failed)

	row := db.Model(&RemediationAction{}).Where("org_id = ?", orgId).
		Select("COALESCE(SUM(affected_mailboxes), 0)").Row()
	row.Scan(&s.TotalAffected)
	return s, nil
}

// ────────────────────────────────────────────────────────────────
// Phishing Ticket Management
// ────────────────────────────────────────────────────────────────

// Ticket status constants
const (
	TicketStatusOpen         = "open"
	TicketStatusInProgress   = "in_progress"
	TicketStatusResolved     = "resolved"
	TicketStatusAutoResolved = "auto_resolved"
	TicketStatusClosed       = "closed"
	TicketStatusEscalated    = "escalated"
)

// PhishingTicket represents a security incident ticket.
type PhishingTicket struct {
	Id                   int64     `json:"id" gorm:"primary_key;auto_increment"`
	OrgId                int64     `json:"org_id"`
	ReportedEmailId      int64     `json:"reported_email_id"`
	ScanResultId         int64     `json:"scan_result_id"`
	BECDetectionId       int64     `json:"bec_detection_id"`
	TicketNumber         string    `json:"ticket_number"`
	Title                string    `json:"title"`
	Description          string    `json:"description" gorm:"type:text"`
	Severity             string    `json:"severity" gorm:"default:'medium'"`
	Status               string    `json:"status" gorm:"default:'open'"`
	Classification       string    `json:"classification" gorm:"default:'pending'"`
	AutoResolved         bool      `json:"auto_resolved"`
	AutoResolutionReason string    `json:"auto_resolution_reason" gorm:"type:text"`
	AssignedTo           int64     `json:"assigned_to"`
	Escalated            bool      `json:"escalated"`
	EscalatedTo          int64     `json:"escalated_to"`
	ResolutionNotes      string    `json:"resolution_notes" gorm:"type:text"`
	SLADeadline          time.Time `json:"sla_deadline"`
	CreatedDate          time.Time `json:"created_date"`
	UpdatedDate          time.Time `json:"updated_date"`
	ResolvedDate         time.Time `json:"resolved_date"`
}

func (PhishingTicket) TableName() string { return "phishing_tickets" }

var ErrTicketNotFound = errors.New("phishing ticket not found")

// GenerateTicketNumber creates a unique ticket number.
func GenerateTicketNumber() string {
	return fmt.Sprintf("SEC-%d-%04d", time.Now().Year(), rand.Intn(10000))
}

func CreatePhishingTicket(t *PhishingTicket) error {
	t.CreatedDate = time.Now().UTC()
	t.UpdatedDate = time.Now().UTC()
	if t.TicketNumber == "" {
		t.TicketNumber = GenerateTicketNumber()
	}
	if t.SLADeadline.IsZero() {
		// Default SLA: 4 hours for critical, 8 hours for high, 24 hours otherwise
		switch t.Severity {
		case SeverityCritical:
			t.SLADeadline = t.CreatedDate.Add(4 * time.Hour)
		case SeverityHigh:
			t.SLADeadline = t.CreatedDate.Add(8 * time.Hour)
		default:
			t.SLADeadline = t.CreatedDate.Add(24 * time.Hour)
		}
	}
	return db.Create(t).Error
}

func GetPhishingTickets(orgId int64, statusFilter string) ([]PhishingTicket, error) {
	var tickets []PhishingTicket
	q := db.Where("org_id = ?", orgId)
	if statusFilter != "" {
		q = q.Where("status = ?", statusFilter)
	}
	err := q.Order("created_date DESC").Find(&tickets).Error
	return tickets, err
}

func GetPhishingTicket(id, orgId int64) (PhishingTicket, error) {
	var t PhishingTicket
	err := db.Where("id = ? AND org_id = ?", id, orgId).First(&t).Error
	if err != nil {
		return t, ErrTicketNotFound
	}
	return t, nil
}

func GetPhishingTicketByNumber(ticketNumber string, orgId int64) (PhishingTicket, error) {
	var t PhishingTicket
	err := db.Where("ticket_number = ? AND org_id = ?", ticketNumber, orgId).First(&t).Error
	if err != nil {
		return t, ErrTicketNotFound
	}
	return t, nil
}

func UpdatePhishingTicket(t *PhishingTicket) error {
	t.UpdatedDate = time.Now().UTC()
	return db.Save(t).Error
}

func ResolvePhishingTicket(id, orgId int64, notes string, autoResolved bool, reason string) error {
	updates := map[string]interface{}{
		"status":                 TicketStatusResolved,
		"resolution_notes":       notes,
		"auto_resolved":          autoResolved,
		"auto_resolution_reason": reason,
		"resolved_date":          time.Now().UTC(),
		"updated_date":           time.Now().UTC(),
	}
	if autoResolved {
		updates["status"] = TicketStatusAutoResolved
	}
	return db.Model(&PhishingTicket{}).
		Where("id = ? AND org_id = ?", id, orgId).
		Updates(updates).Error
}

func ClosePhishingTicket(id, orgId int64) error {
	return db.Model(&PhishingTicket{}).
		Where("id = ? AND org_id = ?", id, orgId).
		Updates(map[string]interface{}{
			"status":       TicketStatusClosed,
			"updated_date": time.Now().UTC(),
		}).Error
}

func EscalatePhishingTicket(id, orgId, escalatedTo int64) error {
	return db.Model(&PhishingTicket{}).
		Where("id = ? AND org_id = ?", id, orgId).
		Updates(map[string]interface{}{
			"status":       TicketStatusEscalated,
			"escalated":    true,
			"escalated_to": escalatedTo,
			"updated_date": time.Now().UTC(),
		}).Error
}

// PhishingTicketAutoRule defines automatic resolution rules.
type PhishingTicketAutoRule struct {
	Id             int64     `json:"id" gorm:"primary_key;auto_increment"`
	OrgId          int64     `json:"org_id"`
	RuleName       string    `json:"rule_name"`
	ConditionType  string    `json:"condition_type"` // classification, threat_level, is_simulation
	ConditionValue string    `json:"condition_value"`
	Action         string    `json:"action"` // auto_resolve, auto_close, escalate
	IsActive       bool      `json:"is_active" gorm:"default:true"`
	TriggersCount  int       `json:"triggers_count"`
	CreatedDate    time.Time `json:"created_date"`
	ModifiedDate   time.Time `json:"modified_date"`
}

func (PhishingTicketAutoRule) TableName() string { return "phishing_ticket_auto_rules" }

func GetPhishingTicketAutoRules(orgId int64) ([]PhishingTicketAutoRule, error) {
	var rules []PhishingTicketAutoRule
	err := db.Where("org_id = ? AND is_active = ?", orgId, true).Find(&rules).Error
	return rules, err
}

func SavePhishingTicketAutoRule(r *PhishingTicketAutoRule) error {
	r.ModifiedDate = time.Now().UTC()
	if r.Id == 0 {
		r.CreatedDate = time.Now().UTC()
		return db.Create(r).Error
	}
	return db.Save(r).Error
}

func DeletePhishingTicketAutoRule(id, orgId int64) error {
	return db.Where("id = ? AND org_id = ?", id, orgId).Delete(&PhishingTicketAutoRule{}).Error
}

// ApplyAutoRules evaluates a newly created ticket against auto-rules and
// resolves it if a matching rule is found. Returns true if the ticket was
// auto-resolved.
func ApplyAutoRules(ticket *PhishingTicket) bool {
	rules, err := GetPhishingTicketAutoRules(ticket.OrgId)
	if err != nil || len(rules) == 0 {
		return false
	}

	for _, rule := range rules {
		matched := false
		switch rule.ConditionType {
		case "classification":
			matched = ticket.Classification == rule.ConditionValue
		case "threat_level":
			// Check scan result threat level
			if ticket.ScanResultId > 0 {
				if sr, err := GetInboxScanResult(ticket.ScanResultId, ticket.OrgId); err == nil {
					matched = sr.ThreatLevel == rule.ConditionValue
				}
			}
		case "is_simulation":
			if ticket.ReportedEmailId > 0 {
				if re, err := GetReportedEmail(ticket.ReportedEmailId, ticket.OrgId); err == nil {
					matched = re.IsSimulation && rule.ConditionValue == "true"
				}
			}
		}

		if matched {
			switch rule.Action {
			case "auto_resolve":
				reason := fmt.Sprintf("Auto-resolved by rule '%s': %s = %s", rule.RuleName, rule.ConditionType, rule.ConditionValue)
				ResolvePhishingTicket(ticket.Id, ticket.OrgId, "", true, reason)
				ticket.AutoResolved = true
				ticket.Status = TicketStatusAutoResolved
			case "auto_close":
				ClosePhishingTicket(ticket.Id, ticket.OrgId)
				ticket.Status = TicketStatusClosed
			case "escalate":
				EscalatePhishingTicket(ticket.Id, ticket.OrgId, 0)
				ticket.Status = TicketStatusEscalated
			}
			// Increment trigger count
			db.Model(&PhishingTicketAutoRule{}).Where("id = ?", rule.Id).
				Update("triggers_count", rule.TriggersCount+1)
			log.Infof("auto-rule '%s' triggered for ticket %s", rule.RuleName, ticket.TicketNumber)
			return true
		}
	}
	return false
}

// PhishingTicketSummary holds aggregate ticket stats.
type PhishingTicketSummary struct {
	TotalTickets   int `json:"total_tickets"`
	OpenTickets    int `json:"open_tickets"`
	AutoResolved   int `json:"auto_resolved"`
	ManualResolved int `json:"manual_resolved"`
	Escalated      int `json:"escalated"`
	OverdueSLA     int `json:"overdue_sla"`
}

func GetPhishingTicketSummary(orgId int64) (PhishingTicketSummary, error) {
	var s PhishingTicketSummary
	db.Model(&PhishingTicket{}).Where("org_id = ?", orgId).Count(&s.TotalTickets)
	db.Model(&PhishingTicket{}).Where("org_id = ? AND status = ?", orgId, TicketStatusOpen).Count(&s.OpenTickets)
	db.Model(&PhishingTicket{}).Where("org_id = ? AND status = ?", orgId, TicketStatusAutoResolved).Count(&s.AutoResolved)
	db.Model(&PhishingTicket{}).Where("org_id = ? AND status = ? AND auto_resolved = ?", orgId, TicketStatusResolved, false).Count(&s.ManualResolved)
	db.Model(&PhishingTicket{}).Where("org_id = ? AND escalated = ?", orgId, true).Count(&s.Escalated)
	db.Model(&PhishingTicket{}).
		Where("org_id = ? AND status IN (?) AND sla_deadline < ?",
			orgId, []string{TicketStatusOpen, TicketStatusInProgress}, time.Now().UTC()).
		Count(&s.OverdueSLA)
	return s, nil
}

// ────────────────────────────────────────────────────────────────
// Orchestration: End-to-end email security pipeline
// ────────────────────────────────────────────────────────────────

// EmailSecurityPipeline orchestrates the full analysis flow:
// 1. AI-powered threat analysis
// 2. BEC detection
// 3. Graymail classification
// 4. Auto-remediation (if enabled)
// 5. Ticket creation/auto-resolution
// This is called by the inbox monitor worker and the report button webhook.
type EmailSecurityPipelineResult struct {
	Analysis     *EmailAnalysis          `json:"analysis,omitempty"`
	BECDetection *BECDetection           `json:"bec_detection,omitempty"`
	Graymail     *GraymailClassification `json:"graymail,omitempty"`
	Remediation  *RemediationAction      `json:"remediation,omitempty"`
	Ticket       *PhishingTicket         `json:"ticket,omitempty"`
	AutoResolved bool                    `json:"auto_resolved"`
}

// CreateTicketFromScanResult creates a ticket from an inbox scan result.
func CreateTicketFromScanResult(orgId int64, scan *InboxScanResult) (*PhishingTicket, error) {
	severity := SeverityLow
	switch scan.ThreatLevel {
	case ThreatLevelConfirmedPhishing:
		severity = SeverityCritical
	case ThreatLevelLikelyPhishing:
		severity = SeverityHigh
	case ThreatLevelSuspicious:
		severity = SeverityMedium
	}

	title := fmt.Sprintf("[%s] %s from %s", scan.Classification, scan.Subject, scan.SenderEmail)
	if scan.IsBEC {
		title = "[BEC] " + title
	}

	ticket := &PhishingTicket{
		OrgId:          orgId,
		ScanResultId:   scan.Id,
		TicketNumber:   GenerateTicketNumber(),
		Title:          title,
		Description:    scan.Summary,
		Severity:       severity,
		Classification: scan.Classification,
	}
	if err := CreatePhishingTicket(ticket); err != nil {
		return nil, err
	}

	// Try auto-resolution
	ApplyAutoRules(ticket)
	return ticket, nil
}

// CreateTicketFromReportedEmail creates a ticket from a reported email.
func CreateTicketFromReportedEmail(orgId int64, re *ReportedEmail, analysis *EmailAnalysis) (*PhishingTicket, error) {
	severity := SeverityMedium
	classification := re.Classification
	if analysis != nil {
		classification = analysis.Classification
		switch analysis.ThreatLevel {
		case ThreatLevelConfirmedPhishing:
			severity = SeverityCritical
		case ThreatLevelLikelyPhishing:
			severity = SeverityHigh
		}
	}

	ticket := &PhishingTicket{
		OrgId:           orgId,
		ReportedEmailId: re.Id,
		TicketNumber:    GenerateTicketNumber(),
		Title:           fmt.Sprintf("Reported: %s from %s", re.Subject, re.SenderEmail),
		Description:     fmt.Sprintf("Email reported by %s", re.ReporterEmail),
		Severity:        severity,
		Classification:  classification,
	}
	if err := CreatePhishingTicket(ticket); err != nil {
		return nil, err
	}

	ApplyAutoRules(ticket)
	return ticket, nil
}
