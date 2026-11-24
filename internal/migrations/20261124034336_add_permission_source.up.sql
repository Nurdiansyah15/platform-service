-- Sumber permission: 'sync' (dikirim backend solution lewat sync katalog) atau 'manual'
-- (dibuat platform admin). Sync hanya mengelola permission ber-source sync; code yang
-- bentrok dengan permission manual ditolak.
ALTER TABLE permissions
    ADD COLUMN source VARCHAR(10) NOT NULL DEFAULT 'sync',
    ADD CONSTRAINT ck_permissions_source CHECK (source IN ('sync', 'manual'));
