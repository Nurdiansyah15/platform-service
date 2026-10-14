-- Add-on yang dipasang ke subscription (spec §2.5). Soft delete = status 'canceled'.
CREATE TABLE subscription_addons (
    id                     UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    subscription_id        UUID          NOT NULL,
    addon_id               UUID          NOT NULL,
    solution_id            UUID          NOT NULL,
    quantity               INTEGER       NOT NULL DEFAULT 1,
    unit_price_at_purchase NUMERIC(18,2) NOT NULL,
    currency               VARCHAR(3)    NOT NULL DEFAULT 'IDR',
    status                 VARCHAR(30)   NOT NULL DEFAULT 'active',
    created_at             TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at             TIMESTAMPTZ   NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_subscription_addons_subscription_addon UNIQUE (subscription_id, addon_id),
    CONSTRAINT fk_subscription_addons_subscription FOREIGN KEY (subscription_id, solution_id) REFERENCES subscriptions (id, solution_id),
    CONSTRAINT fk_subscription_addons_addon FOREIGN KEY (addon_id, solution_id) REFERENCES addons (id, solution_id),
    CONSTRAINT ck_subscription_addons_quantity CHECK (quantity > 0),
    CONSTRAINT ck_subscription_addons_status CHECK (status IN ('active', 'canceled'))
);

CREATE INDEX idx_subscription_addons_subscription_id ON subscription_addons (subscription_id);

CREATE TRIGGER trg_subscription_addons_updated_at
    BEFORE UPDATE ON subscription_addons
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
