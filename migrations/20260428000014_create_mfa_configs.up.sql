-- Migration: Create MFA tables (mfa_configs and update email_otp_codes)
-- Up

CREATE TABLE mfa_configs (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                  UUID UNIQUE NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    totp_secret_encrypted    BYTEA,
    totp_secret_iv          BYTEA,
    is_totp_enabled          BOOLEAN NOT NULL DEFAULT FALSE,
    is_email_otp_enabled     BOOLEAN NOT NULL DEFAULT FALSE,
    is_backup_codes_enabled  BOOLEAN NOT NULL DEFAULT FALSE,
    backup_codes_hash        TEXT[],
    mfa_enabled_at           TIMESTAMPTZ,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_mfa_configs_user_id ON mfa_configs(user_id);

-- Add columns to email_otp_codes if they don't exist (for existing installations)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'email_otp_codes' AND column_name = 'code_hash') THEN
        ALTER TABLE email_otp_codes ADD COLUMN code_hash VARCHAR(255);
    END IF;
END $$;

CREATE INDEX idx_email_otp_codes_user_id ON email_otp_codes(user_id);
CREATE INDEX idx_email_otp_codes_expires ON email_otp_codes(expires_at);

-- Down
DROP TABLE IF EXISTS mfa_configs;
ALTER TABLE email_otp_codes DROP COLUMN IF EXISTS code_hash;