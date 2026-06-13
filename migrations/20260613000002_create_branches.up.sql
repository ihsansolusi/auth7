CREATE TABLE IF NOT EXISTS branches (
    id                               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id                           UUID NOT NULL,
    branch_code                      VARCHAR(20) NOT NULL DEFAULT '',
    name                             VARCHAR(255) NOT NULL DEFAULT '',
    is_active                        BOOLEAN NOT NULL DEFAULT false,
    updated_at                       TIMESTAMPTZ NOT NULL,
    CONSTRAINT fk_branches_org_id FOREIGN KEY (org_id) REFERENCES organizations(id),
    CONSTRAINT uq_branches_org_id_branch_code UNIQUE (org_id, branch_code)
);

CREATE INDEX IF NOT EXISTS idx_branches_org_id ON branches(org_id);