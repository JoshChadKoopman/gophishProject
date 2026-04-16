-- +goose Up
-- Network Events: MITRE ATT&CK mapping, Event Correlation & Automated Playbooks

-- ─────────────────────────────────────────────────────────────────────────────
-- Add MITRE technique ID and incident_id columns to network_events
-- ─────────────────────────────────────────────────────────────────────────────
ALTER TABLE network_events ADD COLUMN mitre_technique_id VARCHAR(20) DEFAULT '';
ALTER TABLE network_events ADD COLUMN incident_id BIGINT DEFAULT 0;

CREATE INDEX idx_network_events_mitre ON network_events(org_id, mitre_technique_id);
CREATE INDEX idx_network_events_incident ON network_events(incident_id);

-- ─────────────────────────────────────────────────────────────────────────────
-- Extend network_event_rules for playbook support
-- ─────────────────────────────────────────────────────────────────────────────
ALTER TABLE network_event_rules ADD COLUMN severity_match VARCHAR(20) DEFAULT '';
ALTER TABLE network_event_rules ADD COLUMN is_playbook BOOLEAN DEFAULT 0;
ALTER TABLE network_event_rules ADD COLUMN playbook_actions TEXT;

-- ─────────────────────────────────────────────────────────────────────────────
-- Network Incidents: correlated groups of events
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS network_incidents (
    id            BIGINT AUTO_INCREMENT PRIMARY KEY,
    org_id        BIGINT NOT NULL,
    title         VARCHAR(255) NOT NULL DEFAULT '',
    description   TEXT,
    severity      VARCHAR(20) NOT NULL DEFAULT 'medium',
    status        VARCHAR(20) NOT NULL DEFAULT 'open',
    source_ip     VARCHAR(45) DEFAULT '',
    user_email    VARCHAR(255) DEFAULT '',
    event_count   INT DEFAULT 0,
    first_seen    DATETIME NULL,
    last_seen     DATETIME NULL,
    assigned_to   BIGINT DEFAULT 0,
    created_date  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_network_incidents_org (org_id),
    INDEX idx_network_incidents_org_status (org_id, status)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ─────────────────────────────────────────────────────────────────────────────
-- Playbook Execution Logs
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS playbook_execution_logs (
    id           BIGINT AUTO_INCREMENT PRIMARY KEY,
    org_id       BIGINT NOT NULL,
    rule_id      BIGINT NOT NULL,
    event_id     BIGINT NOT NULL DEFAULT 0,
    rule_name    VARCHAR(255) DEFAULT '',
    actions_run  TEXT,
    status       VARCHAR(20) DEFAULT 'success',
    created_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_playbook_logs_org (org_id),
    INDEX idx_playbook_logs_rule (rule_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS playbook_execution_logs;
DROP TABLE IF EXISTS network_incidents;
ALTER TABLE network_event_rules DROP COLUMN playbook_actions;
ALTER TABLE network_event_rules DROP COLUMN is_playbook;
ALTER TABLE network_event_rules DROP COLUMN severity_match;
ALTER TABLE network_events DROP COLUMN incident_id;
ALTER TABLE network_events DROP COLUMN mitre_technique_id;
