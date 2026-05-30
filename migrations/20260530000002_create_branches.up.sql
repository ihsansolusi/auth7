-- Minimal projection of enterprise branches.
-- auth7 owns identity/access data only; branch hierarchy lives in enterprise domain.
CREATE TABLE IF NOT EXISTS branches (
    id          UUID        PRIMARY KEY,
    org_id      UUID        NOT NULL REFERENCES organizations(id),
    branch_code VARCHAR(20) NOT NULL,
    is_active   BOOLEAN     NOT NULL DEFAULT true,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_branches_org_branch_code UNIQUE (org_id, branch_code)
);

CREATE INDEX IF NOT EXISTS idx_branches_org_id    ON branches(org_id);
CREATE INDEX IF NOT EXISTS idx_branches_is_active ON branches(org_id, is_active);
