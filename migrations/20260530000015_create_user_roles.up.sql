CREATE TABLE IF NOT EXISTS user_roles (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id    UUID        NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    org_id     UUID        NOT NULL REFERENCES organizations(id),
    branch_id  UUID        REFERENCES branches(id) ON DELETE SET NULL,
    granted_by VARCHAR(36) NOT NULL DEFAULT 'system',
    granted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revoked_at TIMESTAMPTZ,
    revoked_by VARCHAR(36) NOT NULL DEFAULT ''
);

-- Partial unique: one active grant per (user, role, org, branch)
CREATE UNIQUE INDEX IF NOT EXISTS idx_user_roles_active_grant
    ON user_roles(user_id, role_id, org_id, COALESCE(branch_id, '00000000-0000-0000-0000-000000000000'))
    WHERE revoked_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_user_roles_user_id  ON user_roles(user_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_role_id  ON user_roles(role_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_org_id   ON user_roles(org_id);
