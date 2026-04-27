-- Migration: Create refresh_tokens table
-- Up

CREATE TABLE refresh_tokens (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    jti             UUID NOT NULL UNIQUE,
    token_hash      TEXT NOT NULL,
    family_id       UUID NOT NULL,
    user_id         UUID NOT NULL,
    client_id       VARCHAR(255) NOT NULL,
    session_id      UUID NOT NULL,
    org_id          UUID NOT NULL,
    scopes          TEXT[],
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL,
    used_at         TIMESTAMPTZ,
    revoked_at      TIMESTAMPTZ,
    replaced_by     UUID
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_family ON refresh_tokens(family_id);
CREATE INDEX idx_refresh_tokens_expires ON refresh_tokens(expires_at);
CREATE INDEX idx_refresh_tokens_session ON refresh_tokens(session_id);

-- Down
DROP TABLE IF EXISTS refresh_tokens;
