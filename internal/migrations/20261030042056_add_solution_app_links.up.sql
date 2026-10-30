-- Tautan aplikasi solution untuk SSO dari console.
-- launch_url: halaman awal solution (dibuka dari launcher / menu apps).
-- icon_url: ikon di launcher / menu apps.
-- redirect_uris: allowlist redirect_uri alur authorize; dicocokkan persis.
ALTER TABLE solutions
    ADD COLUMN launch_url    TEXT   NULL,
    ADD COLUMN icon_url      TEXT   NULL,
    ADD COLUMN redirect_uris TEXT[] NOT NULL DEFAULT '{}';
