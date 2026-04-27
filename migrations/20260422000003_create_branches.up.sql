-- Migration: Create branches table
-- Up

CREATE TABLE branches (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL REFERENCES organizations(id),
    branch_type_id  UUID NOT NULL REFERENCES branch_types(id),
    code            VARCHAR(20) NOT NULL,
    name            VARCHAR(255) NOT NULL,
    status          VARCHAR(50) NOT NULL DEFAULT 'active',
    address         TEXT,
    phone           VARCHAR(50),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,
    UNIQUE (org_id, code)
);

CREATE INDEX idx_branches_org_id ON branches(org_id);
CREATE INDEX idx_branches_type ON branches(branch_type_id);

-- Down
DROP TABLE IF EXISTS branches;
