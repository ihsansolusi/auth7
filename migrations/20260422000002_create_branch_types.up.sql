-- Migration: Create branch_types table
-- Up

CREATE TABLE branch_types (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL REFERENCES organizations(id),
    code            VARCHAR(50) NOT NULL,
    label           VARCHAR(255) NOT NULL,
    short_code      VARCHAR(10) NOT NULL,
    level           INTEGER NOT NULL,
    is_operational  BOOLEAN NOT NULL DEFAULT true,
    can_have_children BOOLEAN NOT NULL DEFAULT true,
    sort_order      INTEGER NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (org_id, code)
);

CREATE INDEX idx_branch_types_org ON branch_types(org_id);
CREATE INDEX idx_branch_types_level ON branch_types(org_id, level);

COMMENT ON TABLE branch_types IS 'Configurable branch type per organization';

-- Down
DROP TABLE IF EXISTS branch_types;
