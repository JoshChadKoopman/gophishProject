-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied.

-- Adaptive engine per-org configuration
CREATE TABLE IF NOT EXISTS "adaptive_engine_configs" (
    "id"                      INTEGER PRIMARY KEY AUTOINCREMENT,
    "org_id"                  INTEGER NOT NULL UNIQUE,
    "enabled"                 BOOLEAN DEFAULT 1,
    "eval_interval_days"      INTEGER DEFAULT 7,
    "brs_weight_pct"          REAL DEFAULT 40.0,
    "click_rate_weight_pct"   REAL DEFAULT 30.0,
    "quiz_score_weight_pct"   REAL DEFAULT 20.0,
    "trend_weight_pct"        REAL DEFAULT 10.0,
    "promote_threshold"       REAL DEFAULT 75.0,
    "demote_threshold"        REAL DEFAULT 35.0,
    "min_simulations_promote" INTEGER DEFAULT 3,
    "cooldown_days"           INTEGER DEFAULT 14,
    "modified_date"           DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Adaptive engine run audit log
CREATE TABLE IF NOT EXISTS "adaptive_engine_run_logs" (
    "id"              INTEGER PRIMARY KEY AUTOINCREMENT,
    "org_id"          INTEGER NOT NULL,
    "users_evaluated" INTEGER DEFAULT 0,
    "promoted"        INTEGER DEFAULT 0,
    "demoted"         INTEGER DEFAULT 0,
    "maintained"      INTEGER DEFAULT 0,
    "skipped"         INTEGER DEFAULT 0,
    "run_date"        DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS "idx_ae_run_logs_org_date" ON "adaptive_engine_run_logs" ("org_id", "run_date");

-- Nanolearning events (micro-intervention tracking)
CREATE TABLE IF NOT EXISTS "nanolearning_events" (
    "id"           INTEGER PRIMARY KEY AUTOINCREMENT,
    "user_id"      INTEGER DEFAULT 0,
    "email"        VARCHAR(255) NOT NULL,
    "campaign_id"  INTEGER NOT NULL,
    "result_id"    VARCHAR(255) DEFAULT '',
    "content_slug" VARCHAR(255) DEFAULT '',
    "tip_text"     TEXT DEFAULT '',
    "category"     VARCHAR(100) DEFAULT '',
    "acknowledged" BOOLEAN DEFAULT 0,
    "created_date" DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS "idx_nano_events_user" ON "nanolearning_events" ("user_id");
CREATE INDEX IF NOT EXISTS "idx_nano_events_email_campaign" ON "nanolearning_events" ("email", "campaign_id");

-- ROI configuration per org
CREATE TABLE IF NOT EXISTS "roi_configs" (
    "id"               INTEGER PRIMARY KEY AUTOINCREMENT,
    "org_id"           INTEGER NOT NULL UNIQUE,
    "program_cost"     REAL DEFAULT 50000.0,
    "avg_breach_cost"  REAL DEFAULT 4450000.0,
    "avg_incident_cost" REAL DEFAULT 1500.0,
    "employee_count"   INTEGER DEFAULT 200,
    "avg_salary_hr"    REAL DEFAULT 45.0,
    "currency"         VARCHAR(10) DEFAULT 'USD',
    "modified_date"    DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Training reminders
CREATE TABLE IF NOT EXISTS "training_reminders" (
    "id"              INTEGER PRIMARY KEY AUTOINCREMENT,
    "user_id"         INTEGER NOT NULL,
    "assignment_id"   INTEGER NOT NULL,
    "presentation_id" INTEGER DEFAULT 0,
    "course_name"     VARCHAR(255) DEFAULT '',
    "due_date"        DATETIME,
    "reminder_type"   VARCHAR(50) DEFAULT 'standard',
    "message"         TEXT DEFAULT '',
    "sent_date"       DATETIME DEFAULT CURRENT_TIMESTAMP,
    "email_sent"      BOOLEAN DEFAULT 0
);
CREATE INDEX IF NOT EXISTS "idx_training_reminders_user" ON "training_reminders" ("user_id");
CREATE INDEX IF NOT EXISTS "idx_training_reminders_assignment" ON "training_reminders" ("assignment_id");

-- +goose Down
-- SQL in section 'Down' is executed when this migration is rolled back.
DROP TABLE IF EXISTS "training_reminders";
DROP TABLE IF EXISTS "roi_configs";
DROP TABLE IF EXISTS "nanolearning_events";
DROP TABLE IF EXISTS "adaptive_engine_run_logs";
DROP TABLE IF EXISTS "adaptive_engine_configs";
