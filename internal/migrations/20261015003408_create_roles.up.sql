-- Role per solution (spec §2.6). tenant_id NULL = role global (system / template
-- platform); terisi = custom role milik tenant. Soft delete via deleted_at.
CREATE TABLE roles (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    solution_id UUID         NOT NULL REFERENCES solutions (id),
    tenant_id   UUID         NULL REFERENCES tenants (id),
    code        VARCHAR(100) NOT NULL,
    name        VARCHAR(255) NOT NULL,
    description TEXT         NULL,
    is_system   BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at  TIMESTAMPTZ  NULL,

    CONSTRAINT uq_roles_id_solution UNIQUE (id, solution_id),
    -- Role sistem tidak boleh milik tenant.
    CONSTRAINT ck_roles_system_global CHECK (NOT is_system OR tenant_id IS NULL)
);

CREATE UNIQUE INDEX uq_roles_global_code ON roles (solution_id, code)
    WHERE tenant_id IS NULL AND deleted_at IS NULL;
CREATE UNIQUE INDEX uq_roles_tenant_code ON roles (solution_id, tenant_id, code)
    WHERE tenant_id IS NOT NULL AND deleted_at IS NULL;
CREATE INDEX idx_roles_solution_id ON roles (solution_id);
CREATE INDEX idx_roles_tenant_solution ON roles (tenant_id, solution_id);

CREATE TRIGGER trg_roles_updated_at
    BEFORE UPDATE ON roles
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
