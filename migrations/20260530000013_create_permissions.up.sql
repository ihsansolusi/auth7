CREATE TABLE IF NOT EXISTS permissions (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    code          VARCHAR(100) NOT NULL,
    name          VARCHAR(100) NOT NULL,
    description   VARCHAR(500) NOT NULL DEFAULT '',
    resource_type VARCHAR(50)  NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_permissions_code UNIQUE (code)
);

CREATE INDEX IF NOT EXISTS idx_permissions_resource_type ON permissions(resource_type);
