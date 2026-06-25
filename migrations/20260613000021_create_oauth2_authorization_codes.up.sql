DO $$ BEGIN
    CREATE TYPE oauth2_authorization_codes_code_challenge_method_enum AS ENUM ('S256', 'plain');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS oauth2_authorization_codes (
    id                               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code                             VARCHAR(64) NOT NULL DEFAULT '',
    client_id                        VARCHAR(128) NOT NULL DEFAULT '',
    redirect_uri                     VARCHAR(512) NOT NULL DEFAULT '',
    scope                            VARCHAR(500) NOT NULL DEFAULT '',
    user_id                          UUID NOT NULL,
    org_id                           UUID NOT NULL,
    code_challenge                   VARCHAR(128) NOT NULL DEFAULT '',
    code_challenge_method            oauth2_authorization_codes_code_challenge_method_enum NOT NULL DEFAULT 'S256',
    roles                            JSONB NOT NULL DEFAULT '{}',
    branch_id                        UUID,
    branch_code                      VARCHAR(20) NOT NULL DEFAULT '',
    username                         VARCHAR(100) NOT NULL DEFAULT '',
    email                            VARCHAR(255) NOT NULL DEFAULT '',
    expires_at                       TIMESTAMPTZ NOT NULL,
    code_used                        BOOLEAN NOT NULL DEFAULT false,
    created_at                       TIMESTAMPTZ NOT NULL,
    CONSTRAINT uq_oauth2_authorization_codes_code UNIQUE (code),
    CONSTRAINT fk_oauth2_authorization_codes_user_id FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT fk_oauth2_authorization_codes_org_id FOREIGN KEY (org_id) REFERENCES organizations(id),
    CONSTRAINT fk_oauth2_authorization_codes_branch_id FOREIGN KEY (branch_id) REFERENCES branches(id)
);

CREATE INDEX IF NOT EXISTS idx_oauth2_authorization_codes_user_id ON oauth2_authorization_codes(user_id);
CREATE INDEX IF NOT EXISTS idx_oauth2_authorization_codes_org_id ON oauth2_authorization_codes(org_id);
CREATE INDEX IF NOT EXISTS idx_oauth2_authorization_codes_expires_at
    ON oauth2_authorization_codes(expires_at DESC);
CREATE INDEX IF NOT EXISTS idx_oauth2_authorization_codes_client_id
    ON oauth2_authorization_codes(client_id)
    WHERE code_used = FALSE;