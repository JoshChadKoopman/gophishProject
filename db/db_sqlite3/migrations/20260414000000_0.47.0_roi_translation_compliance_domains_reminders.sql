-- +goose Up
-- 0.47.0: ROI Dashboard, AI Translation, Compliance Module Progress,
--         Sending Domain Pool, and Enhanced Training Reminders

-- ── ROI Investment Config ──
CREATE TABLE IF NOT EXISTS "roi_investment_configs" (
    "id"               INTEGER PRIMARY KEY AUTOINCREMENT,
    "org_id"           BIGINT NOT NULL UNIQUE,
    "phishing_sim_pct" REAL DEFAULT 30,
    "training_pct"     REAL DEFAULT 25,
    "tooling_pct"      REAL DEFAULT 25,
    "personnel_pct"    REAL DEFAULT 20
);
CREATE INDEX IF NOT EXISTS idx_roi_inv_org ON "roi_investment_configs" ("org_id");

-- ── AI Translation Tables ──
CREATE TABLE IF NOT EXISTS "translation_requests" (
    "id"            INTEGER PRIMARY KEY AUTOINCREMENT,
    "org_id"        BIGINT NOT NULL,
    "user_id"       BIGINT NOT NULL,
    "content_type"  VARCHAR(64) NOT NULL,
    "content_id"    BIGINT NOT NULL,
    "source_lang"   VARCHAR(10) NOT NULL,
    "target_lang"   VARCHAR(10) NOT NULL,
    "status"        VARCHAR(32) DEFAULT 'pending',
    "input_tokens"  INTEGER DEFAULT 0,
    "output_tokens" INTEGER DEFAULT 0,
    "created_date"  DATETIME,
    "completed_at"  DATETIME
);
CREATE INDEX IF NOT EXISTS idx_trans_req_org ON "translation_requests" ("org_id");

CREATE TABLE IF NOT EXISTS "translated_contents" (
    "id"               INTEGER PRIMARY KEY AUTOINCREMENT,
    "org_id"           BIGINT NOT NULL,
    "content_type"     VARCHAR(64) NOT NULL,
    "content_id"       BIGINT NOT NULL,
    "source_lang"      VARCHAR(10) NOT NULL,
    "target_lang"      VARCHAR(10) NOT NULL,
    "translated_title" TEXT,
    "translated_body"  TEXT,
    "translated_html"  TEXT,
    "quality"          REAL DEFAULT 0,
    "reviewed_by"      BIGINT,
    "reviewed_at"      DATETIME,
    "is_approved"      BOOLEAN DEFAULT 0,
    "created_date"     DATETIME,
    "modified_date"    DATETIME
);
CREATE INDEX IF NOT EXISTS idx_trans_content_org ON "translated_contents" ("org_id");
CREATE INDEX IF NOT EXISTS idx_trans_content_lookup ON "translated_contents" ("content_type", "content_id", "target_lang");

CREATE TABLE IF NOT EXISTS "translation_configs" (
    "id"                 INTEGER PRIMARY KEY AUTOINCREMENT,
    "org_id"             BIGINT NOT NULL UNIQUE,
    "enabled"            BOOLEAN DEFAULT 1,
    "auto_translate"     BOOLEAN DEFAULT 0,
    "default_langs"      TEXT,
    "review_required"    BOOLEAN DEFAULT 1,
    "max_monthly_tokens" INTEGER DEFAULT 500000
);
CREATE INDEX IF NOT EXISTS idx_trans_cfg_org ON "translation_configs" ("org_id");

-- ── Compliance Module Progress & Assignments ──
CREATE TABLE IF NOT EXISTS "compliance_module_progress" (
    "id"             INTEGER PRIMARY KEY AUTOINCREMENT,
    "user_id"        BIGINT NOT NULL,
    "org_id"         BIGINT NOT NULL,
    "module_slug"    VARCHAR(128) NOT NULL,
    "status"         VARCHAR(32) DEFAULT 'pending',
    "current_page"   INTEGER DEFAULT 0,
    "quiz_score"     INTEGER DEFAULT 0,
    "passed"         BOOLEAN DEFAULT 0,
    "attempts_count" INTEGER DEFAULT 0,
    "time_spent_secs" INTEGER DEFAULT 0,
    "started_date"   DATETIME,
    "completed_date" DATETIME,
    "created_date"   DATETIME
);
CREATE INDEX IF NOT EXISTS idx_comp_prog_user ON "compliance_module_progress" ("user_id");
CREATE INDEX IF NOT EXISTS idx_comp_prog_org ON "compliance_module_progress" ("org_id");
CREATE UNIQUE INDEX IF NOT EXISTS idx_comp_prog_user_module ON "compliance_module_progress" ("user_id", "module_slug");

CREATE TABLE IF NOT EXISTS "compliance_module_assignments" (
    "id"           INTEGER PRIMARY KEY AUTOINCREMENT,
    "org_id"       BIGINT NOT NULL,
    "module_slug"  VARCHAR(128) NOT NULL,
    "user_id"      BIGINT,
    "group_id"     BIGINT,
    "assigned_by"  BIGINT NOT NULL,
    "due_date"     DATETIME,
    "is_required"  BOOLEAN DEFAULT 1,
    "created_date" DATETIME
);
CREATE INDEX IF NOT EXISTS idx_comp_assign_org ON "compliance_module_assignments" ("org_id");

-- ── Sending Domain Pool ──
CREATE TABLE IF NOT EXISTS "sending_domains" (
    "id"               INTEGER PRIMARY KEY AUTOINCREMENT,
    "org_id"           BIGINT NOT NULL,
    "domain"           VARCHAR(255) NOT NULL,
    "display_name"     VARCHAR(255),
    "category"         VARCHAR(64) DEFAULT 'custom',
    "is_built_in"      BOOLEAN DEFAULT 0,
    "is_active"        BOOLEAN DEFAULT 1,
    "spf_configured"   BOOLEAN DEFAULT 0,
    "dkim_configured"  BOOLEAN DEFAULT 0,
    "dmarc_configured" BOOLEAN DEFAULT 0,
    "warmup_stage"     INTEGER DEFAULT 0,
    "daily_limit"      INTEGER DEFAULT 50,
    "sends_today"      INTEGER DEFAULT 0,
    "total_sent"       BIGINT DEFAULT 0,
    "last_used_date"   DATETIME,
    "health_status"    VARCHAR(32) DEFAULT 'unknown',
    "last_health_check" DATETIME,
    "notes"            TEXT,
    "created_date"     DATETIME,
    "modified_date"    DATETIME
);
CREATE INDEX IF NOT EXISTS idx_send_dom_org ON "sending_domains" ("org_id");
CREATE UNIQUE INDEX IF NOT EXISTS idx_send_dom_org_domain ON "sending_domains" ("org_id", "domain");

CREATE TABLE IF NOT EXISTS "domain_pool_configs" (
    "id"                   INTEGER PRIMARY KEY AUTOINCREMENT,
    "org_id"               BIGINT NOT NULL UNIQUE,
    "enabled"              BOOLEAN DEFAULT 1,
    "auto_rotate"          BOOLEAN DEFAULT 1,
    "rotation_strategy"    VARCHAR(32) DEFAULT 'round_robin',
    "max_daily_per_domain" INTEGER DEFAULT 50,
    "warmup_enabled"       BOOLEAN DEFAULT 1
);
CREATE INDEX IF NOT EXISTS idx_dom_pool_cfg_org ON "domain_pool_configs" ("org_id");

-- ── Reminder Templates (Enhanced Reminders) ──
CREATE TABLE IF NOT EXISTS "reminder_templates" (
    "id"        INTEGER PRIMARY KEY AUTOINCREMENT,
    "org_id"    BIGINT NOT NULL UNIQUE,
    "subject"   TEXT,
    "body_html" TEXT,
    "body_text" TEXT,
    "is_custom" BOOLEAN DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_rem_tpl_org ON "reminder_templates" ("org_id");


-- +goose Down
DROP TABLE IF EXISTS "reminder_templates";
DROP TABLE IF EXISTS "domain_pool_configs";
DROP TABLE IF EXISTS "sending_domains";
DROP TABLE IF EXISTS "compliance_module_assignments";
DROP TABLE IF EXISTS "compliance_module_progress";
DROP TABLE IF EXISTS "translation_configs";
DROP TABLE IF EXISTS "translated_contents";
DROP TABLE IF EXISTS "translation_requests";
DROP TABLE IF EXISTS "roi_investment_configs";
