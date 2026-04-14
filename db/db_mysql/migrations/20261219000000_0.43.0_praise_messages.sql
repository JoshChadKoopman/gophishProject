-- +goose Up
-- Configurable praise/feedback messages for training completion events.
CREATE TABLE IF NOT EXISTS praise_messages (
    id              BIGINT AUTO_INCREMENT PRIMARY KEY,
    org_id          BIGINT NOT NULL DEFAULT 0,
    event_type      VARCHAR(50) NOT NULL DEFAULT 'course_complete',
    heading         VARCHAR(255) NOT NULL DEFAULT 'Well Done!',
    body            TEXT NOT NULL,
    button_text     VARCHAR(100) NOT NULL DEFAULT 'OK',
    icon            VARCHAR(50) NOT NULL DEFAULT '⭐',
    color_scheme    VARCHAR(20) NOT NULL DEFAULT '#27ae60',
    is_active       BOOLEAN NOT NULL DEFAULT 1,
    modified_date   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_praise_messages_org_event (org_id, event_type)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Seed system-level defaults (org_id = 0)
INSERT IGNORE INTO praise_messages (org_id, event_type, heading, body, button_text, icon, color_scheme, is_active, modified_date)
VALUES
    (0, 'course_complete', 'Course Complete!', 'Congratulations! You finished <strong>{{.CourseName}}</strong>', 'Awesome!', '⭐', '#27ae60', 1, NOW()),
    (0, 'quiz_passed', 'Quiz Passed!', 'Great work! You scored {{.Score}}/{{.Total}} on <strong>{{.CourseName}}</strong>', 'Well Done!', '🏆', '#f39c12', 1, NOW()),
    (0, 'cert_earned', 'Certificate Earned!', 'You''ve earned the <strong>{{.CertName}}</strong> certificate. Keep up the great work!', 'View Certificate', '🎓', '#2c3e50', 1, NOW()),
    (0, 'tier_complete', 'Tier Completed!', 'Outstanding! You''ve completed the <strong>{{.TierName}}</strong> tier.', 'Continue Learning', '🏅', '#8e44ad', 1, NOW());

-- +goose Down
DROP TABLE IF EXISTS praise_messages;
