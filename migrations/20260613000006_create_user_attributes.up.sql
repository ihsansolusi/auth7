CREATE TABLE IF NOT EXISTS user_attributes (
    id                               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                          UUID NOT NULL,
    key                              VARCHAR(100) NOT NULL DEFAULT '',
    value                            VARCHAR(500) NOT NULL DEFAULT '',
    created_at                       TIMESTAMPTZ NOT NULL,
    updated_at                       TIMESTAMPTZ,
    CONSTRAINT fk_user_attributes_user_id FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT uq_user_attributes_user_id_key UNIQUE (user_id, key)
);

CREATE INDEX IF NOT EXISTS idx_user_attributes_user_id ON user_attributes(user_id);