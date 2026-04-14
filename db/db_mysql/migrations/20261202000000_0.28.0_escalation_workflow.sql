-- +goose Up
-- Repeat offender escalation workflow tables

CREATE TABLE IF NOT EXISTS escalation_policies (
    id INTEGER PRIMARY KEY AUTO_INCREMENT,
    org_id INTEGER NOT NULL,
    name VARCHAR(255) NOT NULL,
    level INTEGER DEFAULT 1,
    fail_threshold INTEGER DEFAULT 3,
    lookback_days INTEGER DEFAULT 90,
    action VARCHAR(50) DEFAULT 'notify',
    notify_manager BOOLEAN DEFAULT 0,
    notify_admin BOOLEAN DEFAULT 1,
    assign_training_id INTEGER DEFAULT 0,
    is_active BOOLEAN DEFAULT 1,
    created_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    modified_date DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS escalation_events (
    id INTEGER PRIMARY KEY AUTO_INCREMENT,
    org_id INTEGER NOT NULL,
    policy_id INTEGER NOT NULL,
    user_id INTEGER DEFAULT 0,
    user_email VARCHAR(255) NOT NULL,
    level INTEGER DEFAULT 1,
    action VARCHAR(50) DEFAULT 'notify',
    fail_count INTEGER DEFAULT 0,
    details TEXT,
    status VARCHAR(20) DEFAULT 'open',
    resolved_by INTEGER DEFAULT 0,
    resolved_date DATETIME,
    created_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (policy_id) REFERENCES escalation_policies(id)
);

CREATE INDEX idx_escalation_events_org ON escalation_events(org_id, status);
CREATE INDEX idx_escalation_events_user ON escalation_events(user_email, level);
CREATE INDEX idx_escalation_policies_org ON escalation_policies(org_id);

INSERT INTO escalation_policies (org_id, name, level, fail_threshold, lookback_days, action, notify_manager, notify_admin)
VALUES
(1, 'Level 1 — Awareness Reminder', 1, 2, 90, 'notify', 0, 1),
(1, 'Level 2 — Mandatory Training', 2, 4, 90, 'mandatory_training', 1, 1),
(1, 'Level 3 — Manager Escalation', 3, 6, 90, 'manager_escalate', 1, 1);

-- +goose Down
DROP TABLE IF EXISTS escalation_events;
DROP TABLE IF EXISTS escalation_policies;
