-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE TABLE IF NOT EXISTS "quizzes" (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "presentation_id" INTEGER NOT NULL UNIQUE,
    "pass_percentage" INTEGER NOT NULL DEFAULT 70,
    "created_by" INTEGER NOT NULL,
    "created_date" DATETIME DEFAULT CURRENT_TIMESTAMP,
    "modified_date" DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (presentation_id) REFERENCES training_presentations(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS "quiz_questions" (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "quiz_id" INTEGER NOT NULL,
    "question_text" TEXT NOT NULL,
    "options" TEXT NOT NULL,
    "correct_option" INTEGER NOT NULL DEFAULT 0,
    "sort_order" INTEGER NOT NULL DEFAULT 0,
    "created_date" DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (quiz_id) REFERENCES quizzes(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS "quiz_attempts" (
    "id" INTEGER PRIMARY KEY AUTOINCREMENT,
    "quiz_id" INTEGER NOT NULL,
    "user_id" INTEGER NOT NULL,
    "score" INTEGER NOT NULL DEFAULT 0,
    "total_questions" INTEGER NOT NULL DEFAULT 0,
    "passed" BOOLEAN NOT NULL DEFAULT 0,
    "answers" TEXT,
    "completed_date" DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (quiz_id) REFERENCES quizzes(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS "idx_quiz_attempts_user" ON "quiz_attempts"("user_id", "quiz_id");

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE IF EXISTS "quiz_attempts";
DROP TABLE IF EXISTS "quiz_questions";
DROP TABLE IF EXISTS "quizzes";
