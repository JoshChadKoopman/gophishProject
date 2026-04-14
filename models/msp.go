package models

import (
	"errors"
	"time"

	log "github.com/gophish/gophish/logger"
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
		log.Infof("MSP: No partner-level white-label config for partner %d, skipping", partnerId)
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
