package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gophish/gophish/auth"
	"github.com/gophish/gophish/config"
	ctx "github.com/gophish/gophish/context"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/sessions"
)

// Shared test constants for middleware tests.
const (
	mwFmtIncorrectStatus = "incorrect status code received. expected %d got %d"
	mwContentTypeHeader  = "Content-Type"
	mwContentTypeJSON    = "application/json"
)

var successHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("success"))
})

type testContext struct {
	apiKey string
}

func setupTest(t *testing.T) *testContext {
	conf := &config.Config{
		DBName:         "sqlite3",
		DBPath:         ":memory:",
		MigrationsPath: "../db/db_sqlite3/migrations/",
	}
	err := models.Setup(conf)
	if err != nil {
		t.Fatalf("Failed creating database: %v", err)
	}
	// Get the API key to use for these tests
	u, err := models.GetUser(1)
	if err != nil {
		t.Fatalf("error getting user: %v", err)
	}
	ctx := &testContext{}
	ctx.apiKey = u.ApiKey
	return ctx
}

// MiddlewarePermissionTest maps an expected HTTP Method to an expected HTTP
// status code
type MiddlewarePermissionTest map[string]int

// TestEnforceViewOnly ensures that only users with the ModifyObjects
// permission have the ability to send non-GET requests.
func TestEnforceViewOnly(t *testing.T) {
	setupTest(t)
	permissionTests := map[string]MiddlewarePermissionTest{
		models.RoleAdmin: MiddlewarePermissionTest{
			http.MethodGet:     http.StatusOK,
			http.MethodHead:    http.StatusOK,
			http.MethodOptions: http.StatusOK,
			http.MethodPost:    http.StatusOK,
			http.MethodPut:     http.StatusOK,
			http.MethodDelete:  http.StatusOK,
		},
		models.RoleUser: MiddlewarePermissionTest{
			http.MethodGet:     http.StatusOK,
			http.MethodHead:    http.StatusOK,
			http.MethodOptions: http.StatusOK,
			http.MethodPost:    http.StatusOK,
			http.MethodPut:     http.StatusOK,
			http.MethodDelete:  http.StatusOK,
		},
	}
	for r, checks := range permissionTests {
		role, err := models.GetRoleBySlug(r)
		if err != nil {
			t.Fatalf("error getting role by slug: %v", err)
		}

		for method, expected := range checks {
			req := httptest.NewRequest(method, "/", nil)
			response := httptest.NewRecorder()

			req = ctx.Set(req, "user", models.User{
				Role:   role,
				RoleID: role.ID,
			})

			EnforceViewOnly(successHandler).ServeHTTP(response, req)
			got := response.Code
			if got != expected {
				t.Fatalf(mwFmtIncorrectStatus, expected, got)
			}
		}
	}
}

func TestRequirePermission(t *testing.T) {
	setupTest(t)
	middleware := RequirePermission(models.PermissionModifySystem)
	handler := middleware(successHandler)

	permissionTests := map[string]int{
		models.RoleUser:  http.StatusForbidden,
		models.RoleAdmin: http.StatusOK,
	}

	for role, expected := range permissionTests {
		req := httptest.NewRequest(http.MethodGet, "/api/test", nil)
		response := httptest.NewRecorder()
		// Test that with the requested permission, the request succeeds
		role, err := models.GetRoleBySlug(role)
		if err != nil {
			t.Fatalf("error getting role by slug: %v", err)
		}
		req = ctx.Set(req, "user", models.User{
			Role:   role,
			RoleID: role.ID,
		})
		handler.ServeHTTP(response, req)
		got := response.Code
		if got != expected {
			t.Fatalf(mwFmtIncorrectStatus, expected, got)
		}
	}
}

func TestRequireAPIKey(t *testing.T) {
	setupTest(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set(mwContentTypeHeader, mwContentTypeJSON)
	response := httptest.NewRecorder()
	// Test that making a request without an API key is denied
	RequireAPIKey(successHandler).ServeHTTP(response, req)
	expected := http.StatusUnauthorized
	got := response.Code
	if got != expected {
		t.Fatalf(mwFmtIncorrectStatus, expected, got)
	}
}

func TestCORSHeaders(t *testing.T) {
	setupTest(t)
	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	response := httptest.NewRecorder()
	RequireAPIKey(successHandler).ServeHTTP(response, req)
	expected := "POST, GET, OPTIONS, PUT, DELETE"
	got := response.Result().Header.Get("Access-Control-Allow-Methods")
	if got != expected {
		t.Fatalf("incorrect cors options received. expected %s got %s", expected, got)
	}
}

func TestInvalidAPIKey(t *testing.T) {
	setupTest(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	query := req.URL.Query()
	query.Set("api_key", "bogus-api-key")
	req.URL.RawQuery = query.Encode()
	req.Header.Set(mwContentTypeHeader, mwContentTypeJSON)
	response := httptest.NewRecorder()
	RequireAPIKey(successHandler).ServeHTTP(response, req)
	expected := http.StatusUnauthorized
	got := response.Code
	if got != expected {
		t.Fatalf(mwFmtIncorrectStatus, expected, got)
	}
}

func TestBearerToken(t *testing.T) {
	testCtx := setupTest(t)
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", testCtx.apiKey))
	req.Header.Set(mwContentTypeHeader, mwContentTypeJSON)
	response := httptest.NewRecorder()
	RequireAPIKey(successHandler).ServeHTTP(response, req)
	expected := http.StatusOK
	got := response.Code
	if got != expected {
		t.Fatalf(mwFmtIncorrectStatus, expected, got)
	}
}

func TestPasswordResetRequired(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req = ctx.Set(req, "user", models.User{
		PasswordChangeRequired: true,
	})
	response := httptest.NewRecorder()
	RequireLogin(successHandler).ServeHTTP(response, req)
	gotStatus := response.Code
	expectedStatus := http.StatusTemporaryRedirect
	if gotStatus != expectedStatus {
		t.Fatalf(mwFmtIncorrectStatus, expectedStatus, gotStatus)
	}
	expectedLocation := "/reset_password?next=%2F"
	gotLocation := response.Header().Get("Location")
	if gotLocation != expectedLocation {
		t.Fatalf("incorrect location header received. expected %s got %s", expectedLocation, gotLocation)
	}
}

func TestApplySecurityHeaders(t *testing.T) {
	expected := map[string]string{
		"Content-Security-Policy": "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; frame-ancestors 'none';",
		"X-Frame-Options":         "DENY",
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	response := httptest.NewRecorder()
	ApplySecurityHeaders(successHandler).ServeHTTP(response, req)
	for header, value := range expected {
		got := response.Header().Get(header)
		if got != value {
			t.Fatalf("incorrect security header received for %s: expected %s got %s", header, value, got)
		}
	}
}

// TestRequestLogger verifies that the middleware delegates to the inner handler
// and that the status recorder captures the correct code.
func TestRequestLogger(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	})
	req := httptest.NewRequest(http.MethodGet, "/test-path", nil)
	rec := httptest.NewRecorder()
	RequestLogger(inner).ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Errorf("RequestLogger: expected status 201, got %d", rec.Code)
	}
}

// TestRequestLogger_DefaultStatus verifies that when the inner handler does not
// call WriteHeader, the default 200 is captured and passed through.
func TestRequestLogger_DefaultStatus(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	RequestLogger(successHandler).ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("RequestLogger: expected default status 200, got %d", rec.Code)
	}
}

// TestRequireMFAEnrolled verifies the MFA enrollment gate in three scenarios.
func TestRequireMFAEnrolled(t *testing.T) {
	setupTest(t)
	noMFARole, _ := models.GetRoleBySlug(models.RoleCampaignManager)
	mfaRole, _ := models.GetRoleBySlug(models.RoleSuperAdmin)

	t.Run("non-MFA role passes through", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = ctx.Set(req, "user", models.User{Role: noMFARole, RoleID: noMFARole.ID})
		rec := httptest.NewRecorder()
		RequireMFAEnrolled(successHandler).ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("MFA role with no device redirects to enroll", func(t *testing.T) {
		// User ID 9999 has no MFA device record in the test DB.
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = ctx.Set(req, "user", models.User{Id: 9999, Role: mfaRole, RoleID: mfaRole.ID})
		rec := httptest.NewRecorder()
		RequireMFAEnrolled(successHandler).ServeHTTP(rec, req)
		if rec.Code != http.StatusTemporaryRedirect {
			t.Errorf("expected 307, got %d", rec.Code)
		}
		if loc := rec.Header().Get("Location"); loc != "/mfa/enroll" {
			t.Errorf("expected Location /mfa/enroll, got %s", loc)
		}
	})
}

// TestRequireMFAVerified verifies the session MFA challenge gate.
func TestRequireMFAVerified(t *testing.T) {
	setupTest(t)
	noMFARole, _ := models.GetRoleBySlug(models.RoleCampaignManager)
	mfaRole, _ := models.GetRoleBySlug(models.RoleSuperAdmin)

	// mkReq builds a request with the given role and, for MFA-required roles,
	// a session with mfa_verified set to verified.
	mkReq := func(role models.Role, verified bool) *http.Request {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req = ctx.Set(req, "user", models.User{Role: role, RoleID: role.ID})
		if auth.MFARequired(role.Slug) {
			sess := sessions.NewSession(Store, "gophish")
			sess.Values["mfa_verified"] = verified
			req = ctx.Set(req, "session", sess)
		}
		return req
	}

	t.Run("non-MFA role passes through", func(t *testing.T) {
		rec := httptest.NewRecorder()
		RequireMFAVerified(successHandler).ServeHTTP(rec, mkReq(noMFARole, false))
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("MFA role without verified session redirects to verify", func(t *testing.T) {
		rec := httptest.NewRecorder()
		RequireMFAVerified(successHandler).ServeHTTP(rec, mkReq(mfaRole, false))
		if rec.Code != http.StatusTemporaryRedirect {
			t.Errorf("expected 307, got %d", rec.Code)
		}
		if loc := rec.Header().Get("Location"); loc != "/mfa/verify" {
			t.Errorf("expected Location /mfa/verify, got %s", loc)
		}
	})

	t.Run("MFA role with verified session passes through", func(t *testing.T) {
		rec := httptest.NewRecorder()
		RequireMFAVerified(successHandler).ServeHTTP(rec, mkReq(mfaRole, true))
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})
}

// TestRequireReportAccess verifies the report-access gate: admin passes,
// trainer is denied, and unauthenticated requests get 401.
func TestRequireReportAccess(t *testing.T) {
	setupTest(t)

	t.Run("no user in context returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/reports", nil)
		rec := httptest.NewRecorder()
		RequireReportAccess(successHandler).ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401, got %d", rec.Code)
		}
	})

	t.Run("admin user with view_reports passes", func(t *testing.T) {
		u, err := models.GetUser(1)
		if err != nil {
			t.Fatalf("error getting admin user: %v", err)
		}
		req := httptest.NewRequest(http.MethodGet, "/api/reports", nil)
		req = ctx.Set(req, "user", u)
		rec := httptest.NewRecorder()
		RequireReportAccess(successHandler).ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("trainer without view_reports is forbidden", func(t *testing.T) {
		trainerRole, err := models.GetRoleBySlug(models.RoleTrainer)
		if err != nil {
			t.Fatalf("error getting trainer role: %v", err)
		}
		req := httptest.NewRequest(http.MethodGet, "/api/reports", nil)
		req = ctx.Set(req, "user", models.User{Role: trainerRole, RoleID: trainerRole.ID})
		rec := httptest.NewRecorder()
		RequireReportAccess(successHandler).ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Errorf("expected 403, got %d", rec.Code)
		}
	})
}

// TestEnforceTierLimits verifies that GET requests are always passed through
// and that POST requests without a user context return 401.
func TestEnforceTierLimits(t *testing.T) {
	setupTest(t)

	t.Run("GET request bypasses tier check", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/campaigns", nil)
		rec := httptest.NewRecorder()
		EnforceTierLimits("campaign")(successHandler).ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200 for GET, got %d", rec.Code)
		}
	})

	t.Run("POST without user context returns 401", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/api/campaigns", nil)
		rec := httptest.NewRecorder()
		EnforceTierLimits("campaign")(successHandler).ServeHTTP(rec, req)
		if rec.Code != http.StatusUnauthorized {
			t.Errorf("expected 401 for POST without user, got %d", rec.Code)
		}
	})
}
