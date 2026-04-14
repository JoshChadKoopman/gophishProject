package models

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"time"

	log "github.com/gophish/gophish/logger"
)

// Certificate represents a training course completion certificate.
type Certificate struct {
	Id               int64     `json:"id" gorm:"column:id; primary_key:yes"`
	UserId           int64     `json:"user_id" gorm:"column:user_id"`
	PresentationId   int64     `json:"presentation_id" gorm:"column:presentation_id"`
	QuizAttemptId    int64     `json:"quiz_attempt_id,omitempty" gorm:"column:quiz_attempt_id"`
	TemplateSlug     string    `json:"template_slug" gorm:"column:template_slug"`
	VerificationCode string    `json:"verification_code" gorm:"column:verification_code"`
	IssuedDate       time.Time `json:"issued_date" gorm:"column:issued_date"`
	ExpiresDate      time.Time `json:"expires_date,omitempty" gorm:"column:expires_date"`
	RevokedDate      time.Time `json:"revoked_date,omitempty" gorm:"column:revoked_date"`
	IsRevoked        bool      `json:"is_revoked" gorm:"column:is_revoked"`
	Metadata         string    `json:"metadata,omitempty" gorm:"column:metadata;type:text"`
}

const verificationCodeLen = 16
const verificationCodeChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const orderIssuedDateDesc = "issued_date desc"

// ---- Specialized Certificate Template Types ----

// CertificateTemplate defines a specialized certificate template.
type CertificateTemplate struct {
	Slug           string `json:"slug"`
	Name           string `json:"name"`
	Description    string `json:"description"`
	Category       string `json:"category"`
	ValidityMonths int    `json:"validity_months"` // 0 = never expires
	BadgeIconURL   string `json:"badge_icon_url"`
	ColorScheme    string `json:"color_scheme"` // hex color for template header
}

// SpecializedCertTemplates contains the 15+ built-in specialized certificate templates.
var SpecializedCertTemplates = []CertificateTemplate{
	// ---- Cybersecurity Awareness ----
	{
		Slug: "cybersecurity-awareness-foundation",
		Name: "Cybersecurity Awareness Foundation", Description: "Foundational cybersecurity awareness training completion",
		Category: "cybersecurity", ValidityMonths: 12, BadgeIconURL: "/images/certs/cyber-foundation.png", ColorScheme: "#2c3e50",
	},
	{
		Slug: "phishing-defense-specialist",
		Name: "Phishing Defense Specialist", Description: "Advanced phishing identification and response training",
		Category: "cybersecurity", ValidityMonths: 12, BadgeIconURL: "/images/certs/phishing-specialist.png", ColorScheme: "#e74c3c",
	},
	{
		Slug: "social-engineering-defender",
		Name: "Social Engineering Defender", Description: "Social engineering attack recognition and prevention",
		Category: "cybersecurity", ValidityMonths: 12, BadgeIconURL: "/images/certs/social-eng.png", ColorScheme: "#8e44ad",
	},
	{
		Slug: "data-protection-champion",
		Name: "Data Protection Champion", Description: "Data handling, classification, and protection best practices",
		Category: "data_protection", ValidityMonths: 12, BadgeIconURL: "/images/certs/data-protection.png", ColorScheme: "#27ae60",
	},
	{
		Slug: "password-security-expert",
		Name: "Password & Authentication Security Expert", Description: "Password management, MFA, and authentication security",
		Category: "cybersecurity", ValidityMonths: 24, BadgeIconURL: "/images/certs/password-security.png", ColorScheme: "#2980b9",
	},
	// ---- Compliance ----
	{
		Slug: "gdpr-awareness",
		Name: "GDPR Awareness Certificate", Description: "General Data Protection Regulation compliance training",
		Category: "compliance", ValidityMonths: 12, BadgeIconURL: "/images/certs/gdpr.png", ColorScheme: "#003399",
	},
	{
		Slug: "nis2-compliance",
		Name: "NIS2 Directive Compliance", Description: "Network and Information Security Directive 2 training",
		Category: "compliance", ValidityMonths: 12, BadgeIconURL: "/images/certs/nis2.png", ColorScheme: "#1a5276",
	},
	{
		Slug: "hipaa-security-awareness",
		Name: "HIPAA Security Awareness", Description: "Health data privacy and security awareness training",
		Category: "compliance", ValidityMonths: 12, BadgeIconURL: "/images/certs/hipaa.png", ColorScheme: "#148f77",
	},
	{
		Slug: "pci-dss-awareness",
		Name: "PCI DSS Awareness", Description: "Payment Card Industry Data Security Standard training",
		Category: "compliance", ValidityMonths: 12, BadgeIconURL: "/images/certs/pci-dss.png", ColorScheme: "#d4ac0d",
	},
	{
		Slug: "iso27001-awareness",
		Name: "ISO 27001 Awareness", Description: "Information Security Management System awareness training",
		Category: "compliance", ValidityMonths: 12, BadgeIconURL: "/images/certs/iso27001.png", ColorScheme: "#154360",
	},
	{
		Slug: "dora-compliance",
		Name: "DORA Compliance", Description: "Digital Operational Resilience Act compliance training",
		Category: "compliance", ValidityMonths: 12, BadgeIconURL: "/images/certs/dora.png", ColorScheme: "#1b4f72",
	},
	// ---- Specialized Technical ----
	{
		Slug: "incident-response-certified",
		Name: "Incident Response Certified", Description: "Incident detection, reporting, and response procedures",
		Category: "technical", ValidityMonths: 12, BadgeIconURL: "/images/certs/incident-response.png", ColorScheme: "#c0392b",
	},
	{
		Slug: "secure-remote-work",
		Name: "Secure Remote Work Certificate", Description: "VPN, Wi-Fi security, and remote work best practices",
		Category: "technical", ValidityMonths: 12, BadgeIconURL: "/images/certs/remote-work.png", ColorScheme: "#16a085",
	},
	{
		Slug: "cloud-security-awareness",
		Name: "Cloud Security Awareness", Description: "Cloud service security, shared responsibility model training",
		Category: "technical", ValidityMonths: 12, BadgeIconURL: "/images/certs/cloud-security.png", ColorScheme: "#2e86c1",
	},
	{
		Slug: "mobile-device-security",
		Name: "Mobile Device Security", Description: "Mobile device management and BYOD security training",
		Category: "technical", ValidityMonths: 24, BadgeIconURL: "/images/certs/mobile-security.png", ColorScheme: "#17a589",
	},
	{
		Slug: "ai-security-awareness",
		Name: "AI & Machine Learning Security", Description: "AI-powered threats, deepfakes, and AI security awareness",
		Category: "technical", ValidityMonths: 12, BadgeIconURL: "/images/certs/ai-security.png", ColorScheme: "#6c3483",
	},
	// ---- Leadership / Role-based ----
	{
		Slug: "security-champion",
		Name: "Security Champion", Description: "Organizational security leadership and advocacy certification",
		Category: "leadership", ValidityMonths: 24, BadgeIconURL: "/images/certs/security-champion.png", ColorScheme: "#d4ac0d",
	},
	{
		Slug: "executive-cyber-awareness",
		Name: "Executive Cyber Awareness", Description: "Board-level cybersecurity risk and governance awareness",
		Category: "leadership", ValidityMonths: 12, BadgeIconURL: "/images/certs/executive.png", ColorScheme: "#1c2833",
	},
}

// certTemplateMap is a lookup of slug → CertificateTemplate for O(1) access.
var certTemplateMap map[string]CertificateTemplate

func init() {
	certTemplateMap = make(map[string]CertificateTemplate, len(SpecializedCertTemplates))
	for _, t := range SpecializedCertTemplates {
		certTemplateMap[t.Slug] = t
	}
}

// GetCertificateTemplate returns the template definition for a slug, or nil if not found.
func GetCertificateTemplate(slug string) *CertificateTemplate {
	t, ok := certTemplateMap[slug]
	if !ok {
		return nil
	}
	return &t
}

// GetCertificateTemplates returns all available specialized certificate templates.
func GetCertificateTemplates() []CertificateTemplate {
	return SpecializedCertTemplates
}

// GetCertificateTemplatesByCategory returns templates filtered by category.
func GetCertificateTemplatesByCategory(category string) []CertificateTemplate {
	var result []CertificateTemplate
	for _, t := range SpecializedCertTemplates {
		if t.Category == category {
			result = append(result, t)
		}
	}
	return result
}

// GetCertificateTemplateCategories returns all unique template categories.
func GetCertificateTemplateCategories() []string {
	seen := make(map[string]bool)
	var cats []string
	for _, t := range SpecializedCertTemplates {
		if !seen[t.Category] {
			seen[t.Category] = true
			cats = append(cats, t.Category)
		}
	}
	return cats
}

// generateVerificationCode produces a 16-character alphanumeric string using crypto/rand.
func generateVerificationCode() (string, error) {
	result := make([]byte, verificationCodeLen)
	for i := range result {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(verificationCodeChars))))
		if err != nil {
			return "", err
		}
		result[i] = verificationCodeChars[idx.Int64()]
	}
	return string(result), nil
}

// GenerateVerificationCodeFormatted produces a formatted code like "CERT-XXXX-XXXX-XXXX-XXXX" for display.
func GenerateVerificationCodeFormatted() (string, string, error) {
	raw, err := generateVerificationCode()
	if err != nil {
		return "", "", err
	}
	formatted := fmt.Sprintf("CERT-%s-%s-%s-%s", raw[0:4], raw[4:8], raw[8:12], raw[12:16])
	return raw, formatted, nil
}

// IssueCertificate creates a new certificate for a user's course completion.
// quizAttemptId should be 0 if the course has no quiz.
func IssueCertificate(userId, presentationId, quizAttemptId int64) (*Certificate, error) {
	return IssueCertificateWithTemplate(userId, presentationId, quizAttemptId, "cybersecurity-awareness-foundation")
}

// IssueCertificateWithTemplate creates a certificate using a specialized template.
func IssueCertificateWithTemplate(userId, presentationId, quizAttemptId int64, templateSlug string) (*Certificate, error) {
	code, err := generateVerificationCode()
	if err != nil {
		return nil, err
	}

	cert := &Certificate{
		UserId:           userId,
		PresentationId:   presentationId,
		QuizAttemptId:    quizAttemptId,
		TemplateSlug:     templateSlug,
		VerificationCode: code,
		IssuedDate:       time.Now().UTC(),
	}

	// Set expiry based on template validity
	tmpl := GetCertificateTemplate(templateSlug)
	if tmpl != nil && tmpl.ValidityMonths > 0 {
		cert.ExpiresDate = time.Now().UTC().AddDate(0, tmpl.ValidityMonths, 0)
	}

	err = db.Save(cert).Error
	if err != nil {
		log.Error(err)
		return nil, err
	}
	return cert, nil
}

// GetCertificate looks up a certificate by its verification code.
func GetCertificate(verificationCode string) (Certificate, error) {
	c := Certificate{}
	// Strip formatting if present (e.g. "CERT-XXXX-XXXX-XXXX-XXXX" → raw code)
	clean := strings.ReplaceAll(verificationCode, "CERT-", "")
	clean = strings.ReplaceAll(clean, "-", "")
	if len(clean) == verificationCodeLen {
		verificationCode = clean
	}
	err := db.Where("verification_code=?", verificationCode).First(&c).Error
	return c, err
}

// GetCertificatesForUser returns all certificates for a given user.
func GetCertificatesForUser(userId int64) ([]Certificate, error) {
	certs := []Certificate{}
	err := db.Where("user_id=?", userId).Order(orderIssuedDateDesc).Find(&certs).Error
	return certs, err
}

// GetActiveCertificatesForUser returns non-revoked, non-expired certificates for a user.
func GetActiveCertificatesForUser(userId int64) ([]Certificate, error) {
	certs := []Certificate{}
	now := time.Now().UTC()
	err := db.Where(
		"user_id=? AND is_revoked=0 AND (expires_date IS NULL OR expires_date=? OR expires_date > ?)",
		userId, time.Time{}, now,
	).Order(orderIssuedDateDesc).Find(&certs).Error
	return certs, err
}

// GetCertificateForCourse returns the most recent certificate for a user on a specific course.
func GetCertificateForCourse(userId, presentationId int64) (Certificate, error) {
	c := Certificate{}
	err := db.Where("user_id=? AND presentation_id=?", userId, presentationId).
		Order("issued_date desc").First(&c).Error
	return c, err
}

// GetCertificatesByTemplate returns all certificates issued with a specific template.
func GetCertificatesByTemplate(templateSlug string) ([]Certificate, error) {
	certs := []Certificate{}
	err := db.Where("template_slug=?", templateSlug).Order(orderIssuedDateDesc).Find(&certs).Error
	return certs, err
}

// GetCertificateCount returns the total number of certificates for a user.
func GetCertificateCount(userId int64) int {
	var count int
	db.Table("certificates").Where("user_id=?", userId).Count(&count)
	return count
}

// GetActiveCertificateCount returns the number of non-revoked, non-expired certs for a user.
func GetActiveCertificateCount(userId int64) int {
	var count int
	now := time.Now().UTC()
	db.Table("certificates").Where(
		"user_id=? AND is_revoked=0 AND (expires_date IS NULL OR expires_date=? OR expires_date > ?)",
		userId, time.Time{}, now,
	).Count(&count)
	return count
}

// RevokeCertificate revokes a certificate by ID.
func RevokeCertificate(id int64) error {
	return db.Model(&Certificate{}).Where("id=?", id).Updates(map[string]interface{}{
		"is_revoked":   true,
		"revoked_date": time.Now().UTC(),
	}).Error
}

// IsCertificateValid checks if a certificate is valid (not revoked and not expired).
func IsCertificateValid(cert Certificate) bool {
	if cert.IsRevoked {
		return false
	}
	if !cert.ExpiresDate.IsZero() && cert.ExpiresDate.Before(time.Now().UTC()) {
		return false
	}
	return true
}

// RenewCertificate issues a new certificate based on an existing one, extending the validity.
func RenewCertificate(existingCertId int64) (*Certificate, error) {
	existing := Certificate{}
	if err := db.Where("id=?", existingCertId).First(&existing).Error; err != nil {
		return nil, err
	}
	// Issue a new cert with the same template
	return IssueCertificateWithTemplate(
		existing.UserId,
		existing.PresentationId,
		existing.QuizAttemptId,
		existing.TemplateSlug,
	)
}

// GetExpiringCertificates returns certificates expiring within the given number of days.
func GetExpiringCertificates(daysUntilExpiry int) ([]Certificate, error) {
	certs := []Certificate{}
	now := time.Now().UTC()
	threshold := now.AddDate(0, 0, daysUntilExpiry)
	err := db.Where(
		"is_revoked=0 AND expires_date > ? AND expires_date <= ? AND expires_date != ?",
		now, threshold, time.Time{},
	).Order("expires_date asc").Find(&certs).Error
	return certs, err
}

// CertificateSummary provides aggregate statistics about certificates.
type CertificateSummary struct {
	TotalIssued  int            `json:"total_issued"`
	ActiveCerts  int            `json:"active_certs"`
	RevokedCerts int            `json:"revoked_certs"`
	ExpiredCerts int            `json:"expired_certs"`
	ExpiringSoon int            `json:"expiring_soon"` // within 30 days
	ByTemplate   map[string]int `json:"by_template"`
}

// GetCertificateSummary returns aggregate certificate statistics.
func GetCertificateSummary() CertificateSummary {
	summary := CertificateSummary{ByTemplate: make(map[string]int)}
	now := time.Now().UTC()
	soon := now.AddDate(0, 0, 30)

	db.Table("certificates").Count(&summary.TotalIssued)
	db.Table("certificates").Where(
		"is_revoked=0 AND (expires_date IS NULL OR expires_date=? OR expires_date > ?)",
		time.Time{}, now,
	).Count(&summary.ActiveCerts)
	db.Table("certificates").Where("is_revoked=1").Count(&summary.RevokedCerts)
	db.Table("certificates").Where(
		"is_revoked=0 AND expires_date != ? AND expires_date <= ?",
		time.Time{}, now,
	).Count(&summary.ExpiredCerts)
	db.Table("certificates").Where(
		"is_revoked=0 AND expires_date > ? AND expires_date <= ?",
		now, soon,
	).Count(&summary.ExpiringSoon)

	// Count by template
	type templateCount struct {
		TemplateSlug string
		Cnt          int
	}
	var counts []templateCount
	db.Table("certificates").
		Select("template_slug, count(*) as cnt").
		Group("template_slug").
		Scan(&counts)
	for _, tc := range counts {
		slug := tc.TemplateSlug
		if slug == "" {
			slug = "default"
		}
		summary.ByTemplate[slug] = tc.Cnt
	}

	return summary
}

// EnrichedCertificate provides certificate data enriched with presentation and template info.
type EnrichedCertificate struct {
	Certificate
	CourseName    string               `json:"course_name"`
	UserName      string               `json:"user_name"`
	Template      *CertificateTemplate `json:"template,omitempty"`
	IsValid       bool                 `json:"is_valid"`
	FormattedCode string               `json:"formatted_code"`
}

// EnrichCertificate populates extra fields on a certificate for API responses.
func EnrichCertificate(cert Certificate) EnrichedCertificate {
	ec := EnrichedCertificate{
		Certificate: cert,
		IsValid:     IsCertificateValid(cert),
	}
	if len(cert.VerificationCode) == verificationCodeLen {
		ec.FormattedCode = fmt.Sprintf("CERT-%s-%s-%s-%s",
			cert.VerificationCode[0:4], cert.VerificationCode[4:8],
			cert.VerificationCode[8:12], cert.VerificationCode[12:16])
	}
	tmpl := GetCertificateTemplate(cert.TemplateSlug)
	if tmpl != nil {
		ec.Template = tmpl
	}
	return ec
}
