CREATE TABLE IF NOT EXISTS sessions (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id       UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    org_id        UUID        NOT NULL,
    branch_code   VARCHAR(20) NOT NULL DEFAULT '',
    username      VARCHAR(100) NOT NULL DEFAULT '',
    client_id     VARCHAR(255) NOT NULL DEFAULT '',
    ip_address    VARCHAR(50)  NOT NULL DEFAULT '',
    user_agent    VARCHAR(500) NOT NULL DEFAULT '',
    device_info   JSONB        NOT NULL DEFAULT '{}',
    scopes        JSONB        NOT NULL DEFAULT '[]',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    last_used_at  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    expires_at    TIMESTAMPTZ  NOT NULL,
    revoked_at    TIMESTAMPTZ,
    revoked_by    VARCHAR(36)  NOT NULL DEFAULT '',
    revoke_reason VARCHAR(200) NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id    ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_org_id     ON sessions(org_id);
CREATE INDEX IF NOT EXISTS idx_sessions_active     ON sessions(user_id) WHERE revoked_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_sessions_expires_at ON sessions(expires_at);
