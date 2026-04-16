package api

import (
	"encoding/json"
	"net/http"

	"github.com/gophish/gophish/config"
	ctx "github.com/gophish/gophish/context"
	mid "github.com/gophish/gophish/middleware"
	"github.com/gophish/gophish/middleware/ratelimit"
	"github.com/gophish/gophish/models"
	"github.com/gophish/gophish/worker"
	"github.com/gorilla/mux"
)

// getOrgScope builds an OrgScope from the authenticated user in the request context.
func getOrgScope(r *http.Request) models.OrgScope {
	user := ctx.Get(r, "user").(models.User)
	return models.OrgScope{
		OrgId:        user.OrgId,
		UserId:       user.Id,
		IsSuperAdmin: user.Role.Slug == models.RoleSuperAdmin,
	}
}

// ServerOption is an option to apply to the API server.
type ServerOption func(*Server)

// Server represents the routes and functionality of the Gophish API.
// It's not a server in the traditional sense, in that it isn't started and
// stopped. Rather, it's meant to be used as an http.Handler in the
// AdminServer.
type Server struct {
	handler  http.Handler
	worker   worker.Worker
	limiter  *ratelimit.PostLimiter
	aiConfig config.AIConfig
}

// NewServer returns a new instance of the API handler with the provided
// options applied.
func NewServer(options ...ServerOption) *Server {
	defaultWorker, _ := worker.New()
	defaultLimiter := ratelimit.NewPostLimiter()
	as := &Server{
		worker:  defaultWorker,
		limiter: defaultLimiter,
	}
	for _, opt := range options {
		opt(as)
	}
	as.registerRoutes()
	return as
}

// WithWorker is an option that sets the background worker.
func WithWorker(w worker.Worker) ServerOption {
	return func(as *Server) {
		as.worker = w
	}
}

func WithLimiter(limiter *ratelimit.PostLimiter) ServerOption {
	return func(as *Server) {
		as.limiter = limiter
	}
}

// WithAIConfig sets the AI provider configuration for template generation.
func WithAIConfig(cfg config.AIConfig) ServerOption {
	return func(as *Server) {
		as.aiConfig = cfg
	}
}

// MCPQueryRequest represents the request body for the MCP API.
type MCPQueryRequest struct {
	Query string `json:"query"`
}

// MCPQueryResponse represents the response body for the MCP API.
type MCPQueryResponse struct {
	Result string `json:"result"`
}

// MCPQueryHandler handles MCP API queries for Claude or other LLMs.
func (as *Server) MCPQueryHandler(w http.ResponseWriter, r *http.Request) {
	var req MCPQueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}
	// TODO: Replace this with real Claude/LLM integration
	resp := MCPQueryResponse{Result: "Echo: " + req.Query}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func (as *Server) registerRoutes() {
	root := mux.NewRouter()
	root = root.StrictSlash(true)
	router := root.PathPrefix("/api/").Subrouter()
	router.Use(mid.RequireAPIKey)
	router.Use(mid.EnforceViewOnly)
	// Apply a generous rate limit to all API endpoints (30 req/min per IP).
	apiLimiter := ratelimit.NewPostLimiter(ratelimit.WithRequestsPerMinute(30))
	router.Use(func(next http.Handler) http.Handler {
		return apiLimiter.LimitAll(next)
	})
	router.HandleFunc("/imap/", as.IMAPServer)
	router.HandleFunc("/imap/validate", as.IMAPServerValidate)
	router.HandleFunc("/reset", as.Reset)
	router.HandleFunc("/campaigns/", mid.Use(as.Campaigns, mid.EnforceTierLimits("campaign")))
	router.HandleFunc("/campaigns/summary", as.CampaignsSummary)
	router.HandleFunc("/campaigns/{id:[0-9]+}", as.Campaign)
	router.HandleFunc("/campaigns/{id:[0-9]+}/results", as.CampaignResults)
	router.HandleFunc("/campaigns/{id:[0-9]+}/summary", as.CampaignSummary)
	router.HandleFunc("/campaigns/{id:[0-9]+}/complete", as.CampaignComplete)
	// Campaign advanced analytics
	router.HandleFunc("/campaigns/{id:[0-9]+}/analytics/funnel", as.CampaignAnalyticsFunnel)
	router.HandleFunc("/campaigns/{id:[0-9]+}/analytics/time-to-click", as.CampaignAnalyticsTimeToClick)
	router.HandleFunc("/campaigns/{id:[0-9]+}/analytics/repeat-offenders", as.CampaignAnalyticsRepeatOffenders)
	router.HandleFunc("/campaigns/{id:[0-9]+}/analytics/devices", as.CampaignAnalyticsDeviceBreakdown)
	router.HandleFunc("/groups/", as.Groups)
	router.HandleFunc("/groups/summary", as.GroupsSummary)
	router.HandleFunc("/groups/{id:[0-9]+}", as.Group)
	router.HandleFunc("/groups/{id:[0-9]+}/summary", as.GroupSummary)
	router.HandleFunc("/templates/", as.Templates)
	router.HandleFunc("/templates/{id:[0-9]+}", as.Template)
	router.HandleFunc("/template-library/", as.TemplateLibraryList)
	router.HandleFunc("/template-library/categories", as.TemplateLibraryCategories)
	router.HandleFunc("/template-library/{slug}/import", mid.Use(as.TemplateLibraryImport, mid.RequirePermission(models.PermissionModifyObjects)))
	router.HandleFunc("/pages/", as.Pages)
	router.HandleFunc("/pages/{id:[0-9]+}", as.Page)
	router.HandleFunc("/smtp/", as.SendingProfiles)
	router.HandleFunc("/smtp/{id:[0-9]+}", as.SendingProfile)
	router.HandleFunc("/sms/", as.SMSProviders)
	router.HandleFunc("/sms/{id:[0-9]+}", as.SMSProvider)
	router.HandleFunc("/users/", mid.Use(as.Users, mid.EnforceTierLimits("user"), mid.RequirePermission(models.PermissionModifySystem)))
	router.HandleFunc("/users/{id:[0-9]+}", mid.Use(as.User))
	router.HandleFunc("/roles/", as.Roles)
	router.HandleFunc("/tiers/", as.Tiers)
	router.HandleFunc("/org/features", as.OrgFeatures)
	router.HandleFunc("/audit-log", mid.Use(as.AuditLog, mid.RequirePermission(models.PermissionViewReports)))
	router.HandleFunc("/util/send_test_email", as.SendTestEmail)
	router.HandleFunc("/import/group", as.ImportGroup)
	router.HandleFunc("/import/email", as.ImportEmail)
	router.HandleFunc("/import/site", as.ImportSite)
	router.HandleFunc("/webhooks/", mid.Use(as.Webhooks, mid.RequirePermission(models.PermissionModifySystem)))
	router.HandleFunc("/webhooks/{id:[0-9]+}/validate", mid.Use(as.ValidateWebhook, mid.RequirePermission(models.PermissionModifySystem)))
	router.HandleFunc("/webhooks/{id:[0-9]+}", mid.Use(as.Webhook, mid.RequirePermission(models.PermissionModifySystem)))
	// Training presentation routes - GET accessible to all, POST/PUT/DELETE admin only
	router.HandleFunc("/training/", as.TrainingPresentations)
	router.HandleFunc("/training/extract-slides", as.TrainingExtractSlidesUpload)
	router.HandleFunc("/training/my-courses", as.TrainingMyCourses)
	router.HandleFunc("/training/content-library", as.ContentLibrary)
	router.HandleFunc("/training/content-library/detail", as.ContentLibraryDetail)
	router.HandleFunc("/training/content-library/categories", as.ContentLibraryCategories)
	router.HandleFunc("/training/content-library/seed", as.ContentLibrarySeed)
	router.HandleFunc("/training/content-library/seed-single", as.ContentLibrarySeedSingle)
	router.HandleFunc("/training/{id:[0-9]+}", as.TrainingPresentation)
	router.HandleFunc("/training/{id:[0-9]+}/download", as.TrainingPresentationDownload)
	router.HandleFunc("/training/{id:[0-9]+}/thumbnail", as.TrainingPresentationThumbnail)
	router.HandleFunc("/training/{id:[0-9]+}/extract-slides", as.TrainingExtractSlidesExisting)
	router.HandleFunc("/training/{id:[0-9]+}/progress", as.TrainingCourseProgress)
	router.HandleFunc("/training/{id:[0-9]+}/rate", as.TrainingSatisfactionRate)
	router.HandleFunc("/training/satisfaction", as.TrainingSatisfactionStats)
	router.HandleFunc("/training/analytics", as.TrainingAnalytics)
	// Quiz routes
	router.HandleFunc("/training/{id:[0-9]+}/quiz", as.TrainingQuiz)
	router.HandleFunc("/training/{id:[0-9]+}/quiz/attempt", as.TrainingQuizAttempt)
	// Branching narrative scenarios (interactive training)
	router.HandleFunc("/scenarios/", as.TrainingScenarios)
	router.HandleFunc("/scenarios/{id:[0-9]+}", as.TrainingScenario)
	router.HandleFunc("/scenarios/{id:[0-9]+}/start", as.TrainingScenarioStart)
	router.HandleFunc("/scenarios/{id:[0-9]+}/choose", as.TrainingScenarioChoose)
	router.HandleFunc("/scenarios/{id:[0-9]+}/history", as.TrainingScenarioHistory)
	// Custom Training Builder (All-in-One tier) — multi-asset course modules
	router.HandleFunc("/training/{id:[0-9]+}/assets", mid.Use(as.TrainingAssets, mid.RequireFeature(models.FeatureCustomTrainingBuilder)))
	router.HandleFunc("/training/{id:[0-9]+}/assets/reorder", mid.Use(as.TrainingAssetReorder, mid.RequirePermission(models.PermissionManageTraining), mid.RequireFeature(models.FeatureCustomTrainingBuilder)))
	router.HandleFunc("/training/assets/{id:[0-9]+}", mid.Use(as.TrainingAsset, mid.RequireFeature(models.FeatureCustomTrainingBuilder)))
	// Assignment routes
	router.HandleFunc("/training/assignments/", as.TrainingAssignments)
	router.HandleFunc("/training/assignments/group", as.TrainingAssignGroup)
	router.HandleFunc("/training/assignments/summary", as.TrainingAssignmentSummary)
	router.HandleFunc("/training/assignments/bulk-update", as.TrainingBulkAssignmentUpdate)
	router.HandleFunc("/training/assignments/mark-overdue", as.TrainingMarkOverdue)
	router.HandleFunc("/training/assignments/{id:[0-9]+}", as.TrainingAssignment)
	router.HandleFunc("/training/my-assignments", as.TrainingMyAssignments)
	// Anti-skip protection routes
	router.HandleFunc("/training/{id:[0-9]+}/anti-skip-policy", as.AntiSkipPolicy)
	router.HandleFunc("/training/{id:[0-9]+}/engage", as.AntiSkipEngage)
	router.HandleFunc("/training/{id:[0-9]+}/validate-advance", as.AntiSkipValidateAdvance)
	router.HandleFunc("/training/{id:[0-9]+}/validate-complete", as.AntiSkipValidateComplete)
	router.HandleFunc("/training/{id:[0-9]+}/engagement-summary", mid.Use(as.AntiSkipEngagementSummary, mid.RequirePermission(models.PermissionManageTraining)))
	// Certificate routes
	router.HandleFunc("/training/my-certificates", as.TrainingMyCertificates)
	router.HandleFunc("/training/certificates/templates", as.TrainingCertificateTemplates)
	router.HandleFunc("/training/certificates/issue", as.TrainingCertificateIssue)
	router.HandleFunc("/training/certificates/summary", as.TrainingCertificateSummary)
	router.HandleFunc("/training/certificates/verify/{code}", as.TrainingCertificateVerify)
	router.HandleFunc("/training/certificates/{id:[0-9]+}/revoke", as.TrainingCertificateRevoke)
	router.HandleFunc("/training/certificates/{id:[0-9]+}/renew", as.TrainingCertificateRenew)
	// Praise / feedback message routes
	router.HandleFunc("/training/praise-messages", as.TrainingPraiseMessages)
	router.HandleFunc("/training/praise-messages/reset", mid.Use(as.TrainingPraiseMessagesReset, mid.RequirePermission(models.PermissionManageTraining)))
	// Report routes
	router.HandleFunc("/reports/overview", mid.Use(as.ReportOverview, mid.RequireReportAccess))
	router.HandleFunc("/reports/trend", mid.Use(as.ReportTrend, mid.RequireReportAccess))
	router.HandleFunc("/reports/risk-scores", mid.Use(as.ReportRiskScores, mid.RequireReportAccess))
	router.HandleFunc("/reports/training-summary", mid.Use(as.ReportTrainingSummary, mid.RequireReportAccess))
	router.HandleFunc("/reports/group-comparison", mid.Use(as.ReportGroupComparison, mid.RequireReportAccess))
	router.HandleFunc("/reports/export", mid.Use(as.ReportExport, mid.RequireReportAccess))
	// BRS (Behavioral Risk Score) endpoints
	router.HandleFunc("/reports/brs/user/{id:[0-9]+}", mid.Use(as.BRSUserDetail, mid.RequireReportAccess))
	router.HandleFunc("/reports/brs/department", mid.Use(as.BRSDepartment, mid.RequireReportAccess, mid.RequireFeature(models.FeatureAdvancedBRS)))
	router.HandleFunc("/reports/brs/benchmark", mid.Use(as.BRSBenchmark, mid.RequireReportAccess, mid.RequireFeature(models.FeatureAdvancedBRS)))
	router.HandleFunc("/reports/brs/trend", mid.Use(as.BRSTrend, mid.RequireReportAccess))
	router.HandleFunc("/reports/brs/leaderboard", mid.Use(as.BRSLeaderboard, mid.RequireReportAccess))
	router.HandleFunc("/reports/brs/recalculate", mid.Use(as.BRSRecalculate, mid.RequirePermission(models.PermissionModifySystem)))
	// Board-ready report routes (feature-gated)
	router.HandleFunc("/board-reports/", mid.Use(as.BoardReports, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeatureBoardReports)))
	router.HandleFunc("/board-reports/generate", mid.Use(as.BoardReportGenerate, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeatureBoardReports)))
	router.HandleFunc("/board-reports/{id:[0-9]+}", mid.Use(as.BoardReport, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeatureBoardReports)))
	router.HandleFunc("/board-reports/{id:[0-9]+}/export", mid.Use(as.BoardReportExport, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeatureBoardReports)))
	router.HandleFunc("/board-reports/{id:[0-9]+}/export-branded", mid.Use(as.BoardReportExportBranded, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeatureBoardReports)))
	router.HandleFunc("/board-reports/schedules", mid.Use(as.BoardReportSchedules, mid.RequirePermission(models.PermissionModifySystem), mid.RequireFeature(models.FeatureBoardReports)))
	router.HandleFunc("/board-reports/schedules/run", mid.Use(as.BoardReportScheduleRun, mid.RequirePermission(models.PermissionModifySystem), mid.RequireFeature(models.FeatureBoardReports)))
	router.HandleFunc("/board-reports/schedules/{id:[0-9]+}", mid.Use(as.BoardReportScheduleItem, mid.RequirePermission(models.PermissionModifySystem), mid.RequireFeature(models.FeatureBoardReports)))
	// ROI Reporting routes
	router.HandleFunc("/roi/config", mid.Use(as.ROIConfig, mid.RequirePermission(models.PermissionViewReports)))
	router.HandleFunc("/roi/generate", mid.Use(as.ROIGenerate, mid.RequirePermission(models.PermissionViewReports)))
	router.HandleFunc("/roi/generate-and-save", mid.Use(as.ROIGenerateAndSave, mid.RequirePermission(models.PermissionViewReports)))
	router.HandleFunc("/roi/export", mid.Use(as.ROIExport, mid.RequirePermission(models.PermissionViewReports)))
	router.HandleFunc("/roi/export-pdf", mid.Use(as.ROIExportEnhancedPDF, mid.RequirePermission(models.PermissionViewReports)))
	router.HandleFunc("/roi/benchmarks", mid.Use(as.ROIBenchmarks, mid.RequirePermission(models.PermissionViewReports)))
	router.HandleFunc("/roi/benchmarks/{id:[0-9]+}", mid.Use(as.ROIBenchmarkItem, mid.RequirePermission(models.PermissionModifySystem)))
	router.HandleFunc("/roi/benchmarks/seed", mid.Use(as.ROIBenchmarkSeed, mid.RequirePermission(models.PermissionModifySystem)))
	router.HandleFunc("/roi/benchmarks/compare", mid.Use(as.ROIBenchmarkCompare, mid.RequirePermission(models.PermissionViewReports)))
	router.HandleFunc("/roi/monte-carlo", mid.Use(as.ROIMonteCarlo, mid.RequirePermission(models.PermissionViewReports)))
	router.HandleFunc("/roi/history", mid.Use(as.ROIHistory, mid.RequirePermission(models.PermissionViewReports)))
	router.HandleFunc("/roi/history/{id:[0-9]+}", mid.Use(as.ROIHistoryItem, mid.RequirePermission(models.PermissionViewReports)))
	router.HandleFunc("/roi/trend", mid.Use(as.ROIQuarterlyTrend, mid.RequirePermission(models.PermissionViewReports)))
	// Adaptive targeting
	router.HandleFunc("/targeting/profile/{id:[0-9]+}", mid.Use(as.TargetingProfile, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureAutopilot)))
	// Adaptive difficulty routes
	router.HandleFunc("/difficulty/profile/{id:[0-9]+}", mid.Use(as.DifficultyProfile, mid.RequirePermission(models.PermissionModifyObjects)))
	router.HandleFunc("/difficulty/set/{id:[0-9]+}", mid.Use(as.DifficultySet, mid.RequirePermission(models.PermissionModifyObjects)))
	router.HandleFunc("/difficulty/clear/{id:[0-9]+}", mid.Use(as.DifficultyClear, mid.RequirePermission(models.PermissionModifyObjects)))
	router.HandleFunc("/difficulty/history/{id:[0-9]+}", mid.Use(as.DifficultyHistory, mid.RequirePermission(models.PermissionModifyObjects)))
	router.HandleFunc("/difficulty/org-stats", mid.Use(as.DifficultyOrgStats, mid.RequirePermission(models.PermissionViewReports)))
	router.HandleFunc("/difficulty/my-profile", as.DifficultyMyProfile)
	router.HandleFunc("/difficulty/my-set", as.DifficultyMySet)
	router.HandleFunc("/difficulty/my-clear", as.DifficultyMyClear)
	router.HandleFunc("/difficulty/run-adaptive", mid.Use(as.DifficultyRunAdaptive, mid.RequirePermission(models.PermissionModifySystem)))
	// Adaptive engine management routes
	router.HandleFunc("/adaptive-engine/config", mid.Use(as.AdaptiveEngineConfig, mid.RequirePermission(models.PermissionModifySystem)))
	router.HandleFunc("/adaptive-engine/summary", mid.Use(as.AdaptiveEngineSummary, mid.RequirePermission(models.PermissionViewReports)))
	router.HandleFunc("/adaptive-engine/history", mid.Use(as.AdaptiveEngineHistory, mid.RequirePermission(models.PermissionViewReports)))
	router.HandleFunc("/adaptive-engine/run", mid.Use(as.AdaptiveEngineRun, mid.RequirePermission(models.PermissionModifySystem)))
	// Nanolearning analytics routes
	router.HandleFunc("/nanolearning/stats", mid.Use(as.NanolearningStats, mid.RequirePermission(models.PermissionViewReports)))
	router.HandleFunc("/nanolearning/events", as.NanolearningEvents)
	// Automated training reminder routes
	router.HandleFunc("/reminders/config", mid.Use(as.ReminderConfig, mid.RequirePermission(models.PermissionModifySystem)))
	router.HandleFunc("/reminders/stats", mid.Use(as.ReminderStats, mid.RequirePermission(models.PermissionViewReports)))
	router.HandleFunc("/reminders/history", as.ReminderHistory)
	// Content auto-update routes
	router.HandleFunc("/content-updates/config", mid.Use(as.ContentUpdateConfig, mid.RequirePermission(models.PermissionModifySystem)))
	router.HandleFunc("/content-updates/summary", mid.Use(as.ContentUpdateSummary, mid.RequirePermission(models.PermissionViewReports)))
	router.HandleFunc("/content-updates/history", mid.Use(as.ContentUpdateHistory, mid.RequirePermission(models.PermissionViewReports)))
	// AI template generation routes
	router.HandleFunc("/ai/generate-template", mid.Use(as.AIGenerateTemplate, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureAITemplates)))
	router.HandleFunc("/ai/usage", mid.Use(as.AIUsage, mid.RequireFeature(models.FeatureAITemplates)))
	// Autopilot routes
	router.HandleFunc("/autopilot/config", mid.Use(as.AutopilotConfig, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureAutopilot)))
	router.HandleFunc("/autopilot/enable", mid.Use(as.AutopilotEnable, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureAutopilot)))
	router.HandleFunc("/autopilot/disable", mid.Use(as.AutopilotDisable, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureAutopilot)))
	router.HandleFunc("/autopilot/schedule", mid.Use(as.AutopilotSchedule, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureAutopilot)))
	router.HandleFunc("/autopilot/blackout", mid.Use(as.AutopilotBlackoutDates, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureAutopilot)))
	router.HandleFunc("/autopilot/blackout/{id:[0-9]+}", mid.Use(as.AutopilotBlackoutDate, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureAutopilot)))
	// Academy routes
	router.HandleFunc("/academy/tiers", as.AcademyTiers)
	router.HandleFunc("/academy/tiers/{slug}/sessions", as.AcademyTierSessions)
	router.HandleFunc("/academy/tiers/{slug}/complete", as.AcademyTierComplete)
	router.HandleFunc("/academy/my-progress", as.AcademyMyProgress)
	router.HandleFunc("/academy/sessions", mid.Use(as.AcademySessionManage, mid.RequirePermission(models.PermissionManageTraining)))
	router.HandleFunc("/academy/sessions/{id:[0-9]+}", mid.Use(as.AcademySessionDelete, mid.RequirePermission(models.PermissionManageTraining)))
	router.HandleFunc("/academy/compliance", as.ComplianceCertifications)
	router.HandleFunc("/academy/compliance/{id:[0-9]+}/complete", as.ComplianceCertComplete)
	router.HandleFunc("/academy/compliance/my-certs", as.ComplianceMyCerts)
	router.HandleFunc("/academy/compliance/verify/{code}", as.ComplianceCertVerify)
	// Gamification routes
	router.HandleFunc("/gamification/leaderboard", mid.Use(as.GamificationLeaderboard, mid.RequireFeature(models.FeatureGamification)))
	router.HandleFunc("/gamification/my-position", mid.Use(as.GamificationMyPosition, mid.RequireFeature(models.FeatureGamification)))
	router.HandleFunc("/gamification/badges", mid.Use(as.GamificationBadges, mid.RequireFeature(models.FeatureGamification)))
	router.HandleFunc("/gamification/my-badges", mid.Use(as.GamificationMyBadges, mid.RequireFeature(models.FeatureGamification)))
	router.HandleFunc("/gamification/my-streak", mid.Use(as.GamificationMyStreak, mid.RequireFeature(models.FeatureGamification)))
	// Organization management routes
	router.HandleFunc("/orgs/", mid.Use(as.Orgs, mid.RequirePermission(models.PermissionModifySystem)))
	router.HandleFunc("/orgs/{id:[0-9]+}", mid.Use(as.Org, mid.RequirePermission(models.PermissionModifySystem)))
	router.HandleFunc("/orgs/{id:[0-9]+}/members", mid.Use(as.OrgMembers, mid.RequirePermission(models.PermissionModifySystem)))
	router.HandleFunc("/orgs/{id:[0-9]+}/members/{uid:[0-9]+}", mid.Use(as.OrgMember, mid.RequirePermission(models.PermissionModifySystem)))
	// Report button admin routes
	router.HandleFunc("/report-button/config", mid.Use(as.ReportButtonConfig, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureReportButton)))
	router.HandleFunc("/report-button/regenerate-key", mid.Use(as.ReportButtonRegenerateKey, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureReportButton)))
	router.HandleFunc("/reported-emails", mid.Use(as.ReportedEmails, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureReportButton)))
	router.HandleFunc("/reported-emails/{id:[0-9]+}/classify", mid.Use(as.ReportedEmailClassify, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureReportButton)))
	// Threat alert routes
	router.HandleFunc("/threat-alerts", as.ThreatAlerts)
	router.HandleFunc("/threat-alerts/unread-count", as.ThreatAlertUnreadCount)
	router.HandleFunc("/threat-alerts/create", mid.Use(as.ThreatAlertCreate, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureThreatAlertsCreate)))
	router.HandleFunc("/threat-alerts/{id:[0-9]+}", as.ThreatAlert)
	// Escalation workflow routes
	router.HandleFunc("/escalation/policies", mid.Use(as.EscalationPolicies, mid.RequirePermission(models.PermissionModifyObjects)))
	router.HandleFunc("/escalation/policies/{id:[0-9]+}", mid.Use(as.EscalationPolicy, mid.RequirePermission(models.PermissionModifyObjects)))
	router.HandleFunc("/escalation/offenders", mid.Use(as.EscalationOffenders, mid.RequireReportAccess))
	router.HandleFunc("/escalation/evaluate", mid.Use(as.EscalationEvaluate, mid.RequirePermission(models.PermissionModifyObjects)))
	router.HandleFunc("/escalation/events", mid.Use(as.EscalationEvents, mid.RequireReportAccess))
	router.HandleFunc("/escalation/events/{id:[0-9]+}/resolve", mid.Use(as.EscalationResolve, mid.RequirePermission(models.PermissionModifyObjects)))
	router.HandleFunc("/escalation/summary", mid.Use(as.EscalationDashboard, mid.RequireReportAccess))
	// Power BI / OData feed
	router.HandleFunc("/powerbi/", mid.Use(as.PowerBIFeed, mid.RequireReportAccess, mid.RequireFeature(models.FeaturePowerBI)))
	// Compliance framework mapping routes
	router.HandleFunc("/compliance/frameworks", mid.Use(as.ComplianceFrameworks, mid.RequireFeature(models.FeatureComplianceMapping)))
	router.HandleFunc("/compliance/org-frameworks", mid.Use(as.ComplianceOrgFrameworks, mid.RequirePermission(models.PermissionModifySystem), mid.RequireFeature(models.FeatureComplianceMapping)))
	router.HandleFunc("/compliance/org-frameworks/{id:[0-9]+}/disable", mid.Use(as.ComplianceOrgFrameworkDisable, mid.RequirePermission(models.PermissionModifySystem), mid.RequireFeature(models.FeatureComplianceMapping)))
	router.HandleFunc("/compliance/dashboard", mid.Use(as.ComplianceDashboard, mid.RequireFeature(models.FeatureComplianceMapping)))
	router.HandleFunc("/compliance/frameworks/{id:[0-9]+}/detail", mid.Use(as.ComplianceFrameworkDetail, mid.RequireFeature(models.FeatureComplianceMapping)))
	router.HandleFunc("/compliance/frameworks/{id:[0-9]+}/assess", mid.Use(as.ComplianceAssess, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureComplianceMapping)))
	router.HandleFunc("/compliance/controls/{id:[0-9]+}/assess", mid.Use(as.ComplianceManualAssess, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureComplianceMapping)))
	// Framework compliance certificates (auto-issued org-level certs)
	router.HandleFunc("/compliance/framework-certs/definitions", mid.Use(as.FrameworkCertDefinitions, mid.RequireFeature(models.FeatureComplianceMapping)))
	router.HandleFunc("/compliance/framework-certs", mid.Use(as.FrameworkCertOrgCerts, mid.RequireFeature(models.FeatureComplianceMapping)))
	router.HandleFunc("/compliance/framework-certs/evaluate", mid.Use(as.FrameworkCertEvaluate, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureComplianceMapping)))
	router.HandleFunc("/compliance/framework-certs/summary", mid.Use(as.FrameworkCertSummary, mid.RequireFeature(models.FeatureComplianceMapping)))
	router.HandleFunc("/compliance/framework-certs/verify/{code}", mid.Use(as.FrameworkCertVerify, mid.RequireFeature(models.FeatureComplianceMapping)))
	router.HandleFunc("/compliance/framework-certs/{id:[0-9]+}/revoke", mid.Use(as.FrameworkCertRevoke, mid.RequirePermission(models.PermissionModifySystem), mid.RequireFeature(models.FeatureComplianceMapping)))
	// Compliance training modules (framework-specific microlearning)
	router.HandleFunc("/compliance/training-modules", mid.Use(as.ComplianceTrainingModules, mid.RequireFeature(models.FeatureComplianceMapping)))
	router.HandleFunc("/compliance/training-modules/{slug}", mid.Use(as.ComplianceTrainingModule, mid.RequireFeature(models.FeatureComplianceMapping)))
	// Platform certifications and security posture
	router.HandleFunc("/compliance/platform-certifications", as.PlatformCertifications)
	router.HandleFunc("/compliance/platform-certifications/{slug}", as.PlatformCertification)
	router.HandleFunc("/compliance/platform-security-posture", as.PlatformSecurityPosture)
	router.HandleFunc("/compliance/platform-support", mid.Use(as.PlatformComplianceSupport, mid.RequireFeature(models.FeatureComplianceMapping)))
	// Feedback page routes (micro-feedback interstitial)
	router.HandleFunc("/feedback_pages/", mid.Use(as.FeedbackPages, mid.RequirePermission(models.PermissionModifyObjects)))
	router.HandleFunc("/feedback_pages/default", mid.Use(as.FeedbackPageDefault, mid.RequirePermission(models.PermissionModifyObjects)))
	router.HandleFunc("/feedback_pages/{id:[0-9]+}", mid.Use(as.FeedbackPage, mid.RequirePermission(models.PermissionModifyObjects)))
	// i18n routes
	router.HandleFunc("/i18n/languages", as.I18nLanguages)
	router.HandleFunc("/i18n/{locale}", as.I18nTranslations)
	router.HandleFunc("/user/language", as.UserLanguage)
	// Cyber Hygiene: My Apps & Devices routes
	router.HandleFunc("/hygiene/devices/", mid.Use(as.HygieneDevices, mid.RequireFeature(models.FeatureCyberHygiene)))
	router.HandleFunc("/hygiene/devices/{id:[0-9]+}", mid.Use(as.HygieneDevice, mid.RequireFeature(models.FeatureCyberHygiene)))
	router.HandleFunc("/hygiene/devices/{id:[0-9]+}/checks", mid.Use(as.HygieneDeviceChecks, mid.RequireFeature(models.FeatureCyberHygiene)))
	router.HandleFunc("/hygiene/tech-stack", mid.Use(as.HygieneTechStack, mid.RequireFeature(models.FeatureCyberHygiene)))
	router.HandleFunc("/hygiene/personalized-checks", mid.Use(as.HygienePersonalizedChecks, mid.RequireFeature(models.FeatureCyberHygiene)))
	router.HandleFunc("/hygiene/admin/devices", mid.Use(as.HygieneAdminDevices, mid.RequirePermission(models.PermissionModifySystem), mid.RequireFeature(models.FeatureCyberHygiene)))
	router.HandleFunc("/hygiene/admin/devices-enriched", mid.Use(as.HygieneAdminDevicesEnriched, mid.RequirePermission(models.PermissionModifySystem), mid.RequireFeature(models.FeatureCyberHygiene)))
	router.HandleFunc("/hygiene/admin/summary", mid.Use(as.HygieneAdminSummary, mid.RequirePermission(models.PermissionModifySystem), mid.RequireFeature(models.FeatureCyberHygiene)))
	router.HandleFunc("/hygiene/export", mid.Use(as.HygieneExport, mid.RequirePermission(models.PermissionModifySystem), mid.RequireFeature(models.FeatureCyberHygiene)))
	// Repeat offender remediation path routes
	router.HandleFunc("/remediation/paths", mid.Use(as.RemediationPaths, mid.RequirePermission(models.PermissionModifyObjects)))
	router.HandleFunc("/remediation/paths/{id:[0-9]+}", mid.Use(as.RemediationPath, mid.RequirePermission(models.PermissionModifyObjects)))
	router.HandleFunc("/remediation/paths/{id:[0-9]+}/complete-step", mid.Use(as.RemediationCompleteStep, mid.RequirePermission(models.PermissionModifyObjects)))
	router.HandleFunc("/remediation/my-paths", as.RemediationMyPaths)
	router.HandleFunc("/remediation/evaluate", mid.Use(as.RemediationEvaluate, mid.RequirePermission(models.PermissionModifyObjects)))
	router.HandleFunc("/remediation/summary", mid.Use(as.RemediationSummary, mid.RequireReportAccess))
	router.HandleFunc("/remediation/export", mid.Use(as.RemediationExport, mid.RequireReportAccess))
	router.HandleFunc("/remediation/mark-expired", mid.Use(as.RemediationMarkExpired, mid.RequirePermission(models.PermissionModifyObjects)))
	// Zero Incident Mail (ZIM) sandbox routes
	router.HandleFunc("/sandbox/", mid.Use(as.SandboxTests, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureZIM)))
	router.HandleFunc("/sandbox/{id:[0-9]+}", mid.Use(as.SandboxTest, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureZIM)))
	router.HandleFunc("/sandbox/{id:[0-9]+}/review", mid.Use(as.SandboxTestReview, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureZIM)))
	// NLP Email Analysis routes (AI-powered semantic analysis of reported emails)
	router.HandleFunc("/email-analysis/", mid.Use(as.EmailAnalyses, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureNLPEmailAnalysis)))
	router.HandleFunc("/email-analysis/analyze", mid.Use(as.EmailAnalyzeReported, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureNLPEmailAnalysis)))
	router.HandleFunc("/email-analysis/summary", mid.Use(as.EmailAnalysisSummary, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeatureNLPEmailAnalysis)))
	router.HandleFunc("/email-analysis/by-reported/{id:[0-9]+}", mid.Use(as.EmailAnalysisByReported, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureNLPEmailAnalysis)))
	router.HandleFunc("/email-analysis/{id:[0-9]+}", mid.Use(as.EmailAnalysis, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureNLPEmailAnalysis)))
	// Real-time Inbox Monitor routes (AI inbox analysis)
	router.HandleFunc("/inbox-monitor/config", mid.Use(as.InboxMonitorConfig, mid.RequirePermission(models.PermissionModifySystem), mid.RequireFeature(models.FeatureInboxMonitor)))
	router.HandleFunc("/inbox-monitor/results", mid.Use(as.InboxScanResults, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureInboxMonitor)))
	router.HandleFunc("/inbox-monitor/results/{id:[0-9]+}", mid.Use(as.InboxScanResultDetail, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureInboxMonitor)))
	router.HandleFunc("/inbox-monitor/summary", mid.Use(as.InboxScanSummary, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeatureInboxMonitor)))
	// BEC Detection routes
	router.HandleFunc("/bec/profiles", mid.Use(as.BECProfiles, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureBECDetection)))
	router.HandleFunc("/bec/profiles/{id:[0-9]+}", mid.Use(as.BECProfile, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureBECDetection)))
	router.HandleFunc("/bec/detections", mid.Use(as.BECDetections, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureBECDetection)))
	router.HandleFunc("/bec/detections/{id:[0-9]+}/resolve", mid.Use(as.BECDetectionResolve, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureBECDetection)))
	router.HandleFunc("/bec/summary", mid.Use(as.BECDetectionSummary, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeatureBECDetection)))
	router.HandleFunc("/bec/analyze", mid.Use(as.BECAnalyzeEmail, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureBECDetection)))
	// Graymail Classification routes
	router.HandleFunc("/graymail/rules", mid.Use(as.GraymailRules, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureGraymailClassification)))
	router.HandleFunc("/graymail/rules/{id:[0-9]+}", mid.Use(as.GraymailRule, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureGraymailClassification)))
	router.HandleFunc("/graymail/classifications", mid.Use(as.GraymailClassifications, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeatureGraymailClassification)))
	router.HandleFunc("/graymail/summary", mid.Use(as.GraymailSummary, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeatureGraymailClassification)))
	router.HandleFunc("/graymail/analyze", mid.Use(as.GraymailAnalyze, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureGraymailClassification)))
	// One-Click Inbox Remediation routes
	router.HandleFunc("/remediation-actions/", mid.Use(as.InboxRemediationActions, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureOneClickRemediation)))
	router.HandleFunc("/remediation-actions/create", mid.Use(as.InboxRemediationActionCreate, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureOneClickRemediation)))
	router.HandleFunc("/remediation-actions/summary", mid.Use(as.InboxRemediationSummary, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeatureOneClickRemediation)))
	router.HandleFunc("/remediation-actions/{id:[0-9]+}/approve", mid.Use(as.InboxRemediationActionApprove, mid.RequirePermission(models.PermissionModifySystem), mid.RequireFeature(models.FeatureOneClickRemediation)))
	router.HandleFunc("/remediation-actions/{id:[0-9]+}/reject", mid.Use(as.InboxRemediationActionReject, mid.RequirePermission(models.PermissionModifySystem), mid.RequireFeature(models.FeatureOneClickRemediation)))
	// Phishing Ticket Management routes
	router.HandleFunc("/phishing-tickets/", mid.Use(as.PhishingTickets, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeaturePhishingTickets)))
	router.HandleFunc("/phishing-tickets/summary", mid.Use(as.PhishingTicketSummary, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeaturePhishingTickets)))
	router.HandleFunc("/phishing-tickets/auto-rules", mid.Use(as.PhishingTicketAutoRules, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeaturePhishingTickets)))
	router.HandleFunc("/phishing-tickets/auto-rules/{id:[0-9]+}", mid.Use(as.PhishingTicketAutoRuleDelete, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeaturePhishingTickets)))
	router.HandleFunc("/phishing-tickets/{id:[0-9]+}", mid.Use(as.PhishingTicketDetail, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeaturePhishingTickets)))
	router.HandleFunc("/phishing-tickets/{id:[0-9]+}/resolve", mid.Use(as.PhishingTicketResolve, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeaturePhishingTickets)))
	router.HandleFunc("/phishing-tickets/{id:[0-9]+}/escalate", mid.Use(as.PhishingTicketEscalate, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeaturePhishingTickets)))
	// Email Security Dashboard (unified view)
	router.HandleFunc("/email-security/dashboard", mid.Use(as.EmailSecurityDashboard, mid.RequirePermission(models.PermissionViewReports)))
	router.HandleFunc("/email-security/ops", mid.Use(as.EmailSecurityOps, mid.RequirePermission(models.PermissionViewReports)))
	// SCIM token management (admin API, uses normal API key auth)
	router.HandleFunc("/scim/tokens", mid.Use(as.SCIMTokens, mid.RequirePermission(models.PermissionModifySystem), mid.RequireFeature(models.FeatureSCIM)))
	router.HandleFunc("/scim/tokens/{id:[0-9]+}", mid.Use(as.SCIMToken, mid.RequirePermission(models.PermissionModifySystem), mid.RequireFeature(models.FeatureSCIM)))

	// ── MSP: Partner management (superadmin) ──
	router.HandleFunc("/msp/partners/", mid.Use(as.MSPPartners, mid.RequirePermission(models.PermissionModifySystem)))
	router.HandleFunc("/msp/partners/{id:[0-9]+}", mid.Use(as.MSPPartner, mid.RequirePermission(models.PermissionModifySystem)))
	// ── MSP: Partner-client management ──
	router.HandleFunc("/msp/partners/{id:[0-9]+}/clients", mid.Use(as.MSPPartnerClients, mid.RequireMSPPartner, mid.RequireFeature(models.FeatureMSPMultiClient)))
	router.HandleFunc("/msp/partners/{id:[0-9]+}/clients/{oid:[0-9]+}", mid.Use(as.MSPPartnerClientRemove, mid.RequireMSPPartner, mid.RequireFeature(models.FeatureMSPMultiClient)))
	// ── MSP: White-label branding ──
	router.HandleFunc("/msp/whitelabel/config", mid.Use(as.MSPWhiteLabelConfig, mid.RequireMSPWhitelabel))
	router.HandleFunc("/msp/whitelabel/{id:[0-9]+}", mid.Use(as.MSPWhiteLabelDelete, mid.RequirePermission(models.PermissionModifySystem), mid.RequireFeature(models.FeatureMSPWhitelabel)))
	router.HandleFunc("/msp/partners/{id:[0-9]+}/whitelabel", mid.Use(as.MSPWhiteLabelPartnerConfig, mid.RequireMSPPartner, mid.RequireFeature(models.FeatureMSPWhitelabel)))
	// ── MSP: Partner Portal ──
	router.HandleFunc("/msp/portal/dashboard", mid.Use(as.MSPPortalDashboard, mid.RequireMSPPartner, mid.RequireFeature(models.FeatureMSPPartnerPortal)))
	router.HandleFunc("/msp/portal/report", mid.Use(as.MSPPortalReport, mid.RequireMSPPartner, mid.RequireFeature(models.FeatureMSPPartnerPortal)))
	router.HandleFunc("/msp/portal/clients/{oid:[0-9]+}", mid.Use(as.MSPPortalClientDetail, mid.RequireMSPPartner, mid.RequireFeature(models.FeatureMSPPartnerPortal)))
	router.HandleFunc("/msp/portal/switch-org", mid.Use(as.MSPSwitchOrg, mid.RequireMSPPartner, mid.RequireFeature(models.FeatureMSPPartnerPortal)))
	router.HandleFunc("/msp/portal/ranking", mid.Use(as.MSPClientRanking, mid.RequireMSPPartner, mid.RequireFeature(models.FeatureMSPPartnerPortal)))
	router.HandleFunc("/msp/portal/comparison", mid.Use(as.MSPClientComparison, mid.RequireMSPPartner, mid.RequireFeature(models.FeatureMSPPartnerPortal)))
	router.HandleFunc("/msp/portal/billing", mid.Use(as.MSPBillingUsage, mid.RequireMSPPartner, mid.RequireFeature(models.FeatureMSPPartnerPortal)))
	router.HandleFunc("/msp/portal/quarterly-pdf", mid.Use(as.MSPQuarterlyPDF, mid.RequireMSPPartner, mid.RequireFeature(models.FeatureMSPPartnerPortal)))

	// Network Events Dashboard routes
	router.HandleFunc("/network-events/", mid.Use(as.NetworkEvents, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureNetworkEvents)))
	router.HandleFunc("/network-events/ingest", mid.Use(as.NetworkEventIngest, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureNetworkEvents)))
	router.HandleFunc("/network-events/bulk-ingest", mid.Use(as.NetworkEventBulkIngest, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureNetworkEvents)))
	router.HandleFunc("/network-events/dashboard", mid.Use(as.NetworkEventDashboard, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeatureNetworkEvents)))
	router.HandleFunc("/network-events/trend", mid.Use(as.NetworkEventTrend, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeatureNetworkEvents)))
	router.HandleFunc("/network-events/mitre-heatmap", mid.Use(as.NetworkEventMitreHeatmap, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeatureNetworkEvents)))
	router.HandleFunc("/network-events/correlate", mid.Use(as.NetworkEventCorrelate, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureNetworkEvents)))
	router.HandleFunc("/network-events/incidents", mid.Use(as.NetworkIncidents, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureNetworkEvents)))
	router.HandleFunc("/network-events/incidents/{id:[0-9]+}", mid.Use(as.NetworkIncident, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureNetworkEvents)))
	router.HandleFunc("/network-events/playbook-logs", mid.Use(as.NetworkEventPlaybookLogs, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeatureNetworkEvents)))
	router.HandleFunc("/network-events/rules", mid.Use(as.NetworkEventRules, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureNetworkEvents)))
	router.HandleFunc("/network-events/rules/{id:[0-9]+}", mid.Use(as.NetworkEventRule, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureNetworkEvents)))
	router.HandleFunc("/network-events/{id:[0-9]+}", mid.Use(as.NetworkEvent, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureNetworkEvents)))
	router.HandleFunc("/network-events/{id:[0-9]+}/notes", mid.Use(as.NetworkEventAddNote, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureNetworkEvents)))

	// ── Vishing (Voice Phishing) Simulation routes ──
	router.HandleFunc("/vishing/scenarios/", mid.Use(as.VishingScenarios, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureVishing)))
	router.HandleFunc("/vishing/scenarios/library", mid.Use(as.VishingScenarioLibrary, mid.RequireFeature(models.FeatureVishing)))
	router.HandleFunc("/vishing/scenarios/{id:[0-9]+}", mid.Use(as.VishingScenario, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureVishing)))
	router.HandleFunc("/vishing/campaigns/", mid.Use(as.VishingCampaigns, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureVishing)))
	router.HandleFunc("/vishing/campaigns/{id:[0-9]+}", mid.Use(as.VishingCampaign, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureVishing)))
	router.HandleFunc("/vishing/campaigns/{id:[0-9]+}/stats", mid.Use(as.VishingCampaignStats, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeatureVishing)))
	router.HandleFunc("/vishing/campaigns/{id:[0-9]+}/results", mid.Use(as.VishingResultRecord, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureVishing)))
	router.HandleFunc("/vishing/campaigns/{id:[0-9]+}/launch", mid.Use(as.VishingCampaignLaunch, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureVishing)))
	router.HandleFunc("/vishing/campaigns/{id:[0-9]+}/complete", mid.Use(as.VishingCampaignComplete, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureVishing)))
	router.HandleFunc("/vishing/stats", mid.Use(as.VishingOrgStats, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeatureVishing)))
	router.HandleFunc("/vishing/report", mid.Use(as.VishingReportCall, mid.RequireFeature(models.FeatureVishing)))

	// ── Inbox AI Feedback & Outlook/Gmail Add-in routes ──
	router.HandleFunc("/inbox/feedback", mid.Use(as.InboxFeedback, mid.RequireFeature(models.FeatureInboxAIFeedback)))
	router.HandleFunc("/inbox/feedback/unread-count", mid.Use(as.InboxFeedbackUnread, mid.RequireFeature(models.FeatureInboxAIFeedback)))
	router.HandleFunc("/inbox/feedback/{id:[0-9]+}/read", mid.Use(as.InboxFeedbackRead, mid.RequireFeature(models.FeatureInboxAIFeedback)))
	router.HandleFunc("/inbox/feedback/{id:[0-9]+}/acknowledge", mid.Use(as.InboxFeedbackAcknowledge, mid.RequireFeature(models.FeatureInboxAIFeedback)))
	router.HandleFunc("/inbox/addin/analyze", mid.Use(as.InboxAddInAnalyze, mid.RequireFeature(models.FeatureInboxAIFeedback)))
	router.HandleFunc("/inbox/ai-feedback", mid.Use(as.InboxAIFeedbackSubmit, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureInboxAIFeedback)))
	router.HandleFunc("/inbox/ai-accuracy", mid.Use(as.InboxAIAccuracy, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeatureInboxAIFeedback)))
	router.HandleFunc("/inbox/ai-recent-feedback", mid.Use(as.InboxAIRecentFeedback, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeatureInboxAIFeedback)))
	router.HandleFunc("/inbox/webhook", mid.Use(as.InboxWebhookConfig, mid.RequirePermission(models.PermissionModifySystem), mid.RequireFeature(models.FeatureInboxAIFeedback)))

	// ── Enhanced Board Reports (AI Narrative + ROI) ──
	router.HandleFunc("/board-reports/enhanced", mid.Use(as.BoardReportEnhanced, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeatureEnhancedBoardReports)))
	router.HandleFunc("/board-reports/{id:[0-9]+}/narrative", mid.Use(as.BoardReportNarrative, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeatureEnhancedBoardReports)))
	router.HandleFunc("/board-reports/{id:[0-9]+}/full", mid.Use(as.BoardReportFull, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeatureEnhancedBoardReports)))
	router.HandleFunc("/board-reports/{id:[0-9]+}/generate-narrative", mid.Use(as.BoardReportGenerateNarrative, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureEnhancedBoardReports)))
	router.HandleFunc("/board-reports/{id:[0-9]+}/narrative-edit", mid.Use(as.BoardReportEditNarrative, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureEnhancedBoardReports)))
	router.HandleFunc("/board-reports/{id:[0-9]+}/transition", mid.Use(as.BoardReportTransition, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureEnhancedBoardReports)))
	router.HandleFunc("/board-reports/{id:[0-9]+}/approvals", mid.Use(as.BoardReportApprovals, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeatureEnhancedBoardReports)))
	router.HandleFunc("/board-reports/heatmap", mid.Use(as.BoardReportHeatmap, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeatureEnhancedBoardReports)))
	router.HandleFunc("/board-reports/deltas", mid.Use(as.BoardReportDeltas, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeatureEnhancedBoardReports)))

	// ── Real-Time Dashboard (WebSocket + Metrics + Sparklines) ──
	router.HandleFunc("/dashboard/ws", mid.Use(as.DashboardWS, mid.RequireFeature(models.FeatureRealtimeDashboard)))
	router.HandleFunc("/dashboard/metrics", mid.Use(as.DashboardMetrics, mid.RequireFeature(models.FeatureRealtimeDashboard)))
	router.HandleFunc("/dashboard/sparkline", mid.Use(as.DashboardSparkline, mid.RequireFeature(models.FeatureRealtimeDashboard)))
	router.HandleFunc("/dashboard/preference", mid.Use(as.DashboardPreference, mid.RequireFeature(models.FeatureRealtimeDashboard)))
	router.HandleFunc("/dashboard/live-counts", mid.Use(as.DashboardLiveCounts, mid.RequireFeature(models.FeatureRealtimeDashboard)))
	router.HandleFunc("/dashboard/ws-status", mid.Use(as.DashboardWSStatus, mid.RequirePermission(models.PermissionModifySystem), mid.RequireFeature(models.FeatureRealtimeDashboard)))

	// ── Scheduled Reports (admin-configurable recurring report delivery) ──
	router.HandleFunc("/scheduled-reports/", mid.Use(as.ScheduledReports, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeatureScheduledReports)))
	router.HandleFunc("/scheduled-reports/types", mid.Use(as.ScheduledReportTypes, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeatureScheduledReports)))
	router.HandleFunc("/scheduled-reports/summary", mid.Use(as.ScheduledReportSummary, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeatureScheduledReports)))
	router.HandleFunc("/scheduled-reports/{id:[0-9]+}", mid.Use(as.ScheduledReport, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeatureScheduledReports)))
	router.HandleFunc("/scheduled-reports/{id:[0-9]+}/toggle", mid.Use(as.ScheduledReportToggle, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureScheduledReports)))

	// ── Unified Export API (standardized exports for all report types) ──
	router.HandleFunc("/export", mid.Use(as.UnifiedExport, mid.RequirePermission(models.PermissionViewReports)))

	// ── A/B Testing API ──
	router.HandleFunc("/ab-test/{campaignId:[0-9]+}", mid.Use(as.ABTestSummary, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeatureABTesting)))

	// ── DB-Backed Template Library (Admin) ──
	router.HandleFunc("/template-library-db/", mid.Use(as.TemplateLibraryDB, mid.RequireFeature(models.FeatureTemplateLibraryDB)))
	router.HandleFunc("/template-library-db/stats", mid.Use(as.TemplateLibraryDBStats, mid.RequireFeature(models.FeatureTemplateLibraryDB)))
	router.HandleFunc("/template-library-db/categories", mid.Use(as.TemplateLibraryDBCategories, mid.RequireFeature(models.FeatureTemplateLibraryDB)))
	router.HandleFunc("/template-library-db/import", mid.Use(as.TemplateLibraryDBImport, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureTemplateLibraryDB)))
	router.HandleFunc("/template-library-db/export", mid.Use(as.TemplateLibraryDBExport, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeatureTemplateLibraryDB)))
	router.HandleFunc("/template-library-db/seed", mid.Use(as.TemplateLibraryDBSeed, mid.RequirePermission(models.PermissionModifySystem), mid.RequireFeature(models.FeatureTemplateLibraryDB)))
	router.HandleFunc("/template-library-db/seed-all", mid.Use(as.TemplateLibraryDBSeedAll, mid.RequirePermission(models.PermissionModifySystem), mid.RequireFeature(models.FeatureTemplateLibraryDB)))
	router.HandleFunc("/template-library-db/seed-multilingual", mid.Use(as.TemplateLibraryDBSeedMultilingual, mid.RequirePermission(models.PermissionModifySystem), mid.RequireFeature(models.FeatureTemplateLibraryDB)))
	router.HandleFunc("/template-library-db/community/submit", mid.Use(as.TemplateLibraryDBCommunitySubmit, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureTemplateLibraryDB)))
	router.HandleFunc("/template-library-db/community/submissions", mid.Use(as.TemplateLibraryDBCommunityList, mid.RequireFeature(models.FeatureTemplateLibraryDB)))
	router.HandleFunc("/template-library-db/community/review", mid.Use(as.TemplateLibraryDBCommunityReview, mid.RequirePermission(models.PermissionModifySystem), mid.RequireFeature(models.FeatureTemplateLibraryDB)))
	router.HandleFunc("/template-library-db/search", mid.Use(as.TemplateLibraryDBSearch, mid.RequireFeature(models.FeatureTemplateLibraryDB)))
	router.HandleFunc("/template-library-db/similarity", mid.Use(as.TemplateLibraryDBSimilarity, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureTemplateLibraryDB)))
	router.HandleFunc("/template-library-db/reviews", mid.Use(as.TemplateLibraryDBReviews, mid.RequirePermission(models.PermissionModifySystem), mid.RequireFeature(models.FeatureTemplateLibraryDB)))
	router.HandleFunc("/template-library-db/reviews/stats", mid.Use(as.TemplateLibraryDBReviewStats, mid.RequirePermission(models.PermissionModifySystem), mid.RequireFeature(models.FeatureTemplateLibraryDB)))
	router.HandleFunc("/template-library-db/reviews/complete", mid.Use(as.TemplateLibraryDBReviewComplete, mid.RequirePermission(models.PermissionModifySystem), mid.RequireFeature(models.FeatureTemplateLibraryDB)))
	router.HandleFunc("/template-library-db/reviews/revision", mid.Use(as.TemplateLibraryDBReviewRevision, mid.RequirePermission(models.PermissionModifySystem), mid.RequireFeature(models.FeatureTemplateLibraryDB)))
	router.HandleFunc("/template-library-db/reviews/create-pending", mid.Use(as.TemplateLibraryDBCreateReviews, mid.RequirePermission(models.PermissionModifySystem), mid.RequireFeature(models.FeatureTemplateLibraryDB)))
	router.HandleFunc("/template-library-db/export-csv", mid.Use(as.TemplateLibraryDBExportCSV, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeatureTemplateLibraryDB)))
	router.HandleFunc("/template-library-db/import-csv", mid.Use(as.TemplateLibraryDBImportCSV, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureTemplateLibraryDB)))
	router.HandleFunc("/template-library-db/bulk-publish", mid.Use(as.TemplateLibraryDBBulkPublish, mid.RequirePermission(models.PermissionModifySystem), mid.RequireFeature(models.FeatureTemplateLibraryDB)))
	router.HandleFunc("/template-library-db/bulk-unpublish", mid.Use(as.TemplateLibraryDBBulkUnpublish, mid.RequirePermission(models.PermissionModifySystem), mid.RequireFeature(models.FeatureTemplateLibraryDB)))
	router.HandleFunc("/template-library-db/bulk-delete", mid.Use(as.TemplateLibraryDBBulkDelete, mid.RequirePermission(models.PermissionModifySystem), mid.RequireFeature(models.FeatureTemplateLibraryDB)))
	router.HandleFunc("/template-library-db/bulk-tag", mid.Use(as.TemplateLibraryDBBulkTag, mid.RequirePermission(models.PermissionModifySystem), mid.RequireFeature(models.FeatureTemplateLibraryDB)))
	router.HandleFunc("/template-library-db/{id:[0-9]+}", mid.Use(as.TemplateLibraryDBItem, mid.RequireFeature(models.FeatureTemplateLibraryDB)))

	// ── ROI Reporting Dashboard (demonstrate value to leadership) ──
	router.HandleFunc("/roi/dashboard", mid.Use(as.ROIDashboard, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeatureROIDashboard)))
	router.HandleFunc("/roi/investment-config", mid.Use(as.ROIInvestmentConfig, mid.RequirePermission(models.PermissionModifySystem), mid.RequireFeature(models.FeatureROIDashboard)))

	// ── AI-Powered Content Translation ──
	router.HandleFunc("/ai-translation/config", mid.Use(as.AITranslationConfig, mid.RequirePermission(models.PermissionModifySystem), mid.RequireFeature(models.FeatureAITranslation)))
	router.HandleFunc("/ai-translation/languages", mid.Use(as.AITranslationLanguages, mid.RequireFeature(models.FeatureAITranslation)))
	router.HandleFunc("/ai-translation/translate", mid.Use(as.AITranslationTranslate, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureAITranslation)))
	router.HandleFunc("/ai-translation/history", mid.Use(as.AITranslationHistory, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeatureAITranslation)))
	router.HandleFunc("/ai-translation/usage", mid.Use(as.AITranslationUsage, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeatureAITranslation)))
	router.HandleFunc("/ai-translation/content/{id:[0-9]+}", mid.Use(as.AITranslationContent, mid.RequireFeature(models.FeatureAITranslation)))
	router.HandleFunc("/ai-translation/{id:[0-9]+}/approve", mid.Use(as.AITranslationApprove, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureAITranslation)))

	// ── Automated Training Reminders (enhanced) ──
	router.HandleFunc("/reminders/nudge", mid.Use(as.ReminderNudge, mid.RequirePermission(models.PermissionModifyObjects)))
	router.HandleFunc("/reminders/bulk-nudge", mid.Use(as.ReminderBulkNudge, mid.RequirePermission(models.PermissionModifyObjects)))
	router.HandleFunc("/reminders/template", mid.Use(as.ReminderTemplate, mid.RequirePermission(models.PermissionModifySystem)))
	router.HandleFunc("/reminders/assignment/{id:[0-9]+}/history", mid.Use(as.ReminderAssignmentHistory, mid.RequirePermission(models.PermissionViewReports)))

	// ── Pre-Built Compliance Training Modules (progress tracking) ──
	router.HandleFunc("/compliance/module-progress", mid.Use(as.ComplianceModuleProgressHandler, mid.RequireFeature(models.FeatureComplianceMapping)))
	router.HandleFunc("/compliance/module-assign", mid.Use(as.ComplianceModuleAssign, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureComplianceMapping)))
	router.HandleFunc("/compliance/module-stats", mid.Use(as.ComplianceModuleOrgStats, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeatureComplianceMapping)))
	router.HandleFunc("/compliance/module-assignments", mid.Use(as.ComplianceModuleOrgAssignments, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeatureComplianceMapping)))
	router.HandleFunc("/compliance/module-seed", mid.Use(as.ComplianceModuleSeed, mid.RequirePermission(models.PermissionModifySystem), mid.RequireFeature(models.FeatureComplianceMapping)))
	router.HandleFunc("/compliance/training-modules/{slug}/progress", mid.Use(as.ComplianceModuleDetail, mid.RequireFeature(models.FeatureComplianceMapping)))

	// ── AI Admin Assistant (Aria) — guided onboarding + platform navigation ──
	router.HandleFunc("/admin-assistant/chat", mid.Use(as.AdminAssistantChat, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureAIAssistant)))
	router.HandleFunc("/admin-assistant/conversations", mid.Use(as.AdminAssistantConversations, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureAIAssistant)))
	router.HandleFunc("/admin-assistant/conversations/{id:[0-9]+}", mid.Use(as.AdminAssistantConversation, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureAIAssistant)))
	router.HandleFunc("/admin-assistant/onboarding", mid.Use(as.AdminAssistantOnboarding, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureAIAssistant)))
	router.HandleFunc("/admin-assistant/onboarding/{step}/complete", mid.Use(as.AdminAssistantOnboardingComplete, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureAIAssistant)))

	// ── Sending Domain Pool ──
	router.HandleFunc("/domain-pool/", mid.Use(as.DomainPool, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureDomainPool)))
	router.HandleFunc("/domain-pool/config", mid.Use(as.DomainPoolConfigHandler, mid.RequirePermission(models.PermissionModifySystem), mid.RequireFeature(models.FeatureDomainPool)))
	router.HandleFunc("/domain-pool/summary", mid.Use(as.DomainPoolSummary, mid.RequirePermission(models.PermissionViewReports), mid.RequireFeature(models.FeatureDomainPool)))
	router.HandleFunc("/domain-pool/seed", mid.Use(as.DomainPoolSeed, mid.RequirePermission(models.PermissionModifySystem), mid.RequireFeature(models.FeatureDomainPool)))
	router.HandleFunc("/domain-pool/select", mid.Use(as.DomainPoolSelect, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureDomainPool)))
	router.HandleFunc("/domain-pool/{id:[0-9]+}", mid.Use(as.DomainPoolItem, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireFeature(models.FeatureDomainPool)))
	router.HandleFunc("/domain-pool/{id:[0-9]+}/warmup", mid.Use(as.DomainPoolWarmup, mid.RequirePermission(models.PermissionModifySystem), mid.RequireFeature(models.FeatureDomainPool)))

	// SCIM v2 protocol endpoints (IdP-facing, uses SCIM bearer token auth)
	scim := root.PathPrefix("/scim/v2/").Subrouter()
	scim.Use(mid.RequireSCIMToken)
	scim.HandleFunc("/ServiceProviderConfig", as.SCIMServiceProviderConfig).Methods("GET")
	scim.HandleFunc("/ResourceTypes", as.SCIMResourceTypes).Methods("GET")
	scim.HandleFunc("/Users", as.SCIMUsers).Methods("GET", "POST")
	scim.HandleFunc("/Users/{id:[0-9]+}", as.SCIMUser).Methods("GET", "PUT", "PATCH", "DELETE")
	scim.HandleFunc("/Groups", as.SCIMGroups).Methods("GET", "POST")
	scim.HandleFunc("/Groups/{id:[0-9]+}", as.SCIMGroup).Methods("GET", "PUT", "PATCH", "DELETE")

	// ── Telephony Webhook (no API key auth — uses webhook secret) ──
	root.HandleFunc("/webhooks/telephony", as.TelephonyWebhook).Methods("POST")

	as.handler = root
}

func (as *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	as.handler.ServeHTTP(w, r)
}
