-- +goose Up
-- Advanced Email Security: Inbox Analysis, BEC Detection, Graymail,
-- One-Click Remediation, Phishing Ticket Elimination, Enhanced Report Button

-- =========================================================================
-- Google Workspace integration for report button (extends report_button_configs)
-- =========================================================================
ALTER TABLE report_button_configs ADD COLUMN outlook_enabled BOOLEAN DEFAULT 1;
ALTER TABLE report_button_configs ADD COLUMN google_enabled BOOLEAN DEFAULT 0;
ALTER TABLE report_button_configs ADD COLUMN google_workspace_domain VARCHAR(255) DEFAULT '';
ALTER TABLE report_button_configs ADD COLUMN google_service_account_json TEXT DEFAULT '';
ALTER TABLE report_button_configs ADD COLUMN auto_analyze BOOLEAN DEFAULT 1;
ALTER TABLE report_button_configs ADD COLUMN auto_remediate_threshold VARCHAR(30) DEFAULT 'confirmed_phishing';

-- Add raw email body storage to reported emails for deeper analysis
ALTER TABLE reported_emails ADD COLUMN raw_headers TEXT DEFAULT '';
ALTER TABLE reported_emails ADD COLUMN raw_body TEXT DEFAULT '';
ALTER TABLE reported_emails ADD COLUMN source_platform VARCHAR(20) DEFAULT 'outlook';
ALTER TABLE reported_emails ADD COLUMN auto_analyzed BOOLEAN DEFAULT 0;
ALTER TABLE reported_emails ADD COLUMN remediated BOOLEAN DEFAULT 0;
ALTER TABLE reported_emails ADD COLUMN remediated_date DATETIME;

-- =========================================================================
-- Real-time Inbox Threat Monitoring (AI inbox analysis)
-- =========================================================================
CREATE TABLE IF NOT EXISTS inbox_monitor_configs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id INTEGER NOT NULL UNIQUE,
    enabled BOOLEAN DEFAULT 0,
    scan_interval_seconds INTEGER DEFAULT 300,
    monitored_mailboxes TEXT DEFAULT '[]',
    threat_threshold VARCHAR(30) DEFAULT 'suspicious',
    auto_quarantine BOOLEAN DEFAULT 0,
    auto_delete BOOLEAN DEFAULT 0,
    notify_admin BOOLEAN DEFAULT 1,
    notify_user BOOLEAN DEFAULT 0,
    imap_host VARCHAR(255) DEFAULT '',
    imap_port INTEGER DEFAULT 993,
    imap_username VARCHAR(255) DEFAULT '',
    imap_password VARCHAR(255) DEFAULT '',
    imap_tls BOOLEAN DEFAULT 1,
    google_workspace_enabled BOOLEAN DEFAULT 0,
    google_admin_email VARCHAR(255) DEFAULT '',
    ms365_enabled BOOLEAN DEFAULT 0,
    ms365_tenant_id VARCHAR(100) DEFAULT '',
    ms365_client_id VARCHAR(100) DEFAULT '',
    ms365_client_secret VARCHAR(255) DEFAULT '',
    last_scan_date DATETIME,
    created_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    modified_date DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS inbox_scan_results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id INTEGER NOT NULL,
    config_id INTEGER NOT NULL,
    mailbox_email VARCHAR(255) NOT NULL,
    message_id VARCHAR(500) DEFAULT '',
    sender_email VARCHAR(255) DEFAULT '',
    subject VARCHAR(500) DEFAULT '',
    received_date DATETIME,
    threat_level VARCHAR(30) DEFAULT 'safe',
    classification VARCHAR(50) DEFAULT 'unknown',
    confidence_score REAL DEFAULT 0,
    is_bec BOOLEAN DEFAULT 0,
    is_graymail BOOLEAN DEFAULT 0,
    graymail_category VARCHAR(50) DEFAULT '',
    summary TEXT DEFAULT '',
    indicators TEXT DEFAULT '[]',
    action_taken VARCHAR(50) DEFAULT 'none',
    scan_duration_ms INTEGER DEFAULT 0,
    created_date DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_inbox_scan_org ON inbox_scan_results(org_id);
CREATE INDEX IF NOT EXISTS idx_inbox_scan_threat ON inbox_scan_results(org_id, threat_level);
CREATE INDEX IF NOT EXISTS idx_inbox_scan_bec ON inbox_scan_results(org_id, is_bec);
CREATE INDEX IF NOT EXISTS idx_inbox_scan_graymail ON inbox_scan_results(org_id, is_graymail);
CREATE INDEX IF NOT EXISTS idx_inbox_scan_date ON inbox_scan_results(created_date);

-- =========================================================================
-- BEC (Business Email Compromise) Detection
-- =========================================================================
CREATE TABLE IF NOT EXISTS bec_profiles (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id INTEGER NOT NULL,
    executive_email VARCHAR(255) NOT NULL,
    executive_name VARCHAR(255) DEFAULT '',
    title VARCHAR(255) DEFAULT '',
    department VARCHAR(255) DEFAULT '',
    known_domains TEXT DEFAULT '[]',
    known_senders TEXT DEFAULT '[]',
    is_active BOOLEAN DEFAULT 1,
    created_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    modified_date DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_bec_profiles_org ON bec_profiles(org_id);
CREATE INDEX IF NOT EXISTS idx_bec_profiles_email ON bec_profiles(executive_email);

CREATE TABLE IF NOT EXISTS bec_detections (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id INTEGER NOT NULL,
    scan_result_id INTEGER DEFAULT 0,
    reported_email_id INTEGER DEFAULT 0,
    impersonated_email VARCHAR(255) DEFAULT '',
    impersonated_name VARCHAR(255) DEFAULT '',
    actual_sender VARCHAR(255) DEFAULT '',
    attack_type VARCHAR(50) DEFAULT '',
    urgency_level VARCHAR(20) DEFAULT 'medium',
    financial_request BOOLEAN DEFAULT 0,
    wire_transfer_mentioned BOOLEAN DEFAULT 0,
    gift_card_mentioned BOOLEAN DEFAULT 0,
    confidence_score REAL DEFAULT 0,
    summary TEXT DEFAULT '',
    action_taken VARCHAR(50) DEFAULT 'flagged',
    resolved BOOLEAN DEFAULT 0,
    resolved_by INTEGER DEFAULT 0,
    resolved_date DATETIME,
    created_date DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_bec_detections_org ON bec_detections(org_id);
CREATE INDEX IF NOT EXISTS idx_bec_detections_resolved ON bec_detections(org_id, resolved);

-- =========================================================================
-- Graymail Classification
-- =========================================================================
CREATE TABLE IF NOT EXISTS graymail_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id INTEGER NOT NULL,
    rule_type VARCHAR(50) NOT NULL,
    pattern VARCHAR(500) NOT NULL,
    category VARCHAR(50) NOT NULL,
    action VARCHAR(30) DEFAULT 'label',
    is_active BOOLEAN DEFAULT 1,
    match_count INTEGER DEFAULT 0,
    created_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    modified_date DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_graymail_rules_org ON graymail_rules(org_id);

CREATE TABLE IF NOT EXISTS graymail_classifications (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id INTEGER NOT NULL,
    scan_result_id INTEGER DEFAULT 0,
    email_subject VARCHAR(500) DEFAULT '',
    sender_email VARCHAR(255) DEFAULT '',
    category VARCHAR(50) NOT NULL,
    subcategory VARCHAR(100) DEFAULT '',
    confidence_score REAL DEFAULT 0,
    action_taken VARCHAR(30) DEFAULT 'labeled',
    created_date DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_graymail_class_org ON graymail_classifications(org_id);
CREATE INDEX IF NOT EXISTS idx_graymail_class_cat ON graymail_classifications(org_id, category);

-- =========================================================================
-- One-Click Remediation Actions
-- =========================================================================
CREATE TABLE IF NOT EXISTS remediation_actions (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id INTEGER NOT NULL,
    action_type VARCHAR(50) NOT NULL,
    target_type VARCHAR(50) NOT NULL,
    target_id INTEGER DEFAULT 0,
    target_email VARCHAR(255) DEFAULT '',
    message_id VARCHAR(500) DEFAULT '',
    subject VARCHAR(500) DEFAULT '',
    sender_email VARCHAR(255) DEFAULT '',
    status VARCHAR(30) DEFAULT 'pending',
    result_message TEXT DEFAULT '',
    initiated_by INTEGER NOT NULL,
    approved_by INTEGER DEFAULT 0,
    requires_approval BOOLEAN DEFAULT 0,
    scope VARCHAR(30) DEFAULT 'single',
    affected_mailboxes INTEGER DEFAULT 0,
    created_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_date DATETIME
);

CREATE INDEX IF NOT EXISTS idx_remediation_actions_org ON remediation_actions(org_id);
CREATE INDEX IF NOT EXISTS idx_remediation_actions_status ON remediation_actions(org_id, status);

-- =========================================================================
-- Phishing Ticket Management (automated ticket elimination)
-- =========================================================================
CREATE TABLE IF NOT EXISTS phishing_tickets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id INTEGER NOT NULL,
    reported_email_id INTEGER DEFAULT 0,
    scan_result_id INTEGER DEFAULT 0,
    bec_detection_id INTEGER DEFAULT 0,
    ticket_number VARCHAR(50) NOT NULL,
    title VARCHAR(500) NOT NULL,
    description TEXT DEFAULT '',
    severity VARCHAR(20) DEFAULT 'medium',
    status VARCHAR(30) DEFAULT 'open',
    classification VARCHAR(50) DEFAULT 'pending',
    auto_resolved BOOLEAN DEFAULT 0,
    auto_resolution_reason TEXT DEFAULT '',
    assigned_to INTEGER DEFAULT 0,
    escalated BOOLEAN DEFAULT 0,
    escalated_to INTEGER DEFAULT 0,
    resolution_notes TEXT DEFAULT '',
    sla_deadline DATETIME,
    created_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    resolved_date DATETIME
);

CREATE INDEX IF NOT EXISTS idx_phishing_tickets_org ON phishing_tickets(org_id);
CREATE INDEX IF NOT EXISTS idx_phishing_tickets_status ON phishing_tickets(org_id, status);
CREATE INDEX IF NOT EXISTS idx_phishing_tickets_number ON phishing_tickets(ticket_number);

CREATE TABLE IF NOT EXISTS phishing_ticket_auto_rules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id INTEGER NOT NULL,
    rule_name VARCHAR(255) NOT NULL,
    condition_type VARCHAR(50) NOT NULL,
    condition_value VARCHAR(500) NOT NULL,
    action VARCHAR(50) NOT NULL,
    is_active BOOLEAN DEFAULT 1,
    triggers_count INTEGER DEFAULT 0,
    created_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    modified_date DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_ticket_auto_rules_org ON phishing_ticket_auto_rules(org_id);

-- =========================================================================
-- ZIM Sandbox Enhancement: Real-time inbox isolation
-- =========================================================================
ALTER TABLE sandbox_tests ADD COLUMN isolation_mode VARCHAR(30) DEFAULT 'preview';
ALTER TABLE sandbox_tests ADD COLUMN quarantine_enabled BOOLEAN DEFAULT 0;
ALTER TABLE sandbox_tests ADD COLUMN inbox_scan_result_id INTEGER DEFAULT 0;
ALTER TABLE sandbox_tests ADD COLUMN detonation_results TEXT DEFAULT '';

-- Email analysis enhancement for automated pipeline
-- Ensure the table exists (may have been created by AutoMigrate only)
CREATE TABLE IF NOT EXISTS email_analyses (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id INTEGER NOT NULL DEFAULT 0,
    reported_email_id INTEGER DEFAULT 0,
    status VARCHAR(20) DEFAULT 'pending',
    threat_level VARCHAR(30) DEFAULT '',
    confidence_score REAL DEFAULT 0,
    classification VARCHAR(50) DEFAULT '',
    summary TEXT DEFAULT '',
    ai_provider VARCHAR(30) DEFAULT '',
    tokens_used INTEGER DEFAULT 0,
    analysis_duration INTEGER DEFAULT 0,
    created_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_date DATETIME
);
ALTER TABLE email_analyses ADD COLUMN is_automated BOOLEAN DEFAULT 0;
ALTER TABLE email_analyses ADD COLUMN source VARCHAR(50) DEFAULT 'manual';
ALTER TABLE email_analyses ADD COLUMN bec_detected BOOLEAN DEFAULT 0;
ALTER TABLE email_analyses ADD COLUMN graymail_detected BOOLEAN DEFAULT 0;
ALTER TABLE email_analyses ADD COLUMN graymail_category VARCHAR(50) DEFAULT '';
ALTER TABLE email_analyses ADD COLUMN remediation_action_id INTEGER DEFAULT 0;
ALTER TABLE email_analyses ADD COLUMN ticket_id INTEGER DEFAULT 0;

-- +goose Down
ALTER TABLE email_analyses DROP COLUMN is_automated;
ALTER TABLE email_analyses DROP COLUMN source;
ALTER TABLE email_analyses DROP COLUMN bec_detected;
ALTER TABLE email_analyses DROP COLUMN graymail_detected;
ALTER TABLE email_analyses DROP COLUMN graymail_category;
ALTER TABLE email_analyses DROP COLUMN remediation_action_id;
ALTER TABLE email_analyses DROP COLUMN ticket_id;

ALTER TABLE sandbox_tests DROP COLUMN isolation_mode;
ALTER TABLE sandbox_tests DROP COLUMN quarantine_enabled;
ALTER TABLE sandbox_tests DROP COLUMN inbox_scan_result_id;
ALTER TABLE sandbox_tests DROP COLUMN detonation_results;

ALTER TABLE reported_emails DROP COLUMN raw_headers;
ALTER TABLE reported_emails DROP COLUMN raw_body;
ALTER TABLE reported_emails DROP COLUMN source_platform;
ALTER TABLE reported_emails DROP COLUMN auto_analyzed;
ALTER TABLE reported_emails DROP COLUMN remediated;
ALTER TABLE reported_emails DROP COLUMN remediated_date;

ALTER TABLE report_button_configs DROP COLUMN outlook_enabled;
ALTER TABLE report_button_configs DROP COLUMN google_enabled;
ALTER TABLE report_button_configs DROP COLUMN google_workspace_domain;
ALTER TABLE report_button_configs DROP COLUMN google_service_account_json;
ALTER TABLE report_button_configs DROP COLUMN auto_analyze;
ALTER TABLE report_button_configs DROP COLUMN auto_remediate_threshold;

DROP TABLE IF EXISTS phishing_ticket_auto_rules;
DROP TABLE IF EXISTS phishing_tickets;
DROP TABLE IF EXISTS remediation_actions;
DROP TABLE IF EXISTS graymail_classifications;
DROP TABLE IF EXISTS graymail_rules;
DROP TABLE IF EXISTS bec_detections;
DROP TABLE IF EXISTS bec_profiles;
DROP TABLE IF EXISTS inbox_scan_results;
DROP TABLE IF EXISTS inbox_monitor_configs;
