CREATE TABLE IF NOT EXISTS user_attributes (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    key        VARCHAR(100) NOT NULL,
    value      VARCHAR(500) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_user_attributes_user_key UNIQUE (user_id, key)
);

CREATE INDEX IF NOT EXISTS idx_user_attributes_user_id ON user_attributes(user_id);
