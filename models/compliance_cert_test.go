package models

import (
	"testing"

	"github.com/gophish/gophish/config"
)

// setupComplianceCertTest initialises an in-memory DB for compliance cert tests.
func setupComplianceCertTest(t *testing.T) func() {
	t.Helper()
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	if err := Setup(conf); err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	db.Exec("DELETE FROM user_compliance_certs")
	db.Exec("DELETE FROM compliance_certifications")
	return func() {
		db.Exec("DELETE FROM user_compliance_certs")
		db.Exec("DELETE FROM compliance_certifications")
	}
}

// ---------- CreateComplianceCertification ----------

func TestCreateComplianceCertification(t *testing.T) {
	teardown := setupComplianceCertTest(t)
	defer teardown()

	cert := &ComplianceCertification{
		OrgId:       1,
		Slug:        "gdpr-awareness",
		Name:        "GDPR Awareness",
		Description: "Covers GDPR requirements.",
		IsActive:    true,
	}
	if err := CreateComplianceCertification(cert); err != nil {
		t.Fatalf("CreateComplianceCertification failed: %v", err)
	}
	if cert.Id == 0 {
		t.Fatal("expected non-zero cert ID")
	}
	if cert.CreatedDate.IsZero() {
		t.Fatal("expected CreatedDate to be set")
	}
	if cert.RequiredSessionIDs != "[]" {
		t.Fatalf("expected default RequiredSessionIDs '[]', got %q", cert.RequiredSessionIDs)
	}
}

// ---------- GetComplianceCertifications ----------

func TestGetComplianceCertifications(t *testing.T) {
	teardown := setupComplianceCertTest(t)
	defer teardown()

	CreateComplianceCertification(&ComplianceCertification{OrgId: 1, Slug: "cert1", Name: "Cert 1", IsActive: true})
	CreateComplianceCertification(&ComplianceCertification{OrgId: 0, Slug: "global", Name: "Global Cert", IsActive: true}) // system-wide
	CreateComplianceCertification(&ComplianceCertification{OrgId: 2, Slug: "other", Name: "Other Org", IsActive: true})
	CreateComplianceCertification(&ComplianceCertification{OrgId: 1, Slug: "inactive", Name: "Inactive", IsActive: false})

	certs, err := GetComplianceCertifications(1)
	if err != nil {
		t.Fatalf("GetComplianceCertifications failed: %v", err)
	}
	// Should include org 1 + org 0 (global), excluding inactive and org 2
	if len(certs) != 2 {
		t.Fatalf("expected 2 certs, got %d", len(certs))
	}
}

// ---------- GetComplianceCertification ----------

func TestGetComplianceCertification(t *testing.T) {
	teardown := setupComplianceCertTest(t)
	defer teardown()

	cert := &ComplianceCertification{OrgId: 1, Slug: "test", Name: "Test Cert", IsActive: true}
	CreateComplianceCertification(cert)

	fetched, err := GetComplianceCertification(cert.Id)
	if err != nil {
		t.Fatalf("GetComplianceCertification failed: %v", err)
	}
	if fetched.Name != "Test Cert" {
		t.Fatalf("expected 'Test Cert', got %q", fetched.Name)
	}
}

func TestGetComplianceCertificationNotFound(t *testing.T) {
	teardown := setupComplianceCertTest(t)
	defer teardown()

	_, err := GetComplianceCertification(999)
	if err == nil {
		t.Fatal("expected error for non-existent cert")
	}
}

// ---------- UpdateComplianceCertification ----------

func TestUpdateComplianceCertification(t *testing.T) {
	teardown := setupComplianceCertTest(t)
	defer teardown()

	cert := &ComplianceCertification{OrgId: 1, Slug: "upd", Name: "Original", IsActive: true}
	CreateComplianceCertification(cert)

	cert.Name = "Updated"
	cert.Description = "New description"
	if err := UpdateComplianceCertification(cert); err != nil {
		t.Fatalf("UpdateComplianceCertification failed: %v", err)
	}

	fetched, _ := GetComplianceCertification(cert.Id)
	if fetched.Name != "Updated" {
		t.Fatalf("expected 'Updated', got %q", fetched.Name)
	}
	if fetched.Description != "New description" {
		t.Fatalf("expected 'New description', got %q", fetched.Description)
	}
}

// ---------- IssueComplianceCert ----------

func TestIssueComplianceCert(t *testing.T) {
	teardown := setupComplianceCertTest(t)
	defer teardown()

	cert := &ComplianceCertification{OrgId: 1, Slug: "issue-test", Name: "Issue Test", IsActive: true}
	CreateComplianceCertification(cert)

	uc, err := IssueComplianceCert(1, cert.Id)
	if err != nil {
		t.Fatalf("IssueComplianceCert failed: %v", err)
	}
	if uc.VerificationCode == "" {
		t.Fatal("expected verification code to be set")
	}
	if uc.IssuedDate.IsZero() {
		t.Fatal("expected IssuedDate to be set")
	}
	if uc.ExpiresDate.IsZero() {
		t.Fatal("expected ExpiresDate to be set")
	}
}

func TestIssueComplianceCertIdempotent(t *testing.T) {
	teardown := setupComplianceCertTest(t)
	defer teardown()

	cert := &ComplianceCertification{OrgId: 1, Slug: "idempotent", Name: "Idempotent", IsActive: true}
	CreateComplianceCertification(cert)

	uc1, _ := IssueComplianceCert(1, cert.Id)
	uc2, _ := IssueComplianceCert(1, cert.Id)

	if uc1.Id != uc2.Id {
		t.Fatal("expected same cert to be returned on duplicate issue")
	}
}

// ---------- GetUserComplianceCerts ----------

func TestGetUserComplianceCerts(t *testing.T) {
	teardown := setupComplianceCertTest(t)
	defer teardown()

	c1 := &ComplianceCertification{OrgId: 1, Slug: "c1", Name: "Cert A", IsActive: true}
	CreateComplianceCertification(c1)
	c2 := &ComplianceCertification{OrgId: 1, Slug: "c2", Name: "Cert B", IsActive: true}
	CreateComplianceCertification(c2)

	IssueComplianceCert(1, c1.Id)
	IssueComplianceCert(1, c2.Id)

	certs, err := GetUserComplianceCerts(1)
	if err != nil {
		t.Fatalf("GetUserComplianceCerts failed: %v", err)
	}
	if len(certs) != 2 {
		t.Fatalf("expected 2 certs, got %d", len(certs))
	}
	// Verify enrichment
	if certs[0].CertificationName == "" {
		t.Fatal("expected CertificationName to be populated")
	}
}

func TestGetUserComplianceCertsEmpty(t *testing.T) {
	teardown := setupComplianceCertTest(t)
	defer teardown()

	certs, err := GetUserComplianceCerts(999)
	if err != nil {
		t.Fatalf("GetUserComplianceCerts failed: %v", err)
	}
	if len(certs) != 0 {
		t.Fatalf("expected 0 certs, got %d", len(certs))
	}
}

// ---------- VerifyComplianceCert ----------

func TestVerifyComplianceCert(t *testing.T) {
	teardown := setupComplianceCertTest(t)
	defer teardown()

	cert := &ComplianceCertification{OrgId: 1, Slug: "verify", Name: "Verify Cert", IsActive: true}
	CreateComplianceCertification(cert)
	uc, _ := IssueComplianceCert(1, cert.Id)

	verified, err := VerifyComplianceCert(uc.VerificationCode)
	if err != nil {
		t.Fatalf("VerifyComplianceCert failed: %v", err)
	}
	if verified.UserId != 1 {
		t.Fatalf("expected UserId 1, got %d", verified.UserId)
	}
	if verified.CertificationName != "Verify Cert" {
		t.Fatalf("expected name 'Verify Cert', got %q", verified.CertificationName)
	}
}

func TestVerifyComplianceCertInvalidCode(t *testing.T) {
	teardown := setupComplianceCertTest(t)
	defer teardown()

	_, err := VerifyComplianceCert("invalid-code-12345")
	if err == nil {
		t.Fatal("expected error for invalid verification code")
	}
}

// ---------- GetComplianceCertCount ----------

func TestGetComplianceCertCount(t *testing.T) {
	teardown := setupComplianceCertTest(t)
	defer teardown()

	c1 := &ComplianceCertification{OrgId: 1, Slug: "cnt1", Name: "Count 1", IsActive: true}
	CreateComplianceCertification(c1)
	c2 := &ComplianceCertification{OrgId: 1, Slug: "cnt2", Name: "Count 2", IsActive: true}
	CreateComplianceCertification(c2)

	IssueComplianceCert(1, c1.Id)
	IssueComplianceCert(1, c2.Id)

	count := GetComplianceCertCount(1)
	if count != 2 {
		t.Fatalf("expected 2, got %d", count)
	}
}

func TestGetComplianceCertCountZero(t *testing.T) {
	teardown := setupComplianceCertTest(t)
	defer teardown()

	count := GetComplianceCertCount(999)
	if count != 0 {
		t.Fatalf("expected 0, got %d", count)
	}
}
