CREATE TABLE IF NOT EXISTS user_credentials (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    credential_type VARCHAR(50) NOT NULL,
    secret_hash     VARCHAR(255) NOT NULL,
    version         INTEGER     NOT NULL DEFAULT 1,
    is_current      BOOLEAN     NOT NULL DEFAULT true,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_user_credentials_user_id   ON user_credentials(user_id);
CREATE INDEX IF NOT EXISTS idx_user_credentials_current   ON user_credentials(user_id, credential_type) WHERE is_current = true;
