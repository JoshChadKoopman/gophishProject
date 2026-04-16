-- +goose Up
-- Add variant_id to autopilot_schedules for A/B test assignment
ALTER TABLE autopilot_schedules ADD COLUMN variant_id VARCHAR(8) NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE autopilot_schedules DROP COLUMN variant_id;
