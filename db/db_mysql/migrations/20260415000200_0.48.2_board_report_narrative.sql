-- +goose Up
-- Board Report Approvals: audit trail for the approval workflow
CREATE TABLE IF NOT EXISTS board_report_approvals (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    org_id BIGINT NOT NULL DEFAULT 0,
    report_id BIGINT NOT NULL,
    from_status VARCHAR(20) NOT NULL DEFAULT '',
    to_status VARCHAR(20) NOT NULL DEFAULT '',
    user_id BIGINT NOT NULL DEFAULT 0,
    username VARCHAR(200) NOT NULL DEFAULT '',
    comment TEXT,
    created_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_board_report_approvals_report (report_id),
    INDEX idx_board_report_approvals_org (org_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Board Report Narratives: stored AI-generated / admin-edited executive narratives
CREATE TABLE IF NOT EXISTS board_report_narratives (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    org_id BIGINT NOT NULL DEFAULT 0,
    report_id BIGINT NOT NULL,
    ai_generated TINYINT NOT NULL DEFAULT 0,
    paragraph1 TEXT,
    paragraph2 TEXT,
    paragraph3 TEXT,
    full_narrative TEXT,
    edited_by BIGINT NOT NULL DEFAULT 0,
    created_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uk_board_report_narratives_report (report_id),
    INDEX idx_board_report_narratives_org (org_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Enable enhanced_board_reports for Enterprise (id=6) and MSP (id=5) tiers
INSERT IGNORE INTO tier_features (tier_id, feature_slug) VALUES (5, 'enhanced_board_reports');
INSERT IGNORE INTO tier_features (tier_id, feature_slug) VALUES (6, 'enhanced_board_reports');

-- +goose Down
DROP TABLE IF EXISTS board_report_approvals;
DROP TABLE IF EXISTS board_report_narratives;
