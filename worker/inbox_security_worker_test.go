package worker

import (
	"testing"
	"time"

	"github.com/gophish/gophish/models"
)

func TestInboxSecurityCheckInterval(t *testing.T) {
	if InboxSecurityCheckInterval != 5*time.Minute {
		t.Fatalf("expected 5m, got %v", InboxSecurityCheckInterval)
	}
}

func TestInboxSecurityCheckIntervalPositive(t *testing.T) {
	if InboxSecurityCheckInterval <= 0 {
		t.Fatal("interval must be positive")
	}
}

// ─── shouldAutoRemediate tests ───

func TestShouldAutoRemediateConfirmedMeetsConfirmed(t *testing.T) {
	if !shouldAutoRemediate(models.ThreatLevelConfirmedPhishing, models.ThreatLevelConfirmedPhishing) {
		t.Fatal("expected confirmed_phishing to meet confirmed_phishing threshold")
	}
}

func TestShouldAutoRemediateConfirmedMeetsSuspicious(t *testing.T) {
	if !shouldAutoRemediate(models.ThreatLevelConfirmedPhishing, models.ThreatLevelSuspicious) {
		t.Fatal("expected confirmed_phishing to meet suspicious threshold")
	}
}

func TestShouldAutoRemediateSafeDoesNotMeetSuspicious(t *testing.T) {
	if shouldAutoRemediate(models.ThreatLevelSafe, models.ThreatLevelSuspicious) {
		t.Fatal("expected safe to NOT meet suspicious threshold")
	}
}

func TestShouldAutoRemediateSuspiciousMeetsSuspicious(t *testing.T) {
	if !shouldAutoRemediate(models.ThreatLevelSuspicious, models.ThreatLevelSuspicious) {
		t.Fatal("expected suspicious to meet suspicious threshold")
	}
}

func TestShouldAutoRemediateSuspiciousDoesNotMeetConfirmed(t *testing.T) {
	if shouldAutoRemediate(models.ThreatLevelSuspicious, models.ThreatLevelConfirmedPhishing) {
		t.Fatal("expected suspicious to NOT meet confirmed_phishing threshold")
	}
}

func TestShouldAutoRemediateLikelyMeetsSuspicious(t *testing.T) {
	if !shouldAutoRemediate(models.ThreatLevelLikelyPhishing, models.ThreatLevelSuspicious) {
		t.Fatal("expected likely_phishing to meet suspicious threshold")
	}
}

func TestShouldAutoRemediateSafeMeetsSafe(t *testing.T) {
	if !shouldAutoRemediate(models.ThreatLevelSafe, models.ThreatLevelSafe) {
		t.Fatal("expected safe to meet safe threshold")
	}
}

func TestShouldAutoRemediateUnknownThreatLevel(t *testing.T) {
	// Unknown threat levels default to 0, so should meet safe but not suspicious
	if shouldAutoRemediate("unknown", models.ThreatLevelSuspicious) {
		t.Fatal("expected unknown to NOT meet suspicious threshold")
	}
	if !shouldAutoRemediate("unknown", "unknown") {
		t.Fatal("expected unknown to meet unknown threshold")
	}
}

func TestShouldAutoRemediateLikelyDoesNotMeetConfirmed(t *testing.T) {
	if shouldAutoRemediate(models.ThreatLevelLikelyPhishing, models.ThreatLevelConfirmedPhishing) {
		t.Fatal("expected likely_phishing to NOT meet confirmed_phishing threshold")
	}
}

// ─── whereByID constant test ───

func TestWhereByIDConstant(t *testing.T) {
	if whereByID != "id = ?" {
		t.Fatalf("expected 'id = ?', got %q", whereByID)
	}
}

// ─── InboxEmail struct tests ───

func TestInboxEmailStruct(t *testing.T) {
	e := InboxEmail{
		MessageId:    "test-123",
		SenderEmail:  "attacker@evil.com",
		Subject:      "Urgent: Password Reset",
		Headers:      "From: attacker@evil.com",
		Body:         "Click here to reset",
		ReceivedDate: time.Now(),
	}
	if e.MessageId != "test-123" {
		t.Fatal("message id mismatch")
	}
	if e.SenderEmail != "attacker@evil.com" {
		t.Fatal("sender email mismatch")
	}
}

// ─── resolveProviderForOrg tests (unit tests with model structs) ───

func TestResolveProviderForOrgMS365(t *testing.T) {
	cfg := &models.InboxMonitorConfig{
		MS365Enabled:      true,
		MS365TenantId:     "test-tenant",
		MS365ClientId:     "test-client",
		MS365ClientSecret: "test-secret",
	}
	provider, err := resolveProviderForOrg(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.ProviderName() != "microsoft_graph" {
		t.Fatalf("expected microsoft_graph, got %s", provider.ProviderName())
	}
}

func TestResolveProviderForOrgGoogle(t *testing.T) {
	cfg := &models.InboxMonitorConfig{
		GoogleWorkspaceEnabled: true,
		GoogleAdminEmail:       "admin@example.com",
	}
	provider, err := resolveProviderForOrg(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.ProviderName() != "gmail" {
		t.Fatalf("expected gmail, got %s", provider.ProviderName())
	}
}

func TestResolveProviderForOrgIMAP(t *testing.T) {
	cfg := &models.InboxMonitorConfig{
		IMAPHost:     "mail.example.com",
		IMAPPort:     993,
		IMAPUsername: "user",
		IMAPPassword: "pass",
		IMAPTLS:      true,
	}
	provider, err := resolveProviderForOrg(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.ProviderName() != "imap" {
		t.Fatalf("expected imap, got %s", provider.ProviderName())
	}
}

func TestResolveProviderForOrgNoProvider(t *testing.T) {
	cfg := &models.InboxMonitorConfig{
		OrgId: 99,
	}
	_, err := resolveProviderForOrg(cfg)
	if err == nil {
		t.Fatal("expected error for unconfigured org")
	}
}

func TestResolveProviderPriority(t *testing.T) {
	// MS365 should take priority over Google and IMAP
	cfg := &models.InboxMonitorConfig{
		MS365Enabled:           true,
		MS365TenantId:          "tenant",
		MS365ClientId:          "client",
		MS365ClientSecret:      "secret",
		GoogleWorkspaceEnabled: true,
		GoogleAdminEmail:       "admin@example.com",
		IMAPHost:               "mail.example.com",
		IMAPPort:               993,
	}
	provider, err := resolveProviderForOrg(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if provider.ProviderName() != "microsoft_graph" {
		t.Fatalf("expected microsoft_graph priority, got %s", provider.ProviderName())
	}
}

// ─── errFmtResolveProvider constant test ───

func TestErrFmtResolveProviderConstant(t *testing.T) {
	if errFmtResolveProvider == "" {
		t.Fatal("errFmtResolveProvider should not be empty")
	}
}
