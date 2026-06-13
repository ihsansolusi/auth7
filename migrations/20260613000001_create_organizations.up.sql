DO $$ BEGIN
    CREATE TYPE organizations_status_enum AS ENUM ('active', 'suspended', 'inactive');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS organizations (
    id                               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code                             VARCHAR(20) NOT NULL DEFAULT '',
    name                             VARCHAR(255) NOT NULL DEFAULT '',
    domain                           VARCHAR(255) NOT NULL DEFAULT '',
    status                           organizations_status_enum NOT NULL DEFAULT 'active',
    settings                         JSONB NOT NULL DEFAULT '{}',
    created_at                       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by                       VARCHAR(100) NOT NULL DEFAULT 'system',
    updated_at                       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by                       VARCHAR(100) NOT NULL DEFAULT 'system',
    deleted_at                       TIMESTAMPTZ,
    deleted_by                       VARCHAR(100),
    CONSTRAINT uq_organizations_code UNIQUE (code)
);

CREATE INDEX IF NOT EXISTS idx_organizations_active ON organizations(deleted_at) WHERE deleted_at IS NULL;