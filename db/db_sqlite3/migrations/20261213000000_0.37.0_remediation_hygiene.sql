-- +goose Up
-- Repeat offender remediation paths and enhanced cyber hygiene

-- Remediation paths table
CREATE TABLE IF NOT EXISTS "remediation_paths" (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    org_id INTEGER NOT NULL,
    user_id INTEGER NOT NULL DEFAULT 0,
    user_email VARCHAR(255) NOT NULL DEFAULT '',
    escalation_event_id INTEGER DEFAULT 0,
    name VARCHAR(255) NOT NULL DEFAULT '',
    description TEXT DEFAULT '',
    fail_count INTEGER DEFAULT 0,
    risk_level VARCHAR(20) DEFAULT 'low',
    status VARCHAR(20) DEFAULT 'active',
    total_courses INTEGER DEFAULT 0,
    completed_count INTEGER DEFAULT 0,
    due_date DATETIME,
    completed_date DATETIME,
    created_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    modified_date DATETIME DEFAULT CURRENT_TIMESTAMP
);

-- Remediation steps table
CREATE TABLE IF NOT EXISTS "remediation_steps" (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    path_id INTEGER NOT NULL,
    presentation_id INTEGER NOT NULL,
    sort_order INTEGER DEFAULT 1,
    required BOOLEAN DEFAULT 1,
    status VARCHAR(20) DEFAULT 'pending',
    completed_date DATETIME,
    FOREIGN KEY (path_id) REFERENCES remediation_paths(id)
);

-- Indexes for remediation paths
CREATE INDEX IF NOT EXISTS idx_remediation_paths_org ON remediation_paths(org_id);
CREATE INDEX IF NOT EXISTS idx_remediation_paths_user ON remediation_paths(user_id);
CREATE INDEX IF NOT EXISTS idx_remediation_paths_status ON remediation_paths(org_id, status);
CREATE INDEX IF NOT EXISTS idx_remediation_paths_risk ON remediation_paths(org_id, risk_level);
CREATE INDEX IF NOT EXISTS idx_remediation_steps_path ON remediation_steps(path_id);

-- Tech stack profiles for personalized hygiene
CREATE TABLE IF NOT EXISTS "tech_stack_profiles" (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    user_id INTEGER NOT NULL,
    org_id INTEGER NOT NULL,
    primary_os VARCHAR(50) DEFAULT '',
    browser VARCHAR(50) DEFAULT '',
    email_client VARCHAR(50) DEFAULT '',
    cloud_apps TEXT DEFAULT '[]',
    dev_tools TEXT DEFAULT '[]',
    remote_access VARCHAR(100) DEFAULT '',
    mobile_device VARCHAR(100) DEFAULT '',
    created_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    modified_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, org_id)
);

CREATE INDEX IF NOT EXISTS idx_tech_stack_user ON tech_stack_profiles(user_id, org_id);

-- +goose Down
DROP TABLE IF EXISTS "remediation_steps";
DROP TABLE IF EXISTS "remediation_paths";
DROP TABLE IF EXISTS "tech_stack_profiles";
