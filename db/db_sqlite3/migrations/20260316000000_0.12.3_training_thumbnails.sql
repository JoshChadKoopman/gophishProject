-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE training_presentations ADD COLUMN thumbnail_path varchar(512) DEFAULT '';

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
