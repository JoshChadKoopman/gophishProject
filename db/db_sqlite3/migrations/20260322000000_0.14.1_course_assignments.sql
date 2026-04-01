-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE TABLE IF NOT EXISTS "course_assignments" (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "user_id" INTEGER NOT NULL,
    "presentation_id" INTEGER NOT NULL,
    "assigned_by" INTEGER NOT NULL,
    "group_id" INTEGER,
    "campaign_id" INTEGER,
    "due_date" DATETIME,
    "status" VARCHAR(50) NOT NULL DEFAULT 'pending',
    "created_date" DATETIME DEFAULT CURRENT_TIMESTAMP,
    "modified_date" DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (presentation_id) REFERENCES training_presentations(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS "idx_assignment_user_pres" ON "course_assignments"("user_id", "presentation_id");

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE IF EXISTS "course_assignments";
