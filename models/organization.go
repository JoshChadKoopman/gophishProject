package models

import (
	"errors"
	"time"

	log "github.com/gophish/gophish/logger"
	"github.com/jinzhu/gorm"
)

// Organization represents a tenant in the multi-tenant platform.
type Organization struct {
	Id                    int64      `json:"id" gorm:"primary_key"`
	Name                  string     `json:"name" sql:"not null"`
	Slug                  string     `json:"slug" sql:"not null;unique"`
	Tier                  string     `json:"tier" sql:"not null;default:'free'"`
	TierId                int64      `json:"tier_id" gorm:"column:tier_id;default:4"`
	MaxUsers              int        `json:"max_users"`
	MaxCampaigns          int        `json:"max_campaigns"`
	LogoURL               string     `json:"logo_url"`
	PrimaryColor          string     `json:"primary_color"`
	SubscriptionExpiresAt *time.Time `json:"subscription_expires_at" gorm:"column:subscription_expires_at"`
	DefaultLanguage       string     `json:"default_language" gorm:"column:default_language;default:'en'"`
	CreatedDate           time.Time  `json:"created_date"`
	ModifiedDate          time.Time  `json:"modified_date"`
}

// OrgScope replaces the uid parameter in all data-access functions.
// It carries the org and user context needed for scoped queries.
type OrgScope struct {
	OrgId        int64
	UserId       int64
	IsSuperAdmin bool
}

// ErrOrgNameNotSpecified is thrown when no org name is provided.
var ErrOrgNameNotSpecified = errors.New("Organization name not specified")

// ErrOrgSlugNotSpecified is thrown when no org slug is provided.
var ErrOrgSlugNotSpecified = errors.New("Organization slug not specified")

// ErrOrgNotFound is thrown when the requested organization does not exist.
var ErrOrgNotFound = errors.New("Organization not found")

// scopeQuery adds org_id filtering to a GORM query.
// Superadmins bypass org scoping (see all data).
func scopeQuery(query *gorm.DB, scope OrgScope) *gorm.DB {
	if scope.IsSuperAdmin {
		return query
	}
	return query.Where("org_id = ?", scope.OrgId)
}

// scopeRawSQL returns a SQL fragment and value for org_id filtering.
// For superadmins, it returns a tautology ("1=1") so the query sees all data.
func scopeRawSQL(scope OrgScope) (string, interface{}) {
	if scope.IsSuperAdmin {
		return "1=1", nil
	}
	return "org_id = ?", scope.OrgId
}

// GetOrganization returns the organization with the given id.
func GetOrganization(id int64) (Organization, error) {
	o := Organization{}
	err := db.Where("id = ?", id).First(&o).Error
	return o, err
}

// GetOrganizationBySlug returns the organization with the given slug.
func GetOrganizationBySlug(slug string) (Organization, error) {
	o := Organization{}
	err := db.Where("slug = ?", slug).First(&o).Error
	return o, err
}

// GetOrganizations returns all organizations.
func GetOrganizations() ([]Organization, error) {
	orgs := []Organization{}
	err := db.Find(&orgs).Error
	return orgs, err
}

// PostOrganization creates a new organization.
func PostOrganization(o *Organization) error {
	if o.Name == "" {
		return ErrOrgNameNotSpecified
	}
	if o.Slug == "" {
		return ErrOrgSlugNotSpecified
	}
	o.CreatedDate = time.Now().UTC()
	o.ModifiedDate = time.Now().UTC()
	return db.Save(o).Error
}

// PutOrganization updates an existing organization.
func PutOrganization(o *Organization) error {
	if o.Name == "" {
		return ErrOrgNameNotSpecified
	}
	o.ModifiedDate = time.Now().UTC()
	return db.Save(o).Error
}

// DeleteOrganization deletes the organization with the given id.
func DeleteOrganization(id int64) error {
	return db.Where("id = ?", id).Delete(&Organization{}).Error
}

// GetOrgUserCount returns the number of users in the given org.
func GetOrgUserCount(orgId int64) (int, error) {
	var count int
	err := db.Model(&User{}).Where("org_id = ?", orgId).Count(&count).Error
	if err != nil {
		log.Error(err)
	}
	return count, err
}

// GetOrgCampaignCount returns the number of campaigns in the given org.
func GetOrgCampaignCount(orgId int64) (int, error) {
	var count int
	err := db.Table("campaigns").Where("org_id = ?", orgId).Count(&count).Error
	if err != nil {
		log.Error(err)
	}
	return count, err
}
