package models

import (
	"testing"

	"github.com/gophish/gophish/config"
)

func setupWebhookTest(t *testing.T) func() {
	t.Helper()
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	if err := Setup(conf); err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	db.Exec("DELETE FROM webhooks")
	return func() { db.Exec("DELETE FROM webhooks") }
}

// ---------- Validate ----------

func TestWebhookValidateEmptyURL(t *testing.T) {
	wh := Webhook{Name: "test", URL: ""}
	err := wh.Validate()
	if err != ErrURLNotSpecified {
		t.Fatalf("expected ErrURLNotSpecified, got %v", err)
	}
}

func TestWebhookValidateEmptyName(t *testing.T) {
	wh := Webhook{Name: "", URL: "https://example.com"}
	err := wh.Validate()
	if err != ErrNameNotSpecified {
		t.Fatalf("expected ErrNameNotSpecified, got %v", err)
	}
}

func TestWebhookValidateSuccess(t *testing.T) {
	wh := Webhook{Name: "test", URL: "https://example.com"}
	err := wh.Validate()
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
}

// ---------- PostWebhook ----------

func TestPostWebhookSuccess(t *testing.T) {
	teardown := setupWebhookTest(t)
	defer teardown()

	wh := &Webhook{Name: "Hook1", URL: "https://example.com/hook", Secret: "s3cr3t", IsActive: true}
	if err := PostWebhook(wh); err != nil {
		t.Fatalf("PostWebhook failed: %v", err)
	}
	if wh.Id == 0 {
		t.Fatalf("expected non-zero ID after PostWebhook")
	}
}

func TestPostWebhookErrURLNotSpecified(t *testing.T) {
	teardown := setupWebhookTest(t)
	defer teardown()

	wh := &Webhook{Name: "Hook1", URL: ""}
	err := PostWebhook(wh)
	if err != ErrURLNotSpecified {
		t.Fatalf("expected ErrURLNotSpecified, got %v", err)
	}
}

func TestPostWebhookErrNameNotSpecified(t *testing.T) {
	teardown := setupWebhookTest(t)
	defer teardown()

	wh := &Webhook{Name: "", URL: "https://example.com/hook"}
	err := PostWebhook(wh)
	if err != ErrNameNotSpecified {
		t.Fatalf("expected ErrNameNotSpecified, got %v", err)
	}
}

// ---------- GetWebhooks ----------

func TestGetWebhooksEmpty(t *testing.T) {
	teardown := setupWebhookTest(t)
	defer teardown()

	whs, err := GetWebhooks()
	if err != nil {
		t.Fatalf("GetWebhooks failed: %v", err)
	}
	if len(whs) != 0 {
		t.Fatalf("expected 0 webhooks, got %d", len(whs))
	}
}

func TestGetWebhooksMultiple(t *testing.T) {
	teardown := setupWebhookTest(t)
	defer teardown()

	PostWebhook(&Webhook{Name: "A", URL: "https://a.com"})
	PostWebhook(&Webhook{Name: "B", URL: "https://b.com"})

	whs, err := GetWebhooks()
	if err != nil {
		t.Fatalf("GetWebhooks failed: %v", err)
	}
	if len(whs) != 2 {
		t.Fatalf("expected 2 webhooks, got %d", len(whs))
	}
}

// ---------- GetWebhook ----------

func TestGetWebhookFound(t *testing.T) {
	teardown := setupWebhookTest(t)
	defer teardown()

	wh := &Webhook{Name: "Found", URL: "https://found.com"}
	PostWebhook(wh)

	fetched, err := GetWebhook(wh.Id)
	if err != nil {
		t.Fatalf("GetWebhook failed: %v", err)
	}
	if fetched.Name != "Found" {
		t.Fatalf("expected name 'Found', got %q", fetched.Name)
	}
	if fetched.URL != "https://found.com" {
		t.Fatalf("expected URL 'https://found.com', got %q", fetched.URL)
	}
}

func TestGetWebhookNotFound(t *testing.T) {
	teardown := setupWebhookTest(t)
	defer teardown()

	_, err := GetWebhook(99999)
	if err == nil {
		t.Fatalf("expected error for non-existent webhook, got nil")
	}
}

// ---------- GetActiveWebhooks ----------

func TestGetActiveWebhooksFiltersInactive(t *testing.T) {
	teardown := setupWebhookTest(t)
	defer teardown()

	PostWebhook(&Webhook{Name: "Active1", URL: "https://a.com", IsActive: true})
	PostWebhook(&Webhook{Name: "Inactive", URL: "https://b.com", IsActive: false})
	PostWebhook(&Webhook{Name: "Active2", URL: "https://c.com", IsActive: true})

	whs, err := GetActiveWebhooks()
	if err != nil {
		t.Fatalf("GetActiveWebhooks failed: %v", err)
	}
	if len(whs) != 2 {
		t.Fatalf("expected 2 active webhooks, got %d", len(whs))
	}
	for _, w := range whs {
		if !w.IsActive {
			t.Fatalf("expected only active webhooks, got inactive: %q", w.Name)
		}
	}
}

func TestGetActiveWebhooksNoneActive(t *testing.T) {
	teardown := setupWebhookTest(t)
	defer teardown()

	PostWebhook(&Webhook{Name: "Inactive", URL: "https://x.com", IsActive: false})

	whs, err := GetActiveWebhooks()
	if err != nil {
		t.Fatalf("GetActiveWebhooks failed: %v", err)
	}
	if len(whs) != 0 {
		t.Fatalf("expected 0 active webhooks, got %d", len(whs))
	}
}

// ---------- PutWebhook ----------

func TestPutWebhookUpdateNameAndURL(t *testing.T) {
	teardown := setupWebhookTest(t)
	defer teardown()

	wh := &Webhook{Name: "Original", URL: "https://original.com"}
	PostWebhook(wh)

	wh.Name = "Updated"
	wh.URL = "https://updated.com"
	if err := PutWebhook(wh); err != nil {
		t.Fatalf("PutWebhook failed: %v", err)
	}

	fetched, err := GetWebhook(wh.Id)
	if err != nil {
		t.Fatalf("GetWebhook after PutWebhook failed: %v", err)
	}
	if fetched.Name != "Updated" {
		t.Fatalf("expected name 'Updated', got %q", fetched.Name)
	}
	if fetched.URL != "https://updated.com" {
		t.Fatalf("expected URL 'https://updated.com', got %q", fetched.URL)
	}
}

func TestPutWebhookValidationError(t *testing.T) {
	teardown := setupWebhookTest(t)
	defer teardown()

	wh := &Webhook{Name: "Original", URL: "https://original.com"}
	PostWebhook(wh)

	wh.Name = ""
	err := PutWebhook(wh)
	if err != ErrNameNotSpecified {
		t.Fatalf("expected ErrNameNotSpecified, got %v", err)
	}
}

// ---------- DeleteWebhook ----------

func TestDeleteWebhook(t *testing.T) {
	teardown := setupWebhookTest(t)
	defer teardown()

	wh := &Webhook{Name: "ToDelete", URL: "https://delete.com"}
	PostWebhook(wh)

	if err := DeleteWebhook(wh.Id); err != nil {
		t.Fatalf("DeleteWebhook failed: %v", err)
	}

	_, err := GetWebhook(wh.Id)
	if err == nil {
		t.Fatalf("expected error after deleting webhook, got nil")
	}
}

func TestDeleteWebhookVerifyGone(t *testing.T) {
	teardown := setupWebhookTest(t)
	defer teardown()

	wh := &Webhook{Name: "Gone", URL: "https://gone.com"}
	PostWebhook(wh)
	DeleteWebhook(wh.Id)

	whs, err := GetWebhooks()
	if err != nil {
		t.Fatalf("GetWebhooks failed: %v", err)
	}
	if len(whs) != 0 {
		t.Fatalf("expected 0 webhooks after delete, got %d", len(whs))
	}
}
