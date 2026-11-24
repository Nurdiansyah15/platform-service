-- Penugasan role ke user dalam company (spec §2.6).
-- FK (company_id, user_id) → company_members: user wajib anggota company.
--
-- scope_values: nilai untuk setiap scope_type non-company yang dipakai role
-- (role_permissions.scope_type), mis. {"branch": "<uuid>", "department": "42"}. Permission
-- ber-scope company tidak butuh nilai. scope_labels: nama tampilan opsional untuk nilai scope
-- custom, mis. {"department": "Keuangan"}. Role yang sama untuk nilai berbeda = assignment lain.
-- Validasi role vs company, kelengkapan & kepemilikan nilai scope dilakukan service layer.
CREATE TABLE user_role_assignments (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id   UUID        NOT NULL,
    user_id      UUID        NOT NULL,
    role_id      UUID        NOT NULL REFERENCES roles (id),
    scope_values JSONB       NOT NULL DEFAULT '{}',
    scope_labels JSONB       NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_user_role_assignments_member FOREIGN KEY (company_id, user_id) REFERENCES company_members (company_id, user_id),
    CONSTRAINT uq_user_role_assignments UNIQUE (user_id, role_id, company_id, scope_values),
    CONSTRAINT ck_user_role_assignments_scope_values CHECK (jsonb_typeof(scope_values) = 'object'),
    CONSTRAINT ck_user_role_assignments_scope_labels CHECK (jsonb_typeof(scope_labels) = 'object')
);

CREATE INDEX idx_user_role_assignments_user_company ON user_role_assignments (user_id, company_id);
CREATE INDEX idx_user_role_assignments_role ON user_role_assignments (role_id);
CREATE INDEX idx_user_role_assignments_scope_values ON user_role_assignments USING GIN (scope_values);
