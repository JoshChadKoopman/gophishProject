-- +goose Up
-- Board Report Approvals: audit trail for the approval workflow
CREATE TABLE IF NOT EXISTS board_report_approvals (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id INTEGER NOT NULL DEFAULT 0,
    report_id INTEGER NOT NULL,
    from_status VARCHAR(20) NOT NULL DEFAULT '',
    to_status VARCHAR(20) NOT NULL DEFAULT '',
    user_id INTEGER NOT NULL DEFAULT 0,
    username VARCHAR(200) NOT NULL DEFAULT '',
    comment TEXT,
    created_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_board_report_approvals_report ON board_report_approvals(report_id);
CREATE INDEX IF NOT EXISTS idx_board_report_approvals_org ON board_report_approvals(org_id);

-- Board Report Narratives: stored AI-generated / admin-edited executive narratives
CREATE TABLE IF NOT EXISTS board_report_narratives (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id INTEGER NOT NULL DEFAULT 0,
    report_id INTEGER NOT NULL UNIQUE,
    ai_generated INTEGER NOT NULL DEFAULT 0,
    paragraph1 TEXT,
    paragraph2 TEXT,
    paragraph3 TEXT,
    full_narrative TEXT,
    edited_by INTEGER NOT NULL DEFAULT 0,
    created_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    modified_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_board_report_narratives_org ON board_report_narratives(org_id);
CREATE INDEX IF NOT EXISTS idx_board_report_narratives_report ON board_report_narratives(report_id);

-- Ensure tier_features table exists before inserting
CREATE TABLE IF NOT EXISTS tier_features (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tier_id INTEGER NOT NULL,
    feature_slug VARCHAR(100) NOT NULL,
    UNIQUE(tier_id, feature_slug)
);

-- Enable enhanced_board_reports for Enterprise (id=6) and MSP (id=5) tiers
INSERT OR IGNORE INTO tier_features (tier_id, feature_slug) VALUES (5, 'enhanced_board_reports');
INSERT OR IGNORE INTO tier_features (tier_id, feature_slug) VALUES (6, 'enhanced_board_reports');

-- +goose Down
DROP TABLE IF EXISTS board_report_approvals;
DROP TABLE IF EXISTS board_report_narratives;
