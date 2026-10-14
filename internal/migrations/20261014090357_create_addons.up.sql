-- Add-on per solution (spec §2.3). Soft delete via deleted_at.
CREATE TABLE addons (
    id            UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    solution_id   UUID          NOT NULL REFERENCES solutions (id),
    code          VARCHAR(100)  NOT NULL,
    name          VARCHAR(255)  NOT NULL,
    description   TEXT          NULL,
    price         NUMERIC(18,2) NOT NULL DEFAULT 0,
    currency      VARCHAR(3)    NOT NULL DEFAULT 'IDR',
    billing_cycle VARCHAR(30)   NOT NULL DEFAULT 'monthly',
    is_active     BOOLEAN       NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    deleted_at    TIMESTAMPTZ   NULL,

    CONSTRAINT uq_addons_id_solution UNIQUE (id, solution_id),
    CONSTRAINT ck_addons_billing_cycle CHECK (billing_cycle IN ('monthly', 'yearly')),
    CONSTRAINT ck_addons_price CHECK (price >= 0)
);

CREATE UNIQUE INDEX uq_addons_solution_code ON addons (solution_id, code) WHERE deleted_at IS NULL;
CREATE INDEX idx_addons_solution_id ON addons (solution_id);

CREATE TRIGGER trg_addons_updated_at
    BEFORE UPDATE ON addons
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
