
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

-- Add 2 new permissions
INSERT INTO `permissions` (`slug`, `name`, `description`) VALUES
    ('manage_training', 'Manage Training', 'Create, edit, and delete training modules'),
    ('view_reports',    'View Reports',    'Read-only access to reports and audit logs');

-- Rename existing role slugs to Nivoxis roles.
-- IDs do not change, so all existing user.role_id foreign keys remain valid.
UPDATE `roles` SET `slug`='superadmin',       `name`='Super Admin',
    `description`='Nivoxis staff — full platform access across all tenants'
WHERE `slug`='admin';

UPDATE `roles` SET `slug`='campaign_manager', `name`='Campaign Manager',
    `description`='Create and launch phishing campaigns, view results'
WHERE `slug`='user';

UPDATE `roles` SET `slug`='trainer',          `name`='Trainer',
    `description`='Assign and manage training modules'
WHERE `slug`='contributor';

UPDATE `roles` SET `slug`='learner',          `name`='Learner',
    `description`='Complete training and view own results'
WHERE `slug`='reader';

-- Grant new permissions to renamed roles

-- superadmin: gets manage_training + view_reports
INSERT INTO `role_permissions` (`role_id`, `permission_id`)
SELECT r.id, p.id FROM roles AS r, permissions AS p
WHERE r.slug = 'superadmin' AND p.slug IN ('manage_training', 'view_reports');

-- campaign_manager: gets view_reports
INSERT INTO `role_permissions` (`role_id`, `permission_id`)
SELECT r.id, p.id FROM roles AS r, permissions AS p
WHERE r.slug = 'campaign_manager' AND p.slug = 'view_reports';

-- trainer: revoke modify_objects, grant manage_training
DELETE FROM `role_permissions`
WHERE `role_id` = (SELECT id FROM (SELECT id FROM roles WHERE slug = 'trainer') AS t)
  AND `permission_id` = (SELECT id FROM (SELECT id FROM permissions WHERE slug = 'modify_objects') AS p2);

INSERT INTO `role_permissions` (`role_id`, `permission_id`)
SELECT r.id, p.id FROM roles AS r, permissions AS p
WHERE r.slug = 'trainer' AND p.slug = 'manage_training';

-- Insert 2 new roles
INSERT INTO `roles` (`slug`, `name`, `description`) VALUES
    ('org_admin', 'Org Admin', 'Client HR admin — manages their org campaigns and users'),
    ('auditor',   'Auditor',   'Read-only access to reports and audit logs');

-- org_admin: all permissions
INSERT INTO `role_permissions` (`role_id`, `permission_id`)
SELECT r.id, p.id FROM roles AS r, permissions AS p
WHERE r.slug = 'org_admin'
  AND p.slug IN ('view_objects', 'modify_objects', 'modify_system', 'manage_training', 'view_reports');

-- auditor: view_objects + view_reports
INSERT INTO `role_permissions` (`role_id`, `permission_id`)
SELECT r.id, p.id FROM roles AS r, permissions AS p
WHERE r.slug = 'auditor'
  AND p.slug IN ('view_objects', 'view_reports');

-- Audit log table
CREATE TABLE IF NOT EXISTS `audit_logs` (
    `id`             BIGINT NOT NULL AUTO_INCREMENT,
    `timestamp`      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    `actor_id`       BIGINT NOT NULL,
    `actor_username` VARCHAR(255) NOT NULL,
    `action`         VARCHAR(255) NOT NULL,
    `target_type`    VARCHAR(100),
    `target_id`      BIGINT,
    `target_username` VARCHAR(255),
    `details`        TEXT,
    `ip_address`     VARCHAR(45),
    PRIMARY KEY (`id`),
    INDEX `idx_audit_logs_actor`     (`actor_id`),
    INDEX `idx_audit_logs_timestamp` (`timestamp`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

DROP TABLE IF EXISTS `audit_logs`;

DELETE FROM `role_permissions` WHERE `role_id` IN (
    SELECT `id` FROM `roles` WHERE `slug` IN ('org_admin', 'auditor')
);
DELETE FROM `roles` WHERE `slug` IN ('org_admin', 'auditor');

DELETE rp FROM `role_permissions` rp
INNER JOIN `permissions` p ON rp.permission_id = p.id
WHERE p.slug IN ('manage_training', 'view_reports');
DELETE FROM `permissions` WHERE `slug` IN ('manage_training', 'view_reports');

-- NOTE: role slug renames cannot be safely reversed here without data loss risk.
