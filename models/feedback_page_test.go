package models

import (
	"testing"

	"github.com/gophish/gophish/config"
)

func setupFeedbackPageTest(t *testing.T) func() {
	t.Helper()
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	if err := Setup(conf); err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	db.Exec("DELETE FROM feedback_pages")
	return func() {
		db.Exec("DELETE FROM feedback_pages")
	}
}

func TestFeedbackPageValidation(t *testing.T) {
	fp := &FeedbackPage{}
	if err := fp.Validate(); err != ErrFeedbackPageNameNotSpecified {
		t.Fatalf("expected ErrFeedbackPageNameNotSpecified, got %v", err)
	}
	fp.Name = "Test Page"
	if err := fp.Validate(); err != ErrFeedbackPageContentNotSpecified {
		t.Fatalf("expected ErrFeedbackPageContentNotSpecified, got %v", err)
	}
	fp.HTML = "<html><body>Test</body></html>"
	if err := fp.Validate(); err != nil {
		t.Fatalf("valid feedback page should pass validation: %v", err)
	}
}

func TestFeedbackPageValidationDefaults(t *testing.T) {
	fp := &FeedbackPage{
		Name:                 "Page",
		HTML:                 "<p>test</p>",
		RedirectDelaySeconds: -5,
	}
	if err := fp.Validate(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if fp.RedirectDelaySeconds != 0 {
		t.Fatalf("negative redirect delay should be clamped to 0, got %d", fp.RedirectDelaySeconds)
	}
	if fp.Language != "en" {
		t.Fatalf("empty language should default to 'en', got %q", fp.Language)
	}
}

func TestPostAndGetFeedbackPage(t *testing.T) {
	teardown := setupFeedbackPageTest(t)
	defer teardown()

	scope := OrgScope{OrgId: 1}
	fp := &FeedbackPage{
		UserId: 1,
		OrgId:  1,
		Name:   "Phishing Alert",
		HTML:   "<p>You clicked a phishing link!</p>",
	}
	if err := PostFeedbackPage(fp); err != nil {
		t.Fatalf("PostFeedbackPage: %v", err)
	}
	if fp.Id == 0 {
		t.Fatal("expected non-zero ID")
	}
	if fp.ModifiedDate.IsZero() {
		t.Fatal("expected ModifiedDate to be set")
	}

	got, err := GetFeedbackPage(fp.Id, scope)
	if err != nil {
		t.Fatalf("GetFeedbackPage: %v", err)
	}
	if got.Name != "Phishing Alert" {
		t.Fatalf("expected name 'Phishing Alert', got %q", got.Name)
	}
}

func TestGetFeedbackPageByName(t *testing.T) {
	teardown := setupFeedbackPageTest(t)
	defer teardown()

	scope := OrgScope{OrgId: 1}
	fp := &FeedbackPage{OrgId: 1, Name: "Unique Page", HTML: "<p>content</p>"}
	PostFeedbackPage(fp)

	got, err := GetFeedbackPageByName("Unique Page", scope)
	if err != nil {
		t.Fatalf("GetFeedbackPageByName: %v", err)
	}
	if got.Id != fp.Id {
		t.Fatalf("expected id %d, got %d", fp.Id, got.Id)
	}
}

func TestGetFeedbackPages(t *testing.T) {
	teardown := setupFeedbackPageTest(t)
	defer teardown()

	scope := OrgScope{OrgId: 1}
	for _, name := range []string{"Page A", "Page B", "Page C"} {
		PostFeedbackPage(&FeedbackPage{OrgId: 1, Name: name, HTML: "<p>x</p>"})
	}

	pages, err := GetFeedbackPages(scope)
	if err != nil {
		t.Fatalf("GetFeedbackPages: %v", err)
	}
	if len(pages) != 3 {
		t.Fatalf("expected 3 pages, got %d", len(pages))
	}
}

func TestPutFeedbackPage(t *testing.T) {
	teardown := setupFeedbackPageTest(t)
	defer teardown()

	fp := &FeedbackPage{OrgId: 1, Name: "Original", HTML: "<p>original</p>"}
	PostFeedbackPage(fp)

	fp.Name = "Updated"
	fp.HTML = "<p>updated</p>"
	if err := PutFeedbackPage(fp); err != nil {
		t.Fatalf("PutFeedbackPage: %v", err)
	}

	scope := OrgScope{OrgId: 1}
	got, _ := GetFeedbackPage(fp.Id, scope)
	if got.Name != "Updated" {
		t.Fatalf("expected name 'Updated', got %q", got.Name)
	}
}

func TestDeleteFeedbackPage(t *testing.T) {
	teardown := setupFeedbackPageTest(t)
	defer teardown()

	scope := OrgScope{OrgId: 1}
	fp := &FeedbackPage{OrgId: 1, Name: "Doomed", HTML: "<p>bye</p>"}
	PostFeedbackPage(fp)

	if err := DeleteFeedbackPage(fp.Id, scope); err != nil {
		t.Fatalf("DeleteFeedbackPage: %v", err)
	}
	pages, _ := GetFeedbackPages(scope)
	if len(pages) != 0 {
		t.Fatalf("expected 0 pages after delete, got %d", len(pages))
	}
}

func TestFeedbackPageOrgIsolation(t *testing.T) {
	teardown := setupFeedbackPageTest(t)
	defer teardown()

	PostFeedbackPage(&FeedbackPage{OrgId: 1, Name: "Org1 Page", HTML: "<p>1</p>"})
	PostFeedbackPage(&FeedbackPage{OrgId: 2, Name: "Org2 Page", HTML: "<p>2</p>"})

	scope1 := OrgScope{OrgId: 1}
	scope2 := OrgScope{OrgId: 2}

	pages1, _ := GetFeedbackPages(scope1)
	pages2, _ := GetFeedbackPages(scope2)

	if len(pages1) != 1 || pages1[0].Name != "Org1 Page" {
		t.Fatalf("org 1 should only see its own page, got %d pages", len(pages1))
	}
	if len(pages2) != 1 || pages2[0].Name != "Org2 Page" {
		t.Fatalf("org 2 should only see its own page, got %d pages", len(pages2))
	}
}

func TestDefaultFeedbackHTML(t *testing.T) {
	langs := []string{"en", "nl", "fr", "de", "es", "unknown"}
	for _, lang := range langs {
		html := DefaultFeedbackHTML(lang)
		if html == "" {
			t.Fatalf("DefaultFeedbackHTML(%q) returned empty string", lang)
		}
		if len(html) < 100 {
			t.Fatalf("DefaultFeedbackHTML(%q) looks too short: %d chars", lang, len(html))
		}
	}
	// Unknown language should fall back to English.
	if DefaultFeedbackHTML("xx") != DefaultFeedbackHTML("en") {
		t.Fatal("unknown language should return English default")
	}
}
