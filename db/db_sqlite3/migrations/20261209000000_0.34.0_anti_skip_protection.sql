-- +goose Up
-- Per-page engagement evidence for anti-skip protection.
CREATE TABLE IF NOT EXISTS page_engagement (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    presentation_id INTEGER NOT NULL,
    page_index INTEGER NOT NULL,
    entered_at DATETIME NOT NULL,
    dwell_seconds INTEGER DEFAULT 0,
    scroll_depth_pct INTEGER DEFAULT 0,        -- 0-100, how far user scrolled
    interaction_type VARCHAR(30) DEFAULT '',    -- "timer", "acknowledge", "quiz_inline"
    acknowledged BOOLEAN DEFAULT 0,            -- user clicked "I've read this"
    created_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_page_engagement_unique
    ON page_engagement(user_id, presentation_id, page_index);
CREATE INDEX IF NOT EXISTS idx_page_engagement_user_pres
    ON page_engagement(user_id, presentation_id);

-- Anti-skip policy per-presentation (admins can customise).
CREATE TABLE IF NOT EXISTS anti_skip_policy (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    presentation_id INTEGER NOT NULL UNIQUE,
    min_dwell_seconds INTEGER DEFAULT 10,          -- minimum seconds per page
    require_acknowledge BOOLEAN DEFAULT 0,         -- require "I've read this" checkbox
    require_scroll BOOLEAN DEFAULT 0,              -- require scrolling to bottom
    min_scroll_depth_pct INTEGER DEFAULT 80,       -- minimum scroll depth if require_scroll
    enforce_sequential BOOLEAN DEFAULT 1,          -- must visit pages in order
    allow_back_navigation BOOLEAN DEFAULT 1,       -- allow revisiting earlier pages
    created_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    modified_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (presentation_id) REFERENCES training_presentations(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE IF EXISTS page_engagement;
DROP TABLE IF EXISTS anti_skip_policy;
