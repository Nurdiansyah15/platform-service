-- Permission milik role (spec §2.6). Composite FK menjamin role & permission
-- berasal dari solution yang sama.
CREATE TABLE role_permissions (
    role_id       UUID NOT NULL,
    permission_id UUID NOT NULL,
    solution_id   UUID NOT NULL,

    CONSTRAINT pk_role_permissions PRIMARY KEY (role_id, permission_id),
    CONSTRAINT fk_role_permissions_role FOREIGN KEY (role_id, solution_id) REFERENCES roles (id, solution_id),
    CONSTRAINT fk_role_permissions_permission FOREIGN KEY (permission_id, solution_id) REFERENCES permissions (id, solution_id)
);

CREATE INDEX idx_role_permissions_permission_id ON role_permissions (permission_id);
