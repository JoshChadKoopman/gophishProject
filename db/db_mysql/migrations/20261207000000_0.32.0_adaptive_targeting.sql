-- +goose Up
-- Add category column to templates for adaptive targeting
ALTER TABLE templates ADD COLUMN category VARCHAR(100) DEFAULT '';

-- +goose Down
ALTER TABLE templates DROP COLUMN category;
