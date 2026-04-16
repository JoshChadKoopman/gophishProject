-- +goose Up
-- Board report schedules + branding for auto-generation

CREATE TABLE IF NOT EXISTS board_report_schedules (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id          INTEGER NOT NULL DEFAULT 0,
    frequency       VARCHAR(20) NOT NULL DEFAULT 'monthly',
    day_of_month    INTEGER NOT NULL DEFAULT 1,
    enabled         BOOLEAN NOT NULL DEFAULT 1,
    auto_publish    BOOLEAN NOT NULL DEFAULT 0,
    notify_emails   TEXT NOT NULL DEFAULT '',
    created_by      INTEGER NOT NULL DEFAULT 0,
    last_run_date   DATETIME,
    next_run_date   DATETIME,
    created_date    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_date   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_brs_org ON board_report_schedules(org_id);
CREATE INDEX IF NOT EXISTS idx_brs_next ON board_report_schedules(next_run_date);

-- +goose Down
DROP INDEX IF EXISTS idx_brs_next;
DROP INDEX IF EXISTS idx_brs_org;
DROP TABLE IF EXISTS board_report_schedules;
