-- +goose Up
-- 0.47.0: ROI Dashboard, AI Translation, Compliance Module Progress,
--         Sending Domain Pool, and Enhanced Training Reminders

-- ── ROI Investment Config ──
CREATE TABLE IF NOT EXISTS `roi_investment_configs` (
    `id`               BIGINT AUTO_INCREMENT PRIMARY KEY,
    `org_id`           BIGINT NOT NULL UNIQUE,
    `phishing_sim_pct` DOUBLE DEFAULT 30,
    `training_pct`     DOUBLE DEFAULT 25,
    `tooling_pct`      DOUBLE DEFAULT 25,
    `personnel_pct`    DOUBLE DEFAULT 20,
    INDEX idx_roi_inv_org (`org_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ── AI Translation Tables ──
CREATE TABLE IF NOT EXISTS `translation_requests` (
    `id`            BIGINT AUTO_INCREMENT PRIMARY KEY,
    `org_id`        BIGINT NOT NULL,
    `user_id`       BIGINT NOT NULL,
    `content_type`  VARCHAR(64) NOT NULL,
    `content_id`    BIGINT NOT NULL,
    `source_lang`   VARCHAR(10) NOT NULL,
    `target_lang`   VARCHAR(10) NOT NULL,
    `status`        VARCHAR(32) DEFAULT 'pending',
    `input_tokens`  INT DEFAULT 0,
    `output_tokens` INT DEFAULT 0,
    `created_date`  DATETIME,
    `completed_at`  DATETIME,
    INDEX idx_trans_req_org (`org_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `translated_contents` (
    `id`               BIGINT AUTO_INCREMENT PRIMARY KEY,
    `org_id`           BIGINT NOT NULL,
    `content_type`     VARCHAR(64) NOT NULL,
    `content_id`       BIGINT NOT NULL,
    `source_lang`      VARCHAR(10) NOT NULL,
    `target_lang`      VARCHAR(10) NOT NULL,
    `translated_title` TEXT,
    `translated_body`  LONGTEXT,
    `translated_html`  LONGTEXT,
    `quality`          DOUBLE DEFAULT 0,
    `reviewed_by`      BIGINT,
    `reviewed_at`      DATETIME,
    `is_approved`      TINYINT(1) DEFAULT 0,
    `created_date`     DATETIME,
    `modified_date`    DATETIME,
    INDEX idx_trans_content_org (`org_id`),
    INDEX idx_trans_content_lookup (`content_type`, `content_id`, `target_lang`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `translation_configs` (
    `id`                 BIGINT AUTO_INCREMENT PRIMARY KEY,
    `org_id`             BIGINT NOT NULL UNIQUE,
    `enabled`            TINYINT(1) DEFAULT 1,
    `auto_translate`     TINYINT(1) DEFAULT 0,
    `default_langs`      TEXT,
    `review_required`    TINYINT(1) DEFAULT 1,
    `max_monthly_tokens` INT DEFAULT 500000,
    INDEX idx_trans_cfg_org (`org_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ── Compliance Module Progress & Assignments ──
CREATE TABLE IF NOT EXISTS `compliance_module_progress` (
    `id`              BIGINT AUTO_INCREMENT PRIMARY KEY,
    `user_id`         BIGINT NOT NULL,
    `org_id`          BIGINT NOT NULL,
    `module_slug`     VARCHAR(128) NOT NULL,
    `status`          VARCHAR(32) DEFAULT 'pending',
    `current_page`    INT DEFAULT 0,
    `quiz_score`      INT DEFAULT 0,
    `passed`          TINYINT(1) DEFAULT 0,
    `attempts_count`  INT DEFAULT 0,
    `time_spent_secs` INT DEFAULT 0,
    `started_date`    DATETIME,
    `completed_date`  DATETIME,
    `created_date`    DATETIME,
    INDEX idx_comp_prog_user (`user_id`),
    INDEX idx_comp_prog_org (`org_id`),
    UNIQUE KEY idx_comp_prog_user_module (`user_id`, `module_slug`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `compliance_module_assignments` (
    `id`           BIGINT AUTO_INCREMENT PRIMARY KEY,
    `org_id`       BIGINT NOT NULL,
    `module_slug`  VARCHAR(128) NOT NULL,
    `user_id`      BIGINT,
    `group_id`     BIGINT,
    `assigned_by`  BIGINT NOT NULL,
    `due_date`     DATETIME,
    `is_required`  TINYINT(1) DEFAULT 1,
    `created_date` DATETIME,
    INDEX idx_comp_assign_org (`org_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ── Sending Domain Pool ──
CREATE TABLE IF NOT EXISTS `sending_domains` (
    `id`                BIGINT AUTO_INCREMENT PRIMARY KEY,
    `org_id`            BIGINT NOT NULL,
    `domain`            VARCHAR(255) NOT NULL,
    `display_name`      VARCHAR(255),
    `category`          VARCHAR(64) DEFAULT 'custom',
    `is_built_in`       TINYINT(1) DEFAULT 0,
    `is_active`         TINYINT(1) DEFAULT 1,
    `spf_configured`    TINYINT(1) DEFAULT 0,
    `dkim_configured`   TINYINT(1) DEFAULT 0,
    `dmarc_configured`  TINYINT(1) DEFAULT 0,
    `warmup_stage`      INT DEFAULT 0,
    `daily_limit`       INT DEFAULT 50,
    `sends_today`       INT DEFAULT 0,
    `total_sent`        BIGINT DEFAULT 0,
    `last_used_date`    DATETIME,
    `health_status`     VARCHAR(32) DEFAULT 'unknown',
    `last_health_check` DATETIME,
    `notes`             TEXT,
    `created_date`      DATETIME,
    `modified_date`     DATETIME,
    INDEX idx_send_dom_org (`org_id`),
    UNIQUE KEY idx_send_dom_org_domain (`org_id`, `domain`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `domain_pool_configs` (
    `id`                   BIGINT AUTO_INCREMENT PRIMARY KEY,
    `org_id`               BIGINT NOT NULL UNIQUE,
    `enabled`              TINYINT(1) DEFAULT 1,
    `auto_rotate`          TINYINT(1) DEFAULT 1,
    `rotation_strategy`    VARCHAR(32) DEFAULT 'round_robin',
    `max_daily_per_domain` INT DEFAULT 50,
    `warmup_enabled`       TINYINT(1) DEFAULT 1,
    INDEX idx_dom_pool_cfg_org (`org_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- ── Reminder Templates (Enhanced Reminders) ──
CREATE TABLE IF NOT EXISTS `reminder_templates` (
    `id`        BIGINT AUTO_INCREMENT PRIMARY KEY,
    `org_id`    BIGINT NOT NULL UNIQUE,
    `subject`   TEXT,
    `body_html` LONGTEXT,
    `body_text` LONGTEXT,
    `is_custom` TINYINT(1) DEFAULT 0,
    INDEX idx_rem_tpl_org (`org_id`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;


-- +goose Down
DROP TABLE IF EXISTS `reminder_templates`;
DROP TABLE IF EXISTS `domain_pool_configs`;
DROP TABLE IF EXISTS `sending_domains`;
DROP TABLE IF EXISTS `compliance_module_assignments`;
DROP TABLE IF EXISTS `compliance_module_progress`;
DROP TABLE IF EXISTS `translation_configs`;
DROP TABLE IF EXISTS `translated_contents`;
DROP TABLE IF EXISTS `translation_requests`;
DROP TABLE IF EXISTS `roi_investment_configs`;
