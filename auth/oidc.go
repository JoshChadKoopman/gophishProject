package auth

import (
	"context"
	"errors"
	"fmt"

	gooidc "github.com/coreos/go-oidc/v3/oidc"
	"golang.org/x/oauth2"
)

// OIDCClaims contains the user identity claims extracted from a Keycloak ID token.
type OIDCClaims struct {
	Sub   string   `json:"sub"`
	Email string   `json:"email"`
	Name  string   `json:"name"`
	Roles []string `json:"roles"` // mapped from Keycloak realm_access.roles
}

// keycloakRealmAccess is used to decode Keycloak's realm_access claim.
type keycloakRealmAccess struct {
	Roles []string `json:"roles"`
}

// OIDCClient wraps the go-oidc provider and oauth2 config for Keycloak.
// A nil OIDCClient means OIDC is disabled; callers must check before use.
type OIDCClient struct {
	providerURL  string
	oauth2Config oauth2.Config
	verifier     *gooidc.IDTokenVerifier
}

// NewOIDCClient initialises the OIDC provider by fetching its discovery document.
// Returns (nil, nil) when cfg.Enabled is false so callers can use a nil check.
// Returns an error if the provider is unreachable or misconfigured.
func NewOIDCClient(providerURL, clientID, clientSecret, redirectURL string, enabled bool) (*OIDCClient, error) {
	if !enabled {
		return nil, nil
	}
	if providerURL == "" || clientID == "" {
		return nil, errors.New("OIDC: provider_url and client_id are required")
	}

	provider, err := gooidc.NewProvider(context.Background(), providerURL)
	if err != nil {
		return nil, fmt.Errorf("OIDC: failed to fetch provider discovery document from %s: %w", providerURL, err)
	}

	oc := &OIDCClient{
		providerURL: providerURL,
		oauth2Config: oauth2.Config{
			ClientID:     clientID,
			ClientSecret: clientSecret,
			RedirectURL:  redirectURL,
			Endpoint:     provider.Endpoint(),
			Scopes:       []string{gooidc.ScopeOpenID, "email", "profile", "roles"},
		},
		verifier: provider.Verifier(&gooidc.Config{ClientID: clientID}),
	}
	return oc, nil
}

// LogoutURL returns the Keycloak end-session endpoint URL.
func (c *OIDCClient) LogoutURL() string {
	return c.providerURL + "/protocol/openid-connect/logout"
}

// AuthCodeURL returns the OAuth2 authorisation redirect URL with the given
// state and nonce embedded. Store both in the user's session before redirecting.
func (c *OIDCClient) AuthCodeURL(state, nonce string) string {
	return c.oauth2Config.AuthCodeURL(
		state,
		oauth2.SetAuthURLParam("nonce", nonce),
	)
}

// Exchange converts an authorisation code into a verified set of claims.
// It validates the ID token signature, expiry, nonce, and audience.
func (c *OIDCClient) Exchange(ctx context.Context, code, nonce string) (*OIDCClaims, error) {
	token, err := c.oauth2Config.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("OIDC: token exchange failed: %w", err)
	}

	rawIDToken, ok := token.Extra("id_token").(string)
	if !ok {
		return nil, errors.New("OIDC: no id_token in token response")
	}

	idToken, err := c.verifier.Verify(ctx, rawIDToken)
	if err != nil {
		return nil, fmt.Errorf("OIDC: id_token verification failed: %w", err)
	}

	// Verify the nonce to prevent replay attacks.
	var raw map[string]interface{}
	if err := idToken.Claims(&raw); err != nil {
		return nil, fmt.Errorf("OIDC: failed to parse claims: %w", err)
	}
	if tokenNonce, _ := raw["nonce"].(string); tokenNonce != nonce {
		return nil, errors.New("OIDC: nonce mismatch")
	}

	claims := &OIDCClaims{
		Sub:   idToken.Subject,
		Email: stringClaim(raw, "email"),
		Name:  stringClaim(raw, "name"),
	}

	// Extract Keycloak realm roles from realm_access.roles claim.
	if ra, ok := raw["realm_access"].(map[string]interface{}); ok {
		if roles, ok := ra["roles"].([]interface{}); ok {
			for _, r := range roles {
				if s, ok := r.(string); ok {
					claims.Roles = append(claims.Roles, s)
				}
			}
		}
	}

	return claims, nil
}

// ExtractRoleSlug maps a list of Keycloak realm role names to the highest-
// privilege Nivoxis role slug. The priority order mirrors the permission
// hierarchy: superadmin > org_admin > campaign_manager > trainer > auditor > learner.
func ExtractRoleSlug(keycloakRoles []string) string {
	roleSet := make(map[string]bool, len(keycloakRoles))
	for _, r := range keycloakRoles {
		roleSet[r] = true
	}

	priority := []string{"superadmin", "org_admin", "campaign_manager", "trainer", "auditor", "learner"}
	for _, slug := range priority {
		if roleSet[slug] {
			return slug
		}
	}
	return "learner" // safe default
}

// stringClaim safely extracts a string claim from a raw claims map.
func stringClaim(raw map[string]interface{}, key string) string {
	if v, ok := raw[key].(string); ok {
		return v
	}
	return ""
}
