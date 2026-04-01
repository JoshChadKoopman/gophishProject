-- +goose Up
-- Phase 12: Multi-Language Support (i18n)
-- Add preferred_language to users and default_language to organizations

ALTER TABLE users ADD COLUMN preferred_language VARCHAR(5) DEFAULT 'en';
ALTER TABLE organizations ADD COLUMN default_language VARCHAR(5) DEFAULT 'en';

-- +goose Down
ALTER TABLE users DROP COLUMN preferred_language;
ALTER TABLE organizations DROP COLUMN default_language;
