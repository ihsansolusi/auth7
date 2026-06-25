# Auth7 — Technical Specifications

> **Auth7** adalah sistem autentikasi & otorisasi terpadu yang dibangun khusus untuk ekosistem Core7.
> Desain terinspirasi dari [Ory Stack](https://www.ory.sh/) (Kratos + Hydra + Keto + Oathkeeper)
> dan [Zitadel](https://zitadel.com/), disesuaikan dengan kebutuhan banking Indonesia.

Folder ini adalah **dokumentasi teknis detail** auth7. Semua spec di sini sudah ✅ **terimplementasi (v1.0)** —
gunakan sebagai referensi desain & kontrak; sumber kebenaran perilaku tetap kode di `internal/`.
Untuk fitur yang **belum** terimplementasi lihat [`../ROADMAP.md`](../ROADMAP.md).

---

## Index

| # | File | Topik | Domain di `internal/` |
|---|------|-------|------------------------|
| 00 | [00-overview.md](./00-overview.md) | Visi, filosofi, scope v1.0, stack | — |
| 01 | [01-architecture.md](./01-architecture.md) | Clean architecture, API surface, deployment | `api/`, `service/`, `store/`, `domain/` |
| 02 | [02-identity.md](./02-identity.md) | Identity & user lifecycle (≈ Kratos) | `service/password`, `store/postgres` |
| 03 | [03-oauth2-oidc.md](./03-oauth2-oidc.md) | OAuth2 + OIDC server (≈ Hydra) | `service/oauth2`, `api/rest/oauth2.go` |
| 04 | [04-authorization.md](./04-authorization.md) | RBAC + ABAC engine (≈ Keto + Casbin) | `service/authz`, `service/opacache` |
| 05 | [05-session-token.md](./05-session-token.md) | Session & token lifecycle, revocation | `service/session`, `service/jwt` |
| 06 | [06-mfa.md](./06-mfa.md) | Multi-factor auth (TOTP + Email OTP) | `service/mfa`, `api/rest/mfa.go` |
| 07 | [07-admin-api.md](./07-admin-api.md) | Admin & management API + facade | `api/rest/admin`, `admin_routes.go` |
| 08 | [08-data-model.md](./08-data-model.md) | PostgreSQL schema + Redis key patterns | `migrations/`, `store/` |
| 09 | [09-integration.md](./09-integration.md) | Integrasi Core7 (gRPC, M2M, notif7, NATS) | `integration/`, `messaging/nats`, `proto/` |
| 10 | [10-security.md](./10-security.md) | Security posture & compliance banking | `api/middleware`, `service/security` |

---

## Prinsip Desain

1. **API-First, Headless** — tidak ada UI baked-in; semua via REST/gRPC (UI di repo `auth7-ui`).
2. **Single Binary** — satu service monolitik terstruktur, clean architecture seperti `service7-template`.
3. **Banking-Grade** — comply regulasi OJK/BI; argon2id, audit append-only 5 tahun.
4. **Multi-Tenant** — isolasi per-org/per-branch; branch = projeksi dari `core7-service-enterprise`.
5. **Zero External Auth Dependency** — tidak bergantung cloud auth provider.

## Boundary (dikunci di Plan 13)

- `auth7` = source of truth untuk **identity, credential, session, role, permission, authz decision**.
- `policy7` = source of truth untuk policy/parameter bisnis yang dikonsumsi auth7 sebagai input ABAC.
- `core7-service-enterprise` = source of truth untuk **branch master operasional, employee, department, position, office**.
- `bos7-enterprise` = admin UI utama untuk operasi admin auth7; `auth7-ui` = auth-facing flow (login/recovery), bukan admin console utama.

## Keputusan Desain (v1.0)

| Keputusan | Value | | Keputusan | Value |
|---|---|---|---|---|
| Access token TTL | 15 menit | | OIDC Discovery | Ya |
| Refresh token TTL | 8 jam | | Audit retention | 5 tahun |
| Redis | Wajib | | Admin rate limit | 10 req/s |
| Token format | JWT + Opaque (per client) | | DCR (RFC 7591) | v1.0 |
| Casbin adapter | Custom pgx | | Bulk import CSV | v1.0 |
| ABAC | Hybrid JSON Rules + OPA Rego | | HSM | → v2.0 |
| MFA | TOTP + Email OTP (tiap login) | | Consent screen | → v2.0 |
| Max concurrent sessions | Configurable per org | | User impersonation | → v1.1 |
| Event streaming | NATS (token revocation, session, security) | | Dual approval | → v2.0 (via workflow7) |
