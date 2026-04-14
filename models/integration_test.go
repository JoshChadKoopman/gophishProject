package models

import (
	"testing"
	"time"

	"github.com/gophish/gophish/config"
)

// setupIntegrationTest initialises an in-memory DB with all tables for
// cross-module integration tests covering board reports, cyber hygiene,
// and remediation features together.
func setupIntegrationTest(t *testing.T) func() {
	t.Helper()
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	if err := Setup(conf); err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	cleanIntegration()
	return func() { cleanIntegration() }
}

func cleanIntegration() {
	db.Exec("DELETE FROM board_reports")
	db.Exec("DELETE FROM device_hygiene_checks")
	db.Exec("DELETE FROM user_devices")
	db.Exec("DELETE FROM tech_stack_profiles")
	db.Exec("DELETE FROM remediation_steps")
	db.Exec("DELETE FROM remediation_paths")
}

// ===================================================================
// SMOKE TESTS — verify basic operations work end-to-end
// ===================================================================

func TestSmoke_DeviceLifecycle(t *testing.T) {
	teardown := setupIntegrationTest(t)
	defer teardown()

	// Register a device
	d := &UserDevice{UserId: 1, OrgId: 1, Name: "Smoke Laptop", DeviceType: DeviceTypeLaptop, OS: "macOS"}
	if err := PostDevice(d); err != nil {
		t.Fatalf("PostDevice: %v", err)
	}

	// Run hygiene checks
	checks := []struct {
		ct     string
		status string
	}{
		{HygieneCheckOSUpdated, HygieneStatusPass},
		{HygieneCheckAntivirusActive, HygieneStatusPass},
		{HygieneCheckDiskEncrypted, HygieneStatusPass},
		{HygieneCheckScreenLock, HygieneStatusFail},
		{HygieneCheckPasswordManager, HygieneStatusPass},
		{HygieneCheckVPNEnabled, HygieneStatusUnknown},
		{HygieneCheckMFAEnabled, HygieneStatusPass},
	}
	for _, ch := range checks {
		if err := UpsertDeviceCheck(d.Id, 1, ch.ct, ch.status, ""); err != nil {
			t.Fatalf("UpsertDeviceCheck(%s): %v", ch.ct, err)
		}
	}

	// Verify score: 5 pass / 7 total ≈ 71%
	fetched, _ := GetDevice(d.Id, 1)
	if fetched.HygieneScore != 71 {
		t.Fatalf("expected score 71, got %d", fetched.HygieneScore)
	}

	// Update device
	fetched.Name = "Updated Laptop"
	PutDevice(&fetched)
	refetched, _ := GetDevice(d.Id, 1)
	if refetched.Name != "Updated Laptop" {
		t.Fatalf("expected Updated Laptop, got %s", refetched.Name)
	}

	// Delete
	DeleteDevice(d.Id, 1)
	_, err := GetDevice(d.Id, 1)
	if err != ErrDeviceNotFound {
		t.Fatal("expected device deleted")
	}
}

func TestSmoke_BoardReportLifecycle(t *testing.T) {
	teardown := setupIntegrationTest(t)
	defer teardown()

	now := time.Now().UTC()
	br := &BoardReport{
		OrgId: 1, CreatedBy: 1, Title: "Smoke Board Report",
		PeriodStart: now.AddDate(0, -3, 0), PeriodEnd: now,
	}
	if err := PostBoardReport(br); err != nil {
		t.Fatalf("create: %v", err)
	}

	// List
	list, _ := GetBoardReports(1)
	if len(list) != 1 {
		t.Fatalf("expected 1, got %d", len(list))
	}

	// Generate snapshot
	snap, err := GenerateBoardReportSnapshot(1, br.PeriodStart, br.PeriodEnd)
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}
	if snap == nil {
		t.Fatal("nil snapshot")
	}

	// Publish
	br.Status = BoardReportStatusPublished
	PutBoardReport(br)
	fetched, _ := GetBoardReport(br.Id, 1)
	if fetched.Status != BoardReportStatusPublished {
		t.Fatalf("expected published, got %s", fetched.Status)
	}

	// Delete
	DeleteBoardReport(br.Id, 1)
	list, _ = GetBoardReports(1)
	if len(list) != 0 {
		t.Fatal("expected 0 after delete")
	}
}

func TestSmoke_TechStackProfile(t *testing.T) {
	teardown := setupIntegrationTest(t)
	defer teardown()

	p := &TechStackProfile{
		UserId: 1, OrgId: 1,
		PrimaryOS: "Windows", Browser: "Chrome", EmailClient: "Outlook",
		RemoteAccess: "Cisco AnyConnect", MobileDevice: "iPhone 15",
	}
	if err := UpsertTechStackProfile(p); err != nil {
		t.Fatalf("create profile: %v", err)
	}

	checks := GetPersonalizedChecks(1, 1)
	if len(checks) != 7 {
		t.Fatalf("expected 7 personalized checks, got %d", len(checks))
	}

	// Verify OS-specific reason
	for _, c := range checks {
		if c.CheckType == HygieneCheckDiskEncrypted {
			if c.Reason == "" {
				t.Fatal("expected BitLocker reason for Windows")
			}
		}
		if c.CheckType == HygieneCheckVPNEnabled {
			if c.Reason == "" {
				t.Fatal("expected VPN reason with Cisco AnyConnect")
			}
		}
	}
}

// ===================================================================
// INTEGRATION TESTS — cross-module data flow
// ===================================================================

func TestIntegration_HygieneInBoardReport(t *testing.T) {
	teardown := setupIntegrationTest(t)
	defer teardown()

	// Create two devices: one fully compliant, one at-risk
	d1 := &UserDevice{UserId: 1, OrgId: 1, Name: "Compliant", DeviceType: DeviceTypeLaptop, OS: "macOS"}
	d2 := &UserDevice{UserId: 1, OrgId: 1, Name: "At Risk", DeviceType: DeviceTypeMobile, OS: "Android"}
	PostDevice(d1)
	PostDevice(d2)

	// d1: all pass
	allCheckTypes := []string{
		HygieneCheckOSUpdated, HygieneCheckAntivirusActive, HygieneCheckDiskEncrypted,
		HygieneCheckScreenLock, HygieneCheckPasswordManager, HygieneCheckVPNEnabled, HygieneCheckMFAEnabled,
	}
	for _, ct := range allCheckTypes {
		UpsertDeviceCheck(d1.Id, 1, ct, HygieneStatusPass, "")
	}

	// d2: all fail
	for _, ct := range allCheckTypes {
		UpsertDeviceCheck(d2.Id, 1, ct, HygieneStatusFail, "")
	}

	// Generate board report snapshot — should reflect hygiene data
	snap, err := GenerateBoardReportSnapshot(1, time.Now().AddDate(0, -3, 0), time.Now())
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	if snap.Hygiene.TotalDevices != 2 {
		t.Fatalf("expected 2 devices, got %d", snap.Hygiene.TotalDevices)
	}
	if snap.Hygiene.FullyCompliant != 1 {
		t.Fatalf("expected 1 fully compliant, got %d", snap.Hygiene.FullyCompliant)
	}
	if snap.Hygiene.AtRiskDevices != 1 {
		t.Fatalf("expected 1 at-risk, got %d", snap.Hygiene.AtRiskDevices)
	}
	// Avg: (100 + 0) / 2 = 50
	if snap.Hygiene.AvgScore != 50 {
		t.Fatalf("expected avg score 50, got %.0f", snap.Hygiene.AvgScore)
	}

	// At-risk hygiene should trigger a recommendation
	hasHygieneRec := false
	for _, r := range snap.Recommendations {
		if len(r) > 0 {
			hasHygieneRec = true
		}
	}
	if !hasHygieneRec {
		t.Fatal("expected at least one recommendation given at-risk device")
	}
}

func TestIntegration_EnrichedSummaryBreakdown(t *testing.T) {
	teardown := setupIntegrationTest(t)
	defer teardown()

	// Create devices across different OSes and types
	PostDevice(&UserDevice{UserId: 1, OrgId: 1, Name: "Mac Laptop", DeviceType: DeviceTypeLaptop, OS: "macOS"})
	PostDevice(&UserDevice{UserId: 1, OrgId: 1, Name: "Win Desktop", DeviceType: DeviceTypeDesktop, OS: "Windows"})
	PostDevice(&UserDevice{UserId: 1, OrgId: 1, Name: "Android Phone", DeviceType: DeviceTypeMobile, OS: "Android"})
	PostDevice(&UserDevice{UserId: 1, OrgId: 1, Name: "iPad", DeviceType: DeviceTypeTablet, OS: "iPadOS"})

	enriched, err := GetOrgHygieneEnrichedSummary(1)
	if err != nil {
		t.Fatalf("enriched summary: %v", err)
	}
	if enriched.TotalDevices != 4 {
		t.Fatalf("expected 4 devices, got %d", enriched.TotalDevices)
	}
	if enriched.OSBreakdown["macOS"] != 1 {
		t.Fatal("expected 1 macOS device")
	}
	if enriched.OSBreakdown["Windows"] != 1 {
		t.Fatal("expected 1 Windows device")
	}
	if enriched.DeviceTypeBreakdown[DeviceTypeLaptop] != 1 {
		t.Fatal("expected 1 laptop")
	}
	if enriched.DeviceTypeBreakdown[DeviceTypeMobile] != 1 {
		t.Fatal("expected 1 mobile")
	}
	if enriched.DeviceTypeBreakdown[DeviceTypeTablet] != 1 {
		t.Fatal("expected 1 tablet")
	}
}

func TestIntegration_TechStackPersonalization(t *testing.T) {
	teardown := setupIntegrationTest(t)
	defer teardown()

	// macOS user
	UpsertTechStackProfile(&TechStackProfile{
		UserId: 1, OrgId: 1, PrimaryOS: "macOS",
		EmailClient: "Apple Mail", RemoteAccess: "WireGuard",
	})

	checks := GetPersonalizedChecks(1, 1)
	for _, c := range checks {
		switch c.CheckType {
		case HygieneCheckDiskEncrypted:
			if c.Reason == "" {
				t.Fatal("expected FileVault reason for macOS")
			}
		case HygieneCheckAntivirusActive:
			if c.Reason == "" {
				t.Fatal("expected XProtect reason for macOS")
			}
		case HygieneCheckVPNEnabled:
			if c.Reason == "" {
				t.Fatal("expected WireGuard-specific VPN reason")
			}
		case HygieneCheckMFAEnabled:
			if c.Reason == "" {
				t.Fatal("expected Apple Mail MFA reason")
			}
		}
	}

	// Windows user
	UpsertTechStackProfile(&TechStackProfile{
		UserId: 2, OrgId: 1, PrimaryOS: "Windows",
		EmailClient: "Outlook",
	})
	checks2 := GetPersonalizedChecks(2, 1)
	for _, c := range checks2 {
		if c.CheckType == HygieneCheckDiskEncrypted {
			if c.Reason == "" {
				t.Fatal("expected BitLocker reason for Windows")
			}
		}
		if c.CheckType == HygieneCheckAntivirusActive {
			if c.Reason == "" {
				t.Fatal("expected Windows Defender reason")
			}
		}
	}
}

func TestIntegration_MultipleUsersOrgSummary(t *testing.T) {
	teardown := setupIntegrationTest(t)
	defer teardown()

	// User 1 in org 1: 2 devices
	d1 := &UserDevice{UserId: 1, OrgId: 1, Name: "User1 Laptop", DeviceType: DeviceTypeLaptop}
	d2 := &UserDevice{UserId: 1, OrgId: 1, Name: "User1 Phone", DeviceType: DeviceTypeMobile}
	PostDevice(d1)
	PostDevice(d2)

	// User 2 in org 1: 1 device
	d3 := &UserDevice{UserId: 2, OrgId: 1, Name: "User2 Laptop", DeviceType: DeviceTypeLaptop}
	PostDevice(d3)

	// User 3 in org 2: 1 device (should NOT appear in org 1 summary)
	d4 := &UserDevice{UserId: 3, OrgId: 2, Name: "Other Org", DeviceType: DeviceTypeLaptop}
	PostDevice(d4)

	summary, _ := GetOrgHygieneSummary(1)
	if summary.TotalDevices != 3 {
		t.Fatalf("expected 3 devices for org 1, got %d", summary.TotalDevices)
	}

	summary2, _ := GetOrgHygieneSummary(2)
	if summary2.TotalDevices != 1 {
		t.Fatalf("expected 1 device for org 2, got %d", summary2.TotalDevices)
	}
}

func TestIntegration_CheckBreakdownAccuracy(t *testing.T) {
	teardown := setupIntegrationTest(t)
	defer teardown()

	d1 := &UserDevice{UserId: 1, OrgId: 1, Name: "Device A"}
	d2 := &UserDevice{UserId: 2, OrgId: 1, Name: "Device B"}
	PostDevice(d1)
	PostDevice(d2)

	// d1: OS=pass, MFA=fail
	UpsertDeviceCheck(d1.Id, 1, HygieneCheckOSUpdated, HygieneStatusPass, "")
	UpsertDeviceCheck(d1.Id, 1, HygieneCheckMFAEnabled, HygieneStatusFail, "")

	// d2: OS=pass, MFA=pass
	UpsertDeviceCheck(d2.Id, 1, HygieneCheckOSUpdated, HygieneStatusPass, "")
	UpsertDeviceCheck(d2.Id, 1, HygieneCheckMFAEnabled, HygieneStatusPass, "")

	summary, _ := GetOrgHygieneSummary(1)

	osCheck := summary.CheckBreakdown[HygieneCheckOSUpdated]
	if osCheck.Pass != 2 {
		t.Fatalf("expected 2 OS pass, got %d", osCheck.Pass)
	}

	mfaCheck := summary.CheckBreakdown[HygieneCheckMFAEnabled]
	if mfaCheck.Pass != 1 || mfaCheck.Fail != 1 {
		t.Fatalf("expected MFA 1 pass 1 fail, got pass=%d fail=%d", mfaCheck.Pass, mfaCheck.Fail)
	}

	if summary.PassCount != 3 {
		t.Fatalf("expected 3 total passes, got %d", summary.PassCount)
	}
	if summary.FailCount != 1 {
		t.Fatalf("expected 1 total fail, got %d", summary.FailCount)
	}
}

// ===================================================================
// FUNCTIONAL TESTS — end-to-end feature behavior
// ===================================================================

func TestFunctional_BoardReportWithFullData(t *testing.T) {
	teardown := setupIntegrationTest(t)
	defer teardown()

	// Populate hygiene data
	d := &UserDevice{UserId: 1, OrgId: 1, Name: "Full Data Device", DeviceType: DeviceTypeLaptop, OS: "macOS"}
	PostDevice(d)
	UpsertDeviceCheck(d.Id, 1, HygieneCheckOSUpdated, HygieneStatusPass, "")
	UpsertDeviceCheck(d.Id, 1, HygieneCheckAntivirusActive, HygieneStatusPass, "")
	UpsertDeviceCheck(d.Id, 1, HygieneCheckMFAEnabled, HygieneStatusPass, "")

	// Populate tech stack
	UpsertTechStackProfile(&TechStackProfile{
		UserId: 1, OrgId: 1, PrimaryOS: "macOS", Browser: "Safari",
	})

	// Create and verify board report
	now := time.Now().UTC()
	br := &BoardReport{
		OrgId: 1, CreatedBy: 1, Title: "Full Functional Test",
		PeriodStart: now.AddDate(0, -6, 0), PeriodEnd: now,
	}
	PostBoardReport(br)

	snap, _ := GenerateBoardReportSnapshot(1, br.PeriodStart, br.PeriodEnd)

	// Hygiene section should be populated
	if snap.Hygiene.TotalDevices != 1 {
		t.Fatalf("expected 1 device, got %d", snap.Hygiene.TotalDevices)
	}
	// 3 pass / 3 total = 100%
	if snap.Hygiene.AvgScore != 100 {
		t.Fatalf("expected avg score 100, got %.0f", snap.Hygiene.AvgScore)
	}
	if snap.Hygiene.FullyCompliant != 1 {
		t.Fatalf("expected 1 fully compliant, got %d", snap.Hygiene.FullyCompliant)
	}

	// Security posture should be computed
	if snap.SecurityPostureScore <= 0 {
		t.Fatalf("expected positive security posture, got %f", snap.SecurityPostureScore)
	}

	// Recommendations should be present
	if len(snap.Recommendations) == 0 {
		t.Fatal("expected at least one recommendation")
	}
}

func TestFunctional_OrgIsolation(t *testing.T) {
	teardown := setupIntegrationTest(t)
	defer teardown()

	// Org 1 data
	PostDevice(&UserDevice{UserId: 1, OrgId: 1, Name: "Org1 Device"})
	PostBoardReport(&BoardReport{
		OrgId: 1, CreatedBy: 1, Title: "Org1 Report",
		PeriodStart: time.Now().UTC(), PeriodEnd: time.Now().UTC(),
	})

	// Org 2 data
	PostDevice(&UserDevice{UserId: 2, OrgId: 2, Name: "Org2 Device"})
	PostBoardReport(&BoardReport{
		OrgId: 2, CreatedBy: 2, Title: "Org2 Report",
		PeriodStart: time.Now().UTC(), PeriodEnd: time.Now().UTC(),
	})

	// Verify isolation
	devices1, _ := GetOrgDevices(1)
	devices2, _ := GetOrgDevices(2)
	if len(devices1) != 1 || len(devices2) != 1 {
		t.Fatalf("expected 1 device per org, got org1=%d org2=%d", len(devices1), len(devices2))
	}

	reports1, _ := GetBoardReports(1)
	reports2, _ := GetBoardReports(2)
	if len(reports1) != 1 || len(reports2) != 1 {
		t.Fatalf("expected 1 report per org, got org1=%d org2=%d", len(reports1), len(reports2))
	}

	// Cross-org access should fail
	_, err := GetBoardReport(reports1[0].Id, 2)
	if err == nil {
		t.Fatal("expected error accessing org1 report from org2")
	}
}

func TestFunctional_UpsertCheckIdempotent(t *testing.T) {
	teardown := setupIntegrationTest(t)
	defer teardown()

	d := &UserDevice{UserId: 1, OrgId: 1, Name: "Idempotent Device"}
	PostDevice(d)

	// Upsert the same check 3 times
	UpsertDeviceCheck(d.Id, 1, HygieneCheckMFAEnabled, HygieneStatusFail, "First")
	UpsertDeviceCheck(d.Id, 1, HygieneCheckMFAEnabled, HygieneStatusFail, "Second")
	UpsertDeviceCheck(d.Id, 1, HygieneCheckMFAEnabled, HygieneStatusPass, "Third")

	fetched, _ := GetDevice(d.Id, 1)
	if len(fetched.Checks) != 1 {
		t.Fatalf("expected 1 check (upsert), got %d", len(fetched.Checks))
	}
	if fetched.Checks[0].Status != HygieneStatusPass {
		t.Fatalf("expected latest status pass, got %s", fetched.Checks[0].Status)
	}
	if fetched.Checks[0].Note != "Third" {
		t.Fatalf("expected latest note 'Third', got %s", fetched.Checks[0].Note)
	}
}

func TestFunctional_DeleteDeviceCascadesChecks(t *testing.T) {
	teardown := setupIntegrationTest(t)
	defer teardown()

	d := &UserDevice{UserId: 1, OrgId: 1, Name: "Cascade Device"}
	PostDevice(d)

	UpsertDeviceCheck(d.Id, 1, HygieneCheckOSUpdated, HygieneStatusPass, "")
	UpsertDeviceCheck(d.Id, 1, HygieneCheckMFAEnabled, HygieneStatusPass, "")
	UpsertDeviceCheck(d.Id, 1, HygieneCheckVPNEnabled, HygieneStatusFail, "")

	DeleteDevice(d.Id, 1)

	// Verify checks are gone too
	var count int
	db.Table("device_hygiene_checks").Where("device_id = ?", d.Id).Count(&count)
	if count != 0 {
		t.Fatalf("expected 0 orphan checks after device delete, got %d", count)
	}
}

func TestFunctional_EmptyOrgSummary(t *testing.T) {
	teardown := setupIntegrationTest(t)
	defer teardown()

	summary, err := GetOrgHygieneSummary(999)
	if err != nil {
		t.Fatalf("should not error on empty org: %v", err)
	}
	if summary.TotalDevices != 0 {
		t.Fatalf("expected 0 devices, got %d", summary.TotalDevices)
	}
	if summary.AvgScore != 0 {
		t.Fatalf("expected 0 avg score for empty org, got %f", summary.AvgScore)
	}
}

func TestFunctional_SnapshotRecommendations_Hygiene(t *testing.T) {
	teardown := setupIntegrationTest(t)
	defer teardown()

	// Create 3 at-risk devices (score < 50)
	for i := 0; i < 3; i++ {
		d := &UserDevice{UserId: 1, OrgId: 1, Name: "AtRisk"}
		PostDevice(d)
		UpsertDeviceCheck(d.Id, 1, HygieneCheckOSUpdated, HygieneStatusFail, "")
		UpsertDeviceCheck(d.Id, 1, HygieneCheckMFAEnabled, HygieneStatusFail, "")
	}

	snap, _ := GenerateBoardReportSnapshot(1, time.Now().AddDate(0, -3, 0), time.Now())

	if snap.Hygiene.AtRiskDevices != 3 {
		t.Fatalf("expected 3 at-risk, got %d", snap.Hygiene.AtRiskDevices)
	}

	// Should have hygiene-related recommendation
	found := false
	for _, r := range snap.Recommendations {
		if len(r) > 10 {
			found = true
		}
	}
	if !found {
		t.Fatal("expected recommendation for at-risk devices")
	}
}

func TestFunctional_ScoreEdgeCases(t *testing.T) {
	teardown := setupIntegrationTest(t)
	defer teardown()

	// Device with no checks — score should remain 0
	d := &UserDevice{UserId: 1, OrgId: 1, Name: "No Checks"}
	PostDevice(d)
	fetched, _ := GetDevice(d.Id, 1)
	if fetched.HygieneScore != 0 {
		t.Fatalf("expected 0 with no checks, got %d", fetched.HygieneScore)
	}

	// Device with 1 pass, 1 fail = 50%
	d2 := &UserDevice{UserId: 1, OrgId: 1, Name: "Half"}
	PostDevice(d2)
	UpsertDeviceCheck(d2.Id, 1, HygieneCheckOSUpdated, HygieneStatusPass, "")
	UpsertDeviceCheck(d2.Id, 1, HygieneCheckMFAEnabled, HygieneStatusFail, "")
	fetched2, _ := GetDevice(d2.Id, 1)
	if fetched2.HygieneScore != 50 {
		t.Fatalf("expected 50, got %d", fetched2.HygieneScore)
	}

	// Device with 1 pass, 2 unknown = 33%
	d3 := &UserDevice{UserId: 1, OrgId: 1, Name: "ThirdPass"}
	PostDevice(d3)
	UpsertDeviceCheck(d3.Id, 1, HygieneCheckOSUpdated, HygieneStatusPass, "")
	UpsertDeviceCheck(d3.Id, 1, HygieneCheckMFAEnabled, HygieneStatusUnknown, "")
	UpsertDeviceCheck(d3.Id, 1, HygieneCheckVPNEnabled, HygieneStatusUnknown, "")
	fetched3, _ := GetDevice(d3.Id, 1)
	if fetched3.HygieneScore != 33 {
		t.Fatalf("expected 33, got %d", fetched3.HygieneScore)
	}
}
