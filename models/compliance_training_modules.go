package models

// ── Framework-Specific Training Content ──
// Dedicated microlearning modules mapped to specific compliance framework
// requirements. These modules are automatically available when an org
// enables a compliance framework and complement the general content library.

// ComplianceTrainingModule is a pre-built training module dedicated to a
// specific compliance framework. Each module maps to one or more framework
// controls and includes pages and a quiz.
type ComplianceTrainingModule struct {
	Slug             string         `json:"slug"`
	FrameworkSlug    string         `json:"framework_slug"`
	ControlRefs      []string       `json:"control_refs"` // Framework control references covered
	Title            string         `json:"title"`
	Description      string         `json:"description"`
	DifficultyLevel  int            `json:"difficulty_level"`
	EstimatedMinutes int            `json:"estimated_minutes"`
	Tags             []string       `json:"tags"`
	Pages            []TrainingPage `json:"pages"`
	Quiz             *BuiltInQuiz   `json:"quiz,omitempty"`
}

// BuiltInComplianceModules contains the framework-specific training modules.
var BuiltInComplianceModules = []ComplianceTrainingModule{

	// ══════════════════════════════════════
	// NIS2 FRAMEWORK MODULES
	// ══════════════════════════════════════
	{
		Slug: "nis2-awareness-obligations", FrameworkSlug: "nis2",
		ControlRefs: []string{"Art.21(2)(g)"}, Title: "NIS2: Cybersecurity Awareness Obligations",
		Description:     "Understand NIS2 Art.21(2)(g) requirements for cybersecurity awareness training, including management responsibilities and regular training programmes.",
		DifficultyLevel: ContentDiffSilver, EstimatedMinutes: 12, Tags: []string{"nis2", "compliance", "eu"},
		Pages: []TrainingPage{
			{Title: "What is NIS2?", Body: "The **NIS2 Directive** (2022/2555) is the EU's updated cybersecurity law, replacing the original NIS Directive. It significantly expands the scope and requirements for organizations to manage cybersecurity risks.\n\n**Key changes from NIS1:**\n- Broader scope: covers more sectors and entity types\n- Stricter requirements for risk management measures\n- Mandatory incident reporting within 24 hours\n- Personal accountability for management bodies\n- Significant fines for non-compliance (up to €10M or 2% of global turnover)", TipBox: "NIS2 applies to 'essential' and 'important' entities across 18 sectors including energy, transport, health, digital infrastructure, and more."},
			{Title: "Art.21(2)(g): Training Requirements", Body: "Article 21(2)(g) specifically requires:\n\n> *\"Basic cyber hygiene practices and cybersecurity training\"*\n\nThis means your organization must:\n\n1. **Implement regular cybersecurity awareness programmes** for all staff\n2. **Include management** — board members and executives must also receive training\n3. **Cover basic cyber hygiene** — passwords, phishing, safe browsing, device security\n4. **Demonstrate compliance** — maintain records of training completion\n5. **Keep content current** — update training to reflect emerging threats", TipBox: "NIS2 explicitly requires management bodies to undergo training too — not just general staff."},
			{Title: "Phishing Simulation Requirements", Body: "While NIS2 doesn't prescribe specific phishing simulation frequencies, Art.21(2)(f) requires organizations to:\n\n> *\"Assess the effectiveness of cybersecurity risk-management measures\"*\n\nPhishing simulations are a recognized best practice for this. Industry standards recommend:\n\n- **Minimum 4 simulations per year** (quarterly)\n- **Varied difficulty levels** to test different scenarios\n- **Track improvement** over time to demonstrate effectiveness\n- **Link to remediation** — failed simulations trigger targeted training"},
			{Title: "Incident Reporting Under NIS2", Body: "Art.23 establishes strict incident reporting timelines:\n\n⏰ **Within 24 hours:** Early warning notification to your CSIRT/authority\n⏰ **Within 72 hours:** Full incident notification with initial assessment\n⏰ **Within 1 month:** Final report with root cause analysis\n\n**Your role:**\n- Know how to recognize a potential incident\n- Report suspicious activity immediately through proper channels\n- Don't wait to be sure — early warnings save time\n- Preserve evidence — don't delete suspicious emails", TipBox: "The 24-hour clock starts from when the incident is 'detected' — report suspicions immediately to your security team."},
			{Title: "Your Responsibilities Under NIS2", Body: "As an employee, you play a critical role in NIS2 compliance:\n\n✅ **Complete all assigned security training** on time\n✅ **Report suspicious emails** using the Report Button\n✅ **Follow password and MFA policies** without exception\n✅ **Report security incidents** immediately\n✅ **Keep software updated** on all devices\n✅ **Handle data carefully** — follow classification and sharing policies\n\nNon-compliance can result in significant penalties for your organization, including fines up to €10 million."},
		},
		Quiz: &BuiltInQuiz{
			PassPercentage: 80,
			Questions: []BuiltInQuestion{
				{QuestionText: "What does NIS2 Article 21(2)(g) specifically require?", Options: []string{"Annual penetration testing", "Basic cyber hygiene practices and cybersecurity training", "Encryption of all data at rest", "Monthly security audits"}, CorrectOption: 1},
				{QuestionText: "How quickly must a NIS2 early warning be sent after detecting an incident?", Options: []string{"Within 1 hour", "Within 24 hours", "Within 1 week", "Within 72 hours"}, CorrectOption: 1},
				{QuestionText: "Under NIS2, who must receive cybersecurity training?", Options: []string{"Only IT staff", "Only new employees", "All staff including management bodies", "Only employees handling customer data"}, CorrectOption: 2},
				{QuestionText: "What is the maximum fine for NIS2 non-compliance for essential entities?", Options: []string{"€1 million", "€5 million", "€10 million or 2% of global turnover", "€100,000"}, CorrectOption: 2},
			},
		},
	},

	{
		Slug: "nis2-risk-management", FrameworkSlug: "nis2",
		ControlRefs: []string{"Art.21(2)(a)", "Art.21(2)(f)"}, Title: "NIS2: Risk Analysis & Security Policies",
		Description:     "Learn about NIS2 risk analysis requirements and how your organization's security policies protect against network and information system threats.",
		DifficultyLevel: ContentDiffGold, EstimatedMinutes: 10, Tags: []string{"nis2", "risk", "policy"},
		Pages: []TrainingPage{
			{Title: "Risk-Based Approach", Body: "NIS2 requires a risk-based approach to cybersecurity. This means:\n\n1. **Identify assets** — Know what systems and data need protection\n2. **Assess threats** — Understand what could go wrong\n3. **Evaluate impact** — Determine the consequences of a breach\n4. **Implement controls** — Apply proportionate security measures\n5. **Monitor and review** — Continuously evaluate effectiveness\n\nThe Behavioural Risk Score (BRS) used in our platform is one way we measure the human risk factor."},
			{Title: "Your Role in Risk Management", Body: "Every employee contributes to risk management by:\n\n- **Following security policies** consistently\n- **Reporting vulnerabilities** when you notice them\n- **Maintaining good cyber hygiene** (updates, passwords, MFA)\n- **Participating in simulations** honestly — they help measure and reduce risk\n- **Completing training** on schedule — this directly impacts our compliance score"},
		},
		Quiz: &BuiltInQuiz{
			PassPercentage: 75,
			Questions: []BuiltInQuestion{
				{QuestionText: "What approach does NIS2 require for cybersecurity?", Options: []string{"One-size-fits-all", "Risk-based approach", "Technology-only approach", "Compliance-only approach"}, CorrectOption: 1},
				{QuestionText: "How does phishing simulation contribute to NIS2 compliance?", Options: []string{"It satisfies the encryption requirement", "It helps assess the effectiveness of cybersecurity measures", "It replaces the need for training", "It automates incident reporting"}, CorrectOption: 1},
			},
		},
	},

	// ══════════════════════════════════════
	// DORA FRAMEWORK MODULES
	// ══════════════════════════════════════
	{
		Slug: "dora-ict-awareness", FrameworkSlug: "dora",
		ControlRefs: []string{"Art.13(6)"}, Title: "DORA: ICT Security Awareness for Financial Services",
		Description:     "Understand DORA's requirements for ICT security awareness programmes and digital operational resilience training in the financial sector.",
		DifficultyLevel: ContentDiffSilver, EstimatedMinutes: 12, Tags: []string{"dora", "financial", "eu"},
		Pages: []TrainingPage{
			{Title: "What is DORA?", Body: "The **Digital Operational Resilience Act** (EU 2022/2554) ensures financial entities can withstand, respond to, and recover from all types of ICT-related disruptions and threats.\n\n**Who it applies to:**\n- Banks and credit institutions\n- Investment firms\n- Insurance companies\n- Payment service providers\n- Crypto-asset service providers\n- And their critical ICT third-party providers\n\n**Key pillars:** ICT Risk Management, Incident Reporting, Resilience Testing, Third-Party Risk, Information Sharing", TipBox: "DORA became applicable on 17 January 2025 — all covered entities must comply."},
			{Title: "Art.13(6): Training Requirements", Body: "Article 13(6) specifically mandates:\n\n> *\"Financial entities shall develop ICT security awareness programmes and digital operational resilience training as compulsory modules in their staff training schemes.\"*\n\nThis includes:\n- **Compulsory** training for all staff (not optional)\n- **Phishing simulation exercises** as part of the programme\n- Training must cover **ICT security awareness** and **operational resilience**\n- Management must be trained on **ICT risk governance**", TipBox: "DORA goes beyond general awareness — it requires specific 'digital operational resilience' training."},
			{Title: "ICT Risk Management Under DORA", Body: "DORA requires a comprehensive ICT risk management framework:\n\n🔒 **Protection & Prevention** — Secure systems, access controls, encryption\n🔍 **Detection** — Monitor for anomalous activities\n📋 **Response & Recovery** — Business continuity and disaster recovery plans\n📊 **Learning & Evolving** — Improve based on incidents and testing\n\nYour daily actions directly impact these areas:\n- Use MFA on all financial systems\n- Report unusual system behaviour\n- Follow data handling procedures\n- Keep devices and software updated"},
			{Title: "Incident Classification & Reporting", Body: "DORA requires classification and reporting of **major ICT-related incidents**:\n\n📌 Criteria for 'major' incidents include:\n- Number of affected clients/counterparties\n- Duration of the incident\n- Geographic spread\n- Data loss\n- Impact on critical services\n\n⏰ **Reporting timeline:** Initial notification → intermediate report → final report\n\n**Your role:** Report any ICT disruption or security incident immediately to your IT/Security team."},
		},
		Quiz: &BuiltInQuiz{
			PassPercentage: 80,
			Questions: []BuiltInQuestion{
				{QuestionText: "What does DORA stand for?", Options: []string{"Digital Operations Regulatory Act", "Digital Operational Resilience Act", "Data Operations and Reporting Act", "Directive on Operational Risk Assessment"}, CorrectOption: 1},
				{QuestionText: "What does Art.13(6) require regarding training?", Options: []string{"Optional workshops for IT staff", "Compulsory ICT security awareness and resilience training for all staff", "Annual compliance meetings", "External audit certifications"}, CorrectOption: 1},
				{QuestionText: "Which of the following is NOT a DORA pillar?", Options: []string{"ICT Risk Management", "Resilience Testing", "Marketing Strategy", "Third-Party Risk"}, CorrectOption: 2},
			},
		},
	},

	// ══════════════════════════════════════
	// HIPAA FRAMEWORK MODULES
	// ══════════════════════════════════════
	{
		Slug: "hipaa-security-training", FrameworkSlug: "hipaa",
		ControlRefs: []string{"§164.308(a)(5)", "§164.308(a)(5)(ii)(A)"}, Title: "HIPAA: Security Awareness Training",
		Description:     "Complete HIPAA-specific security awareness training covering ePHI protection, security reminders, and malicious software protection.",
		DifficultyLevel: ContentDiffSilver, EstimatedMinutes: 15, Tags: []string{"hipaa", "healthcare", "us"},
		Pages: []TrainingPage{
			{Title: "HIPAA and Your Responsibilities", Body: "The **Health Insurance Portability and Accountability Act (HIPAA)** protects the privacy and security of patient health information.\n\n**What is Protected Health Information (PHI)?**\n- Patient names, addresses, birth dates\n- Social Security numbers\n- Medical records and diagnoses\n- Health insurance information\n- Any data that could identify a patient\n\n**ePHI** = PHI in electronic form — emails, databases, files, messages\n\nAs a covered entity employee, you are legally responsible for protecting this data.", TipBox: "HIPAA violations can result in fines from $100 to $50,000 per violation, up to $1.5 million per year per violation category."},
			{Title: "The Security Rule Requirements", Body: "HIPAA's Security Rule (§164.308) establishes three types of safeguards:\n\n🏛️ **Administrative Safeguards**\n- Security awareness training (§164.308(a)(5))\n- Security incident procedures\n- Access management\n- Risk analysis\n\n🔧 **Technical Safeguards**\n- Access controls and audit logs\n- Integrity controls\n- Transmission security\n\n🏢 **Physical Safeguards**\n- Facility access controls\n- Workstation security\n- Device and media controls"},
			{Title: "Security Awareness Specifics", Body: "§164.308(a)(5) requires your organization to:\n\n1. **Security reminders** — Regular updates about security practices (phishing simulations count!)\n2. **Malicious software protection** — Know how to detect and avoid malware\n3. **Login monitoring** — Report unusual account activity\n4. **Password management** — Follow strong password practices\n\n**What you must do:**\n- Never share login credentials\n- Lock your workstation when away\n- Encrypt ePHI in emails\n- Report any suspected breach immediately\n- Use secure messaging for patient data"},
			{Title: "Breach Notification", Body: "If a breach of unsecured PHI occurs:\n\n⏰ **Individual notification:** Within 60 days of discovery\n📋 **HHS notification:** Within 60 days (if 500+ individuals affected)\n📰 **Media notification:** Required for breaches affecting 500+ residents of a state\n\n**What constitutes a breach?**\nAny unauthorized acquisition, access, use, or disclosure of PHI that compromises its security or privacy.\n\n**Your critical role:** Immediate reporting of suspected breaches to your Privacy/Security Officer. The 60-day clock starts from discovery."},
		},
		Quiz: &BuiltInQuiz{
			PassPercentage: 80,
			Questions: []BuiltInQuestion{
				{QuestionText: "What does ePHI stand for?", Options: []string{"Emergency Personal Health Info", "Electronic Protected Health Information", "External Patient Health Index", "Encrypted Private Health Input"}, CorrectOption: 1},
				{QuestionText: "Which HIPAA section specifically requires security awareness training?", Options: []string{"§164.312", "§164.310", "§164.308(a)(5)", "§164.306"}, CorrectOption: 2},
				{QuestionText: "Within how many days must individuals be notified of a PHI breach?", Options: []string{"30 days", "60 days", "90 days", "7 days"}, CorrectOption: 1},
				{QuestionText: "What is an acceptable way to transmit ePHI?", Options: []string{"Personal email", "Encrypted secure messaging", "Text message", "Social media direct message"}, CorrectOption: 1},
			},
		},
	},

	// ══════════════════════════════════════
	// PCI DSS FRAMEWORK MODULES
	// ══════════════════════════════════════
	{
		Slug: "pci-dss-security-awareness", FrameworkSlug: "pci_dss",
		ControlRefs: []string{"Req 12.6.1", "Req 12.6.2", "Req 12.6.3.1"}, Title: "PCI DSS: Security Awareness Program",
		Description:     "Learn about PCI DSS v4.0 security awareness requirements including formal training programs, phishing awareness, and annual acknowledgment.",
		DifficultyLevel: ContentDiffSilver, EstimatedMinutes: 12, Tags: []string{"pci-dss", "payment", "global"},
		Pages: []TrainingPage{
			{Title: "PCI DSS Overview", Body: "The **Payment Card Industry Data Security Standard (PCI DSS)** protects cardholder data wherever it is processed, stored, or transmitted.\n\n**PCI DSS v4.0** (effective March 2025) strengthens requirements with:\n- **Customized approach** as alternative to defined approach\n- **Enhanced authentication** requirements\n- **Expanded awareness training** with phishing simulations\n- **Targeted risk analysis** for flexible requirements\n\nIf your organization handles credit/debit card data, PCI DSS compliance is mandatory.", TipBox: "Non-compliance can result in fines from $5,000 to $100,000 per month from card brands, plus liability for fraud losses."},
			{Title: "Requirement 12.6: Security Awareness", Body: "PCI DSS v4.0 Requirement 12.6 establishes:\n\n📋 **12.6.1** — Formal security awareness program educating all personnel about cardholder data protection\n📋 **12.6.2** — Annual review and acknowledgment by all personnel\n📋 **12.6.3.1** — Training includes phishing and social engineering awareness\n📋 **12.6.3.2** — Phishing simulation exercises conducted at least quarterly\n\nThese requirements are **mandatory** — failure to comply can result in loss of card processing privileges."},
			{Title: "Protecting Cardholder Data", Body: "**What is Cardholder Data (CHD)?**\n- Primary Account Number (PAN)\n- Cardholder name\n- Expiration date\n- Service code\n\n**Sensitive Authentication Data (SAD):**\n- Full magnetic stripe data\n- CVV/CVC codes\n- PINs and PIN blocks\n\n**Rules:**\n- Never store SAD after authorization\n- Mask PAN when displayed (show only last 4 digits)\n- Encrypt PAN in storage and transmission\n- Never send CHD via unencrypted email, chat, or messaging"},
		},
		Quiz: &BuiltInQuiz{
			PassPercentage: 80,
			Questions: []BuiltInQuestion{
				{QuestionText: "How often must phishing simulations be conducted under PCI DSS v4.0?", Options: []string{"Monthly", "At least quarterly", "Annually", "Semi-annually"}, CorrectOption: 1},
				{QuestionText: "Which data must NEVER be stored after card authorization?", Options: []string{"Cardholder name", "Expiration date", "CVV/CVC code", "Last 4 digits of PAN"}, CorrectOption: 2},
				{QuestionText: "What does PCI DSS Requirement 12.6.2 require?", Options: []string{"Monthly penetration testing", "Annual review and acknowledgment by all personnel", "Quarterly vulnerability scans", "Daily log monitoring"}, CorrectOption: 1},
			},
		},
	},

	// ══════════════════════════════════════
	// NIST CSF FRAMEWORK MODULES
	// ══════════════════════════════════════
	{
		Slug: "nist-csf-awareness", FrameworkSlug: "nist_csf",
		ControlRefs: []string{"GV.AT-01", "GV.AT-02", "PR.AT-01"}, Title: "NIST CSF: Cybersecurity Awareness & Training",
		Description:     "Understand NIST Cybersecurity Framework 2.0 awareness and training requirements across the Govern and Protect functions.",
		DifficultyLevel: ContentDiffSilver, EstimatedMinutes: 10, Tags: []string{"nist", "csf", "us"},
		Pages: []TrainingPage{
			{Title: "NIST CSF 2.0 Overview", Body: "The **NIST Cybersecurity Framework (CSF) 2.0** is a voluntary framework used worldwide to manage cybersecurity risk.\n\n**Six Core Functions:**\n\n🏛️ **Govern** — Establish cybersecurity strategy and policies\n🔍 **Identify** — Understand and manage cybersecurity risk\n🛡️ **Protect** — Implement safeguards for critical services\n🔎 **Detect** — Identify cybersecurity events\n⚡ **Respond** — Take action on detected events\n🔄 **Recover** — Restore capabilities after incidents\n\n**Govern is NEW in CSF 2.0** — it emphasizes that cybersecurity is a business-level risk, not just an IT issue.", TipBox: "CSF 2.0 is designed for organizations of all sizes and sectors. It complements sector-specific regulations like HIPAA and PCI DSS."},
			{Title: "Awareness & Training Controls", Body: "NIST CSF 2.0 has specific controls for awareness and training:\n\n📋 **GV.AT-01:** Personnel are provided awareness and training to perform tasks with cybersecurity risk in mind\n📋 **GV.AT-02:** Individuals in specialized roles receive role-based training\n📋 **PR.AT-01:** All users are informed and trained\n📋 **PR.AT-02:** Users understand phishing and social engineering risks\n\nThese controls recognize that people are both the weakest link and the strongest defense."},
			{Title: "Practical NIST CSF Application", Body: "How NIST CSF maps to what we do:\n\n| CSF Function | Our Platform Feature |\n|---|---|\n| Govern | Adaptive difficulty engine, risk scoring |\n| Identify | BRS (Behavioural Risk Score) per user |\n| Protect | Training modules, phishing awareness |\n| Detect | Report Button, phishing simulations |\n| Respond | Escalation workflows, incident response training |\n| Recover | Remediation paths, continuous improvement |\n\nYour participation in training and simulations directly supports multiple NIST CSF controls."},
		},
		Quiz: &BuiltInQuiz{
			PassPercentage: 75,
			Questions: []BuiltInQuestion{
				{QuestionText: "How many core functions does NIST CSF 2.0 have?", Options: []string{"4", "5", "6", "7"}, CorrectOption: 2},
				{QuestionText: "Which function is NEW in NIST CSF 2.0?", Options: []string{"Protect", "Detect", "Govern", "Recover"}, CorrectOption: 2},
				{QuestionText: "What does control GV.AT-01 require?", Options: []string{"Encryption of all data", "Awareness and training for all personnel", "Annual penetration testing", "24/7 security monitoring"}, CorrectOption: 1},
			},
		},
	},

	// ══════════════════════════════════════
	// ISO 27001 FRAMEWORK MODULES
	// ══════════════════════════════════════
	{
		Slug: "iso27001-awareness", FrameworkSlug: "iso27001",
		ControlRefs: []string{"A.6.3", "A.7.2.2"}, Title: "ISO 27001: Information Security Awareness",
		Description:     "Understand ISO 27001 ISMS requirements for information security awareness, education, and training (Annex A controls A.6.3 and A.7).",
		DifficultyLevel: ContentDiffSilver, EstimatedMinutes: 10, Tags: []string{"iso27001", "isms", "global"},
		Pages: []TrainingPage{
			{Title: "What is ISO 27001?", Body: "**ISO/IEC 27001** is the international standard for Information Security Management Systems (ISMS). It provides a systematic approach to managing sensitive information.\n\n**Key principles:**\n- **Confidentiality** — Information accessible only to authorized individuals\n- **Integrity** — Information accurate and complete\n- **Availability** — Information accessible when needed\n\n**ISO 27001:2022** includes 93 controls organized into 4 themes:\n- Organizational (37 controls)\n- People (8 controls)\n- Physical (14 controls)\n- Technological (34 controls)", TipBox: "ISO 27001 certification is often required by enterprise customers and partners as proof of security maturity."},
			{Title: "Annex A.6.3: Awareness, Education & Training", Body: "Control A.6.3 requires:\n\n> *\"Personnel of the organization and relevant interested parties shall receive appropriate information security awareness, education and training and regular updates of the organization's information security policy.\"*\n\nThis means:\n- **All personnel** must receive security awareness training\n- Training must be **appropriate** to their role\n- Regular **updates** as policies change\n- **Documented evidence** of training delivery and completion\n- Training must cover the organization's **specific policies**"},
			{Title: "Your ISMS Responsibilities", Body: "Under your organization's ISMS, you must:\n\n✅ **Follow the information security policy** — Read it, understand it, comply with it\n✅ **Classify information correctly** — Use the right labels (Public, Internal, Confidential, Restricted)\n✅ **Handle data according to its classification** — Don't email restricted documents\n✅ **Report security events** — Any suspected breach or weakness\n✅ **Maintain clean desk/screen** — Lock devices, clear desks\n✅ **Use approved tools** — Don't use shadow IT or personal cloud storage for work data"},
		},
		Quiz: &BuiltInQuiz{
			PassPercentage: 75,
			Questions: []BuiltInQuestion{
				{QuestionText: "What are the three principles of information security (CIA triad)?", Options: []string{"Cost, Impact, Authority", "Confidentiality, Integrity, Availability", "Compliance, Integration, Access", "Control, Investigation, Audit"}, CorrectOption: 1},
				{QuestionText: "What does ISO 27001 Annex A.6.3 require?", Options: []string{"Annual financial audits", "Appropriate security awareness, education and training for all personnel", "Bi-annual penetration tests", "Weekly vulnerability scans"}, CorrectOption: 1},
				{QuestionText: "How many controls does ISO 27001:2022 contain?", Options: []string{"27", "53", "93", "114"}, CorrectOption: 2},
			},
		},
	},

	// ══════════════════════════════════════
	// GDPR FRAMEWORK MODULE
	// ══════════════════════════════════════
	{
		Slug: "gdpr-data-protection", FrameworkSlug: "gdpr",
		ControlRefs: []string{"Art.39(1)(b)", "Art.47(2)(n)"}, Title: "GDPR: Data Protection Awareness",
		Description:     "Comprehensive GDPR data protection awareness covering personal data handling, lawful bases, data subject rights, and breach reporting.",
		DifficultyLevel: ContentDiffSilver, EstimatedMinutes: 15, Tags: []string{"gdpr", "privacy", "eu"},
		Pages: []TrainingPage{
			{Title: "GDPR Fundamentals", Body: "The **General Data Protection Regulation (GDPR)** is the EU's comprehensive data protection law. It applies to any organization that processes personal data of EU/EEA residents.\n\n**Key definitions:**\n- **Personal data** — Any information relating to an identifiable person (name, email, IP address, location data, etc.)\n- **Processing** — Any operation performed on personal data (collection, storage, use, sharing, deletion)\n- **Data controller** — Determines why and how data is processed\n- **Data processor** — Processes data on behalf of the controller\n\n**Penalties:** Up to €20 million or 4% of global annual turnover, whichever is higher.", TipBox: "GDPR applies even if your organization is outside the EU — if you process EU residents' data, you must comply."},
			{Title: "Lawful Bases for Processing", Body: "You must have a **lawful basis** to process personal data. The 6 bases are:\n\n1. **Consent** — Freely given, specific, informed\n2. **Contract** — Necessary to fulfil a contract\n3. **Legal obligation** — Required by law\n4. **Vital interests** — Protecting someone's life\n5. **Public task** — Necessary for public interest\n6. **Legitimate interests** — Balanced against data subject rights\n\n**Special category data** (health, biometrics, political opinions, etc.) requires additional justification."},
			{Title: "Data Subject Rights", Body: "Individuals have extensive rights under GDPR:\n\n📋 **Right of access** — Know what data is held about them\n✏️ **Right to rectification** — Correct inaccurate data\n🗑️ **Right to erasure** — Request deletion ('right to be forgotten')\n⏸️ **Right to restrict processing** — Limit how data is used\n📦 **Right to data portability** — Receive data in machine-readable format\n❌ **Right to object** — Object to certain processing\n🤖 **Rights related to automated decisions** — Not be subject to solely automated decisions\n\n**Response time:** You must respond to requests within 1 month."},
			{Title: "Breach Reporting Under GDPR", Body: "If a personal data breach occurs:\n\n⏰ **72 hours** — Notify the supervisory authority (unless unlikely to result in risk to individuals)\n📧 **Without undue delay** — Notify affected individuals if high risk\n\n**What constitutes a breach?**\nA breach of security leading to accidental or unlawful destruction, loss, alteration, unauthorized disclosure of, or access to personal data.\n\n**Examples:**\n- Sending an email with personal data to the wrong recipient\n- Lost/stolen laptop with unencrypted personal data\n- Ransomware encrypting a database containing personal data\n- Unauthorized employee accessing patient records"},
		},
		Quiz: &BuiltInQuiz{
			PassPercentage: 80,
			Questions: []BuiltInQuestion{
				{QuestionText: "What is the maximum GDPR fine?", Options: []string{"€1 million", "€10 million or 2% of turnover", "€20 million or 4% of turnover", "€50 million"}, CorrectOption: 2},
				{QuestionText: "How quickly must a data breach be reported to the supervisory authority?", Options: []string{"24 hours", "72 hours", "1 week", "1 month"}, CorrectOption: 1},
				{QuestionText: "Which is NOT a lawful basis for processing personal data?", Options: []string{"Consent", "Legitimate interests", "Business convenience", "Legal obligation"}, CorrectOption: 2},
				{QuestionText: "Within what timeframe must you respond to a data subject access request?", Options: []string{"72 hours", "1 week", "1 month", "3 months"}, CorrectOption: 2},
			},
		},
	},

	// ══════════════════════════════════════
	// SOC 2 FRAMEWORK MODULE
	// ══════════════════════════════════════
	{
		Slug: "soc2-trust-services", FrameworkSlug: "soc2",
		ControlRefs: []string{"CC1.4", "CC2.2"}, Title: "SOC 2: Trust Services Criteria Awareness",
		Description:     "Understand SOC 2 Trust Services Criteria covering security, availability, processing integrity, confidentiality, and privacy.",
		DifficultyLevel: ContentDiffGold, EstimatedMinutes: 10, Tags: []string{"soc2", "audit", "global"},
		Pages: []TrainingPage{
			{Title: "What is SOC 2?", Body: "**SOC 2** (System and Organization Controls 2) is an audit framework developed by the AICPA that evaluates an organization's controls related to data security.\n\n**Five Trust Services Criteria (TSC):**\n\n🔒 **Security** — Protection against unauthorized access\n⬆️ **Availability** — System accessible as agreed\n⚙️ **Processing Integrity** — Processing is complete, valid, and authorized\n🔏 **Confidentiality** — Information designated as confidential is protected\n🔐 **Privacy** — Personal information is collected, used, and disclosed properly\n\n**Two types of reports:**\n- **Type I** — Controls are suitably designed at a point in time\n- **Type II** — Controls are operating effectively over a period (usually 6-12 months)", TipBox: "SOC 2 Type II reports are considered the gold standard and are commonly required by enterprise customers."},
			{Title: "Your Role in SOC 2 Compliance", Body: "SOC 2 compliance requires consistent security practices from everyone:\n\n✅ **Complete security training** — Documented training is audited (CC1.4)\n✅ **Follow access policies** — Use MFA, don't share credentials\n✅ **Report incidents** — Every incident must be tracked\n✅ **Handle data properly** — Follow classification and handling procedures\n✅ **Maintain system hygiene** — Keep software updated\n\nAuditors will review evidence of training completion, incident handling, and policy compliance during the audit period."},
		},
		Quiz: &BuiltInQuiz{
			PassPercentage: 75,
			Questions: []BuiltInQuestion{
				{QuestionText: "How many Trust Services Criteria does SOC 2 cover?", Options: []string{"3", "4", "5", "6"}, CorrectOption: 2},
				{QuestionText: "What is the difference between SOC 2 Type I and Type II?", Options: []string{"Type I covers more criteria", "Type II evaluates controls over a period, Type I at a point in time", "Type I is for cloud providers, Type II for on-premise", "There is no difference"}, CorrectOption: 1},
			},
		},
	},
}

// complianceModuleMap provides O(1) lookup by slug.
var complianceModuleMap map[string]ComplianceTrainingModule

// complianceModulesByFramework groups modules by framework slug.
var complianceModulesByFramework map[string][]ComplianceTrainingModule

func init() {
	complianceModuleMap = make(map[string]ComplianceTrainingModule, len(BuiltInComplianceModules))
	complianceModulesByFramework = make(map[string][]ComplianceTrainingModule)
	for _, m := range BuiltInComplianceModules {
		complianceModuleMap[m.Slug] = m
		complianceModulesByFramework[m.FrameworkSlug] = append(complianceModulesByFramework[m.FrameworkSlug], m)
	}
}

// GetComplianceTrainingModules returns all framework-specific training modules.
func GetComplianceTrainingModules() []ComplianceTrainingModule {
	return BuiltInComplianceModules
}

// GetComplianceTrainingModule returns a single module by slug.
func GetComplianceTrainingModule(slug string) *ComplianceTrainingModule {
	m, ok := complianceModuleMap[slug]
	if !ok {
		return nil
	}
	return &m
}

// GetComplianceModulesForFramework returns all modules for a specific framework.
func GetComplianceModulesForFramework(frameworkSlug string) []ComplianceTrainingModule {
	return complianceModulesByFramework[frameworkSlug]
}

// ComplianceModuleSummary provides a lightweight view of a compliance module.
type ComplianceModuleSummary struct {
	Slug             string   `json:"slug"`
	FrameworkSlug    string   `json:"framework_slug"`
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	ControlRefs      []string `json:"control_refs"`
	DifficultyLevel  int      `json:"difficulty_level"`
	EstimatedMinutes int      `json:"estimated_minutes"`
	PageCount        int      `json:"page_count"`
	HasQuiz          bool     `json:"has_quiz"`
	QuestionCount    int      `json:"question_count"`
}

// GetComplianceModuleSummaries returns lightweight summaries of all compliance modules.
func GetComplianceModuleSummaries() []ComplianceModuleSummary {
	summaries := make([]ComplianceModuleSummary, len(BuiltInComplianceModules))
	for i, m := range BuiltInComplianceModules {
		qCount := 0
		if m.Quiz != nil {
			qCount = len(m.Quiz.Questions)
		}
		summaries[i] = ComplianceModuleSummary{
			Slug: m.Slug, FrameworkSlug: m.FrameworkSlug, Title: m.Title,
			Description: m.Description, ControlRefs: m.ControlRefs,
			DifficultyLevel: m.DifficultyLevel, EstimatedMinutes: m.EstimatedMinutes,
			PageCount: len(m.Pages), HasQuiz: m.Quiz != nil, QuestionCount: qCount,
		}
	}
	return summaries
}
