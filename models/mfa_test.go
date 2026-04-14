package models

import (
	"testing"
	"time"

	"github.com/gophish/gophish/config"
)

func setupMFATest(t *testing.T) func() {
	t.Helper()
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	if err := Setup(conf); err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	db.Exec("DELETE FROM mfa_devices")
	db.Exec("DELETE FROM mfa_backup_codes")
	db.Exec("DELETE FROM mfa_attempts")
	db.Exec("DELETE FROM device_fingerprints")
	return func() {
		db.Exec("DELETE FROM mfa_devices")
		db.Exec("DELETE FROM mfa_backup_codes")
		db.Exec("DELETE FROM mfa_attempts")
		db.Exec("DELETE FROM device_fingerprints")
	}
}

// ---------- CreateOrUpdateMFADevice ----------

func TestCreateMFADevice(t *testing.T) {
	teardown := setupMFATest(t)
	defer teardown()

	d := &MFADevice{
		UserID:     1,
		TOTPSecret: "encrypted-secret-data",
		Enabled:    false,
	}
	if err := CreateOrUpdateMFADevice(d); err != nil {
		t.Fatalf("CreateOrUpdateMFADevice (create) failed: %v", err)
	}
	if d.ID == 0 {
		t.Fatalf("expected non-zero ID after create")
	}
}

func TestUpdateMFADevice(t *testing.T) {
	teardown := setupMFATest(t)
	defer teardown()

	d := &MFADevice{
		UserID:     1,
		TOTPSecret: "original-secret",
		Enabled:    false,
	}
	CreateOrUpdateMFADevice(d)
	originalID := d.ID

	d2 := &MFADevice{
		UserID:     1,
		TOTPSecret: "updated-secret",
		Enabled:    true,
	}
	if err := CreateOrUpdateMFADevice(d2); err != nil {
		t.Fatalf("CreateOrUpdateMFADevice (update) failed: %v", err)
	}
	if d2.ID != originalID {
		t.Fatalf("expected ID to remain %d on update, got %d", originalID, d2.ID)
	}

	fetched, err := GetMFADevice(1)
	if err != nil {
		t.Fatalf("GetMFADevice failed: %v", err)
	}
	if fetched.TOTPSecret != "updated-secret" {
		t.Fatalf("expected TOTPSecret 'updated-secret', got %q", fetched.TOTPSecret)
	}
}

// ---------- GetMFADevice ----------

func TestGetMFADeviceFound(t *testing.T) {
	teardown := setupMFATest(t)
	defer teardown()

	d := &MFADevice{UserID: 1, TOTPSecret: "secret", Enabled: false}
	CreateOrUpdateMFADevice(d)

	fetched, err := GetMFADevice(1)
	if err != nil {
		t.Fatalf("GetMFADevice failed: %v", err)
	}
	if fetched.UserID != 1 {
		t.Fatalf("expected UserID 1, got %d", fetched.UserID)
	}
}

func TestGetMFADeviceNotFound(t *testing.T) {
	teardown := setupMFATest(t)
	defer teardown()

	_, err := GetMFADevice(99999)
	if err == nil {
		t.Fatalf("expected error for non-existent MFA device, got nil")
	}
}

// ---------- EnableMFADevice ----------

func TestEnableMFADevice(t *testing.T) {
	teardown := setupMFATest(t)
	defer teardown()

	d := &MFADevice{UserID: 1, TOTPSecret: "secret", Enabled: false}
	CreateOrUpdateMFADevice(d)

	if err := EnableMFADevice(1); err != nil {
		t.Fatalf("EnableMFADevice failed: %v", err)
	}

	fetched, err := GetMFADevice(1)
	if err != nil {
		t.Fatalf("GetMFADevice after enable failed: %v", err)
	}
	if !fetched.Enabled {
		t.Fatalf("expected Enabled to be true after EnableMFADevice")
	}
	if fetched.EnrolledAt == nil {
		t.Fatalf("expected EnrolledAt to be set after EnableMFADevice")
	}
}

// ---------- SaveMFABackupCodes / GetUnusedBackupCodes ----------

func TestSaveMFABackupCodesAndGetUnused(t *testing.T) {
	teardown := setupMFATest(t)
	defer teardown()

	hashes := []string{"hash-aaa", "hash-bbb", "hash-ccc"}
	if err := SaveMFABackupCodes(1, hashes); err != nil {
		t.Fatalf("SaveMFABackupCodes failed: %v", err)
	}

	codes, err := GetUnusedBackupCodes(1)
	if err != nil {
		t.Fatalf("GetUnusedBackupCodes failed: %v", err)
	}
	if len(codes) != 3 {
		t.Fatalf("expected 3 unused backup codes, got %d", len(codes))
	}
	for _, c := range codes {
		if c.Used {
			t.Fatalf("expected code to be unused, but Used=true for ID %d", c.ID)
		}
	}
}

func TestSaveMFABackupCodesReplacesExisting(t *testing.T) {
	teardown := setupMFATest(t)
	defer teardown()

	SaveMFABackupCodes(1, []string{"old-hash-1", "old-hash-2"})
	SaveMFABackupCodes(1, []string{"new-hash-1", "new-hash-2", "new-hash-3"})

	codes, err := GetUnusedBackupCodes(1)
	if err != nil {
		t.Fatalf("GetUnusedBackupCodes failed: %v", err)
	}
	if len(codes) != 3 {
		t.Fatalf("expected 3 codes after replacement, got %d", len(codes))
	}
}

// ---------- MarkBackupCodeUsed ----------

func TestMarkBackupCodeUsed(t *testing.T) {
	teardown := setupMFATest(t)
	defer teardown()

	SaveMFABackupCodes(1, []string{"hash-aaa", "hash-bbb", "hash-ccc"})

	codes, _ := GetUnusedBackupCodes(1)
	if len(codes) != 3 {
		t.Fatalf("expected 3 unused codes, got %d", len(codes))
	}

	// Mark the first code as used
	if err := MarkBackupCodeUsed(codes[0].ID); err != nil {
		t.Fatalf("MarkBackupCodeUsed failed: %v", err)
	}

	remaining, err := GetUnusedBackupCodes(1)
	if err != nil {
		t.Fatalf("GetUnusedBackupCodes after mark failed: %v", err)
	}
	if len(remaining) != 2 {
		t.Fatalf("expected 2 unused codes after marking one, got %d", len(remaining))
	}
}

// ---------- RecordMFAAttempt / CountRecentMFAFailures ----------

func TestRecordMFAAttemptAndCountFailures(t *testing.T) {
	teardown := setupMFATest(t)
	defer teardown()

	since := time.Now().UTC().Add(-1 * time.Minute)

	// Record 3 failures and 1 success
	if err := RecordMFAAttempt(1, false, "10.0.0.1"); err != nil {
		t.Fatalf("RecordMFAAttempt (fail 1) failed: %v", err)
	}
	if err := RecordMFAAttempt(1, false, "10.0.0.1"); err != nil {
		t.Fatalf("RecordMFAAttempt (fail 2) failed: %v", err)
	}
	if err := RecordMFAAttempt(1, false, "10.0.0.2"); err != nil {
		t.Fatalf("RecordMFAAttempt (fail 3) failed: %v", err)
	}
	if err := RecordMFAAttempt(1, true, "10.0.0.1"); err != nil {
		t.Fatalf("RecordMFAAttempt (success) failed: %v", err)
	}

	count, err := CountRecentMFAFailures(1, since)
	if err != nil {
		t.Fatalf("CountRecentMFAFailures failed: %v", err)
	}
	if count != 3 {
		t.Fatalf("expected 3 failures, got %d", count)
	}
}

func TestCountRecentMFAFailuresNoFailures(t *testing.T) {
	teardown := setupMFATest(t)
	defer teardown()

	since := time.Now().UTC().Add(-1 * time.Minute)

	count, err := CountRecentMFAFailures(1, since)
	if err != nil {
		t.Fatalf("CountRecentMFAFailures failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected 0 failures, got %d", count)
	}
}

// ---------- CreateDeviceFingerprint / FindDeviceFingerprint ----------

func TestCreateAndFindDeviceFingerprint(t *testing.T) {
	teardown := setupMFATest(t)
	defer teardown()

	fp := &DeviceFingerprint{
		UserID:          1,
		FingerprintHash: "fp-hash-abc123",
		ExpiresAt:       time.Now().UTC().Add(30 * 24 * time.Hour),
	}
	if err := CreateDeviceFingerprint(fp); err != nil {
		t.Fatalf("CreateDeviceFingerprint failed: %v", err)
	}
	if fp.ID == 0 {
		t.Fatalf("expected non-zero ID after create")
	}

	found, err := FindDeviceFingerprint(1, "fp-hash-abc123")
	if err != nil {
		t.Fatalf("FindDeviceFingerprint failed: %v", err)
	}
	if found.FingerprintHash != "fp-hash-abc123" {
		t.Fatalf("expected hash 'fp-hash-abc123', got %q", found.FingerprintHash)
	}
}

func TestFindDeviceFingerprintNotFound(t *testing.T) {
	teardown := setupMFATest(t)
	defer teardown()

	_, err := FindDeviceFingerprint(1, "nonexistent-hash")
	if err == nil {
		t.Fatalf("expected error for non-existent fingerprint, got nil")
	}
}

func TestFindDeviceFingerprintExpired(t *testing.T) {
	teardown := setupMFATest(t)
	defer teardown()

	fp := &DeviceFingerprint{
		UserID:          1,
		FingerprintHash: "expired-hash",
		ExpiresAt:       time.Now().UTC().Add(-1 * time.Hour), // already expired
	}
	CreateDeviceFingerprint(fp)

	_, err := FindDeviceFingerprint(1, "expired-hash")
	if err == nil {
		t.Fatalf("expected error for expired fingerprint, got nil")
	}
}

// ---------- PurgeExpiredFingerprints ----------

func TestPurgeExpiredFingerprints(t *testing.T) {
	teardown := setupMFATest(t)
	defer teardown()

	// Create one expired and one valid fingerprint
	expired := &DeviceFingerprint{
		UserID:          1,
		FingerprintHash: "expired-fp",
		ExpiresAt:       time.Now().UTC().Add(-2 * time.Hour),
	}
	valid := &DeviceFingerprint{
		UserID:          1,
		FingerprintHash: "valid-fp",
		ExpiresAt:       time.Now().UTC().Add(30 * 24 * time.Hour),
	}
	CreateDeviceFingerprint(expired)
	CreateDeviceFingerprint(valid)

	if err := PurgeExpiredFingerprints(); err != nil {
		t.Fatalf("PurgeExpiredFingerprints failed: %v", err)
	}

	// The expired one should be gone
	_, err := FindDeviceFingerprint(1, "expired-fp")
	if err == nil {
		t.Fatalf("expected expired fingerprint to be purged, but found it")
	}

	// The valid one should still be there
	found, err := FindDeviceFingerprint(1, "valid-fp")
	if err != nil {
		t.Fatalf("expected valid fingerprint to survive purge, got error: %v", err)
	}
	if found.FingerprintHash != "valid-fp" {
		t.Fatalf("expected 'valid-fp', got %q", found.FingerprintHash)
	}
}
