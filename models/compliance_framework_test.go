package models

import (
	"testing"
	"time"

	"github.com/gophish/gophish/config"
)

// Test constants to avoid duplicate literal warnings (SonarLint S1192).
const (
	testControlRefNIS2_1 = "NIS2-1"
	testControlRefNIS2_2 = "NIS2-2"
	errUnexpectedValue   = "unexpected value: %q"
)

// setupComplianceFrameworkTest initialises an in-memory DB for compliance framework tests.
func setupComplianceFrameworkTest(t *testing.T) func() {
	t.Helper()
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	if err := Setup(conf); err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	db.Exec("DELETE FROM compliance_assessments")
	db.Exec("DELETE FROM org_compliance_mappings")
	db.Exec("DELETE FROM compliance_controls")
	db.Exec("DELETE FROM compliance_frameworks")
	return func() {
		db.Exec("DELETE FROM compliance_assessments")
		db.Exec("DELETE FROM org_compliance_mappings")
		db.Exec("DELETE FROM compliance_controls")
		db.Exec("DELETE FROM compliance_frameworks")
	}
}

func createTestFramework(t *testing.T, slug, name string) ComplianceFramework {
	t.Helper()
	f := ComplianceFramework{
		Slug:        slug,
		Name:        name,
		Version:     "1.0",
		Description: "Test framework",
		Region:      "EU",
		IsActive:    true,
		CreatedDate: time.Now().UTC(),
	}
	if err := db.Create(&f).Error; err != nil {
		t.Fatalf("failed to create framework: %v", err)
	}
	return f
}

func createTestControl(t *testing.T, frameworkId int64, ref, evidenceType, critJSON string, order int) ComplianceControl {
	t.Helper()
	c := ComplianceControl{
		FrameworkId:     frameworkId,
		ControlRef:      ref,
		Title:           "Test Control " + ref,
		Description:     "Description for " + ref,
		Category:        "awareness",
		EvidenceType:    evidenceType,
		EvidenceCritera: critJSON,
		SortOrder:       order,
	}
	if err := db.Create(&c).Error; err != nil {
		t.Fatalf("failed to create control: %v", err)
	}
	return c
}

// ---------- GetComplianceFrameworks ----------

func TestGetComplianceFrameworks(t *testing.T) {
	teardown := setupComplianceFrameworkTest(t)
	defer teardown()

	createTestFramework(t, "nis2", "NIS2")
	createTestFramework(t, "dora", "DORA")
	// Create an inactive one
	db.Create(&ComplianceFramework{Slug: "inactive", Name: "Inactive", IsActive: false, CreatedDate: time.Now()})

	frameworks, err := GetComplianceFrameworks()
	if err != nil {
		t.Fatalf("GetComplianceFrameworks failed: %v", err)
	}
	if len(frameworks) != 2 {
		t.Fatalf("expected 2 active frameworks, got %d", len(frameworks))
	}
}

// ---------- GetComplianceFramework ----------

func TestGetComplianceFramework(t *testing.T) {
	teardown := setupComplianceFrameworkTest(t)
	defer teardown()

	f := createTestFramework(t, "nis2", "NIS2")

	fetched, err := GetComplianceFramework(f.Id)
	if err != nil {
		t.Fatalf("GetComplianceFramework failed: %v", err)
	}
	if fetched.Slug != "nis2" {
		t.Fatalf("expected slug 'nis2', got %q", fetched.Slug)
	}
}

func TestGetComplianceFrameworkNotFound(t *testing.T) {
	teardown := setupComplianceFrameworkTest(t)
	defer teardown()

	_, err := GetComplianceFramework(999)
	if err == nil {
		t.Fatal("expected error for non-existent framework")
	}
}

// ---------- GetFrameworkControls ----------

func TestGetFrameworkControls(t *testing.T) {
	teardown := setupComplianceFrameworkTest(t)
	defer teardown()

	f := createTestFramework(t, "nis2", "NIS2")
	createTestControl(t, f.Id, testControlRefNIS2_1, EvidenceTypeManual, "{}", 1)
	createTestControl(t, f.Id, testControlRefNIS2_2, EvidenceTypeManual, "{}", 2)

	controls, err := GetFrameworkControls(f.Id)
	if err != nil {
		t.Fatalf("GetFrameworkControls failed: %v", err)
	}
	if len(controls) != 2 {
		t.Fatalf("expected 2 controls, got %d", len(controls))
	}
	if controls[0].SortOrder > controls[1].SortOrder {
		t.Fatal("expected controls ordered by sort_order ascending")
	}
}

// ---------- EnableOrgFramework / DisableOrgFramework ----------

func TestEnableOrgFramework(t *testing.T) {
	teardown := setupComplianceFrameworkTest(t)
	defer teardown()

	f := createTestFramework(t, "nis2", "NIS2")
	if err := EnableOrgFramework(1, f.Id); err != nil {
		t.Fatalf("EnableOrgFramework failed: %v", err)
	}

	frameworks, err := GetOrgFrameworks(1)
	if err != nil {
		t.Fatalf("GetOrgFrameworks failed: %v", err)
	}
	if len(frameworks) != 1 {
		t.Fatalf("expected 1 org framework, got %d", len(frameworks))
	}
}

func TestEnableOrgFrameworkIdempotent(t *testing.T) {
	teardown := setupComplianceFrameworkTest(t)
	defer teardown()

	f := createTestFramework(t, "nis2", "NIS2")
	EnableOrgFramework(1, f.Id)
	// Enable again — should not duplicate
	if err := EnableOrgFramework(1, f.Id); err != nil {
		t.Fatalf("EnableOrgFramework (2nd) failed: %v", err)
	}

	frameworks, _ := GetOrgFrameworks(1)
	if len(frameworks) != 1 {
		t.Fatalf("expected 1 org framework after re-enable, got %d", len(frameworks))
	}
}

func TestDisableOrgFramework(t *testing.T) {
	teardown := setupComplianceFrameworkTest(t)
	defer teardown()

	f := createTestFramework(t, "nis2", "NIS2")
	EnableOrgFramework(1, f.Id)
	if err := DisableOrgFramework(1, f.Id); err != nil {
		t.Fatalf("DisableOrgFramework failed: %v", err)
	}

	frameworks, _ := GetOrgFrameworks(1)
	if len(frameworks) != 0 {
		t.Fatalf("expected 0 org frameworks after disable, got %d", len(frameworks))
	}
}

func TestReEnableOrgFramework(t *testing.T) {
	teardown := setupComplianceFrameworkTest(t)
	defer teardown()

	f := createTestFramework(t, "nis2", "NIS2")
	EnableOrgFramework(1, f.Id)
	DisableOrgFramework(1, f.Id)
	EnableOrgFramework(1, f.Id)

	frameworks, _ := GetOrgFrameworks(1)
	if len(frameworks) != 1 {
		t.Fatalf("expected 1 framework after re-enable, got %d", len(frameworks))
	}
}

// ---------- SaveComplianceAssessment / GetLatestAssessment ----------

func TestSaveAndGetLatestAssessment(t *testing.T) {
	teardown := setupComplianceFrameworkTest(t)
	defer teardown()

	f := createTestFramework(t, "nis2", "NIS2")
	c := createTestControl(t, f.Id, testControlRefNIS2_1, EvidenceTypeManual, "{}", 1)

	a := &ComplianceAssessment{
		OrgId:       1,
		FrameworkId: f.Id,
		ControlId:   c.Id,
		Status:      ComplianceStatusCompliant,
		Score:       100,
		Evidence:    "Manual check passed",
		AssessedBy:  1,
	}
	if err := SaveComplianceAssessment(a); err != nil {
		t.Fatalf("SaveComplianceAssessment failed: %v", err)
	}
	if a.AssessedDate.IsZero() {
		t.Fatal("expected AssessedDate to be set")
	}

	latest, err := GetLatestAssessment(1, c.Id)
	if err != nil {
		t.Fatalf("GetLatestAssessment failed: %v", err)
	}
	if latest.Status != ComplianceStatusCompliant {
		t.Fatalf("expected status 'compliant', got %q", latest.Status)
	}
}

// ---------- GetFrameworkAssessments ----------

func TestGetFrameworkAssessments(t *testing.T) {
	teardown := setupComplianceFrameworkTest(t)
	defer teardown()

	f := createTestFramework(t, "nis2", "NIS2")
	c1 := createTestControl(t, f.Id, testControlRefNIS2_1, EvidenceTypeManual, "{}", 1)
	c2 := createTestControl(t, f.Id, testControlRefNIS2_2, EvidenceTypeManual, "{}", 2)

	SaveComplianceAssessment(&ComplianceAssessment{OrgId: 1, FrameworkId: f.Id, ControlId: c1.Id, Status: ComplianceStatusCompliant, Score: 100})
	SaveComplianceAssessment(&ComplianceAssessment{OrgId: 1, FrameworkId: f.Id, ControlId: c2.Id, Status: ComplianceStatusPartial, Score: 65})

	assessments, err := GetFrameworkAssessments(1, f.Id)
	if err != nil {
		t.Fatalf("GetFrameworkAssessments failed: %v", err)
	}
	if len(assessments) != 2 {
		t.Fatalf("expected 2 assessments, got %d", len(assessments))
	}
}

// ---------- GetFrameworkSummary ----------

func TestGetFrameworkSummary(t *testing.T) {
	teardown := setupComplianceFrameworkTest(t)
	defer teardown()

	f := createTestFramework(t, "nis2", "NIS2")
	c1 := createTestControl(t, f.Id, testControlRefNIS2_1, EvidenceTypeManual, "{}", 1)
	c2 := createTestControl(t, f.Id, testControlRefNIS2_2, EvidenceTypeManual, "{}", 2)
	createTestControl(t, f.Id, "NIS2-3", EvidenceTypeManual, "{}", 3) // not assessed

	SaveComplianceAssessment(&ComplianceAssessment{OrgId: 1, FrameworkId: f.Id, ControlId: c1.Id, Status: ComplianceStatusCompliant, Score: 100})
	SaveComplianceAssessment(&ComplianceAssessment{OrgId: 1, FrameworkId: f.Id, ControlId: c2.Id, Status: ComplianceStatusNonCompliant, Score: 0})

	summary, err := GetFrameworkSummary(1, f.Id, true)
	if err != nil {
		t.Fatalf("GetFrameworkSummary failed: %v", err)
	}
	if summary.TotalControls != 3 {
		t.Fatalf("expected 3 total controls, got %d", summary.TotalControls)
	}
	if summary.Compliant != 1 {
		t.Fatalf("expected 1 compliant, got %d", summary.Compliant)
	}
	if summary.NonCompliant != 1 {
		t.Fatalf("expected 1 non-compliant, got %d", summary.NonCompliant)
	}
	if summary.NotAssessed != 1 {
		t.Fatalf("expected 1 not assessed, got %d", summary.NotAssessed)
	}
	if len(summary.Controls) != 3 {
		t.Fatalf("expected 3 controls in response, got %d", len(summary.Controls))
	}
}

func TestGetFrameworkSummaryWithoutControls(t *testing.T) {
	teardown := setupComplianceFrameworkTest(t)
	defer teardown()

	f := createTestFramework(t, "nis2", "NIS2")
	createTestControl(t, f.Id, testControlRefNIS2_1, EvidenceTypeManual, "{}", 1)

	summary, err := GetFrameworkSummary(1, f.Id, false)
	if err != nil {
		t.Fatalf("GetFrameworkSummary failed: %v", err)
	}
	if len(summary.Controls) != 0 {
		t.Fatalf("expected 0 controls when includeControls=false, got %d", len(summary.Controls))
	}
}

// ---------- ComplianceDashboard ----------

func TestGetComplianceDashboardEmpty(t *testing.T) {
	teardown := setupComplianceFrameworkTest(t)
	defer teardown()

	dashboard, err := GetComplianceDashboard(1)
	if err != nil {
		t.Fatalf("GetComplianceDashboard failed: %v", err)
	}
	if len(dashboard.Frameworks) != 0 {
		t.Fatalf("expected 0 frameworks, got %d", len(dashboard.Frameworks))
	}
}

func TestGetComplianceDashboardWithFrameworks(t *testing.T) {
	teardown := setupComplianceFrameworkTest(t)
	defer teardown()

	f1 := createTestFramework(t, "nis2", "NIS2")
	f2 := createTestFramework(t, "dora", "DORA")
	EnableOrgFramework(1, f1.Id)
	EnableOrgFramework(1, f2.Id)

	c1 := createTestControl(t, f1.Id, testControlRefNIS2_1, EvidenceTypeManual, "{}", 1)
	createTestControl(t, f2.Id, "DORA-1", EvidenceTypeManual, "{}", 1)

	SaveComplianceAssessment(&ComplianceAssessment{OrgId: 1, FrameworkId: f1.Id, ControlId: c1.Id, Status: ComplianceStatusCompliant, Score: 100})

	dashboard, err := GetComplianceDashboard(1)
	if err != nil {
		t.Fatalf("GetComplianceDashboard failed: %v", err)
	}
	if len(dashboard.Frameworks) != 2 {
		t.Fatalf("expected 2 frameworks in dashboard, got %d", len(dashboard.Frameworks))
	}
	if dashboard.TotalControls != 2 {
		t.Fatalf("expected 2 total controls, got %d", dashboard.TotalControls)
	}
}

// ---------- Compliance status constants ----------

func TestComplianceStatusConstants(t *testing.T) {
	if ComplianceStatusCompliant != "compliant" {
		t.Fatalf(errUnexpectedValue, ComplianceStatusCompliant)
	}
	if ComplianceStatusPartial != "partial" {
		t.Fatalf(errUnexpectedValue, ComplianceStatusPartial)
	}
	if ComplianceStatusNonCompliant != "non_compliant" {
		t.Fatalf(errUnexpectedValue, ComplianceStatusNonCompliant)
	}
	if ComplianceStatusNotAssessed != "not_assessed" {
		t.Fatalf(errUnexpectedValue, ComplianceStatusNotAssessed)
	}
}

// ---------- Evidence type constants ----------

func TestEvidenceTypeConstants(t *testing.T) {
	types := []string{
		EvidenceTypeSimulationRate,
		EvidenceTypeTrainingRate,
		EvidenceTypeQuizPassRate,
		EvidenceTypeBRSScore,
		EvidenceTypeReportRate,
		EvidenceTypeCertification,
		EvidenceTypeManual,
	}
	for _, et := range types {
		if et == "" {
			t.Fatal("evidence type constant must not be empty")
		}
	}
}

// ---------- evaluateCriteria ----------

func TestEvaluateCriteria(t *testing.T) {
	tests := []struct {
		value    float64
		criteria EvidenceCriteria
		expected bool
	}{
		{80, EvidenceCriteria{Threshold: 70, Operator: "gte"}, true},
		{60, EvidenceCriteria{Threshold: 70, Operator: "gte"}, false},
		{5, EvidenceCriteria{Threshold: 10, Operator: "lte"}, true},
		{15, EvidenceCriteria{Threshold: 10, Operator: "lte"}, false},
		{50, EvidenceCriteria{Threshold: 50, Operator: "eq"}, true},
		{51, EvidenceCriteria{Threshold: 50, Operator: "eq"}, false},
		{80, EvidenceCriteria{Threshold: 70, Operator: "gt"}, true},
		{70, EvidenceCriteria{Threshold: 70, Operator: "gt"}, false},
		{5, EvidenceCriteria{Threshold: 10, Operator: "lt"}, true},
		{10, EvidenceCriteria{Threshold: 10, Operator: "lt"}, false},
		// Default (no operator) should behave like >=
		{80, EvidenceCriteria{Threshold: 70, Operator: ""}, true},
		{60, EvidenceCriteria{Threshold: 70, Operator: ""}, false},
	}

	for _, tc := range tests {
		got := evaluateCriteria(tc.value, tc.criteria)
		if got != tc.expected {
			t.Errorf("evaluateCriteria(%.1f, %+v) = %v, want %v", tc.value, tc.criteria, got, tc.expected)
		}
	}
}

// ---------- deriveComplianceStatus ----------

func TestDeriveComplianceStatusCompliant(t *testing.T) {
	status, score := deriveComplianceStatus(80, EvidenceCriteria{Threshold: 70, Operator: "gte"})
	if status != ComplianceStatusCompliant {
		t.Fatalf("expected 'compliant', got %q", status)
	}
	if score != 100 {
		t.Fatalf("expected score 100, got %f", score)
	}
}

func TestDeriveComplianceStatusPartial(t *testing.T) {
	status, score := deriveComplianceStatus(50, EvidenceCriteria{Threshold: 80, Operator: "gte"})
	if status != ComplianceStatusPartial {
		t.Fatalf("expected 'partial', got %q", status)
	}
	if score <= 0 || score >= 100 {
		t.Fatalf("expected partial score between 0 and 100, got %f", score)
	}
}

func TestDeriveComplianceStatusNonCompliant(t *testing.T) {
	status, _ := deriveComplianceStatus(0, EvidenceCriteria{Threshold: 80, Operator: "gte"})
	if status != ComplianceStatusNonCompliant {
		t.Fatalf("expected 'non_compliant', got %q", status)
	}
}
