-- +goose Up
-- Phase 10: Automated Campaign Scheduling (Autopilot)

CREATE TABLE IF NOT EXISTS autopilot_configs (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id INTEGER NOT NULL UNIQUE,
    enabled BOOLEAN NOT NULL DEFAULT 0,
    cadence_days INTEGER NOT NULL DEFAULT 15,
    active_hours_start INTEGER NOT NULL DEFAULT 9,
    active_hours_end INTEGER NOT NULL DEFAULT 17,
    timezone VARCHAR(50) NOT NULL DEFAULT 'UTC',
    target_group_ids TEXT NOT NULL DEFAULT '[]',
    sending_profile_id INTEGER NOT NULL DEFAULT 0,
    landing_page_id INTEGER NOT NULL DEFAULT 0,
    phish_url VARCHAR(255) NOT NULL DEFAULT '',
    last_run DATETIME,
    next_run DATETIME,
    created_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS autopilot_schedules (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id INTEGER NOT NULL,
    user_email VARCHAR(255) NOT NULL DEFAULT '',
    campaign_id INTEGER NOT NULL DEFAULT 0,
    difficulty_level INTEGER NOT NULL DEFAULT 2,
    scheduled_date DATETIME NOT NULL,
    sent BOOLEAN NOT NULL DEFAULT 0,
    created_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_autopilot_sched_org ON autopilot_schedules(org_id);
CREATE INDEX IF NOT EXISTS idx_autopilot_sched_sent ON autopilot_schedules(org_id, sent);

CREATE TABLE IF NOT EXISTS autopilot_blackout_dates (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id INTEGER NOT NULL,
    date VARCHAR(10) NOT NULL,
    reason VARCHAR(255) NOT NULL DEFAULT '',
    created_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_autopilot_blackout_org ON autopilot_blackout_dates(org_id);

-- +goose Down
DROP TABLE IF EXISTS autopilot_blackout_dates;
DROP TABLE IF EXISTS autopilot_schedules;
DROP TABLE IF EXISTS autopilot_configs;
