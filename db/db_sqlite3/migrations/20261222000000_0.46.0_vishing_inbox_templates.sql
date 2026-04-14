-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

-- ── Vishing Scenarios ──
CREATE TABLE IF NOT EXISTS "vishing_scenarios" (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "org_id" INTEGER NOT NULL DEFAULT 0,
    "user_id" INTEGER NOT NULL DEFAULT 0,
    "name" VARCHAR(255) NOT NULL DEFAULT '',
    "description" TEXT DEFAULT '',
    "category" VARCHAR(100) DEFAULT '',
    "difficulty_level" INTEGER NOT NULL DEFAULT 2,
    "language" VARCHAR(10) DEFAULT 'en',
    "caller_id_name" VARCHAR(255) DEFAULT '',
    "caller_id_number" VARCHAR(50) DEFAULT '',
    "script" TEXT DEFAULT '',
    "script_type" VARCHAR(50) DEFAULT 'text',
    "greeting" TEXT DEFAULT '',
    "pretext" TEXT DEFAULT '',
    "success_criteria" TEXT DEFAULT '',
    "recording_enabled" BOOLEAN DEFAULT 0,
    "consent_message" TEXT DEFAULT '',
    "max_duration_sec" INTEGER DEFAULT 120,
    "created_date" DATETIME DEFAULT CURRENT_TIMESTAMP,
    "modified_date" DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_vishing_scenarios_org ON vishing_scenarios(org_id);

-- ── Vishing Campaigns ──
CREATE TABLE IF NOT EXISTS "vishing_campaigns" (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "org_id" INTEGER NOT NULL DEFAULT 0,
    "user_id" INTEGER NOT NULL DEFAULT 0,
    "name" VARCHAR(255) NOT NULL DEFAULT '',
    "status" VARCHAR(50) DEFAULT 'Created',
    "scenario_id" INTEGER NOT NULL DEFAULT 0,
    "group_ids" TEXT DEFAULT '[]',
    "telephony_provider" VARCHAR(50) DEFAULT 'twilio',
    "sms_provider_id" INTEGER DEFAULT 0,
    "schedule_start" DATETIME,
    "schedule_end" DATETIME,
    "active_hours_start" INTEGER DEFAULT 9,
    "active_hours_end" INTEGER DEFAULT 17,
    "timezone" VARCHAR(50) DEFAULT 'UTC',
    "retry_attempts" INTEGER DEFAULT 1,
    "launch_date" DATETIME,
    "completed_date" DATETIME,
    "created_date" DATETIME DEFAULT CURRENT_TIMESTAMP,
    "modified_date" DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_vishing_campaigns_org ON vishing_campaigns(org_id);
CREATE INDEX IF NOT EXISTS idx_vishing_campaigns_status ON vishing_campaigns(status);

-- ── Vishing Results ──
CREATE TABLE IF NOT EXISTS "vishing_results" (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "campaign_id" INTEGER NOT NULL DEFAULT 0,
    "org_id" INTEGER NOT NULL DEFAULT 0,
    "email" VARCHAR(255) DEFAULT '',
    "first_name" VARCHAR(255) DEFAULT '',
    "last_name" VARCHAR(255) DEFAULT '',
    "phone_number" VARCHAR(50) DEFAULT '',
    "status" VARCHAR(50) DEFAULT 'pending',
    "call_sid" VARCHAR(255) DEFAULT '',
    "call_duration_sec" INTEGER DEFAULT 0,
    "ivr_path" TEXT DEFAULT '[]',
    "info_disclosed" TEXT DEFAULT '{}',
    "recording_url" TEXT DEFAULT '',
    "attempt_count" INTEGER DEFAULT 1,
    "reported" BOOLEAN DEFAULT 0,
    "reported_date" DATETIME,
    "send_date" DATETIME,
    "completed_date" DATETIME,
    "created_date" DATETIME DEFAULT CURRENT_TIMESTAMP,
    "modified_date" DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_vishing_results_campaign ON vishing_results(campaign_id);
CREATE INDEX IF NOT EXISTS idx_vishing_results_email ON vishing_results(email);
CREATE INDEX IF NOT EXISTS idx_vishing_results_status ON vishing_results(status);

-- ── User Email Feedback (Inbox AI feedback for end users) ──
CREATE TABLE IF NOT EXISTS "user_email_feedback" (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "org_id" INTEGER NOT NULL DEFAULT 0,
    "user_id" INTEGER NOT NULL DEFAULT 0,
    "email" VARCHAR(255) DEFAULT '',
    "message_id" VARCHAR(255) DEFAULT '',
    "subject" TEXT DEFAULT '',
    "sender_email" VARCHAR(255) DEFAULT '',
    "threat_level" VARCHAR(50) DEFAULT 'safe',
    "confidence_score" REAL DEFAULT 0,
    "summary" TEXT DEFAULT '',
    "indicators" TEXT DEFAULT '[]',
    "recommendation" TEXT DEFAULT '',
    "was_simulation" BOOLEAN DEFAULT 0,
    "simulation_result" VARCHAR(50) DEFAULT '',
    "learning_tip" TEXT DEFAULT '',
    "feedback_read" BOOLEAN DEFAULT 0,
    "user_acknowledged" BOOLEAN DEFAULT 0,
    "created_date" DATETIME DEFAULT CURRENT_TIMESTAMP,
    "modified_date" DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_user_email_feedback_user ON user_email_feedback(user_id);
CREATE INDEX IF NOT EXISTS idx_user_email_feedback_org ON user_email_feedback(org_id);
CREATE INDEX IF NOT EXISTS idx_user_email_feedback_read ON user_email_feedback(user_id, feedback_read);

-- ── AI Classification Feedback (admin false-positive loop) ──
CREATE TABLE IF NOT EXISTS "ai_classification_feedback" (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "org_id" INTEGER NOT NULL DEFAULT 0,
    "scan_result_id" INTEGER DEFAULT 0,
    "reported_email_id" INTEGER DEFAULT 0,
    "original_threat_level" VARCHAR(50) DEFAULT '',
    "corrected_threat_level" VARCHAR(50) DEFAULT '',
    "original_classification" VARCHAR(100) DEFAULT '',
    "corrected_classification" VARCHAR(100) DEFAULT '',
    "feedback_type" VARCHAR(50) DEFAULT '',
    "admin_notes" TEXT DEFAULT '',
    "admin_user_id" INTEGER DEFAULT 0,
    "created_date" DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_ai_class_feedback_org ON ai_classification_feedback(org_id);
CREATE INDEX IF NOT EXISTS idx_ai_class_feedback_type ON ai_classification_feedback(org_id, feedback_type);

-- ── Inbox Webhook Configs (Microsoft Graph / Gmail Pub/Sub) ──
CREATE TABLE IF NOT EXISTS "inbox_webhook_configs" (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "org_id" INTEGER NOT NULL DEFAULT 0,
    "provider" VARCHAR(50) DEFAULT 'microsoft_graph',
    "enabled" BOOLEAN DEFAULT 0,
    "subscription_id" VARCHAR(255) DEFAULT '',
    "webhook_url" TEXT DEFAULT '',
    "expiration_date" DATETIME,
    "pubsub_topic" VARCHAR(255) DEFAULT '',
    "pubsub_subscription" VARCHAR(255) DEFAULT '',
    "history_id" VARCHAR(255) DEFAULT '',
    "last_notification" DATETIME,
    "created_date" DATETIME DEFAULT CURRENT_TIMESTAMP,
    "modified_date" DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_inbox_webhook_org_provider ON inbox_webhook_configs(org_id, provider);

-- ── DB-Backed Template Library ──
CREATE TABLE IF NOT EXISTS "library_templates" (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "slug" VARCHAR(255) NOT NULL DEFAULT '',
    "name" VARCHAR(255) NOT NULL DEFAULT '',
    "category" VARCHAR(100) DEFAULT '',
    "difficulty_level" INTEGER DEFAULT 2,
    "description" TEXT DEFAULT '',
    "subject" TEXT DEFAULT '',
    "text" TEXT DEFAULT '',
    "html" TEXT DEFAULT '',
    "envelope_sender" VARCHAR(255) DEFAULT '',
    "language" VARCHAR(10) DEFAULT 'en',
    "target_role" VARCHAR(100) DEFAULT '',
    "tags" TEXT DEFAULT '[]',
    "similarity_hash" VARCHAR(64) DEFAULT '',
    "source" VARCHAR(50) DEFAULT 'builtin',
    "org_id" INTEGER NOT NULL DEFAULT 0,
    "created_by" INTEGER DEFAULT 0,
    "is_published" BOOLEAN DEFAULT 1,
    "usage_count" INTEGER DEFAULT 0,
    "avg_click_rate" REAL DEFAULT 0,
    "created_date" DATETIME DEFAULT CURRENT_TIMESTAMP,
    "modified_date" DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_library_templates_slug ON library_templates(slug);
CREATE INDEX IF NOT EXISTS idx_library_templates_category ON library_templates(category);
CREATE INDEX IF NOT EXISTS idx_library_templates_org ON library_templates(org_id);
CREATE INDEX IF NOT EXISTS idx_library_templates_source ON library_templates(source);
CREATE INDEX IF NOT EXISTS idx_library_templates_hash ON library_templates(similarity_hash);
CREATE INDEX IF NOT EXISTS idx_library_templates_search ON library_templates(name, category, language, is_published);

-- ── Add admin_override columns to inbox_scan_results ──
-- These support the false positive feedback loop
ALTER TABLE inbox_scan_results ADD COLUMN "admin_override" BOOLEAN DEFAULT 0;
ALTER TABLE inbox_scan_results ADD COLUMN "admin_override_by" INTEGER DEFAULT 0;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE IF EXISTS "vishing_results";
DROP TABLE IF EXISTS "vishing_campaigns";
DROP TABLE IF EXISTS "vishing_scenarios";
DROP TABLE IF EXISTS "user_email_feedback";
DROP TABLE IF EXISTS "ai_classification_feedback";
DROP TABLE IF EXISTS "inbox_webhook_configs";
DROP TABLE IF EXISTS "library_templates";
