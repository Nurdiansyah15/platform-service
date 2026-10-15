-- Endpoint webhook milik solution (spec §2.9). Secret dienkripsi (AES-256-GCM)
-- untuk tanda tangan HMAC; secret mentah hanya ditampilkan sekali.
CREATE TABLE webhook_endpoints (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    solution_id      UUID        NOT NULL REFERENCES solutions (id),
    url              TEXT        NOT NULL,
    secret_encrypted TEXT        NOT NULL,
    event_types      TEXT[]      NOT NULL,
    status           VARCHAR(30) NOT NULL DEFAULT 'active',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT ck_webhook_endpoints_status CHECK (status IN ('active', 'disabled'))
);

CREATE INDEX idx_webhook_endpoints_solution_id ON webhook_endpoints (solution_id);

CREATE TRIGGER trg_webhook_endpoints_updated_at
    BEFORE UPDATE ON webhook_endpoints
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
