package auth

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"image/png"
	"io"
	"time"

	"github.com/pquerna/otp"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

const (
	// MFAMaxAttempts is the number of failed MFA attempts before a lockout.
	MFAMaxAttempts = 5
	// MFALockoutDuration is how long a user is locked out after MFAMaxAttempts failures.
	MFALockoutDuration = 15 * time.Minute
	// DeviceRememberDuration is how long a trusted device fingerprint is valid.
	DeviceRememberDuration = 30 * 24 * time.Hour
	// BackupCodeLength is the number of backup codes generated on MFA enrollment.
	BackupCodeLength = 8
	// totpIssuer is the issuer name shown in authenticator apps.
	totpIssuer = "Nivoxis CyberAwareness"
)

// MFARequired returns true if the given role slug mandates MFA.
// Currently org_admin and superadmin must complete TOTP on every login
// unless a trusted device fingerprint is present.
func MFARequired(roleSlug string) bool {
	return roleSlug == "superadmin" || roleSlug == "org_admin"
}

// GenerateTOTPSecret creates a new TOTP secret for the given account email.
// Returns the raw secret string and a data-URI PNG of the QR code.
func GenerateTOTPSecret(accountEmail string) (secret, qrDataURI string, err error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      totpIssuer,
		AccountName: accountEmail,
		Algorithm:   otp.AlgorithmSHA1,
		Digits:      otp.DigitsSix,
		Period:      30,
	})
	if err != nil {
		return "", "", fmt.Errorf("mfa: failed to generate TOTP key: %w", err)
	}

	// Generate QR code as a base64-encoded PNG data URI.
	img, err := key.Image(256, 256)
	if err != nil {
		return "", "", fmt.Errorf("mfa: failed to render QR code: %w", err)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return "", "", fmt.Errorf("mfa: failed to encode QR PNG: %w", err)
	}
	qrDataURI = "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())

	return key.Secret(), qrDataURI, nil
}

// ValidateTOTP verifies a 6-digit TOTP code against an encrypted secret.
// The encryptedSecret should have been stored by EncryptTOTPSecret.
func ValidateTOTP(encryptedSecret, code string, aesKey []byte) bool {
	secret, err := DecryptTOTPSecret(encryptedSecret, aesKey)
	if err != nil {
		return false
	}
	return totp.Validate(code, secret)
}

// EncryptTOTPSecret encrypts a plain TOTP secret using AES-256-GCM and returns
// a base64-encoded ciphertext. aesKey must be exactly 32 bytes.
func EncryptTOTPSecret(secret string, aesKey []byte) (string, error) {
	if len(aesKey) != 32 {
		return "", errors.New("mfa: AES key must be exactly 32 bytes")
	}
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", fmt.Errorf("mfa: AES cipher init failed: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("mfa: GCM init failed: %w", err)
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("mfa: nonce generation failed: %w", err)
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(secret), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptTOTPSecret reverses EncryptTOTPSecret and returns the plain TOTP secret.
func DecryptTOTPSecret(encoded string, aesKey []byte) (string, error) {
	if len(aesKey) != 32 {
		return "", errors.New("mfa: AES key must be exactly 32 bytes")
	}
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("mfa: base64 decode failed: %w", err)
	}
	block, err := aes.NewCipher(aesKey)
	if err != nil {
		return "", fmt.Errorf("mfa: AES cipher init failed: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("mfa: GCM init failed: %w", err)
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("mfa: ciphertext too short")
	}
	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plain, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("mfa: GCM decryption failed: %w", err)
	}
	return string(plain), nil
}

// TOTPEncryptionKeyFromBase64 decodes a base64-encoded 32-byte AES key.
// Returns an error if the key is missing or the wrong size.
func TOTPEncryptionKeyFromBase64(b64 string) ([]byte, error) {
	if b64 == "" {
		return nil, errors.New("mfa: MFA_TOTP_ENCRYPTION_KEY is not set; refusing to store unencrypted secrets")
	}
	key, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, fmt.Errorf("mfa: failed to decode encryption key: %w", err)
	}
	if len(key) != 32 {
		return nil, fmt.Errorf("mfa: encryption key must be 32 bytes, got %d", len(key))
	}
	return key, nil
}

// GenerateBackupCodes generates n random hex backup codes and returns both
// the plaintext codes (shown once to the user) and their bcrypt hashes
// (stored in the database).
func GenerateBackupCodes(n int) (plaintext []string, hashed []string, err error) {
	for i := 0; i < n; i++ {
		raw := make([]byte, 5) // 10 hex chars per code
		if _, err := rand.Read(raw); err != nil {
			return nil, nil, fmt.Errorf("mfa: failed to generate backup code: %w", err)
		}
		code := fmt.Sprintf("%X", raw) // e.g. "A3F9C12B7E"
		hash, err := bcrypt.GenerateFromPassword([]byte(code), bcrypt.DefaultCost)
		if err != nil {
			return nil, nil, fmt.Errorf("mfa: failed to hash backup code: %w", err)
		}
		plaintext = append(plaintext, code)
		hashed = append(hashed, string(hash))
	}
	return plaintext, hashed, nil
}

// ValidateBackupCode checks whether plain matches a bcrypt hash.
func ValidateBackupCode(plain, bcryptHash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(bcryptHash), []byte(plain)) == nil
}

// DeviceFingerprintHash computes a bcrypt hash of the raw fingerprint string.
// The raw fingerprint is derived from User-Agent + Accept-Language headers.
func DeviceFingerprintHash(raw string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.MinCost)
	if err != nil {
		return "", fmt.Errorf("mfa: fingerprint hash failed: %w", err)
	}
	return string(hash), nil
}

// ValidateDeviceFingerprint checks whether the raw fingerprint matches a stored hash.
func ValidateDeviceFingerprint(raw, bcryptHash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(bcryptHash), []byte(raw)) == nil
}

// RawDeviceFingerprint builds a fingerprint string from the HTTP request headers.
// This is intentionally simple — not a security boundary, just a UX convenience.
func RawDeviceFingerprint(userAgent, acceptLang string) string {
	return userAgent + "|" + acceptLang
}
