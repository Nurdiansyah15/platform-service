-- Identity (spec §2.1). Email disimpan ter-normalisasi (lowercase) oleh domain.
-- is_platform_admin: tambahan di luar spec — penanda administrator platform.
CREATE TABLE users (
    id                UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    email             VARCHAR(255) NOT NULL,
    name              VARCHAR(255) NOT NULL,
    password_hash     TEXT         NULL,
    status            VARCHAR(30)  NOT NULL DEFAULT 'active',
    is_platform_admin BOOLEAN      NOT NULL DEFAULT FALSE,
    created_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at        TIMESTAMPTZ  NOT NULL DEFAULT NOW(),

    CONSTRAINT uq_users_email UNIQUE (email),
    CONSTRAINT ck_users_email_lowercase CHECK (email = LOWER(email)),
    CONSTRAINT ck_users_status CHECK (status IN ('active', 'inactive', 'suspended'))
);

CREATE TRIGGER trg_users_updated_at
    BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
