# Auth7 — Spec 08: Data Model

> **Versi**: 1.0-draft | **Tanggal**: 2026-04-22 | **Fase**: Brainstorming

---

## 1. PostgreSQL Schema

### 1.1 Organizations

```sql
CREATE TABLE organizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(50) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    settings JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON COLUMN organizations.settings IS 'Org-level config: session_policy, mfa_policy, password_policy, branding';
```

### 1.2 Branches

```sql
CREATE TABLE branches (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id),
    code VARCHAR(50) NOT NULL,
    name VARCHAR(255) NOT NULL,
    branch_type VARCHAR(50) NOT NULL DEFAULT 'cabang',
    parent_id UUID REFERENCES branches(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(org_id, code)
);

CREATE INDEX idx_branches_org ON branches(org_id);
CREATE INDEX idx_branches_parent ON branches(parent_id);
```

### 1.3 Users

```sql
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id),
    branch_id UUID REFERENCES branches(id),
    username VARCHAR(100) NOT NULL,
    email VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    email_verified BOOLEAN NOT NULL DEFAULT false,
    mfa_enabled BOOLEAN NOT NULL DEFAULT false,
    mfa_method VARCHAR(20),
    failed_attempts INT NOT NULL DEFAULT 0,
    locked_until TIMESTAMPTZ,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ,
    UNIQUE(org_id, username),
    UNIQUE(org_id, email)
);

CREATE INDEX idx_users_org ON users(org_id);
CREATE INDEX idx_users_branch ON users(branch_id);
CREATE INDEX idx_users_status ON users(status);
CREATE INDEX idx_users_deleted ON users(deleted_at) WHERE deleted_at IS NOT NULL;
```

### 1.4 Roles

```sql
CREATE TABLE roles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id),
    name VARCHAR(100) NOT NULL,
    description TEXT,
    is_system BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(org_id, name)
);

CREATE INDEX idx_roles_org ON roles(org_id);
```

### 1.5 User Roles

```sql
CREATE TABLE user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    assigned_by UUID REFERENCES users(id),
    assigned_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, role_id)
);

CREATE INDEX idx_user_roles_role ON user_roles(role_id);
```

### 1.6 MFA Configs

```sql
CREATE TABLE mfa_configs (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    method VARCHAR(20) NOT NULL,
    totp_secret VARCHAR(255),
    totp_activated BOOLEAN NOT NULL DEFAULT false,
    backup_codes JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMENT ON COLUMN mfa_configs.totp_secret IS 'Encrypted at-rest';
COMMENT ON COLUMN mfa_configs.backup_codes IS 'Array of hashed backup codes';
```

### 1.7 OAuth2 Clients

```sql
CREATE TABLE oauth2_clients (
    id VARCHAR(100) PRIMARY KEY,
    org_id UUID NOT NULL REFERENCES organizations(id),
    name VARCHAR(255) NOT NULL,
    client_type VARCHAR(20) NOT NULL DEFAULT 'confidential',
    client_secret VARCHAR(255),
    redirect_uris JSONB NOT NULL DEFAULT '[]',
    allowed_scopes JSONB NOT NULL DEFAULT '[]',
    allowed_grant_types JSONB NOT NULL DEFAULT '[]',
    require_pkce BOOLEAN NOT NULL DEFAULT true,
    token_format VARCHAR(20) NOT NULL DEFAULT 'jwt',
    access_token_ttl INT NOT NULL DEFAULT 900,
    refresh_token_ttl INT NOT NULL DEFAULT 28800,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_oauth2_clients_org ON oauth2_clients(org_id);
```

### 1.8 Authorization Codes

```sql
CREATE TABLE authorization_codes (
    code VARCHAR(100) PRIMARY KEY,
    client_id VARCHAR(100) NOT NULL REFERENCES oauth2_clients(id),
    user_id UUID NOT NULL REFERENCES users(id),
    redirect_uri TEXT NOT NULL,
    scope VARCHAR(255) NOT NULL,
    code_challenge VARCHAR(255),
    code_challenge_method VARCHAR(10),
    nonce VARCHAR(255),
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_auth_codes_expires ON authorization_codes(expires_at);

-- Auto-cleanup expired codes
-- Application-level cleanup or pg_cron job
```

### 1.9 Refresh Tokens

```sql
CREATE TABLE refresh_tokens (
    jti UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    client_id VARCHAR(100) NOT NULL REFERENCES oauth2_clients(id),
    session_id UUID NOT NULL,
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    revoked BOOLEAN NOT NULL DEFAULT false,
    revoked_at TIMESTAMPTZ,
    replaced_by UUID REFERENCES refresh_tokens(jti),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_refresh_tokens_user ON refresh_tokens(user_id);
CREATE INDEX idx_refresh_tokens_session ON refresh_tokens(session_id);
CREATE INDEX idx_refresh_tokens_expires ON refresh_tokens(expires_at);
```

### 1.10 Audit Logs

```sql
CREATE TABLE audit_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id),
    user_id UUID REFERENCES users(id),
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100),
    resource_id UUID,
    ip_address INET,
    user_agent TEXT,
    details JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
) PARTITION BY RANGE (created_at);

-- Monthly partitions (managed by migration script)
CREATE TABLE audit_logs_2026_04 PARTITION OF audit_logs
    FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');

CREATE INDEX idx_audit_logs_org ON audit_logs(org_id);
CREATE INDEX idx_audit_logs_user ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
CREATE INDEX idx_audit_logs_created ON audit_logs(created_at);

COMMENT ON TABLE audit_logs IS 'Immutable audit trail - 5 year retention';
```

### 1.11 Casbin Rules

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

### 1.12 ABAC Policies

```sql
CREATE TABLE abac_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    org_id UUID NOT NULL REFERENCES organizations(id),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    policy_type VARCHAR(20) NOT NULL DEFAULT 'json',
    policy JSONB NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_abac_policies_org ON abac_policies(org_id);
CREATE INDEX idx_abac_policies_status ON abac_policies(status);

COMMENT ON COLUMN abac_policies.policy_type IS 'json or rego';
COMMENT ON COLUMN abac_policies.policy IS 'JSON rule or Rego policy';
```

### 1.13 Password History

```sql
CREATE TABLE password_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_password_history_user ON password_history(user_id);

-- Keep last 5 passwords per user
-- Application-level cleanup
```

### 1.14 Recovery Tokens

```sql
CREATE TABLE recovery_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used BOOLEAN NOT NULL DEFAULT false,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_recovery_tokens_user ON recovery_tokens(user_id);
CREATE INDEX idx_recovery_tokens_expires ON recovery_tokens(expires_at);

-- TTL: 15 menit
```

### 1.15 Email Verification Tokens

```sql
CREATE TABLE email_verification_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id),
    token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    used BOOLEAN NOT NULL DEFAULT false,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_email_verif_tokens_user ON email_verification_tokens(user_id);
CREATE INDEX idx_email_verif_tokens_expires ON email_verification_tokens(expires_at);

-- TTL: 24 jam
```

---

## 2. Redis Key Patterns

### 2.1 Session Store

```
Key: session:{session_id}
Value: JSON session object
TTL: 8 jam
```

### 2.2 Rate Limiting

```
Key: ratelimit:{ip}:{endpoint}
Value: counter
TTL: 1 menit (sliding window)
```

### 2.3 Login Token (MFA step)

```
Key: login_token:{token}
Value: {user_id, client_id, scope, created_at}
TTL: 5 menit
```

### 2.4 Policy Cache

```
Key: policy:casbin:{org_id}
Value: serialized Casbin rules
TTL: 1 jam (invalidated via pub/sub on change)
```

### 2.5 Policy Pub/Sub

```
Channel: policy:updated
Message: {org_id, timestamp, action}
```

### 2.6 Lockout

```
Key: lockout:{user_id}
Value: {attempts, locked_until}
TTL: 15 menit
```

---

## 3. Indexing Strategy

| Table | Indexes | Alasan |
|---|---|---|
| users | org_id, branch_id, status, deleted_at | Multi-tenant queries |
| branches | org_id, parent_id | Hierarchy queries |
| roles | org_id | Multi-tenant queries |
| oauth2_clients | org_id | Multi-tenant queries |
| authorization_codes | expires_at | Cleanup expired codes |
| refresh_tokens | user_id, session_id, expires_at | Lookup + cleanup |
| audit_logs | org_id, user_id, action, created_at | Query + partitioning |
| casbin_rule | org_id, ptype+v0 | Policy lookup |
| abac_policies | org_id, status | Policy evaluation |

---

## 4. Migration Strategy

- **Tool**: golang-migrate
- **Directory**: `migrations/`
- **Naming**: `20260422000001_create_organizations.up.sql`
- **Down migrations**: Paired untuk setiap up migration
- **Idempotent**: Safe to re-run

---

## 5. Open Questions

1. **Apakah perlu read replica untuk v1.0?**
   → Tidak (bisa ditambahkan v1.1 jika diperlukan)

2. **Apakah perlu full-text search untuk user lookup?**
   → v1.0: LIKE query saja
   → v1.1: pg_trgm + GIN index

---

*Prev: [07-admin-api.md](./07-admin-api.md) | Next: [09-integration.md](./09-integration.md)*
