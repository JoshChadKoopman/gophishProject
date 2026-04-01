
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

-- Add the new "reader" and "contributor" roles
INSERT INTO "roles" ("slug", "name", "description")
VALUES
    ("contributor", "Contributor", "Can modify email templates, landing pages, and sending profiles but cannot manage users or account settings"),
    ("reader", "Reader", "Read-only access to the dashboard and campaign results");

-- Allow readers to view objects
INSERT INTO "role_permissions" ("role_id", "permission_id")
SELECT r.id, p.id FROM roles AS r, permissions AS p
WHERE r.slug = 'reader'
AND p.slug = 'view_objects';

-- Allow contributors to view objects
INSERT INTO "role_permissions" ("role_id", "permission_id")
SELECT r.id, p.id FROM roles AS r, permissions AS p
WHERE r.slug = 'contributor'
AND p.slug = 'view_objects';

-- Allow contributors to modify objects (campaigns, groups, etc.)
INSERT INTO "role_permissions" ("role_id", "permission_id")
SELECT r.id, p.id FROM roles AS r, permissions AS p
WHERE r.slug = 'contributor'
AND p.slug = 'modify_objects';

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

DELETE FROM "role_permissions" WHERE "role_id" IN (
    SELECT "id" FROM "roles" WHERE "slug" IN ('contributor', 'reader')
);

DELETE FROM "roles" WHERE "slug" IN ('contributor', 'reader');
