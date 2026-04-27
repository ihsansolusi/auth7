-- Migration: Create email_otp_codes table
-- Up

CREATE TABLE email_otp_codes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id),
    code        VARCHAR(6) NOT NULL,
    purpose     VARCHAR(50) NOT NULL,
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ,
    attempts    INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_email_otp_user_id ON email_otp_codes(user_id);
CREATE INDEX idx_email_otp_expires ON email_otp_codes(expires_at);

-- Down
DROP TABLE IF EXISTS email_otp_codes;
