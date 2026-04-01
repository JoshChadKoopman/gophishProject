-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE TABLE IF NOT EXISTS `certificates` (
    `id` BIGINT AUTO_INCREMENT PRIMARY KEY,
    `user_id` BIGINT NOT NULL,
    `presentation_id` BIGINT NOT NULL,
    `quiz_attempt_id` BIGINT,
    `verification_code` VARCHAR(32) NOT NULL UNIQUE,
    `issued_date` DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE IF EXISTS `certificates`;
