-- +goose Up
-- Add category column to templates for adaptive targeting
ALTER TABLE templates ADD COLUMN category VARCHAR(100) DEFAULT '';

-- +goose Down
-- SQLite doesn't support DROP COLUMN directly; this is a best-effort rollback.
