-- Master benefit per solution (spec §2.4).
-- UNIQUE (id, solution_id) = target composite FK dari plan_benefits / addon_benefits / usage
-- agar DB menolak relasi lintas solution.
CREATE TABLE benefits (
    id                   UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    solution_id          UUID         NOT NULL REFERENCES solutions (id),
    code                 VARCHAR(100) NOT NULL,
    name                 VARCHAR(255) NOT NULL,
    description          TEXT         NULL,
    value_type           VARCHAR(30)  NOT NULL,
    aggregation_strategy VARCHAR(30)  NOT NULL,
    unit                 VARCHAR(50)  NULL,
    metering             VARCHAR(30)  NOT NULL DEFAULT 'none',
    usage_reset          VARCHAR(30)  NOT NULL DEFAULT 'never',
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_benefits_solution_code UNIQUE (solution_id, code),
    CONSTRAINT uq_benefits_id_solution UNIQUE (id, solution_id),
    CONSTRAINT ck_benefits_value_type CHECK (value_type IN ('boolean', 'integer', 'decimal', 'string', 'json')),
    CONSTRAINT ck_benefits_aggregation_strategy CHECK (aggregation_strategy IN ('sum', 'max', 'min', 'override', 'any_true')),
    CONSTRAINT ck_benefits_metering CHECK (metering IN ('none', 'counter', 'gauge')),
    CONSTRAINT ck_benefits_usage_reset CHECK (usage_reset IN ('never', 'billing_period', 'monthly'))
);

CREATE INDEX idx_benefits_solution_id ON benefits (solution_id);

CREATE TRIGGER trg_benefits_updated_at
    BEFORE UPDATE ON benefits
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
