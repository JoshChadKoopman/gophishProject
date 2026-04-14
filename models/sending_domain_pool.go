package models

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	log "github.com/gophish/gophish/logger"
)

// ── Sending Domain Pool ─────────────────────────────────────────
// Pre-configured realistic spoofing domains that can be randomly
// selected for phishing campaigns. Provides realistic-looking sender
// domains, automatic rotation, warm-up tracking, and health monitoring.

// SendingDomain represents a pre-configured or custom domain for phishing simulations.
type SendingDomain struct {
	Id              int64     `json:"id" gorm:"primary_key"`
	OrgId           int64     `json:"org_id" gorm:"column:org_id"`
	Domain          string    `json:"domain" gorm:"column:domain"`
	DisplayName     string    `json:"display_name" gorm:"column:display_name"`
	Category        string    `json:"category" gorm:"column:category"` // "corporate", "cloud", "financial", "shipping", "social", "custom"
	IsBuiltIn       bool      `json:"is_built_in" gorm:"column:is_built_in"`
	IsActive        bool      `json:"is_active" gorm:"column:is_active;default:true"`
	SPFConfigured   bool      `json:"spf_configured" gorm:"column:spf_configured"`
	DKIMConfigured  bool      `json:"dkim_configured" gorm:"column:dkim_configured"`
	DMARCConfigured bool      `json:"dmarc_configured" gorm:"column:dmarc_configured"`
	WarmupStage     int       `json:"warmup_stage" gorm:"column:warmup_stage;default:0"` // 0=cold, 1-5=warming, 6=ready
	DailyLimit      int       `json:"daily_limit" gorm:"column:daily_limit;default:50"`
	SendsToday      int       `json:"sends_today" gorm:"column:sends_today;default:0"`
	TotalSent       int64     `json:"total_sent" gorm:"column:total_sent;default:0"`
	LastUsedDate    time.Time `json:"last_used_date,omitempty" gorm:"column:last_used_date"`
	HealthStatus    string    `json:"health_status" gorm:"column:health_status;default:'unknown'"` // "healthy", "warning", "blacklisted", "unknown"
	LastHealthCheck time.Time `json:"last_health_check,omitempty" gorm:"column:last_health_check"`
	Notes           string    `json:"notes,omitempty" gorm:"column:notes;type:text"`
	CreatedDate     time.Time `json:"created_date" gorm:"column:created_date"`
	ModifiedDate    time.Time `json:"modified_date" gorm:"column:modified_date"`
}

// DomainPoolConfig stores org-level domain pool preferences.
type DomainPoolConfig struct {
	Id               int64  `json:"id" gorm:"primary_key"`
	OrgId            int64  `json:"org_id" gorm:"column:org_id;unique_index"`
	Enabled          bool   `json:"enabled" gorm:"column:enabled;default:true"`
	AutoRotate       bool   `json:"auto_rotate" gorm:"column:auto_rotate;default:true"`
	RotationStrategy string `json:"rotation_strategy" gorm:"column:rotation_strategy;default:'round_robin'"` // "round_robin", "random", "weighted"
	MaxDailyPerDomain int   `json:"max_daily_per_domain" gorm:"column:max_daily_per_domain;default:50"`
	WarmupEnabled    bool   `json:"warmup_enabled" gorm:"column:warmup_enabled;default:true"`
}

// DomainPoolSummary shows aggregate pool statistics.
type DomainPoolSummary struct {
	TotalDomains    int                       `json:"total_domains"`
	ActiveDomains   int                       `json:"active_domains"`
	HealthyDomains  int                       `json:"healthy_domains"`
	WarningDomains  int                       `json:"warning_domains"`
	BlacklistedCount int                      `json:"blacklisted_count"`
	TotalSent       int64                     `json:"total_sent"`
	ByCategory      map[string]int            `json:"by_category"`
}

// Domain category constants.
const (
	DomainCategoryCorporate = "corporate"
	DomainCategoryCloud     = "cloud"
	DomainCategoryFinancial = "financial"
	DomainCategoryShipping  = "shipping"
	DomainCategorySocial    = "social"
	DomainCategoryCustom    = "custom"
)

// Domain health status constants.
const (
	DomainHealthHealthy     = "healthy"
	DomainHealthWarning     = "warning"
	DomainHealthBlacklisted = "blacklisted"
	DomainHealthUnknown     = "unknown"
)

// Rotation strategy constants.
const (
	RotationRoundRobin = "round_robin"
	RotationRandom     = "random"
	RotationWeighted   = "weighted"
)

// Warmup stage constants.
const (
	WarmupStageCold  = 0
	WarmupStageReady = 6
)

// DailyLimitForWarmupStage returns the recommended daily send limit
// for a given warmup stage.
func DailyLimitForWarmupStage(stage int) int {
	limits := map[int]int{
		0: 10,
		1: 25,
		2: 50,
		3: 100,
		4: 200,
		5: 400,
		6: 1000,
	}
	if l, ok := limits[stage]; ok {
		return l
	}
	return 10
}

// Shared query constants for domain pool.
const (
	queryWhereDomainOrgID = "org_id = ?"
	queryWhereDomainID    = "id = ?"
)

// Domain pool errors.
var (
	ErrDomainNotFound     = errors.New("domain not found")
	ErrDomainExists       = errors.New("domain already exists in pool")
	ErrDomainBlacklisted  = errors.New("domain is blacklisted")
	ErrDailyLimitReached  = errors.New("daily sending limit reached for this domain")
	ErrNoDomainAvailable  = errors.New("no active domain available in pool")
)

// Table names.
func (SendingDomain) TableName() string    { return "sending_domains" }
func (DomainPoolConfig) TableName() string { return "domain_pool_configs" }

// ValidDomainCategories contains the set of valid domain categories.
var ValidDomainCategories = map[string]bool{
	DomainCategoryCorporate: true,
	DomainCategoryCloud:     true,
	DomainCategoryFinancial: true,
	DomainCategoryShipping:  true,
	DomainCategorySocial:    true,
	DomainCategoryCustom:    true,
}

// ValidRotationStrategies contains the set of valid rotation strategies.
var ValidRotationStrategies = map[string]bool{
	RotationRoundRobin: true,
	RotationRandom:     true,
	RotationWeighted:   true,
}

// GetDomainPoolConfig returns the domain pool config for an org (or defaults).
func GetDomainPoolConfig(orgId int64) DomainPoolConfig {
	cfg := DomainPoolConfig{}
	err := db.Where(queryWhereDomainOrgID, orgId).First(&cfg).Error
	if err != nil {
		return DomainPoolConfig{
			OrgId:             orgId,
			Enabled:           true,
			AutoRotate:        true,
			RotationStrategy:  RotationRoundRobin,
			MaxDailyPerDomain: 50,
			WarmupEnabled:     true,
		}
	}
	return cfg
}

// SaveDomainPoolConfig upserts the domain pool config.
func SaveDomainPoolConfig(cfg *DomainPoolConfig) error {
	existing := DomainPoolConfig{}
	err := db.Where(queryWhereDomainOrgID, cfg.OrgId).First(&existing).Error
	if err != nil {
		return db.Save(cfg).Error
	}
	cfg.Id = existing.Id
	return db.Save(cfg).Error
}

// GetSendingDomains returns all domains for an org.
func GetSendingDomains(orgId int64) ([]SendingDomain, error) {
	domains := []SendingDomain{}
	err := db.Where(queryWhereDomainOrgID, orgId).Order("category asc, domain asc").Find(&domains).Error
	return domains, err
}

// GetActiveSendingDomains returns only active, non-blacklisted domains.
func GetActiveSendingDomains(orgId int64) ([]SendingDomain, error) {
	domains := []SendingDomain{}
	err := db.Where("org_id = ? AND is_active = ? AND health_status != ?",
		orgId, true, DomainHealthBlacklisted).
		Order("last_used_date asc").
		Find(&domains).Error
	return domains, err
}

// GetSendingDomain returns a single domain by ID.
func GetSendingDomain(id int64) (*SendingDomain, error) {
	d := &SendingDomain{}
	err := db.Where(queryWhereDomainID, id).First(d).Error
	if err != nil {
		return nil, ErrDomainNotFound
	}
	return d, nil
}

// CreateSendingDomain adds a new domain to the pool.
func CreateSendingDomain(d *SendingDomain) error {
	// Check for duplicates
	var count int
	db.Model(&SendingDomain{}).Where("org_id = ? AND domain = ?", d.OrgId, d.Domain).Count(&count)
	if count > 0 {
		return ErrDomainExists
	}
	d.CreatedDate = time.Now().UTC()
	d.ModifiedDate = d.CreatedDate
	if d.HealthStatus == "" {
		d.HealthStatus = DomainHealthUnknown
	}
	return db.Save(d).Error
}

// UpdateSendingDomain updates an existing domain.
func UpdateSendingDomain(d *SendingDomain) error {
	d.ModifiedDate = time.Now().UTC()
	return db.Save(d).Error
}

// DeleteSendingDomain removes a domain from the pool.
func DeleteSendingDomain(id int64) error {
	return db.Where(queryWhereDomainID, id).Delete(&SendingDomain{}).Error
}

// SelectNextDomain picks the next domain to use for sending, based on
// the org's rotation strategy.
func SelectNextDomain(orgId int64) (*SendingDomain, error) {
	cfg := GetDomainPoolConfig(orgId)
	domains, err := GetActiveSendingDomains(orgId)
	if err != nil || len(domains) == 0 {
		return nil, ErrNoDomainAvailable
	}

	// Filter by daily limit
	available := make([]SendingDomain, 0)
	for _, d := range domains {
		limit := cfg.MaxDailyPerDomain
		if cfg.WarmupEnabled && d.WarmupStage < WarmupStageReady {
			limit = DailyLimitForWarmupStage(d.WarmupStage)
		}
		if d.SendsToday < limit {
			available = append(available, d)
		}
	}
	if len(available) == 0 {
		return nil, ErrDailyLimitReached
	}

	var selected *SendingDomain
	switch cfg.RotationStrategy {
	case RotationRandom:
		n, _ := rand.Int(rand.Reader, big.NewInt(int64(len(available))))
		idx := int(n.Int64())
		selected = &available[idx]
	case RotationWeighted:
		// Weight by remaining daily capacity
		selected = selectWeightedDomain(available, cfg)
	default: // round_robin
		// Already sorted by last_used_date asc, pick first
		selected = &available[0]
	}

	// Update usage stats
	selected.SendsToday++
	selected.TotalSent++
	selected.LastUsedDate = time.Now().UTC()
	selected.ModifiedDate = time.Now().UTC()
	if err := db.Save(selected).Error; err != nil {
		log.Errorf("domain pool: failed to update usage for domain %d: %v", selected.Id, err)
	}

	return selected, nil
}

// selectWeightedDomain picks a domain weighted by remaining daily capacity.
func selectWeightedDomain(domains []SendingDomain, cfg DomainPoolConfig) *SendingDomain {
	totalWeight := 0
	weights := make([]int, len(domains))
	for i, d := range domains {
		limit := cfg.MaxDailyPerDomain
		if cfg.WarmupEnabled && d.WarmupStage < WarmupStageReady {
			limit = DailyLimitForWarmupStage(d.WarmupStage)
		}
		remaining := limit - d.SendsToday
		if remaining < 1 {
			remaining = 1
		}
		weights[i] = remaining
		totalWeight += remaining
	}
	if totalWeight == 0 {
		return &domains[0]
	}
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(totalWeight)))
	target := int(n.Int64())
	cumulative := 0
	for i, w := range weights {
		cumulative += w
		if target < cumulative {
			return &domains[i]
		}
	}
	return &domains[len(domains)-1]
}

// ResetDailySendCounts resets the daily send counter for all domains.
// This should be called by a daily cron/worker.
func ResetDailySendCounts() error {
	return db.Model(&SendingDomain{}).
		Where("sends_today > 0").
		Update("sends_today", 0).Error
}

// AdvanceWarmup moves a domain to the next warmup stage if eligible.
func AdvanceWarmup(id int64) error {
	d, err := GetSendingDomain(id)
	if err != nil {
		return err
	}
	if d.WarmupStage >= WarmupStageReady {
		return nil
	}
	d.WarmupStage++
	d.DailyLimit = DailyLimitForWarmupStage(d.WarmupStage)
	d.ModifiedDate = time.Now().UTC()
	return db.Save(d).Error
}

// GetDomainPoolSummary returns aggregate statistics for the org's domain pool.
func GetDomainPoolSummary(orgId int64) (*DomainPoolSummary, error) {
	summary := &DomainPoolSummary{
		ByCategory: make(map[string]int),
	}
	domains, err := GetSendingDomains(orgId)
	if err != nil {
		return summary, err
	}
	summary.TotalDomains = len(domains)
	for _, d := range domains {
		if d.IsActive {
			summary.ActiveDomains++
		}
		switch d.HealthStatus {
		case DomainHealthHealthy:
			summary.HealthyDomains++
		case DomainHealthWarning:
			summary.WarningDomains++
		case DomainHealthBlacklisted:
			summary.BlacklistedCount++
		}
		summary.TotalSent += d.TotalSent
		summary.ByCategory[d.Category]++
	}
	return summary, nil
}

// GenerateFromAddress constructs a realistic From address using a domain
// from the pool with an appropriate display name.
func GenerateFromAddress(domain SendingDomain, senderName string) string {
	if senderName == "" {
		senderName = domain.DisplayName
	}
	localPart := strings.ToLower(strings.ReplaceAll(senderName, " ", "."))
	return fmt.Sprintf("%s <%s@%s>", senderName, localPart, domain.Domain)
}

// SeedBuiltInDomains populates the pool with realistic pre-configured domains.
func SeedBuiltInDomains(orgId int64) (int, error) {
	seeded := 0
	for _, d := range BuiltInSendingDomains {
		d.OrgId = orgId
		if err := CreateSendingDomain(&d); err != nil {
			if err == ErrDomainExists {
				continue
			}
			log.Errorf("domain pool: failed to seed domain %s: %v", d.Domain, err)
			continue
		}
		seeded++
	}
	if seeded > 0 {
		log.Infof("domain pool: seeded %d built-in domains for org %d", seeded, orgId)
	}
	return seeded, nil
}

// BuiltInSendingDomains provides realistic-looking domains for phishing simulations.
var BuiltInSendingDomains = []SendingDomain{
	// Corporate / IT support
	{Domain: "it-servicedesk.net", DisplayName: "IT Service Desk", Category: DomainCategoryCorporate, IsBuiltIn: true, IsActive: true, HealthStatus: DomainHealthHealthy},
	{Domain: "hr-notifications.net", DisplayName: "HR Department", Category: DomainCategoryCorporate, IsBuiltIn: true, IsActive: true, HealthStatus: DomainHealthHealthy},
	{Domain: "helpdesk-portal.com", DisplayName: "Help Desk", Category: DomainCategoryCorporate, IsBuiltIn: true, IsActive: true, HealthStatus: DomainHealthHealthy},
	{Domain: "security-alerts.net", DisplayName: "Security Team", Category: DomainCategoryCorporate, IsBuiltIn: true, IsActive: true, HealthStatus: DomainHealthHealthy},
	{Domain: "admin-notifications.com", DisplayName: "System Admin", Category: DomainCategoryCorporate, IsBuiltIn: true, IsActive: true, HealthStatus: DomainHealthHealthy},

	// Cloud services
	{Domain: "msft-365-security.com", DisplayName: "Microsoft 365", Category: DomainCategoryCloud, IsBuiltIn: true, IsActive: true, HealthStatus: DomainHealthHealthy},
	{Domain: "google-workspace-alerts.com", DisplayName: "Google Workspace", Category: DomainCategoryCloud, IsBuiltIn: true, IsActive: true, HealthStatus: DomainHealthHealthy},
	{Domain: "dropbox-sharing.net", DisplayName: "Dropbox", Category: DomainCategoryCloud, IsBuiltIn: true, IsActive: true, HealthStatus: DomainHealthHealthy},
	{Domain: "onedrive-share.net", DisplayName: "OneDrive", Category: DomainCategoryCloud, IsBuiltIn: true, IsActive: true, HealthStatus: DomainHealthHealthy},
	{Domain: "sharepoint-docs.net", DisplayName: "SharePoint Online", Category: DomainCategoryCloud, IsBuiltIn: true, IsActive: true, HealthStatus: DomainHealthHealthy},

	// Financial
	{Domain: "payroll-updates.com", DisplayName: "Payroll Department", Category: DomainCategoryFinancial, IsBuiltIn: true, IsActive: true, HealthStatus: DomainHealthHealthy},
	{Domain: "expense-portal.net", DisplayName: "Expense Management", Category: DomainCategoryFinancial, IsBuiltIn: true, IsActive: true, HealthStatus: DomainHealthHealthy},
	{Domain: "invoice-processing.com", DisplayName: "Accounts Payable", Category: DomainCategoryFinancial, IsBuiltIn: true, IsActive: true, HealthStatus: DomainHealthHealthy},

	// Shipping / logistics
	{Domain: "delivery-tracking.net", DisplayName: "Delivery Updates", Category: DomainCategoryShipping, IsBuiltIn: true, IsActive: true, HealthStatus: DomainHealthHealthy},
	{Domain: "parcel-notifications.com", DisplayName: "Parcel Service", Category: DomainCategoryShipping, IsBuiltIn: true, IsActive: true, HealthStatus: DomainHealthHealthy},

	// Social / communications
	{Domain: "linkedin-connect.net", DisplayName: "LinkedIn", Category: DomainCategorySocial, IsBuiltIn: true, IsActive: true, HealthStatus: DomainHealthHealthy},
	{Domain: "teams-meeting.net", DisplayName: "Microsoft Teams", Category: DomainCategorySocial, IsBuiltIn: true, IsActive: true, HealthStatus: DomainHealthHealthy},
	{Domain: "zoom-invitation.com", DisplayName: "Zoom", Category: DomainCategorySocial, IsBuiltIn: true, IsActive: true, HealthStatus: DomainHealthHealthy},
	{Domain: "slack-workspace.net", DisplayName: "Slack", Category: DomainCategorySocial, IsBuiltIn: true, IsActive: true, HealthStatus: DomainHealthHealthy},
	{Domain: "docusign-review.com", DisplayName: "DocuSign", Category: DomainCategorySocial, IsBuiltIn: true, IsActive: true, HealthStatus: DomainHealthHealthy},
}
