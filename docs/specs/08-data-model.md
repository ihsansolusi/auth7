# Auth7 — Spec 08: Data Model (PostgreSQL Schema)

> **Versi**: 1.0-draft | **Tanggal**: 2026-04-22 | **Fase**: Brainstorming

---

## 1. Schema Design Principles

- Semua tabel pakai `UUID` sebagai primary key
- Audit fields standard: `created_at`, `updated_at`, `created_by`, `updated_by`
- Soft delete: `deleted_at` nullable (bukan hard delete)
- Multi-tenant: semua tabel memiliki `org_id`
- No cascade delete di application level (hanya FK constraint)
- Naming: `snake_case`, nama tabel plural

---

## 2. Core Tables

### 2.1 `organizations` — Bank/Tenant

```sql
CREATE TABLE organizations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code            VARCHAR(20) NOT NULL UNIQUE,   -- "BJBS", "BSI"
    name            VARCHAR(255) NOT NULL,
    domain          VARCHAR(255),
    status          VARCHAR(50) NOT NULL DEFAULT 'active',
    settings        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ
);

COMMENT ON COLUMN organizations.settings IS 'Org-level config: session_policy, mfa_policy, password_policy, branding';
```

**settings JSONB structure:**
```json
{
  "password_policy": {
    "min_length": 8,
    "require_uppercase": true,
    "require_number": true,
    "require_symbol": false,
    "max_age_days": 90,
    "history_count": 5
  },
  "session_policy": {
    "max_concurrent": 3,
    "idle_timeout_minutes": 30,
    "absolute_timeout_hours": 8
  },
  "mfa_policy": "optional"  // optional | required | required_for_roles
}
```

### 2.2 `branch_types` — Tipe Kantor (Configurable per Org)

Setiap organization mendefinisikan sendiri jenis-jenis kantornya. Tidak di-hardcode — bank berbeda bisa punya struktur berbeda.

```sql
CREATE TABLE branch_types (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL REFERENCES organizations(id),
    code            VARCHAR(50) NOT NULL,          -- 'HEAD_OFFICE', 'REGIONAL', dll
    label           VARCHAR(255) NOT NULL,         -- 'Kantor Pusat', 'Kantor Wilayah', dll
    short_code      VARCHAR(10) NOT NULL,          -- 'KP', 'KW', 'KC', dll
    level           INTEGER NOT NULL,               -- 0 = tertinggi, makin besar = makin rendah
    is_operational  BOOLEAN NOT NULL DEFAULT true,  -- true = melakukan transaksi nasabah
    can_have_children BOOLEAN NOT NULL DEFAULT true, -- true = bisa punya sub-branch
    sort_order      INTEGER NOT NULL DEFAULT 0,    -- urutan tampil di UI
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (org_id, code)
);
CREATE INDEX idx_branch_types_org ON branch_types(org_id);
CREATE INDEX idx_branch_types_level ON branch_types(org_id, level);
```

**Default seed per org (contoh untuk bank dengan struktur 5 level):**

| code | label | short_code | level | is_operational | can_have_children |
|------|-------|-----------|-------|---------------|-------------------|
| `HEAD_OFFICE` | Kantor Pusat | KP | 0 | true | true |
| `REGIONAL` | Kantor Wilayah | KW | 1 | false | true |
| `BRANCH` | Kantor Cabang | KC | 2 | true | true |
| `SUB_BRANCH` | Kantor Cabang Pembantu | KCP | 3 | true | true |
| `CASH_OFFICE` | Kantor Kas | KK | 4 | true | false |

> **Catatan**: Bank lain bisa punya konfigurasi berbeda — misal digital bank hanya punya
> `HEAD_OFFICE` dan `BRANCH` (2 level). `branch_types` membuat sistem fleksibel tanpa code changes.

### 2.3 `branches` — Kantor/Cabang Bank

```sql
CREATE TABLE branches (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL REFERENCES organizations(id),
    branch_type_id  UUID NOT NULL REFERENCES branch_types(id),  -- FK ke branch_types
    code            VARCHAR(20) NOT NULL,
    name            VARCHAR(255) NOT NULL,
    status          VARCHAR(50) NOT NULL DEFAULT 'active',
    address         TEXT,
    phone           VARCHAR(50),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,
    UNIQUE (org_id, code)
);
CREATE INDEX idx_branches_org_id ON branches(org_id);
CREATE INDEX idx_branches_type ON branches(branch_type_id);
```

### 2.4 `branch_hierarchies` — Hierarki Kantor

Hierarki disimpan terpisah dari branch_type, memungkinkan struktur tree yang fleksibel:

```sql
CREATE TABLE branch_hierarchies (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL REFERENCES organizations(id),
    parent_id       UUID REFERENCES branches(id),  -- NULL untuk root (HEAD_OFFICE)
    child_id        UUID NOT NULL REFERENCES branches(id),
    path            VARCHAR(500) NOT NULL,         -- '/{root_id}/.../{parent_id}/{child_id}/'
    depth           INTEGER NOT NULL,               -- 0=root, 1, 2, ...
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (org_id, parent_id, child_id),
    UNIQUE (org_id, child_id)  -- one child has one parent
);

CREATE INDEX idx_branch_hierarchies_parent ON branch_hierarchies(org_id, parent_id);
CREATE INDEX idx_branch_hierarchies_child ON branch_hierarchies(org_id, child_id);
CREATE INDEX idx_branch_hierarchies_path ON branch_hierarchies(org_id, path);
```

**Contoh hierarki (bank dengan 5 level):**
```
HEAD_OFFICE (KP Olshop)
  └── REGIONAL (Kanwil Jawa Barat)
        └── BRANCH (KC Bandung)
              ├── SUB_BRANCH (KCP Dago)
              │     └── CASH_OFFICE (KK Dago)
              └── SUB_BRANCH (KCP setiabudhi)
        └── BRANCH (KC Bekasi)
              └── SUB_BRANCH (KCP Galaxy)
  └── REGIONAL (Kanwil Jawa Timur)
        └── BRANCH (KC Surabaya)
              └── ...
```

**Contoh hierarki (digital bank — hanya 2 level):**
```
HEAD_OFFICE (Head Office)
  └── BRANCH (Branch Jakarta)
  └── BRANCH (Branch Surabaya)
  └── BRANCH (Branch Bandung)
```
```

### 2.5 `user_branch_assignments` — Akses User ke Branch

User bisa punya akses ke beberapa branch dengan role/permission berbeda per branch.
Satu branch ditandai sebagai `is_primary` = default saat login.

```sql
CREATE TABLE user_branch_assignments (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    branch_id       UUID NOT NULL REFERENCES branches(id),
    is_primary      BOOLEAN NOT NULL DEFAULT FALSE,  -- true = default branch saat login
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, branch_id)                       -- 1 user per branch = 1 assignment
);

-- Setiap user wajib punya minimal 1 primary branch
-- is_primary = true hanya boleh 1 per user (enalised at application level)
CREATE INDEX idx_user_branch_user ON user_branch_assignments(user_id);
CREATE INDEX idx_user_branch_branch ON user_branch_assignments(branch_id);
CREATE INDEX idx_user_branch_primary ON user_branch_assignments(user_id) WHERE is_primary = TRUE;
```

**Contoh data:**

| user_id | branch_id | is_primary |
|---------|-----------|-----------|
| john | KC Bandung | true |
| john | KCP Dago | false |
| john | KC Jakarta | false |

> john login → default di KC Bandung (primary).
> john switch → bisa pilih KCP Dago atau KC Jakarta.
> Saat switch, role/permission bisa berbeda per branch (diatur via `user_roles`).

### 2.6 `users` — Identitas User

```sql
CREATE TABLE users (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id                  UUID NOT NULL REFERENCES organizations(id),
    -- branch_id dihapus dari users; lihat user_branch_assignments untuk multi-branch access
    username                VARCHAR(100) NOT NULL,
    email                   VARCHAR(255) NOT NULL,
    full_name               VARCHAR(255) NOT NULL,
    status                  VARCHAR(50) NOT NULL DEFAULT 'pending_verification',
    email_verified          BOOLEAN NOT NULL DEFAULT FALSE,
    mfa_enabled             BOOLEAN NOT NULL DEFAULT FALSE,
    mfa_method              VARCHAR(20),
    mfa_reset_required      BOOLEAN NOT NULL DEFAULT FALSE,
    require_password_change BOOLEAN NOT NULL DEFAULT FALSE,
    failed_login_attempts   INTEGER NOT NULL DEFAULT 0,
    locked_until            TIMESTAMPTZ,
    last_login_at           TIMESTAMPTZ,
    last_login_ip           INET,
    password_changed_at     TIMESTAMPTZ,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at              TIMESTAMPTZ,
    created_by              UUID,
    updated_by              UUID,
    UNIQUE (org_id, username),
    UNIQUE (org_id, email)
);

CREATE INDEX idx_users_org_id ON users(org_id);
CREATE INDEX idx_users_email ON users(org_id, email);
CREATE INDEX idx_users_status ON users(org_id, status);
CREATE INDEX idx_users_deleted ON users(deleted_at) WHERE deleted_at IS NOT NULL;
```

### 2.7 `user_credentials` — Password & Hashes

```sql
CREATE TABLE user_credentials (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    credential_type VARCHAR(50) NOT NULL DEFAULT 'password',  -- 'password', 'api_key'
    secret_hash     TEXT NOT NULL,    -- argon2id hash
    version         INTEGER NOT NULL DEFAULT 1,  -- untuk key rotation
    is_current      BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ      -- untuk password expiry policy
);

-- History untuk prevent password reuse
CREATE TABLE user_credential_history (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    secret_hash     TEXT NOT NULL,
    retired_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_user_credentials_user_id ON user_credentials(user_id);
CREATE INDEX idx_user_cred_history_user_id ON user_credential_history(user_id);
```

### 2.8 `user_attributes` — Extensible User Metadata

```sql
CREATE TABLE user_attributes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id),
    key         VARCHAR(100) NOT NULL,
    value       TEXT NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, key)
);

CREATE INDEX idx_user_attrs_user_id ON user_attributes(user_id);
```

---

## 3. MFA Tables

### 3.1 `mfa_configs` — MFA Settings

```sql
CREATE TABLE mfa_configs (
    user_id         UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    method          VARCHAR(20) NOT NULL,
    totp_secret     VARCHAR(255),     -- encrypted at-rest
    totp_activated  BOOLEAN NOT NULL DEFAULT FALSE,
    backup_codes    JSONB,            -- array of hashed backup codes
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON COLUMN mfa_configs.totp_secret IS 'AES-256-GCM encrypted TOTP secret';
COMMENT ON COLUMN mfa_configs.backup_codes IS 'Array of SHA-256 hashed backup codes';
```

### 3.2 `backup_codes` — MFA Recovery Codes (detailed)

```sql
CREATE TABLE backup_codes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id),
    code_hash   TEXT NOT NULL,   -- SHA-256 hash
    used_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_backup_codes_user_id ON backup_codes(user_id);
```

---

## 4. Session & Token Tables

### 4.1 `sessions` — Persisted Session Metadata

```sql
CREATE TABLE sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id),
    org_id          UUID NOT NULL REFERENCES organizations(id),
    client_id       VARCHAR(255),      -- OAuth2 client (nullable untuk direct login)
    ip_address      INET,
    user_agent      TEXT,
    device_info     JSONB,
    scopes          TEXT[],
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_used_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL,
    revoked_at      TIMESTAMPTZ,
    revoked_by      UUID,
    revoke_reason   TEXT
);
-- Note: actual session data di Redis (fast lookup), tabel ini untuk audit/history

CREATE INDEX idx_sessions_user_id ON sessions(user_id);
CREATE INDEX idx_sessions_org_id ON sessions(org_id);
CREATE INDEX idx_sessions_expires_at ON sessions(expires_at);
```

### 4.2 `refresh_tokens` — OAuth2 Refresh Tokens

```sql
CREATE TABLE refresh_tokens (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    jti             UUID NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    token_hash      TEXT NOT NULL,           -- SHA-256 hash dari opaque token
    family_id       UUID NOT NULL,           -- token family untuk reuse detection
    user_id         UUID NOT NULL REFERENCES users(id),
    client_id       VARCHAR(255) NOT NULL REFERENCES oauth2_clients(id),
    session_id      UUID NOT NULL,
    org_id          UUID NOT NULL REFERENCES organizations(id),
    scopes          TEXT[],
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at      TIMESTAMPTZ NOT NULL,
    used_at         TIMESTAMPTZ,
    revoked         BOOLEAN NOT NULL DEFAULT FALSE,
    revoked_at      TIMESTAMPTZ,
    replaced_by     UUID REFERENCES refresh_tokens(jti)
);

CREATE INDEX idx_refresh_tokens_user_id ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_family ON refresh_tokens(family_id);
CREATE INDEX idx_refresh_tokens_expires ON refresh_tokens(expires_at);
CREATE INDEX idx_refresh_tokens_session ON refresh_tokens(session_id);
```

### 4.3 `token_jwks` — JWT Signing Keys

```sql
CREATE TABLE token_jwks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    kid             VARCHAR(100) NOT NULL UNIQUE,   -- "auth7-2026-04"
    algorithm       VARCHAR(20) NOT NULL DEFAULT 'RS256',
    public_key_pem  TEXT NOT NULL,
    private_key_enc TEXT NOT NULL,    -- AES-256 encrypted private key
    nonce           BYTEA NOT NULL,
    is_active       BOOLEAN NOT NULL DEFAULT TRUE,   -- current signing key
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    rotated_at      TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ     -- serve di JWKS sampai sini
);
```

---

## 5. Authorization Tables

### 5.1 `roles` — Role Definitions

```sql
CREATE TABLE roles (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL REFERENCES organizations(id),
    name            VARCHAR(100) NOT NULL,
    display_name    VARCHAR(255),
    description     TEXT,
    is_system       BOOLEAN NOT NULL DEFAULT FALSE,  -- built-in roles
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at      TIMESTAMPTZ,
    UNIQUE (org_id, name)
);

CREATE INDEX idx_roles_org_id ON roles(org_id);
```

### 5.2 `role_inheritances` — Role Hierarchy

```sql
CREATE TABLE role_inheritances (
    parent_role_id  UUID NOT NULL REFERENCES roles(id),
    child_role_id   UUID NOT NULL REFERENCES roles(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (parent_role_id, child_role_id)
);
```

### 5.3 `permissions` — Permission Definitions

```sql
CREATE TABLE permissions (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id      UUID REFERENCES organizations(id),  -- NULL = global permission
    resource    VARCHAR(100) NOT NULL,
    action      VARCHAR(100) NOT NULL,
    description TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (org_id, resource, action)
);

CREATE INDEX idx_permissions_org_id ON permissions(org_id);
```

### 5.4 `role_permissions` — Role ↔ Permission Mapping

```sql
CREATE TABLE role_permissions (
    role_id         UUID NOT NULL REFERENCES roles(id),
    permission_id   UUID NOT NULL REFERENCES permissions(id),
    effect          VARCHAR(10) NOT NULL DEFAULT 'allow',  -- allow | deny
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (role_id, permission_id)
);
```

### 5.5 `user_roles` — User ↔ Role Assignment (per Branch)

Roles di-assign per user per branch, memungkinkan user punya role berbeda di cabang berbeda.

```sql
CREATE TABLE user_roles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id     UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    branch_id   UUID NOT NULL REFERENCES branches(id) ON DELETE CASCADE,
    org_id      UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    assigned_by UUID REFERENCES users(id),
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, role_id, branch_id)
);

CREATE INDEX idx_user_roles_user ON user_roles(user_id);
CREATE INDEX idx_user_roles_branch ON user_roles(branch_id);
CREATE INDEX idx_user_roles_role ON user_roles(role_id);
```

**Contoh:**

| user_id | role_id | branch_id | Artinya |
|---|---|---|---|
| john | supervisor | KC Bandung | John = supervisor di KC Bandung |
| john | teller | KCP Dago | John = teller di KCP Dago |
| jane | org_admin | KP Pusat | Jane = org admin di KP Pusat |

### 5.6 `permission_field_masks` — Field-Level Permission Masking

```sql
CREATE TABLE permission_field_masks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL REFERENCES organizations(id),
    resource        VARCHAR(100) NOT NULL,     -- 'account', 'customer', 'gl_account'
    action          VARCHAR(50) NOT NULL,      -- 'read'
    role_id         UUID REFERENCES roles(id),  -- NULL = default mask untuk role
    fields_allowed  TEXT[],                      -- kosong = semua field allowed
    fields_denied   TEXT[],                      -- field yang di-mask
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (org_id, resource, action, role_id)
);

CREATE INDEX idx_perm_field_masks_org ON permission_field_masks(org_id);
CREATE INDEX idx_perm_field_masks_resource ON permission_field_masks(org_id, resource);
```

**Contoh data:**

| resource | action | role | fields_denied |
|---|---|---|---|
| `account` | `read` | teller | `{balance, limit}` |
| `customer` | `read` | cs | `{credit_score, internal_notes}` |
| `account` | `read` | supervisor | `{}` (kosong = semua field) |

### 5.6 `casbin_rule` — Casbin Policy Storage

```sql
CREATE TABLE casbin_rule (
    id SERIAL PRIMARY KEY,
    org_id UUID NOT NULL,
    ptype VARCHAR(100) NOT NULL,
    v0 VARCHAR(100) NOT NULL,
    v1 VARCHAR(100) NOT NULL,
    v2 VARCHAR(100) NOT NULL,
    v3 VARCHAR(100) DEFAULT '',
    v4 VARCHAR(100) DEFAULT '',
    v5 VARCHAR(100) DEFAULT ''
);

CREATE INDEX idx_casbin_rule_org ON casbin_rule(org_id);
CREATE INDEX idx_casbin_rule_ptype ON casbin_rule(ptype, v0);
```

### 5.7 `abac_policies` — ABAC Policy Conditions

```sql
CREATE TABLE abac_policies (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL REFERENCES organizations(id),
    name            VARCHAR(255) NOT NULL,
    description     TEXT,
    policy_type     VARCHAR(20) NOT NULL DEFAULT 'json',  -- 'json' | 'rego'
    policy          JSONB NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_abac_policies_org ON abac_policies(org_id);
CREATE INDEX idx_abac_policies_status ON abac_policies(status);

COMMENT ON COLUMN abac_policies.policy_type IS 'json or rego';
COMMENT ON COLUMN abac_policies.policy IS 'JSON rule or Rego policy';
```

---

## 6. OAuth2 Tables

### 6.1 `oauth2_clients` — Registered Clients

```sql
CREATE TABLE oauth2_clients (
    id                  VARCHAR(255) PRIMARY KEY,   -- client_id
    org_id              UUID NOT NULL REFERENCES organizations(id),
    name                VARCHAR(255) NOT NULL,
    client_type         VARCHAR(50) NOT NULL DEFAULT 'public',
    secret_hash         TEXT,           -- NULL untuk public clients
    redirect_uris       TEXT[] NOT NULL,
    allowed_scopes      TEXT[] NOT NULL,
    allowed_grant_types TEXT[] NOT NULL,
    require_pkce        BOOLEAN NOT NULL DEFAULT TRUE,
    skip_consent        BOOLEAN NOT NULL DEFAULT FALSE,
    token_format        VARCHAR(20) NOT NULL DEFAULT 'jwt',  -- 'jwt' | 'opaque'
    access_token_ttl    INTEGER NOT NULL DEFAULT 900,    -- 15 minutes
    refresh_token_ttl   INTEGER NOT NULL DEFAULT 28800,  -- 8 hours
    id_token_ttl        INTEGER NOT NULL DEFAULT 900,
    registration_policy VARCHAR(50) DEFAULT 'manual',    -- 'manual' | 'automatic' (DCR)
    active              BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_by          UUID
);

CREATE INDEX idx_oauth2_clients_org_id ON oauth2_clients(org_id);
```

### 6.2 `authorization_codes` — Auth Code Flow

```sql
CREATE TABLE authorization_codes (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code                    TEXT NOT NULL UNIQUE,   -- opaque auth code
    client_id               VARCHAR(255) NOT NULL REFERENCES oauth2_clients(id),
    user_id                 UUID NOT NULL REFERENCES users(id),
    org_id                  UUID NOT NULL REFERENCES organizations(id),
    redirect_uri            TEXT NOT NULL,
    scopes                  TEXT[],
    code_challenge          TEXT,    -- PKCE
    code_challenge_method   VARCHAR(10),
    nonce                   TEXT,    -- OIDC nonce
    auth_time               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at              TIMESTAMPTZ NOT NULL,
    used_at                 TIMESTAMPTZ
);

CREATE INDEX idx_auth_codes_expires ON authorization_codes(expires_at);
```

---

## 7. Verification & Recovery Tables

### 7.1 `verification_tokens` — Email & Recovery Tokens

```sql
CREATE TABLE verification_tokens (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id),
    token       TEXT NOT NULL UNIQUE,    -- opaque token (UUID)
    token_type  VARCHAR(50) NOT NULL,    -- email_verification | password_recovery
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_verification_tokens_user_id ON verification_tokens(user_id);
CREATE INDEX idx_verification_tokens_expires ON verification_tokens(expires_at);
```

### 7.2 `email_otp_codes` — Email OTP for MFA

```sql
CREATE TABLE email_otp_codes (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id),
    code        VARCHAR(6) NOT NULL,
    purpose     VARCHAR(50) NOT NULL,  -- 'mfa_verify', 'login', 'password_reset'
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ,
    attempts    INTEGER NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_email_otp_user_id ON email_otp_codes(user_id);
CREATE INDEX idx_email_otp_expires ON email_otp_codes(expires_at);
```

### 7.3 `bulk_import_batches` — CSV Import Tracking

```sql
CREATE TABLE bulk_import_batches (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id          UUID NOT NULL REFERENCES organizations(id),
    filename        VARCHAR(255) NOT NULL,
    total_rows      INTEGER NOT NULL,
    success_count   INTEGER NOT NULL DEFAULT 0,
    failure_count   INTEGER NOT NULL DEFAULT 0,
    status          VARCHAR(50) NOT NULL DEFAULT 'processing',  -- processing | completed | failed
    error_log       JSONB,
    started_by      UUID NOT NULL,
    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at    TIMESTAMPTZ
);

CREATE INDEX idx_bulk_import_org_id ON bulk_import_batches(org_id);
```

---

## 8. Audit Tables

### 8.1 `audit_logs` — Immutable Event Log

```sql
CREATE TABLE audit_logs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type      VARCHAR(100) NOT NULL,
    user_id         UUID,
    actor_id        UUID,          -- siapa yang melakukan (bisa berbeda dari user_id untuk admin actions)
    org_id          UUID NOT NULL REFERENCES organizations(id),
    branch_id       UUID,
    client_id       VARCHAR(255),
    ip_address      INET,
    user_agent      TEXT,
    details         JSONB,         -- event-specific details
    occurred_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
    -- NO updated_at, NO deleted_at — immutable!
) PARTITION BY RANGE (occurred_at);

-- Monthly partitions
CREATE TABLE audit_logs_2026_01 PARTITION OF audit_logs
    FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');
-- ... etc.

CREATE INDEX idx_audit_org_id ON audit_logs(org_id, occurred_at DESC);
CREATE INDEX idx_audit_user_id ON audit_logs(user_id, occurred_at DESC);
CREATE INDEX idx_audit_event_type ON audit_logs(event_type, occurred_at DESC);
```

**Note**: Audit logs di-partition per bulan untuk performa query + archival.
Retention: 5 tahun (sesuai regulasi perbankan Indonesia).

---

## 9. Redis Key Patterns

```
# Active sessions
session:{session_id}                     → JSON session data, TTL = session expiry

# Rate limiting & brute force
rate:login:{org_id}:{username}           → attempt count, TTL = window
rate:totp:{user_id}                      → TOTP attempt count, TTL = 5min
rate:api:{ip}                            → API rate limit counter

# Token blacklist
blacklist:jti:{jti}                      → "1", TTL = token remaining expiry

# TOTP replay prevention
totp:used:{user_id}:{period}:{code}      → "1", TTL = 90s

# Refresh token lock (thundering herd prevention)
lock:refresh:{token_hash}                → "1", TTL = 5s

# Policy cache invalidation
policy:version:{org_id}                  → version number

# Policy pub/sub (Redis pub/sub channel)
policy:updated                           → channel for policy change notifications

# MFA login temp tokens
mfa_login:{token}                        → JSON (user_id, org_id, ...), TTL = 5min

# Email OTP codes
email_otp:{user_id}:{purpose}            → JSON (code, expires_at, attempts), TTL = 10min

# Lockout
lockout:{user_id}                        → JSON (attempts, locked_until), TTL = 15min
```

---

## 10. Indexing Strategy

| Table | Indexes | Alasan |
|---|---|---|
| users | org_id, status, deleted_at | Multi-tenant queries |
| user_branch_assignments | user_id, branch_id, is_primary | Branch access queries |
| branches | org_id, branch_type_id | Multi-tenant + type queries |
| roles | org_id | Multi-tenant queries |
| oauth2_clients | org_id | Multi-tenant queries |
| authorization_codes | expires_at | Cleanup expired codes |
| refresh_tokens | user_id, session_id, family_id, expires_at | Lookup + cleanup + reuse detection |
| audit_logs | org_id, user_id, event_type, occurred_at | Query + partitioning |
| casbin_rule | org_id, ptype+v0 | Policy lookup |
| abac_policies | org_id, status | Policy evaluation |

---

## 11. Migration Strategy

- **Tool**: golang-migrate
- **Directory**: `migrations/`
- **Naming**: `20260422000001_create_organizations.up.sql`
- **Down migrations**: Paired untuk setiap up migration
- **Idempotent**: Safe to re-run

---

> Semua open questions telah dijawab di [OPEN-QUESTIONS.md](../OPEN-QUESTIONS.md).

*Prev: [07-admin-api.md](./07-admin-api.md) | Next: [09-integration.md](./09-integration.md)*
