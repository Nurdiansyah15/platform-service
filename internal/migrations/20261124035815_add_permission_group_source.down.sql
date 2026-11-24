ALTER TABLE permission_groups
    DROP CONSTRAINT ck_permission_groups_source,
    DROP COLUMN source;
