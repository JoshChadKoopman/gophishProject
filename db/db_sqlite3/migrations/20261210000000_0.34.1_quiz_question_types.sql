-- +goose Up
-- Add question_type, correct_options, and explanation columns to quiz_questions
-- to support multi-select questions, per-question explanations, and typed questions.
ALTER TABLE "quiz_questions" ADD COLUMN "question_type" VARCHAR(30) NOT NULL DEFAULT 'multiple_choice';
ALTER TABLE "quiz_questions" ADD COLUMN "correct_options" TEXT DEFAULT '';
ALTER TABLE "quiz_questions" ADD COLUMN "explanation" TEXT DEFAULT '';

-- +goose Down
-- SQLite does not support DROP COLUMN before 3.35.0; these are effectively no-ops
-- on older versions. For newer SQLite the columns will be removed.
