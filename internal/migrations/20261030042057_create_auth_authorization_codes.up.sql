-- Sesi turunan: sesi solution yang dibuat lewat SSO menyimpan sesi console asalnya,
-- agar logout console ikut mencabut sesi solution.
ALTER TABLE auth_sessions
    ADD COLUMN parent_session_id UUID NULL REFERENCES auth_sessions (id);

CREATE INDEX idx_auth_sessions_parent ON auth_sessions (parent_session_id) WHERE parent_session_id IS NOT NULL;

-- Kode otorisasi sekali pakai (OAuth2 authorization code + PKCE) untuk SSO console → solution.
-- code_hash = SHA-256 hex dari kode mentah; kode mentah tidak disimpan.
CREATE TABLE auth_authorization_codes (
    id                    UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    code_hash             TEXT        NOT NULL,
    user_id               UUID        NOT NULL REFERENCES users (id),
    solution_id           UUID        NOT NULL REFERENCES solutions (id),
    company_id            UUID        NULL REFERENCES companies (id),
    parent_session_id     UUID        NOT NULL REFERENCES auth_sessions (id),
    redirect_uri          TEXT        NOT NULL,
    code_challenge        TEXT        NOT NULL,
    code_challenge_method VARCHAR(10) NOT NULL,
    expires_at            TIMESTAMPTZ NOT NULL,
    used_at               TIMESTAMPTZ NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_auth_authorization_codes_code_hash UNIQUE (code_hash),
    CONSTRAINT ck_auth_authorization_codes_method CHECK (code_challenge_method IN ('S256'))
);

CREATE INDEX idx_auth_authorization_codes_expires_at ON auth_authorization_codes (expires_at);
