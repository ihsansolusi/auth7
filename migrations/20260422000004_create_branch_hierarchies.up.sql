-- Migration: Create branch_hierarchies table
-- Up

CREATE TABLE branch_hierarchies (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL REFERENCES organizations(id),
    parent_id       UUID REFERENCES branches(id),
    child_id        UUID NOT NULL REFERENCES branches(id),
    path            VARCHAR(500) NOT NULL,
    depth           INTEGER NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (org_id, parent_id, child_id),
    UNIQUE (org_id, child_id)
);

CREATE INDEX idx_branch_hierarchies_parent ON branch_hierarchies(org_id, parent_id);
CREATE INDEX idx_branch_hierarchies_child ON branch_hierarchies(org_id, child_id);
CREATE INDEX idx_branch_hierarchies_path ON branch_hierarchies(org_id, path);

-- Down
DROP TABLE IF EXISTS branch_hierarchies;
