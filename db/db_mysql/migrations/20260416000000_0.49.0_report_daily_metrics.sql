-- +goose Up
-- report_daily_metrics: pre-computed daily rollups per org.
-- Populated by a nightly worker; queried by dashboard APIs
-- instead of scanning raw campaign/result/ticket tables.
CREATE TABLE IF NOT EXISTS report_daily_metrics (
    id                          BIGINT AUTO_INCREMENT PRIMARY KEY,
    org_id                      BIGINT NOT NULL DEFAULT 0,
    metric_date                 DATE   NOT NULL,

    -- Campaign / email metrics
    emails_sent                 BIGINT NOT NULL DEFAULT 0,
    emails_opened               BIGINT NOT NULL DEFAULT 0,
    links_clicked               BIGINT NOT NULL DEFAULT 0,
    data_submitted              BIGINT NOT NULL DEFAULT 0,
    emails_reported             BIGINT NOT NULL DEFAULT 0,
    campaigns_launched          BIGINT NOT NULL DEFAULT 0,
    campaigns_completed         BIGINT NOT NULL DEFAULT 0,

    -- Training metrics
    training_assigned           BIGINT NOT NULL DEFAULT 0,
    training_completed          BIGINT NOT NULL DEFAULT 0,
    training_overdue            BIGINT NOT NULL DEFAULT 0,
    avg_quiz_score              DOUBLE NOT NULL DEFAULT 0,
    certificates_issued         BIGINT NOT NULL DEFAULT 0,

    -- Ticket / incident metrics
    tickets_opened              BIGINT NOT NULL DEFAULT 0,
    tickets_resolved            BIGINT NOT NULL DEFAULT 0,
    incidents_created           BIGINT NOT NULL DEFAULT 0,
    incidents_resolved          BIGINT NOT NULL DEFAULT 0,
    network_events_ingested     BIGINT NOT NULL DEFAULT 0,

    -- Risk / compliance snapshot
    avg_risk_score              DOUBLE NOT NULL DEFAULT 0,
    high_risk_user_count        BIGINT NOT NULL DEFAULT 0,
    compliance_score            DOUBLE NOT NULL DEFAULT 0,

    -- Hygiene snapshot
    avg_hygiene_score           DOUBLE NOT NULL DEFAULT 0,
    devices_compliant           BIGINT NOT NULL DEFAULT 0,
    devices_total               BIGINT NOT NULL DEFAULT 0,

    -- Computed rates (stored for fast retrieval)
    click_rate                  DOUBLE NOT NULL DEFAULT 0,
    report_rate                 DOUBLE NOT NULL DEFAULT 0,
    training_completion_rate    DOUBLE NOT NULL DEFAULT 0,

    -- Active users metric
    active_campaigns            BIGINT NOT NULL DEFAULT 0,
    total_users                 BIGINT NOT NULL DEFAULT 0,

    -- Timestamps
    computed_at                 DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE KEY uq_rdm_org_date (org_id, metric_date)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_rdm_org_date ON report_daily_metrics(org_id, metric_date);
CREATE INDEX idx_rdm_date ON report_daily_metrics(metric_date);

-- +goose Down
DROP TABLE IF EXISTS report_daily_metrics;
