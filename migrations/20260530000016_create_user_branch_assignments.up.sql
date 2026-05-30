CREATE TABLE IF NOT EXISTS user_branch_assignments (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    branch_id   UUID        NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    org_id      UUID        NOT NULL REFERENCES organizations(id),
    is_primary  BOOLEAN     NOT NULL DEFAULT false,
    assigned_by VARCHAR(36) NOT NULL DEFAULT 'system',
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at  TIMESTAMPTZ,
    revoked_by  VARCHAR(36) NOT NULL DEFAULT ''
);

-- Partial unique: one active assignment per (user, branch)
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_branch_assignments_active
    ON user_branch_assignments(user_id, branch_id)
    WHERE revoked_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_branch_assignments_user_id   ON user_branch_assignments(user_id);
CREATE INDEX IF NOT EXISTS idx_user_branch_assignments_branch_id ON user_branch_assignments(branch_id);
CREATE INDEX IF NOT EXISTS idx_user_branch_assignments_primary   ON user_branch_assignments(user_id) WHERE is_primary = true;
