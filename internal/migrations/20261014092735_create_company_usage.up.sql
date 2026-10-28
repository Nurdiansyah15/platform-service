-- Pemakaian terkini per tenant per benefit per periode (spec §2.10).
-- period_start untuk usage_reset = never diisi tanggal tetap 1970-01-01.
-- last_occurred_at (tambahan di luar spec): occurred_at laporan terbaru yang
-- diterapkan, agar laporan gauge yang datang terlambat tidak menimpa nilai lebih baru.
CREATE TABLE tenant_usage (
    tenant_id        UUID          NOT NULL REFERENCES tenants (id),
    solution_id      UUID          NOT NULL,
    benefit_id       UUID          NOT NULL,
    period_start     TIMESTAMPTZ   NOT NULL,
    period_end       TIMESTAMPTZ   NULL,
    used             NUMERIC(18,4) NOT NULL DEFAULT 0,
    last_occurred_at TIMESTAMPTZ   NULL,
    updated_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT pk_tenant_usage PRIMARY KEY (tenant_id, benefit_id, period_start),
    CONSTRAINT fk_tenant_usage_benefit FOREIGN KEY (benefit_id, solution_id) REFERENCES benefits (id, solution_id)
);
