package models

import (
	"time"

	log "github.com/gophish/gophish/logger"
)

// ── Framework-Specific Compliance Certificates ──
// Pre-built compliance certificates that are automatically issued when an
// organization meets all the controls for a given framework. These are
// distinct from the general training certificates — they attest to
// framework-level compliance readiness.

// FrameworkComplianceCert is a pre-built compliance certificate definition
// that maps directly to a regulatory framework.
type FrameworkComplianceCert struct {
	Slug              string  `json:"slug"`
	FrameworkSlug     string  `json:"framework_slug"`
	Name              string  `json:"name"`
	Description       string  `json:"description"`
	IssuingAuthority  string  `json:"issuing_authority"`
	ValidityMonths    int     `json:"validity_months"`
	MinOverallScore   float64 `json:"min_overall_score"`   // Min framework score to earn
	MinControlsPassed float64 `json:"min_controls_passed"` // Fraction of controls that must be compliant (0-1)
	BadgeIconURL      string  `json:"badge_icon_url"`
	ColorScheme       string  `json:"color_scheme"`
	CertificateHTML   string  `json:"certificate_html"` // HTML template for PDF generation
}

const certIssuer = "Nivoxis Security Platform"

// certColorNavyBlue is the shared dark blue colour used by several cert badges.
const certColorNavyBlue = "#1a5276"

// BuiltInFrameworkCerts contains the 15 pre-built framework-specific compliance certificates.
var BuiltInFrameworkCerts = []FrameworkComplianceCert{
	// ── NIS2 ──
	{
		Slug: "nis2-compliance-readiness", FrameworkSlug: "nis2",
		Name: "NIS2 Compliance Readiness", Description: "Demonstrates organizational readiness for EU NIS2 Directive requirements including cybersecurity risk management, incident reporting, and supply chain security.",
		IssuingAuthority: certIssuer, ValidityMonths: 12, MinOverallScore: 70, MinControlsPassed: 0.7,
		BadgeIconURL: "/images/certs/nis2-compliance.png", ColorScheme: "#003399",
	},
	{
		Slug: "nis2-awareness-training", FrameworkSlug: "nis2",
		Name: "NIS2 Awareness Training Certificate", Description: "Completion of all NIS2-specific cybersecurity awareness training modules covering Art. 21 requirements.",
		IssuingAuthority: certIssuer, ValidityMonths: 12, MinOverallScore: 60, MinControlsPassed: 0.5,
		BadgeIconURL: "/images/certs/nis2-training.png", ColorScheme: certColorNavyBlue,
	},

	// ── DORA ──
	{
		Slug: "dora-compliance-readiness", FrameworkSlug: "dora",
		Name: "DORA Compliance Readiness", Description: "Demonstrates readiness for EU Digital Operational Resilience Act requirements for financial entities.",
		IssuingAuthority: certIssuer, ValidityMonths: 12, MinOverallScore: 75, MinControlsPassed: 0.7,
		BadgeIconURL: "/images/certs/dora-compliance.png", ColorScheme: "#1b4f72",
	},
	{
		Slug: "dora-ict-resilience", FrameworkSlug: "dora",
		Name: "DORA ICT Resilience Training", Description: "ICT security awareness and digital operational resilience training as required by DORA Art. 13(6).",
		IssuingAuthority: certIssuer, ValidityMonths: 12, MinOverallScore: 60, MinControlsPassed: 0.5,
		BadgeIconURL: "/images/certs/dora-resilience.png", ColorScheme: "#2c3e50",
	},

	// ── HIPAA ──
	{
		Slug: "hipaa-compliance-readiness", FrameworkSlug: "hipaa",
		Name: "HIPAA Compliance Readiness", Description: "Demonstrates compliance with HIPAA Security Rule administrative, technical, and physical safeguards.",
		IssuingAuthority: certIssuer, ValidityMonths: 12, MinOverallScore: 80, MinControlsPassed: 0.8,
		BadgeIconURL: "/images/certs/hipaa-compliance.png", ColorScheme: "#148f77",
	},
	{
		Slug: "hipaa-privacy-security-training", FrameworkSlug: "hipaa",
		Name: "HIPAA Privacy & Security Training", Description: "Completion of HIPAA-specific security awareness training covering ePHI protection, incident procedures, and access controls.",
		IssuingAuthority: certIssuer, ValidityMonths: 12, MinOverallScore: 60, MinControlsPassed: 0.5,
		BadgeIconURL: "/images/certs/hipaa-training.png", ColorScheme: "#117a65",
	},

	// ── PCI DSS ──
	{
		Slug: "pci-dss-compliance-readiness", FrameworkSlug: "pci_dss",
		Name: "PCI DSS Compliance Readiness", Description: "Demonstrates compliance readiness with PCI DSS v4.0 security awareness, anti-phishing, and authentication requirements.",
		IssuingAuthority: certIssuer, ValidityMonths: 12, MinOverallScore: 80, MinControlsPassed: 0.8,
		BadgeIconURL: "/images/certs/pci-compliance.png", ColorScheme: "#d4ac0d",
	},
	{
		Slug: "pci-dss-security-awareness", FrameworkSlug: "pci_dss",
		Name: "PCI DSS Security Awareness", Description: "Annual PCI DSS security awareness training completion per Requirement 12.6.",
		IssuingAuthority: certIssuer, ValidityMonths: 12, MinOverallScore: 60, MinControlsPassed: 0.5,
		BadgeIconURL: "/images/certs/pci-training.png", ColorScheme: "#b7950b",
	},

	// ── NIST CSF ──
	{
		Slug: "nist-csf-compliance-readiness", FrameworkSlug: "nist_csf",
		Name: "NIST CSF Compliance Readiness", Description: "Demonstrates alignment with NIST Cybersecurity Framework 2.0 across Govern, Identify, Protect, Detect, Respond, and Recover functions.",
		IssuingAuthority: certIssuer, ValidityMonths: 12, MinOverallScore: 70, MinControlsPassed: 0.7,
		BadgeIconURL: "/images/certs/nist-compliance.png", ColorScheme: "#154360",
	},
	{
		Slug: "nist-csf-awareness-training", FrameworkSlug: "nist_csf",
		Name: "NIST CSF Awareness Training", Description: "Completion of NIST CSF-aligned cybersecurity awareness and role-based training (GV.AT-01, GV.AT-02).",
		IssuingAuthority: certIssuer, ValidityMonths: 12, MinOverallScore: 60, MinControlsPassed: 0.5,
		BadgeIconURL: "/images/certs/nist-training.png", ColorScheme: certColorNavyBlue,
	},

	// ── ISO 27001 ──
	{
		Slug: "iso27001-compliance-readiness", FrameworkSlug: "iso27001",
		Name: "ISO 27001 Compliance Readiness", Description: "Demonstrates alignment with ISO 27001 information security management system requirements including Annex A awareness controls.",
		IssuingAuthority: certIssuer, ValidityMonths: 12, MinOverallScore: 70, MinControlsPassed: 0.7,
		BadgeIconURL: "/images/certs/iso27001-compliance.png", ColorScheme: "#1c2833",
	},
	{
		Slug: "iso27001-awareness-training", FrameworkSlug: "iso27001",
		Name: "ISO 27001 Security Awareness Training", Description: "Completion of ISO 27001-aligned security awareness training covering A.6.3 and A.7 controls.",
		IssuingAuthority: certIssuer, ValidityMonths: 12, MinOverallScore: 60, MinControlsPassed: 0.5,
		BadgeIconURL: "/images/certs/iso27001-training.png", ColorScheme: "#2c3e50",
	},

	// ── GDPR ──
	{
		Slug: "gdpr-compliance-readiness", FrameworkSlug: "gdpr",
		Name: "GDPR Compliance Readiness", Description: "Demonstrates organizational readiness for GDPR data protection requirements including staff awareness and data handling.",
		IssuingAuthority: certIssuer, ValidityMonths: 12, MinOverallScore: 70, MinControlsPassed: 0.7,
		BadgeIconURL: "/images/certs/gdpr-compliance.png", ColorScheme: "#003399",
	},

	// ── SOC 2 ──
	{
		Slug: "soc2-compliance-readiness", FrameworkSlug: "soc2",
		Name: "SOC 2 Compliance Readiness", Description: "Demonstrates alignment with SOC 2 Trust Services Criteria for security, availability, and confidentiality.",
		IssuingAuthority: certIssuer, ValidityMonths: 12, MinOverallScore: 75, MinControlsPassed: 0.7,
		BadgeIconURL: "/images/certs/soc2-compliance.png", ColorScheme: "#2e4053",
	},

	// ── Cyber Essentials (UK) ──
	{
		Slug: "cyber-essentials-readiness", FrameworkSlug: "cyber_essentials",
		Name: "Cyber Essentials Readiness", Description: "Demonstrates alignment with UK Cyber Essentials certification requirements for basic cyber hygiene.",
		IssuingAuthority: certIssuer, ValidityMonths: 12, MinOverallScore: 70, MinControlsPassed: 0.7,
		BadgeIconURL: "/images/certs/cyber-essentials.png", ColorScheme: certColorNavyBlue,
	},
}

// frameworkCertMap provides O(1) lookup of certs by slug.
var frameworkCertMap map[string]FrameworkComplianceCert

// frameworkCertsByFramework provides O(1) lookup of certs by framework slug.
var frameworkCertsByFramework map[string][]FrameworkComplianceCert

func init() {
	frameworkCertMap = make(map[string]FrameworkComplianceCert, len(BuiltInFrameworkCerts))
	frameworkCertsByFramework = make(map[string][]FrameworkComplianceCert)
	for _, c := range BuiltInFrameworkCerts {
		frameworkCertMap[c.Slug] = c
		frameworkCertsByFramework[c.FrameworkSlug] = append(frameworkCertsByFramework[c.FrameworkSlug], c)
	}
}

// GetFrameworkComplianceCert returns a framework cert definition by slug.
func GetFrameworkComplianceCert(slug string) *FrameworkComplianceCert {
	c, ok := frameworkCertMap[slug]
	if !ok {
		return nil
	}
	return &c
}

// GetFrameworkComplianceCerts returns all framework compliance certificates.
func GetFrameworkComplianceCerts() []FrameworkComplianceCert {
	return BuiltInFrameworkCerts
}

// GetCertsForFramework returns all compliance certificates for a specific framework.
func GetCertsForFramework(frameworkSlug string) []FrameworkComplianceCert {
	return frameworkCertsByFramework[frameworkSlug]
}

// ── Org-Level Framework Compliance Certificate Records ──

// OrgFrameworkCert records an earned framework compliance certificate for an org.
type OrgFrameworkCert struct {
	Id               int64     `json:"id" gorm:"primary_key"`
	OrgId            int64     `json:"org_id" gorm:"column:org_id"`
	CertSlug         string    `json:"cert_slug" gorm:"column:cert_slug"`
	FrameworkSlug    string    `json:"framework_slug" gorm:"column:framework_slug"`
	VerificationCode string    `json:"verification_code" gorm:"column:verification_code"`
	FrameworkScore   float64   `json:"framework_score" gorm:"column:framework_score"`
	ControlsPassed   int       `json:"controls_passed" gorm:"column:controls_passed"`
	TotalControls    int       `json:"total_controls" gorm:"column:total_controls"`
	IssuedDate       time.Time `json:"issued_date" gorm:"column:issued_date"`
	ExpiresDate      time.Time `json:"expires_date" gorm:"column:expires_date"`
	IsRevoked        bool      `json:"is_revoked" gorm:"column:is_revoked"`
	RevokedDate      time.Time `json:"revoked_date,omitempty" gorm:"column:revoked_date"`

	// Populated at query time
	CertName string `json:"cert_name,omitempty" gorm:"-"`
}

func (OrgFrameworkCert) TableName() string { return "org_framework_certs" }

// EvaluateAndIssueFrameworkCerts checks all framework compliance certs for an org
// and auto-issues any that the org now qualifies for.
func EvaluateAndIssueFrameworkCerts(orgId int64) ([]OrgFrameworkCert, error) {
	var issued []OrgFrameworkCert

	frameworks, err := GetOrgFrameworks(orgId)
	if err != nil {
		return nil, err
	}

	for _, f := range frameworks {
		certs := GetCertsForFramework(f.Slug)
		if len(certs) == 0 {
			continue
		}

		summary, err := GetFrameworkSummary(orgId, f.Id, false)
		if err != nil {
			log.Errorf("framework_cert: error getting summary for %s: %v", f.Slug, err)
			continue
		}

		for _, certDef := range certs {
			cert, ok := tryIssueFrameworkCert(orgId, certDef, summary)
			if ok {
				issued = append(issued, cert)
			}
		}
	}

	return issued, nil
}

// tryIssueFrameworkCert checks whether a single framework cert should be issued
// for an org and issues it if so. Returns the cert and true when issued.
func tryIssueFrameworkCert(orgId int64, certDef FrameworkComplianceCert, summary FrameworkSummary) (OrgFrameworkCert, bool) {
	// Check if already issued and still valid
	existing := OrgFrameworkCert{}
	err := db.Where("org_id = ? AND cert_slug = ? AND is_revoked = 0", orgId, certDef.Slug).First(&existing).Error
	if err == nil && !existing.ExpiresDate.Before(time.Now().UTC()) {
		return OrgFrameworkCert{}, false // Already has a valid cert
	}

	// Check qualification
	if !meetsFrameworkCertThreshold(certDef, summary) {
		return OrgFrameworkCert{}, false
	}

	// Issue the cert
	code, genErr := generateVerificationCode()
	if genErr != nil {
		log.Errorf("framework_cert: failed to generate code: %v", genErr)
		return OrgFrameworkCert{}, false
	}

	cert := OrgFrameworkCert{
		OrgId:            orgId,
		CertSlug:         certDef.Slug,
		FrameworkSlug:    certDef.FrameworkSlug,
		VerificationCode: code,
		FrameworkScore:   summary.OverallScore,
		ControlsPassed:   summary.Compliant,
		TotalControls:    summary.TotalControls,
		IssuedDate:       time.Now().UTC(),
		ExpiresDate:      time.Now().UTC().AddDate(0, certDef.ValidityMonths, 0),
	}
	if err := db.Save(&cert).Error; err != nil {
		log.Errorf("framework_cert: failed to save cert %s for org %d: %v", certDef.Slug, orgId, err)
		return OrgFrameworkCert{}, false
	}
	cert.CertName = certDef.Name

	log.Infof("framework_cert: issued %s for org %d (score=%.1f, passed=%d/%d)",
		certDef.Name, orgId, summary.OverallScore, summary.Compliant, summary.TotalControls)

	return cert, true
}

// meetsFrameworkCertThreshold checks if a framework summary meets the cert's minimum requirements.
func meetsFrameworkCertThreshold(certDef FrameworkComplianceCert, summary FrameworkSummary) bool {
	if summary.OverallScore < certDef.MinOverallScore {
		return false
	}
	passedFraction := 0.0
	if summary.TotalControls > 0 {
		passedFraction = float64(summary.Compliant) / float64(summary.TotalControls)
	}
	return passedFraction >= certDef.MinControlsPassed
}

// GetOrgFrameworkCerts returns all framework compliance certs for an org.
func GetOrgFrameworkCerts(orgId int64) ([]OrgFrameworkCert, error) {
	certs := []OrgFrameworkCert{}
	err := db.Where(queryWhereOrgID, orgId).Order("issued_date desc").Find(&certs).Error
	if err != nil {
		return nil, err
	}
	for i := range certs {
		if def := GetFrameworkComplianceCert(certs[i].CertSlug); def != nil {
			certs[i].CertName = def.Name
		}
	}
	return certs, nil
}

// GetActiveOrgFrameworkCerts returns non-revoked, non-expired framework certs.
func GetActiveOrgFrameworkCerts(orgId int64) ([]OrgFrameworkCert, error) {
	certs := []OrgFrameworkCert{}
	now := time.Now().UTC()
	err := db.Where(
		"org_id = ? AND is_revoked = 0 AND expires_date > ?", orgId, now,
	).Order("issued_date desc").Find(&certs).Error
	if err != nil {
		return nil, err
	}
	for i := range certs {
		if def := GetFrameworkComplianceCert(certs[i].CertSlug); def != nil {
			certs[i].CertName = def.Name
		}
	}
	return certs, nil
}

// VerifyOrgFrameworkCert verifies a framework cert by code.
func VerifyOrgFrameworkCert(code string) (*OrgFrameworkCert, error) {
	cert := OrgFrameworkCert{}
	err := db.Where("verification_code = ?", code).First(&cert).Error
	if err != nil {
		return nil, err
	}
	if def := GetFrameworkComplianceCert(cert.CertSlug); def != nil {
		cert.CertName = def.Name
	}
	return &cert, nil
}

// RevokeOrgFrameworkCert revokes an org-level framework cert.
func RevokeOrgFrameworkCert(id int64) error {
	return db.Model(&OrgFrameworkCert{}).Where("id = ?", id).Updates(map[string]interface{}{
		"is_revoked":   true,
		"revoked_date": time.Now().UTC(),
	}).Error
}

// FrameworkCertSummary provides a summary of framework cert status for an org.
type FrameworkCertSummary struct {
	Available  int `json:"available"`  // Total framework certs available
	Earned     int `json:"earned"`     // Currently active/valid
	Expired    int `json:"expired"`    // Expired but not revoked
	Qualifying int `json:"qualifying"` // Could be earned based on current scores
}

// GetFrameworkCertSummary returns cert summary stats for an org.
func GetFrameworkCertSummary(orgId int64) FrameworkCertSummary {
	summary := FrameworkCertSummary{}
	now := time.Now().UTC()

	// Count frameworks enabled for this org
	frameworks, _ := GetOrgFrameworks(orgId)
	for _, f := range frameworks {
		summary.Available += len(GetCertsForFramework(f.Slug))
	}

	// Count earned (active)
	db.Table("org_framework_certs").
		Where("org_id = ? AND is_revoked = 0 AND expires_date > ?", orgId, now).
		Count(&summary.Earned)

	// Count expired
	db.Table("org_framework_certs").
		Where("org_id = ? AND is_revoked = 0 AND expires_date <= ?", orgId, now).
		Count(&summary.Expired)

	// Count qualifying (could earn if evaluated now)
	summary.Qualifying = countQualifyingCerts(orgId, frameworks, now)

	return summary
}

// countQualifyingCerts counts how many framework certs an org qualifies for but hasn't earned yet.
func countQualifyingCerts(orgId int64, frameworks []ComplianceFramework, now time.Time) int {
	qualifying := 0
	for _, f := range frameworks {
		fSummary, err := GetFrameworkSummary(orgId, f.Id, false)
		if err != nil {
			continue
		}
		for _, certDef := range GetCertsForFramework(f.Slug) {
			if !meetsFrameworkCertThreshold(certDef, fSummary) {
				continue
			}
			// Check if not already earned
			existing := OrgFrameworkCert{}
			err := db.Where("org_id = ? AND cert_slug = ? AND is_revoked = 0 AND expires_date > ?",
				orgId, certDef.Slug, now).First(&existing).Error
			if err != nil { // Not found = qualifies but not earned
				qualifying++
			}
		}
	}
	return qualifying
}
