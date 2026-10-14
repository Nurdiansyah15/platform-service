-- Kredensial service-to-service untuk backend solution (spec §2.8).
-- client_secret_hash = SHA-256 hex dari secret acak 256 bit; secret mentah hanya
-- ditampilkan sekali saat create / rotate.
CREATE TABLE solution_clients (
    id                 UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    solution_id        UUID         NOT NULL REFERENCES solutions (id),
    name               VARCHAR(255) NOT NULL,
    client_id          VARCHAR(100) NOT NULL,
    client_secret_hash TEXT         NOT NULL,
    scopes             TEXT[]       NOT NULL DEFAULT '{}',
    status             VARCHAR(30)  NOT NULL DEFAULT 'active',
    last_used_at       TIMESTAMPTZ  NULL,
    created_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    revoked_at         TIMESTAMPTZ  NULL,

    CONSTRAINT uq_solution_clients_client_id UNIQUE (client_id),
    CONSTRAINT ck_solution_clients_status CHECK (status IN ('active', 'revoked'))
);

CREATE INDEX idx_solution_clients_solution_id ON solution_clients (solution_id);

CREATE TRIGGER trg_solution_clients_updated_at
    BEFORE UPDATE ON solution_clients
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
