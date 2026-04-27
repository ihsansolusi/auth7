-- Migration: Create organizations table
-- Up

CREATE TABLE organizations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code            VARCHAR(20) NOT NULL UNIQUE,
    name            VARCHAR(255) NOT NULL,
    domain          VARCHAR(255),
    status          VARCHAR(50) NOT NULL DEFAULT 'active',
    settings        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

COMMENT ON TABLE organizations IS 'Bank/Tenant organization';
COMMENT ON COLUMN organizations.settings IS 'Org-level config: session_policy, mfa_policy, password_policy, branding';

-- Down
DROP TABLE IF EXISTS organizations;
