package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	ctx "github.com/gophish/gophish/context"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/mux"
)

// ── Helpers ──

func setupIntegrationTest(t *testing.T) *testContext {
	t.Helper()
	return setupTest(t)
}

// makeRequestWithVars creates a test request with Gorilla mux URL vars pre-set.
// Use this for handlers that call mux.Vars(r) to extract path parameters.
func makeRequestWithVars(t *testing.T, tc *testContext, method, path string, body interface{}, vars map[string]string) (*httptest.ResponseRecorder, *http.Request) {
	t.Helper()
	w, r := makeRequest(t, tc, method, path, body)
	r = mux.SetURLVars(r, vars)
	return w, r
}

func makeRequest(t *testing.T, tc *testContext, method, path string, body interface{}) (*httptest.ResponseRecorder, *http.Request) {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			t.Fatalf("error encoding body: %v", err)
		}
	}
	r := httptest.NewRequest(method, path, &buf)
	r.Header.Set("Content-Type", "application/json")
	r = ctx.Set(r, "user", tc.admin)
	w := httptest.NewRecorder()
	return w, r
}

func expectStatus(t *testing.T, w *httptest.ResponseRecorder, expected int) {
	t.Helper()
	if w.Code != expected {
		t.Fatalf("expected status %d, got %d; body: %s", expected, w.Code, w.Body.String())
	}
}

// ── Template API ──

func TestTemplatesCRUDAPI(t *testing.T) {
	tc := setupIntegrationTest(t)

	// Create
	tmpl := models.Template{
		Name:    "Integration Test Template",
		Subject: "Test Subject",
		Text:    "Hello {{.FirstName}}",
		HTML:    "<html>Hello {{.FirstName}}</html>",
		UserId:  tc.admin.Id,
	}
	body, _ := json.Marshal(tmpl)
	w, r := makeRequest(t, tc, http.MethodPost, "/api/templates/", nil)
	r.Body = http.NoBody
	r = httptest.NewRequest(http.MethodPost, "/api/templates/", bytes.NewBuffer(body))
	r.Header.Set("Content-Type", "application/json")
	r = ctx.Set(r, "user", tc.admin)
	w = httptest.NewRecorder()
	tc.apiServer.Templates(w, r)
	expectStatus(t, w, http.StatusCreated)

	var created models.Template
	if err := json.NewDecoder(w.Body).Decode(&created); err != nil {
		t.Fatalf("error decoding created template: %v", err)
	}
	if created.Name != "Integration Test Template" {
		t.Errorf("expected name 'Integration Test Template', got '%s'", created.Name)
	}

	// List
	w, r = makeRequest(t, tc, http.MethodGet, "/api/templates/", nil)
	tc.apiServer.Templates(w, r)
	expectStatus(t, w, http.StatusOK)

	var templates []models.Template
	if err := json.NewDecoder(w.Body).Decode(&templates); err != nil {
		t.Fatalf("error decoding templates: %v", err)
	}
	if len(templates) == 0 {
		t.Error("expected at least one template")
	}
}

func TestTemplateCreateMissingNameAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	tmpl := models.Template{Text: "Some content"}
	body, _ := json.Marshal(tmpl)
	r := httptest.NewRequest(http.MethodPost, "/api/templates/", bytes.NewBuffer(body))
	r.Header.Set("Content-Type", "application/json")
	r = ctx.Set(r, "user", tc.admin)
	w := httptest.NewRecorder()
	tc.apiServer.Templates(w, r)
	if w.Code == http.StatusCreated {
		t.Error("expected error when creating template without name")
	}
}

// ── Group API ──

func TestGroupsCRUDAPI(t *testing.T) {
	tc := setupIntegrationTest(t)

	// Create
	group := models.Group{
		Name: "Integration Test Group",
		Targets: []models.Target{
			{BaseRecipient: models.BaseRecipient{Email: "test@example.com", FirstName: "Test", LastName: "User"}},
		},
		UserId: tc.admin.Id,
	}
	body, _ := json.Marshal(group)
	r := httptest.NewRequest(http.MethodPost, "/api/groups/", bytes.NewBuffer(body))
	r.Header.Set("Content-Type", "application/json")
	r = ctx.Set(r, "user", tc.admin)
	w := httptest.NewRecorder()
	tc.apiServer.Groups(w, r)
	expectStatus(t, w, http.StatusCreated)

	// List
	w, r = makeRequest(t, tc, http.MethodGet, "/api/groups/", nil)
	tc.apiServer.Groups(w, r)
	expectStatus(t, w, http.StatusOK)

	var groups []models.Group
	if err := json.NewDecoder(w.Body).Decode(&groups); err != nil {
		t.Fatalf("error decoding groups: %v", err)
	}
	if len(groups) == 0 {
		t.Error("expected at least one group")
	}
}

// ── Page API ──

func TestPagesCRUDAPI(t *testing.T) {
	tc := setupIntegrationTest(t)

	// Create
	page := models.Page{
		Name:   "Integration Test Page",
		HTML:   "<html><body>Test</body></html>",
		UserId: tc.admin.Id,
	}
	body, _ := json.Marshal(page)
	r := httptest.NewRequest(http.MethodPost, "/api/pages/", bytes.NewBuffer(body))
	r.Header.Set("Content-Type", "application/json")
	r = ctx.Set(r, "user", tc.admin)
	w := httptest.NewRecorder()
	tc.apiServer.Pages(w, r)
	expectStatus(t, w, http.StatusCreated)

	// List
	w, r = makeRequest(t, tc, http.MethodGet, "/api/pages/", nil)
	tc.apiServer.Pages(w, r)
	expectStatus(t, w, http.StatusOK)

	var pages []models.Page
	if err := json.NewDecoder(w.Body).Decode(&pages); err != nil {
		t.Fatalf("error decoding pages: %v", err)
	}
	if len(pages) == 0 {
		t.Error("expected at least one page")
	}
}

// ── SMTP API ──

func TestSMTPProfileListAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/smtp/", nil)
	tc.apiServer.SendingProfiles(w, r)
	expectStatus(t, w, http.StatusOK)
}

// ── Report API ──

func TestReportOverviewAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/reports/overview", nil)
	tc.apiServer.ReportOverview(w, r)
	expectStatus(t, w, http.StatusOK)

	var overview models.ReportOverview
	if err := json.NewDecoder(w.Body).Decode(&overview); err != nil {
		t.Fatalf("error decoding overview: %v", err)
	}
	if overview.TotalCampaigns != 0 {
		t.Errorf("expected 0 campaigns in fresh DB, got %d", overview.TotalCampaigns)
	}
}

func TestReportTrendAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/reports/trend?days=7", nil)
	tc.apiServer.ReportTrend(w, r)
	expectStatus(t, w, http.StatusOK)

	var points []models.TrendPoint
	if err := json.NewDecoder(w.Body).Decode(&points); err != nil {
		t.Fatalf("error decoding trend: %v", err)
	}
	if len(points) < 7 {
		t.Errorf("expected at least 7 trend points, got %d", len(points))
	}
}

func TestReportRiskScoresAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/reports/risk-scores", nil)
	tc.apiServer.ReportRiskScores(w, r)
	expectStatus(t, w, http.StatusOK)
}

func TestReportTrainingSummaryAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/reports/training-summary", nil)
	tc.apiServer.ReportTrainingSummary(w, r)
	expectStatus(t, w, http.StatusOK)

	var summary models.TrainingSummary
	if err := json.NewDecoder(w.Body).Decode(&summary); err != nil {
		t.Fatalf("error decoding training summary: %v", err)
	}
}

// ── Template Library (Phishing) API ──

func TestTemplateLibraryListAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/template-library/", nil)
	tc.apiServer.TemplateLibraryList(w, r)
	expectStatus(t, w, http.StatusOK)

	var templates []models.LibraryTemplate
	if err := json.NewDecoder(w.Body).Decode(&templates); err != nil {
		t.Fatalf("error decoding template library: %v", err)
	}
	if len(templates) == 0 {
		t.Error("expected non-empty template library")
	}
}

// ── Content Library API ──

func TestContentLibraryListAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/content-library/", nil)
	tc.apiServer.ContentLibrary(w, r)
	expectStatus(t, w, http.StatusOK)
}

func TestContentLibraryCategoriesAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/content-library/categories", nil)
	tc.apiServer.ContentLibraryCategories(w, r)
	expectStatus(t, w, http.StatusOK)
}

// ── Gamification API ──

func TestGamificationLeaderboardAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/gamification/leaderboard", nil)
	tc.apiServer.GamificationLeaderboard(w, r)
	expectStatus(t, w, http.StatusOK)
}

func TestGamificationBadgesAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/gamification/badges", nil)
	tc.apiServer.GamificationBadges(w, r)
	expectStatus(t, w, http.StatusOK)
}

// ── Training API ──

func TestTrainingListAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/training/", nil)
	tc.apiServer.TrainingPresentations(w, r)
	expectStatus(t, w, http.StatusOK)
}

func TestTrainingAssignmentListAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/training/assignments/", nil)
	tc.apiServer.TrainingAssignments(w, r)
	expectStatus(t, w, http.StatusOK)
}

// ── Academy API ──

func TestAcademyTiersAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/academy/tiers", nil)
	tc.apiServer.AcademyTiers(w, r)
	expectStatus(t, w, http.StatusOK)
}

// ── Compliance API ──

func TestComplianceFrameworksAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/compliance/frameworks", nil)
	tc.apiServer.ComplianceFrameworks(w, r)
	expectStatus(t, w, http.StatusOK)
}

func TestComplianceModulesAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/compliance/training-modules", nil)
	tc.apiServer.ComplianceTrainingModules(w, r)
	expectStatus(t, w, http.StatusOK)
}

func TestComplianceCertsDefinitionsAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/compliance/cert-definitions", nil)
	tc.apiServer.FrameworkCertDefinitions(w, r)
	expectStatus(t, w, http.StatusOK)
}

// ── Remediation API ──

func TestRemediationPathsAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/remediation/", nil)
	tc.apiServer.RemediationPaths(w, r)
	expectStatus(t, w, http.StatusOK)
}

func TestRemediationSummaryAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/remediation/summary", nil)
	tc.apiServer.RemediationSummary(w, r)
	expectStatus(t, w, http.StatusOK)
}

// ── BRS API ──

func TestBRSOverviewAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/brs/overview", nil)
	tc.apiServer.BRSBenchmark(w, r)
	expectStatus(t, w, http.StatusOK)
}

// ── Certificate API ──

func TestCertificateListAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/certificates/", nil)
	tc.apiServer.TrainingMyCertificates(w, r)
	expectStatus(t, w, http.StatusOK)
}

// ── Tier API ──

func TestTiersListAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/tiers/", nil)
	tc.apiServer.Tiers(w, r)
	expectStatus(t, w, http.StatusOK)
}

// ── Threat Alerts API ──

func TestThreatAlertsListAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/threat-alerts/", nil)
	tc.apiServer.ThreatAlerts(w, r)
	expectStatus(t, w, http.StatusOK)
}

// ── Autopilot API ──

func TestAutopilotConfigAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/autopilot/config", nil)
	tc.apiServer.AutopilotConfig(w, r)
	// Returns 404 when no config exists in a fresh DB
	expectStatus(t, w, http.StatusNotFound)
}

// ── Report Button API ──

func TestReportButtonConfigAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/report-button/config", nil)
	tc.apiServer.ReportButtonConfig(w, r)
	expectStatus(t, w, http.StatusOK)
}

// ── Network Events API ──

func TestNetworkEventsListAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/network-events/?limit=10", nil)
	tc.apiServer.NetworkEvents(w, r)
	expectStatus(t, w, http.StatusOK)
}

func TestNetworkEventDashboardAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/network-events/dashboard", nil)
	tc.apiServer.NetworkEventDashboard(w, r)
	expectStatus(t, w, http.StatusOK)
}

func TestNetworkEventRulesAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/network-events/rules/", nil)
	tc.apiServer.NetworkEventRules(w, r)
	expectStatus(t, w, http.StatusOK)
}

// ── Email Analysis API ──

func TestEmailAnalysesListAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/email-analysis/", nil)
	tc.apiServer.EmailAnalyses(w, r)
	expectStatus(t, w, http.StatusOK)
}

func TestEmailAnalysisSummaryAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/email-analysis/summary", nil)
	tc.apiServer.EmailAnalysisSummary(w, r)
	expectStatus(t, w, http.StatusOK)
}

// ── JSON Response Format ──

func TestAPIResponseIsJSON(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/templates/", nil)
	tc.apiServer.Templates(w, r)
	expectStatus(t, w, http.StatusOK)

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type 'application/json', got '%s'", ct)
	}
}

// ── Campaign Analytics API ──

func TestCampaignAnalyticsFunnelAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequestWithVars(t, tc, http.MethodGet, "/api/campaigns/1/analytics/funnel", nil, map[string]string{"id": "1"})
	tc.apiServer.CampaignAnalyticsFunnel(w, r)
	// Campaign 1 doesn't exist, but handler should return 200 with empty funnel or 404
	if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
		t.Errorf("expected 200 or 404, got %d", w.Code)
	}
}

func TestCampaignAnalyticsTimeToClickAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequestWithVars(t, tc, http.MethodGet, "/api/campaigns/1/analytics/time-to-click", nil, map[string]string{"id": "1"})
	tc.apiServer.CampaignAnalyticsTimeToClick(w, r)
	if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
		t.Errorf("expected 200 or 404, got %d", w.Code)
	}
}

func TestCampaignAnalyticsRepeatOffendersAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequestWithVars(t, tc, http.MethodGet, "/api/campaigns/1/analytics/repeat-offenders", nil, map[string]string{"id": "1"})
	tc.apiServer.CampaignAnalyticsRepeatOffenders(w, r)
	if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
		t.Errorf("expected 200 or 404, got %d", w.Code)
	}
}

func TestCampaignAnalyticsDeviceBreakdownAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequestWithVars(t, tc, http.MethodGet, "/api/campaigns/1/analytics/devices", nil, map[string]string{"id": "1"})
	tc.apiServer.CampaignAnalyticsDeviceBreakdown(w, r)
	if w.Code != http.StatusOK && w.Code != http.StatusNotFound {
		t.Errorf("expected 200 or 404, got %d", w.Code)
	}
}

// ── ROI Enhanced API ──

func TestROIBenchmarksListAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/roi/benchmarks", nil)
	tc.apiServer.ROIBenchmarks(w, r)
	expectStatus(t, w, http.StatusOK)
}

func TestROIMonteCarloAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/roi/monte-carlo", nil)
	tc.apiServer.ROIMonteCarlo(w, r)
	expectStatus(t, w, http.StatusOK)
}

func TestROIHistoryListAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/roi/history", nil)
	tc.apiServer.ROIHistory(w, r)
	expectStatus(t, w, http.StatusOK)
}

func TestROIQuarterlyTrendAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/roi/trend", nil)
	tc.apiServer.ROIQuarterlyTrend(w, r)
	expectStatus(t, w, http.StatusOK)
}

// ── Scheduled Reports API ──

func TestScheduledReportsListAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/scheduled-reports/", nil)
	tc.apiServer.ScheduledReports(w, r)
	expectStatus(t, w, http.StatusOK)
}

func TestScheduledReportTypesAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/scheduled-reports/types", nil)
	tc.apiServer.ScheduledReportTypes(w, r)
	expectStatus(t, w, http.StatusOK)
}

func TestScheduledReportSummaryAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/scheduled-reports/summary", nil)
	tc.apiServer.ScheduledReportSummary(w, r)
	expectStatus(t, w, http.StatusOK)
}

// ── Unified Export API ──

func TestUnifiedExportMissingTypeAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/export", nil)
	tc.apiServer.UnifiedExport(w, r)
	// Should return 400 because report_type query param is missing
	if w.Code != http.StatusBadRequest && w.Code != http.StatusOK {
		t.Errorf("expected 400 or 200, got %d", w.Code)
	}
}

// ── Email Security Ops API ──

func TestEmailSecurityDashboardAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/email-security/dashboard", nil)
	tc.apiServer.EmailSecurityDashboard(w, r)
	expectStatus(t, w, http.StatusOK)
}

func TestEmailSecurityOpsAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/email-security/ops", nil)
	tc.apiServer.EmailSecurityOps(w, r)
	expectStatus(t, w, http.StatusOK)
}

// ── Admin Assistant API ──

func TestAdminAssistantOnboardingAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/admin-assistant/onboarding", nil)
	tc.apiServer.AdminAssistantOnboarding(w, r)
	expectStatus(t, w, http.StatusOK)
}

func TestAdminAssistantConversationsAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/admin-assistant/conversations", nil)
	tc.apiServer.AdminAssistantConversations(w, r)
	expectStatus(t, w, http.StatusOK)
}

// ── Enhanced Board Reports API ──

func TestBoardReportHeatmapAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/board-reports/heatmap", nil)
	tc.apiServer.BoardReportHeatmap(w, r)
	expectStatus(t, w, http.StatusOK)
}

func TestBoardReportDeltasAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	payload := map[string]string{"period_start": "2026-01-01", "period_end": "2026-03-31"}
	w, r := makeRequest(t, tc, http.MethodPost, "/api/board-reports/deltas", payload)
	tc.apiServer.BoardReportDeltas(w, r)
	expectStatus(t, w, http.StatusOK)
}

// ── Inbox AI Feedback API ──

func TestInboxFeedbackListAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/inbox/feedback", nil)
	tc.apiServer.InboxFeedback(w, r)
	expectStatus(t, w, http.StatusOK)
}

func TestInboxFeedbackUnreadAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/inbox/feedback/unread-count", nil)
	tc.apiServer.InboxFeedbackUnread(w, r)
	expectStatus(t, w, http.StatusOK)
}

func TestInboxAIAccuracyAPI(t *testing.T) {
	tc := setupIntegrationTest(t)
	w, r := makeRequest(t, tc, http.MethodGet, "/api/inbox/ai-accuracy", nil)
	tc.apiServer.InboxAIAccuracy(w, r)
	expectStatus(t, w, http.StatusOK)
}
