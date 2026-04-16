-- +goose Up
-- AI Admin Assistant (Aria): conversations, messages, onboarding progress.

CREATE TABLE IF NOT EXISTS "assistant_conversations" (
    "id"            INTEGER PRIMARY KEY AUTOINCREMENT,
    "org_id"        INTEGER NOT NULL,
    "user_id"       INTEGER NOT NULL,
    "title"         VARCHAR(255) DEFAULT '',
    "created_date"  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "modified_date" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_assistant_conv_org_user ON assistant_conversations(org_id, user_id);
CREATE INDEX IF NOT EXISTS idx_assistant_conv_modified ON assistant_conversations(modified_date);

CREATE TABLE IF NOT EXISTS "assistant_messages" (
    "id"              INTEGER PRIMARY KEY AUTOINCREMENT,
    "conversation_id" INTEGER NOT NULL,
    "role"            VARCHAR(20) NOT NULL DEFAULT 'admin',
    "content"         TEXT DEFAULT '',
    "tokens_used"     INTEGER DEFAULT 0,
    "created_date"    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_assistant_msg_conv ON assistant_messages(conversation_id);

CREATE TABLE IF NOT EXISTS "admin_onboarding_progress" (
    "id"             INTEGER PRIMARY KEY AUTOINCREMENT,
    "org_id"         INTEGER NOT NULL,
    "user_id"        INTEGER NOT NULL,
    "step"           VARCHAR(50) NOT NULL DEFAULT '',
    "completed_date" DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_admin_onboarding_unique ON admin_onboarding_progress(org_id, user_id, step);

-- AI admin assistant available on Professional (3), Enterprise (4), All-in-One (5).
INSERT OR IGNORE INTO tier_features (tier_id, feature_slug) VALUES (3, 'ai_assistant');
INSERT OR IGNORE INTO tier_features (tier_id, feature_slug) VALUES (4, 'ai_assistant');
INSERT OR IGNORE INTO tier_features (tier_id, feature_slug) VALUES (5, 'ai_assistant');

-- +goose Down
DELETE FROM tier_features WHERE feature_slug = 'ai_assistant';
DROP TABLE IF EXISTS "admin_onboarding_progress";
DROP TABLE IF EXISTS "assistant_messages";
DROP TABLE IF EXISTS "assistant_conversations";
