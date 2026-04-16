-- +goose Up
-- ROI Benchmarks: configurable industry benchmark comparison data
CREATE TABLE IF NOT EXISTS roi_benchmarks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id INTEGER NOT NULL DEFAULT 0,
    metric_key VARCHAR(100) NOT NULL,
    metric_label VARCHAR(200) NOT NULL DEFAULT '',
    industry_avg REAL NOT NULL DEFAULT 0,
    industry_p25 REAL NOT NULL DEFAULT 0,
    industry_p75 REAL NOT NULL DEFAULT 0,
    source VARCHAR(200) NOT NULL DEFAULT '',
    year INTEGER NOT NULL DEFAULT 2025,
    category VARCHAR(100) NOT NULL DEFAULT '',
    modified_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(org_id, metric_key)
);

CREATE INDEX IF NOT EXISTS idx_roi_benchmarks_org ON roi_benchmarks(org_id);

-- ROI Reports: historical report snapshots for time-series trending
CREATE TABLE IF NOT EXISTS roi_reports (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id INTEGER NOT NULL DEFAULT 0,
    period_start DATETIME NOT NULL,
    period_end DATETIME NOT NULL,
    quarter VARCHAR(10) NOT NULL DEFAULT '',
    roi_percentage REAL NOT NULL DEFAULT 0,
    cost_avoidance REAL NOT NULL DEFAULT 0,
    program_cost REAL NOT NULL DEFAULT 0,
    click_rate REAL NOT NULL DEFAULT 0,
    report_rate REAL NOT NULL DEFAULT 0,
    incidents_avoided INTEGER NOT NULL DEFAULT 0,
    risk_reduction REAL NOT NULL DEFAULT 0,
    training_completion REAL NOT NULL DEFAULT 0,
    compliance_score REAL NOT NULL DEFAULT 0,
    report_json TEXT,
    generated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    generated_by INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_roi_reports_org ON roi_reports(org_id);
CREATE INDEX IF NOT EXISTS idx_roi_reports_quarter ON roi_reports(org_id, quarter);

-- +goose Down
DROP TABLE IF EXISTS roi_benchmarks;
DROP TABLE IF EXISTS roi_reports;
