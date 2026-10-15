-- Pengiriman satu event ke satu endpoint (spec §2.9). At-least-once; retry dengan
-- exponential backoff, lalu 'dead'.
CREATE TABLE webhook_deliveries (
    id                  UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id            UUID        NOT NULL REFERENCES events (id),
    webhook_endpoint_id UUID        NOT NULL REFERENCES webhook_endpoints (id),
    status              VARCHAR(30) NOT NULL DEFAULT 'pending',
    attempt_count       INTEGER     NOT NULL DEFAULT 0,
    next_attempt_at     TIMESTAMPTZ NULL,
    last_response_code  INTEGER     NULL,
    last_error          TEXT        NULL,
    delivered_at        TIMESTAMPTZ NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_webhook_deliveries_event_endpoint UNIQUE (event_id, webhook_endpoint_id),
    CONSTRAINT ck_webhook_deliveries_status CHECK (status IN ('pending', 'success', 'failed', 'dead'))
);

CREATE INDEX idx_webhook_deliveries_pending ON webhook_deliveries (next_attempt_at)
    WHERE status IN ('pending', 'failed');
