# Auth7 — Spec 00: Overview & Vision

> **Versi**: 1.0-draft | **Tanggal**: 2026-04-22 | **Fase**: Brainstorming

---

## 1. Latar Belakang

Core7 adalah platform core banking baru yang menggantikan DAF BJBS. Setiap service dalam ekosistem
Core7 membutuhkan **autentikasi dan otorisasi** yang andal, aman, dan dapat di-audit sesuai standar
perbankan Indonesia.

Opsi yang sudah dipertimbangkan:
- **Ory Stack** (Kratos + Hydra + Keto + Oathkeeper) — open-source, modular, battle-tested
- **Zitadel** — open-source, event-driven, multi-tenant, OIDC compliant
- **Auth7** *(pilihan ini)* — custom-built, terintegrasi penuh dengan Core7, zero external dependency

### Mengapa membangun sendiri?

| Pertimbangan | Ory/Zitadel | Auth7 |
|---|---|---|
| Kontrol penuh | ❌ terbatas | ✅ full control |
| Integrasi Core7 native | ❌ perlu adapter | ✅ native |
| Compliance OJK/BI custom | ❌ generic | ✅ by design |
| Multi-tenant banking model | ⚠️ general purpose | ✅ bank-specific (branch/cabang) |
| Audit trail format custom | ❌ | ✅ |
| Dependency eksternal | ❌ banyak | ✅ zero |
| Kurva belajar tim | ❌ tinggi | ✅ rendah (familiar dengan Go+PG) |

---

## 2. Visi Auth7

> **Auth7** adalah *identity & access platform* untuk ekosistem Core7 yang menyediakan:
> autentikasi berbasis standar (OAuth2/OIDC), manajemen identitas headless,
> otorisasi fine-grained berbasis roles dan relasi, serta audit trail yang comply
> dengan regulasi perbankan Indonesia.

---

## 3. Komponen Utama

Auth7 didesain sebagai **satu Go service** dengan domain internal yang jelas:

```
auth7/
├── cmd/
├── internal/
│   ├── identity/       # User identity, credentials, lifecycle flows
│   ├── oauth2/         # OAuth2 + OIDC server (token issuance)
│   ├── authz/          # Authorization engine (RBAC + ABAC)
│   ├── session/        # Session management
│   ├── mfa/            # Multi-factor authentication
│   ├── audit/          # Audit log & event sourcing
│   ├── tenant/         # Multi-tenancy (branch/org isolation)
│   ├── admin/          # Admin API
│   └── gateway/        # Middleware / request verification
├── docs/
├── migrations/
└── proto/
```

Repo ini murni untuk service Go. UI terpisah di repo `ihsansolusi/auth7-ui`.

### Analoginya dengan Ory Stack

| Auth7 Domain | Ory Equivalent | Fungsi |
|---|---|---|
| `identity/` | Ory Kratos | User management, login, register, recovery |
| `oauth2/` | Ory Hydra | OAuth2 + OIDC token server |
| `authz/` | Ory Keto | Permission check, RBAC + relasi |
| `gateway/` | Ory Oathkeeper | Request auth middleware |
| `audit/` | Zitadel event sourcing | Immutable audit trail |
| `tenant/` | Zitadel organizations | Bank branch/cabang isolation |

---

## 4. Scope v1.0

### In Scope
- [x] Username/password login dengan argon2id
- [x] JWT Access Token (15 menit) + Refresh Token (8 jam)
- [x] Session-based auth (browser apps, HTTP-only cookies)
- [x] OAuth2 Authorization Code Flow + PKCE
- [x] OpenID Connect (OIDC) — ID Token, userinfo, Discovery
- [x] TOTP-based MFA (Google Authenticator compatible) + Email OTP
- [x] RBAC + ABAC (JSON Rules + OPA Rego hybrid)
- [x] Multi-tenant isolation (per-cabang/per-bank)
- [x] User self-service: ganti password, lupa password, verifikasi email
- [x] Admin API: CRUD user, role, client OAuth2
- [x] Audit log: semua auth events tersimpan permanen (5 tahun)
- [x] Machine-to-machine (M2M): client credentials flow
- [x] Dynamic Client Registration (RFC 7591)
- [x] Bulk import user dari CSV
- [x] Token format: JWT (default) + Opaque (configurable per client)

### Out of Scope v1.0 (Future)
- [ ] Passkeys / FIDO2 / WebAuthn
- [ ] SAML 2.0
- [ ] Social login (Google, GitHub, etc.)
- [ ] SMS-based OTP (dipertimbangkan v1.1)
- [ ] Zanzibar-style fine-grained authz (lebih ke v2.0)
- [ ] Identity federation (external IdP brokering)
- [ ] User impersonation (v1.1)
- [ ] Consent screen (v2.0)
- [ ] HSM untuk JWT key (v2.0)
- [ ] Dual approval / 4-eyes (v2.0 via workflow7)

---

## 5. Prinsip Desain

### 5.1 API-First & Headless
Auth7 hanya menyediakan REST API + gRPC. UI (login page, admin panel) adalah aplikasi terpisah:
- `auth7-ui` — login, register, password recovery pages (Next.js, repo sendiri)
- `auth7-playground` — testing & demo (akan di-reset dari awal)

### 5.2 Single Binary, Modular Internals
Satu binary Go, tapi domain internal terisolasi dengan clean architecture persis seperti `service7-template`:
```
cmd/ → internal/api/ → internal/service/ → internal/store/ → internal/domain/
```

### 5.3 Banking-Grade Security
- Semua password di-hash dengan **Argon2id** (bukan bcrypt)
- Token disimpan dengan enkripsi at-rest (AES-256-GCM)
- Rate limiting per-IP dan per-user
- Brute force protection: lockout setelah 5 gagal
- Audit trail tidak dapat dihapus (append-only, 5 tahun)

### 5.4 Multi-Tenant (Bank/Cabang Model)
```
Organization (Bank)
  └── Branch (Cabang)
        └── User
              └── Roles
```
Setiap request auth di-scope ke `org_id` dan `branch_id`.

### 5.5 Standards Compliant
- **OAuth 2.0**: RFC 6749, RFC 7636 (PKCE), RFC 7009 (Token Revocation), RFC 7591 (DCR)
- **OIDC**: OpenID Connect Core 1.0, Discovery (RFC 8414)
- **JWT**: RFC 7519, RFC 7523 (JWT Profile)
- **MFA**: RFC 6238 (TOTP), RFC 4226 (HOTP)

---

## 6. Teknologi Stack

| Komponen | Teknologi |
|---|---|
| Language | Go 1.22+ |
| Framework | Gin (HTTP REST) + net/grpc |
| Database | PostgreSQL 16 (pgx + sqlc) |
| Cache / Session | Redis (wajib v1.0) |
| Password Hashing | Argon2id (`golang.org/x/crypto`) |
| JWT | `golang-jwt/jwt/v5` (RS256) |
| TOTP | `pquerna/otp` |
| Authorization | Casbin (RBAC) + OPA (ABAC/Rego) |
| Observability | zerolog + OpenTelemetry + Prometheus |
| Migrations | golang-migrate |
| Config | env-based, no secrets in files |

---

## 7. Deployment

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
│  │   16     │       │  (wajib)  │                │
│  └──────────┘       └───────────┘                │
│                                                   │
│  ◄── Protected Services ─────────────►           │
│  workflow7-svc, notif7-svc, ...                  │
└───────────────────────────────────────────────────┘
```

Protected services memverifikasi token via:
1. **JWT verification** (public key dari JWKS endpoint) — stateless, zero-latency
2. **Introspection endpoint** — untuk token yang perlu dicek real-time validity
3. **gRPC AuthCheck** — untuk inter-service communication

---

## 8. Keputusan Desain (v1.0)

Semua keputusan tercatat di [`../OPEN-QUESTIONS.md`](../OPEN-QUESTIONS.md). Ringkasan:

| Keputusan | Value |
|---|---|
| Access token TTL | 15 menit |
| Refresh token TTL | 8 jam |
| Redis | Wajib v1.0 |
| Token format | JWT + Opaque (configurable per client) |
| Casbin adapter | Custom pgx |
| ABAC | Hybrid JSON Rules + OPA Rego |
| Policy sync | Redis pub/sub |
| MFA | TOTP + Email OTP (setiap login, no trusted device) |
| Max concurrent sessions | Configurable per org |
| IP binding | Soft (warn saja) |
| Bulk import CSV | v1.0 |
| DCR (RFC 7591) | v1.0 |
| OIDC Discovery | Ya |
| Audit retention | 5 tahun |
| Admin rate limit | 10 req/s |
| HSM | v2.0 |
| Consent screen | v2.0 |
| User impersonation | v1.1 |
| Dual approval | v2.0 via workflow7 |

---

*Next: [01-architecture.md](./01-architecture.md)*
