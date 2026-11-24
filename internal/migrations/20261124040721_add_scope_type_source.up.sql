-- Sumber scope type custom: 'sync' (katalog dari backend solution) atau 'manual' (dibuat
-- platform admin). Sync hanya mengelola scope type ber-source sync; code yang bentrok dengan
-- scope type manual ditolak.
ALTER TABLE solution_scope_types
    ADD COLUMN source VARCHAR(10) NOT NULL DEFAULT 'sync',
    ADD CONSTRAINT ck_solution_scope_types_source CHECK (source IN ('sync', 'manual'));
