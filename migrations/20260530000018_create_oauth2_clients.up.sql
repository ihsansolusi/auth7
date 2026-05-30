CREATE TABLE IF NOT EXISTS oauth2_clients (
    id                          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id                   VARCHAR(128) NOT NULL,
    org_id                      UUID         NOT NULL REFERENCES organizations(id),
    name                        VARCHAR(256) NOT NULL,
    description                 VARCHAR(500) NOT NULL DEFAULT '',
    client_type                 VARCHAR(32)  NOT NULL DEFAULT 'confidential',
    token_endpoint_auth_method  VARCHAR(32)  NOT NULL DEFAULT 'client_secret_basic',
    allowed_scopes              JSONB        NOT NULL DEFAULT '[]',
    allowed_redirect_uris       JSONB        NOT NULL DEFAULT '[]',
    allowed_origins             JSONB        NOT NULL DEFAULT '[]',
    client_secret_hash          VARCHAR(256) NOT NULL DEFAULT '',
    public_key_jwk              VARCHAR(2000) NOT NULL DEFAULT '',
    app_url                     VARCHAR(512) NOT NULL DEFAULT '',
    icon_name                   VARCHAR(64)  NOT NULL DEFAULT '',
    icon_color                  VARCHAR(32)  NOT NULL DEFAULT '',
    token_expiration            INTEGER      NOT NULL DEFAULT 900,
    refresh_token_expiration    INTEGER      NOT NULL DEFAULT 28800,
    allow_multiple_tokens       BOOLEAN      NOT NULL DEFAULT false,
    skip_consent_screen         BOOLEAN      NOT NULL DEFAULT true,
    is_active                   BOOLEAN      NOT NULL DEFAULT true,
    created_at                  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_oauth2_clients_client_id UNIQUE (client_id)
);

CREATE INDEX IF NOT EXISTS idx_oauth2_clients_org_id    ON oauth2_clients(org_id);
CREATE INDEX IF NOT EXISTS idx_oauth2_clients_is_active ON oauth2_clients(is_active);
