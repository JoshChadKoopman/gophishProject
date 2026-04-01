package api

import (
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
	router.HandleFunc("/groups/", as.Groups)
	router.HandleFunc("/groups/summary", as.GroupsSummary)
	router.HandleFunc("/groups/{id:[0-9]+}", as.Group)
	router.HandleFunc("/groups/{id:[0-9]+}/summary", as.GroupSummary)
	router.HandleFunc("/templates/", as.Templates)
	router.HandleFunc("/templates/{id:[0-9]+}", as.Template)
	router.HandleFunc("/pages/", as.Pages)
	router.HandleFunc("/pages/{id:[0-9]+}", as.Page)
	router.HandleFunc("/smtp/", as.SendingProfiles)
	router.HandleFunc("/smtp/{id:[0-9]+}", as.SendingProfile)
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
	router.HandleFunc("/training/{id:[0-9]+}", as.TrainingPresentation)
	router.HandleFunc("/training/{id:[0-9]+}/download", as.TrainingPresentationDownload)
	router.HandleFunc("/training/{id:[0-9]+}/thumbnail", as.TrainingPresentationThumbnail)
	router.HandleFunc("/training/{id:[0-9]+}/extract-slides", as.TrainingExtractSlidesExisting)
	router.HandleFunc("/training/{id:[0-9]+}/progress", as.TrainingCourseProgress)
	// Quiz routes
	router.HandleFunc("/training/{id:[0-9]+}/quiz", as.TrainingQuiz)
	router.HandleFunc("/training/{id:[0-9]+}/quiz/attempt", as.TrainingQuizAttempt)
	// Assignment routes
	router.HandleFunc("/training/assignments/", as.TrainingAssignments)
	router.HandleFunc("/training/assignments/group", as.TrainingAssignGroup)
	router.HandleFunc("/training/assignments/{id:[0-9]+}", as.TrainingAssignment)
	router.HandleFunc("/training/my-assignments", as.TrainingMyAssignments)
	// Certificate routes
	router.HandleFunc("/training/my-certificates", as.TrainingMyCertificates)
	router.HandleFunc("/training/certificates/verify/{code}", as.TrainingCertificateVerify)
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
	// Feedback page routes (micro-feedback interstitial)
	router.HandleFunc("/feedback_pages/", mid.Use(as.FeedbackPages, mid.RequirePermission(models.PermissionModifyObjects)))
	router.HandleFunc("/feedback_pages/default", mid.Use(as.FeedbackPageDefault, mid.RequirePermission(models.PermissionModifyObjects)))
	router.HandleFunc("/feedback_pages/{id:[0-9]+}", mid.Use(as.FeedbackPage, mid.RequirePermission(models.PermissionModifyObjects)))
	// i18n routes
	router.HandleFunc("/i18n/languages", as.I18nLanguages)
	router.HandleFunc("/i18n/{locale}", as.I18nTranslations)
	router.HandleFunc("/user/language", as.UserLanguage)

	as.handler = router
}

func (as *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	as.handler.ServeHTTP(w, r)
}
