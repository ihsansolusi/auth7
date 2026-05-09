-- Migration: Recreate mfa_configs if dropped by buggy migration 14
-- Safe to run even if table already exists (uses IF NOT EXISTS)

CREATE TABLE IF NOT EXISTS mfa_configs (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                  UUID UNIQUE NOT NULL REFERENCES public.users(id) ON DELETE CASCADE,
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

CREATE INDEX IF NOT EXISTS idx_mfa_configs_user_id ON mfa_configs(user_id);

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name = 'email_otp_codes' AND column_name = 'code_hash') THEN
        ALTER TABLE email_otp_codes ADD COLUMN code_hash VARCHAR(255);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_email_otp_codes_user_id ON email_otp_codes(user_id);
CREATE INDEX IF NOT EXISTS idx_email_otp_codes_expires ON email_otp_codes(expires_at);
