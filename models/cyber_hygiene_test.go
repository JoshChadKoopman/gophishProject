package models

import (
	"testing"

	"github.com/gophish/gophish/config"
)

func setupHygieneTest(t *testing.T) func() {
	t.Helper()
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	if err := Setup(conf); err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	db.Exec("DELETE FROM device_hygiene_checks")
	db.Exec("DELETE FROM user_devices")
	db.Exec("DELETE FROM tech_stack_profiles")
	return func() {
		db.Exec("DELETE FROM device_hygiene_checks")
		db.Exec("DELETE FROM user_devices")
		db.Exec("DELETE FROM tech_stack_profiles")
	}
}

// ---- Device CRUD ----

func TestPostDevice(t *testing.T) {
	teardown := setupHygieneTest(t)
	defer teardown()

	d := &UserDevice{UserId: 1, OrgId: 1, Name: "My Laptop", DeviceType: DeviceTypeLaptop, OS: "macOS"}
	if err := PostDevice(d); err != nil {
		t.Fatalf("PostDevice: %v", err)
	}
	if d.Id == 0 {
		t.Fatal("expected non-zero ID")
	}
	if d.HygieneScore != 0 {
		t.Fatalf("expected initial score 0, got %d", d.HygieneScore)
	}
}

func TestGetUserDevices(t *testing.T) {
	teardown := setupHygieneTest(t)
	defer teardown()

	PostDevice(&UserDevice{UserId: 1, OrgId: 1, Name: "Laptop", DeviceType: DeviceTypeLaptop})
	PostDevice(&UserDevice{UserId: 1, OrgId: 1, Name: "Phone", DeviceType: DeviceTypeMobile})

	devices, err := GetUserDevices(1, 1)
	if err != nil {
		t.Fatalf("GetUserDevices: %v", err)
	}
	if len(devices) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(devices))
	}
}

func TestGetDevice(t *testing.T) {
	teardown := setupHygieneTest(t)
	defer teardown()

	d := &UserDevice{UserId: 1, OrgId: 1, Name: "Test Device"}
	PostDevice(d)

	fetched, err := GetDevice(d.Id, 1)
	if err != nil {
		t.Fatalf("GetDevice: %v", err)
	}
	if fetched.Name != "Test Device" {
		t.Fatalf("wrong name: %s", fetched.Name)
	}
}

func TestGetDeviceNotFound(t *testing.T) {
	teardown := setupHygieneTest(t)
	defer teardown()

	_, err := GetDevice(999, 1)
	if err != ErrDeviceNotFound {
		t.Fatalf("expected ErrDeviceNotFound, got %v", err)
	}
}

func TestPutDevice(t *testing.T) {
	teardown := setupHygieneTest(t)
	defer teardown()

	d := &UserDevice{UserId: 1, OrgId: 1, Name: "Original"}
	PostDevice(d)

	d.Name = "Updated"
	d.OS = "Windows 11"
	if err := PutDevice(d); err != nil {
		t.Fatalf("PutDevice: %v", err)
	}
	fetched, _ := GetDevice(d.Id, 1)
	if fetched.Name != "Updated" {
		t.Fatalf("expected Updated, got %s", fetched.Name)
	}
}

func TestDeleteDevice(t *testing.T) {
	teardown := setupHygieneTest(t)
	defer teardown()

	d := &UserDevice{UserId: 1, OrgId: 1, Name: "Delete Me"}
	PostDevice(d)
	UpsertDeviceCheck(d.Id, 1, HygieneCheckOSUpdated, HygieneStatusPass, "")

	if err := DeleteDevice(d.Id, 1); err != nil {
		t.Fatalf("DeleteDevice: %v", err)
	}
	_, err := GetDevice(d.Id, 1)
	if err != ErrDeviceNotFound {
		t.Fatal("expected device deleted")
	}
}

func TestDeviceValidation(t *testing.T) {
	d := &UserDevice{}
	if err := d.Validate(); err == nil {
		t.Fatal("expected error for empty name")
	}
	d.Name = "Valid"
	if err := d.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if d.DeviceType != DeviceTypeOther {
		t.Fatalf("expected default device type 'other', got %s", d.DeviceType)
	}
}

// ---- Hygiene Checks ----

func TestUpsertDeviceCheck(t *testing.T) {
	teardown := setupHygieneTest(t)
	defer teardown()

	d := &UserDevice{UserId: 1, OrgId: 1, Name: "Check Device"}
	PostDevice(d)

	err := UpsertDeviceCheck(d.Id, 1, HygieneCheckOSUpdated, HygieneStatusPass, "Updated today")
	if err != nil {
		t.Fatalf("UpsertDeviceCheck: %v", err)
	}

	fetched, _ := GetDevice(d.Id, 1)
	if len(fetched.Checks) != 1 {
		t.Fatalf("expected 1 check, got %d", len(fetched.Checks))
	}
	if fetched.Checks[0].Status != HygieneStatusPass {
		t.Fatalf("expected pass, got %s", fetched.Checks[0].Status)
	}
}

func TestUpsertDeviceCheckUpdate(t *testing.T) {
	teardown := setupHygieneTest(t)
	defer teardown()

	d := &UserDevice{UserId: 1, OrgId: 1, Name: "Update Check"}
	PostDevice(d)

	UpsertDeviceCheck(d.Id, 1, HygieneCheckMFAEnabled, HygieneStatusFail, "")
	UpsertDeviceCheck(d.Id, 1, HygieneCheckMFAEnabled, HygieneStatusPass, "Enabled now")

	fetched, _ := GetDevice(d.Id, 1)
	if len(fetched.Checks) != 1 {
		t.Fatalf("expected 1 check (upsert), got %d", len(fetched.Checks))
	}
	if fetched.Checks[0].Status != HygieneStatusPass {
		t.Fatalf("expected updated to pass, got %s", fetched.Checks[0].Status)
	}
}

func TestHygieneScoreCalculation(t *testing.T) {
	teardown := setupHygieneTest(t)
	defer teardown()

	d := &UserDevice{UserId: 1, OrgId: 1, Name: "Score Device"}
	PostDevice(d)

	UpsertDeviceCheck(d.Id, 1, HygieneCheckOSUpdated, HygieneStatusPass, "")
	UpsertDeviceCheck(d.Id, 1, HygieneCheckAntivirusActive, HygieneStatusPass, "")
	UpsertDeviceCheck(d.Id, 1, HygieneCheckDiskEncrypted, HygieneStatusFail, "")
	UpsertDeviceCheck(d.Id, 1, HygieneCheckScreenLock, HygieneStatusPass, "")

	fetched, _ := GetDevice(d.Id, 1)
	// 3 pass / 4 total = 75%
	if fetched.HygieneScore != 75 {
		t.Fatalf("expected score 75, got %d", fetched.HygieneScore)
	}
}

func TestHygieneScore100(t *testing.T) {
	teardown := setupHygieneTest(t)
	defer teardown()

	d := &UserDevice{UserId: 1, OrgId: 1, Name: "Perfect"}
	PostDevice(d)

	allChecks := []string{
		HygieneCheckOSUpdated, HygieneCheckAntivirusActive, HygieneCheckDiskEncrypted,
		HygieneCheckScreenLock, HygieneCheckPasswordManager, HygieneCheckVPNEnabled, HygieneCheckMFAEnabled,
	}
	for _, c := range allChecks {
		UpsertDeviceCheck(d.Id, 1, c, HygieneStatusPass, "")
	}

	fetched, _ := GetDevice(d.Id, 1)
	if fetched.HygieneScore != 100 {
		t.Fatalf("expected 100, got %d", fetched.HygieneScore)
	}
}

func TestUpsertCheckDeviceNotFound(t *testing.T) {
	teardown := setupHygieneTest(t)
	defer teardown()

	err := UpsertDeviceCheck(999, 1, HygieneCheckMFAEnabled, HygieneStatusPass, "")
	if err != ErrDeviceNotFound {
		t.Fatalf("expected ErrDeviceNotFound, got %v", err)
	}
}

// ---- Tech Stack Profile ----

func TestUpsertTechStackProfile(t *testing.T) {
	teardown := setupHygieneTest(t)
	defer teardown()

	p := &TechStackProfile{
		UserId: 1, OrgId: 1, PrimaryOS: "macOS", Browser: "Chrome",
		EmailClient: "Outlook", RemoteAccess: "Cisco AnyConnect",
	}
	if err := UpsertTechStackProfile(p); err != nil {
		t.Fatalf("UpsertTechStackProfile: %v", err)
	}
	if p.Id == 0 {
		t.Fatal("expected non-zero ID")
	}

	// Update
	p.Browser = "Firefox"
	if err := UpsertTechStackProfile(p); err != nil {
		t.Fatalf("update: %v", err)
	}

	fetched, err := GetTechStackProfile(1, 1)
	if err != nil {
		t.Fatalf("GetTechStackProfile: %v", err)
	}
	if fetched.Browser != "Firefox" {
		t.Fatalf("expected Firefox, got %s", fetched.Browser)
	}
}

func TestGetTechStackProfileNotFound(t *testing.T) {
	teardown := setupHygieneTest(t)
	defer teardown()

	_, err := GetTechStackProfile(999, 1)
	if err == nil {
		t.Fatal("expected error for missing profile")
	}
}

// ---- Personalized Checks ----

func TestGetPersonalizedChecksNoProfile(t *testing.T) {
	teardown := setupHygieneTest(t)
	defer teardown()

	checks := GetPersonalizedChecks(999, 1)
	if len(checks) != 7 {
		t.Fatalf("expected 7 checks, got %d", len(checks))
	}
	for _, c := range checks {
		if !c.Relevant {
			t.Fatalf("expected all checks relevant when no profile, %s was not", c.CheckType)
		}
	}
}

func TestGetPersonalizedChecksWithProfile(t *testing.T) {
	teardown := setupHygieneTest(t)
	defer teardown()

	UpsertTechStackProfile(&TechStackProfile{
		UserId: 1, OrgId: 1, PrimaryOS: "Windows", Browser: "Edge",
		EmailClient: "Outlook", RemoteAccess: "FortiClient VPN",
	})

	checks := GetPersonalizedChecks(1, 1)
	if len(checks) != 7 {
		t.Fatalf("expected 7 checks, got %d", len(checks))
	}

	// Verify personalized reasons contain OS info
	for _, c := range checks {
		if c.CheckType == HygieneCheckOSUpdated && c.Reason == "" {
			t.Fatal("expected personalized reason for OS check")
		}
		if c.CheckType == HygieneCheckVPNEnabled {
			if c.Reason == "" {
				t.Fatal("expected VPN reason with FortiClient")
			}
		}
	}
}

// ---- Org Summary ----

func TestGetOrgHygieneSummary(t *testing.T) {
	teardown := setupHygieneTest(t)
	defer teardown()

	d := &UserDevice{UserId: 1, OrgId: 1, Name: "Summary Device"}
	PostDevice(d)
	UpsertDeviceCheck(d.Id, 1, HygieneCheckOSUpdated, HygieneStatusPass, "")
	UpsertDeviceCheck(d.Id, 1, HygieneCheckMFAEnabled, HygieneStatusFail, "")

	summary, err := GetOrgHygieneSummary(1)
	if err != nil {
		t.Fatalf("GetOrgHygieneSummary: %v", err)
	}
	if summary.TotalDevices != 1 {
		t.Fatalf("expected 1 device, got %d", summary.TotalDevices)
	}
	if summary.PassCount != 1 {
		t.Fatalf("expected 1 pass, got %d", summary.PassCount)
	}
	if summary.FailCount != 1 {
		t.Fatalf("expected 1 fail, got %d", summary.FailCount)
	}
}

func TestGetOrgHygieneEnrichedSummary(t *testing.T) {
	teardown := setupHygieneTest(t)
	defer teardown()

	d := &UserDevice{UserId: 1, OrgId: 1, Name: "Enriched", DeviceType: DeviceTypeLaptop, OS: "macOS"}
	PostDevice(d)
	// All checks pass = fully compliant
	allChecks := []string{
		HygieneCheckOSUpdated, HygieneCheckAntivirusActive, HygieneCheckDiskEncrypted,
		HygieneCheckScreenLock, HygieneCheckPasswordManager, HygieneCheckVPNEnabled, HygieneCheckMFAEnabled,
	}
	for _, c := range allChecks {
		UpsertDeviceCheck(d.Id, 1, c, HygieneStatusPass, "")
	}

	enriched, err := GetOrgHygieneEnrichedSummary(1)
	if err != nil {
		t.Fatalf("GetOrgHygieneEnrichedSummary: %v", err)
	}
	if enriched.TotalDevices != 1 {
		t.Fatalf("expected 1 device, got %d", enriched.TotalDevices)
	}
	if enriched.FullyCompliant != 1 {
		t.Fatalf("expected 1 fully compliant, got %d", enriched.FullyCompliant)
	}
	if enriched.OSBreakdown["macOS"] != 1 {
		t.Fatal("expected macOS in OS breakdown")
	}
	if enriched.DeviceTypeBreakdown[DeviceTypeLaptop] != 1 {
		t.Fatal("expected laptop in device type breakdown")
	}
}

func TestGetOrgDevicesEnriched(t *testing.T) {
	teardown := setupHygieneTest(t)
	defer teardown()

	d := &UserDevice{UserId: 1, OrgId: 1, Name: "Enriched Device", DeviceType: DeviceTypeLaptop}
	PostDevice(d)

	views, err := GetOrgDevicesEnriched(1)
	if err != nil {
		t.Fatalf("GetOrgDevicesEnriched: %v", err)
	}
	if len(views) != 1 {
		t.Fatalf("expected 1 view, got %d", len(views))
	}
}

// ---- Multiple Devices ----

func TestMultipleDevicesScoring(t *testing.T) {
	teardown := setupHygieneTest(t)
	defer teardown()

	d1 := &UserDevice{UserId: 1, OrgId: 1, Name: "Device 1"}
	d2 := &UserDevice{UserId: 1, OrgId: 1, Name: "Device 2"}
	PostDevice(d1)
	PostDevice(d2)

	UpsertDeviceCheck(d1.Id, 1, HygieneCheckOSUpdated, HygieneStatusPass, "")
	UpsertDeviceCheck(d1.Id, 1, HygieneCheckMFAEnabled, HygieneStatusPass, "")

	UpsertDeviceCheck(d2.Id, 1, HygieneCheckOSUpdated, HygieneStatusFail, "")
	UpsertDeviceCheck(d2.Id, 1, HygieneCheckMFAEnabled, HygieneStatusFail, "")

	fetched1, _ := GetDevice(d1.Id, 1)
	fetched2, _ := GetDevice(d2.Id, 1)

	if fetched1.HygieneScore != 100 {
		t.Fatalf("device1 expected 100, got %d", fetched1.HygieneScore)
	}
	if fetched2.HygieneScore != 0 {
		t.Fatalf("device2 expected 0, got %d", fetched2.HygieneScore)
	}
}
