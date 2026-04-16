-- +goose Up
-- report_daily_metrics: pre-computed daily rollups per org.
-- Populated by a nightly worker; queried by dashboard APIs
-- instead of scanning raw campaign/result/ticket tables.
CREATE TABLE IF NOT EXISTS report_daily_metrics (
    id                          INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id                      INTEGER NOT NULL DEFAULT 0,
    metric_date                 DATE    NOT NULL,

    -- Campaign / email metrics
    emails_sent                 INTEGER NOT NULL DEFAULT 0,
    emails_opened               INTEGER NOT NULL DEFAULT 0,
    links_clicked               INTEGER NOT NULL DEFAULT 0,
    data_submitted              INTEGER NOT NULL DEFAULT 0,
    emails_reported             INTEGER NOT NULL DEFAULT 0,
    campaigns_launched          INTEGER NOT NULL DEFAULT 0,
    campaigns_completed         INTEGER NOT NULL DEFAULT 0,

    -- Training metrics
    training_assigned           INTEGER NOT NULL DEFAULT 0,
    training_completed          INTEGER NOT NULL DEFAULT 0,
    training_overdue            INTEGER NOT NULL DEFAULT 0,
    avg_quiz_score              REAL    NOT NULL DEFAULT 0,
    certificates_issued         INTEGER NOT NULL DEFAULT 0,

    -- Ticket / incident metrics
    tickets_opened              INTEGER NOT NULL DEFAULT 0,
    tickets_resolved            INTEGER NOT NULL DEFAULT 0,
    incidents_created           INTEGER NOT NULL DEFAULT 0,
    incidents_resolved          INTEGER NOT NULL DEFAULT 0,
    network_events_ingested     INTEGER NOT NULL DEFAULT 0,

    -- Risk / compliance snapshot
    avg_risk_score              REAL    NOT NULL DEFAULT 0,
    high_risk_user_count        INTEGER NOT NULL DEFAULT 0,
    compliance_score            REAL    NOT NULL DEFAULT 0,

    -- Hygiene snapshot
    avg_hygiene_score           REAL    NOT NULL DEFAULT 0,
    devices_compliant           INTEGER NOT NULL DEFAULT 0,
    devices_total               INTEGER NOT NULL DEFAULT 0,

    -- Computed rates (stored for fast retrieval)
    click_rate                  REAL    NOT NULL DEFAULT 0,
    report_rate                 REAL    NOT NULL DEFAULT 0,
    training_completion_rate    REAL    NOT NULL DEFAULT 0,

    -- Active users metric
    active_campaigns            INTEGER NOT NULL DEFAULT 0,
    total_users                 INTEGER NOT NULL DEFAULT 0,

    -- Timestamps
    computed_at                 DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

    UNIQUE(org_id, metric_date)
);

CREATE INDEX IF NOT EXISTS idx_rdm_org_date ON report_daily_metrics(org_id, metric_date);
CREATE INDEX IF NOT EXISTS idx_rdm_date ON report_daily_metrics(metric_date);

-- +goose Down
DROP TABLE IF EXISTS report_daily_metrics;
