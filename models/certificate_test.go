package models

import (
	"strings"
	"testing"
	"time"

	"github.com/gophish/gophish/config"
)

// Shared test constants to avoid lint warnings about literal duplication.
const (
	testSlugPhishing = "phishing-defense-specialist"
	testSlugGDPR     = "gdpr-awareness"
	testSlugNIS2     = "nis2-compliance"
	testMsgWrongCert = "wrong certificate returned"
)

// setupCertificateTest initialises an in-memory DB for certificate tests.
func setupCertificateTest(t *testing.T) func() {
	t.Helper()
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	if err := Setup(conf); err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	db.Exec("DELETE FROM certificates")
	return func() {
		db.Exec("DELETE FROM certificates")
	}
}

// =====================================================================
// IssueCertificate (default template)
// =====================================================================

func TestIssueCertificate(t *testing.T) {
	teardown := setupCertificateTest(t)
	defer teardown()

	cert, err := IssueCertificate(1, 100, 0)
	if err != nil {
		t.Fatalf("IssueCertificate failed: %v", err)
	}
	if cert.Id == 0 {
		t.Fatal("expected non-zero ID")
	}
	if cert.VerificationCode == "" {
		t.Fatal("expected verification code to be set")
	}
	if len(cert.VerificationCode) != verificationCodeLen {
		t.Fatalf("expected code length %d, got %d", verificationCodeLen, len(cert.VerificationCode))
	}
	if cert.IssuedDate.IsZero() {
		t.Fatal("expected IssuedDate to be set")
	}
	if cert.TemplateSlug != "cybersecurity-awareness-foundation" {
		t.Fatalf("expected default template slug, got %q", cert.TemplateSlug)
	}
}

func TestIssueCertificateWithQuiz(t *testing.T) {
	teardown := setupCertificateTest(t)
	defer teardown()

	cert, err := IssueCertificate(1, 100, 42)
	if err != nil {
		t.Fatalf("IssueCertificate failed: %v", err)
	}
	if cert.QuizAttemptId != 42 {
		t.Fatalf("expected quiz_attempt_id 42, got %d", cert.QuizAttemptId)
	}
}

// =====================================================================
// IssueCertificateWithTemplate
// =====================================================================

func TestIssueCertificateWithTemplate(t *testing.T) {
	teardown := setupCertificateTest(t)
	defer teardown()

	cert, err := IssueCertificateWithTemplate(1, 100, 0, testSlugPhishing)
	if err != nil {
		t.Fatalf("IssueCertificateWithTemplate failed: %v", err)
	}
	if cert.TemplateSlug != testSlugPhishing {
		t.Fatalf("expected slug 'phishing-defense-specialist', got %q", cert.TemplateSlug)
	}
	// Should have expiry set (12 months for phishing-defense-specialist)
	if cert.ExpiresDate.IsZero() {
		t.Fatal("expected ExpiresDate to be set for template with validity")
	}
	// Expiry should be ~12 months from now
	expectedExpiry := time.Now().UTC().AddDate(0, 12, 0)
	diff := cert.ExpiresDate.Sub(expectedExpiry)
	if diff > time.Minute || diff < -time.Minute {
		t.Fatalf("expiry date differs from expected by %v", diff)
	}
}

func TestIssueCertificateMultipleTemplates(t *testing.T) {
	teardown := setupCertificateTest(t)
	defer teardown()

	templates := []string{
		testSlugGDPR, testSlugNIS2, "hipaa-security-awareness",
		"pci-dss-awareness", "iso27001-awareness", "dora-compliance",
	}
	for _, slug := range templates {
		cert, err := IssueCertificateWithTemplate(1, 100, 0, slug)
		if err != nil {
			t.Fatalf("IssueCertificateWithTemplate(%s) failed: %v", slug, err)
		}
		if cert.TemplateSlug != slug {
			t.Fatalf("expected slug %q, got %q", slug, cert.TemplateSlug)
		}
	}
}

// =====================================================================
// Verification code generation
// =====================================================================

func TestGenerateVerificationCode(t *testing.T) {
	code, err := generateVerificationCode()
	if err != nil {
		t.Fatalf("generateVerificationCode failed: %v", err)
	}
	if len(code) != verificationCodeLen {
		t.Fatalf("expected length %d, got %d", verificationCodeLen, len(code))
	}
	// Verify it's alphanumeric
	for _, c := range code {
		if !strings.Contains(verificationCodeChars, string(c)) {
			t.Fatalf("unexpected character %q in verification code", c)
		}
	}
}

func TestGenerateVerificationCodeUniqueness(t *testing.T) {
	codes := make(map[string]bool)
	for i := 0; i < 100; i++ {
		code, err := generateVerificationCode()
		if err != nil {
			t.Fatalf("generateVerificationCode failed: %v", err)
		}
		if codes[code] {
			t.Fatalf("duplicate code generated: %s", code)
		}
		codes[code] = true
	}
}

func TestGenerateVerificationCodeFormatted(t *testing.T) {
	raw, formatted, err := GenerateVerificationCodeFormatted()
	if err != nil {
		t.Fatalf("GenerateVerificationCodeFormatted failed: %v", err)
	}
	if len(raw) != verificationCodeLen {
		t.Fatalf("expected raw length %d, got %d", verificationCodeLen, len(raw))
	}
	if !strings.HasPrefix(formatted, "CERT-") {
		t.Fatalf("expected formatted to start with 'CERT-', got %q", formatted)
	}
	parts := strings.Split(formatted, "-")
	if len(parts) != 5 {
		t.Fatalf("expected 5 parts in formatted code, got %d", len(parts))
	}
}

// =====================================================================
// GetCertificate (lookup)
// =====================================================================

func TestGetCertificate(t *testing.T) {
	teardown := setupCertificateTest(t)
	defer teardown()

	cert, _ := IssueCertificate(1, 100, 0)

	found, err := GetCertificate(cert.VerificationCode)
	if err != nil {
		t.Fatalf("GetCertificate failed: %v", err)
	}
	if found.Id != cert.Id {
		t.Fatal(testMsgWrongCert)
	}
}

func TestGetCertificateFormattedCode(t *testing.T) {
	teardown := setupCertificateTest(t)
	defer teardown()

	cert, _ := IssueCertificate(1, 100, 0)
	// Look up using formatted code
	formatted := "CERT-" + cert.VerificationCode[0:4] + "-" + cert.VerificationCode[4:8] +
		"-" + cert.VerificationCode[8:12] + "-" + cert.VerificationCode[12:16]

	found, err := GetCertificate(formatted)
	if err != nil {
		t.Fatalf("GetCertificate with formatted code failed: %v", err)
	}
	if found.Id != cert.Id {
		t.Fatal(testMsgWrongCert)
	}
}

func TestGetCertificateNotFound(t *testing.T) {
	teardown := setupCertificateTest(t)
	defer teardown()

	_, err := GetCertificate("nonexistent1234567890")
	if err == nil {
		t.Fatal("expected error for non-existent code")
	}
}

// =====================================================================
// GetCertificatesForUser
// =====================================================================

func TestGetCertificatesForUser(t *testing.T) {
	teardown := setupCertificateTest(t)
	defer teardown()

	IssueCertificate(1, 100, 0)
	IssueCertificate(1, 200, 0)
	IssueCertificate(2, 100, 0) // different user

	certs, err := GetCertificatesForUser(1)
	if err != nil {
		t.Fatalf("GetCertificatesForUser failed: %v", err)
	}
	if len(certs) != 2 {
		t.Fatalf("expected 2 certs, got %d", len(certs))
	}
}

func TestGetCertificatesForUserEmpty(t *testing.T) {
	teardown := setupCertificateTest(t)
	defer teardown()

	certs, err := GetCertificatesForUser(999)
	if err != nil {
		t.Fatalf("GetCertificatesForUser failed: %v", err)
	}
	if len(certs) != 0 {
		t.Fatalf("expected 0 certs, got %d", len(certs))
	}
}

// =====================================================================
// GetCertificateForCourse
// =====================================================================

func TestGetCertificateForCourse(t *testing.T) {
	teardown := setupCertificateTest(t)
	defer teardown()

	IssueCertificate(1, 100, 0)

	found, err := GetCertificateForCourse(1, 100)
	if err != nil {
		t.Fatalf("GetCertificateForCourse failed: %v", err)
	}
	if found.UserId != 1 || found.PresentationId != 100 {
		t.Fatal(testMsgWrongCert)
	}
}

// =====================================================================
// GetCertificatesByTemplate
// =====================================================================

func TestGetCertificatesByTemplate(t *testing.T) {
	teardown := setupCertificateTest(t)
	defer teardown()

	IssueCertificateWithTemplate(1, 100, 0, testSlugGDPR)
	IssueCertificateWithTemplate(2, 200, 0, testSlugGDPR)
	IssueCertificateWithTemplate(3, 300, 0, testSlugNIS2)

	certs, err := GetCertificatesByTemplate(testSlugGDPR)
	if err != nil {
		t.Fatalf("GetCertificatesByTemplate failed: %v", err)
	}
	if len(certs) != 2 {
		t.Fatalf("expected 2, got %d", len(certs))
	}
}

// =====================================================================
// Counts
// =====================================================================

func TestGetCertificateCount(t *testing.T) {
	teardown := setupCertificateTest(t)
	defer teardown()

	IssueCertificate(1, 100, 0)
	IssueCertificate(1, 200, 0)

	count := GetCertificateCount(1)
	if count != 2 {
		t.Fatalf("expected 2, got %d", count)
	}
}

func TestGetActiveCertificateCount(t *testing.T) {
	teardown := setupCertificateTest(t)
	defer teardown()

	c1, _ := IssueCertificate(1, 100, 0)
	IssueCertificate(1, 200, 0)

	// Revoke one
	RevokeCertificate(c1.Id)

	count := GetActiveCertificateCount(1)
	if count != 1 {
		t.Fatalf("expected 1 active, got %d", count)
	}
}

// =====================================================================
// Revocation
// =====================================================================

func TestRevokeCertificate(t *testing.T) {
	teardown := setupCertificateTest(t)
	defer teardown()

	cert, _ := IssueCertificate(1, 100, 0)

	if err := RevokeCertificate(cert.Id); err != nil {
		t.Fatalf("RevokeCertificate failed: %v", err)
	}

	found, _ := GetCertificate(cert.VerificationCode)
	if !found.IsRevoked {
		t.Fatal("expected certificate to be revoked")
	}
	if found.RevokedDate.IsZero() {
		t.Fatal("expected RevokedDate to be set")
	}
}

// =====================================================================
// Validity check
// =====================================================================

func TestIsCertificateValid(t *testing.T) {
	teardown := setupCertificateTest(t)
	defer teardown()

	cert, _ := IssueCertificate(1, 100, 0)

	if !IsCertificateValid(*cert) {
		t.Fatal("newly issued cert should be valid")
	}
}

func TestIsCertificateValidRevoked(t *testing.T) {
	teardown := setupCertificateTest(t)
	defer teardown()

	cert, _ := IssueCertificate(1, 100, 0)
	RevokeCertificate(cert.Id)

	found, _ := GetCertificate(cert.VerificationCode)
	if IsCertificateValid(found) {
		t.Fatal("revoked cert should not be valid")
	}
}

func TestIsCertificateValidExpired(t *testing.T) {
	cert := Certificate{
		ExpiresDate: time.Now().UTC().Add(-24 * time.Hour),
	}
	if IsCertificateValid(cert) {
		t.Fatal("expired cert should not be valid")
	}
}

func TestIsCertificateValidNoExpiry(t *testing.T) {
	cert := Certificate{} // zero ExpiresDate = never expires
	if !IsCertificateValid(cert) {
		t.Fatal("cert with no expiry should be valid")
	}
}

// =====================================================================
// Renewal
// =====================================================================

func TestRenewCertificate(t *testing.T) {
	teardown := setupCertificateTest(t)
	defer teardown()

	original, _ := IssueCertificateWithTemplate(1, 100, 0, testSlugGDPR)

	renewed, err := RenewCertificate(original.Id)
	if err != nil {
		t.Fatalf("RenewCertificate failed: %v", err)
	}
	if renewed.Id == original.Id {
		t.Fatal("expected new cert ID")
	}
	if renewed.TemplateSlug != testSlugGDPR {
		t.Fatalf("expected template preserved, got %q", renewed.TemplateSlug)
	}
	if renewed.VerificationCode == original.VerificationCode {
		t.Fatal("expected new verification code")
	}
}

func TestRenewCertificateNotFound(t *testing.T) {
	teardown := setupCertificateTest(t)
	defer teardown()

	_, err := RenewCertificate(999)
	if err == nil {
		t.Fatal("expected error for non-existent cert")
	}
}

// =====================================================================
// Expiring certificates
// =====================================================================

func TestGetExpiringCertificates(t *testing.T) {
	teardown := setupCertificateTest(t)
	defer teardown()

	// Create cert expiring in 15 days
	cert := &Certificate{
		UserId:           1,
		PresentationId:   100,
		VerificationCode: "expiring12345678",
		TemplateSlug:     testSlugGDPR,
		IssuedDate:       time.Now().UTC(),
		ExpiresDate:      time.Now().UTC().AddDate(0, 0, 15),
	}
	db.Save(cert)

	// Create cert expiring in 60 days
	cert2 := &Certificate{
		UserId:           2,
		PresentationId:   200,
		VerificationCode: "notexpiring12345",
		TemplateSlug:     testSlugNIS2,
		IssuedDate:       time.Now().UTC(),
		ExpiresDate:      time.Now().UTC().AddDate(0, 0, 60),
	}
	db.Save(cert2)

	expiring, err := GetExpiringCertificates(30)
	if err != nil {
		t.Fatalf("GetExpiringCertificates failed: %v", err)
	}
	if len(expiring) != 1 {
		t.Fatalf("expected 1 expiring within 30 days, got %d", len(expiring))
	}
}

// =====================================================================
// Active certs filter
// =====================================================================

func TestGetActiveCertificatesForUser(t *testing.T) {
	teardown := setupCertificateTest(t)
	defer teardown()

	c1, _ := IssueCertificate(1, 100, 0)
	IssueCertificate(1, 200, 0)

	// Revoke one
	RevokeCertificate(c1.Id)

	active, err := GetActiveCertificatesForUser(1)
	if err != nil {
		t.Fatalf("GetActiveCertificatesForUser failed: %v", err)
	}
	if len(active) != 1 {
		t.Fatalf("expected 1 active cert, got %d", len(active))
	}
}

// =====================================================================
// Template system
// =====================================================================

func TestSpecializedCertTemplatesCount(t *testing.T) {
	if len(SpecializedCertTemplates) < 15 {
		t.Fatalf("expected at least 15 specialized templates, got %d", len(SpecializedCertTemplates))
	}
}

func TestGetCertificateTemplate(t *testing.T) {
	tmpl := GetCertificateTemplate(testSlugPhishing)
	if tmpl == nil {
		t.Fatal("expected template 'phishing-defense-specialist' to exist")
	}
	if tmpl.Name != "Phishing Defense Specialist" {
		t.Fatalf("expected name 'Phishing Defense Specialist', got %q", tmpl.Name)
	}
	if tmpl.Category != "cybersecurity" {
		t.Fatalf("expected category 'cybersecurity', got %q", tmpl.Category)
	}
}

func TestGetCertificateTemplateNotFound(t *testing.T) {
	tmpl := GetCertificateTemplate("nonexistent-slug")
	if tmpl != nil {
		t.Fatal("expected nil for non-existent template")
	}
}

func TestGetCertificateTemplates(t *testing.T) {
	templates := GetCertificateTemplates()
	if len(templates) < 15 {
		t.Fatalf("expected at least 15 templates, got %d", len(templates))
	}
}

func TestGetCertificateTemplatesByCategory(t *testing.T) {
	compliance := GetCertificateTemplatesByCategory("compliance")
	if len(compliance) < 5 {
		t.Fatalf("expected at least 5 compliance templates, got %d", len(compliance))
	}
	for _, tmpl := range compliance {
		if tmpl.Category != "compliance" {
			t.Fatalf("expected category 'compliance', got %q", tmpl.Category)
		}
	}
}

func TestGetCertificateTemplateCategories(t *testing.T) {
	categories := GetCertificateTemplateCategories()
	if len(categories) < 4 {
		t.Fatalf("expected at least 4 categories, got %d", len(categories))
	}
	// Verify expected categories exist
	catMap := make(map[string]bool)
	for _, c := range categories {
		catMap[c] = true
	}
	for _, expected := range []string{"cybersecurity", "compliance", "technical", "leadership"} {
		if !catMap[expected] {
			t.Fatalf("expected category %q to exist", expected)
		}
	}
}

func TestAllTemplatesHaveRequiredFields(t *testing.T) {
	for _, tmpl := range SpecializedCertTemplates {
		if tmpl.Slug == "" {
			t.Fatal("template has empty slug")
		}
		if tmpl.Name == "" {
			t.Fatalf("template %q has empty name", tmpl.Slug)
		}
		if tmpl.Description == "" {
			t.Fatalf("template %q has empty description", tmpl.Slug)
		}
		if tmpl.Category == "" {
			t.Fatalf("template %q has empty category", tmpl.Slug)
		}
		if tmpl.ColorScheme == "" {
			t.Fatalf("template %q has empty color scheme", tmpl.Slug)
		}
	}
}

func TestTemplateSlugsAreUnique(t *testing.T) {
	seen := make(map[string]bool)
	for _, tmpl := range SpecializedCertTemplates {
		if seen[tmpl.Slug] {
			t.Fatalf("duplicate template slug: %q", tmpl.Slug)
		}
		seen[tmpl.Slug] = true
	}
}

// =====================================================================
// EnrichCertificate
// =====================================================================

func TestEnrichCertificate(t *testing.T) {
	teardown := setupCertificateTest(t)
	defer teardown()

	cert, _ := IssueCertificateWithTemplate(1, 100, 0, testSlugGDPR)

	enriched := EnrichCertificate(*cert)
	if !enriched.IsValid {
		t.Fatal("expected IsValid to be true")
	}
	if enriched.FormattedCode == "" {
		t.Fatal("expected FormattedCode to be set")
	}
	if !strings.HasPrefix(enriched.FormattedCode, "CERT-") {
		t.Fatalf("expected FormattedCode to start with 'CERT-', got %q", enriched.FormattedCode)
	}
	if enriched.Template == nil {
		t.Fatal("expected Template to be set")
	}
	if enriched.Template.Name != "GDPR Awareness Certificate" {
		t.Fatalf("expected template name 'GDPR Awareness Certificate', got %q", enriched.Template.Name)
	}
}

func TestEnrichCertificateRevoked(t *testing.T) {
	teardown := setupCertificateTest(t)
	defer teardown()

	cert, _ := IssueCertificate(1, 100, 0)
	RevokeCertificate(cert.Id)

	found, _ := GetCertificate(cert.VerificationCode)
	enriched := EnrichCertificate(found)
	if enriched.IsValid {
		t.Fatal("expected revoked cert to be invalid")
	}
}

// =====================================================================
// CertificateSummary
// =====================================================================

func TestGetCertificateSummary(t *testing.T) {
	teardown := setupCertificateTest(t)
	defer teardown()

	IssueCertificateWithTemplate(1, 100, 0, testSlugGDPR)
	IssueCertificateWithTemplate(2, 200, 0, testSlugGDPR)
	c3, _ := IssueCertificateWithTemplate(3, 300, 0, testSlugNIS2)

	// Revoke one
	RevokeCertificate(c3.Id)

	summary := GetCertificateSummary()
	if summary.TotalIssued != 3 {
		t.Fatalf("expected total 3, got %d", summary.TotalIssued)
	}
	if summary.RevokedCerts != 1 {
		t.Fatalf("expected 1 revoked, got %d", summary.RevokedCerts)
	}
	if summary.ByTemplate[testSlugGDPR] != 2 {
		t.Fatalf("expected 2 gdpr certs, got %d", summary.ByTemplate[testSlugGDPR])
	}
}
