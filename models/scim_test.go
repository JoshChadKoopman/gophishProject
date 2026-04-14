package models

import (
	"strings"
	"testing"

	"github.com/gophish/gophish/config"
)

func setupSCIMTest(t *testing.T) func() {
	t.Helper()
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	if err := Setup(conf); err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	return func() { /* no teardown needed for in-memory DB */ }
}

// ---------- Token lifecycle ----------

func TestCreateSCIMToken(t *testing.T) {
	teardown := setupSCIMTest(t)
	defer teardown()

	raw, tok, err := CreateSCIMToken(1, 1, "test token")
	if err != nil {
		t.Fatalf("CreateSCIMToken: %v", err)
	}
	if !strings.HasPrefix(raw, "scim_") {
		t.Fatalf("expected raw token to start with 'scim_', got %q", raw[:10])
	}
	if tok.Id == 0 {
		t.Fatal("expected non-zero token ID")
	}
	if tok.OrgId != 1 {
		t.Fatalf("expected OrgId 1, got %d", tok.OrgId)
	}
	if !tok.IsActive {
		t.Fatal("expected token to be active")
	}
	if tok.Description != "test token" {
		t.Fatalf("expected description 'test token', got %q", tok.Description)
	}
}

func TestGetSCIMTokens(t *testing.T) {
	teardown := setupSCIMTest(t)
	defer teardown()

	CreateSCIMToken(1, 1, "first")
	CreateSCIMToken(1, 1, "second")
	CreateSCIMToken(2, 2, "other org")

	tokens, err := GetSCIMTokens(1)
	if err != nil {
		t.Fatalf("GetSCIMTokens: %v", err)
	}
	if len(tokens) != 2 {
		t.Fatalf("expected 2 tokens for org 1, got %d", len(tokens))
	}

	tokens2, _ := GetSCIMTokens(2)
	if len(tokens2) != 1 {
		t.Fatalf("expected 1 token for org 2, got %d", len(tokens2))
	}
}

func TestValidateSCIMToken(t *testing.T) {
	teardown := setupSCIMTest(t)
	defer teardown()

	raw, _, err := CreateSCIMToken(1, 1, "validate me")
	if err != nil {
		t.Fatalf("CreateSCIMToken: %v", err)
	}

	orgId, err := ValidateSCIMToken(raw)
	if err != nil {
		t.Fatalf("ValidateSCIMToken: %v", err)
	}
	if orgId != 1 {
		t.Fatalf("expected orgId 1, got %d", orgId)
	}
}

func TestValidateSCIMTokenInvalid(t *testing.T) {
	teardown := setupSCIMTest(t)
	defer teardown()

	_, err := ValidateSCIMToken("scim_bogus_token_value")
	if err != ErrSCIMTokenNotFound {
		t.Fatalf("expected ErrSCIMTokenNotFound, got %v", err)
	}
}

func TestDeleteSCIMToken(t *testing.T) {
	teardown := setupSCIMTest(t)
	defer teardown()

	raw, tok, _ := CreateSCIMToken(1, 1, "deletable")

	if err := DeleteSCIMToken(tok.Id, 1); err != nil {
		t.Fatalf("DeleteSCIMToken: %v", err)
	}

	// Token should no longer validate
	_, err := ValidateSCIMToken(raw)
	if err != ErrSCIMTokenNotFound {
		t.Fatalf("expected ErrSCIMTokenNotFound after delete, got %v", err)
	}
}

func TestDeleteSCIMTokenWrongOrg(t *testing.T) {
	teardown := setupSCIMTest(t)
	defer teardown()

	raw, tok, _ := CreateSCIMToken(1, 1, "org1 token")

	// Try deleting with wrong org — should not affect the token
	DeleteSCIMToken(tok.Id, 999)

	orgId, err := ValidateSCIMToken(raw)
	if err != nil {
		t.Fatalf("token should still be valid after wrong-org delete: %v", err)
	}
	if orgId != 1 {
		t.Fatalf("expected orgId 1, got %d", orgId)
	}
}

// ---------- External ID mapping ----------

func TestSetAndGetSCIMExternalID(t *testing.T) {
	teardown := setupSCIMTest(t)
	defer teardown()

	err := SetSCIMExternalID(1, "User", "ext-abc-123", 42)
	if err != nil {
		t.Fatalf("SetSCIMExternalID: %v", err)
	}

	extId := GetSCIMExternalID(1, "User", 42)
	if extId != "ext-abc-123" {
		t.Fatalf("expected 'ext-abc-123', got %q", extId)
	}
}

func TestSetSCIMExternalIDUpdate(t *testing.T) {
	teardown := setupSCIMTest(t)
	defer teardown()

	SetSCIMExternalID(1, "User", "old-ext-id", 42)
	SetSCIMExternalID(1, "User", "new-ext-id", 42)

	extId := GetSCIMExternalID(1, "User", 42)
	if extId != "new-ext-id" {
		t.Fatalf("expected updated 'new-ext-id', got %q", extId)
	}
}

func TestGetSCIMExternalIDNotFound(t *testing.T) {
	teardown := setupSCIMTest(t)
	defer teardown()

	extId := GetSCIMExternalID(1, "User", 99999)
	if extId != "" {
		t.Fatalf("expected empty string for non-existent mapping, got %q", extId)
	}
}

func TestGetInternalIDByExternalID(t *testing.T) {
	teardown := setupSCIMTest(t)
	defer teardown()

	SetSCIMExternalID(1, "Group", "ext-grp-1", 100)

	internalId, err := GetInternalIDByExternalID(1, "Group", "ext-grp-1")
	if err != nil {
		t.Fatalf("GetInternalIDByExternalID: %v", err)
	}
	if internalId != 100 {
		t.Fatalf("expected internal ID 100, got %d", internalId)
	}
}

func TestGetInternalIDByExternalIDNotFound(t *testing.T) {
	teardown := setupSCIMTest(t)
	defer teardown()

	_, err := GetInternalIDByExternalID(1, "User", "nonexistent")
	if err == nil {
		t.Fatal("expected error for non-existent external ID")
	}
}

func TestDeleteSCIMExternalID(t *testing.T) {
	teardown := setupSCIMTest(t)
	defer teardown()

	SetSCIMExternalID(1, "User", "ext-to-delete", 50)
	DeleteSCIMExternalID(1, "User", 50)

	extId := GetSCIMExternalID(1, "User", 50)
	if extId != "" {
		t.Fatalf("expected empty string after delete, got %q", extId)
	}
}

// ---------- Hash function ----------

func TestHashSCIMTokenDeterministic(t *testing.T) {
	h1 := hashSCIMToken("test-token-value")
	h2 := hashSCIMToken("test-token-value")
	if h1 != h2 {
		t.Fatal("same input should produce same hash")
	}

	h3 := hashSCIMToken("different-token")
	if h1 == h3 {
		t.Fatal("different inputs should produce different hashes")
	}
}

const testSCIMBaseURL = "https://example.com"

// ---------- SCIM resource conversion ----------

func TestUserToSCIMResource(t *testing.T) {
	u := User{
		Id:        42,
		Username:  "jdoe",
		FirstName: "John",
		LastName:  "Doe",
		Email:     "jdoe@example.com",
		JobTitle:  "Engineer",
	}

	res := UserToSCIMResource(u, 1, testSCIMBaseURL)
	if res.Id != "42" {
		t.Fatalf("expected Id '42', got %q", res.Id)
	}
	if res.UserName != "jdoe" {
		t.Fatalf("expected UserName 'jdoe', got %q", res.UserName)
	}
	if res.Name.GivenName != "John" {
		t.Fatalf("expected GivenName 'John', got %q", res.Name.GivenName)
	}
	if !res.Active {
		t.Fatal("expected Active to be true for non-locked user")
	}
	if res.Meta.ResourceType != "User" {
		t.Fatalf("expected ResourceType 'User', got %q", res.Meta.ResourceType)
	}
	if res.Meta.Location != "https://example.com/scim/v2/Users/42" {
		t.Fatalf("unexpected Location: %q", res.Meta.Location)
	}
	if len(res.Emails) != 1 || res.Emails[0].Value != "jdoe@example.com" {
		t.Fatalf("expected email jdoe@example.com, got %+v", res.Emails)
	}
}

func TestUserToSCIMResourceWithDepartment(t *testing.T) {
	u := User{
		Id:         1,
		Username:   "test",
		Department: "Engineering",
	}

	res := UserToSCIMResource(u, 1, testSCIMBaseURL)
	if res.Enterprise == nil {
		t.Fatal("expected enterprise extension for user with department")
	}
	if res.Enterprise.Department != "Engineering" {
		t.Fatalf("expected department 'Engineering', got %q", res.Enterprise.Department)
	}
	// Should include enterprise schema
	found := false
	for _, s := range res.Schemas {
		if s == SCIMSchemaEnterpriseUser {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected enterprise user schema in schemas list")
	}
}

func TestUserToSCIMResourceLockedUser(t *testing.T) {
	u := User{
		Id:            1,
		Username:      "locked",
		AccountLocked: true,
	}

	res := UserToSCIMResource(u, 1, testSCIMBaseURL)
	if res.Active {
		t.Fatal("expected Active to be false for locked user")
	}
}

// ---------- Schema constants ----------

func TestSCIMSchemaConstants(t *testing.T) {
	constants := map[string]string{
		"User":         SCIMSchemaUser,
		"Enterprise":   SCIMSchemaEnterpriseUser,
		"Group":        SCIMSchemaGroup,
		"ListResponse": SCIMSchemaListResponse,
		"Error":        SCIMSchemaError,
	}
	for name, val := range constants {
		if val == "" {
			t.Fatalf("SCIM schema constant %s should not be empty", name)
		}
		if !strings.HasPrefix(val, "urn:ietf:params:scim:") {
			t.Fatalf("SCIM schema constant %s should start with urn:ietf:params:scim:, got %q", name, val)
		}
	}
}
