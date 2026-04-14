-- +goose Up
-- Assignment enhancements: priority, reminders, escalation, notes, completion tracking
ALTER TABLE course_assignments ADD COLUMN priority VARCHAR(20) NOT NULL DEFAULT 'normal';
ALTER TABLE course_assignments ADD COLUMN reminder_sent TINYINT(1) NOT NULL DEFAULT 0;
ALTER TABLE course_assignments ADD COLUMN reminder_date DATETIME NULL;
ALTER TABLE course_assignments ADD COLUMN escalated_to BIGINT NOT NULL DEFAULT 0;
ALTER TABLE course_assignments ADD COLUMN escalated_date DATETIME NULL;
ALTER TABLE course_assignments ADD COLUMN completed_date DATETIME NULL;
ALTER TABLE course_assignments ADD COLUMN notes TEXT NOT NULL DEFAULT '';

CREATE INDEX idx_assignment_status ON course_assignments(status);
CREATE INDEX idx_assignment_priority ON course_assignments(priority);
CREATE INDEX idx_assignment_due_date ON course_assignments(due_date);
CREATE INDEX idx_assignment_reminder ON course_assignments(reminder_sent, due_date);

-- Certificate enhancements: templates, expiry, revocation, metadata
ALTER TABLE certificates ADD COLUMN template_slug VARCHAR(100) NOT NULL DEFAULT 'cybersecurity-awareness-foundation';
ALTER TABLE certificates ADD COLUMN expires_date DATETIME NULL;
ALTER TABLE certificates ADD COLUMN revoked_date DATETIME NULL;
ALTER TABLE certificates ADD COLUMN is_revoked TINYINT(1) NOT NULL DEFAULT 0;
ALTER TABLE certificates ADD COLUMN metadata TEXT NOT NULL DEFAULT '';

CREATE INDEX idx_cert_template ON certificates(template_slug);
CREATE INDEX idx_cert_expires ON certificates(expires_date);
CREATE INDEX idx_cert_revoked ON certificates(is_revoked);

-- +goose Down
DROP INDEX idx_assignment_status ON course_assignments;
DROP INDEX idx_assignment_priority ON course_assignments;
DROP INDEX idx_assignment_due_date ON course_assignments;
DROP INDEX idx_assignment_reminder ON course_assignments;
ALTER TABLE course_assignments DROP COLUMN priority;
ALTER TABLE course_assignments DROP COLUMN reminder_sent;
ALTER TABLE course_assignments DROP COLUMN reminder_date;
ALTER TABLE course_assignments DROP COLUMN escalated_to;
ALTER TABLE course_assignments DROP COLUMN escalated_date;
ALTER TABLE course_assignments DROP COLUMN completed_date;
ALTER TABLE course_assignments DROP COLUMN notes;
DROP INDEX idx_cert_template ON certificates;
DROP INDEX idx_cert_expires ON certificates;
DROP INDEX idx_cert_revoked ON certificates;
ALTER TABLE certificates DROP COLUMN template_slug;
ALTER TABLE certificates DROP COLUMN expires_date;
ALTER TABLE certificates DROP COLUMN revoked_date;
ALTER TABLE certificates DROP COLUMN is_revoked;
ALTER TABLE certificates DROP COLUMN metadata;
