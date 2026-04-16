-- +goose Up
-- Full-text search index for the template library (MySQL FULLTEXT)

ALTER TABLE library_templates ADD FULLTEXT INDEX idx_lt_fts (name, description, subject, tags);

-- Template reviews table for human review workflow
CREATE TABLE IF NOT EXISTS template_reviews (
    id              BIGINT AUTO_INCREMENT PRIMARY KEY,
    template_id     BIGINT NOT NULL,
    reviewer_id     BIGINT NOT NULL DEFAULT 0,
    status          VARCHAR(32) NOT NULL DEFAULT 'pending',
    review_type     VARCHAR(32) NOT NULL DEFAULT 'translation',
    notes           TEXT NOT NULL,
    quality_score   INT NOT NULL DEFAULT 0,
    created_date    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reviewed_date   DATETIME NULL,
    INDEX idx_tr_status (status),
    INDEX idx_tr_template (template_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
ALTER TABLE library_templates DROP INDEX idx_lt_fts;
DROP TABLE IF EXISTS template_reviews;
