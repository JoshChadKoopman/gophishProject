-- +goose Up
-- Phase 14: Micro-feedback (educational interstitial on phish click)

CREATE TABLE IF NOT EXISTS feedback_pages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    org_id INTEGER NOT NULL DEFAULT 0,
    name VARCHAR(255) NOT NULL,
    language VARCHAR(10) DEFAULT 'en',
    html TEXT NOT NULL,
    redirect_url VARCHAR(500) DEFAULT '',
    redirect_delay_seconds INTEGER DEFAULT 10,
    modified_date DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_feedback_pages_org ON feedback_pages(org_id);

-- Add feedback fields to campaigns
ALTER TABLE campaigns ADD COLUMN feedback_enabled BOOLEAN DEFAULT 0;
ALTER TABLE campaigns ADD COLUMN feedback_page_id INTEGER DEFAULT 0;

-- +goose Down
ALTER TABLE campaigns DROP COLUMN feedback_enabled;
ALTER TABLE campaigns DROP COLUMN feedback_page_id;
DROP TABLE IF EXISTS feedback_pages;
