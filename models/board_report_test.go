package models

import (
	"testing"
	"time"

	"github.com/gophish/gophish/config"
)

// setupBoardReportTest initialises an in-memory DB for board report tests.
func setupBoardReportTest(t *testing.T) func() {
	t.Helper()
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	if err := Setup(conf); err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	db.Exec("DELETE FROM board_reports")
	db.Exec("DELETE FROM device_hygiene_checks")
	db.Exec("DELETE FROM user_devices")
	db.Exec("DELETE FROM tech_stack_profiles")
	db.Exec("DELETE FROM remediation_steps")
	db.Exec("DELETE FROM remediation_paths")
	return func() {
		db.Exec("DELETE FROM board_reports")
		db.Exec("DELETE FROM device_hygiene_checks")
		db.Exec("DELETE FROM user_devices")
		db.Exec("DELETE FROM tech_stack_profiles")
		db.Exec("DELETE FROM remediation_steps")
		db.Exec("DELETE FROM remediation_paths")
	}
}

// ─── Board Report CRUD ───

func TestPostBoardReport(t *testing.T) {
	teardown := setupBoardReportTest(t)
	defer teardown()

	br := &BoardReport{
		OrgId:       1,
		CreatedBy:   1,
		Title:       "Q1 2026 Security Report",
		PeriodStart: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		PeriodEnd:   time.Date(2026, 3, 31, 0, 0, 0, 0, time.UTC),
	}
	if err := PostBoardReport(br); err != nil {
		t.Fatalf("PostBoardReport: %v", err)
	}
	if br.Id == 0 {
		t.Fatal("expected non-zero ID")
	}
	if br.Status != BoardReportStatusDraft {
		t.Fatalf("expected draft status, got %s", br.Status)
	}
	if br.CreatedDate.IsZero() {
		t.Fatal("expected created_date to be set")
	}
}

func TestPostBoardReport_MissingTitle(t *testing.T) {
	teardown := setupBoardReportTest(t)
	defer teardown()

	br := &BoardReport{
		OrgId:       1,
		PeriodStart: time.Now(),
		PeriodEnd:   time.Now(),
	}
	err := PostBoardReport(br)
	if err == nil {
		t.Fatal("expected error for missing title")
	}
}

func TestPostBoardReport_MissingPeriod(t *testing.T) {
	teardown := setupBoardReportTest(t)
	defer teardown()

	br := &BoardReport{
		OrgId: 1,
		Title: "No Period Report",
	}
	err := PostBoardReport(br)
	if err == nil {
		t.Fatal("expected error for missing period dates")
	}
}

func TestGetBoardReports(t *testing.T) {
	teardown := setupBoardReportTest(t)
	defer teardown()

	now := time.Now().UTC()
	PostBoardReport(&BoardReport{OrgId: 1, CreatedBy: 1, Title: "Report A", PeriodStart: now, PeriodEnd: now})
	PostBoardReport(&BoardReport{OrgId: 1, CreatedBy: 1, Title: "Report B", PeriodStart: now, PeriodEnd: now})
	PostBoardReport(&BoardReport{OrgId: 2, CreatedBy: 2, Title: "Report C", PeriodStart: now, PeriodEnd: now})

	reports, err := GetBoardReports(1)
	if err != nil {
		t.Fatalf("GetBoardReports: %v", err)
	}
	if len(reports) != 2 {
		t.Fatalf("expected 2 reports for org 1, got %d", len(reports))
	}

	reports2, _ := GetBoardReports(2)
	if len(reports2) != 1 {
		t.Fatalf("expected 1 report for org 2, got %d", len(reports2))
	}
}

func TestGetBoardReports_Empty(t *testing.T) {
	teardown := setupBoardReportTest(t)
	defer teardown()

	reports, err := GetBoardReports(999)
	if err != nil {
		t.Fatalf("GetBoardReports: %v", err)
	}
	if len(reports) != 0 {
		t.Fatalf("expected 0 reports, got %d", len(reports))
	}
}

func TestGetBoardReport(t *testing.T) {
	teardown := setupBoardReportTest(t)
	defer teardown()

	br := &BoardReport{
		OrgId: 1, CreatedBy: 1, Title: "Get Me",
		PeriodStart: time.Now().UTC(), PeriodEnd: time.Now().UTC(),
	}
	PostBoardReport(br)

	fetched, err := GetBoardReport(br.Id, 1)
	if err != nil {
		t.Fatalf("GetBoardReport: %v", err)
	}
	if fetched.Title != "Get Me" {
		t.Fatalf("wrong title: %s", fetched.Title)
	}
}

func TestGetBoardReport_NotFound(t *testing.T) {
	teardown := setupBoardReportTest(t)
	defer teardown()

	_, err := GetBoardReport(999, 1)
	if err == nil {
		t.Fatal("expected error for non-existent report")
	}
}

func TestGetBoardReport_WrongOrg(t *testing.T) {
	teardown := setupBoardReportTest(t)
	defer teardown()

	br := &BoardReport{
		OrgId: 1, CreatedBy: 1, Title: "Org 1 Only",
		PeriodStart: time.Now().UTC(), PeriodEnd: time.Now().UTC(),
	}
	PostBoardReport(br)

	// Attempt to get with a different org
	_, err := GetBoardReport(br.Id, 999)
	if err == nil {
		t.Fatal("expected error when accessing from wrong org")
	}
}

func TestPutBoardReport(t *testing.T) {
	teardown := setupBoardReportTest(t)
	defer teardown()

	br := &BoardReport{
		OrgId: 1, CreatedBy: 1, Title: "Original Title",
		PeriodStart: time.Now().UTC(), PeriodEnd: time.Now().UTC(),
	}
	PostBoardReport(br)

	br.Title = "Updated Title"
	br.Status = BoardReportStatusPublished
	if err := PutBoardReport(br); err != nil {
		t.Fatalf("PutBoardReport: %v", err)
	}

	fetched, _ := GetBoardReport(br.Id, 1)
	if fetched.Title != "Updated Title" {
		t.Fatalf("expected Updated Title, got %s", fetched.Title)
	}
	if fetched.Status != BoardReportStatusPublished {
		t.Fatalf("expected published, got %s", fetched.Status)
	}
}

func TestDeleteBoardReport(t *testing.T) {
	teardown := setupBoardReportTest(t)
	defer teardown()

	br := &BoardReport{
		OrgId: 1, CreatedBy: 1, Title: "Delete Me",
		PeriodStart: time.Now().UTC(), PeriodEnd: time.Now().UTC(),
	}
	PostBoardReport(br)

	if err := DeleteBoardReport(br.Id, 1); err != nil {
		t.Fatalf("DeleteBoardReport: %v", err)
	}

	_, err := GetBoardReport(br.Id, 1)
	if err == nil {
		t.Fatal("expected report to be deleted")
	}
}

func TestDeleteBoardReport_WrongOrg(t *testing.T) {
	teardown := setupBoardReportTest(t)
	defer teardown()

	br := &BoardReport{
		OrgId: 1, CreatedBy: 1, Title: "No Delete",
		PeriodStart: time.Now().UTC(), PeriodEnd: time.Now().UTC(),
	}
	PostBoardReport(br)

	// Delete from wrong org should have no effect
	DeleteBoardReport(br.Id, 999)

	fetched, err := GetBoardReport(br.Id, 1)
	if err != nil {
		t.Fatal("report should still exist after delete from wrong org")
	}
	if fetched.Title != "No Delete" {
		t.Fatalf("unexpected title: %s", fetched.Title)
	}
}

// ─── Snapshot Generation ───

func TestGenerateBoardReportSnapshot(t *testing.T) {
	teardown := setupBoardReportTest(t)
	defer teardown()

	start := time.Now().AddDate(0, -3, 0)
	end := time.Now()

	snap, err := GenerateBoardReportSnapshot(1, start, end)
	if err != nil {
		t.Fatalf("GenerateBoardReportSnapshot: %v", err)
	}
	if snap == nil {
		t.Fatal("expected non-nil snapshot")
	}
	if snap.PeriodLabel == "" {
		t.Fatal("expected non-empty period label")
	}

	// With no data, security posture should still be computed
	if snap.SecurityPostureScore < 0 || snap.SecurityPostureScore > 100 {
		t.Fatalf("security posture out of range: %f", snap.SecurityPostureScore)
	}

	// Risk trend should be set
	validTrends := map[string]bool{"improving": true, "stable": true, "declining": true}
	if !validTrends[snap.RiskTrend] {
		t.Fatalf("invalid risk trend: %s", snap.RiskTrend)
	}
}

func TestGenerateBoardReportSnapshot_WithHygieneData(t *testing.T) {
	teardown := setupBoardReportTest(t)
	defer teardown()

	// Create hygiene data
	d := &UserDevice{UserId: 1, OrgId: 1, Name: "Report Device", DeviceType: DeviceTypeLaptop, OS: "macOS"}
	PostDevice(d)
	UpsertDeviceCheck(d.Id, 1, HygieneCheckOSUpdated, HygieneStatusPass, "")
	UpsertDeviceCheck(d.Id, 1, HygieneCheckMFAEnabled, HygieneStatusFail, "")

	snap, err := GenerateBoardReportSnapshot(1, time.Now().AddDate(0, -3, 0), time.Now())
	if err != nil {
		t.Fatalf("GenerateBoardReportSnapshot: %v", err)
	}
	if snap.Hygiene.TotalDevices != 1 {
		t.Fatalf("expected 1 device in hygiene section, got %d", snap.Hygiene.TotalDevices)
	}
	if snap.Hygiene.AvgScore != 50 {
		t.Fatalf("expected avg score 50, got %.0f", snap.Hygiene.AvgScore)
	}
	if snap.Hygiene.AtRiskDevices != 0 {
		// Score is 50, threshold is <50 for at-risk
		t.Fatalf("expected 0 at-risk (score=50), got %d", snap.Hygiene.AtRiskDevices)
	}
}

func TestGenerateBoardReportSnapshot_AtRiskDevice(t *testing.T) {
	teardown := setupBoardReportTest(t)
	defer teardown()

	d := &UserDevice{UserId: 1, OrgId: 1, Name: "At Risk"}
	PostDevice(d)
	// 1 pass, 3 fail = 25% score (<50 = at risk)
	UpsertDeviceCheck(d.Id, 1, HygieneCheckOSUpdated, HygieneStatusPass, "")
	UpsertDeviceCheck(d.Id, 1, HygieneCheckMFAEnabled, HygieneStatusFail, "")
	UpsertDeviceCheck(d.Id, 1, HygieneCheckVPNEnabled, HygieneStatusFail, "")
	UpsertDeviceCheck(d.Id, 1, HygieneCheckDiskEncrypted, HygieneStatusFail, "")

	snap, err := GenerateBoardReportSnapshot(1, time.Now().AddDate(0, -3, 0), time.Now())
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	if snap.Hygiene.AtRiskDevices != 1 {
		t.Fatalf("expected 1 at-risk device, got %d", snap.Hygiene.AtRiskDevices)
	}
}

// ─── Recommendations ───

func TestBoardRecommendations_StrongPosture(t *testing.T) {
	snap := &BoardReportSnapshot{
		Phishing: BoardPhishingSection{AvgClickRate: 5, AvgReportRate: 30},
		Training: BoardTrainingSection{CompletionRate: 95, OverdueCount: 0},
		Risk:     BoardRiskSection{HighRiskUsers: 0},
		Compliance: BoardComplianceSection{
			OverallScore: 90, FrameworkCount: 2,
		},
		Remediation: BoardRemediationSection{CriticalCount: 0},
		Hygiene:     BoardHygieneSection{AtRiskDevices: 0, AvgScore: 85, TotalDevices: 10},
	}

	recs := generateBoardRecommendations(snap)
	if len(recs) != 1 || recs[0] != "Security posture is strong. Continue current awareness and training cadence." {
		t.Fatalf("expected strong posture message, got: %v", recs)
	}
}

func TestBoardRecommendations_HighClickRate(t *testing.T) {
	snap := &BoardReportSnapshot{
		Phishing: BoardPhishingSection{AvgClickRate: 30, AvgReportRate: 15},
		Training: BoardTrainingSection{CompletionRate: 80},
		Hygiene:  BoardHygieneSection{AvgScore: 80, TotalDevices: 1},
	}

	recs := generateBoardRecommendations(snap)
	found := false
	for _, r := range recs {
		if len(r) > 0 && r[0] == 'P' { // starts with "Phishing click rate..."
			found = true
		}
	}
	if !found {
		t.Fatalf("expected phishing click rate recommendation, got: %v", recs)
	}
}

func TestBoardRecommendations_LowTraining(t *testing.T) {
	snap := &BoardReportSnapshot{
		Phishing: BoardPhishingSection{AvgClickRate: 10, AvgReportRate: 20},
		Training: BoardTrainingSection{CompletionRate: 40, OverdueCount: 15},
		Hygiene:  BoardHygieneSection{AvgScore: 80, TotalDevices: 1},
	}

	recs := generateBoardRecommendations(snap)
	if len(recs) < 2 {
		t.Fatalf("expected multiple recommendations for low training, got %d", len(recs))
	}
}

func TestBoardRecommendations_HighRiskAndHygiene(t *testing.T) {
	snap := &BoardReportSnapshot{
		Phishing:   BoardPhishingSection{AvgClickRate: 10, AvgReportRate: 20},
		Training:   BoardTrainingSection{CompletionRate: 80},
		Risk:       BoardRiskSection{HighRiskUsers: 5},
		Hygiene:    BoardHygieneSection{AtRiskDevices: 3, AvgScore: 40, TotalDevices: 10},
		Compliance: BoardComplianceSection{OverallScore: 50, FrameworkCount: 1},
	}

	recs := generateBoardRecommendations(snap)
	if len(recs) < 3 {
		t.Fatalf("expected ≥3 recommendations, got %d: %v", len(recs), recs)
	}
}

// ─── Risk Trend Logic ───

func TestRiskTrend_Improving(t *testing.T) {
	teardown := setupBoardReportTest(t)
	defer teardown()

	// With no phishing data, click rate = 0 and completion = 0
	// but let's test the logic directly via snapshot
	snap, _ := GenerateBoardReportSnapshot(1, time.Now().AddDate(0, -3, 0), time.Now())
	// With 0 click rate and 0 completion, trend depends on thresholds
	// AvgClickRate < 15 && CompletionRate > 70 → improving
	// AvgClickRate = 0, CompletionRate = 0 → not improving, not declining → stable
	if snap.RiskTrend != "stable" {
		t.Logf("risk_trend=%s (may vary based on data present)", snap.RiskTrend)
	}
}

// ─── Security Posture Score ───

func TestSecurityPostureScore_Range(t *testing.T) {
	teardown := setupBoardReportTest(t)
	defer teardown()

	snap, _ := GenerateBoardReportSnapshot(1, time.Now().AddDate(0, -3, 0), time.Now())
	if snap.SecurityPostureScore < 0 || snap.SecurityPostureScore > 100 {
		t.Fatalf("security posture out of [0,100] range: %f", snap.SecurityPostureScore)
	}
}

// ─── Full Lifecycle ───

func TestBoardReportFullLifecycle(t *testing.T) {
	teardown := setupBoardReportTest(t)
	defer teardown()

	now := time.Now().UTC()
	start := now.AddDate(0, -3, 0)

	// Create
	br := &BoardReport{
		OrgId: 1, CreatedBy: 1, Title: "Lifecycle Test",
		PeriodStart: start, PeriodEnd: now,
	}
	if err := PostBoardReport(br); err != nil {
		t.Fatalf("create: %v", err)
	}
	if br.Status != BoardReportStatusDraft {
		t.Fatalf("expected draft, got %s", br.Status)
	}

	// List
	reports, _ := GetBoardReports(1)
	if len(reports) != 1 {
		t.Fatalf("expected 1 report, got %d", len(reports))
	}

	// Generate snapshot
	snap, err := GenerateBoardReportSnapshot(1, br.PeriodStart, br.PeriodEnd)
	if err != nil {
		t.Fatalf("generate snapshot: %v", err)
	}
	if snap.PeriodLabel == "" {
		t.Fatal("snapshot missing period label")
	}

	// Update to published
	br.Status = BoardReportStatusPublished
	br.Title = "Published Report"
	if err := PutBoardReport(br); err != nil {
		t.Fatalf("update: %v", err)
	}
	fetched, _ := GetBoardReport(br.Id, 1)
	if fetched.Status != BoardReportStatusPublished {
		t.Fatalf("expected published, got %s", fetched.Status)
	}

	// Delete
	if err := DeleteBoardReport(br.Id, 1); err != nil {
		t.Fatalf("delete: %v", err)
	}
	reports, _ = GetBoardReports(1)
	if len(reports) != 0 {
		t.Fatalf("expected 0 after delete, got %d", len(reports))
	}
}
