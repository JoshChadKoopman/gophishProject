package middleware

import (
	"net/http"

	"github.com/gophish/gophish/models"
)

// RequireMSPPartner checks that the authenticated user is associated with an
// MSP partner (either as the primary admin or through the msp_partner role).
// If the user is a superadmin, access is granted unconditionally.
func RequireMSPPartner(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := getUserFromContext(r)
		if !ok {
			JSONError(w, http.StatusUnauthorized, errUserNotAuthenticated)
			return
		}
		if user.Role.Slug == models.RoleSuperAdmin {
			next.ServeHTTP(w, r)
			return
		}
		if !models.UserIsMSPPartner(user.Id) {
			JSONError(w, http.StatusForbidden, "MSP partner access required")
			return
		}
		next.ServeHTTP(w, r)
	}
}

// RequireMSPWhitelabel checks that the authenticated user's organization has
// the msp_whitelabel feature enabled (or is a superadmin).
func RequireMSPWhitelabel(next http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := getUserFromContext(r)
		if !ok {
			JSONError(w, http.StatusUnauthorized, errUserNotAuthenticated)
			return
		}
		if user.Role.Slug == models.RoleSuperAdmin {
			next.ServeHTTP(w, r)
			return
		}
		if !models.OrgHasFeature(user.OrgId, models.FeatureMSPWhitelabel) {
			JSONError(w, http.StatusForbidden, "White-label branding requires a plan upgrade")
			return
		}
		next.ServeHTTP(w, r)
	}
}
