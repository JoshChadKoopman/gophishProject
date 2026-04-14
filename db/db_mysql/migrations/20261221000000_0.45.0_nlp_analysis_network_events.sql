-- +goose Up
-- NLP Email Analysis and Network Events Dashboard

CREATE TABLE IF NOT EXISTS email_analyses (
    id                BIGINT AUTO_INCREMENT PRIMARY KEY,
    org_id            BIGINT NOT NULL,
    reported_email_id BIGINT NOT NULL,
    status            VARCHAR(20) NOT NULL DEFAULT 'pending',
    threat_level      VARCHAR(30) DEFAULT '',
    confidence_score  DOUBLE DEFAULT 0.0,
    classification    VARCHAR(30) DEFAULT '',
    summary           TEXT,
    ai_provider       VARCHAR(30) DEFAULT '',
    tokens_used       INT DEFAULT 0,
    analysis_duration INT DEFAULT 0,
    created_date      DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_date    DATETIME NULL,
    INDEX idx_email_analyses_org (org_id),
    INDEX idx_email_analyses_reported (reported_email_id),
    INDEX idx_email_analyses_status (org_id, status),
    INDEX idx_email_analyses_threat (org_id, threat_level)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS email_indicators (
    id          BIGINT AUTO_INCREMENT PRIMARY KEY,
    analysis_id BIGINT NOT NULL,
    type        VARCHAR(30) NOT NULL DEFAULT '',
    value       TEXT,
    severity    VARCHAR(20) DEFAULT 'info',
    description TEXT,
    sort_order  INT DEFAULT 0,
    INDEX idx_email_indicators_analysis (analysis_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS network_events (
    id             BIGINT AUTO_INCREMENT PRIMARY KEY,
    org_id         BIGINT NOT NULL,
    source         VARCHAR(30) NOT NULL DEFAULT 'custom',
    event_type     VARCHAR(50) NOT NULL DEFAULT '',
    severity       VARCHAR(20) NOT NULL DEFAULT 'info',
    title          VARCHAR(255) NOT NULL DEFAULT '',
    description    TEXT,
    source_ip      VARCHAR(45) DEFAULT '',
    destination_ip VARCHAR(45) DEFAULT '',
    user_id        BIGINT DEFAULT 0,
    user_email     VARCHAR(255) DEFAULT '',
    device_id      VARCHAR(255) DEFAULT '',
    raw_payload    TEXT,
    status         VARCHAR(20) NOT NULL DEFAULT 'new',
    assigned_to    BIGINT DEFAULT 0,
    resolved_by    BIGINT DEFAULT 0,
    resolved_date  DATETIME NULL,
    event_date     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_date   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_date  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_network_events_org (org_id),
    INDEX idx_network_events_org_status (org_id, status),
    INDEX idx_network_events_org_severity (org_id, severity),
    INDEX idx_network_events_org_source (org_id, source),
    INDEX idx_network_events_org_type (org_id, event_type),
    INDEX idx_network_events_event_date (org_id, event_date),
    INDEX idx_network_events_user_email (user_email)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS network_event_notes (
    id           BIGINT AUTO_INCREMENT PRIMARY KEY,
    event_id     BIGINT NOT NULL,
    user_id      BIGINT NOT NULL DEFAULT 0,
    content      TEXT,
    created_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_network_event_notes_event (event_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS network_event_rules (
    id               BIGINT AUTO_INCREMENT PRIMARY KEY,
    org_id           BIGINT NOT NULL,
    name             VARCHAR(255) NOT NULL DEFAULT '',
    description      TEXT,
    source_match     VARCHAR(100) DEFAULT '',
    event_type_match VARCHAR(100) DEFAULT '',
    auto_severity    VARCHAR(20) DEFAULT '',
    auto_assign      BIGINT DEFAULT 0,
    enabled          BOOLEAN NOT NULL DEFAULT 1,
    created_date     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_date    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_network_event_rules_org (org_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Feature flags
INSERT IGNORE INTO tier_features (tier_id, feature_slug) VALUES (3, 'nlp_email_analysis');
INSERT IGNORE INTO tier_features (tier_id, feature_slug) VALUES (4, 'nlp_email_analysis');
INSERT IGNORE INTO tier_features (tier_id, feature_slug) VALUES (5, 'nlp_email_analysis');
INSERT IGNORE INTO tier_features (tier_id, feature_slug) VALUES (4, 'network_events');
INSERT IGNORE INTO tier_features (tier_id, feature_slug) VALUES (5, 'network_events');

-- +goose Down
DELETE FROM tier_features WHERE feature_slug IN ('nlp_email_analysis', 'network_events');
DROP TABLE IF EXISTS network_event_rules;
DROP TABLE IF EXISTS network_event_notes;
DROP TABLE IF EXISTS network_events;
DROP TABLE IF EXISTS email_indicators;
DROP TABLE IF EXISTS email_analyses;
