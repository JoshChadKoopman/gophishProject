package models

import (
	"time"

	log "github.com/gophish/gophish/logger"
)

// queryWhereID is the shared WHERE clause fragment for primary key lookups.
const queryWhereID = "id = ?"

// ComplianceCertification defines a compliance certification path (e.g., GDPR, NIS2).
type ComplianceCertification struct {
	Id                 int64     `json:"id" gorm:"column:id; primary_key:yes"`
	OrgId              int64     `json:"org_id" gorm:"column:org_id"`
	Slug               string    `json:"slug" gorm:"column:slug"`
	Name               string    `json:"name" gorm:"column:name"`
	Description        string    `json:"description" gorm:"column:description"`
	RequiredSessionIDs string    `json:"required_session_ids" gorm:"column:required_session_ids;type:text"`
	IsActive           bool      `json:"is_active" gorm:"column:is_active"`
	CreatedDate        time.Time `json:"created_date" gorm:"column:created_date"`

	// Populated at query time
	TotalRequired int  `json:"total_required" gorm:"-"`
	UserCompleted int  `json:"user_completed" gorm:"-"`
	Earned        bool `json:"earned" gorm:"-"`
}

// UserComplianceCert records a user's earned compliance certification.
type UserComplianceCert struct {
	Id               int64     `json:"id" gorm:"column:id; primary_key:yes"`
	UserId           int64     `json:"user_id" gorm:"column:user_id"`
	CertificationId  int64     `json:"certification_id" gorm:"column:certification_id"`
	VerificationCode string    `json:"verification_code" gorm:"column:verification_code"`
	IssuedDate       time.Time `json:"issued_date" gorm:"column:issued_date"`
	ExpiresDate      time.Time `json:"expires_date" gorm:"column:expires_date"`

	// Populated at query time
	CertificationName string `json:"certification_name,omitempty" gorm:"-"`
	CertificationSlug string `json:"certification_slug,omitempty" gorm:"-"`
}

// GetComplianceCertifications returns all active certifications for an org.
func GetComplianceCertifications(orgId int64) ([]ComplianceCertification, error) {
	certs := []ComplianceCertification{}
	err := db.Where("(org_id = ? OR org_id = 0) AND is_active = 1", orgId).
		Order("name asc").Find(&certs).Error
	return certs, err
}

// GetComplianceCertificationsWithProgress returns certifications with user completion status.
func GetComplianceCertificationsWithProgress(orgId, userId int64) ([]ComplianceCertification, error) {
	certs, err := GetComplianceCertifications(orgId)
	if err != nil {
		return certs, err
	}
	for i := range certs {
		sessionIDs := ParseSessionIDs(certs[i].RequiredSessionIDs)
		certs[i].TotalRequired = len(sessionIDs)
		completed := 0
		for _, sid := range sessionIDs {
			// Check if the session's presentation has been completed
			session := AcademySession{}
			if err := db.Where(queryWhereID, sid).First(&session).Error; err != nil {
				continue
			}
			cp := CourseProgress{}
			if err := db.Where("user_id = ? AND presentation_id = ? AND status = 'complete'", userId, session.PresentationId).First(&cp).Error; err == nil {
				completed++
			}
		}
		certs[i].UserCompleted = completed

		// Check if user has earned this cert
		uc := UserComplianceCert{}
		if err := db.Where("user_id = ? AND certification_id = ?", userId, certs[i].Id).First(&uc).Error; err == nil {
			certs[i].Earned = true
		}
	}
	return certs, err
}

// GetComplianceCertification returns a single certification by ID.
func GetComplianceCertification(id int64) (ComplianceCertification, error) {
	cert := ComplianceCertification{}
	err := db.Where(queryWhereID, id).First(&cert).Error
	return cert, err
}

// CreateComplianceCertification creates a new compliance certification.
func CreateComplianceCertification(c *ComplianceCertification) error {
	c.CreatedDate = time.Now().UTC()
	if c.RequiredSessionIDs == "" {
		c.RequiredSessionIDs = "[]"
	}
	return db.Save(c).Error
}

// UpdateComplianceCertification updates a certification.
func UpdateComplianceCertification(c *ComplianceCertification) error {
	return db.Table("compliance_certifications").Where(queryWhereID, c.Id).Updates(map[string]interface{}{
		"name":                 c.Name,
		"description":          c.Description,
		"required_session_ids": c.RequiredSessionIDs,
		"is_active":            c.IsActive,
	}).Error
}

// IssueComplianceCert creates a user compliance certificate with a unique verification code.
func IssueComplianceCert(userId, certificationId int64) (*UserComplianceCert, error) {
	// Check if already earned
	existing := UserComplianceCert{}
	if err := db.Where("user_id = ? AND certification_id = ?", userId, certificationId).First(&existing).Error; err == nil {
		return &existing, nil // Already earned
	}

	code, err := generateVerificationCode()
	if err != nil {
		log.Error(err)
		return nil, err
	}
	uc := UserComplianceCert{
		UserId:           userId,
		CertificationId:  certificationId,
		VerificationCode: code,
		IssuedDate:       time.Now().UTC(),
		ExpiresDate:      time.Now().UTC().AddDate(1, 0, 0), // Expires in 1 year
	}
	if err := db.Save(&uc).Error; err != nil {
		log.Error(err)
		return nil, err
	}
	return &uc, nil
}

// CheckAndIssueComplianceCert checks if a user has completed all required sessions for a cert
// and issues it if so.
func CheckAndIssueComplianceCert(userId int64, cert ComplianceCertification) (*UserComplianceCert, bool) {
	sessionIDs := ParseSessionIDs(cert.RequiredSessionIDs)
	if len(sessionIDs) == 0 {
		return nil, false
	}
	for _, sid := range sessionIDs {
		session := AcademySession{}
		if err := db.Where(queryWhereID, sid).First(&session).Error; err != nil {
			return nil, false
		}
		cp := CourseProgress{}
		if err := db.Where("user_id = ? AND presentation_id = ? AND status = 'complete'", userId, session.PresentationId).First(&cp).Error; err != nil {
			return nil, false
		}
	}
	// All sessions completed — issue cert
	uc, err := IssueComplianceCert(userId, cert.Id)
	if err != nil {
		return nil, false
	}
	return uc, true
}

// GetUserComplianceCerts returns all compliance certs earned by a user.
func GetUserComplianceCerts(userId int64) ([]UserComplianceCert, error) {
	certs := []UserComplianceCert{}
	err := db.Where("user_id = ?", userId).Order("issued_date desc").Find(&certs).Error
	if err != nil {
		return certs, err
	}
	for i := range certs {
		cc := ComplianceCertification{}
		if err := db.Where(queryWhereID, certs[i].CertificationId).First(&cc).Error; err == nil {
			certs[i].CertificationName = cc.Name
			certs[i].CertificationSlug = cc.Slug
		}
	}
	return certs, nil
}

// VerifyComplianceCert looks up a compliance cert by verification code.
func VerifyComplianceCert(code string) (*UserComplianceCert, error) {
	uc := UserComplianceCert{}
	err := db.Where("verification_code = ?", code).First(&uc).Error
	if err != nil {
		return nil, err
	}
	cc := ComplianceCertification{}
	if err := db.Where(queryWhereID, uc.CertificationId).First(&cc).Error; err == nil {
		uc.CertificationName = cc.Name
		uc.CertificationSlug = cc.Slug
	}
	return &uc, nil
}

// GetComplianceCertCount returns the number of compliance certs earned by a user.
func GetComplianceCertCount(userId int64) int {
	var count int
	db.Table("user_compliance_certs").Where("user_id = ?", userId).Count(&count)
	return count
}
