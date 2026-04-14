-- +goose Up
-- Custom training builder (All-in-One tier): multi-module courses. Each
-- training presentation may have many assets (PDF/PPTX/video/image) that
-- make up ordered modules in a custom course.
CREATE TABLE IF NOT EXISTS training_assets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    presentation_id INTEGER NOT NULL DEFAULT 0,
    org_id INTEGER NOT NULL DEFAULT 1,
    title VARCHAR(255) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    file_name VARCHAR(255) NOT NULL DEFAULT '',
    file_path VARCHAR(512) NOT NULL DEFAULT '',
    file_size INTEGER NOT NULL DEFAULT 0,
    content_type VARCHAR(255) NOT NULL DEFAULT '',
    asset_type VARCHAR(30) NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0,
    uploaded_by INTEGER NOT NULL DEFAULT 0,
    created_date DATETIME
);

CREATE INDEX IF NOT EXISTS idx_training_assets_pres ON training_assets(presentation_id);
CREATE INDEX IF NOT EXISTS idx_training_assets_org ON training_assets(org_id);

-- Seed the custom training builder feature on existing tiers with ids >= 3
-- (typically the All-in-One / Enterprise tiers in this fork's seed data).
INSERT INTO tier_features (tier_id, feature_slug)
    SELECT id, 'custom_training_builder'
    FROM subscription_tiers
    WHERE id >= 3
    AND NOT EXISTS (
        SELECT 1 FROM tier_features
        WHERE tier_id = subscription_tiers.id AND feature_slug = 'custom_training_builder'
    );

-- +goose Down
DELETE FROM tier_features WHERE feature_slug = 'custom_training_builder';
DROP INDEX IF EXISTS idx_training_assets_org;
DROP INDEX IF EXISTS idx_training_assets_pres;
DROP TABLE IF EXISTS training_assets;
