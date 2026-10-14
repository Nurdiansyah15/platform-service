-- Outbox event (spec §2.9). Ditulis dalam transaksi yang sama dengan perubahan
-- data, lalu dikirim oleh worker (Fase 16).
-- tenant_id / solution_id sengaja tanpa FK: event tetap tersimpan sebagai
-- histori walau entitas sumbernya berubah.
CREATE TABLE events (
    id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type  VARCHAR(100) NOT NULL,
    tenant_id   UUID         NULL,
    solution_id UUID         NULL,
    payload     JSONB        NOT NULL,
    occurred_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_events_occurred_at ON events (occurred_at);
CREATE INDEX idx_events_tenant_id ON events (tenant_id, occurred_at);
