-- +goose Up
-- ROI Benchmarks: configurable industry benchmark comparison data
CREATE TABLE IF NOT EXISTS roi_benchmarks (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    org_id BIGINT NOT NULL DEFAULT 0,
    metric_key VARCHAR(100) NOT NULL,
    metric_label VARCHAR(200) NOT NULL DEFAULT '',
    industry_avg DOUBLE NOT NULL DEFAULT 0,
    industry_p25 DOUBLE NOT NULL DEFAULT 0,
    industry_p75 DOUBLE NOT NULL DEFAULT 0,
    source VARCHAR(200) NOT NULL DEFAULT '',
    year INT NOT NULL DEFAULT 2025,
    category VARCHAR(100) NOT NULL DEFAULT '',
    modified_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uq_roi_bench_org_key (org_id, metric_key)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_roi_benchmarks_org ON roi_benchmarks(org_id);

-- ROI Reports: historical report snapshots for time-series trending
CREATE TABLE IF NOT EXISTS roi_reports (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    org_id BIGINT NOT NULL DEFAULT 0,
    period_start DATETIME NOT NULL,
    period_end DATETIME NOT NULL,
    quarter VARCHAR(10) NOT NULL DEFAULT '',
    roi_percentage DOUBLE NOT NULL DEFAULT 0,
    cost_avoidance DOUBLE NOT NULL DEFAULT 0,
    program_cost DOUBLE NOT NULL DEFAULT 0,
    click_rate DOUBLE NOT NULL DEFAULT 0,
    report_rate DOUBLE NOT NULL DEFAULT 0,
    incidents_avoided INT NOT NULL DEFAULT 0,
    risk_reduction DOUBLE NOT NULL DEFAULT 0,
    training_completion DOUBLE NOT NULL DEFAULT 0,
    compliance_score DOUBLE NOT NULL DEFAULT 0,
    report_json LONGTEXT,
    generated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    generated_by BIGINT NOT NULL DEFAULT 0
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_roi_reports_org ON roi_reports(org_id);
CREATE INDEX idx_roi_reports_quarter ON roi_reports(org_id, quarter);

-- +goose Down
DROP TABLE IF EXISTS roi_benchmarks;
DROP TABLE IF EXISTS roi_reports;
