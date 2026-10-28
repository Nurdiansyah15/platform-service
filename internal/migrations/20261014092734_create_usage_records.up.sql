-- Laporan usage dari solution (spec §2.10): append-only, idempotent per
-- (solution_id, idempotency_key).
CREATE TABLE usage_records (
    id                    UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id            UUID          NOT NULL REFERENCES companies (id),
    solution_id           UUID          NOT NULL,
    benefit_id            UUID          NOT NULL,
    mode                  VARCHAR(30)   NOT NULL,
    quantity              NUMERIC(18,4) NOT NULL,
    idempotency_key       VARCHAR(150)  NOT NULL,
    reported_by_client_id UUID          NOT NULL REFERENCES solution_clients (id),
    occurred_at           TIMESTAMPTZ   NOT NULL,
    created_at            TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT fk_usage_records_benefit FOREIGN KEY (benefit_id, solution_id) REFERENCES benefits (id, solution_id),
    CONSTRAINT uq_usage_records_idempotency UNIQUE (solution_id, idempotency_key),
    CONSTRAINT ck_usage_records_mode CHECK (mode IN ('delta', 'absolute'))
);

CREATE INDEX idx_usage_records_company_benefit ON usage_records (company_id, benefit_id, occurred_at);
