# Auth7 — Spec 01: Architecture

> **Versi**: 1.0-draft | **Tanggal**: 2026-04-22 | **Fase**: Brainstorming

---

## 1. Arsitektur Sistem

Auth7 mengikuti **clean architecture** yang sama dengan `service7-template`, dengan modifikasi
untuk kebutuhan identity platform:

```
┌─────────────────────────────────────────────────────────────┐
│                        auth7-svc                            │
│                                                             │
│  ┌──────────────────────────────────────────────────────┐  │
│  │                    cmd/                               │  │
│  │  main.go → wire.go (DI) → server.go (bootstrap)      │  │
│  └──────────────────────┬───────────────────────────────┘  │
│                         │                                   │
│  ┌──────────────────────▼───────────────────────────────┐  │
│  │                  internal/api/                        │  │
│  │  REST Handlers + gRPC handlers + Middleware           │  │
│  │  ┌─────────┐ ┌────────┐ ┌────────┐ ┌─────────────┐  │  │
│  │  │identity │ │oauth2  │ │authz   │ │admin        │  │  │
│  │  │handlers │ │handlers│ │handlers│ │handlers     │  │  │
│  │  └────┬────┘ └───┬────┘ └───┬────┘ └──────┬──────┘  │  │
│  └───────┼──────────┼──────────┼─────────────┼──────────┘  │
│          │          │          │             │             │
│  ┌───────▼──────────▼──────────▼─────────────▼──────────┐  │
│  │                 internal/service/                     │  │
│  │  Business logic, flows, orchestration                 │  │
│  │  ┌─────────┐ ┌────────┐ ┌────────┐ ┌─────────────┐  │  │
│  │  │identity │ │oauth2  │ │authz   │ │audit        │  │  │
│  │  │service  │ │service │ │service │ │service      │  │  │
│  │  └────┬────┘ └───┬────┘ └───┬────┘ └──────┬──────┘  │  │
│  └───────┼──────────┼──────────┼─────────────┼──────────┘  │
│          │          │          │             │             │
│  ┌───────▼──────────▼──────────▼─────────────▼──────────┐  │
│  │                  internal/store/                      │  │
│  │  Data access layer (pgx + sqlc)                       │  │
│  └───────────────────────┬───────────────────────────────┘  │
│                          │                                  │
│  ┌───────────────────────▼───────────────────────────────┐  │
│  │                  internal/domain/                     │  │
│  │  Entities, value objects, interfaces, errors          │  │
│  └───────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### 1.1 Clean Architecture Layers

```
cmd/ → internal/api/ → internal/service/ → internal/store/ → internal/domain/
```

| Layer | Fungsi |
|---|---|
| `cmd/` | Entry point, DI wiring, config loading |
| `internal/api/` | HTTP handlers (Gin), gRPC handlers, middleware |
| `internal/service/` | Business logic, orchestration |
| `internal/store/` | Database access (pgx + sqlc), Redis operations |
| `internal/domain/` | Entities, value objects, interfaces, errors |

### 1.2 Domain Modules

```
internal/
├── identity/       # User lifecycle, credentials, profiles
├── oauth2/         # OAuth2/OIDC server, token issuance
├── authz/          # RBAC + ABAC engine (Casbin + OPA)
├── session/        # Session management (Redis-backed)
├── mfa/            # TOTP + Email OTP
├── audit/          # Immutable audit trail
├── tenant/         # Multi-tenancy (org + branch)
├── admin/          # Admin API handlers
└── gateway/        # Request auth middleware (JWT verify, introspection)
```

### 1.3 Domain Module Details

#### `identity/` — Identity & User Management
Mengurus seluruh lifecycle user:
- Register: create user + credential
- Login: verify credential → issue session/token
- Logout: revoke session/token
- Password change, recovery (forgot password)
- Email verification
- Profile management (self-service)

**Analogi**: Ory Kratos

#### `oauth2/` — OAuth2 & OIDC Server
Menjadi authorization server standar:
- Authorization Code Flow + PKCE
- Client Credentials Flow (M2M)
- Implicit Flow (deprecated, disabled by default)
- Token introspection (RFC 7662)
- Token revocation (RFC 7009)
- JWKS endpoint (public keys)
- OIDC Discovery (`/.well-known/openid-configuration`)
- UserInfo endpoint

**Analogi**: Ory Hydra

#### `authz/` — Authorization Engine
Fine-grained access control:
- RBAC: `user → role → permission → resource:action`
- ABAC extension: policy conditions (branch, time, IP)
- Permission check API (batch + single)
- Policy management (CRUD roles, permissions)
- Casbin enforcement backend

**Analogi**: Ory Keto (simplified, Casbin-based)

#### `session/` — Session Management
- Server-side sessions (Redis-backed)
- Session metadata: IP, user-agent, device fingerprint
- Session listing & revocation (admin + self-service)
- Sliding window expiry

#### `mfa/` — Multi-Factor Authentication
- TOTP enrollment & verification
- Backup codes
- MFA policy per user/role/tenant
- Recovery flow when MFA device lost

#### `audit/` — Audit Trail
- Append-only event log (tidak bisa dihapus)
- Events: login, logout, failed attempt, password change, permission change, etc.
- Query API untuk admin
- Compliance dengan standar perbankan OJK

#### `tenant/` — Multi-Tenancy
- Organization (Bank level)
- Branch hierarchy (HEAD_OFFICE → REGIONAL → BRANCH → SUB_BRANCH → CASH_OFFICE)
- Branch-to-branch relationships (parent/child)
- Tenant-scoped user, role, permission
- Cross-tenant admin capability

#### `admin/` — Administration API
- User CRUD
- Role & permission management
- OAuth2 client management
- Tenant management
- Audit log query

#### `gateway/` — Request Verification Middleware
- Verifikasi JWT dari request header
- Introspection proxy
- gRPC interceptor untuk inter-service auth
- Rate limiting middleware

### 1.3 Analogi Ory Stack

| Auth7 Domain | Ory Equivalent | Fungsi |
|---|---|---|
| `identity/` | Ory Kratos | User management, login, register, recovery |
| `oauth2/` | Ory Hydra | OAuth2 + OIDC token server |
| `authz/` | Ory Keto | Permission check, RBAC + ABAC |
| `gateway/` | Ory Oathkeeper | Request auth middleware |
| `audit/` | Zitadel events | Immutable audit trail |
| `tenant/` | Zitadel orgs | Bank branch/cabang isolation |

---

## 2. API Surface

### 2.1 Public API (`/api/v1/`)

| Endpoint | Method | Deskripsi |
|---|---|---|
| `/auth/login` | POST | Username/password login |
| `/auth/login/mfa` | POST | MFA verification |
| `/auth/logout` | POST | Logout + revoke session |
| `/auth/refresh` | POST | Refresh token |
| `/auth/recover` | POST | Request password recovery |
| `/auth/recover/{token}` | PUT | Reset password |
| `/auth/verify-email` | POST | Email verification |
| `/auth/setup-password` | POST | First-time password setup |
| `/me` | GET | Current user profile |
| `/me/password` | PUT | Change password |
| `/me/mfa/totp/setup` | POST | Generate TOTP secret |
| `/me/mfa/totp/activate` | POST | Activate TOTP |
| `/me/mfa/totp/deactivate` | DELETE | Deactivate TOTP |
| `/me/sessions` | GET | List active sessions |
| `/me/sessions/{id}` | DELETE | Revoke session |

### 2.2 OAuth2/OIDC API (`/oauth2/`)

| Endpoint | Method | Deskripsi |
|---|---|---|
| `/oauth2/authorize` | GET | Authorization endpoint |
| `/oauth2/token` | POST | Token endpoint |
| `/oauth2/revoke` | POST | Token revocation (RFC 7009) |
| `/oauth2/introspect` | POST | Token introspection |
| `/oauth2/userinfo` | GET | OIDC userinfo |
| `/.well-known/openid-configuration` | GET | OIDC Discovery |
| `/.well-known/jwks.json` | GET | JWKS endpoint |

### 2.3 Admin API (`/admin/v1/`)

| Endpoint | Method | Deskripsi |
|---|---|---|
| `/admin/v1/users` | GET/POST | User list, create |
| `/admin/v1/users/{id}` | GET/PUT/DELETE | User CRUD |
| `/admin/v1/users/import` | POST | Bulk CSV import |
| `/admin/v1/roles` | GET/POST | Role list, create |
| `/admin/v1/roles/{id}` | GET/PUT/DELETE | Role CRUD |
| `/admin/v1/roles/{id}/permissions` | POST | Assign permissions |
| `/admin/v1/branches` | GET/POST | Branch list, create |
| `/admin/v1/branches/{id}` | GET/PUT/DELETE | Branch CRUD |
| `/admin/v1/clients` | GET/POST | OAuth2 client list, create |
| `/admin/v1/clients/{id}` | GET/PUT/DELETE | Client CRUD |
| `/admin/v1/permissions` | GET | List all permissions |
| `/admin/v1/audit-logs` | GET | Query audit logs |
| `/admin/v1/branding/{org}` | GET/PUT | Branding config |

### 2.4 gRPC Service

| Service | Method | Deskripsi |
|---|---|---|
| `AuthService` | `Authenticate` | Verify credentials |
| `AuthService` | `IntrospectToken` | Token introspection |
| `AuthzService` | `CheckPermission` | Permission check |
| `AuthzService` | `ListPermissions` | List user permissions |
| `TenantService` | `GetOrg` | Get org details |
| `TenantService` | `GetBranch` | Get branch details |

---

## 4. Deployment

### 4.1 Topology

```
┌─────────────────────────────────────────────────────┐
│                  Core7 Ecosystem                    │
│                                                     │
│  ┌──────────┐    ┌──────────────┐                  │
│  │ auth7-ui │    │ bos7-portal  │                  │
│  │ (Next.js)│    │              │                  │
│  └────┬─────┘    └──────┬───────┘                  │
│       │                │                           │
│       ▼                ▼                           │
│  ┌─────────────────────────────┐                  │
│  │         auth7               │                  │
│  │    (Port 8080 REST/gRPC)    │                  │
│  └──────────────┬──────────────┘                  │
│                 │                                 │
│       ┌─────────┴─────────┐                      │
│       ▼                   ▼                      │
│  ┌──────────┐       ┌───────────┐                │
│  │PostgreSQL│       │   Redis   │                │
│  │   16     │       │   (req)   │                │
│  └──────────┘       └───────────┘                │
│                                                   │
│  ◄── Protected Services ─────────────►           │
│  workflow7-svc, notif7-svc, ...                  │
└───────────────────────────────────────────────────┘
```

### 4.2 Production Topology

```
                    ┌──────────────┐
                    │   Nginx /    │
                    │  API Gateway │
                    └──────┬───────┘
                           │
              ┌────────────┼────────────┐
              ▼            ▼            ▼
         ┌─────────┐  ┌─────────┐  ┌─────────┐
         │auth7-svc│  │auth7-svc│  │auth7-svc│  ← horizontally scalable
         │  :8080  │  │  :8080  │  │  :8080  │
         └────┬────┘  └────┬────┘  └────┬────┘
              └────────────┼────────────┘
                           │
              ┌────────────┼────────────┐
              ▼                         ▼
         ┌──────────┐           ┌──────────────┐
         │PostgreSQL│           │  Redis Cluster│
         │ (primary)│           │  (session +   │
         │  + read  │           │   rate limit) │
         │ replicas │           └──────────────┘
         └──────────┘
```

### 4.3 Development

```
docker-compose.yml:
  auth7-svc:    localhost:8080
  postgres:     localhost:5432
  redis:        localhost:6379
  auth7-ui:     localhost:3000 (Next.js dev server)
```

### 4.4 Domain Setup

| Domain | Target | Deskripsi |
|---|---|---|
| `auth.bank.co.id` | auth7-ui (Next.js) | Login, recovery, admin UI |
| `auth7.bank.co.id` | auth7-svc (Go API) | REST + gRPC API |
| `*.bank.co.id` | Aplikasi lain | bos7-portal, workflow7-web, dll |

### 4.5 Environment Variables

```env
# Server
AUTH7_HTTP_PORT=8080
AUTH7_GRPC_PORT=9090
AUTH7_LOG_LEVEL=info

# Database
AUTH7_DB_HOST=localhost
AUTH7_DB_PORT=5432
AUTH7_DB_NAME=auth7
AUTH7_DB_USER=${AUTH7_DB_USER}
AUTH7_DB_PASSWORD=${AUTH7_DB_PASSWORD}
AUTH7_DB_SSL_MODE=disable

# Redis (wajib)
AUTH7_REDIS_HOST=localhost
AUTH7_REDIS_PORT=6379
AUTH7_REDIS_PASSWORD=${AUTH7_REDIS_PASSWORD}
AUTH7_REDIS_DB=0

# JWT
AUTH7_JWT_PRIVATE_KEY_PATH=/etc/auth7/jwt_private.pem
AUTH7_JWT_PUBLIC_KEY_PATH=/etc/auth7/jwt_public.pem
AUTH7_ACCESS_TOKEN_TTL=900        # 15 menit
AUTH7_REFRESH_TOKEN_TTL=28800     # 8 jam

# OAuth2
AUTH7_OAUTH2_ISSUER=https://auth7.bank.co.id

# Encryption
AUTH7_ENCRYPTION_KEY=${AUTH7_ENCRYPTION_KEY}  # KEK dari env var
```

---

## 5. Request Flow: Login

```
Browser/App                auth7-ui              auth7-svc         PostgreSQL    Redis
    │                          │                     │                  │          │
    │── POST /login ──────────►│                     │                  │          │
    │                          │── POST /api/v1/     │                  │          │
    │                          │   auth/login ──────►│                  │          │
    │                          │                     │── SELECT user ──►│          │
    │                          │                     │◄── user row ─────│          │
    │                          │                     │                  │          │
    │                          │                     │ [verify argon2id]│          │
    │                          │                     │                  │          │
    │                          │                     │── [if MFA needed]│          │
    │                          │                     │    return mfa_   │          │
    │                          │                     │    required      │          │
    │                          │                     │                  │          │
    │                          │                     │── [if OK] create │          │
    │                          │                     │   session ──────────────►  │
    │                          │                     │                  │          │
    │                          │                     │── INSERT audit ──►│          │
    │                          │                     │                  │          │
    │                          │◄── {session_id,     │                  │          │
    │                          │     access_token,   │                  │          │
    │                          │     refresh_token}  │                  │          │
    │◄── Set-Cookie: session ──│                     │                  │          │
    │    + tokens              │                     │                  │          │
```

---

## 6. Request Flow: Protected Resource Access

```
Client App           core7-service         auth7-svc (verify)      PostgreSQL
    │                     │                       │                     │
    │── GET /resource ───►│                       │                     │
    │   Authorization:    │                       │                     │
    │   Bearer <token>    │                       │                     │
    │                     │── [middleware]         │                     │
    │                     │   verify JWT locally  │                     │
    │                     │   (JWKS public key)    │                     │
    │                     │                       │                     │
    │                     │── [if introspect needed]                    │
    │                     │   POST /oauth2/        │                     │
    │                     │   introspect ─────────►│                     │
    │                     │                       │── check token ──────►│
    │                     │                       │◄── active/inactive ──│
    │                     │◄── {active, sub, ..}  │                     │
    │                     │                       │                     │
    │                     │── [check permission]   │                     │
    │                     │   gRPC CheckPerm ─────►│                     │
    │                     │◄── {allowed: true}     │                     │
    │                     │                       │                     │
    │◄── 200 OK ──────────│                       │                     │
```

---

## 7. Konvensi Arsitektur

Konsisten dengan `service7-template`:

- Setiap Go method: `const op = "package.Type.Method"`
- Setiap Go method: `logging.WithTrace(ctx, logger)` + `ctx, span := tracer.Start(ctx, op)`
- Semua error di-wrap: `fmt.Errorf("%s: %w", op, err)`
- Store method multi-tenant: wajib terima `orgID uuid.UUID` eksplisit
- Tidak ada secret di config file — hanya `"${ENV_VAR}"`

---

## 8. Open Questions

1. **Apakah perlu health check endpoint terpisah?**
   → Ya: `/health` (liveness) dan `/ready` (readiness)

2. **Apakah perlu graceful shutdown?**
   → Ya: drain connections, close DB pool, flush audit buffer

3. **Apakah perlu gRPC server terpisah atau multiplexed dengan HTTP?**
   → Rekomendasi: cmux untuk multiplex HTTP + gRPC di port yang sama

4. **Rate limiting: Redis-based atau in-memory bucket?**
   → Redis-based lebih cocok untuk deployment multi-instance

5. **JWKS key rotation: manual atau auto-rotate?**
   → v1.0: manual rotation via admin API
   → v2.0: auto-rotate dengan grace period

6. **Apakah auth7-svc perlu read replica support?**
   → Iya, untuk scale query-heavy operations (audit log query, token introspection)

---

*Prev: [00-overview.md](./00-overview.md) | Next: [02-identity.md](./02-identity.md)*
