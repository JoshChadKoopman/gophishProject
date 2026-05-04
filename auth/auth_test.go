package auth

import (
	"encoding/base64"
	"strings"
	"testing"
)

// Test-scoped constants to satisfy lint rules about repeated literals.
const (
	testEmail       = "test@example.com"
	testProviderURL = "http://keycloak:8080/realms/test"
	testRedirectURL = "http://localhost/cb"
)

// fatalIfErr is a test helper that fails the test if err is non-nil.
func fatalIfErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// expectErr is a test helper that asserts got matches expected.
func expectErr(t *testing.T, expected, got error) {
	t.Helper()
	if got != expected {
		t.Fatalf("expected error %v, got %v", expected, got)
	}
}

// ---------- Password policy tests ----------

func TestPasswordPolicy(t *testing.T) {
	candidate := "short"
	got := CheckPasswordPolicy(candidate)
	expectErr(t, ErrPasswordTooShort, got)

	// Must also fail for missing uppercase
	candidate = "alllowercase1"
	got = CheckPasswordPolicy(candidate)
	expectErr(t, ErrPasswordNoUpper, got)

	// Must fail for missing lowercase
	candidate = "ALLUPPERCASE1"
	got = CheckPasswordPolicy(candidate)
	expectErr(t, ErrPasswordNoLower, got)

	// Must fail for missing digit
	candidate = "NoDigitsHere"
	got = CheckPasswordPolicy(candidate)
	expectErr(t, ErrPasswordNoDigit, got)

	// Valid password: meets all requirements
	candidate = "ValidPass1ok!"
	got = CheckPasswordPolicy(candidate)
	expectErr(t, nil, got)
}

func TestPasswordPolicyEmpty(t *testing.T) {
	err := CheckPasswordPolicy("")
	if err != ErrEmptyPassword {
		t.Fatalf("expected ErrEmptyPassword, got %v", err)
	}
}

func TestPasswordPolicyExactMinLength(t *testing.T) {
	// Exactly MinPasswordLength chars, meets all char requirements including special char
	pw := "Abcdefg1x!" // 10 chars
	if len(pw) != MinPasswordLength {
		t.Fatalf("test setup error: password length is %d, expected %d", len(pw), MinPasswordLength)
	}
	err := CheckPasswordPolicy(pw)
	if err != nil {
		t.Fatalf("expected valid password at exact min length, got %v", err)
	}
}

func TestPasswordPolicyOneBelowMinLength(t *testing.T) {
	pw := "Abcdefg1x" // 9 chars
	if len(pw) != MinPasswordLength-1 {
		t.Fatalf("test setup error: password length is %d, expected %d", len(pw), MinPasswordLength-1)
	}
	err := CheckPasswordPolicy(pw)
	if err != ErrPasswordTooShort {
		t.Fatalf("expected ErrPasswordTooShort, got %v", err)
	}
}

// ---------- ValidatePasswordChange tests ----------

func TestValidatePasswordChange(t *testing.T) {
	newPassword := "ValidPass1ok!"
	confirmPassword := "Mismatch1no!"
	currentPassword := "CurrentPass1!"
	currentHash, err := GeneratePasswordHash(currentPassword)
	fatalIfErr(t, err)

	_, got := ValidatePasswordChange(currentHash, newPassword, confirmPassword)
	expectErr(t, ErrPasswordMismatch, got)

	newPassword = currentPassword
	confirmPassword = newPassword
	_, got = ValidatePasswordChange(currentHash, newPassword, confirmPassword)
	expectErr(t, ErrReusedPassword, got)
}

func TestValidatePasswordChangeSuccess(t *testing.T) {
	currentPassword := "OldPassword1!"
	currentHash, err := GeneratePasswordHash(currentPassword)
	if err != nil {
		t.Fatalf("error generating hash: %v", err)
	}
	newPassword := "NewPassword2@"
	hash, err := ValidatePasswordChange(currentHash, newPassword, newPassword)
	if err != nil {
		fatalIfErr(t, err)
	}
	// The returned hash should match the new password
	if err := ValidatePassword(newPassword, hash); err != nil {
		t.Fatal("returned hash does not match new password")
	}
}

func TestValidatePasswordChangePolicyViolation(t *testing.T) {
	currentHash, _ := GeneratePasswordHash("OldPassword1")
	// New password violates policy (no digit)
	_, err := ValidatePasswordChange(currentHash, "NoDigitHereX", "NoDigitHereX")
	if err != ErrPasswordNoDigit {
		t.Fatalf("expected ErrPasswordNoDigit, got %v", err)
	}
}

// ---------- GenerateSecureKey tests ----------

func TestGenerateSecureKey(t *testing.T) {
	key := GenerateSecureKey(32)
	if len(key) != 64 { // 32 bytes → 64 hex chars
		t.Fatalf("expected 64 hex chars, got %d: %q", len(key), key)
	}
}

func TestGenerateSecureKeyUniqueness(t *testing.T) {
	keys := make(map[string]bool)
	for i := 0; i < 100; i++ {
		k := GenerateSecureKey(16)
		if keys[k] {
			t.Fatalf("duplicate key generated: %q", k)
		}
		keys[k] = true
	}
}

func TestGenerateSecureKeyZeroLength(t *testing.T) {
	key := GenerateSecureKey(0)
	if key != "" {
		t.Fatalf("expected empty string for zero-length key, got %q", key)
	}
}

// ---------- GeneratePasswordHash / ValidatePassword tests ----------

func TestGeneratePasswordHashAndValidate(t *testing.T) {
	password := "TestPassword1"
	hash, err := GeneratePasswordHash(password)
	if err != nil {
		fatalIfErr(t, err)
	}
	if hash == password {
		t.Fatal("hash should not equal plaintext password")
	}
	if err := ValidatePassword(password, hash); err != nil {
		t.Fatalf("ValidatePassword failed for correct password: %v", err)
	}
}

func TestValidatePasswordWrongPassword(t *testing.T) {
	hash, _ := GeneratePasswordHash("CorrectPass1")
	err := ValidatePassword("WrongPassword2", hash)
	if err == nil {
		t.Fatal("expected error for wrong password")
	}
}

// ---------- MFA TOTP encryption roundtrip tests ----------

func TestEncryptDecryptTOTPSecret(t *testing.T) {
	// Generate a valid 32-byte key
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	secret := "JBSWY3DPEHPK3PXP"

	encrypted, err := EncryptTOTPSecret(secret, key)
	if err != nil {
		t.Fatalf("EncryptTOTPSecret failed: %v", err)
	}
	if encrypted == secret {
		t.Fatal("encrypted should not equal plaintext")
	}

	decrypted, err := DecryptTOTPSecret(encrypted, key)
	if err != nil {
		t.Fatalf("DecryptTOTPSecret failed: %v", err)
	}
	if decrypted != secret {
		t.Fatalf("roundtrip failed: expected %q, got %q", secret, decrypted)
	}
}

func TestEncryptTOTPSecretWrongKeySize(t *testing.T) {
	shortKey := make([]byte, 16)
	_, err := EncryptTOTPSecret("secret", shortKey)
	if err == nil {
		t.Fatal("EncryptTOTPSecret should reject a 16-byte key")
	}
	if !strings.Contains(err.Error(), "32 bytes") {
		t.Fatalf("expected key-size error, got: %v", err)
	}
}

func TestDecryptTOTPSecretWrongKeySize(t *testing.T) {
	_, err := DecryptTOTPSecret("dGVzdA==", make([]byte, 16))
	if err == nil {
		t.Fatal("DecryptTOTPSecret should reject a 16-byte key")
	}
}

func TestDecryptTOTPSecretInvalidBase64(t *testing.T) {
	_, err := DecryptTOTPSecret("not-valid-base64!!!", make([]byte, 32))
	if err == nil {
		t.Fatal("expected error for invalid base64")
	}
}

func TestDecryptTOTPSecretTooShort(t *testing.T) {
	// Valid base64 but too short for nonce+ciphertext
	short := base64.StdEncoding.EncodeToString([]byte("ab"))
	_, err := DecryptTOTPSecret(short, make([]byte, 32))
	if err == nil {
		t.Fatal("expected error for ciphertext too short")
	}
}

func TestDecryptTOTPSecretWrongKey(t *testing.T) {
	key1 := make([]byte, 32)
	key2 := make([]byte, 32)
	key2[0] = 0xFF // different key

	encrypted, err := EncryptTOTPSecret("JBSWY3DPEHPK3PXP", key1)
	if err != nil {
		t.Fatalf("encrypt failed: %v", err)
	}
	_, err = DecryptTOTPSecret(encrypted, key2)
	if err == nil {
		t.Fatal("expected error when decrypting with wrong key")
	}
}

// ---------- TOTPEncryptionKeyFromBase64 tests ----------

func TestTOTPEncryptionKeyFromBase64Valid(t *testing.T) {
	rawKey := make([]byte, 32)
	for i := range rawKey {
		rawKey[i] = byte(i)
	}
	b64 := base64.StdEncoding.EncodeToString(rawKey)
	got, err := TOTPEncryptionKeyFromBase64(b64)
	if err != nil {
		fatalIfErr(t, err)
	}
	if len(got) != 32 {
		t.Fatalf("expected 32 bytes, got %d", len(got))
	}
}

func TestTOTPEncryptionKeyFromBase64Empty(t *testing.T) {
	_, err := TOTPEncryptionKeyFromBase64("")
	if err == nil {
		t.Fatal("expected error for empty key")
	}
}

func TestTOTPEncryptionKeyFromBase64WrongSize(t *testing.T) {
	b64 := base64.StdEncoding.EncodeToString(make([]byte, 16))
	_, err := TOTPEncryptionKeyFromBase64(b64)
	if err == nil {
		t.Fatal("TOTPEncryptionKeyFromBase64 should reject a 16-byte key")
	}
}

// ---------- Backup code tests ----------

func TestGenerateBackupCodes(t *testing.T) {
	n := 8
	plains, hashes, err := GenerateBackupCodes(n)
	if err != nil {
		fatalIfErr(t, err)
	}
	if len(plains) != n {
		t.Fatalf("expected %d plaintext codes, got %d", n, len(plains))
	}
	if len(hashes) != n {
		t.Fatalf("expected %d hashed codes, got %d", n, len(hashes))
	}
	// Each plaintext code should be 10 hex characters
	for i, code := range plains {
		if len(code) != 10 {
			t.Fatalf("code %d has length %d, expected 10", i, len(code))
		}
	}
	// Each hash should validate against its plaintext
	for i, code := range plains {
		if !ValidateBackupCode(code, hashes[i]) {
			t.Fatalf("backup code %d failed validation", i)
		}
	}
}

func TestValidateBackupCodeWrongCode(t *testing.T) {
	_, hashes, err := GenerateBackupCodes(1)
	if err != nil {
		fatalIfErr(t, err)
	}
	if ValidateBackupCode("AAAAAAAAAA", hashes[0]) {
		t.Fatal("expected wrong backup code to fail validation")
	}
}

func TestGenerateBackupCodesUniqueness(t *testing.T) {
	plains, _, err := GenerateBackupCodes(100)
	if err != nil {
		fatalIfErr(t, err)
	}
	seen := make(map[string]bool)
	for _, code := range plains {
		if seen[code] {
			t.Fatalf("duplicate backup code: %q", code)
		}
		seen[code] = true
	}
}

// ---------- Device fingerprint tests ----------

func TestRawDeviceFingerprint(t *testing.T) {
	fp := RawDeviceFingerprint("Mozilla/5.0", "en-US")
	if fp != "Mozilla/5.0|en-US" {
		t.Fatalf("unexpected fingerprint: %q", fp)
	}
}

func TestDeviceFingerprintHashAndValidate(t *testing.T) {
	raw := RawDeviceFingerprint("TestAgent", "en-GB")
	hash, err := DeviceFingerprintHash(raw)
	if err != nil {
		fatalIfErr(t, err)
	}
	if !ValidateDeviceFingerprint(raw, hash) {
		t.Fatal("expected fingerprint validation to succeed")
	}
	if ValidateDeviceFingerprint("different-agent|fr-FR", hash) {
		t.Fatal("expected different fingerprint to fail validation")
	}
}

// ---------- MFARequired tests ----------

func TestMFARequired(t *testing.T) {
	tests := []struct {
		role     string
		expected bool
	}{
		{"superadmin", true},
		{"org_admin", true},
		{"admin", false},
		{"user", false},
		{"learner", false},
		{"", false},
	}
	for _, tt := range tests {
		got := MFARequired(tt.role)
		if got != tt.expected {
			t.Errorf("MFARequired(%q) = %v, want %v", tt.role, got, tt.expected)
		}
	}
}

// ---------- GenerateTOTPSecret tests ----------

func TestGenerateTOTPSecret(t *testing.T) {
	secret, qrURI, err := GenerateTOTPSecret(testEmail)
	if err != nil {
		fatalIfErr(t, err)
	}
	if secret == "" {
		t.Fatal("expected non-empty secret")
	}
	if !strings.HasPrefix(qrURI, "data:image/png;base64,") {
		t.Fatalf("expected data URI, got: %q", qrURI[:50])
	}
}

// ---------- ExtractRoleSlug tests ----------

func TestExtractRoleSlugPriority(t *testing.T) {
	tests := []struct {
		roles    []string
		expected string
	}{
		{[]string{"superadmin", "org_admin", "learner"}, "superadmin"},
		{[]string{"org_admin", "learner"}, "org_admin"},
		{[]string{"campaign_manager", "learner"}, "campaign_manager"},
		{[]string{"trainer", "learner"}, "trainer"},
		{[]string{"auditor", "learner"}, "auditor"},
		{[]string{"learner"}, "learner"},
		{[]string{}, "learner"},               // empty → safe default
		{[]string{"unknown_role"}, "learner"}, // unknown → safe default
		{nil, "learner"},                      // nil → safe default
	}
	for _, tt := range tests {
		got := ExtractRoleSlug(tt.roles)
		if got != tt.expected {
			t.Errorf("ExtractRoleSlug(%v) = %q, want %q", tt.roles, got, tt.expected)
		}
	}
}

// ---------- stringClaim tests (via Exchange internals — test the helper) ----------

func TestStringClaimHelper(t *testing.T) {
	raw := map[string]interface{}{
		"email": testEmail,
		"name":  "Test User",
		"count": 42,
	}
	if got := stringClaim(raw, "email"); got != testEmail {
		t.Fatalf("expected email, got %q", got)
	}
	if got := stringClaim(raw, "name"); got != "Test User" {
		t.Fatalf("expected name, got %q", got)
	}
	// Non-string value should return empty
	if got := stringClaim(raw, "count"); got != "" {
		t.Fatalf("expected empty for non-string claim, got %q", got)
	}
	// Missing key should return empty
	if got := stringClaim(raw, "missing"); got != "" {
		t.Fatalf("expected empty for missing claim, got %q", got)
	}
}

// ---------- NewOIDCClient tests ----------

func TestNewOIDCClientDisabled(t *testing.T) {
	client, err := NewOIDCClient(testProviderURL, "client", "secret", testRedirectURL, false)
	if err != nil {
		fatalIfErr(t, err)
	}
	if client != nil {
		t.Fatal("expected nil client when OIDC is disabled")
	}
}

func TestNewOIDCClientMissingParams(t *testing.T) {
	// Missing provider URL
	_, err := NewOIDCClient("", "client", "secret", testRedirectURL, true)
	if err == nil {
		t.Fatal("expected error for empty provider URL")
	}
	// Missing client ID
	_, err = NewOIDCClient(testProviderURL, "", "secret", testRedirectURL, true)
	if err == nil {
		t.Fatal("expected error for empty client ID")
	}
}

// ---------- ValidateTOTP (with encrypted secret) ----------

func TestValidateTOTPInvalidEncryptedSecret(t *testing.T) {
	// ValidateTOTP with bad encrypted data should return false, not panic
	key := make([]byte, 32)
	result := ValidateTOTP("not-valid-encrypted-data", "123456", key)
	if result {
		t.Fatal("expected ValidateTOTP to return false for invalid encrypted secret")
	}
}

// ---------- EncryptTOTPSecret unique ciphertexts ----------

func TestEncryptTOTPSecretUniqueNonces(t *testing.T) {
	key := make([]byte, 32)
	secret := "JBSWY3DPEHPK3PXP"
	ciphertexts := make(map[string]bool)
	for i := 0; i < 20; i++ {
		ct, err := EncryptTOTPSecret(secret, key)
		if err != nil {
			fatalIfErr(t, err)
		}
		if ciphertexts[ct] {
			t.Fatalf("duplicate ciphertext on iteration %d — nonce reuse", i)
		}
		ciphertexts[ct] = true
	}
}

// ---------- OIDCClient LogoutURL ----------

func TestOIDCClientLogoutURL(t *testing.T) {
	// We can't construct a full OIDCClient without a live OIDC provider,
	// but we can test the LogoutURL method on a manually constructed struct.
	c := &OIDCClient{providerURL: testProviderURL}
	expected := testProviderURL + "/protocol/openid-connect/logout"
	if got := c.LogoutURL(); got != expected {
		t.Fatalf("expected %q, got %q", expected, got)
	}
}

// ---------- Password hash determinism ----------

func TestGeneratePasswordHashUnique(t *testing.T) {
	password := "SamePassword1"
	h1, _ := GeneratePasswordHash(password)
	h2, _ := GeneratePasswordHash(password)
	if h1 == h2 {
		t.Fatal("expected different bcrypt hashes for the same password (different salts)")
	}
}
