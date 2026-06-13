CREATE TABLE IF NOT EXISTS permissions (
    id                               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code                             VARCHAR(100) NOT NULL DEFAULT '',
    name                             VARCHAR(100) NOT NULL DEFAULT '',
    description                      VARCHAR(500) NOT NULL DEFAULT '',
    resource_type                    VARCHAR(50) NOT NULL DEFAULT '',
    created_at                       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by                       VARCHAR(100) NOT NULL DEFAULT 'system',
    updated_at                       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by                       VARCHAR(100) NOT NULL DEFAULT 'system',
    deleted_at                       TIMESTAMPTZ,
    deleted_by                       VARCHAR(100),
    CONSTRAINT uq_permissions_code UNIQUE (code)
);

CREATE INDEX IF NOT EXISTS idx_permissions_active ON permissions(deleted_at) WHERE deleted_at IS NULL;