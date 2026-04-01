-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE "campaigns" ADD COLUMN "training_presentation_id" INTEGER DEFAULT 0;

-- +goose Down
-- SQLite < 3.35 cannot DROP COLUMN; column left in place on rollback
