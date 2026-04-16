-- +goose Up
-- Scheduled Reports: admin-configurable recurring report delivery
CREATE TABLE IF NOT EXISTS scheduled_reports (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id        INTEGER NOT NULL DEFAULT 0,
    user_id       INTEGER NOT NULL,
    name          VARCHAR(255) NOT NULL DEFAULT '',
    report_type   VARCHAR(64) NOT NULL DEFAULT 'executive_summary',
    format        VARCHAR(16) NOT NULL DEFAULT 'pdf',
    frequency     VARCHAR(32) NOT NULL DEFAULT 'weekly',
    day_of_week   INTEGER NOT NULL DEFAULT 1,
    day_of_month  INTEGER NOT NULL DEFAULT 1,
    hour          INTEGER NOT NULL DEFAULT 8,
    minute        INTEGER NOT NULL DEFAULT 0,
    timezone      VARCHAR(64) NOT NULL DEFAULT 'UTC',
    recipients    TEXT NOT NULL DEFAULT '',
    subject       VARCHAR(512) NOT NULL DEFAULT '',
    include_branding BOOLEAN NOT NULL DEFAULT 1,
    is_active     BOOLEAN NOT NULL DEFAULT 1,
    filters       TEXT NOT NULL DEFAULT '{}',
    last_run_at   DATETIME NULL,
    next_run_at   DATETIME NULL,
    last_status   VARCHAR(32) NOT NULL DEFAULT '',
    last_error    TEXT NOT NULL DEFAULT '',
    run_count     INTEGER NOT NULL DEFAULT 0,
    created_date  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_sr_org ON scheduled_reports(org_id);
CREATE INDEX IF NOT EXISTS idx_sr_active_next ON scheduled_reports(is_active, next_run_at);
CREATE INDEX IF NOT EXISTS idx_sr_user ON scheduled_reports(user_id);

-- +goose Down
DROP TABLE IF EXISTS scheduled_reports;
