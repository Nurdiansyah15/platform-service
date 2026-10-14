-- Sesi login user pada satu solution (spec §2.7). id dipakai sebagai claim sid.
-- refresh_token_hash = SHA-256 hex dari refresh token opaque; token mentah tidak disimpan.
-- tenant_id: FK ke tenants ditambahkan saat tabel tenants dibuat (Fase 5).
CREATE TABLE auth_sessions (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id            UUID        NOT NULL REFERENCES users (id),
    solution_id        UUID        NOT NULL REFERENCES solutions (id),
    tenant_id          UUID        NULL,
    refresh_token_hash TEXT        NOT NULL,
    user_agent         TEXT        NULL,
    ip_address         INET        NULL,
    expires_at         TIMESTAMPTZ NOT NULL,
    last_used_at       TIMESTAMPTZ NULL,
    revoked_at         TIMESTAMPTZ NULL,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_auth_sessions_refresh_token_hash UNIQUE (refresh_token_hash)
);

CREATE INDEX idx_auth_sessions_user_id ON auth_sessions (user_id);
CREATE INDEX idx_auth_sessions_active ON auth_sessions (user_id, solution_id) WHERE revoked_at IS NULL;
