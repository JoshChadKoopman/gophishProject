-- +goose Up
-- Template Library Scale-Up: Community marketplace + FTS

CREATE TABLE IF NOT EXISTS community_submissions (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    template_id     INTEGER NOT NULL,
    org_id          INTEGER NOT NULL DEFAULT 0,
    submitted_by    INTEGER NOT NULL DEFAULT 0,
    status          VARCHAR(32) NOT NULL DEFAULT 'pending_review',
    reviewed_by     INTEGER NOT NULL DEFAULT 0,
    review_notes    TEXT NOT NULL DEFAULT '',
    anonymize_org   BOOLEAN NOT NULL DEFAULT 1,
    created_date    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reviewed_date   DATETIME
);
CREATE INDEX IF NOT EXISTS idx_cs_status ON community_submissions(status);
CREATE INDEX IF NOT EXISTS idx_cs_org ON community_submissions(org_id);

-- +goose Down
DROP INDEX IF EXISTS idx_cs_org;
DROP INDEX IF EXISTS idx_cs_status;
DROP TABLE IF EXISTS community_submissions;
