ALTER TABLE permissions
    DROP CONSTRAINT ck_permissions_source,
    DROP COLUMN source;
