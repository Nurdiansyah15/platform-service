DROP TABLE IF EXISTS auth_authorization_codes;
DROP INDEX IF EXISTS idx_auth_sessions_parent;
ALTER TABLE auth_sessions DROP COLUMN IF EXISTS parent_session_id;
