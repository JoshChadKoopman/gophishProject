-- +goose Up

-- ─────────────────────────────────────────────────────────────────────────────
-- MSP Partners
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS msp_partners (
    id                 INTEGER PRIMARY KEY AUTOINCREMENT,
    name               VARCHAR(255) NOT NULL,
    slug               VARCHAR(100) NOT NULL UNIQUE,
    contact_email      VARCHAR(255) DEFAULT '',
    contact_phone      VARCHAR(50)  DEFAULT '',
    website            TEXT         DEFAULT '',
    max_clients        INTEGER NOT NULL DEFAULT 50,
    is_active          BOOLEAN NOT NULL DEFAULT 1,
    primary_user_id    INTEGER NOT NULL DEFAULT 0,
    notes              TEXT DEFAULT '',
    contract_starts_at DATETIME,
    contract_ends_at   DATETIME,
    created_date       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_date      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_msp_partners_slug ON msp_partners(slug);
CREATE INDEX idx_msp_partners_primary_user ON msp_partners(primary_user_id);

-- ─────────────────────────────────────────────────────────────────────────────
-- MSP Partner ↔ Client Organization mapping
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS msp_partner_clients (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    partner_id INTEGER NOT NULL,
    org_id     INTEGER NOT NULL,
    added_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    is_active  BOOLEAN NOT NULL DEFAULT 1,
    UNIQUE(partner_id, org_id)
);

CREATE INDEX idx_msp_partner_clients_partner ON msp_partner_clients(partner_id);
CREATE INDEX idx_msp_partner_clients_org     ON msp_partner_clients(org_id);

-- ─────────────────────────────────────────────────────────────────────────────
-- White-label branding configuration
-- ─────────────────────────────────────────────────────────────────────────────
CREATE TABLE IF NOT EXISTS white_label_configs (
    id                INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id            INTEGER NOT NULL DEFAULT 0,
    partner_id        INTEGER NOT NULL DEFAULT 0,
    company_name      VARCHAR(255) DEFAULT '',
    logo_url          TEXT DEFAULT '',
    logo_small_url    TEXT DEFAULT '',
    primary_color     VARCHAR(20) DEFAULT '#007bff',
    secondary_color   VARCHAR(20) DEFAULT '#6c757d',
    accent_color      VARCHAR(20) DEFAULT '#28a745',
    background_color  VARCHAR(20) DEFAULT '#ffffff',
    font_family       VARCHAR(100) DEFAULT '',
    login_page_title  VARCHAR(255) DEFAULT '',
    login_page_message TEXT DEFAULT '',
    footer_text       TEXT DEFAULT '',
    support_email     VARCHAR(255) DEFAULT '',
    support_url       TEXT DEFAULT '',
    custom_css        TEXT DEFAULT '',
    email_from_name   VARCHAR(255) DEFAULT '',
    email_footer_html TEXT DEFAULT '',
    hide_powered_by   BOOLEAN NOT NULL DEFAULT 0,
    is_active         BOOLEAN NOT NULL DEFAULT 1,
    created_date      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_date     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_white_label_org     ON white_label_configs(org_id);
CREATE INDEX idx_white_label_partner ON white_label_configs(partner_id);

-- ─────────────────────────────────────────────────────────────────────────────
-- Add the msp_partner role
-- ─────────────────────────────────────────────────────────────────────────────
INSERT OR IGNORE INTO roles (slug, name, description) VALUES
    ('msp_partner', 'MSP Partner', 'Managed Service Provider partner admin with multi-client access');

-- Grant permissions to the msp_partner role
INSERT OR IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug = 'msp_partner'
  AND p.slug IN ('view_objects', 'modify_objects', 'view_reports');

-- Add manage_msp permission
INSERT OR IGNORE INTO permissions (slug, name, description) VALUES
    ('manage_msp', 'Manage MSP', 'Manage MSP partner operations including multi-client management and white-label branding');

-- Grant manage_msp to msp_partner and superadmin roles
INSERT OR IGNORE INTO role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM roles r, permissions p
WHERE r.slug IN ('msp_partner', 'superadmin')
  AND p.slug = 'manage_msp';

-- ─────────────────────────────────────────────────────────────────────────────
-- Add new MSP feature flags to Enterprise tier
-- ─────────────────────────────────────────────────────────────────────────────
INSERT OR IGNORE INTO tier_features (tier_id, feature_slug) VALUES
    (4, 'msp_partner_portal'),
    (4, 'msp_multi_client');

-- Also ensure msp_whitelabel is in Enterprise (may already exist)
INSERT OR IGNORE INTO tier_features (tier_id, feature_slug) VALUES
    (4, 'msp_whitelabel');

-- +goose Down
DROP TABLE IF EXISTS white_label_configs;
DROP TABLE IF EXISTS msp_partner_clients;
DROP TABLE IF EXISTS msp_partners;

DELETE FROM role_permissions WHERE role_id IN (SELECT id FROM roles WHERE slug = 'msp_partner');
DELETE FROM roles WHERE slug = 'msp_partner';
DELETE FROM permissions WHERE slug = 'manage_msp';
DELETE FROM tier_features WHERE feature_slug IN ('msp_partner_portal', 'msp_multi_client');
