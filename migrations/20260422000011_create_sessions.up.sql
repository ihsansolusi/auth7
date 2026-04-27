-- Migration: Create sessions table (for audit/history)
-- Up

CREATE TABLE sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL,
    org_id          UUID NOT NULL,
    client_id       VARCHAR(255),
    ip_address      INET,
    user_agent      TEXT,
    device_info     JSONB,
    scopes          TEXT[],
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL,
    revoked_at      TIMESTAMPTZ,
    revoked_by     UUID,
    revoke_reason   TEXT
);

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_org_id ON sessions(org_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);

-- Down
DROP TABLE IF EXISTS sessions;
