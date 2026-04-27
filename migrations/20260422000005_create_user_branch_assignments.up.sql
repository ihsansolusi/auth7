-- Migration: Create user_branch_assignments table
-- Up

CREATE TABLE user_branch_assignments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL,
    branch_id       UUID NOT NULL,
    is_primary      BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, branch_id)
);

CREATE INDEX idx_user_branch_user ON user_branch_assignments(user_id);
CREATE INDEX idx_user_branch_branch ON user_branch_assignments(branch_id);
CREATE INDEX idx_user_branch_primary ON user_branch_assignments(user_id) WHERE is_primary = TRUE;

-- Down
DROP TABLE IF EXISTS user_branch_assignments;
