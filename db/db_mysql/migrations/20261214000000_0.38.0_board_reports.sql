-- +goose Up
-- Board Reports: executive summary reporting for board / C-suite

CREATE TABLE IF NOT EXISTS `board_reports` (
    `id` BIGINT AUTO_INCREMENT PRIMARY KEY,
    `org_id` BIGINT NOT NULL,
    `created_by` BIGINT NOT NULL DEFAULT 0,
    `title` VARCHAR(255) NOT NULL DEFAULT '',
    `period_start` DATETIME NOT NULL,
    `period_end` DATETIME NOT NULL,
    `status` VARCHAR(20) NOT NULL DEFAULT 'draft',
    `created_date` DATETIME DEFAULT CURRENT_TIMESTAMP,
    `modified_date` DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_board_reports_org ON board_reports(org_id);
CREATE INDEX idx_board_reports_status ON board_reports(org_id, status);

-- Enable board_reports for MSP (id=5) tier if not already present
INSERT IGNORE INTO tier_features (tier_id, feature_slug) VALUES (5, 'board_reports');

-- +goose Down
DROP TABLE IF EXISTS `board_reports`;
