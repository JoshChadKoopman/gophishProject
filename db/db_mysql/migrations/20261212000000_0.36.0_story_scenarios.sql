-- +goose Up
-- Branching narrative training: interactive "choose-your-own-adventure" scenarios
-- for immersive cybersecurity decision training (phished.io parity feature).

CREATE TABLE IF NOT EXISTS `story_scenarios` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `presentation_id` BIGINT NOT NULL DEFAULT 0,
    `title` VARCHAR(255) NOT NULL DEFAULT '',
    `description` TEXT,
    `category` VARCHAR(100) NOT NULL DEFAULT '',
    `difficulty` INT NOT NULL DEFAULT 1,
    `start_node_id` BIGINT NOT NULL DEFAULT 0,
    `pass_threshold` INT NOT NULL DEFAULT 70,
    `created_by` BIGINT NOT NULL DEFAULT 0,
    `created_date` DATETIME,
    `modified_date` DATETIME,
    PRIMARY KEY (`id`),
    INDEX `idx_story_presentation` (`presentation_id`),
    INDEX `idx_story_category` (`category`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `story_nodes` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `scenario_id` BIGINT NOT NULL DEFAULT 0,
    `node_key` VARCHAR(100) NOT NULL DEFAULT '',
    `node_type` VARCHAR(30) NOT NULL DEFAULT 'choice',
    `title` VARCHAR(255) NOT NULL DEFAULT '',
    `body` TEXT,
    `media_url` VARCHAR(500) NOT NULL DEFAULT '',
    `score_delta` INT NOT NULL DEFAULT 0,
    `is_terminal` TINYINT(1) NOT NULL DEFAULT 0,
    `outcome` VARCHAR(30) NOT NULL DEFAULT '',
    PRIMARY KEY (`id`),
    INDEX `idx_story_nodes_scenario` (`scenario_id`),
    UNIQUE KEY `idx_story_nodes_key` (`scenario_id`, `node_key`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `story_choices` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `node_id` BIGINT NOT NULL DEFAULT 0,
    `label` VARCHAR(500) NOT NULL DEFAULT '',
    `next_node_id` BIGINT NOT NULL DEFAULT 0,
    `score_delta` INT NOT NULL DEFAULT 0,
    `feedback` TEXT,
    `is_correct` TINYINT(1) NOT NULL DEFAULT 0,
    `sort_order` INT NOT NULL DEFAULT 0,
    PRIMARY KEY (`id`),
    INDEX `idx_story_choices_node` (`node_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `story_progress` (
    `id` BIGINT NOT NULL AUTO_INCREMENT,
    `user_id` BIGINT NOT NULL DEFAULT 0,
    `scenario_id` BIGINT NOT NULL DEFAULT 0,
    `current_node_id` BIGINT NOT NULL DEFAULT 0,
    `score` INT NOT NULL DEFAULT 0,
    `path` TEXT,
    `status` VARCHAR(30) NOT NULL DEFAULT 'in_progress',
    `started_date` DATETIME,
    `completed_date` DATETIME,
    PRIMARY KEY (`id`),
    INDEX `idx_story_progress_user` (`user_id`, `scenario_id`),
    INDEX `idx_story_progress_status` (`status`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS `story_progress`;
DROP TABLE IF EXISTS `story_choices`;
DROP TABLE IF EXISTS `story_nodes`;
DROP TABLE IF EXISTS `story_scenarios`;
