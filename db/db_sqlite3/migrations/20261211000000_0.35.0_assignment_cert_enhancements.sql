-- +goose Up
-- Assignment enhancements: priority, reminders, escalation, notes, completion tracking
ALTER TABLE course_assignments ADD COLUMN priority VARCHAR(20) NOT NULL DEFAULT 'normal';
ALTER TABLE course_assignments ADD COLUMN reminder_sent BOOLEAN NOT NULL DEFAULT 0;
ALTER TABLE course_assignments ADD COLUMN reminder_date DATETIME;
ALTER TABLE course_assignments ADD COLUMN escalated_to INTEGER NOT NULL DEFAULT 0;
ALTER TABLE course_assignments ADD COLUMN escalated_date DATETIME;
ALTER TABLE course_assignments ADD COLUMN completed_date DATETIME;
ALTER TABLE course_assignments ADD COLUMN notes TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_assignment_status ON course_assignments(status);
CREATE INDEX IF NOT EXISTS idx_assignment_priority ON course_assignments(priority);
CREATE INDEX IF NOT EXISTS idx_assignment_due_date ON course_assignments(due_date);
CREATE INDEX IF NOT EXISTS idx_assignment_reminder ON course_assignments(reminder_sent, due_date);

-- Certificate enhancements: templates, expiry, revocation, metadata
ALTER TABLE certificates ADD COLUMN template_slug VARCHAR(100) NOT NULL DEFAULT 'cybersecurity-awareness-foundation';
ALTER TABLE certificates ADD COLUMN expires_date DATETIME;
ALTER TABLE certificates ADD COLUMN revoked_date DATETIME;
ALTER TABLE certificates ADD COLUMN is_revoked BOOLEAN NOT NULL DEFAULT 0;
ALTER TABLE certificates ADD COLUMN metadata TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_cert_template ON certificates(template_slug);
CREATE INDEX IF NOT EXISTS idx_cert_expires ON certificates(expires_date);
CREATE INDEX IF NOT EXISTS idx_cert_revoked ON certificates(is_revoked);

-- +goose Down
-- SQLite does not support DROP COLUMN, so we drop indexes only
DROP INDEX IF EXISTS idx_assignment_status;
DROP INDEX IF EXISTS idx_assignment_priority;
DROP INDEX IF EXISTS idx_assignment_due_date;
DROP INDEX IF EXISTS idx_assignment_reminder;
DROP INDEX IF EXISTS idx_cert_template;
DROP INDEX IF EXISTS idx_cert_expires;
DROP INDEX IF EXISTS idx_cert_revoked;
