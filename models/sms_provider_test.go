package models

import (
	"testing"

	"github.com/gophish/gophish/config"
)

func setupSMSTest(t *testing.T) func() {
	t.Helper()
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	if err := Setup(conf); err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	db.Exec("DELETE FROM sms_providers")
	return func() { db.Exec("DELETE FROM sms_providers") }
}

const (
	testSMSName  = "Test Twilio"
	testSMSPhone = "+15551234567"
)

func validSMSProvider(userId, orgId int64) *SMSProvider {
	return &SMSProvider{
		Name:         testSMSName,
		ProviderType: "twilio",
		AccountSid:   "AC1234567890",
		AuthToken:    "secret-token-abc",
		FromNumber:   testSMSPhone,
		UserId:       userId,
		OrgId:        orgId,
	}
}

func smsScope(userId, orgId int64) OrgScope {
	return OrgScope{UserId: userId, OrgId: orgId}
}

// ---------- Validate ----------

func TestSMSProviderValidateSuccess(t *testing.T) {
	sp := validSMSProvider(1, 1)
	if err := sp.Validate(); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestSMSProviderValidateNameMissing(t *testing.T) {
	sp := validSMSProvider(1, 1)
	sp.Name = ""
	if err := sp.Validate(); err != ErrSMSProviderNameNotSpecified {
		t.Fatalf("expected ErrSMSProviderNameNotSpecified, got %v", err)
	}
}

func TestSMSProviderValidateAccountSidMissing(t *testing.T) {
	sp := validSMSProvider(1, 1)
	sp.AccountSid = ""
	if err := sp.Validate(); err != ErrSMSAccountSidNotSpecified {
		t.Fatalf("expected ErrSMSAccountSidNotSpecified, got %v", err)
	}
}

func TestSMSProviderValidateAuthTokenMissing(t *testing.T) {
	sp := validSMSProvider(1, 1)
	sp.AuthToken = ""
	if err := sp.Validate(); err != ErrSMSAuthTokenNotSpecified {
		t.Fatalf("expected ErrSMSAuthTokenNotSpecified, got %v", err)
	}
}

func TestSMSProviderValidateFromNumberMissing(t *testing.T) {
	sp := validSMSProvider(1, 1)
	sp.FromNumber = ""
	if err := sp.Validate(); err != ErrSMSFromNumberNotSpecified {
		t.Fatalf("expected ErrSMSFromNumberNotSpecified, got %v", err)
	}
}

// ---------- PostSMSProvider ----------

func TestPostSMSProviderSuccess(t *testing.T) {
	teardown := setupSMSTest(t)
	defer teardown()

	sp := validSMSProvider(1, 1)
	scope := smsScope(1, 1)

	if err := PostSMSProvider(sp, scope); err != nil {
		t.Fatalf("PostSMSProvider: %v", err)
	}
	if sp.Id == 0 {
		t.Fatal("expected non-zero ID after save")
	}
	if sp.ModifiedDate.IsZero() {
		t.Fatal("expected ModifiedDate to be set")
	}
}

func TestPostSMSProviderValidationError(t *testing.T) {
	teardown := setupSMSTest(t)
	defer teardown()

	sp := validSMSProvider(1, 1)
	sp.Name = ""
	scope := smsScope(1, 1)

	err := PostSMSProvider(sp, scope)
	if err != ErrSMSProviderNameNotSpecified {
		t.Fatalf("expected ErrSMSProviderNameNotSpecified, got %v", err)
	}
}

// ---------- GetSMSProviders ----------

func TestGetSMSProviders(t *testing.T) {
	teardown := setupSMSTest(t)
	defer teardown()

	scope := smsScope(1, 1)
	PostSMSProvider(validSMSProvider(1, 1), scope)

	sp2 := validSMSProvider(1, 1)
	sp2.Name = "Second Provider"
	PostSMSProvider(sp2, scope)

	providers, err := GetSMSProviders(scope)
	if err != nil {
		t.Fatalf("GetSMSProviders: %v", err)
	}
	if len(providers) != 2 {
		t.Fatalf("expected 2 providers, got %d", len(providers))
	}
}

func TestGetSMSProvidersOrgIsolation(t *testing.T) {
	teardown := setupSMSTest(t)
	defer teardown()

	scope1 := smsScope(1, 1)
	scope2 := smsScope(2, 2)

	PostSMSProvider(validSMSProvider(1, 1), scope1)
	PostSMSProvider(validSMSProvider(2, 2), scope2)

	providers, _ := GetSMSProviders(scope1)
	if len(providers) != 1 {
		t.Fatalf("expected 1 provider for scope1, got %d", len(providers))
	}

	providers2, _ := GetSMSProviders(scope2)
	if len(providers2) != 1 {
		t.Fatalf("expected 1 provider for scope2, got %d", len(providers2))
	}
}

// ---------- GetSMSProvider ----------

func TestGetSMSProvider(t *testing.T) {
	teardown := setupSMSTest(t)
	defer teardown()

	sp := validSMSProvider(1, 1)
	scope := smsScope(1, 1)
	PostSMSProvider(sp, scope)

	found, err := GetSMSProvider(sp.Id, scope)
	if err != nil {
		t.Fatalf("GetSMSProvider: %v", err)
	}
	if found.Name != testSMSName {
		t.Fatalf("expected name %q, got %q", testSMSName, found.Name)
	}
}

func TestGetSMSProviderNotFound(t *testing.T) {
	teardown := setupSMSTest(t)
	defer teardown()

	_, err := GetSMSProvider(99999, smsScope(1, 1))
	if err == nil {
		t.Fatal("expected error for non-existent provider")
	}
}

// ---------- GetSMSProviderByName ----------

func TestGetSMSProviderByName(t *testing.T) {
	teardown := setupSMSTest(t)
	defer teardown()

	scope := smsScope(1, 1)
	PostSMSProvider(validSMSProvider(1, 1), scope)

	found, err := GetSMSProviderByName(testSMSName, scope)
	if err != nil {
		t.Fatalf("GetSMSProviderByName: %v", err)
	}
	if found.AccountSid != "AC1234567890" {
		t.Fatalf("expected AccountSid 'AC1234567890', got %q", found.AccountSid)
	}
}

func TestGetSMSProviderByNameNotFound(t *testing.T) {
	teardown := setupSMSTest(t)
	defer teardown()

	_, err := GetSMSProviderByName("Nonexistent", smsScope(1, 1))
	if err == nil {
		t.Fatal("expected error for non-existent provider name")
	}
}

// ---------- PutSMSProvider ----------

func TestPutSMSProvider(t *testing.T) {
	teardown := setupSMSTest(t)
	defer teardown()

	sp := validSMSProvider(1, 1)
	scope := smsScope(1, 1)
	PostSMSProvider(sp, scope)

	sp.Name = "Updated Name"
	if err := PutSMSProvider(sp, scope); err != nil {
		t.Fatalf("PutSMSProvider: %v", err)
	}

	found, _ := GetSMSProvider(sp.Id, scope)
	if found.Name != "Updated Name" {
		t.Fatalf("expected name 'Updated Name', got %q", found.Name)
	}
}

func TestPutSMSProviderBlankAuthTokenRejectsValidation(t *testing.T) {
	teardown := setupSMSTest(t)
	defer teardown()

	sp := validSMSProvider(1, 1)
	scope := smsScope(1, 1)
	PostSMSProvider(sp, scope)

	// Blank auth token is caught by Validate() before the keep-existing logic
	sp.AuthToken = ""
	sp.Name = "Updated"
	err := PutSMSProvider(sp, scope)
	if err != ErrSMSAuthTokenNotSpecified {
		t.Fatalf("expected ErrSMSAuthTokenNotSpecified, got %v", err)
	}
}

// ---------- DeleteSMSProvider ----------

func TestDeleteSMSProvider(t *testing.T) {
	teardown := setupSMSTest(t)
	defer teardown()

	sp := validSMSProvider(1, 1)
	scope := smsScope(1, 1)
	PostSMSProvider(sp, scope)

	if err := DeleteSMSProvider(sp.Id, scope); err != nil {
		t.Fatalf("DeleteSMSProvider: %v", err)
	}

	_, err := GetSMSProvider(sp.Id, scope)
	if err == nil {
		t.Fatal("expected error after delete")
	}
}

func TestDeleteSMSProviderWrongScope(t *testing.T) {
	teardown := setupSMSTest(t)
	defer teardown()

	sp := validSMSProvider(1, 1)
	PostSMSProvider(sp, smsScope(1, 1))

	// Deleting with a different user scope should fail
	err := DeleteSMSProvider(sp.Id, smsScope(999, 999))
	if err == nil {
		t.Fatal("expected error when deleting with wrong scope")
	}

	// Original should still exist
	_, err = GetSMSProvider(sp.Id, smsScope(1, 1))
	if err != nil {
		t.Fatalf("provider should still exist: %v", err)
	}
}

// ---------- ValidatePhone ----------

func TestValidatePhone(t *testing.T) {
	cases := []struct {
		phone string
		valid bool
	}{
		{testSMSPhone, true},
		{"15551234567", true},
		{"+44 20 7946 0958", true},
		{"+1 (555) 123-4567", true},
		{"", false},
		{"123", false},
		{"abc", false},
		{"+0123456789", false}, // starts with 0
	}
	for _, tc := range cases {
		got := ValidatePhone(tc.phone)
		if got != tc.valid {
			t.Errorf("ValidatePhone(%q) = %v, want %v", tc.phone, got, tc.valid)
		}
	}
}

// ---------- CleanPhone ----------

func TestCleanPhone(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{testSMSPhone, testSMSPhone},
		{"15551234567", testSMSPhone},
		{"+1 (555) 123-4567", testSMSPhone},
		{"  1 555 123 4567  ", testSMSPhone},
	}
	for _, tc := range cases {
		got := CleanPhone(tc.input)
		if got != tc.expected {
			t.Errorf("CleanPhone(%q) = %q, want %q", tc.input, got, tc.expected)
		}
	}
}

// ---------- TableName ----------

func TestSMSProviderTableName(t *testing.T) {
	sp := SMSProvider{}
	if sp.TableName() != "sms_providers" {
		t.Fatalf("expected table name 'sms_providers', got %q", sp.TableName())
	}
}
