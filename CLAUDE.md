# CLAUDE.md — Auth7

Panduan konteks untuk Claude AI saat bekerja di repo `ihsansolusi/auth7`.

## Identitas Proyek

- **Nama**: Auth7 — Identity & Access Management Platform untuk Core7
- **Fase saat ini**: 🔄 **IMPLEMENTATION** — Plans 01-02 complete, Plan 03 ready to start
- **Planning Status**: ✅ **COMPLETE** — All specs reviewed, 12 plans, 111 GitHub issues
- **Implementation Start**: ✅ COMPLETE — Plans 01 & 02 finished (2026-04-27)
- **Root Issue**: [#1 — Auth7 v1.0](https://github.com/ihsansolusi/auth7/issues/1)
- **Total GitHub Issues**: 111 issues (1 root + 12 plan groups + 98 implementation issues)
- **Project Board**: [Core7 v2026.1](https://github.com/orgs/ihsansolusi/projects/8)
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
| Specs Review (1-per-1) | **✅ DONE** | Review 1-per-1 selesai (2026-04-24) |
| Implementation Plans | **✅ DONE** | 10 plans created, 91 GitHub issues |
| GitHub Issues Created | **✅ DONE** | All issues linked to Project #8 |
| Plan 01 — Foundation | **✅ DONE** | Issues #12-21, completed 2026-04-27 |
| Plan 02 — Identity Core | **✅ DONE** | Issues #22-29, completed 2026-04-27 |

### Plan Status

| Plan | Status | Group Issue | Individual Issues |
|------|--------|-------------|-------------------|
| [Plan 01](./docs/plans/PLAN-01.md) | ✅ DONE | [#2](https://github.com/ihsansolusi/auth7/issues/2) | #12-21 (10) |
| [Plan 02](./docs/plans/PLAN-02.md) | ✅ DONE | [#3](https://github.com/ihsansolusi/auth7/issues/3) | #22-29 (8) |
| Plan 03 — Session & Token Management | ✅ DONE | [#4](https://github.com/ihsansolusi/auth7/issues/4) | #30-37 (8) |
| [Plan 04](./docs/plans/PLAN-04.md) | 📋 Planned | [#5](https://github.com/ihsansolusi/auth7/issues/5) | #38-44 (7) |
| [Plan 05](./docs/plans/PLAN-05.md) | 📋 Planned | [#6](https://github.com/ihsansolusi/auth7/issues/6) | #45-50 (6) |
| [Plan 06](./docs/plans/PLAN-06.md) | 📋 Planned | [#7](https://github.com/ihsansolusi/auth7/issues/7) | #51-57 (7) |
| [Plan 07](./docs/plans/PLAN-07.md) | 📋 Planned | [#8](https://github.com/ihsansolusi/auth7/issues/8) | #58-66 (9) |
| [Plan 08](./docs/plans/PLAN-08.md) | 📋 Planned | [#9](https://github.com/ihsansolusi/auth7/issues/9) | #67-75 (9) |
| [Plan 09](./docs/plans/PLAN-09.md) | 📋 Planned | [#10](https://github.com/ihsansolusi/auth7/issues/10) | #76-81 (6) |
| [Plan 10](./docs/plans/PLAN-10.md) | 📋 Planned | [#11](https://github.com/ihsansolusi/auth7/issues/11) | #82-91 (10) |
| [Plan 11](./docs/plans/PLAN-11.md) | 📋 Planned | [#92](https://github.com/ihsansolusi/auth7/issues/92) | #93-104 (12) |
| **[Plan 12](./docs/plans/PLAN-12.md)** | 📋 **NEW** | [#105](https://github.com/ihsansolusi/auth7/issues/105) | **#106-112 (7)** |

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
| Event Streaming | NATS (v1.0) — Hybrid Messaging Model |
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
| Policy sync | Redis pub/sub (cache), NATS (events) |
| Event streaming | NATS — token revocation, session events, security alerts |
| MFA | TOTP + Email OTP (setiap login, no trusted device) |
| Max concurrent sessions | Configurable per org |
| IP binding | Hard (force logout jika IP berubah) |
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
| `docs/plans/PLAN-OVERVIEW.md` | Index semua implementation plans |
| `docs/plans/PLAN-01.md` s/d `PLAN-10.md` | Detail per plan dengan GitHub issue links |
| `../service7-template/` | Template arsitektur Go yang diadopsi |
| `../service7-template/specs/` | Referensi pola clean arch |
| `../notif7/` | Sibling project (unified notification) |
| `../auth7-ui/` | UI repo (Next.js, terpisah) |
| `../policy7/` | Sibling project (business policy service) |

---

## Standard Workflow Pattern (Skill Documentation)

Pattern ini digunakan untuk auth7 dan akan menjadi template untuk auth7-ui dan policy7.

### Phase 1: Brainstorming & Specs

1. **Create specs** in `docs/specs/`:
   - `00-overview.md` — Vision, scope, stack
   - `01-architecture.md` — Clean arch, components
   - `02-*.md` — Feature-specific specs
   - `OPEN-QUESTIONS.md` — All design decisions

2. **Review specs 1-per-1** dengan user:
   - Read spec file
   - Discuss dan catat decisions
   - Update spec jika perlu
   - Tandai ✅ setelah disetujui

### Phase 2: Implementation Planning

1. **Draft plan structure**:
   ```
   Plan 01 — Foundation (repo, CI/CD, Docker)
   Plan 02 — Core Feature A
   Plan 03 — Core Feature B
   ...
   ```

2. **Get user approval** untuk plan structure

3. **Create GitHub Issues** dengan hierarki:
   ```
   core7-devroot#35 (105 - Supported Apps) [sudah ada]
   └── auth7#1 (ROOT: Project v1.0) [create + addSubIssue]
       └── auth7#2 (Plan 01 — Foundation) [create + addSubIssue]
           └── auth7#12-21 (Individual issues) [create + addSubIssue + addProjectV2ItemById]
   ```

### Phase 3: Issue Creation Script

Gunakan Python script dengan GitHub GraphQL API:

```python
# Key mutations:
# 1. createIssue — buat issue
# 2. addSubIssue — link parent-child
# 3. addProjectV2ItemById — add ke project board
# 4. updateProjectV2ItemFieldValue — set status
```

### Required IDs

- **Project ID**: `PVT_kwDOA0OdHM4BPTJK` (Core7 v2026.1)
- **Parent Issue**: `I_kwDORNh3Qc7rKfd2` (core7-devroot#35)
- **Repository IDs**: Query via GraphQL
- **User ID**: `U_kgDOABKAUg` (galihaprilian)

### Issue Template

```markdown
## Description
{title}

## Specs
- {spec_file}

## Parent
- Child dari: #{parent_issue} ({parent_title})

## Estimation
{points} points

## Assignee
@{username}

## Tasks
- [ ] Implement
- [ ] Unit tests
- [ ] Integration tests
- [ ] Documentation
```

### Next Projects Using This Pattern

- **auth7-ui**: Next.js UI for auth7
- **policy7**: Business policy & parameter service

---

*Last updated: 2026-04-24 — Implementation Planning Complete*
