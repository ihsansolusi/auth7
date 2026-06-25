DO $$ BEGIN
    CREATE TYPE oauth2_clients_client_type_enum AS ENUM ('web', 'spa', 'native', 'machine');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE oauth2_clients_token_endpoint_auth_method_enum AS ENUM ('none', 'client_secret_basic', 'client_secret_post', 'private_key_jwt');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS oauth2_clients (
    id                               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id                           UUID NOT NULL,
    client_id                        VARCHAR(128) NOT NULL DEFAULT '',
    name                             VARCHAR(256) NOT NULL DEFAULT '',
    description                      VARCHAR(500) NOT NULL DEFAULT '',
    client_type                      oauth2_clients_client_type_enum NOT NULL DEFAULT 'web',
    token_endpoint_auth_method       oauth2_clients_token_endpoint_auth_method_enum NOT NULL DEFAULT 'none',
    app_url                          VARCHAR(512) NOT NULL DEFAULT '',
    icon_name                        VARCHAR(64) NOT NULL DEFAULT '',
    icon_color                       VARCHAR(32) NOT NULL DEFAULT '',
    is_active                        BOOLEAN NOT NULL DEFAULT false,
    client_secret_hash               VARCHAR(256) NOT NULL DEFAULT '',
    public_key_jwk                   VARCHAR(2000) NOT NULL DEFAULT '',
    allowed_redirect_uris            JSONB NOT NULL DEFAULT '{}',
    allowed_origins                  JSONB NOT NULL DEFAULT '{}',
    allowed_scopes                   JSONB NOT NULL DEFAULT '{}',
    token_expiration                 INTEGER NOT NULL DEFAULT 0,
    refresh_token_expiration         INTEGER NOT NULL DEFAULT 0,
    allow_multiple_tokens            BOOLEAN NOT NULL DEFAULT false,
    skip_consent_screen              BOOLEAN NOT NULL DEFAULT false,
    created_at                       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by                       VARCHAR(100) NOT NULL DEFAULT 'system',
    updated_at                       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by                       VARCHAR(100) NOT NULL DEFAULT 'system',
    deleted_at                       TIMESTAMPTZ,
    deleted_by                       VARCHAR(100),
    CONSTRAINT uq_oauth2_clients_client_id UNIQUE (client_id),
    CONSTRAINT fk_oauth2_clients_org_id FOREIGN KEY (org_id) REFERENCES organizations(id)
);

CREATE INDEX IF NOT EXISTS idx_oauth2_clients_org_id ON oauth2_clients(org_id);
CREATE INDEX IF NOT EXISTS idx_oauth2_clients_active ON oauth2_clients(deleted_at) WHERE deleted_at IS NULL;