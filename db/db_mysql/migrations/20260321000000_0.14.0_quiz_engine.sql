-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE TABLE IF NOT EXISTS `quizzes` (
    `id` BIGINT AUTO_INCREMENT PRIMARY KEY,
    `presentation_id` BIGINT NOT NULL UNIQUE,
    `pass_percentage` INT NOT NULL DEFAULT 70,
    `created_by` BIGINT NOT NULL,
    `created_date` DATETIME DEFAULT CURRENT_TIMESTAMP,
    `modified_date` DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `quiz_questions` (
    `id` BIGINT AUTO_INCREMENT PRIMARY KEY,
    `quiz_id` BIGINT NOT NULL,
    `question_text` TEXT NOT NULL,
    `options` TEXT NOT NULL,
    `correct_option` INT NOT NULL DEFAULT 0,
    `sort_order` INT NOT NULL DEFAULT 0,
    `created_date` DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

CREATE TABLE IF NOT EXISTS `quiz_attempts` (
    `id` BIGINT AUTO_INCREMENT PRIMARY KEY,
    `quiz_id` BIGINT NOT NULL,
    `user_id` BIGINT NOT NULL,
    `score` INT NOT NULL DEFAULT 0,
    `total_questions` INT NOT NULL DEFAULT 0,
    `passed` TINYINT(1) NOT NULL DEFAULT 0,
    `answers` TEXT,
    `completed_date` DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

CREATE INDEX `idx_quiz_attempts_user` ON `quiz_attempts`(`user_id`, `quiz_id`);

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE IF EXISTS `quiz_attempts`;
DROP TABLE IF EXISTS `quiz_questions`;
DROP TABLE IF EXISTS `quizzes`;
