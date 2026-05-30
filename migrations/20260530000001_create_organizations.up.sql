CREATE TABLE IF NOT EXISTS organizations (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code       VARCHAR(20)  NOT NULL,
    name       VARCHAR(255) NOT NULL,
    domain     VARCHAR(255),
    status     VARCHAR(50)  NOT NULL DEFAULT 'active',
    settings   JSONB        NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    CONSTRAINT uq_organizations_code UNIQUE (code)
);

CREATE INDEX IF NOT EXISTS idx_organizations_code   ON organizations(code);
CREATE INDEX IF NOT EXISTS idx_organizations_status ON organizations(status) WHERE deleted_at IS NULL;
