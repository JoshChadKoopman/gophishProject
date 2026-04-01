-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE training_presentations ADD COLUMN youtube_url VARCHAR(500) DEFAULT '';
ALTER TABLE training_presentations ADD COLUMN content_pages TEXT DEFAULT NULL;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
