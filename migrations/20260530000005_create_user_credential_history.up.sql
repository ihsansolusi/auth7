CREATE TABLE IF NOT EXISTS user_credential_history (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    secret_hash VARCHAR(255) NOT NULL,
    retired_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_user_credential_history_user_id ON user_credential_history(user_id);
