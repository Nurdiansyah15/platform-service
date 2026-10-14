-- Nilai benefit pada saat pembelian (spec §2.5). Entitlement SELALU dihitung dari
-- tabel ini sehingga perubahan plan_benefits / addon_benefits tidak memengaruhi
-- subscription berjalan. subscription_addon_id NULL = berasal dari plan.
CREATE TABLE subscription_benefit_snapshots (
    id                    UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id       UUID        NOT NULL REFERENCES subscriptions (id),
    subscription_addon_id UUID        NULL REFERENCES subscription_addons (id),
    benefit_id            UUID        NOT NULL REFERENCES benefits (id),
    value                 JSONB       NOT NULL,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_sub_benefit_snapshots UNIQUE NULLS NOT DISTINCT (subscription_id, subscription_addon_id, benefit_id)
);

CREATE INDEX idx_sub_benefit_snapshots_subscription_id ON subscription_benefit_snapshots (subscription_id);
