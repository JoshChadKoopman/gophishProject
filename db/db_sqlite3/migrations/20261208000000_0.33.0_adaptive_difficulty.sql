-- +goose Up
-- Add adaptive difficulty preference columns to users table.
-- training_difficulty_mode: "adaptive" (AI-driven) or "manual" (user-chosen).
-- training_difficulty_manual: manual override level 1-4, used only when mode = "manual".
ALTER TABLE users ADD COLUMN training_difficulty_mode VARCHAR(20) DEFAULT 'adaptive';
ALTER TABLE users ADD COLUMN training_difficulty_manual INTEGER DEFAULT 0;

-- Difficulty adjustment history log for audit trail and trend analysis.
CREATE TABLE IF NOT EXISTS difficulty_adjustment_log (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    previous_level INTEGER NOT NULL,
    new_level INTEGER NOT NULL,
    source VARCHAR(20) NOT NULL,          -- "adaptive", "manual", "admin"
    reason TEXT NOT NULL,                  -- human-readable explanation
    brs_at_change REAL DEFAULT 0,
    click_rate_at_change REAL DEFAULT 0,
    created_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_difficulty_log_user ON difficulty_adjustment_log(user_id);
CREATE INDEX IF NOT EXISTS idx_difficulty_log_date ON difficulty_adjustment_log(created_date);

-- +goose Down
-- SQLite doesn't support DROP COLUMN directly; best-effort rollback.
DROP TABLE IF EXISTS difficulty_adjustment_log;
