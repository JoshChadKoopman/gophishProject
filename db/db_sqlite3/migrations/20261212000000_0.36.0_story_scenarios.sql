-- +goose Up
-- Branching narrative training: interactive "choose-your-own-adventure" scenarios
-- for immersive cybersecurity decision training (phished.io parity feature).

CREATE TABLE IF NOT EXISTS story_scenarios (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    presentation_id INTEGER NOT NULL DEFAULT 0,
    title VARCHAR(255) NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    category VARCHAR(100) NOT NULL DEFAULT '',
    difficulty INTEGER NOT NULL DEFAULT 1,
    start_node_id INTEGER NOT NULL DEFAULT 0,
    pass_threshold INTEGER NOT NULL DEFAULT 70,
    created_by INTEGER NOT NULL DEFAULT 0,
    created_date DATETIME,
    modified_date DATETIME
);

CREATE INDEX IF NOT EXISTS idx_story_presentation ON story_scenarios(presentation_id);
CREATE INDEX IF NOT EXISTS idx_story_category ON story_scenarios(category);

CREATE TABLE IF NOT EXISTS story_nodes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    scenario_id INTEGER NOT NULL DEFAULT 0,
    node_key VARCHAR(100) NOT NULL DEFAULT '',
    node_type VARCHAR(30) NOT NULL DEFAULT 'choice',
    title VARCHAR(255) NOT NULL DEFAULT '',
    body TEXT NOT NULL DEFAULT '',
    media_url VARCHAR(500) NOT NULL DEFAULT '',
    score_delta INTEGER NOT NULL DEFAULT 0,
    is_terminal BOOLEAN NOT NULL DEFAULT 0,
    outcome VARCHAR(30) NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_story_nodes_scenario ON story_nodes(scenario_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_story_nodes_key ON story_nodes(scenario_id, node_key);

CREATE TABLE IF NOT EXISTS story_choices (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    node_id INTEGER NOT NULL DEFAULT 0,
    label VARCHAR(500) NOT NULL DEFAULT '',
    next_node_id INTEGER NOT NULL DEFAULT 0,
    score_delta INTEGER NOT NULL DEFAULT 0,
    feedback TEXT NOT NULL DEFAULT '',
    is_correct BOOLEAN NOT NULL DEFAULT 0,
    sort_order INTEGER NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_story_choices_node ON story_choices(node_id);

CREATE TABLE IF NOT EXISTS story_progress (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL DEFAULT 0,
    scenario_id INTEGER NOT NULL DEFAULT 0,
    current_node_id INTEGER NOT NULL DEFAULT 0,
    score INTEGER NOT NULL DEFAULT 0,
    path TEXT NOT NULL DEFAULT '',
    status VARCHAR(30) NOT NULL DEFAULT 'in_progress',
    started_date DATETIME,
    completed_date DATETIME
);

CREATE INDEX IF NOT EXISTS idx_story_progress_user ON story_progress(user_id, scenario_id);
CREATE INDEX IF NOT EXISTS idx_story_progress_status ON story_progress(status);

-- +goose Down
DROP INDEX IF EXISTS idx_story_progress_status;
DROP INDEX IF EXISTS idx_story_progress_user;
DROP TABLE IF EXISTS story_progress;
DROP INDEX IF EXISTS idx_story_choices_node;
DROP TABLE IF EXISTS story_choices;
DROP INDEX IF EXISTS idx_story_nodes_key;
DROP INDEX IF EXISTS idx_story_nodes_scenario;
DROP TABLE IF EXISTS story_nodes;
DROP INDEX IF EXISTS idx_story_category;
DROP INDEX IF EXISTS idx_story_presentation;
DROP TABLE IF EXISTS story_scenarios;
