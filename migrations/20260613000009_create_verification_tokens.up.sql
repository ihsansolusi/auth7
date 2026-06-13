DO $$ BEGIN
    CREATE TYPE verification_tokens_token_type_enum AS ENUM ('email_verify', 'password_reset');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS verification_tokens (
    id                               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                          UUID NOT NULL,
    token_hash                       VARCHAR(255) NOT NULL DEFAULT '',
    token_type                       verification_tokens_token_type_enum NOT NULL DEFAULT 'email_verify',
    expires_at                       TIMESTAMPTZ NOT NULL,
    used_at                          TIMESTAMPTZ,
    created_at                       TIMESTAMPTZ NOT NULL,
    CONSTRAINT uq_verification_tokens_token_hash UNIQUE (token_hash),
    CONSTRAINT fk_verification_tokens_user_id FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_verification_tokens_user_id ON verification_tokens(user_id);
CREATE INDEX IF NOT EXISTS idx_verification_tokens_expires_at
    ON verification_tokens(expires_at DESC);