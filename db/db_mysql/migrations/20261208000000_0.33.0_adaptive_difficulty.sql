-- +goose Up
-- Add adaptive difficulty preference columns to users table.
ALTER TABLE `users` ADD COLUMN `training_difficulty_mode` VARCHAR(20) DEFAULT 'adaptive';
ALTER TABLE `users` ADD COLUMN `training_difficulty_manual` INT DEFAULT 0;

-- Difficulty adjustment history log for audit trail and trend analysis.
CREATE TABLE IF NOT EXISTS `difficulty_adjustment_log` (
    `id` BIGINT AUTO_INCREMENT PRIMARY KEY,
    `user_id` BIGINT NOT NULL,
    `previous_level` INT NOT NULL,
    `new_level` INT NOT NULL,
    `source` VARCHAR(20) NOT NULL,
    `reason` TEXT NOT NULL,
    `brs_at_change` REAL DEFAULT 0,
    `click_rate_at_change` REAL DEFAULT 0,
    `created_date` DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
);

CREATE INDEX idx_difficulty_log_user ON difficulty_adjustment_log(user_id);
CREATE INDEX idx_difficulty_log_date ON difficulty_adjustment_log(created_date);

-- +goose Down
ALTER TABLE `users` DROP COLUMN `training_difficulty_mode`;
ALTER TABLE `users` DROP COLUMN `training_difficulty_manual`;
DROP TABLE IF EXISTS `difficulty_adjustment_log`;
