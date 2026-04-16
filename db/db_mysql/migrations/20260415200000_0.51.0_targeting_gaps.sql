-- +goose Up
-- Targeting Engine Gaps: A/B testing, send-time optimization, department threats, feedback loop

-- Add variant_id to results for A/B test tracking
ALTER TABLE results ADD COLUMN variant_id VARCHAR(8) NOT NULL DEFAULT '';

-- variant_id for autopilot_schedules is handled in a later migration
-- (autopilot_schedules is created by 20260715000000_0.21.0_autopilot.sql)

-- A/B Test Results table
CREATE TABLE IF NOT EXISTS ab_test_results (
    id              BIGINT AUTO_INCREMENT PRIMARY KEY,
    org_id          BIGINT NOT NULL DEFAULT 0,
    campaign_id     BIGINT NOT NULL,
    user_id         BIGINT NOT NULL,
    email           VARCHAR(255) NOT NULL DEFAULT '',
    variant_id      VARCHAR(8) NOT NULL DEFAULT 'A',
    template_id     BIGINT NOT NULL DEFAULT 0,
    template_name   VARCHAR(255) NOT NULL DEFAULT '',
    clicked         BOOLEAN NOT NULL DEFAULT FALSE,
    submitted       BOOLEAN NOT NULL DEFAULT FALSE,
    reported        BOOLEAN NOT NULL DEFAULT FALSE,
    time_to_click_s INT NOT NULL DEFAULT 0,
    created_date    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_abtest_campaign ON ab_test_results(campaign_id);
CREATE INDEX idx_abtest_variant ON ab_test_results(campaign_id, variant_id);

-- User email feedback table
CREATE TABLE IF NOT EXISTS user_email_feedback (
    id                BIGINT AUTO_INCREMENT PRIMARY KEY,
    org_id            BIGINT NOT NULL DEFAULT 0,
    user_id           BIGINT NOT NULL,
    email             VARCHAR(255) NOT NULL DEFAULT '',
    message_id        VARCHAR(512) NOT NULL DEFAULT '',
    subject           VARCHAR(512) NOT NULL DEFAULT '',
    sender_email      VARCHAR(255) NOT NULL DEFAULT '',
    threat_level      VARCHAR(32) NOT NULL DEFAULT 'safe',
    confidence_score  DOUBLE NOT NULL DEFAULT 0,
    summary           TEXT NOT NULL,
    indicators        JSON NOT NULL,
    recommendation    TEXT NOT NULL,
    was_simulation    BOOLEAN NOT NULL DEFAULT FALSE,
    simulation_result VARCHAR(32) NOT NULL DEFAULT '',
    learning_tip      TEXT NOT NULL,
    feedback_read     BOOLEAN NOT NULL DEFAULT FALSE,
    user_acknowledged BOOLEAN NOT NULL DEFAULT FALSE,
    created_date      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_date     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
CREATE INDEX idx_uef_user ON user_email_feedback(user_id);
CREATE INDEX idx_uef_unread ON user_email_feedback(user_id, feedback_read);

-- AI Classification Feedback table (false positive loop)
CREATE TABLE IF NOT EXISTS ai_classification_feedback (
    id                         BIGINT AUTO_INCREMENT PRIMARY KEY,
    org_id                     BIGINT NOT NULL DEFAULT 0,
    scan_result_id             BIGINT NOT NULL DEFAULT 0,
    reported_email_id          BIGINT NOT NULL DEFAULT 0,
    original_threat_level      VARCHAR(32) NOT NULL DEFAULT '',
    corrected_threat_level     VARCHAR(32) NOT NULL DEFAULT '',
    original_classification    VARCHAR(64) NOT NULL DEFAULT '',
    corrected_classification   VARCHAR(64) NOT NULL DEFAULT '',
    feedback_type              VARCHAR(32) NOT NULL DEFAULT '',
    admin_notes                TEXT NOT NULL,
    admin_user_id              BIGINT NOT NULL DEFAULT 0,
    created_date               DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_aicf_org ON ai_classification_feedback(org_id);
CREATE INDEX idx_aicf_type ON ai_classification_feedback(org_id, feedback_type);

-- Inbox Webhook Configs table
CREATE TABLE IF NOT EXISTS inbox_webhook_configs (
    id                   BIGINT AUTO_INCREMENT PRIMARY KEY,
    org_id               BIGINT NOT NULL DEFAULT 0,
    provider             VARCHAR(32) NOT NULL DEFAULT 'microsoft_graph',
    enabled              BOOLEAN NOT NULL DEFAULT FALSE,
    subscription_id      VARCHAR(512) NOT NULL DEFAULT '',
    webhook_url          VARCHAR(1024) NOT NULL DEFAULT '',
    expiration_date      DATETIME NULL,
    pubsub_topic         VARCHAR(512) NOT NULL DEFAULT '',
    pubsub_subscription  VARCHAR(512) NOT NULL DEFAULT '',
    history_id           VARCHAR(128) NOT NULL DEFAULT '',
    last_notification    DATETIME NULL,
    created_date         DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_date        DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
);
CREATE INDEX idx_iwc_org ON inbox_webhook_configs(org_id, provider);

-- +goose Down
DROP INDEX idx_iwc_org ON inbox_webhook_configs;
DROP TABLE IF EXISTS inbox_webhook_configs;
DROP INDEX idx_aicf_type ON ai_classification_feedback;
DROP INDEX idx_aicf_org ON ai_classification_feedback;
DROP TABLE IF EXISTS ai_classification_feedback;
DROP INDEX idx_uef_unread ON user_email_feedback;
DROP INDEX idx_uef_user ON user_email_feedback;
DROP TABLE IF EXISTS user_email_feedback;
DROP INDEX idx_abtest_variant ON ab_test_results;
DROP INDEX idx_abtest_campaign ON ab_test_results;
DROP TABLE IF EXISTS ab_test_results;
