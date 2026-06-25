CREATE TABLE IF NOT EXISTS user_branch_assignments (
    id                               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id                           UUID NOT NULL,
    user_id                          UUID NOT NULL,
    branch_id                        UUID NOT NULL,
    is_primary                       BOOLEAN NOT NULL DEFAULT false,
    assigned_by                      VARCHAR(36) NOT NULL DEFAULT '',
    assigned_at                      TIMESTAMPTZ NOT NULL,
    revoked_at                       TIMESTAMPTZ,
    revoked_by                       VARCHAR(36),
    CONSTRAINT fk_user_branch_assignments_org_id FOREIGN KEY (org_id) REFERENCES organizations(id),
    CONSTRAINT fk_user_branch_assignments_user_id FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT fk_user_branch_assignments_branch_id FOREIGN KEY (branch_id) REFERENCES branches(id),
    CONSTRAINT uq_user_branch_assignments_user_id_branch_id UNIQUE (user_id, branch_id)
);

CREATE INDEX IF NOT EXISTS idx_user_branch_assignments_org_id ON user_branch_assignments(org_id);
CREATE INDEX IF NOT EXISTS idx_user_branch_assignments_user_id ON user_branch_assignments(user_id);
CREATE INDEX IF NOT EXISTS idx_user_branch_assignments_branch_id ON user_branch_assignments(branch_id);
CREATE INDEX IF NOT EXISTS idx_user_branch_assignments_user_id
    ON user_branch_assignments(user_id)
    WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_user_branch_assignments_user_id
    ON user_branch_assignments(user_id)
    WHERE is_primary = TRUE AND revoked_at IS NULL;