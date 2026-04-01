-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE TABLE IF NOT EXISTS `course_assignments` (
    `id` BIGINT AUTO_INCREMENT PRIMARY KEY,
    `user_id` BIGINT NOT NULL,
    `presentation_id` BIGINT NOT NULL,
    `assigned_by` BIGINT NOT NULL,
    `group_id` BIGINT,
    `campaign_id` BIGINT,
    `due_date` DATETIME,
    `status` VARCHAR(50) NOT NULL DEFAULT 'pending',
    `created_date` DATETIME DEFAULT CURRENT_TIMESTAMP,
    `modified_date` DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB;

CREATE UNIQUE INDEX `idx_assignment_user_pres` ON `course_assignments`(`user_id`, `presentation_id`);

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE IF EXISTS `course_assignments`;
