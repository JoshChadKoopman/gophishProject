-- +goose Up
-- Phase 8: Enhanced Behavioral Risk Score (BRS)
-- Adds materialized risk score tables and user profile fields.

-- Add department and job_title to users table
ALTER TABLE users ADD COLUMN department VARCHAR(100) DEFAULT '';
ALTER TABLE users ADD COLUMN job_title VARCHAR(100) DEFAULT '';

-- Materialized per-user BRS with component breakdown
CREATE TABLE IF NOT EXISTS user_risk_scores (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id           INTEGER NOT NULL UNIQUE,
    org_id            INTEGER NOT NULL,
    simulation_score  REAL NOT NULL DEFAULT 0,
    academy_score     REAL NOT NULL DEFAULT 0,
    quiz_score        REAL NOT NULL DEFAULT 0,
    trend_score       REAL NOT NULL DEFAULT 0,
    consistency_score REAL NOT NULL DEFAULT 0,
    composite_score   REAL NOT NULL DEFAULT 0,
    percentile        REAL NOT NULL DEFAULT 0,
    last_calculated   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_user_risk_scores_org ON user_risk_scores(org_id);
CREATE INDEX idx_user_risk_scores_composite ON user_risk_scores(composite_score);

-- Department-level aggregated scores
CREATE TABLE IF NOT EXISTS department_risk_scores (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id          INTEGER NOT NULL,
    department      VARCHAR(100) NOT NULL,
    composite_score REAL NOT NULL DEFAULT 0,
    user_count      INTEGER NOT NULL DEFAULT 0,
    last_calculated DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(org_id, department)
);

-- Historical BRS for trend charts
CREATE TABLE IF NOT EXISTS brs_history (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id         INTEGER NOT NULL,
    composite_score REAL NOT NULL DEFAULT 0,
    calculated_date DATE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_brs_history_user_date ON brs_history(user_id, calculated_date);

-- +goose Down
DROP INDEX IF EXISTS idx_brs_history_user_date;
DROP TABLE IF EXISTS brs_history;
DROP TABLE IF EXISTS department_risk_scores;
DROP INDEX IF EXISTS idx_user_risk_scores_composite;
DROP INDEX IF EXISTS idx_user_risk_scores_org;
DROP TABLE IF EXISTS user_risk_scores;
