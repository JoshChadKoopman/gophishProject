package controllers

import (
	"compress/gzip"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/NYTimes/gziphandler"
	"github.com/gophish/gophish/auth"
	"github.com/gophish/gophish/config"
	ctx "github.com/gophish/gophish/context"
	"github.com/gophish/gophish/controllers/api"
	"github.com/gophish/gophish/i18n"
	log "github.com/gophish/gophish/logger"
	mid "github.com/gophish/gophish/middleware"
	"github.com/gophish/gophish/middleware/ratelimit"
	"github.com/gophish/gophish/metrics"
	"github.com/gophish/gophish/models"
	"github.com/gophish/gophish/util"
	"github.com/gophish/gophish/worker"
	"github.com/gorilla/csrf"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/jordan-wright/unindexed"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// AdminServerOption is a functional option that is used to configure the
// admin server
type AdminServerOption func(*AdminServer)

// Route and template path constants to avoid duplicate literals (S1192).
const (
	routeLogin         = "/login"
	routeMFAEnroll     = "/mfa/enroll"
	routeMFAVerify     = "/mfa/verify"
	headerAcceptLang   = "Accept-Language"
	tmplFlashes        = "templates/flashes.html"
	errInternalMessage = "Internal error"
)

// AdminServer is an HTTP server that implements the administrative Gophish
// handlers, including the dashboard and REST API.
type AdminServer struct {
	server     *http.Server
	worker     worker.Worker
	config     config.AdminServer
	limiter    *ratelimit.PostLimiter
	oidcClient *auth.OIDCClient
	samlClient *auth.SAMLClient
	mfaEncKey  string // base64-encoded 32-byte AES key for TOTP secret encryption
	aiConfig   config.AIConfig
}

var defaultTLSConfig = &tls.Config{
	PreferServerCipherSuites: true,
	CurvePreferences: []tls.CurveID{
		tls.X25519,
		tls.CurveP256,
	},
	MinVersion: tls.VersionTLS12,
	CipherSuites: []uint16{
		tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
		tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
	},
}

// WithWorker is an option that sets the background worker.
func WithWorker(w worker.Worker) AdminServerOption {
	return func(as *AdminServer) {
		as.worker = w
	}
}

// NewAdminServer returns a new instance of the AdminServer with the
// provided config and options applied.
func NewAdminServer(config config.AdminServer, options ...AdminServerOption) *AdminServer {
	defaultWorker, _ := worker.New()
	defaultServer := &http.Server{
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
		Addr:         config.ListenURL,
	}
	defaultLimiter := ratelimit.NewPostLimiter()
	as := &AdminServer{
		worker:  defaultWorker,
		server:  defaultServer,
		limiter: defaultLimiter,
		config:  config,
	}
	for _, opt := range options {
		opt(as)
	}
	as.registerRoutes()
	return as
}

// WithOIDCClient is an option that sets the OIDC client for SSO login.
func WithOIDCClient(oidcClient *auth.OIDCClient) AdminServerOption {
	return func(as *AdminServer) {
		as.oidcClient = oidcClient
	}
}

// WithSAMLClient is an option that sets the SAML 2.0 client for SSO login.
func WithSAMLClient(samlClient *auth.SAMLClient) AdminServerOption {
	return func(as *AdminServer) {
		as.samlClient = samlClient
	}
}

// WithMFAEncKey is an option that sets the base64-encoded AES-256 key used to
// encrypt TOTP secrets at rest.
func WithMFAEncKey(key string) AdminServerOption {
	return func(as *AdminServer) {
		as.mfaEncKey = key
	}
}

// WithAIConfig sets the AI provider configuration for template generation.
func WithAIConfig(cfg config.AIConfig) AdminServerOption {
	return func(as *AdminServer) {
		as.aiConfig = cfg
	}
}

// Start launches the admin server, listening on the configured address.
func (as *AdminServer) Start() {
	if as.worker != nil {
		go as.worker.Start()
	}
	if as.config.UseTLS {
		// Only support TLS 1.2 and above - ref #1691, #1689
		as.server.TLSConfig = defaultTLSConfig
		err := util.CheckAndCreateSSL(as.config.CertPath, as.config.KeyPath)
		if err != nil {
			log.Fatal(err)
		}
		log.Infof("Starting admin server at https://%s", as.config.ListenURL)
		log.Fatal(as.server.ListenAndServeTLS(as.config.CertPath, as.config.KeyPath))
	}
	// If TLS isn't configured, just listen on HTTP
	log.Infof("Starting admin server at http://%s", as.config.ListenURL)
	log.Fatal(as.server.ListenAndServe())
}

// Shutdown attempts to gracefully shutdown the server.
func (as *AdminServer) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	return as.server.Shutdown(ctx)
}

// SetupAdminRoutes creates the routes for handling requests to the web interface.
// This function returns an http.Handler to be used in http.ListenAndServe().
func (as *AdminServer) registerRoutes() {
	router := mux.NewRouter()
	// Base Front-end routes
	router.HandleFunc("/", mid.Use(as.Base, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireLogin))
	router.HandleFunc(routeLogin, mid.Use(as.Login, as.limiter.Limit))
	router.HandleFunc("/logout", mid.Use(as.Logout, mid.RequireLogin))
	router.HandleFunc("/reset_password", mid.Use(as.ResetPassword, mid.RequireLogin))
	router.HandleFunc("/campaigns", mid.Use(as.Campaigns, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireLogin))
	router.HandleFunc("/campaigns/{id:[0-9]+}", mid.Use(as.CampaignID, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireLogin))
	router.HandleFunc("/templates", mid.Use(as.Templates, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireLogin))
	router.HandleFunc("/groups", mid.Use(as.Groups, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireLogin))
	router.HandleFunc("/landing_pages", mid.Use(as.LandingPages, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireLogin))
	router.HandleFunc("/sending_profiles", mid.Use(as.SendingProfiles, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireLogin))
	router.HandleFunc("/feedback_pages", mid.Use(as.FeedbackPages, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireLogin))
	router.HandleFunc("/settings", mid.Use(as.Settings, mid.RequirePermission(models.PermissionModifySystem), mid.RequireLogin))
	router.HandleFunc("/users", mid.Use(as.UserManagement, mid.RequirePermission(models.PermissionModifySystem), mid.RequireLogin))
	router.HandleFunc("/webhooks", mid.Use(as.Webhooks, mid.RequirePermission(models.PermissionModifySystem), mid.RequireLogin))
	router.HandleFunc("/impersonate", mid.Use(as.Impersonate, mid.RequirePermission(models.PermissionModifySystem), mid.RequireLogin))
	router.HandleFunc("/training", mid.Use(as.Training, mid.RequireLogin))
	router.HandleFunc("/my-courses", mid.Use(as.MyCourses, mid.RequireLogin))
	router.HandleFunc("/autopilot", mid.Use(as.Autopilot, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireLogin))
	router.HandleFunc("/academy", mid.Use(as.Academy, mid.RequireLogin))
	router.HandleFunc("/difficulty", mid.Use(as.Difficulty, mid.RequireLogin))
	router.HandleFunc("/leaderboard", mid.Use(as.Leaderboard, mid.RequireLogin))
	router.HandleFunc("/reports", mid.Use(as.Reports, mid.RequireReportAccess, mid.RequireLogin))
	router.HandleFunc("/audit-log", mid.Use(as.AuditLogPage, mid.RequirePermission(models.PermissionViewReports), mid.RequireLogin))
	router.HandleFunc("/org-settings", mid.Use(as.OrgSettings, mid.RequirePermission(models.PermissionModifySystem), mid.RequireLogin))
	// OIDC / SSO routes
	router.HandleFunc("/auth/oidc/login", mid.Use(as.OIDCLogin))
	router.HandleFunc("/auth/oidc/callback", mid.Use(as.OIDCCallback, as.limiter.Limit))
	router.HandleFunc("/auth/oidc/logout", mid.Use(as.OIDCLogout, mid.RequireLogin))
	// SAML 2.0 SSO routes (separate admin/user paths when split mode enabled)
	router.HandleFunc("/auth/saml/login", mid.Use(as.SAMLLogin))
	router.HandleFunc("/auth/saml/admin/login", mid.Use(as.SAMLAdminLogin))
	router.HandleFunc("/auth/saml/user/login", mid.Use(as.SAMLUserLogin))
	router.HandleFunc("/auth/saml/acs", mid.Use(as.SAMLCallback, as.limiter.Limit))
	router.HandleFunc("/auth/saml/admin/acs", mid.Use(as.SAMLAdminCallback, as.limiter.Limit))
	router.HandleFunc("/auth/saml/user/acs", mid.Use(as.SAMLUserCallback, as.limiter.Limit))
	// MFA routes — /mfa/enroll and /mfa/verify are pre-login so must NOT use RequireLogin
	router.HandleFunc(routeMFAEnroll, mid.Use(as.MFAEnroll))
	router.HandleFunc(routeMFAVerify, mid.Use(as.MFAVerify))
	router.HandleFunc("/mfa/backup-codes", mid.Use(as.MFABackupCodes, mid.RequireLogin))
	// New page routes for Phase 13
	router.HandleFunc("/reported-emails", mid.Use(as.ReportedEmailsPage, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireLogin))
	router.HandleFunc("/threat-alerts", mid.Use(as.ThreatAlertsPage, mid.RequireLogin))
	// New page routes for Remediation Paths and Cyber Hygiene
	router.HandleFunc("/remediation", mid.Use(as.RemediationPage, mid.RequireLogin))
	router.HandleFunc("/cyber-hygiene", mid.Use(as.CyberHygienePage, mid.RequireLogin))
	// Board-ready reports page
	router.HandleFunc("/board-reports", mid.Use(as.BoardReportsPage, mid.RequirePermission(models.PermissionViewReports), mid.RequireLogin))
	// ROI Reporting page
	router.HandleFunc("/roi", mid.Use(as.ROIPage, mid.RequirePermission(models.PermissionViewReports), mid.RequireLogin))
	// Email Security Dashboard page
	router.HandleFunc("/email-security", mid.Use(as.EmailSecurityPage, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireLogin))
	// MSP Partner Portal page
	router.HandleFunc("/partner-portal", mid.Use(as.PartnerPortalPage, mid.RequireMSPPartner, mid.RequireLogin))
	// Scheduled Reports page
	router.HandleFunc("/scheduled-reports", mid.Use(as.ScheduledReportsPage, mid.RequirePermission(models.PermissionViewReports), mid.RequireLogin))
	// Network Events Dashboard page
	router.HandleFunc("/network-events", mid.Use(as.NetworkEventsPage, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireLogin))
	// AI Admin Assistant page
	router.HandleFunc("/admin-assistant", mid.Use(as.AdminAssistantPage, mid.RequirePermission(models.PermissionModifyObjects), mid.RequireLogin))
	// Health / readiness / metrics endpoints — unauthenticated, no CSRF.
	// /metrics should be restricted to internal IPs at the nginx layer.
	router.HandleFunc("/healthz", as.Health).Methods(http.MethodGet)
	router.HandleFunc("/readyz", as.Readyz).Methods(http.MethodGet)
	router.Handle("/metrics", promhttp.Handler()).Methods(http.MethodGet)
	// Create the API routes
	apiServer := api.NewServer(
		api.WithWorker(as.worker),
		api.WithLimiter(as.limiter),
		api.WithAIConfig(as.aiConfig),
	)
	router.PathPrefix("/api/").Handler(apiServer)
	// Plugin API route — uses plugin API key auth, not user sessions.
	// The /api prefix makes it CSRF-exempt via CSRFExemptPrefixes.
	router.HandleFunc("/api/plugin/report-email", mid.Use(as.PluginReportEmail))

	// Setup static file serving
	router.PathPrefix("/").Handler(http.FileServer(unindexed.Dir("./static/")))

	// Setup CSRF Protection
	csrfKey := []byte(as.config.CSRFKey)
	if len(csrfKey) == 0 {
		csrfKey = []byte(auth.GenerateSecureKey(auth.APIKeyLength))
	}
	csrfHandler := csrf.Protect(csrfKey,
		csrf.FieldName("csrf_token"),
		csrf.Secure(as.config.UseTLS),
		csrf.TrustedOrigins(as.config.TrustedOrigins))
	adminHandler := csrfHandler(router)
	adminHandler = mid.Use(adminHandler.ServeHTTP, mid.CSRFExceptions, mid.GetContext, mid.ApplySecurityHeaders, mid.RequestID, mid.RequestLogger)

	// Setup GZIP compression
	gzipWrapper, _ := gziphandler.NewGzipLevelHandler(gzip.BestCompression)
	adminHandler = gzipWrapper(adminHandler)

	// Respect X-Forwarded-For and X-Real-IP headers in case we're behind a
	// reverse proxy.
	adminHandler = handlers.ProxyHeaders(adminHandler)

	// Prometheus HTTP instrumentation — wraps outermost so it captures every
	// request including static files, health checks, and the metrics endpoint.
	adminHandler = metrics.Instrument("admin", adminHandler)

	// Setup logging
	adminHandler = handlers.CombinedLoggingHandler(log.Writer(), adminHandler)
	as.server.Handler = adminHandler
}

type templateParams struct {
	Title          string
	Flashes        []interface{}
	User           models.User
	Org            models.Organization
	IsSuperAdmin   bool
	Token          string
	Version        string
	ModifySystem   bool
	ModifyObjects  bool
	ViewReports    bool
	ManageTraining bool
	OIDCEnabled    bool
	OrgFeatures    map[string]bool
	Locale         string
	Languages      []i18n.LanguageInfo
}

// newTemplateParams returns the default template parameters for a user and
// the CSRF token.
func newTemplateParams(r *http.Request) templateParams {
	user := ctx.Get(r, "user").(models.User)
	session := ctx.Get(r, "session").(*sessions.Session)
	modifySystem, _ := user.HasPermission(models.PermissionModifySystem)
	modifyObjects, _ := user.HasPermission(models.PermissionModifyObjects)
	viewReports, _ := user.HasPermission(models.PermissionViewReports)
	manageTraining, _ := user.HasPermission(models.PermissionManageTraining)
	org, _ := models.GetOrganization(user.OrgId)
	locale := i18n.DetectLocale(user.PreferredLanguage, org.DefaultLanguage, r.Header.Get(headerAcceptLang))
	return templateParams{
		Token:          csrf.Token(r),
		User:           user,
		Org:            org,
		IsSuperAdmin:   user.Role.Slug == models.RoleSuperAdmin,
		ModifySystem:   modifySystem,
		ModifyObjects:  modifyObjects,
		ViewReports:    viewReports,
		ManageTraining: manageTraining,
		Version:        config.Version,
		Flashes:        session.Flashes(),
		OrgFeatures:    models.GetOrgFeatures(user.OrgId),
		Locale:         locale,
		Languages:      i18n.GetLanguages(),
	}
}

// Base handles the default path and template execution
func (as *AdminServer) Base(w http.ResponseWriter, r *http.Request) {
	params := newTemplateParams(r)
	params.Title = "Dashboard"
	getTemplate(w, "dashboard").ExecuteTemplate(w, "base", params)
}

// Campaigns handles the default path and template execution
func (as *AdminServer) Campaigns(w http.ResponseWriter, r *http.Request) {
	params := newTemplateParams(r)
	params.Title = "Campaigns"
	getTemplate(w, "campaigns").ExecuteTemplate(w, "base", params)
}

// CampaignID handles the default path and template execution
func (as *AdminServer) CampaignID(w http.ResponseWriter, r *http.Request) {
	params := newTemplateParams(r)
	params.Title = "Campaign Results"
	getTemplate(w, "campaign_results").ExecuteTemplate(w, "base", params)
}

// Templates handles the default path and template execution
func (as *AdminServer) Templates(w http.ResponseWriter, r *http.Request) {
	params := newTemplateParams(r)
	params.Title = "Email Templates"
	getTemplate(w, "templates").ExecuteTemplate(w, "base", params)
}

// Groups handles the default path and template execution
func (as *AdminServer) Groups(w http.ResponseWriter, r *http.Request) {
	params := newTemplateParams(r)
	params.Title = "Users & Groups"
	getTemplate(w, "groups").ExecuteTemplate(w, "base", params)
}

// LandingPages handles the default path and template execution
func (as *AdminServer) LandingPages(w http.ResponseWriter, r *http.Request) {
	params := newTemplateParams(r)
	params.Title = "Landing Pages"
	getTemplate(w, "landing_pages").ExecuteTemplate(w, "base", params)
}

// FeedbackPages handles the default path and template execution
func (as *AdminServer) FeedbackPages(w http.ResponseWriter, r *http.Request) {
	params := newTemplateParams(r)
	params.Title = "Feedback Pages"
	getTemplate(w, "feedback_pages").ExecuteTemplate(w, "base", params)
}

// SendingProfiles handles the default path and template execution
func (as *AdminServer) SendingProfiles(w http.ResponseWriter, r *http.Request) {
	params := newTemplateParams(r)
	params.Title = "Sending Profiles"
	getTemplate(w, "sending_profiles").ExecuteTemplate(w, "base", params)
}

// Settings handles the changing of settings
func (as *AdminServer) Settings(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.Method == "GET":
		params := newTemplateParams(r)
		params.Title = "Settings"
		session := ctx.Get(r, "session").(*sessions.Session)
		session.Save(r, w)
		getTemplate(w, "settings").ExecuteTemplate(w, "base", params)
	case r.Method == "POST":
		u := ctx.Get(r, "user").(models.User)
		currentPw := r.FormValue("current_password")
		newPassword := r.FormValue("new_password")
		confirmPassword := r.FormValue("confirm_new_password")
		// Check the current password
		err := auth.ValidatePassword(currentPw, u.Hash)
		msg := models.Response{Success: true, Message: "Settings Updated Successfully"}
		if err != nil {
			msg.Message = err.Error()
			msg.Success = false
			api.JSONResponse(w, msg, http.StatusBadRequest)
			return
		}
		newHash, err := auth.ValidatePasswordChange(u.Hash, newPassword, confirmPassword)
		if err != nil {
			msg.Message = err.Error()
			msg.Success = false
			api.JSONResponse(w, msg, http.StatusBadRequest)
			return
		}
		u.Hash = string(newHash)
		if err = models.PutUser(&u); err != nil {
			msg.Message = err.Error()
			msg.Success = false
			api.JSONResponse(w, msg, http.StatusInternalServerError)
			return
		}
		api.JSONResponse(w, msg, http.StatusOK)
	}
}

// UserManagement is an admin-only handler that allows for the registration
// and management of user accounts within Gophish.
func (as *AdminServer) UserManagement(w http.ResponseWriter, r *http.Request) {
	params := newTemplateParams(r)
	params.Title = "User Management"
	getTemplate(w, "users").ExecuteTemplate(w, "base", params)
}

func (as *AdminServer) nextOrIndex(w http.ResponseWriter, r *http.Request, u *models.User) {
	next := "/"
	if raw := r.FormValue("next"); raw != "" {
		parsed, err := url.Parse(raw)
		// Accept only relative paths: no scheme, no host, path must start with "/".
		// This prevents open-redirect to external origins (e.g. next=https://evil.com
		// or next=/%2F%2Fevil.com which some browsers normalise to //evil.com).
		if err == nil && parsed.Scheme == "" && parsed.Host == "" {
			path := parsed.EscapedPath()
			if strings.HasPrefix(path, "/") && !strings.HasPrefix(path, "//") {
				next = path
			}
		}
	}
	// If the user is a reader (no ModifyObjects permission), redirect to training
	// instead of the dashboard, since they can't access dashboard/campaigns.
	// The user is passed explicitly because context may not be populated yet
	// (e.g. right after login before middleware re-runs).
	if next == "/" && u != nil {
		hasModify, _ := u.HasPermission(models.PermissionModifyObjects)
		if !hasModify {
			next = "/training"
		}
	}
	http.Redirect(w, r, next, http.StatusFound)
}

func (as *AdminServer) handleInvalidLogin(w http.ResponseWriter, r *http.Request, message string) {
	session := ctx.Get(r, "session").(*sessions.Session)
	Flash(w, r, "danger", message)
	locale := i18n.DetectLocale("", "", r.Header.Get(headerAcceptLang))
	params := struct {
		User        models.User
		Title       string
		Flashes     []interface{}
		Token       string
		OIDCEnabled bool
		Locale      string
		Languages   []i18n.LanguageInfo
	}{Title: "Login", Token: csrf.Token(r), OIDCEnabled: as.oidcClient != nil, Locale: locale, Languages: i18n.GetLanguages()}
	params.Flashes = session.Flashes()
	session.Save(r, w)
	templates := template.New("template").Funcs(templateFuncs)
	_, err := templates.ParseFiles("templates/login.html", tmplFlashes)
	if err != nil {
		log.Error(err)
	}
	w.WriteHeader(http.StatusUnauthorized)
	template.Must(templates, err).ExecuteTemplate(w, "base", params)
}

// Webhooks is an admin-only handler that handles webhooks
func (as *AdminServer) Webhooks(w http.ResponseWriter, r *http.Request) {
	params := newTemplateParams(r)
	params.Title = "Webhooks"
	getTemplate(w, "webhooks").ExecuteTemplate(w, "base", params)
}

// Training handles the training presentations page accessible to all users
func (as *AdminServer) Training(w http.ResponseWriter, r *http.Request) {
	params := newTemplateParams(r)
	params.Title = "Training"
	getTemplate(w, "training").ExecuteTemplate(w, "base", params)
}

// MyCourses handles the display of the user's course progress page
func (as *AdminServer) MyCourses(w http.ResponseWriter, r *http.Request) {
	params := newTemplateParams(r)
	params.Title = "My Courses"
	getTemplate(w, "my_courses").ExecuteTemplate(w, "base", params)
}

// Autopilot handles the autopilot configuration page
func (as *AdminServer) Autopilot(w http.ResponseWriter, r *http.Request) {
	params := newTemplateParams(r)
	params.Title = "Autopilot"
	getTemplate(w, "autopilot").ExecuteTemplate(w, "base", params)
}

// Academy handles the academy tier progression page
func (as *AdminServer) Academy(w http.ResponseWriter, r *http.Request) {
	params := newTemplateParams(r)
	params.Title = "Academy"
	getTemplate(w, "academy").ExecuteTemplate(w, "base", params)
}

// Difficulty handles the adaptive difficulty management page
func (as *AdminServer) Difficulty(w http.ResponseWriter, r *http.Request) {
	params := newTemplateParams(r)
	params.Title = "Training Difficulty"
	getTemplate(w, "difficulty").ExecuteTemplate(w, "base", params)
}

// Leaderboard handles the gamification leaderboard page
func (as *AdminServer) Leaderboard(w http.ResponseWriter, r *http.Request) {
	params := newTemplateParams(r)
	params.Title = "Leaderboard"
	getTemplate(w, "leaderboard").ExecuteTemplate(w, "base", params)
}

// Reports handles the reports page accessible to users with view_reports or modify_objects
func (as *AdminServer) Reports(w http.ResponseWriter, r *http.Request) {
	params := newTemplateParams(r)
	params.Title = "Reports"
	getTemplate(w, "reports").ExecuteTemplate(w, "base", params)
}

// AuditLogPage handles the audit log web UI page
func (as *AdminServer) AuditLogPage(w http.ResponseWriter, r *http.Request) {
	params := newTemplateParams(r)
	params.Title = "Audit Log"
	getTemplate(w, "audit_log").ExecuteTemplate(w, "base", params)
}

// OrgSettings renders the organization settings page.
func (as *AdminServer) OrgSettings(w http.ResponseWriter, r *http.Request) {
	params := newTemplateParams(r)
	params.Title = "Organization Settings"
	getTemplate(w, "org_settings").ExecuteTemplate(w, "base", params)
}

// ReportedEmailsPage renders the reported emails admin dashboard.
func (as *AdminServer) ReportedEmailsPage(w http.ResponseWriter, r *http.Request) {
	params := newTemplateParams(r)
	params.Title = "Reported Emails"
	getTemplate(w, "reported_emails").ExecuteTemplate(w, "base", params)
}

// ThreatAlertsPage renders the threat alerts page.
func (as *AdminServer) ThreatAlertsPage(w http.ResponseWriter, r *http.Request) {
	params := newTemplateParams(r)
	params.Title = "Threat Alerts"
	getTemplate(w, "threat_alerts").ExecuteTemplate(w, "base", params)
}

// RemediationPage handles the /remediation route.
func (as *AdminServer) RemediationPage(w http.ResponseWriter, r *http.Request) {
	params := newTemplateParams(r)
	params.Title = "Remediation Paths"
	getTemplate(w, "remediation").ExecuteTemplate(w, "base", params)
}

// CyberHygienePage handles the /cyber-hygiene route.
func (as *AdminServer) CyberHygienePage(w http.ResponseWriter, r *http.Request) {
	params := newTemplateParams(r)
	params.Title = "Cyber Hygiene"
	getTemplate(w, "cyber_hygiene").ExecuteTemplate(w, "base", params)
}

// BoardReportsPage handles the /board-reports route.
func (as *AdminServer) BoardReportsPage(w http.ResponseWriter, r *http.Request) {
	params := newTemplateParams(r)
	params.Title = "Board Reports"
	getTemplate(w, "board_reports").ExecuteTemplate(w, "base", params)
}

// ROIPage handles the /roi route.
func (as *AdminServer) ROIPage(w http.ResponseWriter, r *http.Request) {
	params := newTemplateParams(r)
	params.Title = "ROI Reporting"
	getTemplate(w, "roi").ExecuteTemplate(w, "base", params)
}

// EmailSecurityPage handles the /email-security route.
func (as *AdminServer) EmailSecurityPage(w http.ResponseWriter, r *http.Request) {
	params := newTemplateParams(r)
	params.Title = "Email Security"
	getTemplate(w, "email_security").ExecuteTemplate(w, "base", params)
}

// PartnerPortalPage renders the MSP Partner Portal page.
func (as *AdminServer) PartnerPortalPage(w http.ResponseWriter, r *http.Request) {
	params := newTemplateParams(r)
	params.Title = "Partner Portal"
	getTemplate(w, "partner_portal").ExecuteTemplate(w, "base", params)
}

// ScheduledReportsPage renders the Scheduled Reports management page.
func (as *AdminServer) ScheduledReportsPage(w http.ResponseWriter, r *http.Request) {
	params := newTemplateParams(r)
	params.Title = "Scheduled Reports"
	getTemplate(w, "scheduled_reports").ExecuteTemplate(w, "base", params)
}

// NetworkEventsPage handles the /network-events route.
func (as *AdminServer) NetworkEventsPage(w http.ResponseWriter, r *http.Request) {
	params := newTemplateParams(r)
	params.Title = "Network Events"
	getTemplate(w, "network_events").ExecuteTemplate(w, "base", params)
}

// AdminAssistantPage handles the /admin-assistant route.
func (as *AdminServer) AdminAssistantPage(w http.ResponseWriter, r *http.Request) {
	params := newTemplateParams(r)
	params.Title = "AI Admin Assistant"
	getTemplate(w, "admin_assistant").ExecuteTemplate(w, "base", params)
}

// PluginReportEmail handles POST /api/plugin/report-email from the
// Outlook/Gmail report button plugin. Uses plugin API key auth.
func (as *AdminServer) PluginReportEmail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	apiKey := extractPluginAPIKey(r)
	if apiKey == "" {
		api.JSONResponse(w, models.Response{Success: false, Message: "Plugin API key not provided"}, http.StatusUnauthorized)
		return
	}
	config, err := models.GetReportButtonConfigByAPIKey(apiKey)
	if err != nil {
		api.JSONResponse(w, models.Response{Success: false, Message: "Invalid plugin API key"}, http.StatusUnauthorized)
		return
	}
	var re models.ReportedEmail
	if err := json.NewDecoder(r.Body).Decode(&re); err != nil {
		api.JSONResponse(w, models.Response{Success: false, Message: "Invalid request body"}, http.StatusBadRequest)
		return
	}
	if re.ReporterEmail == "" {
		api.JSONResponse(w, models.Response{Success: false, Message: "reporter_email is required"}, http.StatusBadRequest)
		return
	}
	re.OrgId = config.OrgId
	// Set source platform (default to outlook, Google Workspace plugin sends "google")
	if re.SourcePlatform == "" {
		re.SourcePlatform = "outlook"
	}
	isSimulation := models.ClassifyEmailBySimulation(&re)
	if !isSimulation {
		re.Classification = "pending"
	}
	if err := models.CreateReportedEmail(&re); err != nil {
		api.JSONResponse(w, models.Response{Success: false, Message: err.Error()}, http.StatusInternalServerError)
		return
	}
	feedback := config.FeedbackReal
	if isSimulation {
		feedback = config.FeedbackSimulation
	}
	if feedback == "" {
		if isSimulation {
			feedback = "Good catch! This was a simulated phishing email."
		} else {
			feedback = "Thank you for reporting. Our security team will review this email."
		}
	}
	api.JSONResponse(w, struct {
		Success      bool   `json:"success"`
		Message      string `json:"message"`
		IsSimulation bool   `json:"is_simulation"`
	}{Success: true, Message: feedback, IsSimulation: isSimulation}, http.StatusOK)
}

// extractPluginAPIKey extracts the Bearer token from the Authorization header.
func extractPluginAPIKey(r *http.Request) string {
	tokens, ok := r.Header["Authorization"]
	if !ok || len(tokens) == 0 {
		return ""
	}
	token := tokens[0]
	if len(token) > 7 && token[:7] == "Bearer " {
		return token[7:]
	}
	return token
}

// Impersonate allows an admin to login to a user account without needing the password
func (as *AdminServer) Impersonate(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		username := r.FormValue("username")
		u, err := models.GetUserByUsername(username)
		if err != nil {
			log.Error(err)
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		session := ctx.Get(r, "session").(*sessions.Session)
		session.Values["id"] = u.Id
		// The acting admin has already authenticated; grant MFA credit so the
		// impersonated session is not immediately redirected to /mfa/verify.
		session.Values["mfa_verified"] = true
		session.Save(r, w)
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

// Login handles the authentication flow for a user. If credentials are valid,
// a session is created
func (as *AdminServer) Login(w http.ResponseWriter, r *http.Request) {
	locale := i18n.DetectLocale("", "", r.Header.Get(headerAcceptLang))
	params := struct {
		User           models.User
		Title          string
		Flashes        []interface{}
		Token          string
		OIDCEnabled    bool
		SAMLEnabled    bool
		SAMLSplitMode  bool
		SAMLAdminURL   string
		SAMLUserURL    string
		SAMLDefaultURL string
		Locale         string
		Languages      []i18n.LanguageInfo
	}{
		Title:       "Login",
		Token:       csrf.Token(r),
		OIDCEnabled: as.oidcClient != nil,
		SAMLEnabled: as.samlClient != nil,
		Locale:      locale,
		Languages:   i18n.GetLanguages(),
	}
	if as.samlClient != nil {
		params.SAMLSplitMode = as.samlClient.IsSplitMode()
		params.SAMLAdminURL = as.samlClient.AdminLoginURL()
		params.SAMLUserURL = as.samlClient.UserLoginURL()
		params.SAMLDefaultURL = "/auth/saml/login"
	}
	session := ctx.Get(r, "session").(*sessions.Session)
	switch {
	case r.Method == "GET":
		params.Flashes = session.Flashes()
		session.Save(r, w)
		templates := template.New("template").Funcs(templateFuncs)
		_, err := templates.ParseFiles("templates/login.html", tmplFlashes)
		if err != nil {
			log.Error(err)
		}
		template.Must(templates, err).ExecuteTemplate(w, "base", params)
	case r.Method == "POST":
		as.handleLoginPost(w, r, session)
	}
}

// handleLoginPost processes the POST side of the login flow.
//
// Security: on success this function does NOT immediately set session["id"]
// or session["mfa_verified"] for roles that require MFA. Instead it parks
// the authenticated user ID in session["pending_user_id"] and routes to the
// appropriate MFA step:
//
//   - MFA required, not enrolled  → /mfa/enroll
//   - MFA required, enrolled      → /mfa/verify
//   - MFA not required            → full session granted here
//
// mfa_verified is ONLY ever set in mfaVerifyPost / mfaEnrollPost after a
// successful TOTP/backup-code challenge, preventing the bypass that would
// occur if it were set here before the challenge completes.
func (as *AdminServer) handleLoginPost(w http.ResponseWriter, r *http.Request, session *sessions.Session) {
	loginInput, password := r.FormValue("username"), r.FormValue("password")
	u, err := models.GetUserByEmail(loginInput)
	if err != nil {
		u, err = models.GetUserByUsername(loginInput)
		if err != nil {
			log.Error(err)
			as.handleInvalidLogin(w, r, "Invalid Email/Password")
			return
		}
	}
	if models.IsLoginLockedOut(&u) {
		as.handleInvalidLogin(w, r, "Account temporarily locked. Try again in 15 minutes.")
		return
	}
	if u.AccountLocked {
		as.handleInvalidLogin(w, r, "Account Locked")
		return
	}
	if err := auth.ValidatePassword(password, u.Hash); err != nil {
		log.Error(err)
		models.RecordFailedLogin(u.Id)
		if auditErr := models.CreateAuditLog(&models.AuditLog{
			OrgId:         u.OrgId,
			ActorID:       u.Id,
			ActorUsername: u.Username,
			Action:        models.AuditActionLoginFailed,
			IPAddress:     r.RemoteAddr,
		}); auditErr != nil {
			log.Error(auditErr)
		}
		remaining := auth.MaxFailedLogins - (u.FailedLogins + 1)
		loginMsg := "Invalid Email/Password"
		if remaining > 0 {
			loginMsg = fmt.Sprintf("Invalid Email/Password. %d attempt(s) remaining before temporary lockout.", remaining)
		}
		as.handleInvalidLogin(w, r, loginMsg)
		return
	}
	u.LastLogin = time.Now().UTC()
	if err := models.PutUser(&u); err != nil {
		log.Error(err)
	}
	models.ResetFailedLogins(u.Id)
	if auditErr := models.CreateAuditLog(&models.AuditLog{
		OrgId:         u.OrgId,
		ActorID:       u.Id,
		ActorUsername: u.Username,
		Action:        models.AuditActionLoginSuccess,
		IPAddress:     r.RemoteAddr,
	}); auditErr != nil {
		log.Error(auditErr)
	}

	// If the role requires MFA, route through the challenge before granting
	// a full session. We only store the pending user ID — no "id" or
	// "mfa_verified" — so the user cannot access protected resources yet.
	if auth.MFARequired(u.Role.Slug) {
		// Skip the interactive challenge if this device is already trusted
		// (remember-device cookie valid for 30 days).
		if as.isDeviceRemembered(r, u.Id) {
			session.Values["id"] = u.Id
			session.Values["mfa_verified"] = true
			session.Save(r, w)
			as.nextOrIndex(w, r, &u)
			return
		}
		session.Values["pending_user_id"] = u.Id
		session.Save(r, w)
		device, devErr := models.GetMFADevice(u.Id)
		if devErr != nil || !device.Enabled {
			// MFA required but not yet enrolled — send to enrollment.
			http.Redirect(w, r, routeMFAEnroll, http.StatusFound)
			return
		}
		// MFA enrolled — send to TOTP challenge.
		http.Redirect(w, r, routeMFAVerify, http.StatusFound)
		return
	}

	// MFA not required for this role — grant full session immediately.
	session.Values["id"] = u.Id
	session.Values["mfa_verified"] = true
	session.Save(r, w)
	as.nextOrIndex(w, r, &u)
}

// Logout destroys the current user session
func (as *AdminServer) Logout(w http.ResponseWriter, r *http.Request) {
	session := ctx.Get(r, "session").(*sessions.Session)
	delete(session.Values, "id")
	Flash(w, r, "success", "You have successfully logged out")
	session.Save(r, w)
	http.Redirect(w, r, routeLogin, http.StatusFound)
}

// ResetPassword handles the password reset flow when a password change is
// required either by the Gophish system or an administrator.
//
// This handler is meant to be used when a user is required to reset their
// password, not just when they want to.
//
// This is an important distinction since in this handler we don't require
// the user to re-enter their current password, as opposed to the flow
// through the settings handler.
//
// To that end, if the user doesn't require a password change, we will
// redirect them to the settings page.
func (as *AdminServer) ResetPassword(w http.ResponseWriter, r *http.Request) {
	u := ctx.Get(r, "user").(models.User)
	session := ctx.Get(r, "session").(*sessions.Session)
	if !u.PasswordChangeRequired {
		Flash(w, r, "info", "Please reset your password through the settings page")
		session.Save(r, w)
		http.Redirect(w, r, "/settings", http.StatusTemporaryRedirect)
		return
	}
	params := newTemplateParams(r)
	params.Title = "Reset Password"
	switch {
	case r.Method == http.MethodGet:
		params.Flashes = session.Flashes()
		session.Save(r, w)
		getTemplate(w, "reset_password").ExecuteTemplate(w, "base", params)
		return
	case r.Method == http.MethodPost:
		newPassword := r.FormValue("password")
		confirmPassword := r.FormValue("confirm_password")
		newHash, err := auth.ValidatePasswordChange(u.Hash, newPassword, confirmPassword)
		if err != nil {
			Flash(w, r, "danger", err.Error())
			params.Flashes = session.Flashes()
			session.Save(r, w)
			w.WriteHeader(http.StatusBadRequest)
			getTemplate(w, "reset_password").ExecuteTemplate(w, "base", params)
			return
		}
		u.PasswordChangeRequired = false
		u.Hash = newHash
		if err = models.PutUser(&u); err != nil {
			Flash(w, r, "danger", err.Error())
			params.Flashes = session.Flashes()
			session.Save(r, w)
			w.WriteHeader(http.StatusInternalServerError)
			getTemplate(w, "reset_password").ExecuteTemplate(w, "base", params)
			return
		}
		// Flash a success message so the user knows the password was updated.
		Flash(w, r, "success", "Password changed successfully!")
		session.Save(r, w)
		as.nextOrIndex(w, r, &u)
	}
}

// isDeviceRemembered checks whether the current request carries a valid
// "remember device" cookie whose fingerprint hash matches a DB record.
func (as *AdminServer) isDeviceRemembered(r *http.Request, userID int64) bool {
	cookie, err := r.Cookie("device_fp")
	if err != nil {
		return false
	}
	fp, err := models.FindDeviceFingerprint(userID, cookie.Value)
	if err != nil {
		return false
	}
	return fp.ID > 0
}

// mfaLoginParams is the minimal template data for pre-login MFA pages.
type mfaLoginParams struct {
	Title   string
	Token   string
	Flashes []interface{}
}

// OIDCLogin redirects the browser to the Keycloak authorisation endpoint.
// When OIDC is disabled it falls back to the native /login page.
func (as *AdminServer) OIDCLogin(w http.ResponseWriter, r *http.Request) {
	if as.oidcClient == nil {
		http.Redirect(w, r, routeLogin, http.StatusFound)
		return
	}
	session := ctx.Get(r, "session").(*sessions.Session)
	state := auth.GenerateSecureKey(16)
	nonce := auth.GenerateSecureKey(16)
	session.Values["oidc_state"] = state
	session.Values["oidc_nonce"] = nonce
	session.Save(r, w)
	http.Redirect(w, r, as.oidcClient.AuthCodeURL(state, nonce), http.StatusFound)
}

// OIDCCallback handles the Keycloak redirect after the user authenticates.
// It validates the state, exchanges the code, and establishes a full session.
func (as *AdminServer) OIDCCallback(w http.ResponseWriter, r *http.Request) {
	if as.oidcClient == nil {
		http.Redirect(w, r, routeLogin, http.StatusFound)
		return
	}
	session := ctx.Get(r, "session").(*sessions.Session)

	// Validate state to prevent CSRF on the OIDC callback.
	expectedState, _ := session.Values["oidc_state"].(string)
	if r.URL.Query().Get("state") != expectedState || expectedState == "" {
		http.Error(w, "Invalid OIDC state", http.StatusBadRequest)
		return
	}
	expectedNonce, _ := session.Values["oidc_nonce"].(string)
	delete(session.Values, "oidc_state")
	delete(session.Values, "oidc_nonce")

	claims, err := as.oidcClient.Exchange(r.Context(), r.URL.Query().Get("code"), expectedNonce)
	if err != nil {
		log.Errorf("OIDC callback exchange error: %v", err)
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
		return
	}

	// Find or create the local user record, keyed by email.
	u, err := models.GetUserByEmail(claims.Email)
	if err != nil {
		// Auto-provision the user with the role from OIDC claims.
		roleSlug := auth.ExtractRoleSlug(claims.Roles)
		role, roleErr := models.GetRoleBySlug(roleSlug)
		if roleErr != nil {
			log.Errorf("OIDC callback: unknown role %s: %v", roleSlug, roleErr)
			http.Error(w, "Configuration error", http.StatusInternalServerError)
			return
		}
		u = models.User{
			Username:  claims.Email,
			Email:     claims.Email,
			FirstName: claims.Name,
			OrgId:     1, // Default organization
			ApiKey:    auth.GenerateSecureKey(auth.APIKeyLength),
			Role:      role,
			RoleID:    role.ID,
		}
		if putErr := models.PutUser(&u); putErr != nil {
			log.Errorf("OIDC callback: failed to create user: %v", putErr)
			http.Error(w, errInternalMessage, http.StatusInternalServerError)
			return
		}
	}

	// Keycloak enforced TOTP before issuing the token, so we can grant MFA credit.
	session.Values["id"] = u.Id
	session.Values["mfa_verified"] = true
	session.Save(r, w)
	as.nextOrIndex(w, r, &u)
}

// OIDCLogout clears the local session and redirects to the Keycloak logout endpoint.
func (as *AdminServer) OIDCLogout(w http.ResponseWriter, r *http.Request) {
	session := ctx.Get(r, "session").(*sessions.Session)
	delete(session.Values, "id")
	delete(session.Values, "mfa_verified")
	session.Save(r, w)
	if as.oidcClient != nil {
		http.Redirect(w, r, as.oidcClient.LogoutURL(), http.StatusFound)
		return
	}
	http.Redirect(w, r, routeLogin, http.StatusFound)
}

// MFAEnroll handles the TOTP enrollment wizard.
// GET: generates a new TOTP secret, stores it encrypted in the session, renders the QR page.
// POST: validates the submitted 6-digit code against the session secret; on success saves
//
//	the MFADevice and backup codes, then redirects to /mfa/backup-codes.
func (as *AdminServer) MFAEnroll(w http.ResponseWriter, r *http.Request) {
	session := ctx.Get(r, "session").(*sessions.Session)
	u, ok := as.getMFAUser(session, r, w)
	if !ok {
		return
	}
	aesKey, err := auth.TOTPEncryptionKeyFromBase64(as.mfaEncKey)
	if err != nil {
		log.Errorf("MFA enroll: encryption key error: %v", err)
		http.Error(w, "MFA not configured — contact your administrator", http.StatusServiceUnavailable)
		return
	}
	switch r.Method {
	case http.MethodGet:
		as.mfaEnrollGet(w, r, session, u, aesKey)
	case http.MethodPost:
		as.mfaEnrollPost(w, r, session, u, aesKey)
	}
}

// getMFAUser resolves the user for MFA flows from either pending_user_id or the context.
func (as *AdminServer) getMFAUser(session *sessions.Session, r *http.Request, w http.ResponseWriter) (models.User, bool) {
	if pendingID, ok := session.Values["pending_user_id"].(int64); ok && pendingID != 0 {
		u, err := models.GetUser(pendingID)
		if err != nil {
			http.Redirect(w, r, routeLogin, http.StatusFound)
			return models.User{}, false
		}
		return u, true
	}
	if ctxUser := ctx.Get(r, "user"); ctxUser != nil {
		return ctxUser.(models.User), true
	}
	http.Redirect(w, r, routeLogin, http.StatusFound)
	return models.User{}, false
}

func (as *AdminServer) mfaEnrollGet(w http.ResponseWriter, r *http.Request, session *sessions.Session, u models.User, aesKey []byte) {
	accountName := u.Email
	if accountName == "" {
		accountName = u.Username
	}
	secret, qrURI, genErr := auth.GenerateTOTPSecret(accountName)
	if genErr != nil {
		log.Errorf("MFA enroll: failed to generate secret: %v", genErr)
		http.Error(w, errInternalMessage, http.StatusInternalServerError)
		return
	}
	encrypted, encErr := auth.EncryptTOTPSecret(secret, aesKey)
	if encErr != nil {
		log.Errorf("MFA enroll: encryption failed: %v", encErr)
		http.Error(w, errInternalMessage, http.StatusInternalServerError)
		return
	}
	session.Values["mfa_enroll_secret"] = encrypted
	session.Save(r, w)
	templates := template.New("template").Funcs(templateFuncs)
	_, parseErr := templates.ParseFiles("templates/mfa_enroll.html", tmplFlashes)
	if parseErr != nil {
		log.Error(parseErr)
	}
	locale := i18n.DetectLocale("", "", r.Header.Get(headerAcceptLang))
	params := struct {
		Title     string
		Token     string
		QRCode    string
		ManualKey string
		Flashes   []interface{}
		Locale    string
	}{
		Title:     "Enable Two-Factor Authentication",
		Token:     csrf.Token(r),
		QRCode:    qrURI,
		ManualKey: secret,
		Flashes:   session.Flashes(),
		Locale:    locale,
	}
	session.Save(r, w)
	template.Must(templates, parseErr).ExecuteTemplate(w, "base", params)
}

func (as *AdminServer) mfaEnrollPost(w http.ResponseWriter, r *http.Request, session *sessions.Session, u models.User, aesKey []byte) {
	encryptedSecret, ok := session.Values["mfa_enroll_secret"].(string)
	if !ok || encryptedSecret == "" {
		http.Redirect(w, r, routeMFAEnroll, http.StatusFound)
		return
	}
	code := r.FormValue("code")
	if !auth.ValidateTOTP(encryptedSecret, code, aesKey) {
		Flash(w, r, "danger", "Invalid code — please try again")
		http.Redirect(w, r, routeMFAEnroll, http.StatusFound)
		return
	}
	device := &models.MFADevice{UserID: u.Id, TOTPSecret: encryptedSecret}
	if saveErr := models.CreateOrUpdateMFADevice(device); saveErr != nil {
		log.Errorf("MFA enroll: save device failed: %v", saveErr)
		http.Error(w, errInternalMessage, http.StatusInternalServerError)
		return
	}
	if enableErr := models.EnableMFADevice(u.Id); enableErr != nil {
		log.Errorf("MFA enroll: enable device failed: %v", enableErr)
	}
	plain, hashed, codeErr := auth.GenerateBackupCodes(auth.BackupCodeLength)
	if codeErr != nil {
		log.Errorf("MFA enroll: backup code generation failed: %v", codeErr)
	} else {
		_ = models.SaveMFABackupCodes(u.Id, hashed)
		session.Values["new_backup_codes"] = plain
	}
	delete(session.Values, "mfa_enroll_secret")
	if _, isPending := session.Values["pending_user_id"]; isPending {
		delete(session.Values, "pending_user_id")
		session.Values["id"] = u.Id
		session.Values["mfa_verified"] = true
	}
	session.Save(r, w)
	http.Redirect(w, r, "/mfa/backup-codes", http.StatusFound)
}

// MFAVerify handles the TOTP challenge during login (before the full session is set).
// GET: renders the TOTP input page.
// POST: validates the code, promotes the pending session to a full session on success.
func (as *AdminServer) MFAVerify(w http.ResponseWriter, r *http.Request) {
	session := ctx.Get(r, "session").(*sessions.Session)

	// Require a pending user from the login flow.
	pendingID, ok := session.Values["pending_user_id"].(int64)
	if !ok || pendingID == 0 {
		http.Redirect(w, r, routeLogin, http.StatusFound)
		return
	}
	u, err := models.GetUser(pendingID)
	if err != nil {
		http.Redirect(w, r, routeLogin, http.StatusFound)
		return
	}

	switch r.Method {
	case http.MethodGet:
		as.mfaVerifyGet(w, r, session, u)
	case http.MethodPost:
		as.mfaVerifyPost(w, r, session, u)
	}
}

// mfaVerifyGet renders the TOTP input page for the MFA verification step.
func (as *AdminServer) mfaVerifyGet(w http.ResponseWriter, r *http.Request, session *sessions.Session, u models.User) {
	locale := i18n.DetectLocale("", "", r.Header.Get(headerAcceptLang))
	failCount, _ := models.CountRecentMFAFailures(u.Id, time.Now().Add(-auth.MFALockoutDuration))
	remaining := auth.MFAMaxAttempts - failCount
	if remaining < 0 {
		remaining = 0
	}
	params := struct {
		Title             string
		Token             string
		Flashes           []interface{}
		Locale            string
		RemainingAttempts int
	}{
		Title:             "Two-Factor Verification",
		Token:             csrf.Token(r),
		Locale:            locale,
		RemainingAttempts: remaining,
	}
	params.Flashes = session.Flashes()
	session.Save(r, w)
	templates := template.New("template").Funcs(templateFuncs)
	_, parseErr := templates.ParseFiles("templates/mfa_verify.html", tmplFlashes)
	if parseErr != nil {
		log.Error(parseErr)
	}
	template.Must(templates, parseErr).ExecuteTemplate(w, "base", params)
}

// mfaVerifyPost validates the submitted TOTP/backup code and promotes the
// pending session to a full session on success.
func (as *AdminServer) mfaVerifyPost(w http.ResponseWriter, r *http.Request, session *sessions.Session, u models.User) {
	// Lockout check.
	failCount, _ := models.CountRecentMFAFailures(u.Id, time.Now().Add(-auth.MFALockoutDuration))
	if failCount >= auth.MFAMaxAttempts {
		_ = models.RecordMFAAttempt(u.Id, false, r.RemoteAddr)
		if auditErr := models.CreateAuditLog(&models.AuditLog{
			OrgId:         u.OrgId,
			ActorID:       u.Id,
			ActorUsername: u.Username,
			Action:        models.AuditActionMFALockout,
			IPAddress:     r.RemoteAddr,
		}); auditErr != nil {
			log.Error(auditErr)
		}
		Flash(w, r, "danger", "Account temporarily locked due to too many failed attempts. Please try again in 15 minutes.")
		http.Redirect(w, r, routeMFAVerify, http.StatusFound)
		return
	}

	code := r.FormValue("code")
	aesKey, keyErr := auth.TOTPEncryptionKeyFromBase64(as.mfaEncKey)
	if keyErr != nil {
		http.Error(w, "MFA not configured", http.StatusServiceUnavailable)
		return
	}
	device, devErr := models.GetMFADevice(u.Id)
	if devErr != nil {
		http.Redirect(w, r, routeLogin, http.StatusFound)
		return
	}

	verified := as.verifyMFACode(code, device.TOTPSecret, aesKey, u.Id)
	_ = models.RecordMFAAttempt(u.Id, verified, r.RemoteAddr)

	if !verified {
		if auditErr := models.CreateAuditLog(&models.AuditLog{
			OrgId:         u.OrgId,
			ActorID:       u.Id,
			ActorUsername: u.Username,
			Action:        models.AuditActionMFAFailed,
			IPAddress:     r.RemoteAddr,
		}); auditErr != nil {
			log.Error(auditErr)
		}
		Flash(w, r, "danger", "Invalid code — please try again")
		http.Redirect(w, r, routeMFAVerify, http.StatusFound)
		return
	}

	if auditErr := models.CreateAuditLog(&models.AuditLog{
		OrgId:         u.OrgId,
		ActorID:       u.Id,
		ActorUsername: u.Username,
		Action:        models.AuditActionMFAVerified,
		IPAddress:     r.RemoteAddr,
	}); auditErr != nil {
		log.Error(auditErr)
	}

	// Promote the session.
	delete(session.Values, "pending_user_id")
	session.Values["id"] = u.Id
	session.Values["mfa_verified"] = true

	// Handle "remember this device for 30 days".
	as.rememberDeviceIfRequested(w, r, u.Id)

	session.Save(r, w)
	as.nextOrIndex(w, r, &u)
}

// verifyMFACode checks the code against the TOTP secret and, if that fails,
// against unused backup codes for the given user.
func (as *AdminServer) verifyMFACode(code, totpSecret string, aesKey []byte, userID int64) bool {
	if auth.ValidateTOTP(totpSecret, code, aesKey) {
		return true
	}
	backupCodes, _ := models.GetUnusedBackupCodes(userID)
	for _, bc := range backupCodes {
		if auth.ValidateBackupCode(code, bc.CodeHash) {
			_ = models.MarkBackupCodeUsed(bc.ID)
			return true
		}
	}
	return false
}

// rememberDeviceIfRequested stores a device fingerprint cookie so that the
// user can skip MFA on subsequent logins from the same device for 30 days.
func (as *AdminServer) rememberDeviceIfRequested(w http.ResponseWriter, r *http.Request, userID int64) {
	if r.FormValue("remember_device") != "1" {
		return
	}
	rawFP := auth.RawDeviceFingerprint(r.Header.Get("User-Agent"), r.Header.Get(headerAcceptLang))
	fpHash, fpErr := auth.DeviceFingerprintHash(rawFP)
	if fpErr != nil {
		return
	}
	expires := time.Now().Add(auth.DeviceRememberDuration)
	_ = models.CreateDeviceFingerprint(&models.DeviceFingerprint{
		UserID:          userID,
		FingerprintHash: fpHash,
		ExpiresAt:       expires,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "device_fp",
		Value:    rawFP,
		Expires:  expires,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	})
}

// MFABackupCodes renders the one-time display of newly generated backup codes.
// After the first view the codes are cleared from the session and cannot be retrieved again.
func (as *AdminServer) MFABackupCodes(w http.ResponseWriter, r *http.Request) {
	session := ctx.Get(r, "session").(*sessions.Session)
	rawCodes, hasCodes := session.Values["new_backup_codes"].([]string)

	locale := i18n.DetectLocale("", "", r.Header.Get(headerAcceptLang))
	params := struct {
		Title       string
		Token       string
		BackupCodes []string
		Flashes     []interface{}
		Locale      string
	}{
		Title:   "Your Backup Codes",
		Token:   csrf.Token(r),
		Flashes: session.Flashes(),
		Locale:  locale,
	}

	if hasCodes && len(rawCodes) > 0 {
		params.BackupCodes = rawCodes
		delete(session.Values, "new_backup_codes")
	}
	session.Save(r, w)

	templates := template.New("template").Funcs(templateFuncs)
	_, parseErr := templates.ParseFiles("templates/mfa_backup_codes.html", tmplFlashes)
	if parseErr != nil {
		log.Error(parseErr)
	}
	template.Must(templates, parseErr).ExecuteTemplate(w, "base", params)
}

// templateFuncs returns the template.FuncMap used by all page templates.
var templateFuncs = template.FuncMap{
	"T": func(locale, key string) string {
		return i18n.T(locale, key)
	},
}

// getTemplate parses the base layout and the given page template, returning
// a ready-to-execute *template.Template. Callers typically chain
// .ExecuteTemplate(w, "base", data) immediately after.
func getTemplate(w http.ResponseWriter, tmpl string) *template.Template {
	templates := template.New("template").Funcs(templateFuncs)
	_, err := templates.ParseFiles("templates/base.html", "templates/nav.html", "templates/"+tmpl+".html", tmplFlashes)
	if err != nil {
		log.Error(err)
	}
	return template.Must(templates, err)
}

// Health handles GET /healthz. It pings the database and returns 200 OK when
// the process is alive and the DB is reachable, or 503 when the DB is down.
func (as *AdminServer) Health(w http.ResponseWriter, r *http.Request) {
	type healthResponse struct {
		Status  string `json:"status"`
		DB      string `json:"db"`
		Version string `json:"version"`
	}
	resp := healthResponse{Status: "ok", Version: config.Version}
	code := http.StatusOK
	if err := models.GetDB().DB().Ping(); err != nil {
		resp.Status = "unhealthy"
		resp.DB = "unreachable"
		code = http.StatusServiceUnavailable
	} else {
		resp.DB = "connected"
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(resp) //nolint:errcheck
}

// Readyz handles GET /readyz. It checks both DB connectivity and whether the
// background campaign worker has run within the last 5 minutes. Returns 200
// when ready, 503 when not.
func (as *AdminServer) Readyz(w http.ResponseWriter, r *http.Request) {
	type readyResponse struct {
		Status  string `json:"status"`
		DB      string `json:"db"`
		Worker  string `json:"worker"`
		Version string `json:"version"`
	}
	resp := readyResponse{Status: "ok", Version: config.Version}
	code := http.StatusOK
	if err := models.GetDB().DB().Ping(); err != nil {
		resp.Status = "not ready"
		resp.DB = "unreachable"
		code = http.StatusServiceUnavailable
	} else {
		resp.DB = "connected"
	}
	last := worker.LastHeartbeat()
	if last == 0 || time.Since(time.Unix(last, 0)) > 5*time.Minute {
		resp.Status = "not ready"
		resp.Worker = "stalled"
		code = http.StatusServiceUnavailable
	} else {
		resp.Worker = "ok"
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(resp) //nolint:errcheck
}

// Flash handles the rendering flash messages
func Flash(w http.ResponseWriter, r *http.Request, t string, m string) {
	session := ctx.Get(r, "session").(*sessions.Session)
	session.AddFlash(models.Flash{
		Type:    t,
		Message: m,
	})
}
