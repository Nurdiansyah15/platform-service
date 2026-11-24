-- Sumber permission group: 'sync' (katalog dari backend solution) atau 'manual' (dibuat
-- platform admin). Sync hanya mengelola group ber-source sync; code yang bentrok dengan
-- group manual ditolak.
ALTER TABLE permission_groups
    ADD COLUMN source VARCHAR(10) NOT NULL DEFAULT 'sync',
    ADD CONSTRAINT ck_permission_groups_source CHECK (source IN ('sync', 'manual'));
