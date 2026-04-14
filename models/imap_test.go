package models

import (
	"testing"

	"github.com/gophish/gophish/config"
)

func setupIMAPTest(t *testing.T) func() {
	t.Helper()
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	if err := Setup(conf); err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	db.Exec("DELETE FROM imap")
	return func() { db.Exec("DELETE FROM imap") }
}

// validIMAP returns an IMAP struct with all required fields populated.
// Host is set to 127.0.0.1 which always passes net.ParseIP.
func validIMAP(uid int64) *IMAP {
	return &IMAP{
		UserId:   uid,
		Host:     "127.0.0.1",
		Port:     993,
		Username: "user@example.com",
		Password: "password123",
		TLS:      true,
		IMAPFreq: 60,
	}
}

// ---------- DefaultIMAPFolder / DefaultIMAPFreq ----------

func TestDefaultIMAPFolder(t *testing.T) {
	if DefaultIMAPFolder != "INBOX" {
		t.Fatalf("expected DefaultIMAPFolder to be 'INBOX', got %q", DefaultIMAPFolder)
	}
}

func TestDefaultIMAPFreq(t *testing.T) {
	if DefaultIMAPFreq != 60 {
		t.Fatalf("expected DefaultIMAPFreq to be 60, got %d", DefaultIMAPFreq)
	}
}

// ---------- Validate ----------

func TestIMAPValidateSuccess(t *testing.T) {
	im := validIMAP(1)
	if err := im.Validate(); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

func TestIMAPValidateHostNotSpecified(t *testing.T) {
	im := validIMAP(1)
	im.Host = ""
	err := im.Validate()
	if err != ErrIMAPHostNotSpecified {
		t.Fatalf("expected ErrIMAPHostNotSpecified, got %v", err)
	}
}

func TestIMAPValidatePortNotSpecified(t *testing.T) {
	im := validIMAP(1)
	im.Port = 0
	err := im.Validate()
	if err != ErrIMAPPortNotSpecified {
		t.Fatalf("expected ErrIMAPPortNotSpecified, got %v", err)
	}
}

func TestIMAPValidateUsernameNotSpecified(t *testing.T) {
	im := validIMAP(1)
	im.Username = ""
	err := im.Validate()
	if err != ErrIMAPUsernameNotSpecified {
		t.Fatalf("expected ErrIMAPUsernameNotSpecified, got %v", err)
	}
}

func TestIMAPValidatePasswordNotSpecified(t *testing.T) {
	im := validIMAP(1)
	im.Password = ""
	err := im.Validate()
	if err != ErrIMAPPasswordNotSpecified {
		t.Fatalf("expected ErrIMAPPasswordNotSpecified, got %v", err)
	}
}

func TestIMAPValidateInvalidHost(t *testing.T) {
	im := validIMAP(1)
	// Use a hostname guaranteed to fail both net.ParseIP and net.LookupHost
	im.Host = "this-host-definitely-does-not-exist-zzzzz.invalid"
	err := im.Validate()
	if err != ErrInvalidIMAPHost {
		t.Fatalf("expected ErrInvalidIMAPHost, got %v", err)
	}
}

func TestIMAPValidateInvalidFreqSetsDefault(t *testing.T) {
	im := validIMAP(1)
	im.IMAPFreq = 0
	// Freq < 30 triggers the default reset inside Validate
	err := im.Validate()
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if im.IMAPFreq != DefaultIMAPFreq {
		t.Fatalf("expected IMAPFreq to be reset to %d, got %d", DefaultIMAPFreq, im.IMAPFreq)
	}
}

func TestIMAPValidateDefaultFolder(t *testing.T) {
	im := validIMAP(1)
	im.Folder = ""
	err := im.Validate()
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if im.Folder != DefaultIMAPFolder {
		t.Fatalf("expected Folder to be set to %q, got %q", DefaultIMAPFolder, im.Folder)
	}
}

// ---------- PostIMAP ----------

func TestPostIMAPSuccess(t *testing.T) {
	teardown := setupIMAPTest(t)
	defer teardown()

	im := validIMAP(1)
	if err := PostIMAP(im, 1); err != nil {
		t.Fatalf("PostIMAP failed: %v", err)
	}

	results, err := GetIMAP(1)
	if err != nil {
		t.Fatalf("GetIMAP failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 IMAP record, got %d", len(results))
	}
	if results[0].Host != "127.0.0.1" {
		t.Fatalf("expected host '127.0.0.1', got %q", results[0].Host)
	}
}

func TestPostIMAPValidationError(t *testing.T) {
	teardown := setupIMAPTest(t)
	defer teardown()

	im := validIMAP(1)
	im.Host = ""
	err := PostIMAP(im, 1)
	if err != ErrIMAPHostNotSpecified {
		t.Fatalf("expected ErrIMAPHostNotSpecified, got %v", err)
	}
}

// ---------- GetIMAP ----------

func TestGetIMAPFound(t *testing.T) {
	teardown := setupIMAPTest(t)
	defer teardown()

	im := validIMAP(42)
	PostIMAP(im, 42)

	results, err := GetIMAP(42)
	if err != nil {
		t.Fatalf("GetIMAP failed: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 IMAP record, got %d", len(results))
	}
	if results[0].Username != "user@example.com" {
		t.Fatalf("expected username 'user@example.com', got %q", results[0].Username)
	}
}

func TestGetIMAPEmpty(t *testing.T) {
	teardown := setupIMAPTest(t)
	defer teardown()

	results, err := GetIMAP(999)
	if err != nil {
		t.Fatalf("GetIMAP failed: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 IMAP records for non-existent user, got %d", len(results))
	}
}

// ---------- DeleteIMAP ----------

func TestDeleteIMAP(t *testing.T) {
	teardown := setupIMAPTest(t)
	defer teardown()

	im := validIMAP(1)
	PostIMAP(im, 1)

	if err := DeleteIMAP(1); err != nil {
		t.Fatalf("DeleteIMAP failed: %v", err)
	}

	results, err := GetIMAP(1)
	if err != nil {
		t.Fatalf("GetIMAP after delete failed: %v", err)
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 IMAP records after delete, got %d", len(results))
	}
}

// ---------- SuccessfulLogin ----------

func TestSuccessfulLogin(t *testing.T) {
	teardown := setupIMAPTest(t)
	defer teardown()

	im := validIMAP(1)
	PostIMAP(im, 1)

	if err := SuccessfulLogin(im); err != nil {
		t.Fatalf("SuccessfulLogin failed: %v", err)
	}

	results, _ := GetIMAP(1)
	if len(results) != 1 {
		t.Fatalf("expected 1 IMAP record, got %d", len(results))
	}
	if results[0].LastLogin.IsZero() {
		t.Fatalf("expected LastLogin to be set after SuccessfulLogin")
	}
}
