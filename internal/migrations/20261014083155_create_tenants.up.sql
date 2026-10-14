-- Tenant (spec §2.1). slug ter-normalisasi lowercase oleh domain.
CREATE TABLE tenants (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(255) NOT NULL,
    slug       VARCHAR(100) NOT NULL,
    status     VARCHAR(30)  NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_tenants_slug UNIQUE (slug),
    CONSTRAINT ck_tenants_status CHECK (status IN ('active', 'suspended', 'closed'))
);

CREATE TRIGGER trg_tenants_updated_at
    BEFORE UPDATE ON tenants
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- auth_sessions.tenant_id dibuat di Fase 3 sebelum tabel tenants ada.
ALTER TABLE auth_sessions
    ADD CONSTRAINT fk_auth_sessions_tenant FOREIGN KEY (tenant_id) REFERENCES tenants (id);
