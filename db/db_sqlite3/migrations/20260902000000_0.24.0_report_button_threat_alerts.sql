-- +goose Up
-- Phase 13: Report Button & Threat Alerts

CREATE TABLE IF NOT EXISTS report_button_configs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id INTEGER NOT NULL,
    plugin_api_key VARCHAR(255) NOT NULL,
    feedback_simulation TEXT DEFAULT 'Good catch! This was a simulated phishing email.',
    feedback_real TEXT DEFAULT 'Thank you for reporting. Our security team will review this email.',
    enabled BOOLEAN DEFAULT 1,
    created_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    modified_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(org_id)
);

CREATE TABLE IF NOT EXISTS reported_emails (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id INTEGER NOT NULL,
    reporter_email VARCHAR(255) NOT NULL,
    sender_email VARCHAR(255) DEFAULT '',
    subject VARCHAR(500) DEFAULT '',
    headers_hash VARCHAR(64) DEFAULT '',
    is_simulation BOOLEAN DEFAULT 0,
    campaign_id INTEGER DEFAULT 0,
    result_id INTEGER DEFAULT 0,
    classification VARCHAR(50) DEFAULT 'pending',
    admin_notes TEXT DEFAULT '',
    created_date DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_reported_emails_org ON reported_emails(org_id);
CREATE INDEX IF NOT EXISTS idx_reported_emails_reporter ON reported_emails(reporter_email);

CREATE TABLE IF NOT EXISTS threat_alerts (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id INTEGER NOT NULL,
    title VARCHAR(500) NOT NULL,
    body TEXT NOT NULL,
    severity VARCHAR(20) DEFAULT 'info',
    target_roles TEXT DEFAULT '[]',
    target_departments TEXT DEFAULT '[]',
    published BOOLEAN DEFAULT 0,
    published_date DATETIME,
    created_by INTEGER NOT NULL,
    created_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    modified_date DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_threat_alerts_org ON threat_alerts(org_id);

CREATE TABLE IF NOT EXISTS threat_alert_reads (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    alert_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    read_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(alert_id, user_id)
);

-- +goose Down
DROP TABLE IF EXISTS threat_alert_reads;
DROP TABLE IF EXISTS threat_alerts;
DROP TABLE IF EXISTS reported_emails;
DROP TABLE IF EXISTS report_button_configs;
