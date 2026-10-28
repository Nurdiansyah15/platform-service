ALTER TABLE auth_sessions DROP CONSTRAINT IF EXISTS fk_auth_sessions_tenant;
DROP TABLE IF EXISTS tenants;
