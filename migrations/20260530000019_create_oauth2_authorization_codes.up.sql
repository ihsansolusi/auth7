CREATE TABLE IF NOT EXISTS oauth2_authorization_codes (
    id                    UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    code                  VARCHAR(64) NOT NULL,
    client_id             VARCHAR(128) NOT NULL,
    redirect_uri          VARCHAR(512) NOT NULL DEFAULT '',
    scope                 VARCHAR(500) NOT NULL DEFAULT '',
    user_id               UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    org_id                UUID        NOT NULL REFERENCES organizations(id),
    code_challenge        VARCHAR(128) NOT NULL DEFAULT '',
    code_challenge_method VARCHAR(8)   NOT NULL DEFAULT 'S256',
    roles                 JSONB        NOT NULL DEFAULT '[]',
    branch_id             UUID        REFERENCES branches(id) ON DELETE SET NULL,
    branch_code           VARCHAR(20)  NOT NULL DEFAULT '',
    username              VARCHAR(100) NOT NULL DEFAULT '',
    email                 VARCHAR(255) NOT NULL DEFAULT '',
    expires_at            TIMESTAMPTZ  NOT NULL,
    code_used             BOOLEAN      NOT NULL DEFAULT false,
    created_at            TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_oauth2_authorization_codes_code UNIQUE (code)
);

CREATE INDEX IF NOT EXISTS idx_oauth2_auth_codes_client_id ON oauth2_authorization_codes(client_id);
CREATE INDEX IF NOT EXISTS idx_oauth2_auth_codes_user_id   ON oauth2_authorization_codes(user_id);
