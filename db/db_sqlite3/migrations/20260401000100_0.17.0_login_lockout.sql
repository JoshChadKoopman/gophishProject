-- +goose Up
-- Phase 6: Add login lockout tracking columns to users table.
ALTER TABLE users ADD COLUMN failed_logins INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN last_failed_login DATETIME;

-- +goose Down
-- SQLite does not support DROP COLUMN; these columns will remain as no-ops.
