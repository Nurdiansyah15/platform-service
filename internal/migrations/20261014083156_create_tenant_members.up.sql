-- Keanggotaan user di tenant (spec §2.1). Soft delete = status 'inactive'.
CREATE TABLE tenant_members (
    tenant_id UUID        NOT NULL REFERENCES tenants (id),
    user_id   UUID        NOT NULL REFERENCES users (id),
    status    VARCHAR(30) NOT NULL DEFAULT 'active',
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT pk_tenant_members PRIMARY KEY (tenant_id, user_id),
    CONSTRAINT ck_tenant_members_status CHECK (status IN ('invited', 'active', 'inactive'))
);

CREATE INDEX idx_tenant_members_user_id ON tenant_members (user_id);
