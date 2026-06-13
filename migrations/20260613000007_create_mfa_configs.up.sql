CREATE TABLE IF NOT EXISTS mfa_configs (
    id                               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                          UUID NOT NULL,
    totp_secret_encrypted            VARCHAR(500) NOT NULL DEFAULT '',
    totp_secret_iv                   VARCHAR(100) NOT NULL DEFAULT '',
    is_totp_enabled                  BOOLEAN NOT NULL DEFAULT false,
    is_email_otp_enabled             BOOLEAN NOT NULL DEFAULT false,
    is_backup_codes_enabled          BOOLEAN NOT NULL DEFAULT false,
    backup_codes_hash                JSONB NOT NULL DEFAULT '{}',
    mfa_enabled_at                   TIMESTAMPTZ,
    created_at                       TIMESTAMPTZ NOT NULL,
    updated_at                       TIMESTAMPTZ,
    CONSTRAINT uq_mfa_configs_user_id UNIQUE (user_id),
    CONSTRAINT fk_mfa_configs_user_id FOREIGN KEY (user_id) REFERENCES users(id)
);
