-- +goose Up
-- Zero Incident Mail (ZIM) sandbox tests table

CREATE TABLE IF NOT EXISTS sandbox_tests (
    id INTEGER PRIMARY KEY AUTO_INCREMENT,
    org_id INTEGER NOT NULL,
    created_by INTEGER NOT NULL DEFAULT 0,
    template_id INTEGER NOT NULL DEFAULT 0,
    smtp_id INTEGER NOT NULL DEFAULT 0,
    to_email VARCHAR(255) NOT NULL DEFAULT '',
    subject VARCHAR(500) NOT NULL DEFAULT '',
    rendered_html MEDIUMTEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    error_msg TEXT NOT NULL,
    notes TEXT NOT NULL,
    sent_at DATETIME,
    reviewed_at DATETIME,
    reviewed_by INTEGER NOT NULL DEFAULT 0,
    created_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    modified_date DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_sandbox_tests_org ON sandbox_tests(org_id);
CREATE INDEX idx_sandbox_tests_status ON sandbox_tests(org_id, status);

-- Enable ZIM for All-In-One (id=3) and MSP (id=5) tiers
INSERT IGNORE INTO tier_features (tier_id, feature_slug) VALUES (3, 'zim');
INSERT IGNORE INTO tier_features (tier_id, feature_slug) VALUES (5, 'zim');

-- +goose Down
DROP TABLE IF EXISTS sandbox_tests;
