-- Penugasan role ke user dalam tenant pada scope tertentu (spec §2.6).
-- FK (tenant_id, user_id) → tenant_members: user wajib anggota tenant.
-- Validasi role vs subscription, kepemilikan custom role, dan scope_id dilakukan service layer.
CREATE TABLE user_role_assignments (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID        NOT NULL,
    user_id    UUID        NOT NULL,
    role_id    UUID        NOT NULL REFERENCES roles (id),
    scope_type VARCHAR(30) NOT NULL,
    scope_id   UUID        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_user_role_assignments_member FOREIGN KEY (tenant_id, user_id) REFERENCES tenant_members (tenant_id, user_id),
    CONSTRAINT uq_user_role_assignments UNIQUE (user_id, role_id, scope_type, scope_id),
    CONSTRAINT ck_user_role_assignments_scope_type CHECK (scope_type IN ('tenant', 'company', 'branch'))
);

CREATE INDEX idx_user_role_assignments_user_tenant ON user_role_assignments (user_id, tenant_id);
CREATE INDEX idx_user_role_assignments_scope ON user_role_assignments (scope_type, scope_id);
