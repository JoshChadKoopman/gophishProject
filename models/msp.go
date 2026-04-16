package models

import (
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/jung-kurt/gofpdf"
)

// Feature flag constants for MSP capabilities.
const (
	FeatureMSPPartnerPortal = "msp_partner_portal"
	FeatureMSPMultiClient   = "msp_multi_client"
)

// Shared query-clause constants (avoids literal duplication).
const (
	mspQryID            = "id = ?"
	mspQryPartnerID     = "partner_id = ?"
	mspQryPartnerActive = "partner_id = ? AND is_active = ?"
	mspQryOrgActive     = "org_id = ? AND is_active = ?"
	mspQryPartnerOrgDfl = "partner_id = ? AND org_id = 0 AND is_active = ?"
	mspQryPartnerOrg    = "partner_id = ? AND org_id = ?"
)

// MSPPartner represents a Managed Service Provider partner that manages
// multiple client organizations through the platform.
type MSPPartner struct {
	Id               int64      `json:"id" gorm:"primary_key"`
	Name             string     `json:"name" sql:"not null"`
	Slug             string     `json:"slug" sql:"not null;unique"`
	ContactEmail     string     `json:"contact_email"`
	ContactPhone     string     `json:"contact_phone"`
	Website          string     `json:"website"`
	MaxClients       int        `json:"max_clients" gorm:"default:50"`
	IsActive         bool       `json:"is_active" gorm:"default:1"`
	PrimaryUserId    int64      `json:"primary_user_id"`
	Notes            string     `json:"notes"`
	ContractStartsAt *time.Time `json:"contract_starts_at"`
	ContractEndsAt   *time.Time `json:"contract_ends_at"`
	CreatedDate      time.Time  `json:"created_date"`
	ModifiedDate     time.Time  `json:"modified_date"`
}

// MSPPartnerClient maps an MSP partner to a client organization it manages.
type MSPPartnerClient struct {
	Id        int64     `json:"id" gorm:"primary_key"`
	PartnerId int64     `json:"partner_id" sql:"not null"`
	OrgId     int64     `json:"org_id" sql:"not null"`
	AddedDate time.Time `json:"added_date"`
	IsActive  bool      `json:"is_active" gorm:"default:1"`
	OrgName   string    `json:"org_name,omitempty" gorm:"-"`
	OrgSlug   string    `json:"org_slug,omitempty" gorm:"-"`
}

// WhiteLabelConfig holds branding overrides for an org or MSP partner.
type WhiteLabelConfig struct {
	Id               int64     `json:"id" gorm:"primary_key"`
	OrgId            int64     `json:"org_id" sql:"not null"`
	PartnerId        int64     `json:"partner_id"`
	CompanyName      string    `json:"company_name"`
	LogoURL          string    `json:"logo_url"`
	LogoSmallURL     string    `json:"logo_small_url"`
	PrimaryColor     string    `json:"primary_color"`
	SecondaryColor   string    `json:"secondary_color"`
	AccentColor      string    `json:"accent_color"`
	BackgroundColor  string    `json:"background_color"`
	FontFamily       string    `json:"font_family"`
	LoginPageTitle   string    `json:"login_page_title"`
	LoginPageMessage string    `json:"login_page_message"`
	FooterText       string    `json:"footer_text"`
	SupportEmail     string    `json:"support_email"`
	SupportURL       string    `json:"support_url"`
	CustomCSS        string    `json:"custom_css"`
	EmailFromName    string    `json:"email_from_name"`
	EmailFooterHTML  string    `json:"email_footer_html"`
	HidePoweredBy    bool      `json:"hide_powered_by"`
	IsActive         bool      `json:"is_active" gorm:"default:1"`
	CreatedDate      time.Time `json:"created_date"`
	ModifiedDate     time.Time `json:"modified_date"`
}

// MSPClientSummary is a read-only aggregate for each managed client.
type MSPClientSummary struct {
	OrgId              int64   `json:"org_id"`
	OrgName            string  `json:"org_name"`
	OrgSlug            string  `json:"org_slug"`
	TierName           string  `json:"tier_name"`
	UserCount          int     `json:"user_count"`
	CampaignCount      int     `json:"campaign_count"`
	ActiveCampaigns    int     `json:"active_campaigns"`
	AvgRiskScore       float64 `json:"avg_risk_score"`
	TrainingCompletion float64 `json:"training_completion_pct"`
	IsActive           bool    `json:"is_active"`
}

// MSPPortalDashboard is the top-level partner portal response.
type MSPPortalDashboard struct {
	Partner        MSPPartner         `json:"partner"`
	TotalClients   int                `json:"total_clients"`
	ActiveClients  int                `json:"active_clients"`
	TotalUsers     int                `json:"total_users"`
	TotalCampaigns int                `json:"total_campaigns"`
	Clients        []MSPClientSummary `json:"clients"`
}

// MSPCrossClientReport provides aggregated metrics across all clients.
type MSPCrossClientReport struct {
	PartnerId           int64              `json:"partner_id"`
	PartnerName         string             `json:"partner_name"`
	TotalClients        int                `json:"total_clients"`
	TotalUsers          int                `json:"total_users"`
	TotalCampaigns      int                `json:"total_campaigns"`
	AvgRiskScore        float64            `json:"avg_risk_score"`
	AvgTrainingComplete float64            `json:"avg_training_completion_pct"`
	HighRiskClients     int                `json:"high_risk_clients"`
	ClientBreakdown     []MSPClientSummary `json:"client_breakdown"`
}

// MSPClientRanking represents a client's position in the risk ranking.
type MSPClientRanking struct {
	Rank               int     `json:"rank"`
	OrgId              int64   `json:"org_id"`
	OrgName            string  `json:"org_name"`
	AvgRiskScore       float64 `json:"avg_risk_score"`
	TrainingCompletion float64 `json:"training_completion_pct"`
	UserCount          int     `json:"user_count"`
	CampaignCount      int     `json:"campaign_count"`
	RiskLevel          string  `json:"risk_level"` // "high", "medium", "low"
	RiskColor          string  `json:"risk_color"` // "#e74c3c", "#f39c12", "#27ae60"
}

// MSPClientComparison holds radar/spider chart data for comparing clients.
type MSPClientComparison struct {
	Labels  []string                   `json:"labels"`
	Clients []MSPClientComparisonEntry `json:"clients"`
}

// MSPClientComparisonEntry is one client's data on the radar chart.
type MSPClientComparisonEntry struct {
	OrgId   int64     `json:"org_id"`
	OrgName string    `json:"org_name"`
	Values  []float64 `json:"values"`
}

// MSPBillingUsage holds license/seat usage info for a partner's client.
type MSPBillingUsage struct {
	OrgId          int64   `json:"org_id"`
	OrgName        string  `json:"org_name"`
	TierName       string  `json:"tier_name"`
	SeatsAllocated int     `json:"seats_allocated"`
	SeatsUsed      int     `json:"seats_used"`
	UsagePct       float64 `json:"usage_pct"`
	AtLimit        bool    `json:"at_limit"`
	OverLimit      bool    `json:"over_limit"`
	CampaignsMax   int     `json:"campaigns_max"`
	CampaignsUsed  int     `json:"campaigns_used"`
}

// MSPBillingSummary is the top-level billing response for a partner.
type MSPBillingSummary struct {
	PartnerId        int64             `json:"partner_id"`
	PartnerName      string            `json:"partner_name"`
	TotalSeats       int               `json:"total_seats_allocated"`
	TotalSeatsUsed   int               `json:"total_seats_used"`
	ClientsAtLimit   int               `json:"clients_at_limit"`
	ClientsOverLimit int               `json:"clients_over_limit"`
	Clients          []MSPBillingUsage `json:"clients"`
}

// Error sentinels.
var (
	ErrMSPPartnerNotFound       = errors.New("MSP partner not found")
	ErrMSPPartnerNameRequired   = errors.New("Partner name is required")
	ErrMSPPartnerSlugRequired   = errors.New("Partner slug is required")
	ErrMSPClientAlreadyMapped   = errors.New("Organization is already managed by this partner")
	ErrMSPClientLimitReached    = errors.New("Partner has reached the maximum number of clients")
	ErrMSPClientNotFound        = errors.New("Partner-client mapping not found")
	ErrWhiteLabelConfigNotFound = errors.New("White-label configuration not found")
	ErrMSPNotPartner            = errors.New("User is not associated with an MSP partner")
)

// ═══════════════════════════════════════════════════════════════════════════
// MSP Partner CRUD
// ═══════════════════════════════════════════════════════════════════════════

// GetMSPPartner returns a single partner by id.
func GetMSPPartner(id int64) (MSPPartner, error) {
	p := MSPPartner{}
	if err := db.Where(mspQryID, id).First(&p).Error; err != nil {
		return p, ErrMSPPartnerNotFound
	}
	return p, nil
}

// GetMSPPartnerBySlug returns a partner by slug.
func GetMSPPartnerBySlug(slug string) (MSPPartner, error) {
	p := MSPPartner{}
	if err := db.Where("slug = ?", slug).First(&p).Error; err != nil {
		return p, ErrMSPPartnerNotFound
	}
	return p, nil
}

// GetMSPPartnerByUserId returns the partner whose primary_user_id matches.
func GetMSPPartnerByUserId(userId int64) (MSPPartner, error) {
	p := MSPPartner{}
	if err := db.Where("primary_user_id = ?", userId).First(&p).Error; err != nil {
		return p, ErrMSPNotPartner
	}
	return p, nil
}

// GetMSPPartners returns all partners (superadmin view).
func GetMSPPartners() ([]MSPPartner, error) {
	partners := []MSPPartner{}
	err := db.Order("name asc").Find(&partners).Error
	return partners, err
}

// PostMSPPartner creates a new MSP partner.
func PostMSPPartner(p *MSPPartner) error {
	if p.Name == "" {
		return ErrMSPPartnerNameRequired
	}
	if p.Slug == "" {
		return ErrMSPPartnerSlugRequired
	}
	p.CreatedDate = time.Now().UTC()
	p.ModifiedDate = time.Now().UTC()
	return db.Save(p).Error
}

// PutMSPPartner updates an existing partner.
func PutMSPPartner(p *MSPPartner) error {
	if p.Name == "" {
		return ErrMSPPartnerNameRequired
	}
	p.ModifiedDate = time.Now().UTC()
	return db.Save(p).Error
}

// DeleteMSPPartner deletes a partner and all its client mappings.
func DeleteMSPPartner(id int64) error {
	if err := db.Where(mspQryPartnerID, id).Delete(&MSPPartnerClient{}).Error; err != nil {
		return err
	}
	if err := db.Where(mspQryPartnerID, id).Delete(&WhiteLabelConfig{}).Error; err != nil {
		return err
	}
	return db.Where(mspQryID, id).Delete(&MSPPartner{}).Error
}

// ═══════════════════════════════════════════════════════════════════════════
// Partner-Client mapping CRUD
// ═══════════════════════════════════════════════════════════════════════════

// GetMSPPartnerClients returns all client orgs managed by a partner.
func GetMSPPartnerClients(partnerId int64) ([]MSPPartnerClient, error) {
	mappings := []MSPPartnerClient{}
	err := db.Where(mspQryPartnerID, partnerId).Order("added_date desc").Find(&mappings).Error
	if err != nil {
		return mappings, err
	}
	for i := range mappings {
		if org, oErr := GetOrganization(mappings[i].OrgId); oErr == nil {
			mappings[i].OrgName = org.Name
			mappings[i].OrgSlug = org.Slug
		}
	}
	return mappings, nil
}

// GetMSPPartnerClientOrgIds returns just the active org ids for a partner.
func GetMSPPartnerClientOrgIds(partnerId int64) ([]int64, error) {
	var ids []int64
	err := db.Model(&MSPPartnerClient{}).
		Where(mspQryPartnerActive, partnerId, true).
		Pluck("org_id", &ids).Error
	return ids, err
}

// AddMSPPartnerClient maps a client org to a partner.
func AddMSPPartnerClient(partnerId, orgId int64) (*MSPPartnerClient, error) {
	partner, err := GetMSPPartner(partnerId)
	if err != nil {
		return nil, err
	}
	var count int
	db.Model(&MSPPartnerClient{}).Where(mspQryPartnerActive, partnerId, true).Count(&count)
	if count >= partner.MaxClients {
		return nil, ErrMSPClientLimitReached
	}
	var existing MSPPartnerClient
	notFound := db.Where(mspQryPartnerOrg, partnerId, orgId).First(&existing).RecordNotFound()
	if !notFound {
		if existing.IsActive {
			return nil, ErrMSPClientAlreadyMapped
		}
		existing.IsActive = true
		db.Save(&existing)
		return &existing, nil
	}
	mapping := MSPPartnerClient{
		PartnerId: partnerId,
		OrgId:     orgId,
		AddedDate: time.Now().UTC(),
		IsActive:  true,
	}
	if err := db.Save(&mapping).Error; err != nil {
		return nil, err
	}
	return &mapping, nil
}

// RemoveMSPPartnerClient deactivates a partner-client mapping.
func RemoveMSPPartnerClient(partnerId, orgId int64) error {
	result := db.Model(&MSPPartnerClient{}).
		Where(mspQryPartnerOrg, partnerId, orgId).
		Update("is_active", false)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrMSPClientNotFound
	}
	return nil
}

// IsOrgManagedByPartner checks if a specific org is managed by a partner.
func IsOrgManagedByPartner(partnerId, orgId int64) bool {
	var count int
	db.Model(&MSPPartnerClient{}).
		Where("partner_id = ? AND org_id = ? AND is_active = ?", partnerId, orgId, true).
		Count(&count)
	return count > 0
}

// ═══════════════════════════════════════════════════════════════════════════
// White-label branding CRUD
// ═══════════════════════════════════════════════════════════════════════════

// GetWhiteLabelConfig returns the white-label config for an org.
// Falls back to the partner-level default if no org-level config exists.
func GetWhiteLabelConfig(orgId int64) (WhiteLabelConfig, error) {
	cfg := WhiteLabelConfig{}
	if err := db.Where(mspQryOrgActive, orgId, true).First(&cfg).Error; err == nil {
		return cfg, nil
	}
	var mapping MSPPartnerClient
	if mErr := db.Where(mspQryOrgActive, orgId, true).First(&mapping).Error; mErr == nil {
		if pErr := db.Where(mspQryPartnerOrgDfl, mapping.PartnerId, true).First(&cfg).Error; pErr == nil {
			return cfg, nil
		}
	}
	return cfg, ErrWhiteLabelConfigNotFound
}

// GetWhiteLabelConfigByPartner returns the partner-level default config.
func GetWhiteLabelConfigByPartner(partnerId int64) (WhiteLabelConfig, error) {
	cfg := WhiteLabelConfig{}
	if err := db.Where(mspQryPartnerOrgDfl, partnerId, true).First(&cfg).Error; err != nil {
		return cfg, ErrWhiteLabelConfigNotFound
	}
	return cfg, nil
}

// GetWhiteLabelConfigsByPartner returns all configs for a partner.
func GetWhiteLabelConfigsByPartner(partnerId int64) ([]WhiteLabelConfig, error) {
	configs := []WhiteLabelConfig{}
	err := db.Where(mspQryPartnerActive, partnerId, true).Find(&configs).Error
	return configs, err
}

// SaveWhiteLabelConfig creates or updates a white-label config.
func SaveWhiteLabelConfig(cfg *WhiteLabelConfig) error {
	if cfg.CompanyName == "" {
		return errors.New("Company name is required for white-label config")
	}
	cfg.ModifiedDate = time.Now().UTC()
	if cfg.Id == 0 {
		cfg.CreatedDate = time.Now().UTC()
	}
	return db.Save(cfg).Error
}

// DeleteWhiteLabelConfig removes a white-label config by id.
func DeleteWhiteLabelConfig(id int64) error {
	return db.Where(mspQryID, id).Delete(&WhiteLabelConfig{}).Error
}

// ═══════════════════════════════════════════════════════════════════════════
// Partner Portal dashboard
// ═══════════════════════════════════════════════════════════════════════════

// GetMSPPortalDashboard builds the partner portal dashboard for a partner.
func GetMSPPortalDashboard(partnerId int64) (MSPPortalDashboard, error) {
	dash := MSPPortalDashboard{}
	partner, err := GetMSPPartner(partnerId)
	if err != nil {
		return dash, err
	}
	dash.Partner = partner
	clients, err := GetMSPPartnerClients(partnerId)
	if err != nil {
		return dash, err
	}
	dash.TotalClients = len(clients)
	summaries := make([]MSPClientSummary, 0, len(clients))
	for _, c := range clients {
		s := buildClientSummary(c)
		if c.IsActive {
			dash.ActiveClients++
		}
		dash.TotalUsers += s.UserCount
		dash.TotalCampaigns += s.CampaignCount
		summaries = append(summaries, s)
	}
	dash.Clients = summaries
	return dash, nil
}

// buildClientSummary populates a single MSPClientSummary.
func buildClientSummary(c MSPPartnerClient) MSPClientSummary {
	s := MSPClientSummary{
		OrgId: c.OrgId, OrgName: c.OrgName, OrgSlug: c.OrgSlug, IsActive: c.IsActive,
	}
	if uc, err := GetOrgUserCount(c.OrgId); err == nil {
		s.UserCount = uc
	}
	if cc, err := GetOrgCampaignCount(c.OrgId); err == nil {
		s.CampaignCount = cc
	}
	var active int
	db.Table("campaigns").Where("org_id = ? AND status = ?", c.OrgId, CampaignInProgress).Count(&active)
	s.ActiveCampaigns = active

	if org, err := GetOrganization(c.OrgId); err == nil {
		if tier, tErr := GetSubscriptionTier(org.TierId); tErr == nil {
			s.TierName = tier.Name
		}
	}

	var avgRisk float64
	if rErr := db.Table("behavioral_risk_scores").
		Select("COALESCE(AVG(composite_score), 0)").
		Where(queryWhereOrgID, c.OrgId).Row().Scan(&avgRisk); rErr == nil {
		s.AvgRiskScore = avgRisk
	}

	var total, completed int
	db.Table("course_assignments").Where(queryWhereOrgID, c.OrgId).Count(&total)
	db.Table("course_assignments").Where("org_id = ? AND status = ?", c.OrgId, "completed").Count(&completed)
	if total > 0 {
		s.TrainingCompletion = float64(completed) / float64(total) * 100
	}
	return s
}

// ═══════════════════════════════════════════════════════════════════════════
// Cross-client reporting
// ═══════════════════════════════════════════════════════════════════════════

// GetMSPCrossClientReport builds an aggregated cross-client report.
func GetMSPCrossClientReport(partnerId int64) (MSPCrossClientReport, error) {
	report := MSPCrossClientReport{}
	partner, err := GetMSPPartner(partnerId)
	if err != nil {
		return report, err
	}
	report.PartnerId = partner.Id
	report.PartnerName = partner.Name
	dash, err := GetMSPPortalDashboard(partnerId)
	if err != nil {
		return report, err
	}
	report.TotalClients = dash.ActiveClients
	report.TotalUsers = dash.TotalUsers
	report.TotalCampaigns = dash.TotalCampaigns
	report.ClientBreakdown = dash.Clients

	var totalRisk, totalCompletion float64
	for _, c := range dash.Clients {
		if !c.IsActive {
			continue
		}
		totalRisk += c.AvgRiskScore
		totalCompletion += c.TrainingCompletion
		if c.AvgRiskScore > 70 {
			report.HighRiskClients++
		}
	}
	if dash.ActiveClients > 0 {
		report.AvgRiskScore = totalRisk / float64(dash.ActiveClients)
		report.AvgTrainingComplete = totalCompletion / float64(dash.ActiveClients)
	}
	return report, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// Client Ranking (sorted by risk score, traffic-light indicators)
// ═══════════════════════════════════════════════════════════════════════════

// GetMSPClientRanking returns all active clients sorted by risk score (highest first).
func GetMSPClientRanking(partnerId int64) ([]MSPClientRanking, error) {
	dash, err := GetMSPPortalDashboard(partnerId)
	if err != nil {
		return nil, err
	}

	rankings := make([]MSPClientRanking, 0, len(dash.Clients))
	for _, c := range dash.Clients {
		if !c.IsActive {
			continue
		}
		r := MSPClientRanking{
			OrgId:              c.OrgId,
			OrgName:            c.OrgName,
			AvgRiskScore:       c.AvgRiskScore,
			TrainingCompletion: c.TrainingCompletion,
			UserCount:          c.UserCount,
			CampaignCount:      c.CampaignCount,
		}
		switch {
		case c.AvgRiskScore > 70:
			r.RiskLevel = "high"
			r.RiskColor = "#e74c3c"
		case c.AvgRiskScore > 40:
			r.RiskLevel = "medium"
			r.RiskColor = "#f39c12"
		default:
			r.RiskLevel = "low"
			r.RiskColor = "#27ae60"
		}
		rankings = append(rankings, r)
	}

	// Sort by risk score descending (highest risk first)
	sort.Slice(rankings, func(i, j int) bool {
		return rankings[i].AvgRiskScore > rankings[j].AvgRiskScore
	})
	for i := range rankings {
		rankings[i].Rank = i + 1
	}
	return rankings, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// Client Comparison (radar chart data)
// ═══════════════════════════════════════════════════════════════════════════

// GetMSPClientComparison returns radar chart data for comparing 2-3 clients.
// Dimensions: Risk Score (inverted), Training Completion, Campaign Activity,
// User Coverage, Phish Report Rate.
func GetMSPClientComparison(partnerId int64, orgIds []int64) (MSPClientComparison, error) {
	comparison := MSPClientComparison{
		Labels: []string{
			"Security Score",      // 100 - risk_score (higher is better)
			"Training Completion", // percentage
			"Campaign Activity",   // normalized 0-100
			"User Coverage",       // users / max_users * 100
			"Phish Report Rate",   // percentage of phish reports
		},
	}

	if len(orgIds) < 2 || len(orgIds) > 3 {
		return comparison, errors.New("Please select 2 or 3 clients to compare")
	}

	// Verify all orgs are managed by this partner
	for _, oid := range orgIds {
		if !IsOrgManagedByPartner(partnerId, oid) {
			return comparison, fmt.Errorf("Organization %d is not managed by this partner", oid)
		}
	}

	// Find max campaigns across selected orgs for normalization
	maxCampaigns := 1
	for _, oid := range orgIds {
		if cc, err := GetOrgCampaignCount(oid); err == nil && cc > maxCampaigns {
			maxCampaigns = cc
		}
	}

	for _, oid := range orgIds {
		org, err := GetOrganization(oid)
		if err != nil {
			continue
		}

		entry := MSPClientComparisonEntry{
			OrgId:   oid,
			OrgName: org.Name,
			Values:  make([]float64, 5),
		}

		// Security Score (inverted risk: 100 - avg_risk)
		var avgRisk float64
		if rErr := db.Table("behavioral_risk_scores").
			Select("COALESCE(AVG(composite_score), 0)").
			Where(queryWhereOrgID, oid).Row().Scan(&avgRisk); rErr == nil {
			entry.Values[0] = 100 - avgRisk
		}

		// Training Completion
		var total, completed int
		db.Table("course_assignments").Where(queryWhereOrgID, oid).Count(&total)
		db.Table("course_assignments").Where("org_id = ? AND status = ?", oid, "completed").Count(&completed)
		if total > 0 {
			entry.Values[1] = float64(completed) / float64(total) * 100
		}

		// Campaign Activity (normalized)
		if cc, err := GetOrgCampaignCount(oid); err == nil {
			entry.Values[2] = float64(cc) / float64(maxCampaigns) * 100
		}

		// User Coverage (users / max_users)
		uc, _ := GetOrgUserCount(oid)
		if org.MaxUsers > 0 {
			entry.Values[3] = float64(uc) / float64(org.MaxUsers) * 100
			if entry.Values[3] > 100 {
				entry.Values[3] = 100
			}
		} else {
			entry.Values[3] = 50 // default if no max configured
		}

		// Phish Report Rate
		var reportedCount int
		db.Table("results").Where("org_id = ? AND reported = ?", oid, true).Count(&reportedCount)
		var totalResults int
		db.Table("results").Where(queryWhereOrgID, oid).Count(&totalResults)
		if totalResults > 0 {
			entry.Values[4] = float64(reportedCount) / float64(totalResults) * 100
		}

		comparison.Clients = append(comparison.Clients, entry)
	}

	return comparison, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// Billing / License Usage
// ═══════════════════════════════════════════════════════════════════════════

// GetMSPBillingUsage returns seat/license usage across all partner clients.
func GetMSPBillingUsage(partnerId int64) (MSPBillingSummary, error) {
	summary := MSPBillingSummary{}
	partner, err := GetMSPPartner(partnerId)
	if err != nil {
		return summary, err
	}
	summary.PartnerId = partner.Id
	summary.PartnerName = partner.Name

	clients, err := GetMSPPartnerClients(partnerId)
	if err != nil {
		return summary, err
	}

	for _, c := range clients {
		if !c.IsActive {
			continue
		}
		org, oErr := GetOrganization(c.OrgId)
		if oErr != nil {
			continue
		}

		usage := MSPBillingUsage{
			OrgId:   c.OrgId,
			OrgName: c.OrgName,
		}

		// Get tier info for seat limits
		if tier, tErr := GetSubscriptionTier(org.TierId); tErr == nil {
			usage.TierName = tier.Name
			usage.SeatsAllocated = tier.MaxUsers
			usage.CampaignsMax = tier.MaxCampaigns
		}
		// Fallback to org-level limits
		if usage.SeatsAllocated == 0 && org.MaxUsers > 0 {
			usage.SeatsAllocated = org.MaxUsers
		}
		if usage.CampaignsMax == 0 && org.MaxCampaigns > 0 {
			usage.CampaignsMax = org.MaxCampaigns
		}

		usage.SeatsUsed, _ = GetOrgUserCount(c.OrgId)
		usage.CampaignsUsed, _ = GetOrgCampaignCount(c.OrgId)

		if usage.SeatsAllocated > 0 {
			usage.UsagePct = float64(usage.SeatsUsed) / float64(usage.SeatsAllocated) * 100
			usage.AtLimit = usage.SeatsUsed >= usage.SeatsAllocated
			usage.OverLimit = usage.SeatsUsed > usage.SeatsAllocated
		}

		summary.TotalSeats += usage.SeatsAllocated
		summary.TotalSeatsUsed += usage.SeatsUsed
		if usage.AtLimit {
			summary.ClientsAtLimit++
		}
		if usage.OverLimit {
			summary.ClientsOverLimit++
		}

		summary.Clients = append(summary.Clients, usage)
	}

	return summary, nil
}

// ═══════════════════════════════════════════════════════════════════════════
// MSP Quarterly Review PDF Report
// ═══════════════════════════════════════════════════════════════════════════

// GenerateMSPQuarterlyPDF creates an aggregated PDF report with all client
// metrics, white-label branding, and cross-client benchmarks.
func GenerateMSPQuarterlyPDF(partnerId int64) (*gofpdf.Fpdf, error) {
	partner, err := GetMSPPartner(partnerId)
	if err != nil {
		return nil, err
	}

	report, err := GetMSPCrossClientReport(partnerId)
	if err != nil {
		return nil, err
	}

	rankings, err := GetMSPClientRanking(partnerId)
	if err != nil {
		return nil, err
	}

	billing, err := GetMSPBillingUsage(partnerId)
	if err != nil {
		return nil, err
	}

	// Load white-label branding for the partner
	wl, _ := GetWhiteLabelConfigByPartner(partnerId)
	companyName := wl.CompanyName
	if companyName == "" {
		companyName = partner.Name
	}
	primaryColor := wl.PrimaryColor
	if primaryColor == "" {
		primaryColor = "#1a73e8"
	}
	footerText := wl.FooterText
	if footerText == "" {
		footerText = fmt.Sprintf("© %d %s", time.Now().Year(), companyName)
	}

	// Parse primary color for PDF
	pr, pg, pb := hexToRGB(primaryColor)

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetAutoPageBreak(true, 20)

	// ── Cover Page ──
	pdf.AddPage()
	pdf.SetFillColor(int(pr), int(pg), int(pb))
	pdf.Rect(0, 0, 210, 50, "F")
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 24)
	pdf.SetXY(15, 15)
	pdf.CellFormat(180, 12, "MSP Quarterly Review", "", 1, "C", false, 0, "")
	pdf.SetFont("Arial", "", 14)
	pdf.SetXY(15, 30)
	pdf.CellFormat(180, 8, companyName, "", 1, "C", false, 0, "")

	pdf.SetTextColor(100, 100, 100)
	pdf.SetFont("Arial", "", 11)
	pdf.SetXY(15, 60)
	pdf.CellFormat(180, 6, fmt.Sprintf("Report Generated: %s", time.Now().Format("January 2, 2006")), "", 1, "C", false, 0, "")
	pdf.SetXY(15, 68)
	pdf.CellFormat(180, 6, fmt.Sprintf("Partner: %s", partner.Name), "", 1, "C", false, 0, "")

	// ── Executive Summary ──
	pdf.AddPage()
	pdf.SetTextColor(int(pr), int(pg), int(pb))
	pdf.SetFont("Arial", "B", 18)
	pdf.CellFormat(0, 10, "Executive Summary", "", 1, "", false, 0, "")
	pdf.SetDrawColor(int(pr), int(pg), int(pb))
	pdf.Line(10, pdf.GetY(), 200, pdf.GetY())
	pdf.Ln(8)

	pdf.SetTextColor(50, 50, 50)
	pdf.SetFont("Arial", "", 11)
	summaryItems := []struct{ label, value string }{
		{"Total Active Clients", fmt.Sprintf("%d", report.TotalClients)},
		{"Total Users Managed", fmt.Sprintf("%d", report.TotalUsers)},
		{"Total Campaigns Run", fmt.Sprintf("%d", report.TotalCampaigns)},
		{"Average Risk Score", fmt.Sprintf("%.1f", report.AvgRiskScore)},
		{"Average Training Completion", fmt.Sprintf("%.1f%%", report.AvgTrainingComplete)},
		{"High Risk Clients (>70)", fmt.Sprintf("%d", report.HighRiskClients)},
		{"Total Seats Allocated", fmt.Sprintf("%d", billing.TotalSeats)},
		{"Total Seats Used", fmt.Sprintf("%d", billing.TotalSeatsUsed)},
		{"Clients At Seat Limit", fmt.Sprintf("%d", billing.ClientsAtLimit)},
	}
	for _, item := range summaryItems {
		pdf.SetFont("Arial", "B", 11)
		pdf.CellFormat(80, 7, item.label+":", "", 0, "", false, 0, "")
		pdf.SetFont("Arial", "", 11)
		pdf.CellFormat(0, 7, item.value, "", 1, "", false, 0, "")
	}

	// ── Client Risk Ranking Table ──
	pdf.Ln(10)
	pdf.SetTextColor(int(pr), int(pg), int(pb))
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(0, 10, "Client Risk Ranking", "", 1, "", false, 0, "")
	pdf.SetDrawColor(int(pr), int(pg), int(pb))
	pdf.Line(10, pdf.GetY(), 200, pdf.GetY())
	pdf.Ln(5)

	// Table header
	pdf.SetFillColor(int(pr), int(pg), int(pb))
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 9)
	colWidths := []float64{12, 50, 28, 32, 22, 24, 22}
	headers := []string{"#", "Client", "Risk Score", "Training %", "Users", "Campaigns", "Status"}
	for i, h := range headers {
		pdf.CellFormat(colWidths[i], 8, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	// Table rows
	pdf.SetFont("Arial", "", 9)
	for _, r := range rankings {
		// Risk color indicator
		rr, rg, rb := hexToRGB(r.RiskColor)
		pdf.SetFillColor(int(rr), int(rg), int(rb))
		pdf.SetTextColor(50, 50, 50)

		pdf.CellFormat(colWidths[0], 7, fmt.Sprintf("%d", r.Rank), "1", 0, "C", false, 0, "")

		name := r.OrgName
		if len(name) > 22 {
			name = name[:22] + ".."
		}
		pdf.CellFormat(colWidths[1], 7, name, "1", 0, "", false, 0, "")
		// Colored risk score cell
		pdf.SetFillColor(int(rr), int(rg), int(rb))
		pdf.SetTextColor(255, 255, 255)
		pdf.CellFormat(colWidths[2], 7, fmt.Sprintf("%.1f", r.AvgRiskScore), "1", 0, "C", true, 0, "")
		pdf.SetTextColor(50, 50, 50)
		pdf.CellFormat(colWidths[3], 7, fmt.Sprintf("%.1f%%", r.TrainingCompletion), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colWidths[4], 7, fmt.Sprintf("%d", r.UserCount), "1", 0, "C", false, 0, "")
		pdf.CellFormat(colWidths[5], 7, fmt.Sprintf("%d", r.CampaignCount), "1", 0, "C", false, 0, "")
		pdf.SetFillColor(int(rr), int(rg), int(rb))
		pdf.SetTextColor(255, 255, 255)
		statusText := "Low"
		if r.RiskLevel == "high" {
			statusText = "High"
		} else if r.RiskLevel == "medium" {
			statusText = "Medium"
		}
		pdf.CellFormat(colWidths[6], 7, statusText, "1", 0, "C", true, 0, "")
		pdf.SetTextColor(50, 50, 50)
		pdf.Ln(-1)

		// Page break check
		if pdf.GetY() > 260 {
			pdf.AddPage()
		}
	}

	// ── Billing / License Usage ──
	pdf.AddPage()
	pdf.SetTextColor(int(pr), int(pg), int(pb))
	pdf.SetFont("Arial", "B", 16)
	pdf.CellFormat(0, 10, "Billing & License Usage", "", 1, "", false, 0, "")
	pdf.SetDrawColor(int(pr), int(pg), int(pb))
	pdf.Line(10, pdf.GetY(), 200, pdf.GetY())
	pdf.Ln(5)

	// Billing table header
	pdf.SetFillColor(int(pr), int(pg), int(pb))
	pdf.SetTextColor(255, 255, 255)
	pdf.SetFont("Arial", "B", 9)
	billingCols := []float64{45, 30, 25, 25, 30, 35}
	billingHeaders := []string{"Client", "Tier", "Seats Used", "Allocated", "Usage %", "Alert"}
	for i, h := range billingHeaders {
		pdf.CellFormat(billingCols[i], 8, h, "1", 0, "C", true, 0, "")
	}
	pdf.Ln(-1)

	pdf.SetFont("Arial", "", 9)
	pdf.SetTextColor(50, 50, 50)
	for _, bu := range billing.Clients {
		name := bu.OrgName
		if len(name) > 20 {
			name = name[:20] + ".."
		}
		pdf.CellFormat(billingCols[0], 7, name, "1", 0, "", false, 0, "")
		pdf.CellFormat(billingCols[1], 7, bu.TierName, "1", 0, "C", false, 0, "")
		pdf.CellFormat(billingCols[2], 7, fmt.Sprintf("%d", bu.SeatsUsed), "1", 0, "C", false, 0, "")
		pdf.CellFormat(billingCols[3], 7, fmt.Sprintf("%d", bu.SeatsAllocated), "1", 0, "C", false, 0, "")

		// Usage % with color
		usagePctStr := fmt.Sprintf("%.0f%%", bu.UsagePct)
		if bu.OverLimit {
			pdf.SetFillColor(231, 76, 60)
			pdf.SetTextColor(255, 255, 255)
		} else if bu.AtLimit {
			pdf.SetFillColor(243, 156, 18)
			pdf.SetTextColor(255, 255, 255)
		} else if bu.UsagePct > 80 {
			pdf.SetFillColor(255, 235, 156)
			pdf.SetTextColor(50, 50, 50)
		} else {
			pdf.SetFillColor(255, 255, 255)
			pdf.SetTextColor(50, 50, 50)
		}
		pdf.CellFormat(billingCols[4], 7, usagePctStr, "1", 0, "C", true, 0, "")
		pdf.SetTextColor(50, 50, 50)

		alert := "-"
		if bu.OverLimit {
			alert = "OVER LIMIT"
		} else if bu.AtLimit {
			alert = "AT LIMIT"
		} else if bu.UsagePct > 80 {
			alert = "Near Limit"
		}
		pdf.CellFormat(billingCols[5], 7, alert, "1", 0, "C", false, 0, "")
		pdf.Ln(-1)

		if pdf.GetY() > 260 {
			pdf.AddPage()
		}
	}

	// ── Footer on each page ──
	totalPages := pdf.PageCount()
	for i := 1; i <= totalPages; i++ {
		pdf.SetPage(i)
		pdf.SetY(-15)
		pdf.SetFont("Arial", "I", 8)
		pdf.SetTextColor(150, 150, 150)
		pdf.CellFormat(95, 5, footerText, "", 0, "L", false, 0, "")
		pdf.CellFormat(95, 5, fmt.Sprintf("Page %d of %d", i, totalPages), "", 0, "R", false, 0, "")
	}

	return pdf, nil
}

// hexToRGB converts a hex color string to RGB uint8 values.
func hexToRGB(hex string) (uint8, uint8, uint8) {
	if len(hex) > 0 && hex[0] == '#' {
		hex = hex[1:]
	}
	if len(hex) != 6 {
		return 26, 115, 232 // default blue
	}
	var r, g, b uint8
	fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	return r, g, b
}

// ═══════════════════════════════════════════════════════════════════════════
// Utilities
// ═══════════════════════════════════════════════════════════════════════════

// UserIsMSPPartner returns true if the given user is associated with an MSP partner.
func UserIsMSPPartner(userId int64) bool {
	user, err := GetUser(userId)
	if err != nil {
		return false
	}
	if user.Role.Slug == RoleMSPPartner || user.Role.Slug == RoleSuperAdmin {
		return true
	}
	var count int
	db.Model(&MSPPartner{}).Where("primary_user_id = ?", userId).Count(&count)
	return count > 0
}

// GetPartnerForUser resolves the MSP partner for a user.
func GetPartnerForUser(userId int64) (MSPPartner, error) {
	if p, err := GetMSPPartnerByUserId(userId); err == nil {
		return p, nil
	}
	user, uErr := GetUser(userId)
	if uErr != nil {
		return MSPPartner{}, ErrMSPNotPartner
	}
	var mapping MSPPartnerClient
	if mErr := db.Where(mspQryOrgActive, user.OrgId, true).First(&mapping).Error; mErr != nil {
		return MSPPartner{}, ErrMSPNotPartner
	}
	return GetMSPPartner(mapping.PartnerId)
}

// ApplyWhiteLabelToOrg copies a partner's default branding to a newly added client org.
func ApplyWhiteLabelToOrg(partnerId, orgId int64) error {
	partnerCfg, err := GetWhiteLabelConfigByPartner(partnerId)
	if err != nil {
		// No partner-level white-label config, skip
		return nil
	}
	if _, existErr := GetWhiteLabelConfig(orgId); existErr == nil {
		return nil
	}
	orgCfg := WhiteLabelConfig{
		OrgId: orgId, PartnerId: partnerId,
		CompanyName: partnerCfg.CompanyName, LogoURL: partnerCfg.LogoURL,
		LogoSmallURL: partnerCfg.LogoSmallURL, PrimaryColor: partnerCfg.PrimaryColor,
		SecondaryColor: partnerCfg.SecondaryColor, AccentColor: partnerCfg.AccentColor,
		BackgroundColor: partnerCfg.BackgroundColor, FontFamily: partnerCfg.FontFamily,
		LoginPageTitle: partnerCfg.LoginPageTitle, LoginPageMessage: partnerCfg.LoginPageMessage,
		FooterText: partnerCfg.FooterText, SupportEmail: partnerCfg.SupportEmail,
		SupportURL: partnerCfg.SupportURL, CustomCSS: partnerCfg.CustomCSS,
		EmailFromName: partnerCfg.EmailFromName, EmailFooterHTML: partnerCfg.EmailFooterHTML,
		HidePoweredBy: partnerCfg.HidePoweredBy, IsActive: true,
	}
	return SaveWhiteLabelConfig(&orgCfg)
}
