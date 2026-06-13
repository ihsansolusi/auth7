DO $$ BEGIN
    CREATE TYPE user_credentials_credential_type_enum AS ENUM ('password', 'totp');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS user_credentials (
    id                               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                          UUID NOT NULL,
    credential_type                  user_credentials_credential_type_enum NOT NULL DEFAULT 'password',
    secret_hash                      VARCHAR(255) NOT NULL DEFAULT '',
    created_at                       TIMESTAMPTZ NOT NULL,
    expires_at                       TIMESTAMPTZ,
    CONSTRAINT fk_user_credentials_user_id FOREIGN KEY (user_id) REFERENCES users(id),
    CONSTRAINT uq_user_credentials_user_id_credential_type UNIQUE (user_id, credential_type)
);

CREATE INDEX IF NOT EXISTS idx_user_credentials_user_id ON user_credentials(user_id);