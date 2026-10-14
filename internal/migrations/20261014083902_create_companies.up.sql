-- Company milik tenant (spec §2.1). Soft delete via deleted_at.
-- code dinormalisasi UPPERCASE oleh domain; unik per tenant di antara company yang belum dihapus.
CREATE TABLE companies (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id  UUID         NOT NULL REFERENCES tenants (id),
    name       VARCHAR(255) NOT NULL,
    code       VARCHAR(100) NOT NULL,
    status     VARCHAR(30)  NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ  NULL,

    CONSTRAINT ck_companies_status CHECK (status IN ('active', 'inactive'))
);

CREATE UNIQUE INDEX uq_companies_tenant_code ON companies (tenant_id, code) WHERE deleted_at IS NULL;
CREATE INDEX idx_companies_tenant_id ON companies (tenant_id);

CREATE TRIGGER trg_companies_updated_at
    BEFORE UPDATE ON companies
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
