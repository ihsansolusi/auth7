CREATE TABLE IF NOT EXISTS confidential_groups (
    id                               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id                           UUID NOT NULL,
    cf_code                          VARCHAR(10) NOT NULL DEFAULT '',
    description                      VARCHAR(100),
    updated_at                       TIMESTAMPTZ NOT NULL,
    CONSTRAINT fk_confidential_groups_org_id FOREIGN KEY (org_id) REFERENCES organizations(id),
    CONSTRAINT uq_confidential_groups_org_id_cf_code UNIQUE (org_id, cf_code)
);

CREATE INDEX IF NOT EXISTS idx_confidential_groups_org_id ON confidential_groups(org_id);