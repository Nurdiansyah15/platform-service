-- Solution Catalog (spec §2.2).
CREATE TABLE solutions (
    id                         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    code                       VARCHAR(100) NOT NULL,
    name                       VARCHAR(255) NOT NULL,
    description                TEXT         NULL,
    status                     VARCHAR(30)  NOT NULL DEFAULT 'active',
    permission_catalog_version VARCHAR(50)  NULL,
    created_at                 TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at                 TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_solutions_code UNIQUE (code),
    CONSTRAINT ck_solutions_status CHECK (status IN ('active', 'inactive'))
);

CREATE TRIGGER trg_solutions_updated_at
    BEFORE UPDATE ON solutions
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
