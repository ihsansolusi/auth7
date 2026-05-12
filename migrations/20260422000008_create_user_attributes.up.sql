-- Migration: Create user_attributes table
-- Up

CREATE TABLE IF NOT EXISTS user_attributes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES public.users(id),
    key         VARCHAR(100) NOT NULL,
    value       TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, key)
);

CREATE INDEX IF NOT EXISTS idx_user_attrs_user_id ON user_attributes(user_id);

