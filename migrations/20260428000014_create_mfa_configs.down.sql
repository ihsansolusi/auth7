-- Migration: Create MFA tables
-- Down

DROP TABLE IF EXISTS mfa_configs;
ALTER TABLE email_otp_codes DROP COLUMN IF EXISTS code_hash;