-- Subscription company ke solution (spec §2.5).
CREATE TABLE subscriptions (
    id                   UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    company_id           UUID          NOT NULL REFERENCES companies (id),
    solution_id          UUID          NOT NULL REFERENCES solutions (id),
    plan_id              UUID          NOT NULL,
    status               VARCHAR(30)   NOT NULL DEFAULT 'active',
    billing_cycle        VARCHAR(30)   NOT NULL,
    current_period_start TIMESTAMPTZ   NULL,
    current_period_end   TIMESTAMPTZ   NULL,
    price_at_purchase    NUMERIC(18,2) NOT NULL,
    currency             VARCHAR(3)    NOT NULL DEFAULT 'IDR',
    created_at           TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_subscriptions_id_solution UNIQUE (id, solution_id),
    CONSTRAINT fk_subscriptions_plan FOREIGN KEY (plan_id, solution_id) REFERENCES plans (id, solution_id),
    CONSTRAINT ck_subscriptions_status CHECK (status IN ('trialing', 'active', 'past_due', 'canceled', 'expired')),
    CONSTRAINT ck_subscriptions_billing_cycle CHECK (billing_cycle IN ('monthly', 'yearly'))
);

-- Keputusan Fase 10: maksimal satu subscription "hidup" per company per solution.
CREATE UNIQUE INDEX uq_subscriptions_live_company_solution ON subscriptions (company_id, solution_id)
    WHERE status IN ('trialing', 'active', 'past_due');
CREATE INDEX idx_subscriptions_company_id ON subscriptions (company_id);
CREATE INDEX idx_subscriptions_solution_id ON subscriptions (solution_id);

CREATE TRIGGER trg_subscriptions_updated_at
    BEFORE UPDATE ON subscriptions
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
