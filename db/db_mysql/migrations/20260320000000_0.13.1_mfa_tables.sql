
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

CREATE TABLE IF NOT EXISTS `mfa_devices` (
    `id`          BIGINT NOT NULL AUTO_INCREMENT,
    `user_id`     BIGINT NOT NULL,
    `totp_secret` VARCHAR(512) NOT NULL,
    `enabled`     TINYINT(1) NOT NULL DEFAULT 0,
    `enrolled_at` DATETIME,
    PRIMARY KEY (`id`),
    UNIQUE KEY `uq_mfa_devices_user` (`user_id`),
    CONSTRAINT `fk_mfa_devices_user` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `mfa_backup_codes` (
    `id`         BIGINT NOT NULL AUTO_INCREMENT,
    `user_id`    BIGINT NOT NULL,
    `code_hash`  VARCHAR(255) NOT NULL,
    `used`       TINYINT(1) NOT NULL DEFAULT 0,
    `used_at`    DATETIME,
    `created_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    INDEX `idx_mfa_backup_codes_user` (`user_id`),
    CONSTRAINT `fk_mfa_backup_user` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `mfa_attempts` (
    `id`           BIGINT NOT NULL AUTO_INCREMENT,
    `user_id`      BIGINT NOT NULL,
    `attempted_at` DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `success`      TINYINT(1) NOT NULL DEFAULT 0,
    `ip_address`   VARCHAR(45),
    PRIMARY KEY (`id`),
    INDEX `idx_mfa_attempts_user_time` (`user_id`, `attempted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `device_fingerprints` (
    `id`               BIGINT NOT NULL AUTO_INCREMENT,
    `user_id`          BIGINT NOT NULL,
    `fingerprint_hash` VARCHAR(255) NOT NULL,
    `expires_at`       DATETIME NOT NULL,
    `created_at`       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (`id`),
    INDEX `idx_device_fingerprints_user` (`user_id`),
    CONSTRAINT `fk_device_fp_user` FOREIGN KEY (`user_id`) REFERENCES `users`(`id`) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

DROP TABLE IF EXISTS `device_fingerprints`;
DROP TABLE IF EXISTS `mfa_attempts`;
DROP TABLE IF EXISTS `mfa_backup_codes`;
DROP TABLE IF EXISTS `mfa_devices`;
