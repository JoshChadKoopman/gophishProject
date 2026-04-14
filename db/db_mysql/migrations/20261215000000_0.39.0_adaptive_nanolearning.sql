-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied.

-- Adaptive engine per-org configuration
CREATE TABLE IF NOT EXISTS `adaptive_engine_configs` (
    `id`                      BIGINT AUTO_INCREMENT PRIMARY KEY,
    `org_id`                  BIGINT NOT NULL UNIQUE,
    `enabled`                 BOOLEAN DEFAULT TRUE,
    `eval_interval_days`      INT DEFAULT 7,
    `brs_weight_pct`          DOUBLE DEFAULT 40.0,
    `click_rate_weight_pct`   DOUBLE DEFAULT 30.0,
    `quiz_score_weight_pct`   DOUBLE DEFAULT 20.0,
    `trend_weight_pct`        DOUBLE DEFAULT 10.0,
    `promote_threshold`       DOUBLE DEFAULT 75.0,
    `demote_threshold`        DOUBLE DEFAULT 35.0,
    `min_simulations_promote` INT DEFAULT 3,
    `cooldown_days`           INT DEFAULT 14,
    `modified_date`           DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Adaptive engine run audit log
CREATE TABLE IF NOT EXISTS `adaptive_engine_run_logs` (
    `id`              BIGINT AUTO_INCREMENT PRIMARY KEY,
    `org_id`          BIGINT NOT NULL,
    `users_evaluated` INT DEFAULT 0,
    `promoted`        INT DEFAULT 0,
    `demoted`         INT DEFAULT 0,
    `maintained`      INT DEFAULT 0,
    `skipped`         INT DEFAULT 0,
    `run_date`        DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX `idx_ae_run_logs_org_date` (`org_id`, `run_date`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Nanolearning events (micro-intervention tracking)
CREATE TABLE IF NOT EXISTS `nanolearning_events` (
    `id`           BIGINT AUTO_INCREMENT PRIMARY KEY,
    `user_id`      BIGINT DEFAULT 0,
    `email`        VARCHAR(255) NOT NULL,
    `campaign_id`  BIGINT NOT NULL,
    `result_id`    VARCHAR(255) DEFAULT '',
    `content_slug` VARCHAR(255) DEFAULT '',
    `tip_text`     TEXT,
    `category`     VARCHAR(100) DEFAULT '',
    `acknowledged` BOOLEAN DEFAULT FALSE,
    `created_date` DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX `idx_nano_events_user` (`user_id`),
    INDEX `idx_nano_events_email_campaign` (`email`, `campaign_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ROI configuration per org
CREATE TABLE IF NOT EXISTS `roi_configs` (
    `id`               BIGINT AUTO_INCREMENT PRIMARY KEY,
    `org_id`           BIGINT NOT NULL UNIQUE,
    `program_cost`     DOUBLE DEFAULT 50000.0,
    `avg_breach_cost`  DOUBLE DEFAULT 4450000.0,
    `avg_incident_cost` DOUBLE DEFAULT 1500.0,
    `employee_count`   INT DEFAULT 200,
    `avg_salary_hr`    DOUBLE DEFAULT 45.0,
    `currency`         VARCHAR(10) DEFAULT 'USD',
    `modified_date`    DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Training reminders
CREATE TABLE IF NOT EXISTS `training_reminders` (
    `id`              BIGINT AUTO_INCREMENT PRIMARY KEY,
    `user_id`         BIGINT NOT NULL,
    `assignment_id`   BIGINT NOT NULL,
    `presentation_id` BIGINT DEFAULT 0,
    `course_name`     VARCHAR(255) DEFAULT '',
    `due_date`        DATETIME,
    `reminder_type`   VARCHAR(50) DEFAULT 'standard',
    `message`         TEXT,
    `sent_date`       DATETIME DEFAULT CURRENT_TIMESTAMP,
    INDEX `idx_training_reminders_user` (`user_id`),
    INDEX `idx_training_reminders_assignment` (`assignment_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
-- SQL in section 'Down' is executed when this migration is rolled back.
DROP TABLE IF EXISTS `training_reminders`;
DROP TABLE IF EXISTS `roi_configs`;
DROP TABLE IF EXISTS `nanolearning_events`;
DROP TABLE IF EXISTS `adaptive_engine_run_logs`;
DROP TABLE IF EXISTS `adaptive_engine_configs`;
