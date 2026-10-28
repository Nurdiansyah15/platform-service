-- Company (spec §2.1) — entitas tertinggi (menggantikan tenant). slug
-- ter-normalisasi lowercase oleh domain.
CREATE TABLE companies (
    id         UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name       VARCHAR(255) NOT NULL,
    slug       VARCHAR(100) NOT NULL,
    status     VARCHAR(30)  NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_companies_slug UNIQUE (slug),
    CONSTRAINT ck_companies_status CHECK (status IN ('active', 'suspended', 'closed'))
);

CREATE TRIGGER trg_companies_updated_at
    BEFORE UPDATE ON companies
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- auth_sessions.company_id dibuat di Fase 3 sebelum tabel companies ada.
ALTER TABLE auth_sessions
    ADD CONSTRAINT fk_auth_sessions_company FOREIGN KEY (company_id) REFERENCES companies (id);
