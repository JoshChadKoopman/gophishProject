-- +goose Up
-- Full-text search support for the template library
-- Note: FTS5 requires SQLite compiled with SQLITE_ENABLE_FTS5.
-- If FTS5 is not available, the application falls back to LIKE-based search.

-- Add indexes for fast LIKE-based search (always works)
CREATE INDEX IF NOT EXISTS idx_lt_name ON library_templates(name);
CREATE INDEX IF NOT EXISTS idx_lt_category ON library_templates(category);
CREATE INDEX IF NOT EXISTS idx_lt_language ON library_templates(language);
CREATE INDEX IF NOT EXISTS idx_lt_source ON library_templates(source);

-- Template reviews table for human review workflow
CREATE TABLE IF NOT EXISTS template_reviews (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    template_id     INTEGER NOT NULL,
    reviewer_id     INTEGER NOT NULL DEFAULT 0,
    status          VARCHAR(32) NOT NULL DEFAULT 'pending',
    review_type     VARCHAR(32) NOT NULL DEFAULT 'translation',
    notes           TEXT NOT NULL DEFAULT '',
    quality_score   INTEGER NOT NULL DEFAULT 0,
    created_date    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reviewed_date   DATETIME
);
CREATE INDEX IF NOT EXISTS idx_tr_status ON template_reviews(status);
CREATE INDEX IF NOT EXISTS idx_tr_template ON template_reviews(template_id);

-- +goose Down
DROP INDEX IF EXISTS idx_lt_name;
DROP INDEX IF EXISTS idx_lt_category;
DROP INDEX IF EXISTS idx_lt_language;
DROP INDEX IF EXISTS idx_lt_source;
DROP INDEX IF EXISTS idx_tr_template;
DROP INDEX IF EXISTS idx_tr_status;
DROP TABLE IF EXISTS template_reviews;
