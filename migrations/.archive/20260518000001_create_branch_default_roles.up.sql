-- Migration: branch_default_roles
-- Stores default role templates per branch.
-- When users are assigned to a branch, the bos7-enterprise admin UI suggests
-- (or future automation will auto-apply) these roles.

CREATE TABLE IF NOT EXISTS branch_default_roles (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    branch_id  UUID NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    role_id    UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    is_default BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (branch_id, role_id)
);

CREATE INDEX IF NOT EXISTS idx_branch_default_roles_branch ON branch_default_roles(branch_id);

COMMENT ON TABLE branch_default_roles IS 'Default roles assigned/suggested per branch — consumed by /admin/v1/branches/:id/default-roles';
