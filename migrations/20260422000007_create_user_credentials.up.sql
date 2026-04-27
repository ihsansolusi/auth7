-- Migration: Create user_credentials and user_credential_history tables
-- Up

CREATE TABLE user_credentials (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    credential_type VARCHAR(50) NOT NULL DEFAULT 'password',
    secret_hash     TEXT NOT NULL,
    version         INTEGER NOT NULL DEFAULT 1,
    is_current      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ
);

CREATE TABLE user_credential_history (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    secret_hash     TEXT NOT NULL,
    retired_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_credentials_user_id ON user_credentials(user_id);
CREATE INDEX idx_user_cred_history_user_id ON user_credential_history(user_id);

-- Down
DROP TABLE IF EXISTS user_credentials;
DROP TABLE IF EXISTS user_credential_history;
