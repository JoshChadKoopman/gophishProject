-- +goose Up
-- NLP Email Analysis and Network Events Dashboard

-- ─────────────────────────────────────────────────────────────────────────────
-- Email Analyses: AI-powered NLP analysis of reported emails
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS "email_analyses" (
    "id"                INTEGER PRIMARY KEY AUTOINCREMENT,
    "org_id"            INTEGER NOT NULL,
    "reported_email_id" INTEGER NOT NULL,
    "status"            VARCHAR(20) NOT NULL DEFAULT 'pending',
    "threat_level"      VARCHAR(30) DEFAULT '',
    "confidence_score"  REAL DEFAULT 0.0,
    "classification"    VARCHAR(30) DEFAULT '',
    "summary"           TEXT DEFAULT '',
    "ai_provider"       VARCHAR(30) DEFAULT '',
    "tokens_used"       INTEGER DEFAULT 0,
    "analysis_duration" INTEGER DEFAULT 0,
    "created_date"      DATETIME DEFAULT CURRENT_TIMESTAMP,
    "completed_date"    DATETIME
);

CREATE INDEX IF NOT EXISTS idx_email_analyses_org ON email_analyses(org_id);
CREATE INDEX IF NOT EXISTS idx_email_analyses_reported ON email_analyses(reported_email_id);
CREATE INDEX IF NOT EXISTS idx_email_analyses_status ON email_analyses(org_id, status);
CREATE INDEX IF NOT EXISTS idx_email_analyses_threat ON email_analyses(org_id, threat_level);

-- ─────────────────────────────────────────────────────────────────────────────
-- Email Indicators: threat indicators extracted during NLP analysis
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS "email_indicators" (
    "id"          INTEGER PRIMARY KEY AUTOINCREMENT,
    "analysis_id" INTEGER NOT NULL,
    "type"        VARCHAR(30) NOT NULL DEFAULT '',
    "value"       TEXT DEFAULT '',
    "severity"    VARCHAR(20) DEFAULT 'info',
    "description" TEXT DEFAULT '',
    "sort_order"  INTEGER DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_email_indicators_analysis ON email_indicators(analysis_id);

-- ─────────────────────────────────────────────────────────────────────────────
-- Network Events: security events from external sources
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS "network_events" (
    "id"             INTEGER PRIMARY KEY AUTOINCREMENT,
    "org_id"         INTEGER NOT NULL,
    "source"         VARCHAR(30) NOT NULL DEFAULT 'custom',
    "event_type"     VARCHAR(50) NOT NULL DEFAULT '',
    "severity"       VARCHAR(20) NOT NULL DEFAULT 'info',
    "title"          VARCHAR(255) NOT NULL DEFAULT '',
    "description"    TEXT DEFAULT '',
    "source_ip"      VARCHAR(45) DEFAULT '',
    "destination_ip" VARCHAR(45) DEFAULT '',
    "user_id"        INTEGER DEFAULT 0,
    "user_email"     VARCHAR(255) DEFAULT '',
    "device_id"      VARCHAR(255) DEFAULT '',
    "raw_payload"    TEXT DEFAULT '',
    "status"         VARCHAR(20) NOT NULL DEFAULT 'new',
    "assigned_to"    INTEGER DEFAULT 0,
    "resolved_by"    INTEGER DEFAULT 0,
    "resolved_date"  DATETIME,
    "event_date"     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "created_date"   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "modified_date"  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_network_events_org ON network_events(org_id);
CREATE INDEX IF NOT EXISTS idx_network_events_org_status ON network_events(org_id, status);
CREATE INDEX IF NOT EXISTS idx_network_events_org_severity ON network_events(org_id, severity);
CREATE INDEX IF NOT EXISTS idx_network_events_org_source ON network_events(org_id, source);
CREATE INDEX IF NOT EXISTS idx_network_events_org_type ON network_events(org_id, event_type);
CREATE INDEX IF NOT EXISTS idx_network_events_event_date ON network_events(org_id, event_date);
CREATE INDEX IF NOT EXISTS idx_network_events_user_email ON network_events(user_email);

-- ─────────────────────────────────────────────────────────────────────────────
-- Network Event Notes: analyst notes on events
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS "network_event_notes" (
    "id"           INTEGER PRIMARY KEY AUTOINCREMENT,
    "event_id"     INTEGER NOT NULL,
    "user_id"      INTEGER NOT NULL DEFAULT 0,
    "content"      TEXT DEFAULT '',
    "created_date" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_network_event_notes_event ON network_event_notes(event_id);

-- ─────────────────────────────────────────────────────────────────────────────
-- Network Event Rules: automation rules for auto-severity/assignment
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS "network_event_rules" (
    "id"               INTEGER PRIMARY KEY AUTOINCREMENT,
    "org_id"           INTEGER NOT NULL,
    "name"             VARCHAR(255) NOT NULL DEFAULT '',
    "description"      TEXT DEFAULT '',
    "source_match"     VARCHAR(100) DEFAULT '',
    "event_type_match" VARCHAR(100) DEFAULT '',
    "auto_severity"    VARCHAR(20) DEFAULT '',
    "auto_assign"      INTEGER DEFAULT 0,
    "enabled"          BOOLEAN NOT NULL DEFAULT 1,
    "created_date"     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "modified_date"    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_network_event_rules_org ON network_event_rules(org_id);

-- ─────────────────────────────────────────────────────────────────────────────
-- Feature flags for new modules
-- ─────────────────────────────────────────────────────────────────────────────
-- NLP Email Analysis available on Professional (id=3) and above
INSERT OR IGNORE INTO tier_features (tier_id, feature_slug) VALUES (3, 'nlp_email_analysis');
INSERT OR IGNORE INTO tier_features (tier_id, feature_slug) VALUES (4, 'nlp_email_analysis');
INSERT OR IGNORE INTO tier_features (tier_id, feature_slug) VALUES (5, 'nlp_email_analysis');

-- Network Events Dashboard available on Enterprise (id=4) and above
INSERT OR IGNORE INTO tier_features (tier_id, feature_slug) VALUES (4, 'network_events');
INSERT OR IGNORE INTO tier_features (tier_id, feature_slug) VALUES (5, 'network_events');

-- +goose Down
DELETE FROM tier_features WHERE feature_slug IN ('nlp_email_analysis', 'network_events');
DROP TABLE IF EXISTS "network_event_rules";
DROP TABLE IF EXISTS "network_event_notes";
DROP TABLE IF EXISTS "network_events";
DROP TABLE IF EXISTS "email_indicators";
DROP TABLE IF EXISTS "email_analyses";
