package models

import (
	"testing"
)

func TestBuiltInFrameworkCertsNotEmpty(t *testing.T) {
	if len(BuiltInFrameworkCerts) == 0 {
		t.Fatal("BuiltInFrameworkCerts should not be empty")
	}
}

func TestFrameworkCertUniqueSlugs(t *testing.T) {
	seen := make(map[string]bool)
	for _, c := range BuiltInFrameworkCerts {
		if seen[c.Slug] {
			t.Errorf("duplicate cert slug: %s", c.Slug)
		}
		seen[c.Slug] = true
	}
}

func TestFrameworkCertRequiredFields(t *testing.T) {
	for _, c := range BuiltInFrameworkCerts {
		if c.Slug == "" {
			t.Error("cert has empty slug")
		}
		if c.FrameworkSlug == "" {
			t.Errorf("cert %s has empty framework slug", c.Slug)
		}
		if c.Name == "" {
			t.Errorf("cert %s has empty name", c.Slug)
		}
		if c.Description == "" {
			t.Errorf("cert %s has empty description", c.Slug)
		}
		if c.IssuingAuthority == "" {
			t.Errorf("cert %s has empty issuing authority", c.Slug)
		}
		if c.ValidityMonths <= 0 {
			t.Errorf("cert %s has invalid validity: %d months", c.Slug, c.ValidityMonths)
		}
		if c.MinOverallScore <= 0 || c.MinOverallScore > 100 {
			t.Errorf("cert %s has invalid min overall score: %.1f", c.Slug, c.MinOverallScore)
		}
		if c.MinControlsPassed <= 0 || c.MinControlsPassed > 1 {
			t.Errorf("cert %s has invalid min controls passed: %.2f", c.Slug, c.MinControlsPassed)
		}
	}
}

func TestFrameworkCertIssuingAuthority(t *testing.T) {
	for _, c := range BuiltInFrameworkCerts {
		if c.IssuingAuthority != certIssuer {
			t.Errorf("cert %s has unexpected issuer: %s", c.Slug, c.IssuingAuthority)
		}
	}
}

func TestGetFrameworkComplianceCertFound(t *testing.T) {
	slug := BuiltInFrameworkCerts[0].Slug
	c := GetFrameworkComplianceCert(slug)
	if c == nil {
		t.Fatalf("expected to find cert %s", slug)
	}
	if c.Slug != slug {
		t.Errorf("expected slug %s, got %s", slug, c.Slug)
	}
}

func TestGetFrameworkComplianceCertNotFound(t *testing.T) {
	c := GetFrameworkComplianceCert("nonexistent-cert-xyz")
	if c != nil {
		t.Error("expected nil for nonexistent cert")
	}
}

func TestGetFrameworkComplianceCerts(t *testing.T) {
	certs := GetFrameworkComplianceCerts()
	if len(certs) != len(BuiltInFrameworkCerts) {
		t.Errorf("expected %d certs, got %d", len(BuiltInFrameworkCerts), len(certs))
	}
}

func TestGetCertsForFramework(t *testing.T) {
	fw := BuiltInFrameworkCerts[0].FrameworkSlug
	certs := GetCertsForFramework(fw)
	if len(certs) == 0 {
		t.Errorf("expected certs for framework %s", fw)
	}
	for _, c := range certs {
		if c.FrameworkSlug != fw {
			t.Errorf("expected framework %s, got %s", fw, c.FrameworkSlug)
		}
	}
}

func TestGetCertsForFrameworkNotFound(t *testing.T) {
	certs := GetCertsForFramework("nonexistent-framework")
	if len(certs) != 0 {
		t.Errorf("expected 0 certs for nonexistent framework, got %d", len(certs))
	}
}

func TestMeetsFrameworkCertThresholdPass(t *testing.T) {
	certDef := FrameworkComplianceCert{
		MinOverallScore:   70,
		MinControlsPassed: 0.7,
	}
	summary := FrameworkSummary{
		OverallScore:  80,
		Compliant:     8,
		TotalControls: 10,
	}
	if !meetsFrameworkCertThreshold(certDef, summary) {
		t.Error("expected to meet threshold")
	}
}

func TestMeetsFrameworkCertThresholdFailScore(t *testing.T) {
	certDef := FrameworkComplianceCert{
		MinOverallScore:   70,
		MinControlsPassed: 0.7,
	}
	summary := FrameworkSummary{
		OverallScore:  50,
		Compliant:     8,
		TotalControls: 10,
	}
	if meetsFrameworkCertThreshold(certDef, summary) {
		t.Error("expected not to meet threshold (low score)")
	}
}

func TestMeetsFrameworkCertThresholdFailControls(t *testing.T) {
	certDef := FrameworkComplianceCert{
		MinOverallScore:   70,
		MinControlsPassed: 0.7,
	}
	summary := FrameworkSummary{
		OverallScore:  80,
		Compliant:     5,
		TotalControls: 10,
	}
	if meetsFrameworkCertThreshold(certDef, summary) {
		t.Error("expected not to meet threshold (low controls)")
	}
}

func TestMeetsFrameworkCertThresholdZeroControls(t *testing.T) {
	certDef := FrameworkComplianceCert{
		MinOverallScore:   70,
		MinControlsPassed: 0.7,
	}
	summary := FrameworkSummary{
		OverallScore:  80,
		TotalControls: 0,
	}
	if meetsFrameworkCertThreshold(certDef, summary) {
		t.Error("expected not to meet threshold with zero total controls")
	}
}

func TestOrgFrameworkCertTableName(t *testing.T) {
	c := OrgFrameworkCert{}
	if c.TableName() != "org_framework_certs" {
		t.Errorf("expected table name 'org_framework_certs', got '%s'", c.TableName())
	}
}

func TestOrgFrameworkCertDefaults(t *testing.T) {
	c := OrgFrameworkCert{}
	if c.Id != 0 || c.OrgId != 0 || c.IsRevoked != false {
		t.Error("default OrgFrameworkCert should have zero/false values")
	}
	if c.FrameworkScore != 0 || c.ControlsPassed != 0 || c.TotalControls != 0 {
		t.Error("default OrgFrameworkCert should have zero score values")
	}
}

func TestFrameworkCertSummaryDefaults(t *testing.T) {
	s := FrameworkCertSummary{}
	if s.Available != 0 || s.Earned != 0 || s.Expired != 0 || s.Qualifying != 0 {
		t.Error("default FrameworkCertSummary should have zero values")
	}
}

func TestFrameworkCertCoverage(t *testing.T) {
	// Verify certs exist for expected frameworks
	expectedFrameworks := []string{"nis2", "dora", "hipaa", "pci_dss", "nist_csf", "iso27001", "gdpr", "soc2"}
	for _, fw := range expectedFrameworks {
		certs := GetCertsForFramework(fw)
		if len(certs) == 0 {
			t.Errorf("no certs found for framework %s", fw)
		}
	}
}
