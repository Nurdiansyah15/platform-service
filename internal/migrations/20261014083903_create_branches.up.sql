-- Branch milik company (spec §2.1). Soft delete via deleted_at.
CREATE TABLE branches (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id UUID         NOT NULL REFERENCES companies (id),
    name       VARCHAR(255) NOT NULL,
    code       VARCHAR(100) NOT NULL,
    status     VARCHAR(30)  NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ  NULL,

    CONSTRAINT ck_branches_status CHECK (status IN ('active', 'inactive'))
);

CREATE UNIQUE INDEX uq_branches_company_code ON branches (company_id, code) WHERE deleted_at IS NULL;
CREATE INDEX idx_branches_company_id ON branches (company_id);

CREATE TRIGGER trg_branches_updated_at
    BEFORE UPDATE ON branches
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
