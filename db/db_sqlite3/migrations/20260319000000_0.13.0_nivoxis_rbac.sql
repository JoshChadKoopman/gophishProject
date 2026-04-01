
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

-- Add 2 new permissions
INSERT INTO "permissions" ("slug", "name", "description") VALUES
    ('manage_training', 'Manage Training', 'Create, edit, and delete training modules'),
    ('view_reports',    'View Reports',    'Read-only access to reports and audit logs');

-- Rename existing role slugs to Nivoxis roles.
-- IDs do not change, so all existing user.role_id foreign keys remain valid.
UPDATE "roles" SET "slug"='superadmin',       "name"='Super Admin',
    "description"='Nivoxis staff — full platform access across all tenants'
WHERE "slug"='admin';

UPDATE "roles" SET "slug"='campaign_manager', "name"='Campaign Manager',
    "description"='Create and launch phishing campaigns, view results'
WHERE "slug"='user';

UPDATE "roles" SET "slug"='trainer',          "name"='Trainer',
    "description"='Assign and manage training modules'
WHERE "slug"='contributor';

UPDATE "roles" SET "slug"='learner',          "name"='Learner',
    "description"='Complete training and view own results'
WHERE "slug"='reader';

-- Grant new permissions to renamed roles

-- superadmin: gets manage_training + view_reports (already has view/modify/system)
INSERT INTO "role_permissions" ("role_id", "permission_id")
SELECT r.id, p.id FROM roles AS r, permissions AS p
WHERE r.slug = 'superadmin' AND p.slug IN ('manage_training', 'view_reports');

-- campaign_manager: gets view_reports
INSERT INTO "role_permissions" ("role_id", "permission_id")
SELECT r.id, p.id FROM roles AS r, permissions AS p
WHERE r.slug = 'campaign_manager' AND p.slug = 'view_reports';

-- trainer: revoke modify_objects (was contributor), grant manage_training instead
DELETE FROM "role_permissions"
WHERE "role_id" = (SELECT id FROM roles WHERE slug = 'trainer')
  AND "permission_id" = (SELECT id FROM permissions WHERE slug = 'modify_objects');

INSERT INTO "role_permissions" ("role_id", "permission_id")
SELECT r.id, p.id FROM roles AS r, permissions AS p
WHERE r.slug = 'trainer' AND p.slug = 'manage_training';

-- Insert 2 new roles
INSERT INTO "roles" ("slug", "name", "description") VALUES
    ('org_admin', 'Org Admin', 'Client HR admin — manages their org campaigns and users'),
    ('auditor',   'Auditor',   'Read-only access to reports and audit logs');

-- org_admin: view_objects + modify_objects + modify_system + manage_training + view_reports
INSERT INTO "role_permissions" ("role_id", "permission_id")
SELECT r.id, p.id FROM roles AS r, permissions AS p
WHERE r.slug = 'org_admin'
  AND p.slug IN ('view_objects', 'modify_objects', 'modify_system', 'manage_training', 'view_reports');

-- auditor: view_objects + view_reports
INSERT INTO "role_permissions" ("role_id", "permission_id")
SELECT r.id, p.id FROM roles AS r, permissions AS p
WHERE r.slug = 'auditor'
  AND p.slug IN ('view_objects', 'view_reports');

-- Audit log table for tracking security-relevant actions (role changes, etc.)
CREATE TABLE IF NOT EXISTS "audit_logs" (
    "id"             INTEGER PRIMARY KEY AUTOINCREMENT,
    "timestamp"      DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "actor_id"       INTEGER NOT NULL,
    "actor_username" VARCHAR(255) NOT NULL,
    "action"         VARCHAR(255) NOT NULL,
    "target_type"    VARCHAR(100),
    "target_id"      INTEGER,
    "target_username" VARCHAR(255),
    "details"        TEXT,
    "ip_address"     VARCHAR(45)
);
CREATE INDEX IF NOT EXISTS "idx_audit_logs_actor"     ON "audit_logs"("actor_id");
CREATE INDEX IF NOT EXISTS "idx_audit_logs_timestamp" ON "audit_logs"("timestamp");

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

DROP TABLE IF EXISTS "audit_logs";

DELETE FROM "role_permissions" WHERE "role_id" IN (
    SELECT "id" FROM "roles" WHERE "slug" IN ('org_admin', 'auditor')
);
DELETE FROM "roles" WHERE "slug" IN ('org_admin', 'auditor');

DELETE FROM "role_permissions" WHERE "permission_id" IN (
    SELECT "id" FROM "permissions" WHERE "slug" IN ('manage_training', 'view_reports')
);
DELETE FROM "permissions" WHERE "slug" IN ('manage_training', 'view_reports');

-- NOTE: role slug renames (admin->superadmin etc.) cannot be safely reversed here
-- without risking data loss. Manual intervention required to restore original slugs.
