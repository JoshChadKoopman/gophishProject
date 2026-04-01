-- +goose Up

CREATE TABLE IF NOT EXISTS organizations (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    name          VARCHAR(255) NOT NULL,
    slug          VARCHAR(100) NOT NULL UNIQUE,
    tier          VARCHAR(50) NOT NULL DEFAULT 'free',
    max_users     INTEGER NOT NULL DEFAULT 10,
    max_campaigns INTEGER NOT NULL DEFAULT 5,
    logo_url      TEXT DEFAULT '',
    primary_color VARCHAR(20) DEFAULT '#007bff',
    created_date  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Seed the default organization (all existing data gets org_id=1)
INSERT INTO organizations (id, name, slug, tier, max_users, max_campaigns)
VALUES (1, 'Default', 'default', 'enterprise', 999999, 999999);

-- Add org_id to users
ALTER TABLE users ADD COLUMN org_id INTEGER NOT NULL DEFAULT 1;
CREATE INDEX idx_users_org_id ON users(org_id);

-- Add org_id to core tables
ALTER TABLE campaigns ADD COLUMN org_id INTEGER NOT NULL DEFAULT 1;
CREATE INDEX idx_campaigns_org_id ON campaigns(org_id);

ALTER TABLE "groups" ADD COLUMN org_id INTEGER NOT NULL DEFAULT 1;
CREATE INDEX idx_groups_org_id ON "groups"(org_id);

ALTER TABLE templates ADD COLUMN org_id INTEGER NOT NULL DEFAULT 1;
CREATE INDEX idx_templates_org_id ON templates(org_id);

ALTER TABLE pages ADD COLUMN org_id INTEGER NOT NULL DEFAULT 1;
CREATE INDEX idx_pages_org_id ON pages(org_id);

ALTER TABLE smtp ADD COLUMN org_id INTEGER NOT NULL DEFAULT 1;
CREATE INDEX idx_smtp_org_id ON smtp(org_id);

-- Add org_id to training tables
ALTER TABLE training_presentations ADD COLUMN org_id INTEGER NOT NULL DEFAULT 1;
CREATE INDEX idx_training_presentations_org_id ON training_presentations(org_id);

ALTER TABLE course_assignments ADD COLUMN org_id INTEGER NOT NULL DEFAULT 1;
CREATE INDEX idx_course_assignments_org_id ON course_assignments(org_id);

ALTER TABLE course_progress ADD COLUMN org_id INTEGER NOT NULL DEFAULT 1;
CREATE INDEX idx_course_progress_org_id ON course_progress(org_id);

ALTER TABLE certificates ADD COLUMN org_id INTEGER NOT NULL DEFAULT 1;
CREATE INDEX idx_certificates_org_id ON certificates(org_id);

-- Add org_id to audit_logs
ALTER TABLE audit_logs ADD COLUMN org_id INTEGER NOT NULL DEFAULT 1;
CREATE INDEX idx_audit_logs_org_id ON audit_logs(org_id);

-- +goose Down
DROP TABLE IF EXISTS organizations;
