CREATE TABLE IF NOT EXISTS user_roles (
    id                               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id                           UUID NOT NULL,
    user_id                          UUID NOT NULL,
    role_id                          UUID NOT NULL,
    branch_id                        UUID,
    granted_by                       VARCHAR(36) NOT NULL DEFAULT '',
    granted_at                       TIMESTAMPTZ NOT NULL,
    revoked_at                       TIMESTAMPTZ,
    revoked_by                       VARCHAR(36),
    CONSTRAINT fk_user_roles_org_id FOREIGN KEY (org_id) REFERENCES organizations(id),
    CONSTRAINT fk_user_roles_user_id FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT fk_user_roles_role_id FOREIGN KEY (role_id) REFERENCES roles(id),
    CONSTRAINT fk_user_roles_branch_id FOREIGN KEY (branch_id) REFERENCES branches(id)
);

CREATE INDEX IF NOT EXISTS idx_user_roles_org_id ON user_roles(org_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_user_id ON user_roles(user_id);
CREATE INDEX IF NOT EXISTS idx_user_roles_role_id ON user_roles(role_id);
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_roles_user_id_role_id_branch_id_null
    ON user_roles(user_id, role_id) WHERE branch_id IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS uq_user_roles_user_id_role_id_branch_id_nn
    ON user_roles(user_id, role_id, branch_id) WHERE branch_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_user_roles_user_id
    ON user_roles(user_id)
    WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_user_roles_branch_id
    ON user_roles(branch_id)
    WHERE branch_id IS NOT NULL;