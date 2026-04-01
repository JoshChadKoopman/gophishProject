package middleware

import (
	"net/http"

	"github.com/gophish/gophish/models"
)

// RequireFeature checks that the authenticated user's organization has the
// specified feature enabled via its subscription tier. If the feature is not
// available, a 403 JSON error is returned prompting the user to upgrade.
func RequireFeature(featureSlug string) func(http.Handler) http.HandlerFunc {
	return func(next http.Handler) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			user, ok := getUserFromContext(r)
			if !ok {
				JSONError(w, http.StatusUnauthorized, errUserNotAuthenticated)
				return
			}
			if !models.OrgHasFeature(user.OrgId, featureSlug) {
				JSONError(w, http.StatusForbidden, "This feature requires a plan upgrade")
				return
			}
			next.ServeHTTP(w, r)
		}
	}
}
