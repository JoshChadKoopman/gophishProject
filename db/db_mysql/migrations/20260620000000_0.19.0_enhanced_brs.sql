-- +goose Up
-- Phase 8: Enhanced Behavioral Risk Score (BRS)
-- Adds materialized risk score tables and user profile fields.

-- Add department and job_title to users table
ALTER TABLE users ADD COLUMN department VARCHAR(100) DEFAULT '';
ALTER TABLE users ADD COLUMN job_title VARCHAR(100) DEFAULT '';

-- Materialized per-user BRS with component breakdown
CREATE TABLE IF NOT EXISTS user_risk_scores (
    id                BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id           BIGINT NOT NULL UNIQUE,
    org_id            BIGINT NOT NULL,
    simulation_score  DOUBLE NOT NULL DEFAULT 0,
    academy_score     DOUBLE NOT NULL DEFAULT 0,
    quiz_score        DOUBLE NOT NULL DEFAULT 0,
    trend_score       DOUBLE NOT NULL DEFAULT 0,
    consistency_score DOUBLE NOT NULL DEFAULT 0,
    composite_score   DOUBLE NOT NULL DEFAULT 0,
    percentile        DOUBLE NOT NULL DEFAULT 0,
    last_calculated   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_user_risk_scores_org (org_id),
    INDEX idx_user_risk_scores_composite (composite_score)
);

-- Department-level aggregated scores
CREATE TABLE IF NOT EXISTS department_risk_scores (
    id              BIGINT AUTO_INCREMENT PRIMARY KEY,
    org_id          BIGINT NOT NULL,
    department      VARCHAR(100) NOT NULL,
    composite_score DOUBLE NOT NULL DEFAULT 0,
    user_count      INT NOT NULL DEFAULT 0,
    last_calculated DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uq_dept_org (org_id, department)
);

-- Historical BRS for trend charts
CREATE TABLE IF NOT EXISTS brs_history (
    id              BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id         BIGINT NOT NULL,
    composite_score DOUBLE NOT NULL DEFAULT 0,
    calculated_date DATE NOT NULL DEFAULT (CURRENT_DATE),
    INDEX idx_brs_history_user_date (user_id, calculated_date)
);

-- +goose Down
DROP TABLE IF EXISTS brs_history;
DROP TABLE IF EXISTS department_risk_scores;
DROP TABLE IF EXISTS user_risk_scores;
