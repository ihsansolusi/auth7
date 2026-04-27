-- Migration: Create oauth2_clients table
-- Issue: #38

CREATE TABLE oauth2_clients (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    client_id VARCHAR(128) UNIQUE NOT NULL,
    org_id UUID NOT NULL REFERENCES organizations(id),
    name VARCHAR(256) NOT NULL,
    description TEXT,
    client_type VARCHAR(32) NOT NULL DEFAULT 'web',
    token_endpoint_auth_method VARCHAR(32) NOT NULL DEFAULT 'client_secret_basic',
    allowed_scopes TEXT[] NOT NULL DEFAULT '{}',
    allowed_redirect_uris TEXT[] NOT NULL DEFAULT '{}',
    allowed_origins TEXT[] NOT NULL DEFAULT '{}',
    client_secret_hash VARCHAR(256),
    public_key_jwk TEXT,
    token_expiration INTEGER NOT NULL DEFAULT 900,
    refresh_token_expiration INTEGER NOT NULL DEFAULT 28800,
    allow_multiple_tokens BOOLEAN NOT NULL DEFAULT false,
    skip_consent_screen BOOLEAN NOT NULL DEFAULT false,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_oauth2_clients_org_id ON oauth2_clients(org_id);
CREATE INDEX idx_oauth2_clients_client_id ON oauth2_clients(client_id);

CREATE TABLE oauth2_authorization_codes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(64) UNIQUE NOT NULL,
    client_id VARCHAR(128) NOT NULL REFERENCES oauth2_clients(client_id),
    redirect_uri VARCHAR(512) NOT NULL,
    scope TEXT,
    user_id UUID NOT NULL REFERENCES users(id),
    org_id UUID NOT NULL REFERENCES organizations(id),
    code_challenge VARCHAR(128),
    code_challenge_method VARCHAR(8),
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    code_used BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_oauth2_auth_codes_code ON oauth2_authorization_codes(code);
CREATE INDEX idx_oauth2_auth_codes_client_id ON oauth2_authorization_codes(client_id);
CREATE INDEX idx_oauth2_auth_codes_user_id ON oauth2_authorization_codes(user_id);
CREATE INDEX idx_oauth2_auth_codes_expires_at ON oauth2_authorization_codes(expires_at);

COMMENT ON TABLE oauth2_clients IS 'OAuth2 client registry (DCR RFC 7591)';
COMMENT ON TABLE oauth2_authorization_codes IS 'Temporary authorization codes for code flow';