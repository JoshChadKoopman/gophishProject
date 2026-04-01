package models

import "time"

// MFADevice stores the TOTP configuration for a user.
// totp_secret is stored AES-256-GCM encrypted; never plain-text.
type MFADevice struct {
	ID         int64      `json:"id"`
	UserID     int64      `json:"user_id"`
	TOTPSecret string     `json:"-"` // encrypted; never serialised to JSON
	Enabled    bool       `json:"enabled"`
	EnrolledAt *time.Time `json:"enrolled_at,omitempty"`
}

// MFABackupCode stores a single bcrypt-hashed one-time backup code.
type MFABackupCode struct {
	ID        int64      `json:"id"`
	UserID    int64      `json:"user_id"`
	CodeHash  string     `json:"-"` // bcrypt hash; never serialised
	Used      bool       `json:"used"`
	UsedAt    *time.Time `json:"used_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// MFAAttempt records one MFA verification attempt (success or failure).
type MFAAttempt struct {
	ID          int64     `json:"id"`
	UserID      int64     `json:"user_id"`
	AttemptedAt time.Time `json:"attempted_at"`
	Success     bool      `json:"success"`
	IPAddress   string    `json:"ip_address,omitempty"`
}

// DeviceFingerprint stores a hashed device fingerprint used for the
// "Remember this device for 30 days" feature.
type DeviceFingerprint struct {
	ID              int64     `json:"id"`
	UserID          int64     `json:"user_id"`
	FingerprintHash string    `json:"-"` // bcrypt hash of raw fingerprint
	ExpiresAt       time.Time `json:"expires_at"`
	CreatedAt       time.Time `json:"created_at"`
}

// GetMFADevice returns the MFA device record for the given user.
// Returns an error if no device exists (gorm.ErrRecordNotFound).
func GetMFADevice(userID int64) (MFADevice, error) {
	device := MFADevice{}
	err := db.Where("user_id = ?", userID).First(&device).Error
	return device, err
}

// CreateOrUpdateMFADevice upserts the MFA device record for a user.
func CreateOrUpdateMFADevice(d *MFADevice) error {
	existing := MFADevice{}
	err := db.Where("user_id = ?", d.UserID).First(&existing).Error
	if err != nil {
		// No existing record — create
		return db.Create(d).Error
	}
	d.ID = existing.ID
	return db.Save(d).Error
}

// EnableMFADevice marks a user's MFA device as enrolled and active.
func EnableMFADevice(userID int64) error {
	now := time.Now().UTC()
	return db.Model(&MFADevice{}).
		Where("user_id = ?", userID).
		Updates(map[string]interface{}{
			"enabled":     true,
			"enrolled_at": now,
		}).Error
}

// SaveMFABackupCodes stores a new set of bcrypt-hashed backup codes for a user,
// replacing any existing unused codes.
func SaveMFABackupCodes(userID int64, hashes []string) error {
	// Delete existing codes
	if err := db.Where("user_id = ?", userID).Delete(&MFABackupCode{}).Error; err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, h := range hashes {
		code := MFABackupCode{
			UserID:    userID,
			CodeHash:  h,
			CreatedAt: now,
		}
		if err := db.Create(&code).Error; err != nil {
			return err
		}
	}
	return nil
}

// GetUnusedBackupCodes returns all unused backup codes for the given user.
func GetUnusedBackupCodes(userID int64) ([]MFABackupCode, error) {
	codes := []MFABackupCode{}
	err := db.Where("user_id = ? AND used = ?", userID, false).Find(&codes).Error
	return codes, err
}

// MarkBackupCodeUsed marks a backup code as consumed.
func MarkBackupCodeUsed(id int64) error {
	now := time.Now().UTC()
	return db.Model(&MFABackupCode{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"used":    true,
			"used_at": now,
		}).Error
}

// RecordMFAAttempt inserts an MFA attempt record.
func RecordMFAAttempt(userID int64, success bool, ip string) error {
	attempt := MFAAttempt{
		UserID:      userID,
		AttemptedAt: time.Now().UTC(),
		Success:     success,
		IPAddress:   ip,
	}
	return db.Create(&attempt).Error
}

// CountRecentMFAFailures returns the number of failed MFA attempts for the
// given user since the provided time. Used to enforce lockout policy.
func CountRecentMFAFailures(userID int64, since time.Time) (int, error) {
	var count int
	err := db.Model(&MFAAttempt{}).
		Where("user_id = ? AND success = ? AND attempted_at > ?", userID, false, since).
		Count(&count).Error
	return count, err
}

// FindDeviceFingerprint looks up a non-expired device fingerprint by its hash.
// Returns gorm.ErrRecordNotFound if no match.
func FindDeviceFingerprint(userID int64, hash string) (DeviceFingerprint, error) {
	fp := DeviceFingerprint{}
	err := db.Where("user_id = ? AND fingerprint_hash = ? AND expires_at > ?",
		userID, hash, time.Now().UTC()).First(&fp).Error
	return fp, err
}

// CreateDeviceFingerprint stores a new device fingerprint record.
func CreateDeviceFingerprint(fp *DeviceFingerprint) error {
	fp.CreatedAt = time.Now().UTC()
	return db.Create(fp).Error
}

// PurgeExpiredFingerprints removes device fingerprints that have passed their
// expiry. Should be called periodically (e.g., on server startup or via a cron).
func PurgeExpiredFingerprints() error {
	return db.Where("expires_at < ?", time.Now().UTC()).Delete(&DeviceFingerprint{}).Error
}
