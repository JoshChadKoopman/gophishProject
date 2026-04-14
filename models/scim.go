package models

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/gophish/gophish/auth"
	log "github.com/gophish/gophish/logger"
)

// SCIMToken stores a hashed bearer token used by IdPs to authenticate SCIM requests.
type SCIMToken struct {
	Id          int64     `json:"id" gorm:"column:id; primary_key:yes"`
	OrgId       int64     `json:"org_id" gorm:"column:org_id"`
	TokenHash   string    `json:"-" gorm:"column:token_hash"`
	Description string    `json:"description" gorm:"column:description"`
	CreatedBy   int64     `json:"created_by" gorm:"column:created_by"`
	IsActive    bool      `json:"is_active" gorm:"column:is_active"`
	LastUsed    time.Time `json:"last_used" gorm:"column:last_used"`
	CreatedDate time.Time `json:"created_date" gorm:"column:created_date"`
}

// SCIMExternalID maps an IdP external ID to an internal resource.
type SCIMExternalID struct {
	Id           int64  `json:"id" gorm:"column:id; primary_key:yes"`
	OrgId        int64  `json:"org_id" gorm:"column:org_id"`
	ResourceType string `json:"resource_type" gorm:"column:resource_type"` // "User" or "Group"
	ExternalId   string `json:"external_id" gorm:"column:external_id"`
	InternalId   int64  `json:"internal_id" gorm:"column:internal_id"`
}

// SCIM resource schemas.
const (
	SCIMSchemaUser           = "urn:ietf:params:scim:schemas:core:2.0:User"
	SCIMSchemaEnterpriseUser = "urn:ietf:params:scim:schemas:extension:enterprise:2.0:User"
	SCIMSchemaGroup          = "urn:ietf:params:scim:schemas:core:2.0:Group"
	SCIMSchemaListResponse   = "urn:ietf:params:scim:api:messages:2.0:ListResponse"
	SCIMSchemaError          = "urn:ietf:params:scim:api:messages:2.0:Error"
)

var ErrSCIMTokenNotFound = errors.New("SCIM token not found")

// queryWhereOrgResInternal is the shared WHERE clause for SCIM external ID lookups.
const queryWhereOrgResInternal = "org_id = ? AND resource_type = ? AND internal_id = ?"

// hashSCIMToken returns the SHA-256 hex hash of a raw SCIM token.
func hashSCIMToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

// CreateSCIMToken generates a new SCIM bearer token for an org.
// Returns the raw token (shown once) and the stored record.
func CreateSCIMToken(orgId, createdBy int64, description string) (string, SCIMToken, error) {
	raw := "scim_" + auth.GenerateSecureKey(32)
	t := SCIMToken{
		OrgId:       orgId,
		TokenHash:   hashSCIMToken(raw),
		Description: description,
		CreatedBy:   createdBy,
		IsActive:    true,
		CreatedDate: time.Now().UTC(),
	}
	err := db.Save(&t).Error
	return raw, t, err
}

// GetSCIMTokens returns all SCIM tokens for an org (hashes are excluded from JSON).
func GetSCIMTokens(orgId int64) ([]SCIMToken, error) {
	tokens := []SCIMToken{}
	err := db.Where("org_id = ?", orgId).Order("created_date desc").Find(&tokens).Error
	return tokens, err
}

// DeleteSCIMToken deactivates a SCIM token.
func DeleteSCIMToken(id, orgId int64) error {
	return db.Where("id = ? AND org_id = ?", id, orgId).Delete(&SCIMToken{}).Error
}

// ValidateSCIMToken checks a raw bearer token and returns the org ID if valid.
func ValidateSCIMToken(raw string) (int64, error) {
	h := hashSCIMToken(raw)
	t := SCIMToken{}
	err := db.Where("token_hash = ? AND is_active = ?", h, true).First(&t).Error
	if err != nil {
		return 0, ErrSCIMTokenNotFound
	}
	// Update last_used timestamp (fire-and-forget)
	db.Model(&t).Update("last_used", time.Now().UTC())
	return t.OrgId, nil
}

// --- SCIM External ID mapping ---

// SetSCIMExternalID creates or updates the external ID mapping for a resource.
func SetSCIMExternalID(orgId int64, resourceType, externalId string, internalId int64) error {
	existing := SCIMExternalID{}
	err := db.Where(queryWhereOrgResInternal,
		orgId, resourceType, internalId).First(&existing).Error
	if err == nil {
		// Update existing mapping
		return db.Model(&existing).Update("external_id", externalId).Error
	}
	// Create new mapping
	m := SCIMExternalID{
		OrgId:        orgId,
		ResourceType: resourceType,
		ExternalId:   externalId,
		InternalId:   internalId,
	}
	return db.Save(&m).Error
}

// GetSCIMExternalID returns the external ID for an internal resource.
func GetSCIMExternalID(orgId int64, resourceType string, internalId int64) string {
	m := SCIMExternalID{}
	err := db.Where(queryWhereOrgResInternal,
		orgId, resourceType, internalId).First(&m).Error
	if err != nil {
		return ""
	}
	return m.ExternalId
}

// GetInternalIDByExternalID looks up the internal ID from an external ID.
func GetInternalIDByExternalID(orgId int64, resourceType, externalId string) (int64, error) {
	m := SCIMExternalID{}
	err := db.Where("org_id = ? AND resource_type = ? AND external_id = ?",
		orgId, resourceType, externalId).First(&m).Error
	if err != nil {
		return 0, err
	}
	return m.InternalId, nil
}

// DeleteSCIMExternalID removes the mapping for a deleted resource.
func DeleteSCIMExternalID(orgId int64, resourceType string, internalId int64) {
	db.Where(queryWhereOrgResInternal,
		orgId, resourceType, internalId).Delete(&SCIMExternalID{})
}

// --- SCIM User Provisioning ---

// SCIMProvisionUser creates a new user from SCIM attributes.
func SCIMProvisionUser(orgId int64, username, email, firstName, lastName, department, jobTitle string) (User, error) {
	// Generate a random password (user will authenticate via SSO)
	randomPass := auth.GenerateSecureKey(16)
	hash, err := auth.GeneratePasswordHash(randomPass)
	if err != nil {
		return User{}, fmt.Errorf("hash password: %w", err)
	}

	// Default to learner role for SCIM-provisioned users
	role, err := GetRoleBySlug(RoleLearner)
	if err != nil {
		return User{}, fmt.Errorf("get learner role: %w", err)
	}

	u := User{
		Username:               username,
		Hash:                   hash,
		ApiKey:                 auth.GenerateSecureKey(auth.APIKeyLength),
		FirstName:              firstName,
		LastName:               lastName,
		Email:                  email,
		RoleID:                 role.ID,
		OrgId:                  orgId,
		Department:             department,
		JobTitle:               jobTitle,
		PasswordChangeRequired: true,
	}

	err = db.Save(&u).Error
	if err != nil {
		return User{}, err
	}

	// Reload with role preloaded
	u, err = GetUser(u.Id)
	return u, err
}

// SCIMUpdateUser updates user attributes from SCIM.
func SCIMUpdateUser(u *User, firstName, lastName, email, department, jobTitle string, active bool) error {
	u.FirstName = firstName
	u.LastName = lastName
	u.Email = email
	u.Department = department
	u.JobTitle = jobTitle
	u.AccountLocked = !active
	return PutUser(u)
}

// SCIMDeactivateUser locks a user account (SCIM "active: false" or DELETE).
func SCIMDeactivateUser(userId int64) error {
	return db.Model(&User{}).Where("id = ?", userId).
		Update("account_locked", true).Error
}

// --- SCIM JSON Resource Types ---

// SCIMUserResource represents a SCIM 2.0 User resource for JSON serialization.
type SCIMUserResource struct {
	Schemas    []string            `json:"schemas"`
	Id         string              `json:"id"`
	ExternalId string              `json:"externalId,omitempty"`
	UserName   string              `json:"userName"`
	Name       SCIMName            `json:"name"`
	Emails     []SCIMEmail         `json:"emails,omitempty"`
	Active     bool                `json:"active"`
	Title      string              `json:"title,omitempty"`
	Enterprise *SCIMEnterpriseUser `json:"urn:ietf:params:scim:schemas:extension:enterprise:2.0:User,omitempty"`
	Meta       SCIMMeta            `json:"meta"`
}

type SCIMName struct {
	GivenName  string `json:"givenName"`
	FamilyName string `json:"familyName"`
	Formatted  string `json:"formatted,omitempty"`
}

type SCIMEmail struct {
	Value   string `json:"value"`
	Type    string `json:"type,omitempty"`
	Primary bool   `json:"primary"`
}

type SCIMEnterpriseUser struct {
	Department string `json:"department,omitempty"`
}

type SCIMMeta struct {
	ResourceType string `json:"resourceType"`
	Created      string `json:"created,omitempty"`
	LastModified string `json:"lastModified,omitempty"`
	Location     string `json:"location,omitempty"`
}

// SCIMGroupResource represents a SCIM 2.0 Group resource.
type SCIMGroupResource struct {
	Schemas     []string     `json:"schemas"`
	Id          string       `json:"id"`
	ExternalId  string       `json:"externalId,omitempty"`
	DisplayName string       `json:"displayName"`
	Members     []SCIMMember `json:"members,omitempty"`
	Meta        SCIMMeta     `json:"meta"`
}

type SCIMMember struct {
	Value   string `json:"value"`
	Display string `json:"display,omitempty"`
	Ref     string `json:"$ref,omitempty"`
}

// SCIMListResponse wraps a list of SCIM resources.
type SCIMListResponse struct {
	Schemas      []string    `json:"schemas"`
	TotalResults int         `json:"totalResults"`
	StartIndex   int         `json:"startIndex"`
	ItemsPerPage int         `json:"itemsPerPage"`
	Resources    interface{} `json:"Resources"`
}

// SCIMErrorResponse represents a SCIM error.
type SCIMErrorResponse struct {
	Schemas  []string `json:"schemas"`
	Detail   string   `json:"detail"`
	Status   string   `json:"status"`
	ScimType string   `json:"scimType,omitempty"`
}

// UserToSCIMResource converts a GoPhish User to a SCIM User resource.
func UserToSCIMResource(u User, orgId int64, baseURL string) SCIMUserResource {
	schemas := []string{SCIMSchemaUser}
	var enterprise *SCIMEnterpriseUser
	if u.Department != "" {
		schemas = append(schemas, SCIMSchemaEnterpriseUser)
		enterprise = &SCIMEnterpriseUser{Department: u.Department}
	}

	extId := GetSCIMExternalID(orgId, "User", u.Id)
	idStr := strconv.FormatInt(u.Id, 10)

	return SCIMUserResource{
		Schemas:    schemas,
		Id:         idStr,
		ExternalId: extId,
		UserName:   u.Username,
		Name: SCIMName{
			GivenName:  u.FirstName,
			FamilyName: u.LastName,
			Formatted:  u.FirstName + " " + u.LastName,
		},
		Emails: []SCIMEmail{{
			Value:   u.Email,
			Type:    "work",
			Primary: true,
		}},
		Active:     !u.AccountLocked,
		Title:      u.JobTitle,
		Enterprise: enterprise,
		Meta: SCIMMeta{
			ResourceType: "User",
			Location:     baseURL + "/scim/v2/Users/" + idStr,
		},
	}
}

// GroupToSCIMResource converts a GoPhish Group to a SCIM Group resource.
func GroupToSCIMResource(g Group, orgId int64, baseURL string) SCIMGroupResource {
	idStr := strconv.FormatInt(g.Id, 10)
	extId := GetSCIMExternalID(orgId, "Group", g.Id)

	members := make([]SCIMMember, 0, len(g.Targets))
	for _, t := range g.Targets {
		tidStr := strconv.FormatInt(t.Id, 10)
		members = append(members, SCIMMember{
			Value:   tidStr,
			Display: t.FirstName + " " + t.LastName,
			Ref:     baseURL + "/scim/v2/Users/" + tidStr,
		})
	}

	return SCIMGroupResource{
		Schemas:     []string{SCIMSchemaGroup},
		Id:          idStr,
		ExternalId:  extId,
		DisplayName: g.Name,
		Members:     members,
		Meta: SCIMMeta{
			ResourceType: "Group",
			Location:     baseURL + "/scim/v2/Groups/" + idStr,
		},
	}
}

// SCIMLog logs a SCIM operation for audit trail.
func SCIMLog(orgId int64, action, resourceType, resourceId, detail string) {
	log.Infof("SCIM [org=%d] %s %s/%s: %s", orgId, action, resourceType, resourceId, detail)
}
