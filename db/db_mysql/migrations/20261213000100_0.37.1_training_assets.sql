-- +goose Up
-- Custom training builder (All-in-One tier): multi-module courses. Each
-- training presentation may have many assets (PDF/PPTX/video/image) that
-- make up ordered modules in a custom course.
CREATE TABLE IF NOT EXISTS `training_assets` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `presentation_id` BIGINT NOT NULL DEFAULT 0,
    `org_id` BIGINT NOT NULL DEFAULT 1,
    `title` VARCHAR(255) NOT NULL DEFAULT '',
    `description` TEXT,
    `file_name` VARCHAR(255) NOT NULL DEFAULT '',
    `file_path` VARCHAR(512) NOT NULL DEFAULT '',
    `file_size` BIGINT NOT NULL DEFAULT 0,
    `content_type` VARCHAR(255) NOT NULL DEFAULT '',
    `asset_type` VARCHAR(30) NOT NULL DEFAULT '',
    `sort_order` INT NOT NULL DEFAULT 0,
    `uploaded_by` BIGINT NOT NULL DEFAULT 0,
    `created_date` DATETIME,
    PRIMARY KEY (`id`),
    INDEX `idx_training_assets_pres` (`presentation_id`),
    INDEX `idx_training_assets_org` (`org_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

INSERT INTO `tier_features` (`tier_id`, `feature_slug`)
    SELECT `id`, 'custom_training_builder'
    FROM `subscription_tiers`
    WHERE `id` >= 3
    AND NOT EXISTS (
        SELECT 1 FROM `tier_features`
        WHERE `tier_id` = `subscription_tiers`.`id` AND `feature_slug` = 'custom_training_builder'
    );

-- +goose Down
DELETE FROM `tier_features` WHERE `feature_slug` = 'custom_training_builder';
DROP TABLE IF EXISTS `training_assets`;
