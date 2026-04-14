package models

import (
	"testing"

	"github.com/gophish/gophish/config"
)

func setupTierTest(t *testing.T) func() {
	t.Helper()
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	if err := Setup(conf); err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	return func() { /* no teardown needed for in-memory DB */ }
}

func TestGetSubscriptionTiersOrdering(t *testing.T) {
	teardown := setupTierTest(t)
	defer teardown()

	tiers, err := GetSubscriptionTiers()
	if err != nil {
		t.Fatalf("GetSubscriptionTiers: %v", err)
	}
	if len(tiers) < 2 {
		t.Fatalf("expected at least 2 seeded tiers, got %d", len(tiers))
	}
	// Tiers should be ordered by sort_order
	for i := 1; i < len(tiers); i++ {
		if tiers[i].SortOrder < tiers[i-1].SortOrder {
			t.Fatal("tiers should be ordered by sort_order asc")
		}
	}
}

func TestGetSubscriptionTierById(t *testing.T) {
	teardown := setupTierTest(t)
	defer teardown()

	tier, err := GetSubscriptionTier(1)
	if err != nil {
		t.Fatalf("GetSubscriptionTier(1): %v", err)
	}
	if tier.Name == "" {
		t.Fatal("expected non-empty tier name")
	}
	if tier.Id != 1 {
		t.Fatalf("expected id 1, got %d", tier.Id)
	}

	// Non-existent tier
	_, err = GetSubscriptionTier(99999)
	if err == nil {
		t.Fatal("expected error for non-existent tier")
	}
}

func TestGetSubscriptionTierBySlug(t *testing.T) {
	teardown := setupTierTest(t)
	defer teardown()

	tiers, _ := GetSubscriptionTiers()
	if len(tiers) == 0 {
		t.Skip("no tiers seeded")
	}
	slug := tiers[0].Slug
	tier, err := GetSubscriptionTierBySlug(slug)
	if err != nil {
		t.Fatalf("GetSubscriptionTierBySlug(%q): %v", slug, err)
	}
	if tier.Slug != slug {
		t.Fatalf("expected slug %q, got %q", slug, tier.Slug)
	}

	_, err = GetSubscriptionTierBySlug("nonexistent-tier-slug-zzz")
	if err == nil {
		t.Fatal("expected error for non-existent slug")
	}
}

func TestGetTierFeaturesNonEmpty(t *testing.T) {
	teardown := setupTierTest(t)
	defer teardown()

	tiers, _ := GetSubscriptionTiers()
	// Find a tier that should have features (higher tiers have more)
	var tierWithFeatures SubscriptionTier
	for _, tier := range tiers {
		if len(tier.Features) > 0 {
			tierWithFeatures = tier
			break
		}
	}
	if tierWithFeatures.Id == 0 {
		t.Skip("no tier with features found")
	}

	features, err := GetTierFeatures(tierWithFeatures.Id)
	if err != nil {
		t.Fatalf("GetTierFeatures: %v", err)
	}
	if len(features) == 0 {
		t.Fatal("expected features for this tier")
	}
	// All features should be non-empty strings
	for _, f := range features {
		if f == "" {
			t.Fatal("feature slug should not be empty")
		}
	}
}

func TestOrgHasFeature(t *testing.T) {
	teardown := setupTierTest(t)
	defer teardown()

	// Org 1 is seeded by the migration. Check it has/doesn't have features.
	org, err := GetOrganization(1)
	if err != nil {
		t.Fatalf("GetOrganization(1): %v", err)
	}
	features, _ := GetTierFeatures(org.TierId)
	if len(features) > 0 {
		if !OrgHasFeature(1, features[0]) {
			t.Fatalf("org 1 should have feature %q from its tier", features[0])
		}
	}

	// A feature not in any tier
	if OrgHasFeature(1, "totally_fake_feature_xyzzy") {
		t.Fatal("org should not have a non-existent feature")
	}

	// Non-existent org
	if OrgHasFeature(99999, FeatureBasicBRS) {
		t.Fatal("non-existent org should not have any features")
	}
}

func TestGetOrgFeaturesMap(t *testing.T) {
	teardown := setupTierTest(t)
	defer teardown()

	features := GetOrgFeatures(1)
	if features == nil {
		t.Fatal("expected non-nil feature map")
	}

	// Non-existent org should return empty map
	features2 := GetOrgFeatures(99999)
	if len(features2) != 0 {
		t.Fatalf("expected empty feature map for non-existent org, got %d entries", len(features2))
	}
}

func TestFeatureConstants(t *testing.T) {
	// Verify all feature constants are non-empty and unique
	constants := []string{
		FeatureBasicBRS, FeatureAdvancedBRS, FeatureAITemplates,
		FeatureAutopilot, FeatureAcademyAdvanced, FeatureGamification,
		FeatureReportButton, FeatureThreatAlertsRead, FeatureThreatAlertsCreate,
		FeatureBoardReports, FeatureI18NFull, FeatureSCIM, FeatureZIM,
		FeatureAIAssistant, FeaturePowerBI, FeatureComplianceMapping,
		FeatureMSPWhitelabel, FeatureCyberHygiene, FeatureCustomTrainingBuilder,
	}
	seen := map[string]bool{}
	for _, c := range constants {
		if c == "" {
			t.Fatal("feature constant should not be empty")
		}
		if seen[c] {
			t.Fatalf("duplicate feature constant: %q", c)
		}
		seen[c] = true
	}
}
