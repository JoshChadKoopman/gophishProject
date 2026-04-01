-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE TABLE IF NOT EXISTS training_presentations (
    id INTEGER PRIMARY KEY AUTO_INCREMENT,
    name varchar(255) NOT NULL,
    description TEXT DEFAULT '',
    file_name varchar(255) NOT NULL,
    file_path varchar(512) NOT NULL,
    file_size INTEGER DEFAULT 0,
    content_type varchar(255) DEFAULT '',
    uploaded_by INTEGER NOT NULL,
    created_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    modified_date DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE IF EXISTS training_presentations;
