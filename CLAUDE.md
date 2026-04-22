# CLAUDE.md — Auth7

Panduan konteks untuk Claude AI saat bekerja di repo `ihsansolusi/auth7`.

## Identitas Proyek

- **Nama**: Auth7 — Identity & Access Management Platform untuk Core7
- **Fase saat ini**: **Brainstorming** (specs phase)
- **Repo**: `github.com/ihsansolusi/auth7`
- **Working dir**: `supported-apps/auth7/` (di dalam core7-devroot devroot)
- **Parent project**: [Core7 v2026.1 — Project #8](https://github.com/orgs/ihsansolusi/projects/8)

## Tujuan

Auth7 adalah auth system self-built untuk ekosistem Core7, menggantikan rencana adopsi
Ory Stack / Zitadel. Terinspirasi dari:
- **Ory Kratos** → identity flows
- **Ory Hydra** → OAuth2/OIDC server
- **Ory Keto** → authorization engine
- **Ory Oathkeeper** → request auth middleware
- **Zitadel** → event sourcing, audit trail, multi-tenancy

Repo ini murni untuk **service Go**. UI terpisah di repo `ihsansolusi/auth7-ui`.

## Fase & Status

| Fase | Status | Catatan |
|------|--------|---------|
| Brainstorming (Specs) | **✅ DONE** | 11 spec files (00-10), recreated 2026-04-22 |
| Open Questions Review | **✅ DONE** | Semua 30 questions dijawab |
| Specs Review (1-per-1) | **🔲 TODO** | Review bersama user |
| Implementation Plans | **🔲 TODO** | Setelah specs disetujui |
| Plan 01 — Foundation | **🔲 TODO** | Repo, CI/CD, Docker, DB migrations |

## Struktur Folder (Planned)

```
auth7/
├── CLAUDE.md                  ← file ini
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── identity/              # User lifecycle, credentials, profiles
│   ├── oauth2/                # OAuth2/OIDC server, token issuance
│   ├── authz/                 # RBAC + ABAC engine (Casbin + OPA)
│   ├── session/               # Session management (Redis-backed)
│   ├── mfa/                   # TOTP + Email OTP
│   ├── audit/                 # Immutable audit trail
│   ├── tenant/                # Multi-tenancy (org + branch)
│   ├── admin/                 # Admin API handlers
│   ├── gateway/               # Request auth middleware
│   ├── api/                   # HTTP handlers (Gin), middleware
│   ├── service/               # Business logic
│   ├── store/                 # Database access (pgx + sqlc), Redis
│   └── domain/                # Entities, value objects, interfaces
├── docs/
│   ├── specs/                 ← Spesifikasi sistem (11 files)
│   │   ├── README.md          ← Index + prinsip desain
│   │   ├── 00-overview.md     ← Visi, scope v1.0, stack
│   │   ├── 01-architecture.md ← Clean arch, API surface, deployment
│   │   ├── 02-identity.md     ← User lifecycle, login/register/recovery
│   │   ├── 03-oauth2-oidc.md  ← OAuth2 server, PKCE, token design, JWKS
│   │   ├── 04-authorization.md← RBAC+ABAC, Casbin, permission API
│   │   ├── 05-session-token.md← Session lifecycle, token revocation
│   │   ├── 06-mfa.md          ← TOTP, backup codes, MFA policy
│   │   ├── 07-admin-api.md    ← Admin CRUD, audit log, session mgmt
│   │   ├── 08-data-model.md   ← PostgreSQL schema + Redis key patterns
│   │   ├── 09-integration.md  ← lib7-auth-go, gRPC, M2M, per-service
│   │   └── 10-security.md     ← Crypto, OWASP, OJK compliance
│   └── OPEN-QUESTIONS.md      ← Semua keputusan brainstorming
├── migrations/                # SQL migrations (golang-migrate)
├── proto/                     # gRPC protobuf definitions
├── configs/                   # Config templates
├── scripts/                   # Dev/ops scripts
└── tests/                     # Integration tests
```

## Stack

| Layer | Teknologi |
|-------|-----------|
| Language | Go 1.22+ |
| HTTP Framework | Gin |
| gRPC | net/grpc |
| Database | PostgreSQL 16 (pgx + sqlc) |
| Cache / Session | Redis (wajib v1.0) |
| Password Hashing | Argon2id (`golang.org/x/crypto`) |
| JWT | `golang-jwt/jwt/v5` (RS256) |
| TOTP | `pquerna/otp` |
| Authorization | Casbin (RBAC) + OPA (ABAC/Rego) |
| Observability | zerolog + OpenTelemetry + Prometheus |
| Migrations | golang-migrate |

## Keputusan Desain v1.0

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

## Konvensi Coding

Konsisten dengan `service7-template`:

- Setiap Go method: `const op = "package.Type.Method"`
- Setiap Go method: `logging.WithTrace(ctx, logger)` + `ctx, span := tracer.Start(ctx, op)`
- Semua error di-wrap: `fmt.Errorf("%s: %w", op, err)`
- Store method multi-tenant: wajib terima `orgID uuid.UUID` eksplisit
- Tidak ada secret di config file — hanya `"${ENV_VAR}"`

## Arsitektur

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

## Aturan Penting

- **Jangan modifikasi `docs/specs/`** tanpa persetujuan user — specs adalah sumber kebenaran desain
- **Jangan mulai implementasi** sebelum specs disetujui dan plans dibuat
- Semua perubahan harus di-commit dan di-push ke `ihsansolusi/auth7`

## Referensi

| Path | Keterangan |
|------|------------|
| `docs/specs/README.md` | Index semua spec files |
| `docs/specs/00-overview.md` | Visi dan scope v1.0 |
| `docs/OPEN-QUESTIONS.md` | Semua keputusan brainstorming (30 questions) |
| `../service7-template/` | Template arsitektur Go yang diadopsi |
| `../service7-template/specs/` | Referensi pola clean arch |
| `../notif7/` | Sibling project (unified notification) |
| `../auth7-ui/` | UI repo (Next.js, terpisah) |
