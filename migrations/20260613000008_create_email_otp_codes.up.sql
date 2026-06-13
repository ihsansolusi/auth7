DO $$ BEGIN
    CREATE TYPE email_otp_codes_purpose_enum AS ENUM ('login', 'reset', 'verify');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS email_otp_codes (
    id                               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                          UUID NOT NULL,
    code_hash                        VARCHAR(255) NOT NULL DEFAULT '',
    purpose                          email_otp_codes_purpose_enum NOT NULL DEFAULT 'login',
    expires_at                       TIMESTAMPTZ NOT NULL,
    used_at                          TIMESTAMPTZ,
    attempts                         INTEGER NOT NULL DEFAULT 0,
    created_at                       TIMESTAMPTZ NOT NULL,
    CONSTRAINT fk_email_otp_codes_user_id FOREIGN KEY (user_id) REFERENCES users(id)
);

CREATE INDEX IF NOT EXISTS idx_email_otp_codes_user_id ON email_otp_codes(user_id);
CREATE INDEX IF NOT EXISTS idx_email_otp_codes_user_id_purpose
    ON email_otp_codes(user_id, purpose)
    WHERE used_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_email_otp_codes_expires_at
    ON email_otp_codes(expires_at DESC);