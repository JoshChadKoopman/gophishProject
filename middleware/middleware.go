package middleware

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gophish/gophish/auth"
	ctx "github.com/gophish/gophish/context"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/csrf"
	"github.com/gorilla/sessions"
)

const errUserNotAuthenticated = "User not authenticated"

// trainingFallbackPath is the UI path learners are redirected to when they
// attempt to access a resource they lack permissions for.
const trainingFallbackPath = "/training"

// getUserFromContext safely extracts the authenticated user from the request
// context. Returns the user and true on success, or a zero-value User and
// false if the context value is nil or not a models.User.
func getUserFromContext(r *http.Request) (models.User, bool) {
	val := ctx.Get(r, "user")
	if val == nil {
		return models.User{}, false
	}
	user, ok := val.(models.User)
	return user, ok
}

// CSRFExemptPrefixes are a list of routes that are exempt from CSRF protection
var CSRFExemptPrefixes = []string{
	"/api",
}

// CSRFExceptions is a middleware that prevents CSRF checks on routes listed in
// CSRFExemptPrefixes.
func CSRFExceptions(handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		for _, prefix := range CSRFExemptPrefixes {
			if strings.HasPrefix(r.URL.Path, prefix) {
				r = csrf.UnsafeSkipCheck(r)
				break
			}
		}
		handler.ServeHTTP(w, r)
	}
}

// Use allows us to stack middleware to process the request
// Example taken from https://github.com/gorilla/mux/pull/36#issuecomment-25849172
func Use(handler http.HandlerFunc, mid ...func(http.Handler) http.HandlerFunc) http.HandlerFunc {
	for _, m := range mid {
		handler = m(handler)
	}
	return handler
}

// GetContext wraps each request in a function which fills in the context for a given request.
// This includes setting the User and Session keys and values as necessary for use in later functions.
func GetContext(handler http.Handler) http.HandlerFunc {
	// Set the context here
	return func(w http.ResponseWriter, r *http.Request) {
		// Parse the request form
		err := r.ParseForm()
		if err != nil {
			http.Error(w, "Error parsing request", http.StatusInternalServerError)
		}
		// Set the context appropriately here.
		// Set the session
		session, _ := Store.Get(r, "gophish")
		// Put the session in the context so that we can
		// reuse the values in different handlers
		r = ctx.Set(r, "session", session)
		if id, ok := session.Values["id"]; ok {
			u, err := models.GetUser(id.(int64))
			if err != nil {
				r = ctx.Set(r, "user", nil)
			} else {
				r = ctx.Set(r, "user", u)
			}
		} else {
			r = ctx.Set(r, "user", nil)
		}
		handler.ServeHTTP(w, r)
		// Remove context contents
		ctx.Clear(r)
	}
}

// RequireAPIKey ensures that a valid API key is set as either the api_key GET
// parameter, or a Bearer token.
func RequireAPIKey(handler http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		if r.Method == "OPTIONS" {
			w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS, PUT, DELETE")
			w.Header().Set("Access-Control-Max-Age", "1000")
			w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")
			w.Header().Set("Access-Control-Allow-Credentials", "false")
			return
		}
		r.ParseForm()
		ak := r.Form.Get("api_key")
		// If we can't get the API key, we'll also check for the
		// Authorization Bearer token
		if ak == "" {
			tokens, ok := r.Header["Authorization"]
			if ok && len(tokens) >= 1 {
				ak = tokens[0]
				ak = strings.TrimPrefix(ak, "Bearer ")
			}
		}
		if ak == "" {
			JSONError(w, http.StatusUnauthorized, "API Key not set")
			return
		}
		u, err := models.GetUserByAPIKey(ak)
		if err != nil {
			JSONError(w, http.StatusUnauthorized, "Invalid API Key")
			return
		}
		r = ctx.Set(r, "user", u)
		r = ctx.Set(r, "user_id", u.Id)
		r = ctx.Set(r, "api_key", ak)
		handler.ServeHTTP(w, r)
	})
}

// RequireLogin checks to see if the user is currently logged in.
// If not, the function returns a 302 redirect to the login page.
func RequireLogin(handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if u := ctx.Get(r, "user"); u != nil {
			// If a password change is required for the user, then redirect them
			// to the login page
			currentUser := u.(models.User)
			if currentUser.PasswordChangeRequired && r.URL.Path != "/reset_password" {
				q := r.URL.Query()
				q.Set("next", r.URL.Path)
				http.Redirect(w, r, fmt.Sprintf("/reset_password?%s", q.Encode()), http.StatusTemporaryRedirect)
				return
			}
			handler.ServeHTTP(w, r)
			return
		}
		q := r.URL.Query()
		q.Set("next", r.URL.Path)
		http.Redirect(w, r, fmt.Sprintf("/login?%s", q.Encode()), http.StatusTemporaryRedirect)
	}
}

// writeExemptPrefixes lists API path prefixes where any authenticated user
// may issue non-GET requests. The individual handlers enforce finer-grained
// permission checks. This allows learners to save course progress, submit
// quiz attempts, and read their own assignments/certificates.
var writeExemptPrefixes = []string{
	"/api/training/",
}

// isWriteExempt returns true if the request path is under a write-exempt prefix.
func isWriteExempt(path string) bool {
	for _, prefix := range writeExemptPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// isSafeMethod returns true for HTTP methods that do not modify resources.
func isSafeMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
}

// hasWritePermission returns true if the user has PermissionModifyObjects
// or PermissionManageTraining. Returns an error if the permission check fails.
func hasWritePermission(user models.User) (bool, error) {
	canModify, err := user.HasPermission(models.PermissionModifyObjects)
	if err != nil {
		return false, err
	}
	if canModify {
		return true, nil
	}
	return user.HasPermission(models.PermissionManageTraining)
}

// EnforceViewOnly is a global middleware that limits the ability to edit
// objects to accounts with at least one write permission.
func EnforceViewOnly(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if isSafeMethod(r.Method) || isWriteExempt(r.URL.Path) {
			next.ServeHTTP(w, r)
			return
		}
		user, ok := getUserFromContext(r)
		if !ok {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}
		allowed, err := hasWritePermission(user)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}
		if !allowed {
			http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireMFAEnrolled checks whether the user's role mandates MFA and, if so,
// whether they have a fully enrolled MFA device. If not enrolled, the request
// is redirected to /mfa/enroll. For non-MFA roles this middleware is a no-op.
func RequireMFAEnrolled(handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := getUserFromContext(r)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}
		if auth.MFARequired(user.Role.Slug) {
			device, err := models.GetMFADevice(user.Id)
			if err != nil || !device.Enabled {
				http.Redirect(w, r, "/mfa/enroll", http.StatusTemporaryRedirect)
				return
			}
		}
		handler.ServeHTTP(w, r)
	}
}

// RequireMFAVerified checks whether the active session has a completed MFA
// challenge (session key "mfa_verified" == true) for roles that require MFA.
// If the challenge has not been completed, the request is redirected to
// /mfa/verify. For non-MFA roles this middleware is a no-op.
func RequireMFAVerified(handler http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := getUserFromContext(r)
		if !ok {
			http.Redirect(w, r, "/login", http.StatusTemporaryRedirect)
			return
		}
		if auth.MFARequired(user.Role.Slug) {
			session := ctx.Get(r, "session").(*sessions.Session)
			verified, _ := session.Values["mfa_verified"].(bool)
			if !verified {
				http.Redirect(w, r, "/mfa/verify", http.StatusTemporaryRedirect)
				return
			}
		}
		handler.ServeHTTP(w, r)
	}
}

// respondForbidOrRedirect sends a JSON error for API paths or redirects
// non-API requests to the training fallback path.
func respondForbidOrRedirect(w http.ResponseWriter, r *http.Request, status int, message string) {
	if strings.HasPrefix(r.URL.Path, "/api") {
		JSONError(w, status, message)
	} else {
		http.Redirect(w, r, trainingFallbackPath, http.StatusTemporaryRedirect)
	}
}

// RequirePermission checks to see if the user has the requested permission
// before executing the handler. If the request is unauthorized, a JSONError
// is returned.
func RequirePermission(perm string) func(http.Handler) http.HandlerFunc {
	return func(next http.Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user, ok := getUserFromContext(r)
			if !ok {
				JSONError(w, http.StatusUnauthorized, errUserNotAuthenticated)
				return
			}
			access, err := user.HasPermission(perm)
			if err != nil {
				respondForbidOrRedirect(w, r, http.StatusInternalServerError, err.Error())
				return
			}
			if !access {
				respondForbidOrRedirect(w, r, http.StatusForbidden, http.StatusText(http.StatusForbidden))
				return
			}
			next.ServeHTTP(w, r)
		}
	}
}

// RequireReportAccess checks that the user has either PermissionViewReports
// or PermissionModifyObjects. Used for reporting and dashboard endpoints.
func RequireReportAccess(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := getUserFromContext(r)
		if !ok {
			JSONError(w, http.StatusUnauthorized, errUserNotAuthenticated)
			return
		}
		canView, _ := user.HasPermission(models.PermissionViewReports)
		canModify, _ := user.HasPermission(models.PermissionModifyObjects)
		if !canView && !canModify {
			respondForbidOrRedirect(w, r, http.StatusForbidden, http.StatusText(http.StatusForbidden))
			return
		}
		next.ServeHTTP(w, r)
	}
}

// checkTierLimit checks whether the org has exceeded its quota for the given
// resource type. Returns a non-empty message if the limit is reached.
func checkTierLimit(resourceType string, org models.Organization, tier models.SubscriptionTier) string {
	switch resourceType {
	case "campaign":
		count, _ := models.GetOrgCampaignCount(org.Id)
		if count >= tier.MaxCampaigns {
			return "Campaign limit reached for your organization tier"
		}
	case "user":
		count, _ := models.GetOrgUserCount(org.Id)
		if count >= tier.MaxUsers {
			return "User limit reached for your organization tier"
		}
	}
	return ""
}

// EnforceTierLimits checks org-level quotas before allowing resource creation.
// It wraps POST endpoints to prevent exceeding the org's subscription tier limits.
// Limits are read from the subscription_tiers table via the org's tier_id.
func EnforceTierLimits(resourceType string) func(http.Handler) http.HandlerFunc {
	return func(next http.Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				next.ServeHTTP(w, r)
				return
			}
			user, ok := getUserFromContext(r)
			if !ok {
				JSONError(w, http.StatusUnauthorized, errUserNotAuthenticated)
				return
			}
			org, err := models.GetOrganization(user.OrgId)
			if err != nil {
				JSONError(w, http.StatusInternalServerError, "Error loading organization")
				return
			}
			tier, err := models.GetSubscriptionTier(org.TierId)
			if err != nil {
				JSONError(w, http.StatusInternalServerError, "Error loading subscription tier")
				return
			}
			if msg := checkTierLimit(resourceType, org, tier); msg != "" {
				JSONError(w, http.StatusForbidden, msg)
				return
			}
			next.ServeHTTP(w, r)
		}
	}
}

// ApplySecurityHeaders applies various security headers according to best-
// practices.
func ApplySecurityHeaders(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Security-Policy", "default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; frame-ancestors 'none';")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("X-XSS-Protection", "0")
		if r.TLS != nil {
			w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	}
}

// JSONError returns an error in JSON format with the given
// status code and message
func JSONError(w http.ResponseWriter, c int, m string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(c)
	// Use json.NewEncoder for safe serialisation; if encoding fails the
	// status code has already been written, so we just write a fallback.
	if err := json.NewEncoder(w).Encode(models.Response{Success: false, Message: m}); err != nil {
		fmt.Fprintf(w, `{"success":false,"message":"internal encoding error"}`)
	}
}
