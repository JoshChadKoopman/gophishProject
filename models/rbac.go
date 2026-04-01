package models

/*
Design:

Gophish implements simple Role-Based-Access-Control (RBAC) to control access to
certain resources.

By default, Gophish has two separate roles, with each user being assigned to
a single role:

* Admin  - Can modify all objects as well as system-level configuration
* User   - Can modify all objects

It's important to note that these are global roles. In the future, we'll likely
add the concept of teams, which will include their own roles and permission
system similar to these global permissions.

Each role maps to one or more permissions, making it easy to add more granular
permissions over time.

This is supported through a simple API on a user object,
`HasPermission(Permission)`, which returns a boolean and an error.
This API checks the role associated with the user to see if that role has the
requested permission.
*/

const (
	// Nivoxis platform roles

	// RoleSuperAdmin is for Nivoxis staff with full platform access across all tenants.
	RoleSuperAdmin = "superadmin"
	// RoleOrgAdmin is for client HR admins who manage their org's campaigns and users.
	RoleOrgAdmin = "org_admin"
	// RoleCampaignManager can create and launch phishing campaigns and view results.
	RoleCampaignManager = "campaign_manager"
	// RoleTrainer can assign and manage training modules only.
	RoleTrainer = "trainer"
	// RoleLearner is an end user who completes training and views their own results.
	RoleLearner = "learner"
	// RoleAuditor has read-only access to reports and audit logs.
	RoleAuditor = "auditor"

	// Legacy aliases — resolve to Nivoxis slugs so existing call sites compile
	// correctly after the DB migration renames the old slugs.
	RoleAdmin       = RoleSuperAdmin      // was "admin"
	RoleUser        = RoleCampaignManager // was "user"
	RoleContributor = RoleTrainer         // was "contributor"
	RoleReader      = RoleLearner         // was "reader"

	// PermissionViewObjects determines if a role can view standard Gophish
	// objects such as campaigns, groups, landing pages, etc.
	PermissionViewObjects = "view_objects"
	// PermissionModifyObjects determines if a role can create and modify
	// standard Gophish objects.
	PermissionModifyObjects = "modify_objects"
	// PermissionModifySystem determines if a role can manage system-level
	// configuration.
	PermissionModifySystem = "modify_system"
	// PermissionManageTraining determines if a role can create, edit, and
	// delete training modules and presentations.
	PermissionManageTraining = "manage_training"
	// PermissionViewReports determines if a role has read-only access to
	// reports and audit logs.
	PermissionViewReports = "view_reports"
)

// Role represents a user role within Gophish. Each user has a single role
// which maps to a set of permissions.
type Role struct {
	ID          int64        `json:"-"`
	Slug        string       `json:"slug"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Permissions []Permission `json:"-" gorm:"many2many:role_permissions;"`
}

// Permission determines what a particular role can do. Each role may have one
// or more permissions.
type Permission struct {
	ID          int64  `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

// GetRoleBySlug returns a role that can be assigned to a user.
func GetRoleBySlug(slug string) (Role, error) {
	role := Role{}
	err := db.Where("slug=?", slug).First(&role).Error
	return role, err
}

// GetRoles returns all available roles.
func GetRoles() ([]Role, error) {
	roles := []Role{}
	err := db.Find(&roles).Error
	return roles, err
}

// HasPermission checks to see if the user has a role with the requested
// permission.
func (u *User) HasPermission(slug string) (bool, error) {
	perm := []Permission{}
	err := db.Model(Role{ID: u.RoleID}).Where("slug=?", slug).Association("Permissions").Find(&perm).Error
	if err != nil {
		return false, err
	}
	// Gorm doesn't return an ErrRecordNotFound whe scanning into a slice, so
	// we need to check the length (ref jinzhu/gorm#228)
	if len(perm) == 0 {
		return false, nil
	}
	return true, nil
}
