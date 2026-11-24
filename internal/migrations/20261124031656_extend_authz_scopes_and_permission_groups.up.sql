-- Perluasan RBAC:
--   * permission_groups      → pengelompokan permission untuk UI (didefinisikan solution lewat sync)
--   * solution_scope_types   → scope type custom per solution (company & branch tetap built-in)
--   * permissions.group_id / scope_types → grup & scope yang berlaku untuk permission ('{}' = semua scope)
--   * user_role_assignments.scope_id → string bebas (scope custom milik solution), + scope_label
--     dan scope_parent_id (branch induk untuk scope custom ber-parent branch)
CREATE TABLE permission_groups (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    solution_id UUID         NOT NULL REFERENCES solutions (id),
    code        VARCHAR(100) NOT NULL,
    name        VARCHAR(255) NOT NULL,
    description TEXT         NULL,
    sort_order  INT          NOT NULL DEFAULT 0,
    status      VARCHAR(30)  NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_permission_groups_solution_code UNIQUE (solution_id, code),
    CONSTRAINT uq_permission_groups_id_solution UNIQUE (id, solution_id),
    CONSTRAINT ck_permission_groups_status CHECK (status IN ('active', 'deprecated'))
);

CREATE TRIGGER trg_permission_groups_updated_at
    BEFORE UPDATE ON permission_groups
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE solution_scope_types (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    solution_id UUID         NOT NULL REFERENCES solutions (id),
    code        VARCHAR(30)  NOT NULL,
    name        VARCHAR(255) NOT NULL,
    description TEXT         NULL,
    parent_type VARCHAR(30)  NOT NULL DEFAULT 'company',
    sort_order  INT          NOT NULL DEFAULT 0,
    status      VARCHAR(30)  NOT NULL DEFAULT 'active',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_solution_scope_types_solution_code UNIQUE (solution_id, code),
    CONSTRAINT ck_solution_scope_types_code CHECK (code ~ '^[a-z][a-z0-9_]*$' AND code NOT IN ('company', 'branch')),
    CONSTRAINT ck_solution_scope_types_parent CHECK (parent_type IN ('company', 'branch')),
    CONSTRAINT ck_solution_scope_types_status CHECK (status IN ('active', 'deprecated'))
);

CREATE TRIGGER trg_solution_scope_types_updated_at
    BEFORE UPDATE ON solution_scope_types
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

ALTER TABLE permissions
    ADD COLUMN group_id    UUID   NULL,
    ADD COLUMN scope_types TEXT[] NOT NULL DEFAULT '{}',
    ADD CONSTRAINT fk_permissions_group FOREIGN KEY (group_id, solution_id) REFERENCES permission_groups (id, solution_id);

CREATE INDEX idx_permissions_group ON permissions (group_id);

ALTER TABLE user_role_assignments
    DROP CONSTRAINT ck_user_role_assignments_scope_type,
    ALTER COLUMN scope_id TYPE VARCHAR(150) USING scope_id::text,
    ADD COLUMN scope_label VARCHAR(255) NULL,
    ADD COLUMN scope_parent_id VARCHAR(150) NULL,
    ADD CONSTRAINT ck_user_role_assignments_scope_type CHECK (scope_type ~ '^[a-z][a-z0-9_]*$'),
    ADD CONSTRAINT ck_user_role_assignments_scope_id CHECK (scope_id <> '');
