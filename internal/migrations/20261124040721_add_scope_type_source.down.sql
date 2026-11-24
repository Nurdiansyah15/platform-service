ALTER TABLE solution_scope_types
    DROP CONSTRAINT ck_solution_scope_types_source,
    DROP COLUMN source;
