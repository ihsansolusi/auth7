CREATE TABLE IF NOT EXISTS user_credential_history (
    id                               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                          UUID NOT NULL,
    credential_type                  VARCHAR(50) NOT NULL DEFAULT '',
    secret_hash                      VARCHAR(255) NOT NULL DEFAULT '',
    retired_at                       TIMESTAMPTZ NOT NULL,
    CONSTRAINT fk_user_credential_history_user_id FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_user_credential_history_user_id ON user_credential_history(user_id);