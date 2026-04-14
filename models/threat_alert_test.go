package models

import (
	"testing"

	"github.com/gophish/gophish/config"
)

// setupThreatAlertTest initialises an in-memory DB for threat alert tests.
func setupThreatAlertTest(t *testing.T) func() {
	t.Helper()
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	if err := Setup(conf); err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	db.Exec("DELETE FROM threat_alert_reads")
	db.Exec("DELETE FROM threat_alerts")
	return func() {
		db.Exec("DELETE FROM threat_alert_reads")
		db.Exec("DELETE FROM threat_alerts")
	}
}

// ---------- CreateThreatAlert ----------

func TestCreateThreatAlert(t *testing.T) {
	teardown := setupThreatAlertTest(t)
	defer teardown()

	alert := &ThreatAlert{
		OrgId:    1,
		Title:    "Phishing wave detected",
		Body:     "A new phishing campaign is targeting our org.",
		Severity: "high",
	}
	if err := CreateThreatAlert(alert); err != nil {
		t.Fatalf("CreateThreatAlert failed: %v", err)
	}
	if alert.Id == 0 {
		t.Fatal("expected non-zero alert ID")
	}
	if alert.CreatedDate.IsZero() {
		t.Fatal("expected CreatedDate to be set")
	}
}

func TestCreateThreatAlertPublished(t *testing.T) {
	teardown := setupThreatAlertTest(t)
	defer teardown()

	alert := &ThreatAlert{
		OrgId:     1,
		Title:     "Urgent: credential phishing",
		Body:      "Watch out for credential harvesting emails.",
		Severity:  "critical",
		Published: true,
	}
	if err := CreateThreatAlert(alert); err != nil {
		t.Fatalf("CreateThreatAlert failed: %v", err)
	}
	if alert.PublishedDate.IsZero() {
		t.Fatal("expected PublishedDate to be set when Published=true")
	}
}

// ---------- UpdateThreatAlert ----------

func TestUpdateThreatAlert(t *testing.T) {
	teardown := setupThreatAlertTest(t)
	defer teardown()

	alert := &ThreatAlert{OrgId: 1, Title: "Draft Alert", Body: "Body", Severity: "info"}
	CreateThreatAlert(alert)

	alert.Title = "Updated Alert"
	alert.Published = true
	if err := UpdateThreatAlert(alert); err != nil {
		t.Fatalf("UpdateThreatAlert failed: %v", err)
	}

	fetched, err := GetThreatAlert(alert.Id, 1)
	if err != nil {
		t.Fatalf("GetThreatAlert failed: %v", err)
	}
	if fetched.Title != "Updated Alert" {
		t.Fatalf("expected title 'Updated Alert', got %q", fetched.Title)
	}
	if fetched.PublishedDate.IsZero() {
		t.Fatal("expected PublishedDate to be set after publishing")
	}
}

// ---------- GetThreatAlerts ----------

func TestGetThreatAlerts(t *testing.T) {
	teardown := setupThreatAlertTest(t)
	defer teardown()

	CreateThreatAlert(&ThreatAlert{OrgId: 1, Title: "Alert 1", Body: "B", Severity: "info"})
	CreateThreatAlert(&ThreatAlert{OrgId: 1, Title: "Alert 2", Body: "B", Severity: "high"})
	CreateThreatAlert(&ThreatAlert{OrgId: 2, Title: "Other Org", Body: "B", Severity: "low"})

	alerts, err := GetThreatAlerts(1)
	if err != nil {
		t.Fatalf("GetThreatAlerts failed: %v", err)
	}
	if len(alerts) != 2 {
		t.Fatalf("expected 2 alerts for org 1, got %d", len(alerts))
	}
}

// ---------- GetPublishedThreatAlerts ----------

func TestGetPublishedThreatAlerts(t *testing.T) {
	teardown := setupThreatAlertTest(t)
	defer teardown()

	CreateThreatAlert(&ThreatAlert{OrgId: 1, Title: "Draft", Body: "B", Severity: "info", Published: false})
	CreateThreatAlert(&ThreatAlert{OrgId: 1, Title: "Published", Body: "B", Severity: "high", Published: true})

	alerts, err := GetPublishedThreatAlerts(1, 1)
	if err != nil {
		t.Fatalf("GetPublishedThreatAlerts failed: %v", err)
	}
	if len(alerts) != 1 {
		t.Fatalf("expected 1 published alert, got %d", len(alerts))
	}
	if alerts[0].Title != "Published" {
		t.Fatalf("expected title 'Published', got %q", alerts[0].Title)
	}
}

// ---------- GetThreatAlert ----------

func TestGetThreatAlert(t *testing.T) {
	teardown := setupThreatAlertTest(t)
	defer teardown()

	alert := &ThreatAlert{OrgId: 1, Title: "Specific Alert", Body: "B", Severity: "medium"}
	CreateThreatAlert(alert)

	fetched, err := GetThreatAlert(alert.Id, 1)
	if err != nil {
		t.Fatalf("GetThreatAlert failed: %v", err)
	}
	if fetched.Title != "Specific Alert" {
		t.Fatalf("expected 'Specific Alert', got %q", fetched.Title)
	}
}

func TestGetThreatAlertWrongOrg(t *testing.T) {
	teardown := setupThreatAlertTest(t)
	defer teardown()

	alert := &ThreatAlert{OrgId: 1, Title: "Alert", Body: "B", Severity: "info"}
	CreateThreatAlert(alert)

	_, err := GetThreatAlert(alert.Id, 999)
	if err == nil {
		t.Fatal("expected error when fetching alert from wrong org")
	}
}

// ---------- DeleteThreatAlert ----------

func TestDeleteThreatAlert(t *testing.T) {
	teardown := setupThreatAlertTest(t)
	defer teardown()

	alert := &ThreatAlert{OrgId: 1, Title: "To Delete", Body: "B", Severity: "info"}
	CreateThreatAlert(alert)

	// Also mark it as read to verify cascade cleanup
	MarkThreatAlertRead(alert.Id, 1)

	err := DeleteThreatAlert(alert.Id, 1)
	if err != nil {
		t.Fatalf("DeleteThreatAlert failed: %v", err)
	}

	_, err = GetThreatAlert(alert.Id, 1)
	if err == nil {
		t.Fatal("expected error after deleting alert")
	}
}

func TestDeleteThreatAlertWrongOrg(t *testing.T) {
	teardown := setupThreatAlertTest(t)
	defer teardown()

	alert := &ThreatAlert{OrgId: 1, Title: "Alert", Body: "B", Severity: "info"}
	CreateThreatAlert(alert)

	err := DeleteThreatAlert(alert.Id, 999)
	if err == nil {
		t.Fatal("expected error when deleting alert from wrong org")
	}
}

// ---------- MarkThreatAlertRead ----------

func TestMarkThreatAlertRead(t *testing.T) {
	teardown := setupThreatAlertTest(t)
	defer teardown()

	alert := &ThreatAlert{OrgId: 1, Title: "Readable Alert", Body: "B", Severity: "info", Published: true}
	CreateThreatAlert(alert)

	if err := MarkThreatAlertRead(alert.Id, 42); err != nil {
		t.Fatalf("MarkThreatAlertRead failed: %v", err)
	}

	// Marking again should be idempotent
	if err := MarkThreatAlertRead(alert.Id, 42); err != nil {
		t.Fatalf("MarkThreatAlertRead (duplicate) failed: %v", err)
	}

	// Verify only one read record exists
	var count int
	db.Model(&ThreatAlertRead{}).Where("alert_id = ? AND user_id = ?", alert.Id, 42).Count(&count)
	if count != 1 {
		t.Fatalf("expected 1 read record, got %d", count)
	}
}

// ---------- GetUnreadThreatAlertCount ----------

func TestGetUnreadThreatAlertCount(t *testing.T) {
	teardown := setupThreatAlertTest(t)
	defer teardown()

	CreateThreatAlert(&ThreatAlert{OrgId: 1, Title: "A1", Body: "B", Severity: "info", Published: true})
	a2 := &ThreatAlert{OrgId: 1, Title: "A2", Body: "B", Severity: "info", Published: true}
	CreateThreatAlert(a2)
	CreateThreatAlert(&ThreatAlert{OrgId: 1, Title: "A3", Body: "B", Severity: "info", Published: true})

	// User 1 reads one alert
	MarkThreatAlertRead(a2.Id, 1)

	unread := GetUnreadThreatAlertCount(1, 1)
	if unread != 2 {
		t.Fatalf("expected 2 unread alerts, got %d", unread)
	}
}

func TestGetUnreadThreatAlertCountNoAlerts(t *testing.T) {
	teardown := setupThreatAlertTest(t)
	defer teardown()

	unread := GetUnreadThreatAlertCount(1, 1)
	if unread != 0 {
		t.Fatalf("expected 0 unread alerts, got %d", unread)
	}
}

// ---------- IsRead flag in GetPublishedThreatAlerts ----------

func TestPublishedAlertsIsReadFlag(t *testing.T) {
	teardown := setupThreatAlertTest(t)
	defer teardown()

	a1 := &ThreatAlert{OrgId: 1, Title: "Read Me", Body: "B", Severity: "info", Published: true}
	CreateThreatAlert(a1)
	a2 := &ThreatAlert{OrgId: 1, Title: "Unread", Body: "B", Severity: "info", Published: true}
	CreateThreatAlert(a2)

	MarkThreatAlertRead(a1.Id, 1)

	alerts, _ := GetPublishedThreatAlerts(1, 1)
	readCount := 0
	for _, a := range alerts {
		if a.IsRead {
			readCount++
		}
	}
	if readCount != 1 {
		t.Fatalf("expected 1 read alert, got %d", readCount)
	}
}

// ---------- ReadCount in GetThreatAlerts ----------

func TestThreatAlertReadCount(t *testing.T) {
	teardown := setupThreatAlertTest(t)
	defer teardown()

	alert := &ThreatAlert{OrgId: 1, Title: "Popular Alert", Body: "B", Severity: "info"}
	CreateThreatAlert(alert)

	// 3 users read the alert
	MarkThreatAlertRead(alert.Id, 1)
	MarkThreatAlertRead(alert.Id, 2)
	MarkThreatAlertRead(alert.Id, 3)

	fetched, _ := GetThreatAlert(alert.Id, 1)
	if fetched.ReadCount != 3 {
		t.Fatalf("expected ReadCount 3, got %d", fetched.ReadCount)
	}
}
