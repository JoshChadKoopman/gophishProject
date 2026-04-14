package models

import (
	"testing"

	"github.com/gophish/gophish/config"
)

func setupInboxSecurityTest(t *testing.T) func() {
	t.Helper()
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	if err := Setup(conf); err != nil {
		t.Fatalf("Failed to set up database: %v", err)
	}
	return func() { /* in-memory DB cleaned up automatically */ }
}

// ─── InboxMonitorConfig ───

func TestInboxMonitorConfigTableName(t *testing.T) {
	c := InboxMonitorConfig{}
	if c.TableName() != "inbox_monitor_configs" {
		t.Fatalf("expected 'inbox_monitor_configs', got %q", c.TableName())
	}
}

func TestInboxMonitorConfigMailboxList(t *testing.T) {
	c := InboxMonitorConfig{}
	c.SetMonitoredMailboxList([]string{"inbox@example.com", "admin@example.com"})
	list := c.GetMonitoredMailboxList()
	if len(list) != 2 {
		t.Fatalf("expected 2 mailboxes, got %d", len(list))
	}
	if list[0] != "inbox@example.com" {
		t.Fatalf("expected 'inbox@example.com', got %q", list[0])
	}
}

func TestInboxMonitorConfigEmptyMailboxList(t *testing.T) {
	c := InboxMonitorConfig{}
	list := c.GetMonitoredMailboxList()
	if len(list) != 0 {
		t.Fatalf("expected 0 mailboxes, got %d", len(list))
	}
}

func TestSaveAndGetInboxMonitorConfig(t *testing.T) {
	teardown := setupInboxSecurityTest(t)
	defer teardown()

	cfg := &InboxMonitorConfig{
		OrgId:               1,
		Enabled:             true,
		ScanIntervalSeconds: 600,
		ThreatThreshold:     "suspicious",
		AutoQuarantine:      true,
	}
	cfg.SetMonitoredMailboxList([]string{"test@example.com"})

	if err := SaveInboxMonitorConfig(cfg); err != nil {
		t.Fatalf("SaveInboxMonitorConfig: %v", err)
	}

	got, err := GetInboxMonitorConfig(1)
	if err != nil {
		t.Fatalf("GetInboxMonitorConfig: %v", err)
	}
	if !got.Enabled {
		t.Fatal("expected config to be enabled")
	}
	if got.ScanIntervalSeconds != 600 {
		t.Fatalf("expected 600s interval, got %d", got.ScanIntervalSeconds)
	}
}

func TestGetInboxMonitorConfigNotFound(t *testing.T) {
	teardown := setupInboxSecurityTest(t)
	defer teardown()

	_, err := GetInboxMonitorConfig(9999)
	if err == nil {
		t.Fatal("expected error for non-existent config")
	}
}

func TestGetAllEnabledMonitorConfigs(t *testing.T) {
	teardown := setupInboxSecurityTest(t)
	defer teardown()

	// Create an enabled config
	cfg := &InboxMonitorConfig{OrgId: 1, Enabled: true, ScanIntervalSeconds: 300}
	SaveInboxMonitorConfig(cfg)

	configs, err := GetAllEnabledMonitorConfigs()
	if err != nil {
		t.Fatalf("GetAllEnabledMonitorConfigs: %v", err)
	}
	if len(configs) < 1 {
		t.Fatal("expected at least 1 enabled config")
	}
}

// ─── InboxScanResult ───

func TestInboxScanResultTableName(t *testing.T) {
	r := InboxScanResult{}
	if r.TableName() != "inbox_scan_results" {
		t.Fatalf("expected 'inbox_scan_results', got %q", r.TableName())
	}
}

func TestCreateAndGetInboxScanResult(t *testing.T) {
	teardown := setupInboxSecurityTest(t)
	defer teardown()

	r := &InboxScanResult{
		OrgId:           1,
		MailboxEmail:    "test@example.com",
		MessageId:       "msg-001",
		SenderEmail:     "attacker@evil.com",
		Subject:         "Urgent wire transfer",
		ThreatLevel:     ThreatLevelConfirmedPhishing,
		Classification:  ClassificationBEC,
		ConfidenceScore: 95.5,
		IsBEC:           true,
		Summary:         "BEC attack detected",
		ActionTaken:     ScanActionQuarantined,
	}
	if err := CreateInboxScanResult(r); err != nil {
		t.Fatalf("CreateInboxScanResult: %v", err)
	}
	if r.Id == 0 {
		t.Fatal("expected result to have an ID")
	}

	results, err := GetInboxScanResults(1, 10)
	if err != nil {
		t.Fatalf("GetInboxScanResults: %v", err)
	}
	if len(results) < 1 {
		t.Fatal("expected at least 1 result")
	}
}

func TestGetInboxScanResultsByThreat(t *testing.T) {
	teardown := setupInboxSecurityTest(t)
	defer teardown()

	CreateInboxScanResult(&InboxScanResult{
		OrgId: 1, MailboxEmail: "a@b.com", ThreatLevel: ThreatLevelSafe,
	})
	CreateInboxScanResult(&InboxScanResult{
		OrgId: 1, MailboxEmail: "a@b.com", ThreatLevel: ThreatLevelConfirmedPhishing,
	})

	safe, err := GetInboxScanResultsByThreat(1, ThreatLevelSafe)
	if err != nil {
		t.Fatalf("GetInboxScanResultsByThreat: %v", err)
	}
	if len(safe) != 1 {
		t.Fatalf("expected 1 safe result, got %d", len(safe))
	}
}

func TestGetInboxScanSummary(t *testing.T) {
	teardown := setupInboxSecurityTest(t)
	defer teardown()

	CreateInboxScanResult(&InboxScanResult{
		OrgId: 1, ThreatLevel: ThreatLevelSafe, ConfidenceScore: 90,
	})
	CreateInboxScanResult(&InboxScanResult{
		OrgId: 1, ThreatLevel: ThreatLevelConfirmedPhishing, IsBEC: true, ConfidenceScore: 95,
	})

	summary, err := GetInboxScanSummary(1)
	if err != nil {
		t.Fatalf("GetInboxScanSummary: %v", err)
	}
	if summary.TotalScanned != 2 {
		t.Fatalf("expected 2 total scanned, got %d", summary.TotalScanned)
	}
	if summary.SafeEmails != 1 {
		t.Fatalf("expected 1 safe email, got %d", summary.SafeEmails)
	}
}

func TestInboxScanResultGetIndicatorsList(t *testing.T) {
	r := InboxScanResult{
		Indicators: `[{"type":"url","value":"http://evil.com","severity":"high"}]`,
	}
	indicators := r.GetIndicatorsList()
	if len(indicators) != 1 {
		t.Fatalf("expected 1 indicator, got %d", len(indicators))
	}
}

func TestInboxScanResultGetIndicatorsListEmpty(t *testing.T) {
	r := InboxScanResult{Indicators: ""}
	indicators := r.GetIndicatorsList()
	if len(indicators) != 0 {
		t.Fatalf("expected 0 indicators, got %d", len(indicators))
	}
}

// ─── Scan action constants ───

func TestScanActionConstants(t *testing.T) {
	if ScanActionNone != "none" {
		t.Fatalf("expected 'none', got %q", ScanActionNone)
	}
	if ScanActionQuarantined != "quarantined" {
		t.Fatalf("expected 'quarantined', got %q", ScanActionQuarantined)
	}
	if ScanActionDeleted != "deleted" {
		t.Fatalf("expected 'deleted', got %q", ScanActionDeleted)
	}
}

// ─── BEC constants ───

func TestBECAttackConstants(t *testing.T) {
	if BECAttackCEOFraud != "ceo_fraud" {
		t.Fatalf("expected 'ceo_fraud', got %q", BECAttackCEOFraud)
	}
	if BECAttackInvoiceFraud != "invoice_fraud" {
		t.Fatalf("expected 'invoice_fraud', got %q", BECAttackInvoiceFraud)
	}
}
