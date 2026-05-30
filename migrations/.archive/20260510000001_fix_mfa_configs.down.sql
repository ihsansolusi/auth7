-- Migration: Fix mfa_configs rollback
DROP TABLE IF EXISTS mfa_configs;
ALTER TABLE email_otp_codes DROP COLUMN IF EXISTS code_hash;
