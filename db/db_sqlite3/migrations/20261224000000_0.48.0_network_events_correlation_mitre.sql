-- +goose Up
-- Network Events: MITRE ATT&CK mapping, Event Correlation & Automated Playbooks

-- ─────────────────────────────────────────────────────────────────────────────
-- Add MITRE technique ID and incident_id columns to network_events
-- ─────────────────────────────────────────────────────────────────────────────
ALTER TABLE "network_events" ADD COLUMN "mitre_technique_id" VARCHAR(20) DEFAULT '';
ALTER TABLE "network_events" ADD COLUMN "incident_id" INTEGER DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_network_events_mitre ON network_events(org_id, mitre_technique_id);
CREATE INDEX IF NOT EXISTS idx_network_events_incident ON network_events(incident_id);

-- ─────────────────────────────────────────────────────────────────────────────
-- Extend network_event_rules for playbook support
-- ─────────────────────────────────────────────────────────────────────────────
ALTER TABLE "network_event_rules" ADD COLUMN "severity_match" VARCHAR(20) DEFAULT '';
ALTER TABLE "network_event_rules" ADD COLUMN "is_playbook" BOOLEAN DEFAULT 0;
ALTER TABLE "network_event_rules" ADD COLUMN "playbook_actions" TEXT DEFAULT '';

-- ─────────────────────────────────────────────────────────────────────────────
-- Network Incidents: correlated groups of events
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS "network_incidents" (
    "id"            INTEGER PRIMARY KEY AUTOINCREMENT,
    "org_id"        INTEGER NOT NULL,
    "title"         VARCHAR(255) NOT NULL DEFAULT '',
    "description"   TEXT DEFAULT '',
    "severity"      VARCHAR(20) NOT NULL DEFAULT 'medium',
    "status"        VARCHAR(20) NOT NULL DEFAULT 'open',
    "source_ip"     VARCHAR(45) DEFAULT '',
    "user_email"    VARCHAR(255) DEFAULT '',
    "event_count"   INTEGER DEFAULT 0,
    "first_seen"    DATETIME,
    "last_seen"     DATETIME,
    "assigned_to"   INTEGER DEFAULT 0,
    "created_date"  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "modified_date" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_network_incidents_org ON network_incidents(org_id);
CREATE INDEX IF NOT EXISTS idx_network_incidents_org_status ON network_incidents(org_id, status);

-- ─────────────────────────────────────────────────────────────────────────────
-- Playbook Execution Logs
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS "playbook_execution_logs" (
    "id"           INTEGER PRIMARY KEY AUTOINCREMENT,
    "org_id"       INTEGER NOT NULL,
    "rule_id"      INTEGER NOT NULL,
    "event_id"     INTEGER NOT NULL DEFAULT 0,
    "rule_name"    VARCHAR(255) DEFAULT '',
    "actions_run"  TEXT DEFAULT '',
    "status"       VARCHAR(20) DEFAULT 'success',
    "created_date" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_playbook_logs_org ON playbook_execution_logs(org_id);
CREATE INDEX IF NOT EXISTS idx_playbook_logs_rule ON playbook_execution_logs(rule_id);

-- +goose Down
DROP TABLE IF EXISTS "playbook_execution_logs";
DROP TABLE IF EXISTS "network_incidents";
-- SQLite doesn't support DROP COLUMN, so we leave the added columns in place
-- on downgrade. They are nullable and have defaults, so they are harmless.
