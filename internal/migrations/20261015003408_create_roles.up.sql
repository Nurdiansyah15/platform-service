-- Role per solution (spec §2.6). company_id NULL = role global (system / template
-- platform); terisi = custom role milik company. Soft delete via deleted_at.
CREATE TABLE roles (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    solution_id UUID         NOT NULL REFERENCES solutions (id),
    company_id  UUID         NULL REFERENCES companies (id),
    code        VARCHAR(100) NOT NULL,
    name        VARCHAR(255) NOT NULL,
    description TEXT         NULL,
    is_system   BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ  NULL,

    CONSTRAINT uq_roles_id_solution UNIQUE (id, solution_id),
    -- Role sistem tidak boleh milik company.
    CONSTRAINT ck_roles_system_global CHECK (NOT is_system OR company_id IS NULL)
);

CREATE UNIQUE INDEX uq_roles_global_code ON roles (solution_id, code)
    WHERE company_id IS NULL AND deleted_at IS NULL;
CREATE UNIQUE INDEX uq_roles_company_code ON roles (solution_id, company_id, code)
    WHERE company_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_roles_solution_id ON roles (solution_id);
CREATE INDEX idx_roles_company_solution ON roles (company_id, solution_id);

CREATE TRIGGER trg_roles_updated_at
    BEFORE UPDATE ON roles
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
