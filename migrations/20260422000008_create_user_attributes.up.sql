-- Migration: Create user_attributes table
-- Up

CREATE TABLE user_attributes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id),
    key         VARCHAR(100) NOT NULL,
    value       TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, key)
);

CREATE INDEX idx_user_attrs_user_id ON user_attributes(user_id);

-- Down
DROP TABLE IF EXISTS user_attributes;
