CREATE TABLE IF NOT EXISTS roles (
    id                               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id                           UUID NOT NULL,
    code                             VARCHAR(50) NOT NULL DEFAULT '',
    name                             VARCHAR(100) NOT NULL DEFAULT '',
    description                      VARCHAR(500) NOT NULL DEFAULT '',
    is_default                       BOOLEAN NOT NULL DEFAULT false,
    created_at                       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by                       VARCHAR(100) NOT NULL DEFAULT 'system',
    updated_at                       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by                       VARCHAR(100) NOT NULL DEFAULT 'system',
    deleted_at                       TIMESTAMPTZ,
    deleted_by                       VARCHAR(100),
    CONSTRAINT fk_roles_org_id FOREIGN KEY (org_id) REFERENCES organizations(id),
    CONSTRAINT uq_roles_org_id_code UNIQUE (org_id, code)
);

CREATE INDEX IF NOT EXISTS idx_roles_org_id ON roles(org_id);
CREATE INDEX IF NOT EXISTS idx_roles_active ON roles(deleted_at) WHERE deleted_at IS NULL;