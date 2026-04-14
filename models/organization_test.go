package models

import (
	"testing"

	"github.com/gophish/gophish/config"
)

// setupOrgTest initialises an in-memory DB for organization/tier tests.
// The seed migrations create org id=1 ("Default", tier_id=4) and
// subscription tiers 1-4 with their associated features.
func setupOrgTest(t *testing.T) func() {
	t.Helper()
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	if err := Setup(conf); err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	// clean any seeded data
	return func() {}
}

// ===================== Organization CRUD =====================

func TestGetOrganization_Found(t *testing.T) {
	teardown := setupOrgTest(t)
	defer teardown()

	org, err := GetOrganization(1)
	if err != nil {
		t.Fatalf("GetOrganization(1) returned error: %v", err)
	}
	if org.Id != 1 {
		t.Fatalf("expected org id 1, got %d", org.Id)
	}
	if org.Name != "Default" {
		t.Fatalf("expected org name 'Default', got %q", org.Name)
	}
	if org.Slug != "default" {
		t.Fatalf("expected org slug 'default', got %q", org.Slug)
	}
}

func TestGetOrganization_NotFound(t *testing.T) {
	teardown := setupOrgTest(t)
	defer teardown()

	_, err := GetOrganization(99999)
	if err == nil {
		t.Fatal("expected error for non-existent org, got nil")
	}
}

func TestGetOrganizations(t *testing.T) {
	teardown := setupOrgTest(t)
	defer teardown()

	orgs, err := GetOrganizations()
	if err != nil {
		t.Fatalf("GetOrganizations returned error: %v", err)
	}
	if len(orgs) < 1 {
		t.Fatalf("expected at least 1 seeded organization, got %d", len(orgs))
	}
	found := false
	for _, o := range orgs {
		if o.Id == 1 && o.Slug == "default" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("seeded 'default' organization not found in GetOrganizations result")
	}
}

func TestPostOrganization_Valid(t *testing.T) {
	teardown := setupOrgTest(t)
	defer teardown()

	org := &Organization{
		Name:     "Acme Corp",
		Slug:     "acme-corp",
		Tier:     "free",
		TierId:   1,
		MaxUsers: 10,
	}
	if err := PostOrganization(org); err != nil {
		t.Fatalf("PostOrganization failed: %v", err)
	}
	if org.Id == 0 {
		t.Fatal("expected non-zero id after PostOrganization")
	}
	if org.CreatedDate.IsZero() {
		t.Fatal("expected CreatedDate to be set")
	}
	if org.ModifiedDate.IsZero() {
		t.Fatal("expected ModifiedDate to be set")
	}

	// Verify we can retrieve the newly created org
	fetched, err := GetOrganization(org.Id)
	if err != nil {
		t.Fatalf("GetOrganization(%d) failed: %v", org.Id, err)
	}
	if fetched.Name != "Acme Corp" {
		t.Fatalf("expected name 'Acme Corp', got %q", fetched.Name)
	}
	if fetched.Slug != "acme-corp" {
		t.Fatalf("expected slug 'acme-corp', got %q", fetched.Slug)
	}
}

func TestPostOrganization_MissingName(t *testing.T) {
	teardown := setupOrgTest(t)
	defer teardown()

	org := &Organization{
		Slug: "no-name",
	}
	err := PostOrganization(org)
	if err != ErrOrgNameNotSpecified {
		t.Fatalf("expected ErrOrgNameNotSpecified, got %v", err)
	}
}

func TestPostOrganization_MissingSlug(t *testing.T) {
	teardown := setupOrgTest(t)
	defer teardown()

	org := &Organization{
		Name: "Has Name",
	}
	err := PostOrganization(org)
	if err != ErrOrgSlugNotSpecified {
		t.Fatalf("expected ErrOrgSlugNotSpecified, got %v", err)
	}
}

func TestPutOrganization(t *testing.T) {
	teardown := setupOrgTest(t)
	defer teardown()

	// Create an org first
	org := &Organization{
		Name:   "Original Name",
		Slug:   "original-slug",
		Tier:   "free",
		TierId: 1,
	}
	if err := PostOrganization(org); err != nil {
		t.Fatalf("PostOrganization failed: %v", err)
	}

	// Update the org
	org.Name = "Updated Name"
	org.MaxUsers = 50
	if err := PutOrganization(org); err != nil {
		t.Fatalf("PutOrganization failed: %v", err)
	}

	// Verify update persisted
	fetched, err := GetOrganization(org.Id)
	if err != nil {
		t.Fatalf("GetOrganization(%d) failed: %v", org.Id, err)
	}
	if fetched.Name != "Updated Name" {
		t.Fatalf("expected name 'Updated Name', got %q", fetched.Name)
	}
	if fetched.MaxUsers != 50 {
		t.Fatalf("expected max_users 50, got %d", fetched.MaxUsers)
	}
}

func TestDeleteOrganization(t *testing.T) {
	teardown := setupOrgTest(t)
	defer teardown()

	// Create an org to delete
	org := &Organization{
		Name:   "To Delete",
		Slug:   "to-delete",
		Tier:   "free",
		TierId: 1,
	}
	if err := PostOrganization(org); err != nil {
		t.Fatalf("PostOrganization failed: %v", err)
	}

	if err := DeleteOrganization(org.Id); err != nil {
		t.Fatalf("DeleteOrganization(%d) failed: %v", org.Id, err)
	}

	// Verify it is gone
	_, err := GetOrganization(org.Id)
	if err == nil {
		t.Fatal("expected error after deleting organization, got nil")
	}
}

// ===================== Org Counts =====================

func TestGetOrgUserCount(t *testing.T) {
	teardown := setupOrgTest(t)
	defer teardown()

	// The seeded admin user has org_id=1
	count, err := GetOrgUserCount(1)
	if err != nil {
		t.Fatalf("GetOrgUserCount(1) failed: %v", err)
	}
	if count < 1 {
		t.Fatalf("expected at least 1 user for org 1, got %d", count)
	}

	// Non-existent org should have zero users
	count, err = GetOrgUserCount(99999)
	if err != nil {
		t.Fatalf("GetOrgUserCount(99999) failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 users for non-existent org, got %d", count)
	}
}

func TestGetOrgCampaignCount(t *testing.T) {
	teardown := setupOrgTest(t)
	defer teardown()

	// No campaigns seeded, so org 1 should have 0
	count, err := GetOrgCampaignCount(1)
	if err != nil {
		t.Fatalf("GetOrgCampaignCount(1) failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 campaigns for org 1, got %d", count)
	}
}

// ===================== scopeQuery Isolation =====================

func TestScopeQueryIsolation(t *testing.T) {
	teardown := setupOrgTest(t)
	defer teardown()

	// Create two additional orgs
	org1 := &Organization{Name: "Org Alpha", Slug: "org-alpha", Tier: "free", TierId: 1}
	org2 := &Organization{Name: "Org Beta", Slug: "org-beta", Tier: "free", TierId: 1}
	if err := PostOrganization(org1); err != nil {
		t.Fatalf("PostOrganization(org1) failed: %v", err)
	}
	if err := PostOrganization(org2); err != nil {
		t.Fatalf("PostOrganization(org2) failed: %v", err)
	}

	// Create a group in org Alpha
	g1 := Group{Name: "Alpha Group", UserId: 1, OrgId: org1.Id}
	g1.Targets = []Target{
		{BaseRecipient: BaseRecipient{Email: "alpha@example.com"}},
	}
	if err := PostGroup(&g1); err != nil {
		t.Fatalf("PostGroup for org Alpha failed: %v", err)
	}

	// Create a group in org Beta
	g2 := Group{Name: "Beta Group", UserId: 1, OrgId: org2.Id}
	g2.Targets = []Target{
		{BaseRecipient: BaseRecipient{Email: "beta@example.com"}},
	}
	if err := PostGroup(&g2); err != nil {
		t.Fatalf("PostGroup for org Beta failed: %v", err)
	}

	// Scope to org Alpha -- should only see Alpha's group
	scopeAlpha := OrgScope{OrgId: org1.Id, UserId: 1, IsSuperAdmin: false}
	var alphaGroups []Group
	err := scopeQuery(db.Model(&Group{}), scopeAlpha).Find(&alphaGroups).Error
	if err != nil {
		t.Fatalf("scopeQuery for org Alpha failed: %v", err)
	}
	for _, g := range alphaGroups {
		if g.OrgId != org1.Id {
			t.Fatalf("scope leak: found group with org_id=%d in org Alpha scope", g.OrgId)
		}
	}
	if len(alphaGroups) == 0 {
		t.Fatal("expected at least 1 group for org Alpha scope, got 0")
	}

	// Scope to org Beta -- should only see Beta's group
	scopeBeta := OrgScope{OrgId: org2.Id, UserId: 1, IsSuperAdmin: false}
	var betaGroups []Group
	err = scopeQuery(db.Model(&Group{}), scopeBeta).Find(&betaGroups).Error
	if err != nil {
		t.Fatalf("scopeQuery for org Beta failed: %v", err)
	}
	for _, g := range betaGroups {
		if g.OrgId != org2.Id {
			t.Fatalf("scope leak: found group with org_id=%d in org Beta scope", g.OrgId)
		}
	}
	if len(betaGroups) == 0 {
		t.Fatal("expected at least 1 group for org Beta scope, got 0")
	}

	// Superadmin scope should see all groups
	scopeSuper := OrgScope{OrgId: 0, UserId: 1, IsSuperAdmin: true}
	var allGroups []Group
	err = scopeQuery(db.Model(&Group{}), scopeSuper).Find(&allGroups).Error
	if err != nil {
		t.Fatalf("scopeQuery for superadmin failed: %v", err)
	}
	if len(allGroups) < 2 {
		t.Fatalf("expected at least 2 groups for superadmin scope, got %d", len(allGroups))
	}
}

// ===================== Subscription Tiers =====================

func TestGetSubscriptionTiers(t *testing.T) {
	teardown := setupOrgTest(t)
	defer teardown()

	tiers, err := GetSubscriptionTiers()
	if err != nil {
		t.Fatalf("GetSubscriptionTiers failed: %v", err)
	}
	if len(tiers) < 4 {
		t.Fatalf("expected at least 4 seeded tiers, got %d", len(tiers))
	}

	// Verify ordering by sort_order
	for i := 1; i < len(tiers); i++ {
		if tiers[i].SortOrder < tiers[i-1].SortOrder {
			t.Fatalf("tiers not in ascending sort_order: tier %d (sort=%d) before tier %d (sort=%d)",
				tiers[i-1].Id, tiers[i-1].SortOrder, tiers[i].Id, tiers[i].SortOrder)
		}
	}
}

func TestGetSubscriptionTier_Valid(t *testing.T) {
	teardown := setupOrgTest(t)
	defer teardown()

	tier, err := GetSubscriptionTier(1)
	if err != nil {
		t.Fatalf("GetSubscriptionTier(1) failed: %v", err)
	}
	if tier.Slug != "core" {
		t.Fatalf("expected tier slug 'core', got %q", tier.Slug)
	}
	if tier.Name != "Core" {
		t.Fatalf("expected tier name 'Core', got %q", tier.Name)
	}
}

func TestGetSubscriptionTier_Invalid(t *testing.T) {
	teardown := setupOrgTest(t)
	defer teardown()

	_, err := GetSubscriptionTier(99999)
	if err == nil {
		t.Fatal("expected error for non-existent tier id, got nil")
	}
}

func TestGetSubscriptionTierBySlug_Valid(t *testing.T) {
	teardown := setupOrgTest(t)
	defer teardown()

	tier, err := GetSubscriptionTierBySlug("enterprise")
	if err != nil {
		t.Fatalf("GetSubscriptionTierBySlug('enterprise') failed: %v", err)
	}
	if tier.Id != 4 {
		t.Fatalf("expected tier id 4, got %d", tier.Id)
	}
	if tier.Name != "Enterprise" {
		t.Fatalf("expected tier name 'Enterprise', got %q", tier.Name)
	}
}

func TestGetSubscriptionTierBySlug_Invalid(t *testing.T) {
	teardown := setupOrgTest(t)
	defer teardown()

	_, err := GetSubscriptionTierBySlug("nonexistent-tier")
	if err == nil {
		t.Fatal("expected error for non-existent tier slug, got nil")
	}
}

// ===================== Tier Features =====================

func TestGetTierFeatures(t *testing.T) {
	teardown := setupOrgTest(t)
	defer teardown()

	// Core tier (id=1) should have seeded features
	features, err := GetTierFeatures(1)
	if err != nil {
		t.Fatalf("GetTierFeatures(1) failed: %v", err)
	}
	if len(features) < 1 {
		t.Fatalf("expected at least 1 feature for core tier, got %d", len(features))
	}

	// Verify basic_brs is in core tier
	found := false
	for _, f := range features {
		if f == FeatureBasicBRS {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected '%s' in core tier features, got %v", FeatureBasicBRS, features)
	}

	// Enterprise tier (id=4) should have more features than core
	entFeatures, err := GetTierFeatures(4)
	if err != nil {
		t.Fatalf("GetTierFeatures(4) failed: %v", err)
	}
	if len(entFeatures) <= len(features) {
		t.Fatalf("expected enterprise tier to have more features (%d) than core (%d)",
			len(entFeatures), len(features))
	}
}

func TestGetTierFeatures_Populated(t *testing.T) {
	teardown := setupOrgTest(t)
	defer teardown()

	// GetSubscriptionTier should also populate the Features slice
	tier, err := GetSubscriptionTier(4)
	if err != nil {
		t.Fatalf("GetSubscriptionTier(4) failed: %v", err)
	}
	if len(tier.Features) == 0 {
		t.Fatal("expected Features to be populated on enterprise tier")
	}
}

// ===================== OrgHasFeature =====================

func TestOrgHasFeature_Positive(t *testing.T) {
	teardown := setupOrgTest(t)
	defer teardown()

	// Org 1 is on enterprise tier (id=4) which includes basic_brs
	if !OrgHasFeature(1, FeatureBasicBRS) {
		t.Fatalf("expected org 1 to have feature '%s'", FeatureBasicBRS)
	}
	// Enterprise tier also includes msp_whitelabel
	if !OrgHasFeature(1, FeatureMSPWhitelabel) {
		t.Fatalf("expected org 1 to have feature '%s'", FeatureMSPWhitelabel)
	}
}

func TestOrgHasFeature_Negative(t *testing.T) {
	teardown := setupOrgTest(t)
	defer teardown()

	// A completely made-up feature should not be present
	if OrgHasFeature(1, "nonexistent_feature_slug") {
		t.Fatal("expected org 1 NOT to have feature 'nonexistent_feature_slug'")
	}

	// Non-existent org should return false
	if OrgHasFeature(99999, FeatureBasicBRS) {
		t.Fatal("expected non-existent org NOT to have any feature")
	}
}

func TestOrgHasFeature_TierBoundary(t *testing.T) {
	teardown := setupOrgTest(t)
	defer teardown()

	// Create an org on the core tier (id=1)
	coreOrg := &Organization{
		Name:   "Core Org",
		Slug:   "core-org",
		Tier:   "core",
		TierId: 1,
	}
	if err := PostOrganization(coreOrg); err != nil {
		t.Fatalf("PostOrganization failed: %v", err)
	}

	// Core tier has basic_brs
	if !OrgHasFeature(coreOrg.Id, FeatureBasicBRS) {
		t.Fatalf("expected core org to have feature '%s'", FeatureBasicBRS)
	}

	// Core tier does NOT have msp_whitelabel (enterprise only)
	if OrgHasFeature(coreOrg.Id, FeatureMSPWhitelabel) {
		t.Fatalf("expected core org NOT to have feature '%s'", FeatureMSPWhitelabel)
	}

	// Core tier does NOT have autopilot (all_in_one and above)
	if OrgHasFeature(coreOrg.Id, FeatureAutopilot) {
		t.Fatalf("expected core org NOT to have feature '%s'", FeatureAutopilot)
	}
}

// ===================== GetOrgFeatures =====================

func TestGetOrgFeatures(t *testing.T) {
	teardown := setupOrgTest(t)
	defer teardown()

	features := GetOrgFeatures(1)
	if len(features) == 0 {
		t.Fatal("expected non-empty feature map for org 1")
	}

	// Verify basic_brs is present and true
	if !features[FeatureBasicBRS] {
		t.Fatalf("expected feature map to include '%s' = true", FeatureBasicBRS)
	}

	// A feature not in the map should return the zero value (false)
	if features["nonexistent_feature_slug"] {
		t.Fatal("expected missing feature to be false in map")
	}
}

func TestGetOrgFeatures_NonExistentOrg(t *testing.T) {
	teardown := setupOrgTest(t)
	defer teardown()

	features := GetOrgFeatures(99999)
	if len(features) != 0 {
		t.Fatalf("expected empty feature map for non-existent org, got %d entries", len(features))
	}
}
