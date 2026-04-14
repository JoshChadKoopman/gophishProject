package models

import (
	"testing"
)

func TestDomainCategoryConstants(t *testing.T) {
	cats := map[string]string{
		DomainCategoryCorporate: "corporate",
		DomainCategoryCloud:     "cloud",
		DomainCategoryFinancial: "financial",
		DomainCategoryShipping:  "shipping",
		DomainCategorySocial:    "social",
		DomainCategoryCustom:    "custom",
	}
	for got, expected := range cats {
		if got != expected {
			t.Errorf("expected %q, got %q", expected, got)
		}
	}
}

func TestDomainHealthConstants(t *testing.T) {
	if DomainHealthHealthy != "healthy" {
		t.Errorf("expected 'healthy', got %q", DomainHealthHealthy)
	}
	if DomainHealthWarning != "warning" {
		t.Errorf("expected 'warning', got %q", DomainHealthWarning)
	}
	if DomainHealthBlacklisted != "blacklisted" {
		t.Errorf("expected 'blacklisted', got %q", DomainHealthBlacklisted)
	}
	if DomainHealthUnknown != "unknown" {
		t.Errorf("expected 'unknown', got %q", DomainHealthUnknown)
	}
}

func TestRotationStrategyConstants(t *testing.T) {
	if RotationRoundRobin != "round_robin" {
		t.Errorf("expected 'round_robin', got %q", RotationRoundRobin)
	}
	if RotationRandom != "random" {
		t.Errorf("expected 'random', got %q", RotationRandom)
	}
	if RotationWeighted != "weighted" {
		t.Errorf("expected 'weighted', got %q", RotationWeighted)
	}
}

func TestWarmupStageConstants(t *testing.T) {
	if WarmupStageCold != 0 {
		t.Errorf("expected 0, got %d", WarmupStageCold)
	}
	if WarmupStageReady != 6 {
		t.Errorf("expected 6, got %d", WarmupStageReady)
	}
}

func TestDailyLimitForWarmupStage(t *testing.T) {
	tests := []struct {
		stage int
		limit int
	}{
		{0, 10},
		{1, 25},
		{2, 50},
		{3, 100},
		{4, 200},
		{5, 400},
		{6, 1000},
	}
	for _, tc := range tests {
		got := DailyLimitForWarmupStage(tc.stage)
		if got != tc.limit {
			t.Errorf("stage %d: expected limit %d, got %d", tc.stage, tc.limit, got)
		}
	}
}

func TestDailyLimitForWarmupStageInvalid(t *testing.T) {
	got := DailyLimitForWarmupStage(99)
	if got != 10 {
		t.Errorf("expected fallback limit 10, got %d", got)
	}
}

func TestDailyLimitForWarmupStageNegative(t *testing.T) {
	got := DailyLimitForWarmupStage(-1)
	if got != 10 {
		t.Errorf("expected fallback limit 10, got %d", got)
	}
}

func TestDailyLimitIncreases(t *testing.T) {
	prev := 0
	for stage := 0; stage <= 6; stage++ {
		limit := DailyLimitForWarmupStage(stage)
		if limit <= prev {
			t.Errorf("stage %d: limit %d not greater than previous %d", stage, limit, prev)
		}
		prev = limit
	}
}

func TestValidDomainCategories(t *testing.T) {
	for _, cat := range []string{"corporate", "cloud", "financial", "shipping", "social", "custom"} {
		if !ValidDomainCategories[cat] {
			t.Errorf("expected %q to be valid category", cat)
		}
	}
	if ValidDomainCategories["invalid"] {
		t.Error("expected 'invalid' to be invalid category")
	}
}

func TestValidRotationStrategies(t *testing.T) {
	for _, s := range []string{"round_robin", "random", "weighted"} {
		if !ValidRotationStrategies[s] {
			t.Errorf("expected %q to be valid strategy", s)
		}
	}
	if ValidRotationStrategies["fifo"] {
		t.Error("expected 'fifo' to be invalid strategy")
	}
}

func TestSendingDomainTableName(t *testing.T) {
	d := SendingDomain{}
	if d.TableName() != "sending_domains" {
		t.Errorf("expected 'sending_domains', got %q", d.TableName())
	}
}

func TestDomainPoolConfigTableName(t *testing.T) {
	c := DomainPoolConfig{}
	if c.TableName() != "domain_pool_configs" {
		t.Errorf("expected 'domain_pool_configs', got %q", c.TableName())
	}
}

func TestSendingDomainStruct(t *testing.T) {
	d := SendingDomain{
		Domain:       "test.example.com",
		DisplayName:  "Test Domain",
		Category:     DomainCategoryCorporate,
		IsActive:     true,
		WarmupStage:  3,
		HealthStatus: DomainHealthHealthy,
	}
	if d.Domain != "test.example.com" {
		t.Errorf("expected domain 'test.example.com', got %q", d.Domain)
	}
	if d.WarmupStage != 3 {
		t.Errorf("expected warmup stage 3, got %d", d.WarmupStage)
	}
}

func TestGenerateFromAddress(t *testing.T) {
	d := SendingDomain{
		Domain:      "it-servicedesk.net",
		DisplayName: "IT Service Desk",
	}
	addr := GenerateFromAddress(d, "")
	if addr != "IT Service Desk <it.service.desk@it-servicedesk.net>" {
		t.Errorf("unexpected from address: %q", addr)
	}
}

func TestGenerateFromAddressCustomName(t *testing.T) {
	d := SendingDomain{Domain: "example.com"}
	addr := GenerateFromAddress(d, "John Smith")
	if addr != "John Smith <john.smith@example.com>" {
		t.Errorf("unexpected from address: %q", addr)
	}
}

func TestBuiltInSendingDomainsCount(t *testing.T) {
	if len(BuiltInSendingDomains) < 15 {
		t.Errorf("expected at least 15 built-in domains, got %d", len(BuiltInSendingDomains))
	}
}

func TestBuiltInSendingDomainsAllActive(t *testing.T) {
	for _, d := range BuiltInSendingDomains {
		if !d.IsActive {
			t.Errorf("built-in domain %q should be active", d.Domain)
		}
		if !d.IsBuiltIn {
			t.Errorf("built-in domain %q should have IsBuiltIn=true", d.Domain)
		}
	}
}

func TestBuiltInSendingDomainsHaveCategories(t *testing.T) {
	categories := make(map[string]bool)
	for _, d := range BuiltInSendingDomains {
		if d.Category == "" {
			t.Errorf("domain %q missing category", d.Domain)
		}
		if !ValidDomainCategories[d.Category] {
			t.Errorf("domain %q has invalid category %q", d.Domain, d.Category)
		}
		categories[d.Category] = true
	}
	// Should have at least 4 different categories
	if len(categories) < 4 {
		t.Errorf("expected at least 4 categories, got %d", len(categories))
	}
}

func TestBuiltInSendingDomainsUnique(t *testing.T) {
	seen := make(map[string]bool)
	for _, d := range BuiltInSendingDomains {
		if seen[d.Domain] {
			t.Errorf("duplicate built-in domain: %q", d.Domain)
		}
		seen[d.Domain] = true
	}
}

func TestDomainPoolSummaryStruct(t *testing.T) {
	s := DomainPoolSummary{
		TotalDomains:   20,
		ActiveDomains:  18,
		HealthyDomains: 15,
		TotalSent:      5000,
		ByCategory:     map[string]int{"corporate": 5, "cloud": 5},
	}
	if s.TotalDomains != 20 {
		t.Errorf("expected 20 total domains, got %d", s.TotalDomains)
	}
	if s.ByCategory["corporate"] != 5 {
		t.Errorf("expected 5 corporate, got %d", s.ByCategory["corporate"])
	}
}

func TestDomainPoolErrors(t *testing.T) {
	if ErrDomainNotFound.Error() != "domain not found" {
		t.Errorf("unexpected error message: %q", ErrDomainNotFound.Error())
	}
	if ErrDomainExists.Error() != "domain already exists in pool" {
		t.Errorf("unexpected error message: %q", ErrDomainExists.Error())
	}
	if ErrNoDomainAvailable.Error() != "no active domain available in pool" {
		t.Errorf("unexpected error message: %q", ErrNoDomainAvailable.Error())
	}
}
