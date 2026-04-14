-- +goose Up
-- Phase 15: SMS phishing (smishing) simulation support

CREATE TABLE IF NOT EXISTS sms_providers (
    id INTEGER PRIMARY KEY AUTO_INCREMENT,
    user_id INTEGER NOT NULL,
    org_id INTEGER NOT NULL DEFAULT 0,
    name VARCHAR(255) NOT NULL,
    provider_type VARCHAR(50) NOT NULL DEFAULT 'twilio',
    account_sid VARCHAR(255) NOT NULL DEFAULT '',
    auth_token VARCHAR(255) NOT NULL DEFAULT '',
    from_number VARCHAR(50) NOT NULL DEFAULT '',
    modified_date DATETIME DEFAULT NOW()
);

CREATE INDEX idx_sms_providers_org ON sms_providers(org_id);

ALTER TABLE targets ADD COLUMN phone VARCHAR(50) DEFAULT '';
ALTER TABLE results ADD COLUMN phone VARCHAR(50) DEFAULT '';
ALTER TABLE email_requests ADD COLUMN phone VARCHAR(50) DEFAULT '';

ALTER TABLE campaigns ADD COLUMN campaign_type VARCHAR(10) DEFAULT 'email';
ALTER TABLE campaigns ADD COLUMN sms_provider_id INTEGER DEFAULT 0;

-- +goose Down
ALTER TABLE campaigns DROP COLUMN campaign_type;
ALTER TABLE campaigns DROP COLUMN sms_provider_id;
ALTER TABLE targets DROP COLUMN phone;
DROP TABLE IF EXISTS sms_providers;
