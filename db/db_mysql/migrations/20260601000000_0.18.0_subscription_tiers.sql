-- +goose Up

-- Subscription tier definitions
CREATE TABLE IF NOT EXISTS subscription_tiers (
    id            INTEGER PRIMARY KEY AUTO_INCREMENT,
    slug          VARCHAR(50) NOT NULL UNIQUE,
    name          VARCHAR(100) NOT NULL,
    description   TEXT,
    max_users     INTEGER NOT NULL DEFAULT 10,
    max_campaigns INTEGER NOT NULL DEFAULT 5,
    is_active     BOOLEAN NOT NULL DEFAULT 1,
    sort_order    INTEGER NOT NULL DEFAULT 0,
    created_date  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Feature flags per tier
CREATE TABLE IF NOT EXISTS tier_features (
    id            INTEGER PRIMARY KEY AUTO_INCREMENT,
    tier_id       INTEGER NOT NULL,
    feature_slug  VARCHAR(100) NOT NULL,
    UNIQUE KEY uq_tier_feature (tier_id, feature_slug)
);

-- Seed the four tiers
INSERT INTO subscription_tiers (id, slug, name, description, max_users, max_campaigns, sort_order)
VALUES
    (1, 'core',        'Core',        'Security awareness baseline',            50,    20,   1),
    (2, 'advanced',    'Advanced',    'Holistic cyber resilience',             500,   100,   2),
    (3, 'all_in_one',  'All-In-One',  'Zero-incident rate achievement',      5000,  1000,   3),
    (4, 'enterprise',  'Enterprise',  'Unlimited custom deployment',       999999, 999999,   4);

-- Core features
INSERT INTO tier_features (tier_id, feature_slug) VALUES
    (1, 'basic_brs'),
    (1, 'threat_alerts_read');

-- Advanced features (includes Core)
INSERT INTO tier_features (tier_id, feature_slug) VALUES
    (2, 'basic_brs'),
    (2, 'advanced_brs'),
    (2, 'ai_templates'),
    (2, 'academy_advanced'),
    (2, 'gamification'),
    (2, 'report_button'),
    (2, 'threat_alerts_read'),
    (2, 'threat_alerts_create'),
    (2, 'board_reports'),
    (2, 'i18n_full');

-- All-In-One features (includes Advanced)
INSERT INTO tier_features (tier_id, feature_slug) VALUES
    (3, 'basic_brs'),
    (3, 'advanced_brs'),
    (3, 'ai_templates'),
    (3, 'autopilot'),
    (3, 'academy_advanced'),
    (3, 'gamification'),
    (3, 'report_button'),
    (3, 'threat_alerts_read'),
    (3, 'threat_alerts_create'),
    (3, 'board_reports'),
    (3, 'i18n_full'),
    (3, 'scim'),
    (3, 'zim'),
    (3, 'ai_assistant'),
    (3, 'power_bi');

-- Enterprise features (all features)
INSERT INTO tier_features (tier_id, feature_slug) VALUES
    (4, 'basic_brs'),
    (4, 'advanced_brs'),
    (4, 'ai_templates'),
    (4, 'autopilot'),
    (4, 'academy_advanced'),
    (4, 'gamification'),
    (4, 'report_button'),
    (4, 'threat_alerts_read'),
    (4, 'threat_alerts_create'),
    (4, 'board_reports'),
    (4, 'i18n_full'),
    (4, 'scim'),
    (4, 'zim'),
    (4, 'ai_assistant'),
    (4, 'power_bi'),
    (4, 'msp_whitelabel');

-- Add tier_id to organizations referencing subscription_tiers
ALTER TABLE organizations ADD COLUMN tier_id INTEGER NOT NULL DEFAULT 4;
ALTER TABLE organizations ADD COLUMN subscription_expires_at DATETIME;

-- Migrate existing orgs: map tier string to tier_id
UPDATE organizations SET tier_id = 4 WHERE tier = 'enterprise';
UPDATE organizations SET tier_id = 1 WHERE tier = 'free';

-- +goose Down
ALTER TABLE organizations DROP COLUMN tier_id;
ALTER TABLE organizations DROP COLUMN subscription_expires_at;
DROP TABLE IF EXISTS tier_features;
DROP TABLE IF EXISTS subscription_tiers;
