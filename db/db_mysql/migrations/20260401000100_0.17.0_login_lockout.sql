-- +goose Up
-- Phase 6: Add login lockout tracking columns to users table.
ALTER TABLE users ADD COLUMN failed_logins INTEGER NOT NULL DEFAULT 0;
ALTER TABLE users ADD COLUMN last_failed_login DATETIME;

-- +goose Down
ALTER TABLE users DROP COLUMN failed_logins;
ALTER TABLE users DROP COLUMN last_failed_login;
