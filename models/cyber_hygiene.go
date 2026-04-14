package models

import (
	"errors"
	"time"
)

// Cyber Hygiene — My Apps & Devices module.
// Users self-register their work devices and report hygiene check statuses.
// Admins see an org-wide hygiene dashboard.

// Device types
const (
	DeviceTypeLaptop  = "laptop"
	DeviceTypeDesktop = "desktop"
	DeviceTypeMobile  = "mobile"
	DeviceTypeTablet  = "tablet"
	DeviceTypeOther   = "other"
)

// Hygiene check types
const (
	HygieneCheckOSUpdated       = "os_updated"
	HygieneCheckAntivirusActive = "antivirus_active"
	HygieneCheckDiskEncrypted   = "disk_encrypted"
	HygieneCheckScreenLock      = "screen_lock"
	HygieneCheckPasswordManager = "password_manager"
	HygieneCheckVPNEnabled      = "vpn_enabled"
	HygieneCheckMFAEnabled      = "mfa_enabled"
)

// Hygiene check statuses
const (
	HygieneStatusPass    = "pass"
	HygieneStatusFail    = "fail"
	HygieneStatusUnknown = "unknown"
)

var ErrDeviceNotFound = errors.New("device not found")

// Shared query fragments for cyber hygiene module.
const (
	hygieneQUserOrg  = "user_id = ? AND org_id = ?"
	hygieneQDeviceId = "device_id = ?"
)

// UserDevice represents a user's registered work device.
type UserDevice struct {
	Id           int64         `json:"id" gorm:"column:id; primary_key:yes"`
	UserId       int64         `json:"user_id" gorm:"column:user_id"`
	OrgId        int64         `json:"org_id" gorm:"column:org_id"`
	Name         string        `json:"name" gorm:"column:name"`
	DeviceType   string        `json:"device_type" gorm:"column:device_type"`
	OS           string        `json:"os" gorm:"column:os"`
	HygieneScore int           `json:"hygiene_score" gorm:"column:hygiene_score"`
	Checks       []DeviceCheck `json:"checks" gorm:"-"`
	CreatedDate  time.Time     `json:"created_date" gorm:"column:created_date"`
	ModifiedDate time.Time     `json:"modified_date" gorm:"column:modified_date"`
}

func (d *UserDevice) TableName() string { return "user_devices" }

// Validate checks required fields before saving a device.
func (d *UserDevice) Validate() error {
	if d.Name == "" {
		return errors.New("device name is required")
	}
	if d.DeviceType == "" {
		d.DeviceType = DeviceTypeOther
	}
	return nil
}

// DeviceCheck represents a single hygiene check result for a device.
type DeviceCheck struct {
	Id        int64     `json:"id" gorm:"column:id; primary_key:yes"`
	DeviceId  int64     `json:"device_id" gorm:"column:device_id"`
	CheckType string    `json:"check_type" gorm:"column:check_type"`
	Status    string    `json:"status" gorm:"column:status"`
	Note      string    `json:"note,omitempty" gorm:"column:note"`
	UpdatedAt time.Time `json:"updated_at" gorm:"column:updated_at"`
}

func (c *DeviceCheck) TableName() string { return "device_hygiene_checks" }

// HygieneSummary is a per-org aggregate returned for the admin dashboard.
type HygieneSummary struct {
	TotalDevices   int                         `json:"total_devices"`
	PassCount      int                         `json:"pass_count"`
	FailCount      int                         `json:"fail_count"`
	UnknownCount   int                         `json:"unknown_count"`
	AvgScore       float64                     `json:"avg_score"`
	CheckBreakdown map[string]HygieneCheckStat `json:"check_breakdown"`
}

// HygieneCheckStat summarises pass/fail/unknown for one check type.
type HygieneCheckStat struct {
	Pass    int `json:"pass"`
	Fail    int `json:"fail"`
	Unknown int `json:"unknown"`
}

// --- CRUD ---

// GetUserDevices returns all devices registered by a user.
func GetUserDevices(userId, orgId int64) ([]UserDevice, error) {
	devices := []UserDevice{}
	err := db.Where(hygieneQUserOrg, userId, orgId).
		Order("created_date desc").Find(&devices).Error
	if err != nil {
		return devices, err
	}
	for i := range devices {
		loadDeviceChecks(&devices[i])
	}
	return devices, nil
}

// GetOrgDevices returns all devices for an org (admin view).
func GetOrgDevices(orgId int64) ([]UserDevice, error) {
	devices := []UserDevice{}
	err := db.Where("org_id = ?", orgId).
		Order("user_id asc, created_date desc").Find(&devices).Error
	if err != nil {
		return devices, err
	}
	for i := range devices {
		loadDeviceChecks(&devices[i])
	}
	return devices, nil
}

// GetDevice returns a single device, enforcing org ownership.
func GetDevice(id, orgId int64) (UserDevice, error) {
	d := UserDevice{}
	err := db.Where("id = ? AND org_id = ?", id, orgId).First(&d).Error
	if err != nil {
		return d, ErrDeviceNotFound
	}
	loadDeviceChecks(&d)
	return d, nil
}

// PostDevice creates a new device record.
func PostDevice(d *UserDevice) error {
	d.HygieneScore = 0
	d.CreatedDate = time.Now().UTC()
	d.ModifiedDate = time.Now().UTC()
	return db.Save(d).Error
}

// PutDevice updates device metadata.
func PutDevice(d *UserDevice) error {
	d.ModifiedDate = time.Now().UTC()
	return db.Save(d).Error
}

// DeleteDevice removes a device and its checks (org-scoped).
func DeleteDevice(id, orgId int64) error {
	if err := db.Where(hygieneQDeviceId, id).Delete(&DeviceCheck{}).Error; err != nil {
		return err
	}
	return db.Where("id = ? AND org_id = ?", id, orgId).Delete(&UserDevice{}).Error
}

// UpsertDeviceCheck creates or updates a hygiene check for a device.
// It also recalculates and persists the device's hygiene score.
func UpsertDeviceCheck(deviceId, orgId int64, checkType, status, note string) error {
	// Verify device belongs to org
	if _, err := GetDevice(deviceId, orgId); err != nil {
		return ErrDeviceNotFound
	}

	existing := DeviceCheck{}
	err := db.Where("device_id = ? AND check_type = ?", deviceId, checkType).First(&existing).Error
	if err == nil {
		// Update
		existing.Status = status
		existing.Note = note
		existing.UpdatedAt = time.Now().UTC()
		if err := db.Save(&existing).Error; err != nil {
			return err
		}
	} else {
		// Create
		c := DeviceCheck{
			DeviceId:  deviceId,
			CheckType: checkType,
			Status:    status,
			Note:      note,
			UpdatedAt: time.Now().UTC(),
		}
		if err := db.Save(&c).Error; err != nil {
			return err
		}
	}

	return recalcHygieneScore(deviceId)
}

// GetOrgHygieneSummary returns aggregate hygiene statistics for an org.
func GetOrgHygieneSummary(orgId int64) (HygieneSummary, error) {
	devices, err := GetOrgDevices(orgId)
	if err != nil {
		return HygieneSummary{}, err
	}

	summary := HygieneSummary{
		TotalDevices:   len(devices),
		CheckBreakdown: make(map[string]HygieneCheckStat),
	}

	var totalScore int
	for _, d := range devices {
		totalScore += d.HygieneScore
		for _, c := range d.Checks {
			stat := summary.CheckBreakdown[c.CheckType]
			switch c.Status {
			case HygieneStatusPass:
				stat.Pass++
				summary.PassCount++
			case HygieneStatusFail:
				stat.Fail++
				summary.FailCount++
			default:
				stat.Unknown++
				summary.UnknownCount++
			}
			summary.CheckBreakdown[c.CheckType] = stat
		}
	}
	if len(devices) > 0 {
		summary.AvgScore = float64(totalScore) / float64(len(devices))
	}
	return summary, nil
}

// --- helpers ---

func loadDeviceChecks(d *UserDevice) {
	checks := []DeviceCheck{}
	db.Where(hygieneQDeviceId, d.Id).Find(&checks)
	d.Checks = checks
}

// recalcHygieneScore recomputes the device's hygiene score as the
// percentage of checks that pass (0–100). Written back to the DB.
func recalcHygieneScore(deviceId int64) error {
	checks := []DeviceCheck{}
	if err := db.Where(hygieneQDeviceId, deviceId).Find(&checks).Error; err != nil {
		return err
	}
	if len(checks) == 0 {
		return nil
	}
	pass := 0
	for _, c := range checks {
		if c.Status == HygieneStatusPass {
			pass++
		}
	}
	score := (pass * 100) / len(checks)
	return db.Model(&UserDevice{}).Where("id = ?", deviceId).
		Updates(map[string]interface{}{
			"hygiene_score": score,
			"modified_date": time.Now().UTC(),
		}).Error
}

// =====================================================================
// Personalized Tech-Stack Hygiene — tailored checks per employee device
// =====================================================================

// TechStackProfile represents a user's tech-stack inventory for personalized hygiene.
type TechStackProfile struct {
	Id           int64     `json:"id" gorm:"column:id; primary_key:yes"`
	UserId       int64     `json:"user_id" gorm:"column:user_id"`
	OrgId        int64     `json:"org_id" gorm:"column:org_id"`
	PrimaryOS    string    `json:"primary_os" gorm:"column:primary_os"`
	Browser      string    `json:"browser" gorm:"column:browser"`
	EmailClient  string    `json:"email_client" gorm:"column:email_client"`
	CloudApps    string    `json:"cloud_apps" gorm:"column:cloud_apps;type:text"` // JSON array
	DevTools     string    `json:"dev_tools" gorm:"column:dev_tools;type:text"`   // JSON array
	RemoteAccess string    `json:"remote_access" gorm:"column:remote_access"`     // VPN/RDP type
	MobileDevice string    `json:"mobile_device" gorm:"column:mobile_device"`
	CreatedDate  time.Time `json:"created_date" gorm:"column:created_date"`
	ModifiedDate time.Time `json:"modified_date" gorm:"column:modified_date"`
}

func (TechStackProfile) TableName() string { return "tech_stack_profiles" }

// PersonalizedCheck is a hygiene check recommendation based on tech stack.
type PersonalizedCheck struct {
	CheckType   string `json:"check_type"`
	Label       string `json:"label"`
	Description string `json:"description"`
	Relevant    bool   `json:"relevant"` // true if check is relevant to user's tech stack
	Reason      string `json:"reason"`   // why it's relevant
}

// GetTechStackProfile returns a user's tech stack profile.
func GetTechStackProfile(userId, orgId int64) (TechStackProfile, error) {
	p := TechStackProfile{}
	err := db.Where(hygieneQUserOrg, userId, orgId).First(&p).Error
	return p, err
}

// UpsertTechStackProfile creates or updates a user's tech stack profile.
func UpsertTechStackProfile(p *TechStackProfile) error {
	existing := TechStackProfile{}
	err := db.Where(hygieneQUserOrg, p.UserId, p.OrgId).First(&existing).Error
	if err == nil {
		p.Id = existing.Id
		p.CreatedDate = existing.CreatedDate
	} else {
		p.CreatedDate = time.Now().UTC()
	}
	p.ModifiedDate = time.Now().UTC()
	return db.Save(p).Error
}

// GetPersonalizedChecks returns recommended hygiene checks tailored to the user's tech stack.
func GetPersonalizedChecks(userId, orgId int64) []PersonalizedCheck {
	allChecks := []PersonalizedCheck{
		{CheckType: HygieneCheckOSUpdated, Label: "OS Updated", Description: "Operating system is on the latest version with all security patches applied."},
		{CheckType: HygieneCheckAntivirusActive, Label: "Antivirus Active", Description: "Endpoint protection / antivirus software is running and up to date."},
		{CheckType: HygieneCheckDiskEncrypted, Label: "Disk Encrypted", Description: "Full-disk encryption (BitLocker, FileVault, LUKS) is enabled."},
		{CheckType: HygieneCheckScreenLock, Label: "Screen Lock", Description: "Automatic screen lock is enabled (≤5 minutes timeout)."},
		{CheckType: HygieneCheckPasswordManager, Label: "Password Manager", Description: "A corporate password manager is installed and in active use."},
		{CheckType: HygieneCheckVPNEnabled, Label: "VPN Enabled", Description: "VPN is configured and used when connecting to untrusted networks."},
		{CheckType: HygieneCheckMFAEnabled, Label: "MFA Enabled", Description: "Multi-factor authentication is enabled on all corporate accounts."},
	}

	profile, err := GetTechStackProfile(userId, orgId)
	if err != nil {
		// No tech stack profile — all checks are relevant
		for i := range allChecks {
			allChecks[i].Relevant = true
			allChecks[i].Reason = "Default — no tech stack profile configured"
		}
		return allChecks
	}

	for i := range allChecks {
		allChecks[i].Relevant = true
		allChecks[i].Reason = personalizeCheckReason(allChecks[i].CheckType, profile)
	}
	return allChecks
}

// personalizeCheckReason generates a context-aware reason based on the user's tech stack.
func personalizeCheckReason(checkType string, p TechStackProfile) string {
	switch checkType {
	case HygieneCheckOSUpdated:
		if p.PrimaryOS != "" {
			return "Your " + p.PrimaryOS + " should be on the latest version with security patches."
		}
		return "Keep your operating system updated."
	case HygieneCheckAntivirusActive:
		if p.PrimaryOS == "macOS" {
			return "Even on macOS, ensure Malwarebytes or built-in XProtect is active."
		}
		if p.PrimaryOS == "Windows" {
			return "Windows Defender or your corporate AV should be running."
		}
		return "Ensure endpoint protection is active on your device."
	case HygieneCheckDiskEncrypted:
		if p.PrimaryOS == "macOS" {
			return "Enable FileVault in System Preferences → Security."
		}
		if p.PrimaryOS == "Windows" {
			return "Enable BitLocker in Control Panel → System and Security."
		}
		return "Enable full-disk encryption on your device."
	case HygieneCheckScreenLock:
		return "Set auto-lock timeout to 5 minutes or less."
	case HygieneCheckPasswordManager:
		return "Use your organization's approved password manager for all credentials."
	case HygieneCheckVPNEnabled:
		if p.RemoteAccess != "" {
			return "Use " + p.RemoteAccess + " when connecting from outside the office."
		}
		return "Always use VPN on untrusted networks."
	case HygieneCheckMFAEnabled:
		if p.EmailClient != "" {
			return "Ensure MFA is active on " + p.EmailClient + " and all corporate apps."
		}
		return "Enable MFA on all corporate accounts."
	}
	return ""
}

// HygieneAdminDeviceView is an enriched device record for the admin dashboard.
type HygieneAdminDeviceView struct {
	UserDevice
	UserName   string `json:"user_name"`
	UserEmail  string `json:"user_email"`
	Department string `json:"department"`
}

// GetOrgDevicesEnriched returns all devices with user information for the admin dashboard.
func GetOrgDevicesEnriched(orgId int64) ([]HygieneAdminDeviceView, error) {
	devices, err := GetOrgDevices(orgId)
	if err != nil {
		return nil, err
	}
	views := make([]HygieneAdminDeviceView, len(devices))
	for i, d := range devices {
		views[i] = HygieneAdminDeviceView{UserDevice: d}
		if d.UserId > 0 {
			u, uErr := GetUser(d.UserId)
			if uErr == nil {
				views[i].UserName = u.FirstName + " " + u.LastName
				views[i].UserEmail = u.Email
				views[i].Department = u.Department
			}
		}
	}
	return views, nil
}

// HygieneEnrichedSummary adds tech-stack breakdown and risk analysis.
type HygieneEnrichedSummary struct {
	HygieneSummary
	OSBreakdown         map[string]int `json:"os_breakdown"`
	DeviceTypeBreakdown map[string]int `json:"device_type_breakdown"`
	AtRiskDevices       int            `json:"at_risk_devices"`
	FullyCompliant      int            `json:"fully_compliant"`
	ProfileCount        int            `json:"profile_count"`
}

// GetOrgHygieneEnrichedSummary returns the comprehensive org hygiene dashboard data.
func GetOrgHygieneEnrichedSummary(orgId int64) (HygieneEnrichedSummary, error) {
	base, err := GetOrgHygieneSummary(orgId)
	if err != nil {
		return HygieneEnrichedSummary{}, err
	}

	enriched := HygieneEnrichedSummary{
		HygieneSummary:      base,
		OSBreakdown:         make(map[string]int),
		DeviceTypeBreakdown: make(map[string]int),
	}

	devices, _ := GetOrgDevices(orgId)
	for _, d := range devices {
		os := d.OS
		if os == "" {
			os = "Unknown"
		}
		enriched.OSBreakdown[os]++
		dt := d.DeviceType
		if dt == "" {
			dt = DeviceTypeOther
		}
		enriched.DeviceTypeBreakdown[dt]++

		if d.HygieneScore == 100 {
			enriched.FullyCompliant++
		}
		if d.HygieneScore < 50 {
			enriched.AtRiskDevices++
		}
	}

	// Count tech stack profiles
	db.Table("tech_stack_profiles").Where("org_id = ?", orgId).Count(&enriched.ProfileCount)

	return enriched, nil
}
