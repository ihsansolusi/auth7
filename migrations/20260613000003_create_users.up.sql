DO $$ BEGIN
    CREATE TYPE users_preferred_locale_enum AS ENUM ('id', 'en');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE users_status_enum AS ENUM ('created', 'pending_verification', 'active', 'inactive', 'locked', 'suspended', 'deleted');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$ BEGIN
    CREATE TYPE users_mfa_method_enum AS ENUM ('', 'totp', 'email_otp');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS users (
    id                               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id                           UUID NOT NULL,
    username                         VARCHAR(100) NOT NULL DEFAULT '',
    email                            VARCHAR(255) NOT NULL DEFAULT '',
    full_name                        VARCHAR(255) NOT NULL DEFAULT '',
    status                           users_status_enum NOT NULL DEFAULT 'created',
    email_verified                   BOOLEAN NOT NULL DEFAULT false,
    mfa_enabled                      BOOLEAN NOT NULL DEFAULT false,
    mfa_method                       users_mfa_method_enum NOT NULL DEFAULT '',
    mfa_reset_required               BOOLEAN NOT NULL DEFAULT false,
    require_password_change          BOOLEAN NOT NULL DEFAULT false,
    failed_login_attempts            INTEGER NOT NULL DEFAULT 0,
    locked_until                     TIMESTAMPTZ,
    last_login_at                    TIMESTAMPTZ,
    last_login_ip                    VARCHAR(50),
    password_changed_at              TIMESTAMPTZ,
    preferred_locale                 users_preferred_locale_enum NOT NULL DEFAULT 'id',
    created_at                       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by                       VARCHAR(100) NOT NULL DEFAULT 'system',
    updated_at                       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_by                       VARCHAR(100) NOT NULL DEFAULT 'system',
    deleted_at                       TIMESTAMPTZ,
    deleted_by                       VARCHAR(100),
    CONSTRAINT uq_users_username UNIQUE (username),
    CONSTRAINT uq_users_email UNIQUE (email),
    CONSTRAINT fk_users_org_id FOREIGN KEY (org_id) REFERENCES organizations(id)
);

CREATE INDEX IF NOT EXISTS idx_users_org_id ON users(org_id);
CREATE INDEX IF NOT EXISTS idx_users_active ON users(deleted_at) WHERE deleted_at IS NULL;