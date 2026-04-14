-- +goose Up
-- Configurable praise/feedback messages for training completion events.
CREATE TABLE IF NOT EXISTS praise_messages (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id          INTEGER NOT NULL DEFAULT 0,
    event_type      VARCHAR(50) NOT NULL DEFAULT 'course_complete',
    heading         VARCHAR(255) NOT NULL DEFAULT 'Well Done!',
    body            TEXT NOT NULL DEFAULT '',
    button_text     VARCHAR(100) NOT NULL DEFAULT 'OK',
    icon            VARCHAR(50) NOT NULL DEFAULT '⭐',
    color_scheme    VARCHAR(20) NOT NULL DEFAULT '#27ae60',
    is_active       BOOLEAN NOT NULL DEFAULT 1,
    modified_date   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Index for fast lookup by org + event type
CREATE INDEX IF NOT EXISTS idx_praise_messages_org_event ON praise_messages(org_id, event_type);

-- Seed system-level defaults (org_id = 0)
INSERT OR IGNORE INTO praise_messages (org_id, event_type, heading, body, button_text, icon, color_scheme, is_active, modified_date)
VALUES
    (0, 'course_complete', 'Course Complete!', 'Congratulations! You finished <strong>{{.CourseName}}</strong>', 'Awesome!', '⭐', '#27ae60', 1, CURRENT_TIMESTAMP),
    (0, 'quiz_passed', 'Quiz Passed!', 'Great work! You scored {{.Score}}/{{.Total}} on <strong>{{.CourseName}}</strong>', 'Well Done!', '🏆', '#f39c12', 1, CURRENT_TIMESTAMP),
    (0, 'cert_earned', 'Certificate Earned!', 'You''ve earned the <strong>{{.CertName}}</strong> certificate. Keep up the great work!', 'View Certificate', '🎓', '#2c3e50', 1, CURRENT_TIMESTAMP),
    (0, 'tier_complete', 'Tier Completed!', 'Outstanding! You''ve completed the <strong>{{.TierName}}</strong> tier.', 'Continue Learning', '🏅', '#8e44ad', 1, CURRENT_TIMESTAMP);

-- +goose Down
DROP TABLE IF EXISTS praise_messages;
