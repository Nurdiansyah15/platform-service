-- Assignment pada scope custom tidak bisa direpresentasikan di skema lama → dihapus.
DELETE FROM user_role_assignments WHERE scope_type NOT IN ('company', 'branch');

ALTER TABLE user_role_assignments
    DROP CONSTRAINT ck_user_role_assignments_scope_id,
    DROP CONSTRAINT ck_user_role_assignments_scope_type,
    DROP COLUMN scope_parent_id,
    DROP COLUMN scope_label,
    ALTER COLUMN scope_id TYPE UUID USING scope_id::uuid,
    ADD CONSTRAINT ck_user_role_assignments_scope_type CHECK (scope_type IN ('company', 'branch'));

DROP INDEX IF EXISTS idx_permissions_group;

ALTER TABLE permissions
    DROP CONSTRAINT fk_permissions_group,
    DROP COLUMN scope_types,
    DROP COLUMN group_id;

DROP TABLE solution_scope_types;
DROP TABLE permission_groups;
