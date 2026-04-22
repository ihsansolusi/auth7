# Auth7 — Spec 01: Architecture

> **Versi**: 1.0-draft | **Tanggal**: 2026-04-22 | **Fase**: Brainstorming

---

## 1. Arsitektur Sistem

Auth7 adalah **satu Go service** dengan domain internal yang jelas, mengikuti pola clean architecture dari `service7-template`.

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

## 3. Deployment

### 3.1 Topology

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

### 3.2 Domain Setup

| Domain | Target | Deskripsi |
|---|---|---|
| `auth.bank.co.id` | auth7-ui (Next.js) | Login, recovery, admin UI |
| `auth7.bank.co.id` | auth7-svc (Go API) | REST + gRPC API |
| `*.bank.co.id` | Aplikasi lain | bos7-portal, workflow7-web, dll |

### 3.3 Environment Variables

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

## 4. Konvensi Arsitektur

Konsisten dengan `service7-template`:

- Setiap Go method: `const op = "package.Type.Method"`
- Setiap Go method: `logging.WithTrace(ctx, logger)` + `ctx, span := tracer.Start(ctx, op)`
- Semua error di-wrap: `fmt.Errorf("%s: %w", op, err)`
- Store method multi-tenant: wajib terima `orgID uuid.UUID` eksplisit
- Tidak ada secret di config file — hanya `"${ENV_VAR}"`

---

## 5. Open Questions

1. **Apakah perlu health check endpoint terpisah?**
   → Ya: `/health` (liveness) dan `/ready` (readiness)

2. **Apakah perlu graceful shutdown?**
   → Ya: drain connections, close DB pool, flush audit buffer

---

*Prev: [00-overview.md](./00-overview.md) | Next: [02-identity.md](./02-identity.md)*
