package controllers

import (
	"net/http"
	"strings"

	"github.com/gophish/gophish/auth"
	ctx "github.com/gophish/gophish/context"
	log "github.com/gophish/gophish/logger"
	"github.com/gophish/gophish/models"
	"github.com/gorilla/sessions"
)

// SAMLLogin redirects the browser to the IdP SSO endpoint (default path).
func (as *AdminServer) SAMLLogin(w http.ResponseWriter, r *http.Request) {
	if as.samlClient == nil {
		http.Redirect(w, r, routeLogin, http.StatusFound)
		return
	}
	http.Redirect(w, r, as.samlClient.IDPSSOURL(), http.StatusFound)
}

// SAMLAdminLogin redirects to IdP with admin-specific RelayState.
func (as *AdminServer) SAMLAdminLogin(w http.ResponseWriter, r *http.Request) {
	if as.samlClient == nil || !as.samlClient.IsSplitMode() {
		http.Redirect(w, r, routeLogin, http.StatusFound)
		return
	}
	idpURL := as.samlClient.IDPSSOURL() + "?RelayState=admin"
	http.Redirect(w, r, idpURL, http.StatusFound)
}

// SAMLUserLogin redirects to IdP with user-specific RelayState.
func (as *AdminServer) SAMLUserLogin(w http.ResponseWriter, r *http.Request) {
	if as.samlClient == nil || !as.samlClient.IsSplitMode() {
		http.Redirect(w, r, routeLogin, http.StatusFound)
		return
	}
	idpURL := as.samlClient.IDPSSOURL() + "?RelayState=user"
	http.Redirect(w, r, idpURL, http.StatusFound)
}

// SAMLCallback handles the IdP POST-back to the default ACS endpoint.
func (as *AdminServer) SAMLCallback(w http.ResponseWriter, r *http.Request) {
	as.handleSAMLCallback(w, r, false)
}

// SAMLAdminCallback handles the admin-specific ACS endpoint.
func (as *AdminServer) SAMLAdminCallback(w http.ResponseWriter, r *http.Request) {
	as.handleSAMLCallback(w, r, true)
}

// SAMLUserCallback handles the user-specific ACS endpoint.
func (as *AdminServer) SAMLUserCallback(w http.ResponseWriter, r *http.Request) {
	as.handleSAMLCallback(w, r, false)
}

// handleSAMLCallback is the shared SAML assertion consumer logic.
func (as *AdminServer) handleSAMLCallback(w http.ResponseWriter, r *http.Request, isAdminPath bool) {
	if as.samlClient == nil {
		http.Redirect(w, r, routeLogin, http.StatusFound)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "SAML: expected POST", http.StatusMethodNotAllowed)
		return
	}

	samlResponse := r.FormValue("SAMLResponse")
	if samlResponse == "" {
		http.Error(w, "SAML: missing SAMLResponse", http.StatusBadRequest)
		return
	}

	// Check RelayState for admin path determination
	relayState := r.FormValue("RelayState")
	if strings.ToLower(relayState) == "admin" {
		isAdminPath = true
	}

	claims, err := as.samlClient.ParseSAMLResponse(samlResponse)
	if err != nil {
		log.Errorf("SAML callback: assertion parse error: %v", err)
		http.Error(w, "Authentication failed", http.StatusUnauthorized)
		return
	}

	if claims.Email == "" {
		log.Error("SAML callback: no email in assertion")
		http.Error(w, "Authentication failed: no email provided", http.StatusUnauthorized)
		return
	}

	// Find or create the local user record
	u, err := models.GetUserByEmail(claims.Email)
	if err != nil {
		// Auto-provision the user with the role from SAML claims
		roleSlug := as.samlClient.DetermineRoleSlug(claims, isAdminPath)
		role, roleErr := models.GetRoleBySlug(roleSlug)
		if roleErr != nil {
			log.Errorf("SAML callback: unknown role %s: %v", roleSlug, roleErr)
			http.Error(w, "Configuration error", http.StatusInternalServerError)
			return
		}
		// Resolve the organization — use org attribute from SAML or default to org 1
		orgId := resolveOrgFromSAMLClaims(claims)
		u = models.User{
			Username:  claims.Email,
			Email:     claims.Email,
			FirstName: claims.FirstName,
			LastName:  claims.LastName,
			OrgId:     orgId,
			ApiKey:    auth.GenerateSecureKey(auth.APIKeyLength),
			Role:      role,
			RoleID:    role.ID,
		}
		if putErr := models.PutUser(&u); putErr != nil {
			log.Errorf("SAML callback: failed to create user: %v", putErr)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}
		log.Infof("SAML: auto-provisioned user %s (org=%d, role=%s)", u.Email, orgId, roleSlug)
	} else {
		// Existing user — sync profile fields from SAML assertion on each login
		changed := false
		if claims.FirstName != "" && claims.FirstName != u.FirstName {
			u.FirstName = claims.FirstName
			changed = true
		}
		if claims.LastName != "" && claims.LastName != u.LastName {
			u.LastName = claims.LastName
			changed = true
		}
		// Update role if admin path and user doesn't already have the expected role
		if isAdminPath {
			roleSlug := as.samlClient.DetermineRoleSlug(claims, isAdminPath)
			if roleSlug != u.Role.Slug {
				role, roleErr := models.GetRoleBySlug(roleSlug)
				if roleErr == nil {
					u.RoleID = role.ID
					u.Role = role
					changed = true
				}
			}
		}
		if changed {
			models.PutUser(&u)
		}
	}

	// SAML IdP handled MFA, grant MFA credit
	session := ctx.Get(r, "session").(*sessions.Session)
	session.Values["id"] = u.Id
	session.Values["mfa_verified"] = true
	session.Save(r, w)
	as.nextOrIndex(w, r, &u)
}

// resolveOrgFromSAMLClaims resolves the target organization from SAML assertion
// attributes. Looks for common org-identifying attributes. Falls back to org ID 1.
func resolveOrgFromSAMLClaims(claims *auth.SAMLClaims) int64 {
	// Check for explicit org attribute in SAML assertion
	orgKeys := []string{"organization", "org", "tenant", "company"}
	for _, key := range orgKeys {
		if val, ok := claims.Attributes[key]; ok && val != "" {
			org, err := models.GetOrganizationBySlug(strings.ToLower(val))
			if err == nil {
				return org.Id
			}
		}
	}
	// Default to the first organization
	return 1
}
