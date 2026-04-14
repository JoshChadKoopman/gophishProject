-- +goose Up
-- Repeat offender remediation paths and enhanced cyber hygiene

CREATE TABLE IF NOT EXISTS `remediation_paths` (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    org_id BIGINT NOT NULL,
    user_id BIGINT NOT NULL DEFAULT 0,
    user_email VARCHAR(255) NOT NULL DEFAULT '',
    escalation_event_id BIGINT DEFAULT 0,
    name VARCHAR(255) NOT NULL DEFAULT '',
    description TEXT,
    fail_count INT DEFAULT 0,
    risk_level VARCHAR(20) DEFAULT 'low',
    status VARCHAR(20) DEFAULT 'active',
    total_courses INT DEFAULT 0,
    completed_count INT DEFAULT 0,
    due_date DATETIME NULL,
    completed_date DATETIME NULL,
    created_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    modified_date DATETIME DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE TABLE IF NOT EXISTS `remediation_steps` (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    path_id BIGINT NOT NULL,
    presentation_id BIGINT NOT NULL,
    sort_order INT DEFAULT 1,
    required BOOLEAN DEFAULT TRUE,
    status VARCHAR(20) DEFAULT 'pending',
    completed_date DATETIME NULL,
    FOREIGN KEY (path_id) REFERENCES remediation_paths(id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_remediation_paths_org ON remediation_paths(org_id);
CREATE INDEX idx_remediation_paths_user ON remediation_paths(user_id);
CREATE INDEX idx_remediation_paths_status ON remediation_paths(org_id, status);
CREATE INDEX idx_remediation_paths_risk ON remediation_paths(org_id, risk_level);
CREATE INDEX idx_remediation_steps_path ON remediation_steps(path_id);

CREATE TABLE IF NOT EXISTS `tech_stack_profiles` (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id BIGINT NOT NULL,
    org_id BIGINT NOT NULL,
    primary_os VARCHAR(50) DEFAULT '',
    browser VARCHAR(50) DEFAULT '',
    email_client VARCHAR(50) DEFAULT '',
    cloud_apps TEXT,
    dev_tools TEXT,
    remote_access VARCHAR(100) DEFAULT '',
    mobile_device VARCHAR(100) DEFAULT '',
    created_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    modified_date DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY uk_techstack_user_org (user_id, org_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

CREATE INDEX idx_tech_stack_user ON tech_stack_profiles(user_id, org_id);

-- +goose Down
DROP TABLE IF EXISTS `remediation_steps`;
DROP TABLE IF EXISTS `remediation_paths`;
DROP TABLE IF EXISTS `tech_stack_profiles`;
