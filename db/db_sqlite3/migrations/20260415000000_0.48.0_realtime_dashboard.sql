-- +goose Up
-- Real-time dashboard preferences table
CREATE TABLE IF NOT EXISTS dashboard_preferences (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL UNIQUE,
    org_id INTEGER NOT NULL DEFAULT 0,
    time_window VARCHAR(10) NOT NULL DEFAULT '30d',
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_dashboard_preferences_user ON dashboard_preferences(user_id);

-- +goose Down
DROP TABLE IF EXISTS dashboard_preferences;
