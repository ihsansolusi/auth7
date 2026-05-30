CREATE TABLE IF NOT EXISTS branch_default_roles (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id  UUID        NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    role_id    UUID        NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    is_default BOOLEAN     NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_branch_default_roles UNIQUE (branch_id, role_id)
);

CREATE INDEX IF NOT EXISTS idx_branch_default_roles_branch_id ON branch_default_roles(branch_id);
