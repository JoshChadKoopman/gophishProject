-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE users ADD COLUMN first_name varchar(255) DEFAULT '';
ALTER TABLE users ADD COLUMN last_name varchar(255) DEFAULT '';
ALTER TABLE users ADD COLUMN email varchar(255) DEFAULT '';
ALTER TABLE users ADD COLUMN position varchar(255) DEFAULT '';

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
ALTER TABLE users DROP COLUMN first_name;
ALTER TABLE users DROP COLUMN last_name;
ALTER TABLE users DROP COLUMN email;
ALTER TABLE users DROP COLUMN position;
