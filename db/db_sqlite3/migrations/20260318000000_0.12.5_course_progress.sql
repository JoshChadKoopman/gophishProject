-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE TABLE IF NOT EXISTS "course_progress" (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "user_id" INTEGER NOT NULL,
    "presentation_id" INTEGER NOT NULL,
    "current_page" INTEGER DEFAULT 0,
    "total_pages" INTEGER DEFAULT 0,
    "status" VARCHAR(50) DEFAULT 'no_progress',
    "completed_date" DATETIME,
    "last_accessed_date" DATETIME,
    "created_date" DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE IF EXISTS "course_progress";
