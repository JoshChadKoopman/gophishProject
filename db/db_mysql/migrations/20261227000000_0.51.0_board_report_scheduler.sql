-- +goose Up
-- Board report schedules for auto-generation

CREATE TABLE IF NOT EXISTS board_report_schedules (
    id              BIGINT AUTO_INCREMENT PRIMARY KEY,
    org_id          BIGINT NOT NULL DEFAULT 0,
    frequency       VARCHAR(20) NOT NULL DEFAULT 'monthly',
    day_of_month    INT NOT NULL DEFAULT 1,
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    auto_publish    BOOLEAN NOT NULL DEFAULT FALSE,
    notify_emails   TEXT NOT NULL,
    created_by      BIGINT NOT NULL DEFAULT 0,
    last_run_date   DATETIME NULL,
    next_run_date   DATETIME NULL,
    created_date    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_date   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    INDEX idx_brs_org (org_id),
    INDEX idx_brs_next (next_run_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
DROP TABLE IF EXISTS board_report_schedules;
