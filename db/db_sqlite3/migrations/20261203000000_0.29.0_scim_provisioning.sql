-- +goose Up
-- SCIM 2.0 IdP auto-provisioning tables

CREATE TABLE IF NOT EXISTS scim_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id INTEGER NOT NULL,
    token_hash VARCHAR(128) NOT NULL,
    description VARCHAR(255) DEFAULT '',
    created_by INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT 1,
    last_used DATETIME,
    created_date DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS scim_external_ids (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id INTEGER NOT NULL,
    resource_type VARCHAR(20) NOT NULL,
    external_id VARCHAR(255) NOT NULL,
    internal_id INTEGER NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_scim_tokens_hash ON scim_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_scim_tokens_org ON scim_tokens(org_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_scim_external_org_type_ext ON scim_external_ids(org_id, resource_type, external_id);
CREATE INDEX IF NOT EXISTS idx_scim_external_org_type_int ON scim_external_ids(org_id, resource_type, internal_id);

-- Enable SCIM feature for All-In-One tier (id=3) and MSP tier (id=5)
INSERT OR IGNORE INTO tier_features (tier_id, feature_slug) VALUES (3, 'scim');
INSERT OR IGNORE INTO tier_features (tier_id, feature_slug) VALUES (5, 'scim');

-- +goose Down
DROP TABLE IF EXISTS scim_external_ids;
DROP TABLE IF EXISTS scim_tokens;
