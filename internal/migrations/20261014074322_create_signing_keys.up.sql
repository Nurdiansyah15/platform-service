-- Kunci penandatangan JWT (spec §2.7). Private key tidak pernah disimpan di DB,
-- hanya referensinya ke key store (file lokal / KMS / secret manager).
CREATE TABLE signing_keys (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    kid             VARCHAR(100) NOT NULL,
    algorithm       VARCHAR(20)  NOT NULL DEFAULT 'RS256',
    public_key      TEXT         NOT NULL,
    private_key_ref TEXT         NOT NULL,
    status          VARCHAR(30)  NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    retired_at      TIMESTAMPTZ  NULL,

    CONSTRAINT uq_signing_keys_kid UNIQUE (kid),
    CONSTRAINT ck_signing_keys_status CHECK (status IN ('active', 'retiring', 'retired'))
);

-- Hanya boleh ada satu key active (dipakai menandatangani).
CREATE UNIQUE INDEX uq_signing_keys_single_active ON signing_keys (status) WHERE status = 'active';
