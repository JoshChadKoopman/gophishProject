package auth

import (
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"
)

// SAMLConfig holds the SAML 2.0 SP configuration, read from config.json.
type SAMLConfig struct {
	Enabled         bool   `json:"enabled"`
	IDPURL          string `json:"idp_url"`           // IdP SSO endpoint URL
	IDPMetadataURL  string `json:"idp_metadata_url"`  // IdP Metadata URL for auto-discovery
	SPEntityID      string `json:"sp_entity_id"`      // e.g. https://app.example.com/saml
	AdminGroupClaim string `json:"admin_group_claim"` // SAML attribute for admin group
	AdminGroupValue string `json:"admin_group_value"` // Value that grants admin access
	DefaultRoleSlug string `json:"default_role_slug"` // Default role for SSO-provisioned users
	SplitAdminUser  bool   `json:"split_admin_user"`  // Enable separate admin/user SSO paths
}

// SAMLClaims contains the user identity extracted from SAML assertions.
type SAMLClaims struct {
	NameID     string
	Email      string
	FirstName  string
	LastName   string
	Groups     []string
	Attributes map[string]string
}

// samlResponse represents the top-level SAML Response XML structure.
type samlResponse struct {
	XMLName   xml.Name      `xml:"Response"`
	Assertion samlAssertion `xml:"Assertion"`
}

// samlAssertion represents a SAML Assertion element.
type samlAssertion struct {
	Subject            samlSubject            `xml:"Subject"`
	AttributeStatement samlAttributeStatement `xml:"AttributeStatement"`
}

// samlSubject represents the SAML Subject element.
type samlSubject struct {
	NameID string `xml:"NameID"`
}

// samlAttributeStatement holds the list of SAML Attribute elements.
type samlAttributeStatement struct {
	Attributes []samlAttribute `xml:"Attribute"`
}

// samlAttribute represents a single SAML Attribute.
type samlAttribute struct {
	Name   string   `xml:"Name,attr"`
	Values []string `xml:"AttributeValue"`
}

// SAMLClient wraps SAML 2.0 configuration for the application.
// A nil SAMLClient means SAML is disabled; callers must check before use.
type SAMLClient struct {
	Config SAMLConfig
}

// NewSAMLClient initialises the SAML configuration.
// Returns (nil, nil) when cfg.Enabled is false.
func NewSAMLClient(cfg SAMLConfig) (*SAMLClient, error) {
	if !cfg.Enabled {
		return nil, nil
	}
	if cfg.IDPURL == "" {
		return nil, errors.New("SAML: idp_url is required")
	}
	if cfg.DefaultRoleSlug == "" {
		cfg.DefaultRoleSlug = "learner"
	}
	return &SAMLClient{Config: cfg}, nil
}

// IsSplitMode returns true if admin/user SSO paths are separate.
func (c *SAMLClient) IsSplitMode() bool {
	return c.Config.SplitAdminUser
}

// AdminLoginURL returns the SAML login URL for the admin SSO path.
func (c *SAMLClient) AdminLoginURL() string {
	if c.Config.SplitAdminUser {
		return "/auth/saml/admin/login"
	}
	return "/auth/saml/login"
}

// UserLoginURL returns the SAML login URL for the user SSO path.
func (c *SAMLClient) UserLoginURL() string {
	if c.Config.SplitAdminUser {
		return "/auth/saml/user/login"
	}
	return "/auth/saml/login"
}

// IDPSSOURL returns the IdP SSO URL for initiating SAML login.
func (c *SAMLClient) IDPSSOURL() string {
	return c.Config.IDPURL
}

// ParseSAMLResponse decodes and parses a base64-encoded SAML Response.
// In production, signature verification against the IdP certificate would
// be performed here. This implementation parses the assertion attributes.
func (c *SAMLClient) ParseSAMLResponse(samlResponseB64 string) (*SAMLClaims, error) {
	rawXML, err := base64.StdEncoding.DecodeString(samlResponseB64)
	if err != nil {
		return nil, fmt.Errorf("SAML: failed to decode response: %w", err)
	}

	var resp samlResponse
	if err := xml.Unmarshal(rawXML, &resp); err != nil {
		return nil, fmt.Errorf("SAML: failed to parse response XML: %w", err)
	}

	attrs := make(map[string]string)
	var groups []string
	for _, a := range resp.Assertion.AttributeStatement.Attributes {
		if len(a.Values) > 0 {
			attrs[a.Name] = a.Values[0]
		}
		// Collect group-like attributes
		lowerName := strings.ToLower(a.Name)
		if lowerName == "groups" || lowerName == "memberof" ||
			strings.Contains(lowerName, "group") {
			groups = append(groups, a.Values...)
		}
	}

	claims := &SAMLClaims{
		NameID:     resp.Assertion.Subject.NameID,
		Email:      firstOf(attrs, "email", "mail", "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/emailaddress"),
		FirstName:  firstOf(attrs, "firstName", "givenName", "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/givenname"),
		LastName:   firstOf(attrs, "lastName", "sn", "http://schemas.xmlsoap.org/ws/2005/05/identity/claims/surname"),
		Groups:     groups,
		Attributes: attrs,
	}

	if claims.Email == "" {
		claims.Email = claims.NameID
	}

	return claims, nil
}

// DetermineRoleSlug maps SAML groups to a Nivoxis role slug.
func (c *SAMLClient) DetermineRoleSlug(claims *SAMLClaims, isAdminPath bool) string {
	if c.Config.SplitAdminUser && isAdminPath {
		return c.resolveAdminRole(claims)
	}
	return c.resolveRoleFromGroups(claims)
}

// resolveAdminRole returns the appropriate admin role based on group verification.
func (c *SAMLClient) resolveAdminRole(claims *SAMLClaims) string {
	if c.Config.AdminGroupValue != "" {
		if containsGroup(claims.Groups, c.Config.AdminGroupValue) {
			return "org_admin"
		}
		return c.Config.DefaultRoleSlug
	}
	return "org_admin"
}

// resolveRoleFromGroups maps SAML groups to a role slug.
func (c *SAMLClient) resolveRoleFromGroups(claims *SAMLClaims) string {
	if c.Config.AdminGroupValue != "" && containsGroup(claims.Groups, c.Config.AdminGroupValue) {
		return "org_admin"
	}

	groupRoleMap := map[string]string{
		"superadmin":       "superadmin",
		"org_admin":        "org_admin",
		"campaign_manager": "campaign_manager",
		"trainer":          "trainer",
		"auditor":          "auditor",
	}
	for _, g := range claims.Groups {
		if role, ok := groupRoleMap[strings.ToLower(g)]; ok {
			return role
		}
	}
	return c.Config.DefaultRoleSlug
}

// ── Helpers ──

// firstOf returns the first non-empty value from the attrs map for the given keys.
func firstOf(attrs map[string]string, keys ...string) string {
	for _, k := range keys {
		if v, ok := attrs[k]; ok && v != "" {
			return v
		}
	}
	return ""
}

// containsGroup checks if a group name exists in a slice (case-insensitive).
func containsGroup(groups []string, target string) bool {
	lower := strings.ToLower(target)
	for _, g := range groups {
		if strings.ToLower(g) == lower {
			return true
		}
	}
	return false
}
