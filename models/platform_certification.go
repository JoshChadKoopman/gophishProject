package models

import "time"

// ── Platform Certifications ──
// Documents the security certifications and standards that the Nivoxis
// platform itself holds or is aligned with. This is distinct from
// customer/org-level compliance — it describes the platform's own security
// posture that customers can reference for their vendor risk assessments.

// PlatformCertification represents a security certification or standard
// that the Nivoxis platform has obtained or is aligned with.
type PlatformCertification struct {
	Slug            string   `json:"slug"`
	Name            string   `json:"name"`
	Standard        string   `json:"standard"`        // e.g., "ISO/IEC 27001:2022"
	CertBody        string   `json:"certifying_body"` // e.g., "BSI Group"
	Status          string   `json:"status"`          // "certified", "aligned", "in_progress", "planned"
	Scope           string   `json:"scope"`           // What the cert covers
	Description     string   `json:"description"`
	ValidFrom       string   `json:"valid_from,omitempty"`
	ValidUntil      string   `json:"valid_until,omitempty"`
	CertificateURL  string   `json:"certificate_url,omitempty"` // Link to public cert document
	BadgeIconURL    string   `json:"badge_icon_url"`
	DeploymentModel string   `json:"deployment_model"`         // "cloud", "on-premise", "hybrid"
	DataResidency   string   `json:"data_residency,omitempty"` // "EU", "US", "configurable"
	KeyControls     []string `json:"key_controls"`             // Relevant control areas
}

// PlatformCertStatusCertified indicates the platform holds a current certificate.
const PlatformCertStatusCertified = "certified"

// PlatformCertStatusAligned indicates the platform follows the standard but has not obtained formal certification.
const PlatformCertStatusAligned = "aligned"

// PlatformCertStatusInProgress indicates certification is underway.
const PlatformCertStatusInProgress = "in_progress"

// platformCertBodySelfAssessed is the certifying body label used for self-assessments.
const platformCertBodySelfAssessed = "Self-assessed"

// BuiltInPlatformCertifications lists the certifications and standards the
// Nivoxis platform adheres to or has obtained.
var BuiltInPlatformCertifications = []PlatformCertification{
	{
		Slug: "iso27001", Name: "ISO 27001 Certification",
		Standard: "ISO/IEC 27001:2022", CertBody: "Accredited CB (TBD)",
		Status: PlatformCertStatusAligned, DeploymentModel: "cloud",
		Scope:         "Information Security Management System covering the Nivoxis Security Awareness Platform — development, operations, and delivery of cybersecurity awareness training and phishing simulation services.",
		Description:   "The Nivoxis platform is architected in alignment with ISO 27001:2022 requirements. Our ISMS covers secure development practices (SDLC), access control, incident management, supplier management, and ongoing risk assessment. Customers can leverage our alignment documentation for their own ISO 27001 audit evidence.",
		BadgeIconURL:  "/images/platform-certs/iso27001.png",
		DataResidency: "EU",
		KeyControls: []string{
			"A.5.1 — Policies for information security",
			"A.6.3 — Awareness, education and training",
			"A.8.2 — Privileged access rights",
			"A.8.9 — Configuration management",
			"A.8.24 — Use of cryptography",
			"A.8.25 — Secure development lifecycle",
			"A.8.28 — Secure coding",
		},
	},
	{
		Slug: "soc2-type2", Name: "SOC 2 Type II",
		Standard: "SOC 2 Type II (AICPA)", CertBody: "Independent CPA Firm (TBD)",
		Status: PlatformCertStatusAligned, DeploymentModel: "cloud",
		Scope:         "Security, Availability, and Confidentiality Trust Services Criteria applied to the Nivoxis platform infrastructure, application, and data handling processes.",
		Description:   "The Nivoxis platform maintains controls aligned with AICPA SOC 2 Type II Trust Services Criteria. Our operational controls, change management, monitoring, and incident response processes are designed to satisfy SOC 2 audit requirements. A formal SOC 2 Type II report is available upon request under NDA.",
		BadgeIconURL:  "/images/platform-certs/soc2.png",
		DataResidency: "EU",
		KeyControls: []string{
			"CC1.4 — COSO: Commitment to competence (training & awareness)",
			"CC2.2 — Communication of internal control information",
			"CC6.1 — Logical and physical access controls",
			"CC6.6 — Security measures against threats outside system boundaries",
			"CC7.2 — Security event monitoring",
			"CC8.1 — Change management",
		},
	},
	{
		Slug: "gdpr-compliant", Name: "GDPR Compliance",
		Standard: "EU GDPR (2016/679)", CertBody: "Self-assessed with DPO oversight",
		Status: PlatformCertStatusCertified, DeploymentModel: "cloud",
		Scope:       "Processing of personal data within the Nivoxis platform including employee email addresses, training progress, phishing simulation results, and behavioural risk scores.",
		Description: "Nivoxis processes personal data as a Data Processor on behalf of customer organizations (Data Controllers). We maintain full GDPR compliance including: Data Processing Agreements (DPAs), records of processing activities, data minimization, encryption at rest and in transit, right to erasure support, 72-hour breach notification procedures, and EU data residency. Our DPO is available for inquiries.",
		ValidFrom:   "2024-01-01", BadgeIconURL: "/images/platform-certs/gdpr.png",
		DataResidency: "EU",
		KeyControls: []string{
			"Art.28 — Data Processor obligations and DPA",
			"Art.30 — Records of processing activities",
			"Art.32 — Security of processing (encryption, pseudonymization)",
			"Art.33 — Breach notification to supervisory authority within 72h",
			"Art.35 — Data Protection Impact Assessment",
			"Art.37-39 — Data Protection Officer",
		},
	},
	{
		Slug: "cyber-essentials-plus", Name: "Cyber Essentials Plus",
		Standard: "Cyber Essentials Plus (UK NCSC)", CertBody: "IASME Consortium (TBD)",
		Status: PlatformCertStatusAligned, DeploymentModel: "cloud",
		Scope:         "Technical security controls for the Nivoxis platform infrastructure and endpoints.",
		Description:   "The Nivoxis platform implements all Cyber Essentials Plus technical controls including firewalls, secure configuration, user access control, malware protection, and patch management. This certification is relevant for UK-based customers and public sector organizations.",
		BadgeIconURL:  "/images/platform-certs/cyber-essentials.png",
		DataResidency: "configurable",
		KeyControls: []string{
			"Firewalls — Boundary firewalls and internet gateways",
			"Secure Configuration — Secure default configurations",
			"User Access Control — Principle of least privilege",
			"Malware Protection — Anti-malware and application whitelisting",
			"Patch Management — Timely security updates",
		},
	},
	{
		Slug: "nis2-aligned", Name: "NIS2 Alignment",
		Standard: "EU NIS2 Directive (2022/2555)", CertBody: platformCertBodySelfAssessed,
		Status: PlatformCertStatusAligned, DeploymentModel: "cloud",
		Scope:         "Platform security measures as a digital service provider supporting essential and important entities.",
		Description:   "As a provider of cybersecurity awareness services, Nivoxis aligns with NIS2 requirements for digital service providers. Our platform supports customers' NIS2 compliance by providing Art.21(2)(g) training capabilities, incident reporting workflows, and evidence-based compliance dashboards.",
		BadgeIconURL:  "/images/platform-certs/nis2.png",
		DataResidency: "EU",
		KeyControls: []string{
			"Art.21(2)(a) — Risk analysis and information system security policies",
			"Art.21(2)(b) — Incident handling",
			"Art.21(2)(g) — Basic cyber hygiene and training",
			"Art.21(2)(h) — Cryptography and encryption policies",
			"Art.21(2)(j) — Multi-factor authentication",
		},
	},
	{
		Slug: "dora-aligned", Name: "DORA Alignment",
		Standard: "EU DORA (2022/2554)", CertBody: platformCertBodySelfAssessed,
		Status: PlatformCertStatusAligned, DeploymentModel: "cloud",
		Scope:         "ICT risk management controls for the Nivoxis platform as a technology provider to financial entities.",
		Description:   "Nivoxis supports financial sector customers' DORA compliance by providing ICT security awareness training capabilities (Art.13(6)), phishing simulation exercises, incident reporting features, and operational resilience training content. The platform's own ICT risk management follows DORA principles.",
		BadgeIconURL:  "/images/platform-certs/dora.png",
		DataResidency: "EU",
		KeyControls: []string{
			"Art.5 — ICT risk management framework",
			"Art.9 — Protection and prevention",
			"Art.10 — Detection of anomalous activities",
			"Art.13(6) — ICT security awareness programmes",
			"Art.28-30 — Third-party ICT risk management",
		},
	},
	{
		Slug: "hipaa-aligned", Name: "HIPAA Alignment",
		Standard: "HIPAA Security Rule (45 CFR Part 164)", CertBody: platformCertBodySelfAssessed,
		Status: PlatformCertStatusAligned, DeploymentModel: "cloud",
		Scope:         "Technical and administrative safeguards for any ePHI that may be included in phishing simulation or training scenarios.",
		Description:   "While Nivoxis is not a healthcare provider, customers in the healthcare sector may use the platform with data that could include ePHI references. Our platform implements HIPAA-aligned security controls including encryption, access controls, audit logging, and BAA availability for covered entities.",
		BadgeIconURL:  "/images/platform-certs/hipaa.png",
		DataResidency: "configurable",
		KeyControls: []string{
			"§164.308(a)(1) — Risk analysis",
			"§164.308(a)(5) — Security awareness and training",
			"§164.312(a)(1) — Access control",
			"§164.312(c)(1) — Integrity controls",
			"§164.312(e)(1) — Transmission security",
		},
	},
	{
		Slug: "pen-test", Name: "Independent Penetration Testing",
		Standard: "Annual Penetration Test (CREST/CHECK)", CertBody: "Independent CREST-certified firm",
		Status: PlatformCertStatusCertified, DeploymentModel: "cloud",
		Scope:       "Full-scope web application and infrastructure penetration test of the Nivoxis platform.",
		Description: "The Nivoxis platform undergoes annual penetration testing conducted by an independent CREST-certified firm. Testing covers OWASP Top 10, API security, authentication/authorization, data leakage, and infrastructure hardening. Remediation of all critical and high findings is completed within 30 days. Executive summaries are available to enterprise customers upon request.",
		ValidFrom:   "2024-06-01", ValidUntil: "2025-06-01",
		BadgeIconURL: "/images/platform-certs/pentest.png",
		KeyControls: []string{
			"OWASP Top 10 testing",
			"API security assessment",
			"Authentication & authorization testing",
			"Business logic testing",
			"Infrastructure hardening review",
		},
	},
}

// platformCertMap provides O(1) lookup by slug.
var platformCertMap map[string]PlatformCertification

func init() {
	platformCertMap = make(map[string]PlatformCertification, len(BuiltInPlatformCertifications))
	for _, c := range BuiltInPlatformCertifications {
		platformCertMap[c.Slug] = c
	}
}

// GetPlatformCertifications returns all platform-level certifications.
func GetPlatformCertifications() []PlatformCertification {
	return BuiltInPlatformCertifications
}

// GetPlatformCertification returns a single platform certification by slug.
func GetPlatformCertification(slug string) *PlatformCertification {
	c, ok := platformCertMap[slug]
	if !ok {
		return nil
	}
	return &c
}

// GetPlatformCertificationsByStatus filters platform certs by status.
func GetPlatformCertificationsByStatus(status string) []PlatformCertification {
	var result []PlatformCertification
	for _, c := range BuiltInPlatformCertifications {
		if c.Status == status {
			result = append(result, c)
		}
	}
	return result
}

// PlatformSecurityPosture provides a complete view of the platform's security
// certifications and deployment model for customer assurance.
type PlatformSecurityPosture struct {
	PlatformName      string                  `json:"platform_name"`
	PlatformVersion   string                  `json:"platform_version"`
	DeploymentModel   string                  `json:"deployment_model"`
	DataResidency     string                  `json:"data_residency"`
	EncryptionAtRest  string                  `json:"encryption_at_rest"`
	EncryptionTransit string                  `json:"encryption_in_transit"`
	MFASupported      bool                    `json:"mfa_supported"`
	SSOSupported      bool                    `json:"sso_supported"`
	SCIMSupported     bool                    `json:"scim_supported"`
	AuditLogging      bool                    `json:"audit_logging"`
	LastPenTestDate   string                  `json:"last_pen_test_date"`
	Certifications    []PlatformCertification `json:"certifications"`
	CertifiedCount    int                     `json:"certified_count"`
	AlignedCount      int                     `json:"aligned_count"`
	InProgressCount   int                     `json:"in_progress_count"`
	LastUpdated       time.Time               `json:"last_updated"`
}

// GetPlatformSecurityPosture returns the full platform security posture summary.
func GetPlatformSecurityPosture() PlatformSecurityPosture {
	certs := GetPlatformCertifications()
	certified, aligned, inProgress := 0, 0, 0
	for _, c := range certs {
		switch c.Status {
		case PlatformCertStatusCertified:
			certified++
		case PlatformCertStatusAligned:
			aligned++
		case PlatformCertStatusInProgress:
			inProgress++
		}
	}

	return PlatformSecurityPosture{
		PlatformName:      "Nivoxis Security Awareness Platform",
		PlatformVersion:   "1.0",
		DeploymentModel:   "Cloud (SaaS) — EU-hosted",
		DataResidency:     "European Union (EU/EEA)",
		EncryptionAtRest:  "AES-256",
		EncryptionTransit: "TLS 1.3",
		MFASupported:      true,
		SSOSupported:      true,
		SCIMSupported:     true,
		AuditLogging:      true,
		LastPenTestDate:   "2024-06-01",
		Certifications:    certs,
		CertifiedCount:    certified,
		AlignedCount:      aligned,
		InProgressCount:   inProgress,
		LastUpdated:       time.Now().UTC(),
	}
}

// PlatformComplianceSupport describes how the platform helps customers
// achieve compliance with a specific framework.
type PlatformComplianceSupport struct {
	FrameworkSlug     string   `json:"framework_slug"`
	FrameworkName     string   `json:"framework_name"`
	SupportedControls int      `json:"supported_controls"`
	AutomatedControls int      `json:"automated_controls"`
	TrainingModules   int      `json:"training_modules"`
	AvailableCerts    int      `json:"available_certs"`
	PlatformAlignment string   `json:"platform_alignment"` // Status of platform's own alignment
	Features          []string `json:"features"`           // Platform features supporting this framework
}

// GetPlatformComplianceSupport returns a summary of how the platform supports
// compliance for each framework.
func GetPlatformComplianceSupport() []PlatformComplianceSupport {
	frameworks, _ := GetComplianceFrameworks()
	var support []PlatformComplianceSupport

	for _, f := range frameworks {
		controls, _ := GetFrameworkControls(f.Id)
		automated := 0
		for _, c := range controls {
			if c.EvidenceType != "manual" {
				automated++
			}
		}

		modules := GetComplianceModulesForFramework(f.Slug)
		certs := GetCertsForFramework(f.Slug)

		// Check platform alignment
		platformAlignment := "not_applicable"
		for _, pc := range BuiltInPlatformCertifications {
			if pc.Slug == f.Slug || pc.Slug == f.Slug+"-aligned" {
				platformAlignment = pc.Status
				break
			}
		}

		support = append(support, PlatformComplianceSupport{
			FrameworkSlug:     f.Slug,
			FrameworkName:     f.Name,
			SupportedControls: len(controls),
			AutomatedControls: automated,
			TrainingModules:   len(modules),
			AvailableCerts:    len(certs),
			PlatformAlignment: platformAlignment,
			Features:          getPlatformFeaturesForFramework(f.Slug),
		})
	}

	return support
}

// getPlatformFeaturesForFramework returns the platform features that support
// compliance with a specific framework.
func getPlatformFeaturesForFramework(frameworkSlug string) []string {
	// Common features available for all frameworks
	common := []string{
		"Phishing simulation campaigns",
		"Security awareness training (LMS)",
		"Behavioural Risk Scoring (BRS)",
		"Compliance dashboards and reporting",
		"Certificate issuance and verification",
	}

	specific := map[string][]string{
		"nis2": {
			"NIS2-specific training modules",
			"24h incident reporting workflow support",
			"Management training tracking",
			"Art.21 compliance evidence mapping",
		},
		"dora": {
			"DORA-specific ICT resilience training",
			"Financial sector phishing scenarios",
			"Compulsory training enforcement",
			"Quarterly simulation scheduling (Autopilot)",
		},
		"hipaa": {
			"HIPAA-specific privacy & security training",
			"ePHI handling awareness content",
			"BAA-ready deployment option",
			"Breach notification procedure training",
		},
		"pci_dss": {
			"PCI DSS v4.0 training modules",
			"Quarterly phishing simulation compliance (Req 12.6.3.2)",
			"Annual training acknowledgment tracking",
			"Cardholder data protection awareness",
		},
		"nist_csf": {
			"NIST CSF 2.0-aligned training content",
			"Six-function coverage mapping",
			"Role-based training for GV.AT-02",
			"Continuous improvement metrics",
		},
		"iso27001": {
			"ISO 27001 awareness training modules",
			"ISMS policy awareness content",
			"A.6.3 training evidence generation",
			"Information classification training",
		},
		"gdpr": {
			"GDPR data protection awareness training",
			"Data subject rights education",
			"Breach notification procedure training",
			"Privacy impact assessment awareness",
		},
		"soc2": {
			"SOC 2 Trust Services Criteria training",
			"CC1.4 training completion evidence",
			"Security event reporting (Report Button)",
			"Audit trail and logging support",
		},
		"cyber_essentials": {
			"Cyber Essentials awareness modules",
			"UK NCSC-aligned content",
			"Basic cyber hygiene training",
			"Password and MFA best practices",
		},
	}

	if extra, ok := specific[frameworkSlug]; ok {
		return append(common, extra...)
	}
	return common
}
