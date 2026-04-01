-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE TABLE IF NOT EXISTS "certificates" (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "user_id" INTEGER NOT NULL,
    "presentation_id" INTEGER NOT NULL,
    "quiz_attempt_id" INTEGER,
    "verification_code" VARCHAR(32) NOT NULL UNIQUE,
    "issued_date" DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (presentation_id) REFERENCES training_presentations(id) ON DELETE CASCADE
);

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE IF EXISTS "certificates";
