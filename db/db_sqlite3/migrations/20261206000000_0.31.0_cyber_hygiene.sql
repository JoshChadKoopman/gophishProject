-- +goose Up
-- Cyber Hygiene: My Apps & Devices module

CREATE TABLE IF NOT EXISTS "user_devices" (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    org_id INTEGER NOT NULL,
    name VARCHAR(255) NOT NULL DEFAULT '',
    device_type VARCHAR(20) NOT NULL DEFAULT 'other',
    os VARCHAR(100) NOT NULL DEFAULT '',
    hygiene_score INTEGER NOT NULL DEFAULT 0,
    created_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    modified_date DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS "device_hygiene_checks" (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    device_id INTEGER NOT NULL,
    check_type VARCHAR(50) NOT NULL,
    status VARCHAR(10) NOT NULL DEFAULT 'unknown',
    note TEXT NOT NULL DEFAULT '',
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(device_id, check_type)
);

CREATE INDEX IF NOT EXISTS idx_user_devices_user ON user_devices(user_id);
CREATE INDEX IF NOT EXISTS idx_user_devices_org ON user_devices(org_id);
CREATE INDEX IF NOT EXISTS idx_device_checks_device ON device_hygiene_checks(device_id);

-- Enable cyber_hygiene for Starter (id=2), All-In-One (id=3), and MSP (id=5) tiers
INSERT OR IGNORE INTO tier_features (tier_id, feature_slug) VALUES (2, 'cyber_hygiene');
INSERT OR IGNORE INTO tier_features (tier_id, feature_slug) VALUES (3, 'cyber_hygiene');
INSERT OR IGNORE INTO tier_features (tier_id, feature_slug) VALUES (5, 'cyber_hygiene');

-- +goose Down
DROP TABLE IF EXISTS "device_hygiene_checks";
DROP TABLE IF EXISTS "user_devices";
