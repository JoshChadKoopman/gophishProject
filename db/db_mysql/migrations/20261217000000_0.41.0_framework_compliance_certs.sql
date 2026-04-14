-- +goose Up
-- Framework compliance certificates — org-level certs auto-issued when
-- an organization meets the requirements for a compliance framework.

CREATE TABLE IF NOT EXISTS org_framework_certs (
    id INTEGER PRIMARY KEY AUTO_INCREMENT,
    org_id INTEGER NOT NULL,
    cert_slug VARCHAR(100) NOT NULL,
    framework_slug VARCHAR(50) NOT NULL,
    verification_code VARCHAR(64) NOT NULL UNIQUE,
    framework_score DOUBLE DEFAULT 0,
    controls_passed INTEGER DEFAULT 0,
    total_controls INTEGER DEFAULT 0,
    issued_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_date DATETIME NOT NULL,
    is_revoked BOOLEAN DEFAULT 0,
    revoked_date DATETIME
);

CREATE INDEX idx_org_framework_certs_org ON org_framework_certs(org_id);
CREATE INDEX idx_org_framework_certs_slug ON org_framework_certs(cert_slug);
CREATE INDEX idx_org_framework_certs_code ON org_framework_certs(verification_code);
CREATE INDEX idx_org_framework_certs_active ON org_framework_certs(org_id, is_revoked, expires_date);

-- +goose Down
DROP TABLE IF EXISTS org_framework_certs;
