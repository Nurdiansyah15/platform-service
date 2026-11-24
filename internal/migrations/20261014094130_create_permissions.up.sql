-- Katalog permission per solution (spec §2.6) beserta pengelompokan (permission_groups) dan
-- scope type custom (solution_scope_types). Entri katalog tidak pernah dihapus; yang hilang
-- dari katalog / dihapus admin menjadi 'deprecated' agar role lama tidak rusak.
--
-- source: 'sync' = dikirim backend solution lewat sync katalog (read-only di platform);
--         'manual' = dibuat platform admin (tidak disentuh sync).
--
-- Scope tidak disimpan di permission: role menentukan scope tiap permission
-- (role_permissions.scope_type), assignment menyimpan nilainya (user_role_assignments.scope_values).

CREATE TABLE permission_groups (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    solution_id UUID         NOT NULL REFERENCES solutions (id),
    code        VARCHAR(100) NOT NULL,
    name        VARCHAR(255) NOT NULL,
    description TEXT         NULL,
    sort_order  INT          NOT NULL DEFAULT 0,
    status      VARCHAR(30)  NOT NULL DEFAULT 'active',
    source      VARCHAR(10)  NOT NULL DEFAULT 'sync',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_permission_groups_solution_code UNIQUE (solution_id, code),
    CONSTRAINT uq_permission_groups_id_solution UNIQUE (id, solution_id),
    CONSTRAINT ck_permission_groups_status CHECK (status IN ('active', 'deprecated')),
    CONSTRAINT ck_permission_groups_source CHECK (source IN ('sync', 'manual'))
);

CREATE TRIGGER trg_permission_groups_updated_at
    BEFORE UPDATE ON permission_groups
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Scope type custom (mis. department, project). company & branch built-in, tidak disimpan.
-- parent_type: assignment di parent (company selalu; branch bila parent = branch) ikut mencakup.
CREATE TABLE solution_scope_types (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    solution_id UUID         NOT NULL REFERENCES solutions (id),
    code        VARCHAR(30)  NOT NULL,
    name        VARCHAR(255) NOT NULL,
    description TEXT         NULL,
    parent_type VARCHAR(30)  NOT NULL DEFAULT 'company',
    sort_order  INT          NOT NULL DEFAULT 0,
    status      VARCHAR(30)  NOT NULL DEFAULT 'active',
    source      VARCHAR(10)  NOT NULL DEFAULT 'sync',
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_solution_scope_types_solution_code UNIQUE (solution_id, code),
    CONSTRAINT ck_solution_scope_types_code CHECK (code ~ '^[a-z][a-z0-9_]*$' AND code NOT IN ('company', 'branch')),
    CONSTRAINT ck_solution_scope_types_parent CHECK (parent_type IN ('company', 'branch')),
    CONSTRAINT ck_solution_scope_types_status CHECK (status IN ('active', 'deprecated')),
    CONSTRAINT ck_solution_scope_types_source CHECK (source IN ('sync', 'manual'))
);

CREATE TRIGGER trg_solution_scope_types_updated_at
    BEFORE UPDATE ON solution_scope_types
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE permissions (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    solution_id UUID         NOT NULL REFERENCES solutions (id),
    code        VARCHAR(150) NOT NULL,
    name        VARCHAR(255) NOT NULL,
    description TEXT         NULL,
    status      VARCHAR(30)  NOT NULL DEFAULT 'active',
    source      VARCHAR(10)  NOT NULL DEFAULT 'sync',
    group_id    UUID         NULL,
    created_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_permissions_solution_code UNIQUE (solution_id, code),
    CONSTRAINT uq_permissions_id_solution UNIQUE (id, solution_id),
    CONSTRAINT ck_permissions_status CHECK (status IN ('active', 'deprecated')),
    CONSTRAINT ck_permissions_source CHECK (source IN ('sync', 'manual')),
    CONSTRAINT fk_permissions_group FOREIGN KEY (group_id, solution_id) REFERENCES permission_groups (id, solution_id)
);

CREATE INDEX idx_permissions_group ON permissions (group_id);

CREATE TRIGGER trg_permissions_updated_at
    BEFORE UPDATE ON permissions
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
