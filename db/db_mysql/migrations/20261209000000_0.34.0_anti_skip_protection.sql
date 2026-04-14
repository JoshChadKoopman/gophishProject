-- +goose Up
-- Per-page engagement evidence for anti-skip protection.
CREATE TABLE IF NOT EXISTS `page_engagement` (
    `id` BIGINT AUTO_INCREMENT PRIMARY KEY,
    `user_id` BIGINT NOT NULL,
    `presentation_id` BIGINT NOT NULL,
    `page_index` INT NOT NULL,
    `entered_at` DATETIME NOT NULL,
    `dwell_seconds` INT DEFAULT 0,
    `scroll_depth_pct` INT DEFAULT 0,
    `interaction_type` VARCHAR(30) DEFAULT '',
    `acknowledged` BOOLEAN DEFAULT 0,
    `created_date` DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE,
    UNIQUE KEY `idx_page_engagement_unique` (`user_id`, `presentation_id`, `page_index`)
);

CREATE INDEX idx_page_engagement_user_pres ON page_engagement(user_id, presentation_id);

-- Anti-skip policy per-presentation (admins can customise).
CREATE TABLE IF NOT EXISTS `anti_skip_policy` (
    `id` BIGINT AUTO_INCREMENT PRIMARY KEY,
    `presentation_id` BIGINT NOT NULL UNIQUE,
    `min_dwell_seconds` INT DEFAULT 10,
    `require_acknowledge` BOOLEAN DEFAULT 0,
    `require_scroll` BOOLEAN DEFAULT 0,
    `min_scroll_depth_pct` INT DEFAULT 80,
    `enforce_sequential` BOOLEAN DEFAULT 1,
    `allow_back_navigation` BOOLEAN DEFAULT 1,
    `created_date` DATETIME DEFAULT CURRENT_TIMESTAMP,
    `modified_date` DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (`presentation_id`) REFERENCES `training_presentations`(`id`) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE IF EXISTS `page_engagement`;
DROP TABLE IF EXISTS `anti_skip_policy`;
