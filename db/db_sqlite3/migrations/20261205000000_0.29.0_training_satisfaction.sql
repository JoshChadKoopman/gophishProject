-- +goose Up
-- Phase 29: Training Satisfaction Ratings & Analytics

-- User satisfaction ratings for training sessions
CREATE TABLE IF NOT EXISTS training_satisfaction_ratings (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    presentation_id INTEGER NOT NULL,
    rating INTEGER NOT NULL CHECK(rating >= 1 AND rating <= 5),
    feedback TEXT NOT NULL DEFAULT '',
    created_date DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_satisfaction_user_pres ON training_satisfaction_ratings(user_id, presentation_id);
CREATE INDEX IF NOT EXISTS idx_satisfaction_pres ON training_satisfaction_ratings(presentation_id);

-- +goose Down
DROP TABLE IF EXISTS training_satisfaction_ratings;
