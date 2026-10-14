-- Katalog permission per solution (spec §2.6). Permission tidak pernah dihapus;
-- yang hilang dari katalog menjadi 'deprecated' agar role lama tidak rusak.
CREATE TABLE permissions (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    solution_id UUID         NOT NULL REFERENCES solutions (id),
    code        VARCHAR(150) NOT NULL,
    name        VARCHAR(255) NOT NULL,
    description TEXT         NULL,
    status      VARCHAR(30)  NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_permissions_solution_code UNIQUE (solution_id, code),
    CONSTRAINT uq_permissions_id_solution UNIQUE (id, solution_id),
    CONSTRAINT ck_permissions_status CHECK (status IN ('active', 'deprecated'))
);

CREATE TRIGGER trg_permissions_updated_at
    BEFORE UPDATE ON permissions
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
