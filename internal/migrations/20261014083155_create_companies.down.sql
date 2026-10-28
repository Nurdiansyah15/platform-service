ALTER TABLE auth_sessions DROP CONSTRAINT IF EXISTS fk_auth_sessions_company;
DROP TABLE IF EXISTS companies;
