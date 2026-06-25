CREATE TABLE IF NOT EXISTS confidential_accesses (
    id                               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id                           UUID NOT NULL,
    role_id                          UUID NOT NULL,
    confidential_group_id            UUID NOT NULL,
    granted_by                       VARCHAR(36),
    granted_at                       TIMESTAMPTZ NOT NULL,
    CONSTRAINT fk_confidential_accesses_org_id FOREIGN KEY (org_id) REFERENCES organizations(id),
    CONSTRAINT fk_confidential_accesses_role_id FOREIGN KEY (role_id) REFERENCES roles(id),
    CONSTRAINT fk_confidential_accesses_confidential_group_id FOREIGN KEY (confidential_group_id) REFERENCES confidential_groups(id),
    CONSTRAINT uq_confidential_accesses_org_id_role_id_confidential_group_id UNIQUE (org_id, role_id, confidential_group_id)
);

CREATE INDEX IF NOT EXISTS idx_confidential_accesses_org_id ON confidential_accesses(org_id);
CREATE INDEX IF NOT EXISTS idx_confidential_accesses_role_id ON confidential_accesses(role_id);
CREATE INDEX IF NOT EXISTS idx_confidential_accesses_confidential_group_id ON confidential_accesses(confidential_group_id);
CREATE INDEX IF NOT EXISTS idx_confidential_accesses_role_id
    ON confidential_accesses(role_id);