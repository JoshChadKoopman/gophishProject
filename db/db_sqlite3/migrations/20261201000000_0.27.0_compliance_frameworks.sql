-- +goose Up
-- Compliance Framework Mapping tables

CREATE TABLE IF NOT EXISTS compliance_frameworks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    slug VARCHAR(50) NOT NULL UNIQUE,
    name VARCHAR(100) NOT NULL,
    version VARCHAR(20) DEFAULT '',
    description TEXT DEFAULT '',
    region VARCHAR(50) DEFAULT '',
    is_active BOOLEAN DEFAULT 1,
    created_date DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS compliance_controls (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    framework_id INTEGER NOT NULL,
    control_ref VARCHAR(50) NOT NULL,
    title VARCHAR(255) NOT NULL,
    description TEXT DEFAULT '',
    category VARCHAR(100) DEFAULT '',
    evidence_type VARCHAR(50) DEFAULT 'manual',
    evidence_criteria TEXT DEFAULT '{}',
    sort_order INTEGER DEFAULT 0,
    FOREIGN KEY (framework_id) REFERENCES compliance_frameworks(id)
);

CREATE TABLE IF NOT EXISTS org_compliance_mappings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id INTEGER NOT NULL,
    framework_id INTEGER NOT NULL,
    enabled_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT 1,
    UNIQUE(org_id, framework_id)
);

CREATE TABLE IF NOT EXISTS compliance_assessments (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id INTEGER NOT NULL,
    framework_id INTEGER NOT NULL,
    control_id INTEGER NOT NULL,
    status VARCHAR(20) DEFAULT 'not_assessed',
    score REAL DEFAULT 0,
    evidence TEXT DEFAULT '',
    assessed_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    assessed_by INTEGER DEFAULT 0,
    notes TEXT DEFAULT '',
    FOREIGN KEY (control_id) REFERENCES compliance_controls(id)
);

CREATE INDEX IF NOT EXISTS idx_compliance_assessments_org ON compliance_assessments(org_id, framework_id);
CREATE INDEX IF NOT EXISTS idx_compliance_assessments_control ON compliance_assessments(control_id, assessed_date);
CREATE INDEX IF NOT EXISTS idx_org_compliance_mappings_org ON org_compliance_mappings(org_id);

-- Seed: NIS2 (EU Network and Information Security Directive 2)
INSERT INTO compliance_frameworks (slug, name, version, description, region) VALUES
('nis2', 'NIS2', '2022/2555', 'EU Network and Information Security Directive 2 — cybersecurity risk management and incident reporting for essential and important entities.', 'EU');

INSERT INTO compliance_controls (framework_id, control_ref, title, description, category, evidence_type, evidence_criteria, sort_order) VALUES
((SELECT id FROM compliance_frameworks WHERE slug='nis2'), 'Art.21(2)(g)', 'Cybersecurity awareness training', 'Regular cybersecurity awareness training for all staff including management.', 'Human Resources Security', 'training_rate', '{"metric":"completion_rate","threshold":90,"operator":"gte"}', 1),
((SELECT id FROM compliance_frameworks WHERE slug='nis2'), 'Art.21(2)(a)', 'Risk analysis and information system security policies', 'Policies for risk analysis and information system security.', 'Risk Management', 'brs_score', '{"metric":"org_avg_brs","threshold":60,"operator":"gte"}', 2),
((SELECT id FROM compliance_frameworks WHERE slug='nis2'), 'Art.21(2)(b)', 'Incident handling', 'Procedures for handling and reporting security incidents.', 'Incident Management', 'report_rate', '{"metric":"report_rate","threshold":30,"operator":"gte"}', 3),
((SELECT id FROM compliance_frameworks WHERE slug='nis2'), 'Art.21(2)(d)', 'Supply chain security', 'Security aspects relating to supplier relationships.', 'Supply Chain', 'manual', '{}', 4),
((SELECT id FROM compliance_frameworks WHERE slug='nis2'), 'Art.21(2)(e)', 'Vulnerability handling and disclosure', 'Policies and procedures for vulnerability handling.', 'Vulnerability Management', 'manual', '{}', 5),
((SELECT id FROM compliance_frameworks WHERE slug='nis2'), 'Art.21(2)(f)', 'Assessment of cybersecurity risk-management measures', 'Policies and procedures to assess the effectiveness of measures.', 'Governance', 'simulation_rate', '{"metric":"campaigns_run","threshold":4,"operator":"gte"}', 6),
((SELECT id FROM compliance_frameworks WHERE slug='nis2'), 'Art.21(2)(h)', 'Encryption policies', 'Policies and procedures regarding use of cryptography and encryption.', 'Cryptography', 'manual', '{}', 7),
((SELECT id FROM compliance_frameworks WHERE slug='nis2'), 'Art.21(2)(i)', 'Access control and asset management', 'Human resources security, access control policies and asset management.', 'Access Control', 'manual', '{}', 8),
((SELECT id FROM compliance_frameworks WHERE slug='nis2'), 'Art.21(2)(j)', 'Multi-factor authentication', 'Use of MFA or continuous authentication solutions.', 'Authentication', 'manual', '{}', 9),
((SELECT id FROM compliance_frameworks WHERE slug='nis2'), 'Art.23', 'Incident notification obligations', 'Early warning within 24h, incident notification within 72h, final report within 1 month.', 'Incident Reporting', 'manual', '{}', 10);

-- Seed: DORA (Digital Operational Resilience Act)
INSERT INTO compliance_frameworks (slug, name, version, description, region) VALUES
('dora', 'DORA', '2022/2554', 'EU Digital Operational Resilience Act — ICT risk management for financial entities.', 'EU');

INSERT INTO compliance_controls (framework_id, control_ref, title, description, category, evidence_type, evidence_criteria, sort_order) VALUES
((SELECT id FROM compliance_frameworks WHERE slug='dora'), 'Art.13(6)', 'ICT security awareness programmes', 'Financial entities shall develop ICT security awareness programmes and digital operational resilience training as compulsory modules.', 'Training & Awareness', 'training_rate', '{"metric":"completion_rate","threshold":95,"operator":"gte"}', 1),
((SELECT id FROM compliance_frameworks WHERE slug='dora'), 'Art.13(6)', 'Phishing simulation programmes', 'Programmes shall include phishing simulation exercises.', 'Training & Awareness', 'simulation_rate', '{"metric":"campaigns_run","threshold":6,"operator":"gte"}', 2),
((SELECT id FROM compliance_frameworks WHERE slug='dora'), 'Art.5(2)', 'ICT risk management framework', 'Appropriate ICT risk management framework in place and regularly reviewed.', 'Risk Management', 'brs_score', '{"metric":"org_avg_brs","threshold":65,"operator":"gte"}', 3),
((SELECT id FROM compliance_frameworks WHERE slug='dora'), 'Art.6', 'ICT systems identification and classification', 'Identify, classify and document all ICT supported business functions and assets.', 'Asset Management', 'manual', '{}', 4),
((SELECT id FROM compliance_frameworks WHERE slug='dora'), 'Art.9(1)', 'Protection and prevention', 'Policies covering ICT security and resilience to protect against ICT risks.', 'Protection', 'manual', '{}', 5),
((SELECT id FROM compliance_frameworks WHERE slug='dora'), 'Art.10', 'Detection of anomalous activities', 'Mechanisms to promptly detect anomalous activities.', 'Detection', 'manual', '{}', 6),
((SELECT id FROM compliance_frameworks WHERE slug='dora'), 'Art.11', 'Response and recovery', 'Comprehensive ICT business continuity policy.', 'Response', 'manual', '{}', 7),
((SELECT id FROM compliance_frameworks WHERE slug='dora'), 'Art.17', 'Incident reporting', 'Classify and report major ICT-related incidents.', 'Incident Reporting', 'report_rate', '{"metric":"report_rate","threshold":40,"operator":"gte"}', 8),
((SELECT id FROM compliance_frameworks WHERE slug='dora'), 'Art.24-27', 'Threat-led penetration testing (TLPT)', 'Advanced testing of ICT tools, systems and processes based on TLPT.', 'Testing', 'simulation_rate', '{"metric":"click_rate","threshold":15,"operator":"lte"}', 9),
((SELECT id FROM compliance_frameworks WHERE slug='dora'), 'Art.28-30', 'Third-party risk management', 'Manage ICT third-party provider risk.', 'Third-Party Risk', 'manual', '{}', 10);

-- Seed: HIPAA (Health Insurance Portability and Accountability Act)
INSERT INTO compliance_frameworks (slug, name, version, description, region) VALUES
('hipaa', 'HIPAA', 'Security Rule', 'US Health Insurance Portability and Accountability Act — safeguards for electronic protected health information (ePHI).', 'US');

INSERT INTO compliance_controls (framework_id, control_ref, title, description, category, evidence_type, evidence_criteria, sort_order) VALUES
((SELECT id FROM compliance_frameworks WHERE slug='hipaa'), '§164.308(a)(5)', 'Security awareness and training', 'Implement a security awareness and training program for all workforce members.', 'Administrative Safeguards', 'training_rate', '{"metric":"completion_rate","threshold":100,"operator":"gte"}', 1),
((SELECT id FROM compliance_frameworks WHERE slug='hipaa'), '§164.308(a)(5)(ii)(A)', 'Security reminders', 'Periodic security updates and reminders.', 'Administrative Safeguards', 'simulation_rate', '{"metric":"campaigns_run","threshold":4,"operator":"gte"}', 2),
((SELECT id FROM compliance_frameworks WHERE slug='hipaa'), '§164.308(a)(5)(ii)(B)', 'Protection from malicious software', 'Procedures for guarding against and detecting malicious software.', 'Administrative Safeguards', 'simulation_rate', '{"metric":"click_rate","threshold":10,"operator":"lte"}', 3),
((SELECT id FROM compliance_frameworks WHERE slug='hipaa'), '§164.308(a)(5)(ii)(C)', 'Log-in monitoring', 'Procedures for monitoring log-in attempts and reporting discrepancies.', 'Administrative Safeguards', 'manual', '{}', 4),
((SELECT id FROM compliance_frameworks WHERE slug='hipaa'), '§164.308(a)(5)(ii)(D)', 'Password management', 'Procedures for creating, changing, and safeguarding passwords.', 'Administrative Safeguards', 'manual', '{}', 5),
((SELECT id FROM compliance_frameworks WHERE slug='hipaa'), '§164.308(a)(1)', 'Risk analysis', 'Conduct an accurate and thorough assessment of potential risks and vulnerabilities.', 'Administrative Safeguards', 'brs_score', '{"metric":"org_avg_brs","threshold":55,"operator":"gte"}', 6),
((SELECT id FROM compliance_frameworks WHERE slug='hipaa'), '§164.308(a)(6)', 'Security incident procedures', 'Implement policies and procedures to address security incidents.', 'Administrative Safeguards', 'report_rate', '{"metric":"report_rate","threshold":25,"operator":"gte"}', 7),
((SELECT id FROM compliance_frameworks WHERE slug='hipaa'), '§164.308(a)(8)', 'Evaluation', 'Periodic technical and nontechnical evaluations.', 'Administrative Safeguards', 'simulation_rate', '{"metric":"campaigns_run","threshold":2,"operator":"gte"}', 8),
((SELECT id FROM compliance_frameworks WHERE slug='hipaa'), '§164.312(a)(1)', 'Access control', 'Implement technical policies to allow access only to authorized persons.', 'Technical Safeguards', 'manual', '{}', 9),
((SELECT id FROM compliance_frameworks WHERE slug='hipaa'), '§164.312(c)(1)', 'Integrity controls', 'Policies and procedures to protect ePHI from improper alteration or destruction.', 'Technical Safeguards', 'manual', '{}', 10);

-- Seed: PCI DSS (Payment Card Industry Data Security Standard)
INSERT INTO compliance_frameworks (slug, name, version, description, region) VALUES
('pci_dss', 'PCI DSS', 'v4.0', 'Payment Card Industry Data Security Standard — security requirements for organizations handling cardholder data.', 'Global');

INSERT INTO compliance_controls (framework_id, control_ref, title, description, category, evidence_type, evidence_criteria, sort_order) VALUES
((SELECT id FROM compliance_frameworks WHERE slug='pci_dss'), 'Req 12.6.1', 'Security awareness program', 'Implement a formal security awareness program to educate all personnel.', 'Security Awareness', 'training_rate', '{"metric":"completion_rate","threshold":100,"operator":"gte"}', 1),
((SELECT id FROM compliance_frameworks WHERE slug='pci_dss'), 'Req 12.6.2', 'Annual security awareness training', 'Personnel acknowledge at least annually that they read and understand the security policy.', 'Security Awareness', 'certification', '{"metric":"cert_rate","threshold":95,"operator":"gte"}', 2),
((SELECT id FROM compliance_frameworks WHERE slug='pci_dss'), 'Req 12.6.3.1', 'Phishing awareness training', 'Security awareness training includes awareness of phishing and social engineering.', 'Security Awareness', 'simulation_rate', '{"metric":"campaigns_run","threshold":4,"operator":"gte"}', 3),
((SELECT id FROM compliance_frameworks WHERE slug='pci_dss'), 'Req 12.6.3.2', 'Phishing simulation exercises', 'Phishing simulations conducted at least quarterly.', 'Security Awareness', 'simulation_rate', '{"metric":"click_rate","threshold":10,"operator":"lte"}', 4),
((SELECT id FROM compliance_frameworks WHERE slug='pci_dss'), 'Req 5.4.1', 'Anti-phishing mechanisms', 'Processes and mechanisms to detect and protect against phishing.', 'Anti-Phishing', 'report_rate', '{"metric":"report_rate","threshold":30,"operator":"gte"}', 5),
((SELECT id FROM compliance_frameworks WHERE slug='pci_dss'), 'Req 6.3', 'Security vulnerabilities identified and addressed', 'Security vulnerabilities are identified and addressed in a timely manner.', 'Vulnerability Management', 'manual', '{}', 6),
((SELECT id FROM compliance_frameworks WHERE slug='pci_dss'), 'Req 8.3.6', 'Password/passphrase complexity', 'Passwords/passphrases meet minimum complexity.', 'Authentication', 'manual', '{}', 7),
((SELECT id FROM compliance_frameworks WHERE slug='pci_dss'), 'Req 8.4.2', 'MFA for CDE access', 'MFA for all access into the cardholder data environment.', 'Authentication', 'manual', '{}', 8),
((SELECT id FROM compliance_frameworks WHERE slug='pci_dss'), 'Req 10.2', 'Audit logs', 'Audit logs are enabled and active for all system components.', 'Logging & Monitoring', 'manual', '{}', 9),
((SELECT id FROM compliance_frameworks WHERE slug='pci_dss'), 'Req 11.4', 'Penetration testing', 'External and internal penetration testing regularly performed.', 'Testing', 'manual', '{}', 10);

-- Seed: NIST CSF (Cybersecurity Framework)
INSERT INTO compliance_frameworks (slug, name, version, description, region) VALUES
('nist_csf', 'NIST CSF', '2.0', 'NIST Cybersecurity Framework — voluntary framework of standards and best practices to manage cybersecurity risk.', 'US / Global');

INSERT INTO compliance_controls (framework_id, control_ref, title, description, category, evidence_type, evidence_criteria, sort_order) VALUES
((SELECT id FROM compliance_frameworks WHERE slug='nist_csf'), 'GV.AT-01', 'Awareness and training', 'Personnel are provided awareness and training so that they possess the knowledge and skills to perform tasks with cybersecurity risks in mind.', 'Govern', 'training_rate', '{"metric":"completion_rate","threshold":90,"operator":"gte"}', 1),
((SELECT id FROM compliance_frameworks WHERE slug='nist_csf'), 'GV.AT-02', 'Role-based security training', 'Individuals in specialized roles are provided with awareness and training.', 'Govern', 'quiz_pass_rate', '{"metric":"quiz_pass_rate","threshold":80,"operator":"gte"}', 2),
((SELECT id FROM compliance_frameworks WHERE slug='nist_csf'), 'GV.RM-01', 'Risk management strategy', 'Organizational cybersecurity risk management strategy is established and communicated.', 'Govern', 'brs_score', '{"metric":"org_avg_brs","threshold":60,"operator":"gte"}', 3),
((SELECT id FROM compliance_frameworks WHERE slug='nist_csf'), 'ID.RA-01', 'Asset vulnerabilities identified', 'Vulnerabilities in assets are identified, validated, and recorded.', 'Identify', 'simulation_rate', '{"metric":"campaigns_run","threshold":4,"operator":"gte"}', 4),
((SELECT id FROM compliance_frameworks WHERE slug='nist_csf'), 'PR.AT-01', 'Awareness and training for all users', 'All users are informed and trained on cybersecurity.', 'Protect', 'training_rate', '{"metric":"completion_rate","threshold":95,"operator":"gte"}', 5),
((SELECT id FROM compliance_frameworks WHERE slug='nist_csf'), 'PR.AT-02', 'Phishing-resistant awareness', 'Users understand risks of phishing and social engineering.', 'Protect', 'simulation_rate', '{"metric":"click_rate","threshold":15,"operator":"lte"}', 6),
((SELECT id FROM compliance_frameworks WHERE slug='nist_csf'), 'DE.CM-01', 'Networks and environments monitored', 'Networks and network services are monitored to find potentially adverse events.', 'Detect', 'manual', '{}', 7),
((SELECT id FROM compliance_frameworks WHERE slug='nist_csf'), 'DE.AE-02', 'Anomalous activity detected and analyzed', 'Potentially adverse events are analyzed to better understand associated activities.', 'Detect', 'report_rate', '{"metric":"report_rate","threshold":25,"operator":"gte"}', 8),
((SELECT id FROM compliance_frameworks WHERE slug='nist_csf'), 'RS.AN-03', 'Incident analysis and triage', 'Analysis is performed to determine what has taken place during an incident.', 'Respond', 'manual', '{}', 9),
((SELECT id FROM compliance_frameworks WHERE slug='nist_csf'), 'RC.RP-01', 'Recovery plan executed', 'The recovery portion of the incident response plan is executed.', 'Recover', 'manual', '{}', 10);

-- Add compliance_mapping feature slug to tier features
INSERT OR IGNORE INTO tier_features (tier_id, feature_slug)
SELECT id, 'compliance_mapping' FROM subscription_tiers WHERE slug IN ('advanced', 'all_in_one', 'enterprise');

-- +goose Down
DROP TABLE IF EXISTS compliance_assessments;
DROP TABLE IF EXISTS org_compliance_mappings;
DROP TABLE IF EXISTS compliance_controls;
DROP TABLE IF EXISTS compliance_frameworks;
DELETE FROM tier_features WHERE feature_slug = 'compliance_mapping';
