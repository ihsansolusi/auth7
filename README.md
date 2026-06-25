# Auth7

Identity & Access Management (IAM) platform untuk ekosistem **Core7**. Satu Go service yang menjadi
source of truth untuk identity, credential, session, role/permission, dan keputusan otorisasi —
menggantikan rencana adopsi Ory Stack / Zitadel. Headless: semua via REST + gRPC; UI auth-facing
terpisah di repo `ihsansolusi/auth7-ui`, admin UI utama di `bos7-enterprise`.

Analogi domain: `identity` (≈ Ory Kratos), `oauth2` (≈ Hydra), `authz` (≈ Keto + Casbin),
`middleware` (≈ Oathkeeper), `audit` + `tenant` (≈ Zitadel event sourcing & organizations).

## Isi Repo

```
cmd/            # Entry point + subcommands: start, migrate, seed
internal/       # Implementasi clean architecture (api → service → store → domain)
migrations/     # SQL migrations (golang-migrate) — schema IAM
migrations-seed/# Seed profile-based: demo (users/branches) & prod (org/oauth2)
proto/          # gRPC protobuf (auth/v1)
configs/        # config.example.yaml, casbin_model.conf, nats-dev.conf
pkg/            # Reusable packages (config, dll)
scripts/        # Ops: sync-branches-from-enterprise, seed-data, docker-entrypoint
tests/          # Integration tests
docs/           # Spesifikasi teknis + roadmap + migration guide
```

## Stack

- **Language:** Go 1.22+ · **HTTP:** Gin · **gRPC:** google.golang.org/grpc
- **Database:** PostgreSQL 16 (pgx + sqlc) · **Cache/Session:** Redis (wajib)
- **Event streaming:** NATS (token revocation, session, security events)
- **Auth crypto:** Argon2id (password), golang-jwt RS256 (JWT), pquerna/otp (TOTP)
- **Authorization:** Casbin (RBAC) + OPA/Rego (ABAC)
- **Observability:** zerolog + OpenTelemetry + Prometheus
- **Migration:** golang-migrate

## Arsitektur

Clean architecture dengan layer boundaries ketat (konsisten `service7-template`):

```
cmd/ → internal/api/ → internal/service/ → internal/store/ → internal/domain/
```

Domain di `internal/service/`: `oauth2`, `authz`, `session`, `jwt`, `mfa`, `password`,
`branch` + `branchsync`, `audit`, `security`, `opacache`. Messaging di `internal/messaging/nats`,
integrasi outbound di `internal/integration/notif7` + `internal/mailer`.

### Boundary (dikunci di Plan 13)

- **auth7** — owner identity, credential, session, role, permission, authz decision.
- **policy7** — owner policy/parameter bisnis; dikonsumsi auth7 sebagai input ABAC.
- **core7-service-enterprise** — owner branch master operasional, employee, department, position, office. Branch di auth7 adalah **projeksi**, bukan master.
- **bos7-enterprise** — admin UI utama; **auth7-ui** — auth-facing flow (login/recovery), bukan admin console.

## API Surface

| Grup | Endpoint (ringkas) |
|---|---|
| Auth (`/v1/auth`) | register, login, logout, me, profile, change-password[-initial], forgot-password, reset-password, mfa/{setup,verify,disable} |
| MFA (`/v1/mfa`) | enroll/verify TOTP, enroll/verify email OTP, backup-codes, step-up, config |
| OAuth2/OIDC (`/v1/oauth`) | authorize, authorize-with-session, token, introspect, userinfo, register (DCR) |
| Discovery | `/.well-known/jwks.json`, `/.well-known/openid-configuration` |
| Branch | `/v1/auth/branches`, `/v1/auth/switch-branch`, admin branch & branch-type CRUD |
| Admin (`/admin/v1`) | dashboard stats, apps, users, roles, branches/default-roles, sessions, facade, emergency (revoke/force-logout/key-rotation) |
| Internal (`/internal/v1`) | user-context, + WF-CRUD callbacks (`wf-create/update/delete`) untuk users, roles, oauth2-clients, branch-default-roles |
| Health | `/health/live`, `/health/ready` |

## Status

| Fase | Status |
|---|---|
| Spesifikasi (`docs/specs/`) | ✅ Selesai |
| Implementasi v1.0 (Plan 01–12 + auth-gap) | ✅ **COMPLETE** |
| Integrasi: audit7 (JetStream), notif7 (email), NATS, WF-CRUD callbacks | ✅ Selesai |
| Plan 13 — Enterprise boundary cutover (lintas-modul) | ⏳ Residual — lihat [`docs/ROADMAP.md`](docs/ROADMAP.md) |
| Future v1.1/v2.0 (Passkeys, SAML, impersonation, dual-approval, …) | 📋 Roadmap |

## Dokumentasi

| File | Isi |
|---|---|
| [`docs/specs/README.md`](docs/specs/README.md) | Index spesifikasi teknis + prinsip desain + keputusan v1.0 |
| [`docs/specs/00-overview.md`](docs/specs/00-overview.md) … [`10-security.md`](docs/specs/10-security.md) | Spec detail per domain (lihat index) |
| [`docs/migration-guide.md`](docs/migration-guide.md) | Panduan migration & seed (DEF → SQL, profile demo/prod) |
| [`docs/ROADMAP.md`](docs/ROADMAP.md) | Fitur belum terimplementasi: future scope + residual Plan 13 |

## Quick Start (lokal)

Port default **8083**. Prasyarat: Core7 Unified Infra (Postgres/Redis/NATS) running.

```bash
# 1. Infra (dari root devroot)
docker compose -f docker-compose.infra.yml up -d

# 2. Config
cp .env.example .env        # sesuaikan DATABASE_URL / REDIS_URL / JWT_SECRET

# 3. Migration + seed
make migrate-up
make seed-up SEED_PROFILE=demo

# 4. Jalankan service
SERVER_PORT=8083 ./auth7-bin start    # atau: go run ./cmd/server start
```

Lihat `make help` untuk daftar lengkap perintah migration & seed.

---

*Bagian dari [core7-devroot](../../) · Project Board: [Core7 v2026.1 (#8)](https://github.com/orgs/ihsansolusi/projects/8)*
