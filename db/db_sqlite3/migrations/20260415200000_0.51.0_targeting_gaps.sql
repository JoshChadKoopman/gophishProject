-- +goose Up
-- Targeting Engine Gaps: A/B testing, send-time optimization, department threats, feedback loop

-- Add variant_id to results for A/B test tracking
ALTER TABLE results ADD COLUMN variant_id VARCHAR(8) NOT NULL DEFAULT '';

-- variant_id for autopilot_schedules is handled in a later migration
-- (autopilot_schedules is created by 20260715000000_0.21.0_autopilot.sql)

-- A/B Test Results table (if not already created by send_time_optimizer)
CREATE TABLE IF NOT EXISTS ab_test_results (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id          INTEGER NOT NULL DEFAULT 0,
    campaign_id     INTEGER NOT NULL,
    user_id         INTEGER NOT NULL,
    email           VARCHAR(255) NOT NULL DEFAULT '',
    variant_id      VARCHAR(8) NOT NULL DEFAULT 'A',
    template_id     INTEGER NOT NULL DEFAULT 0,
    template_name   VARCHAR(255) NOT NULL DEFAULT '',
    clicked         BOOLEAN NOT NULL DEFAULT 0,
    submitted       BOOLEAN NOT NULL DEFAULT 0,
    reported        BOOLEAN NOT NULL DEFAULT 0,
    time_to_click_s INTEGER NOT NULL DEFAULT 0,
    created_date    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_abtest_campaign ON ab_test_results(campaign_id);
CREATE INDEX IF NOT EXISTS idx_abtest_variant ON ab_test_results(campaign_id, variant_id);

-- User email feedback table (if not already created)
CREATE TABLE IF NOT EXISTS user_email_feedback (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id            INTEGER NOT NULL DEFAULT 0,
    user_id           INTEGER NOT NULL,
    email             VARCHAR(255) NOT NULL DEFAULT '',
    message_id        VARCHAR(512) NOT NULL DEFAULT '',
    subject           VARCHAR(512) NOT NULL DEFAULT '',
    sender_email      VARCHAR(255) NOT NULL DEFAULT '',
    threat_level      VARCHAR(32) NOT NULL DEFAULT 'safe',
    confidence_score  REAL NOT NULL DEFAULT 0,
    summary           TEXT NOT NULL DEFAULT '',
    indicators        TEXT NOT NULL DEFAULT '[]',
    recommendation    TEXT NOT NULL DEFAULT '',
    was_simulation    BOOLEAN NOT NULL DEFAULT 0,
    simulation_result VARCHAR(32) NOT NULL DEFAULT '',
    learning_tip      TEXT NOT NULL DEFAULT '',
    feedback_read     BOOLEAN NOT NULL DEFAULT 0,
    user_acknowledged BOOLEAN NOT NULL DEFAULT 0,
    created_date      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_date     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_uef_user ON user_email_feedback(user_id);
CREATE INDEX IF NOT EXISTS idx_uef_unread ON user_email_feedback(user_id, feedback_read);

-- AI Classification Feedback table (false positive loop)
CREATE TABLE IF NOT EXISTS ai_classification_feedback (
    id                         INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id                     INTEGER NOT NULL DEFAULT 0,
    scan_result_id             INTEGER NOT NULL DEFAULT 0,
    reported_email_id          INTEGER NOT NULL DEFAULT 0,
    original_threat_level      VARCHAR(32) NOT NULL DEFAULT '',
    corrected_threat_level     VARCHAR(32) NOT NULL DEFAULT '',
    original_classification    VARCHAR(64) NOT NULL DEFAULT '',
    corrected_classification   VARCHAR(64) NOT NULL DEFAULT '',
    feedback_type              VARCHAR(32) NOT NULL DEFAULT '',
    admin_notes                TEXT NOT NULL DEFAULT '',
    admin_user_id              INTEGER NOT NULL DEFAULT 0,
    created_date               DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_aicf_org ON ai_classification_feedback(org_id);
CREATE INDEX IF NOT EXISTS idx_aicf_type ON ai_classification_feedback(org_id, feedback_type);

-- Inbox Webhook Configs table (Microsoft Graph / Gmail push notifications)
CREATE TABLE IF NOT EXISTS inbox_webhook_configs (
    id                   INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id               INTEGER NOT NULL DEFAULT 0,
    provider             VARCHAR(32) NOT NULL DEFAULT 'microsoft_graph',
    enabled              BOOLEAN NOT NULL DEFAULT 0,
    subscription_id      VARCHAR(512) NOT NULL DEFAULT '',
    webhook_url          VARCHAR(1024) NOT NULL DEFAULT '',
    expiration_date      DATETIME NULL,
    pubsub_topic         VARCHAR(512) NOT NULL DEFAULT '',
    pubsub_subscription  VARCHAR(512) NOT NULL DEFAULT '',
    history_id           VARCHAR(128) NOT NULL DEFAULT '',
    last_notification    DATETIME NULL,
    created_date         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_date        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_iwc_org ON inbox_webhook_configs(org_id, provider);

SELECT 1; -- placeholder: admin_override fields handled elsewhere

-- +goose Down
DROP INDEX IF EXISTS idx_iwc_org;
DROP TABLE IF EXISTS inbox_webhook_configs;
DROP INDEX IF EXISTS idx_aicf_type;
DROP INDEX IF EXISTS idx_aicf_org;
DROP TABLE IF EXISTS ai_classification_feedback;
DROP INDEX IF EXISTS idx_uef_unread;
DROP INDEX IF EXISTS idx_uef_user;
DROP TABLE IF EXISTS user_email_feedback;
DROP INDEX IF EXISTS idx_abtest_variant;
DROP INDEX IF EXISTS idx_abtest_campaign;
DROP TABLE IF EXISTS ab_test_results;
