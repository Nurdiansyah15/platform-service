-- Keanggotaan user di company (spec §2.1). Soft delete = status 'inactive'.
CREATE TABLE company_members (
    company_id UUID        NOT NULL REFERENCES companies (id),
    user_id    UUID        NOT NULL REFERENCES users (id),
    status     VARCHAR(30) NOT NULL DEFAULT 'active',
    joined_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT pk_company_members PRIMARY KEY (company_id, user_id),
    CONSTRAINT ck_company_members_status CHECK (status IN ('invited', 'active', 'inactive'))
);

CREATE INDEX idx_company_members_user_id ON company_members (user_id);
