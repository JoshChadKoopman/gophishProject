-- +goose Up
-- AI Admin Assistant ("Aria" equivalent): guided onboarding + conversational
-- platform navigation.

CREATE TABLE IF NOT EXISTS assistant_conversations (
    id            BIGINT AUTO_INCREMENT PRIMARY KEY,
    org_id        BIGINT NOT NULL,
    user_id       BIGINT NOT NULL,
    title         VARCHAR(255) DEFAULT '',
    created_date  DATETIME DEFAULT CURRENT_TIMESTAMP,
    modified_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_assistant_conversations_user (org_id, user_id),
    INDEX idx_assistant_conversations_modified (modified_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS assistant_messages (
    id              BIGINT AUTO_INCREMENT PRIMARY KEY,
    conversation_id BIGINT NOT NULL,
    role            VARCHAR(20) NOT NULL DEFAULT 'admin',
    content         TEXT,
    tokens_used     INT DEFAULT 0,
    created_date    DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_assistant_messages_conv (conversation_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS admin_onboarding_progress (
    id             BIGINT AUTO_INCREMENT PRIMARY KEY,
    org_id         BIGINT NOT NULL,
    user_id        BIGINT NOT NULL,
    step           VARCHAR(64) NOT NULL DEFAULT '',
    completed_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY idx_admin_onboarding_unique (org_id, user_id, step)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS admin_onboarding_progress;
DROP TABLE IF EXISTS assistant_messages;
DROP TABLE IF EXISTS assistant_conversations;
