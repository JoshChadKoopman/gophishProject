-- +goose Up
-- Phase 9: AI-Powered Phishing Template Generation

-- Add AI-related columns to templates table
ALTER TABLE templates ADD COLUMN ai_generated BOOLEAN DEFAULT 0;
ALTER TABLE templates ADD COLUMN difficulty_level INTEGER DEFAULT 0;
ALTER TABLE templates ADD COLUMN language VARCHAR(10) DEFAULT 'en';
ALTER TABLE templates ADD COLUMN target_role VARCHAR(100) DEFAULT '';

-- AI generation log for auditing and token tracking
CREATE TABLE IF NOT EXISTS ai_generation_logs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL,
    provider VARCHAR(20) NOT NULL DEFAULT '',
    model_used VARCHAR(100) NOT NULL DEFAULT '',
    input_tokens INTEGER NOT NULL DEFAULT 0,
    output_tokens INTEGER NOT NULL DEFAULT 0,
    template_id INTEGER NOT NULL DEFAULT 0,
    created_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_ai_gen_logs_org ON ai_generation_logs(org_id);
CREATE INDEX IF NOT EXISTS idx_ai_gen_logs_created ON ai_generation_logs(org_id, created_date);

-- +goose Down
DROP TABLE IF EXISTS ai_generation_logs;
