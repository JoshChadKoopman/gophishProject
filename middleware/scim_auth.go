package middleware

import (
	"encoding/json"
	"net/http"
	"strings"

	ctx "github.com/gophish/gophish/context"
	"github.com/gophish/gophish/models"
)

// RequireSCIMToken authenticates SCIM requests using a per-org bearer token.
// On success it stores the org_id in the request context under "scim_org_id".
// SCIM tokens are separate from user API keys — they represent the IdP itself.
func RequireSCIMToken(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var token string
		if tokens, ok := r.Header["Authorization"]; ok && len(tokens) >= 1 {
			token = strings.TrimPrefix(tokens[0], "Bearer ")
		}
		if token == "" {
			scimError(w, http.StatusUnauthorized, "Bearer token required")
			return
		}

		orgId, err := models.ValidateSCIMToken(token)
		if err != nil {
			scimError(w, http.StatusUnauthorized, "Invalid or inactive SCIM token")
			return
		}

		// Check that the org has the SCIM feature enabled
		if !models.OrgHasFeature(orgId, models.FeatureSCIM) {
			scimError(w, http.StatusForbidden, "SCIM provisioning requires a plan upgrade")
			return
		}

		r = ctx.Set(r, "scim_org_id", orgId)
		next.ServeHTTP(w, r)
	})
}

// scimError writes a SCIM-formatted JSON error response.
// Uses json.Marshal to safely escape the detail string, preventing injection.
func scimError(w http.ResponseWriter, status int, detail string) {
	w.Header().Set("Content-Type", "application/scim+json")
	w.WriteHeader(status)
	resp := struct {
		Schemas []string `json:"schemas"`
		Detail  string   `json:"detail"`
		Status  string   `json:"status"`
	}{
		Schemas: []string{"urn:ietf:params:scim:api:messages:2.0:Error"},
		Detail:  detail,
		Status:  http.StatusText(status),
	}
	json.NewEncoder(w).Encode(resp)
}
