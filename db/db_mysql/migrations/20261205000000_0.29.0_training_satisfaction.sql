-- +goose Up
-- Phase 29: Training Satisfaction Ratings & Analytics

CREATE TABLE IF NOT EXISTS `training_satisfaction_ratings` (
    `id` BIGINT AUTO_INCREMENT PRIMARY KEY,
    `user_id` BIGINT NOT NULL,
    `presentation_id` BIGINT NOT NULL,
    `rating` INT NOT NULL,
    `feedback` TEXT NOT NULL,
    `created_date` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY `idx_satisfaction_user_pres` (`user_id`, `presentation_id`)
);

CREATE INDEX idx_satisfaction_pres ON training_satisfaction_ratings(presentation_id);

-- +goose Down
DROP TABLE IF EXISTS `training_satisfaction_ratings`;
