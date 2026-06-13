CREATE TABLE IF NOT EXISTS branch_default_roles (
    id                               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id                        UUID NOT NULL,
    role_id                          UUID NOT NULL,
    is_default                       BOOLEAN NOT NULL DEFAULT false,
    created_at                       TIMESTAMPTZ NOT NULL,
    CONSTRAINT fk_branch_default_roles_branch_id FOREIGN KEY (branch_id) REFERENCES branches(id),
    CONSTRAINT fk_branch_default_roles_role_id FOREIGN KEY (role_id) REFERENCES roles(id),
    CONSTRAINT uq_branch_default_roles_branch_id_role_id UNIQUE (branch_id, role_id)
);

CREATE INDEX IF NOT EXISTS idx_branch_default_roles_branch_id ON branch_default_roles(branch_id);
CREATE INDEX IF NOT EXISTS idx_branch_default_roles_role_id ON branch_default_roles(role_id);