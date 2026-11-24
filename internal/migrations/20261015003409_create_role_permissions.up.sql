-- Permission milik role (spec §2.6). Composite FK menjamin role & permission
-- berasal dari solution yang sama.
--
-- scope_type: level tempat permission berlaku untuk role ini — 'company' (seluruh company),
-- 'branch', atau code scope type custom solution (solution_scope_types; divalidasi usecase).
-- Nilainya (branch mana, department mana) disimpan per assignment di
-- user_role_assignments.scope_values.
CREATE TABLE role_permissions (
    role_id       UUID        NOT NULL,
    permission_id UUID        NOT NULL,
    solution_id   UUID        NOT NULL,
    scope_type    VARCHAR(30) NOT NULL DEFAULT 'company',

    CONSTRAINT pk_role_permissions PRIMARY KEY (role_id, permission_id),
    CONSTRAINT fk_role_permissions_role FOREIGN KEY (role_id, solution_id) REFERENCES roles (id, solution_id),
    CONSTRAINT fk_role_permissions_permission FOREIGN KEY (permission_id, solution_id) REFERENCES permissions (id, solution_id),
    CONSTRAINT ck_role_permissions_scope_type CHECK (scope_type ~ '^[a-z][a-z0-9_]*$')
);

CREATE INDEX idx_role_permissions_permission_id ON role_permissions (permission_id);
