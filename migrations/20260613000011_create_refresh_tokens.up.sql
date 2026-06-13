CREATE TABLE IF NOT EXISTS refresh_tokens (
    id                               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    jti                              UUID NOT NULL,
    token_hash                       VARCHAR(255) NOT NULL DEFAULT '',
    family_id                        UUID NOT NULL,
    session_id                       UUID NOT NULL,
    user_id                          UUID NOT NULL,
    org_id                           UUID NOT NULL,
    client_id                        VARCHAR(255) NOT NULL DEFAULT '',
    scopes                           JSONB NOT NULL DEFAULT '{}',
    created_at                       TIMESTAMPTZ NOT NULL,
    expires_at                       TIMESTAMPTZ NOT NULL,
    used_at                          TIMESTAMPTZ,
    revoked_at                       TIMESTAMPTZ,
    replaced_by                      UUID,
    CONSTRAINT uq_refresh_tokens_jti UNIQUE (jti),
    CONSTRAINT uq_refresh_tokens_token_hash UNIQUE (token_hash),
    CONSTRAINT fk_refresh_tokens_user_id FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT fk_refresh_tokens_org_id FOREIGN KEY (org_id) REFERENCES organizations(id)
);

CREATE INDEX IF NOT EXISTS idx_refresh_tokens_family_id ON refresh_tokens(family_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_session_id ON refresh_tokens(session_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_org_id ON refresh_tokens(org_id);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_expires_at
    ON refresh_tokens(expires_at DESC);
CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user_id
    ON refresh_tokens(user_id)
    WHERE revoked_at IS NULL;