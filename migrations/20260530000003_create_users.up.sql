CREATE TABLE IF NOT EXISTS users (
    id                       UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id                   UUID        NOT NULL REFERENCES organizations(id),
    username                 VARCHAR(100) NOT NULL,
    email                    VARCHAR(255) NOT NULL,
    full_name                VARCHAR(255) NOT NULL,
    status                   VARCHAR(50)  NOT NULL DEFAULT 'created',
    email_verified           BOOLEAN      NOT NULL DEFAULT false,
    mfa_enabled              BOOLEAN      NOT NULL DEFAULT false,
    mfa_method               VARCHAR(20)  NOT NULL DEFAULT '',
    mfa_reset_required       BOOLEAN      NOT NULL DEFAULT false,
    require_password_change  BOOLEAN      NOT NULL DEFAULT false,
    failed_login_attempts    INTEGER      NOT NULL DEFAULT 0,
    locked_until             TIMESTAMPTZ,
    last_login_at            TIMESTAMPTZ,
    last_login_ip            VARCHAR(50)  NOT NULL DEFAULT '',
    password_changed_at      TIMESTAMPTZ,
    preferred_locale         VARCHAR(8)   NOT NULL DEFAULT 'id',
    created_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at               TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    deleted_at               TIMESTAMPTZ,
    created_by               VARCHAR(36)  NOT NULL DEFAULT 'system',
    updated_by               VARCHAR(36)  NOT NULL DEFAULT 'system',
    CONSTRAINT uq_users_org_email UNIQUE (org_id, email)
);

-- Global partial unique: one username across all orgs (login identity)
CREATE UNIQUE INDEX IF NOT EXISTS idx_users_username_global
    ON users(username) WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_users_org_id  ON users(org_id);
CREATE INDEX IF NOT EXISTS idx_users_email   ON users(org_id, email) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_users_status  ON users(org_id, status) WHERE deleted_at IS NULL;
