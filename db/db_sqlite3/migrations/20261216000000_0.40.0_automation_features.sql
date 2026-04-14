-- +goose Up
-- Automated training reminders, content auto-update configs and audit logs

-- Reminder configuration per organization
CREATE TABLE IF NOT EXISTS "reminder_configs" (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "org_id" INTEGER NOT NULL UNIQUE,
    "enabled" INTEGER DEFAULT 1,
    "first_reminder_hours" INTEGER DEFAULT 48,
    "second_reminder_hours" INTEGER DEFAULT 24,
    "urgent_reminder_hours" INTEGER DEFAULT 4,
    "escalate_overdue_days" INTEGER DEFAULT 2,
    "email_template" TEXT DEFAULT '',
    "sending_profile_id" INTEGER DEFAULT 0
);

-- Content auto-update configuration per organization
CREATE TABLE IF NOT EXISTS "content_update_configs" (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "org_id" INTEGER NOT NULL UNIQUE,
    "enabled" INTEGER DEFAULT 1,
    "auto_assign_new" INTEGER DEFAULT 0,
    "notify_admins" INTEGER DEFAULT 1,
    "content_categories" TEXT DEFAULT '',
    "min_difficulty" INTEGER DEFAULT 0,
    "max_difficulty" INTEGER DEFAULT 0,
    "modified_date" DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Content update audit log
CREATE TABLE IF NOT EXISTS "content_update_logs" (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "org_id" INTEGER NOT NULL,
    "org_name" VARCHAR(255) DEFAULT '',
    "courses_added" INTEGER DEFAULT 0,
    "sessions_added" INTEGER DEFAULT 0,
    "quizzes_added" INTEGER DEFAULT 0,
    "skipped" INTEGER DEFAULT 0,
    "status" VARCHAR(50) DEFAULT 'success',
    "error_message" TEXT DEFAULT '',
    "run_date" DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS "idx_content_update_logs_org" ON "content_update_logs" ("org_id");
CREATE INDEX IF NOT EXISTS "idx_content_update_logs_date" ON "content_update_logs" ("run_date");

-- +goose Down
DROP TABLE IF EXISTS "content_update_logs";
DROP TABLE IF EXISTS "content_update_configs";
DROP TABLE IF EXISTS "reminder_configs";
