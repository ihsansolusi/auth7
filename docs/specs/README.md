# Auth7 — Specifications Index

> **Auth7** adalah sistem autentikasi dan otorisasi terpadu yang dibangun khusus untuk ekosistem Core7.
> Dirancang dengan inspirasi dari [Ory Stack](https://www.ory.sh/) (Kratos + Hydra + Keto + Oathkeeper)
> dan [Zitadel](https://zitadel.com/), namun disesuaikan dengan kebutuhan banking Indonesia.

---

## Status Brainstorming

| # | File | Topik | Status |
|---|------|-------|--------|
| 00 | [00-overview.md](./00-overview.md) | Visi, filosofi, komponen utama | ✅ Recreated |
| 01 | [01-architecture.md](./01-architecture.md) | Arsitektur sistem, API surface, deployment | ✅ Recreated |
| 02 | [02-identity.md](./02-identity.md) | Identity & user management (≈ Kratos) | ✅ Recreated |
| 03 | [03-oauth2-oidc.md](./03-oauth2-oidc.md) | OAuth2 & OpenID Connect server (≈ Hydra) | ✅ Recreated |
| 04 | [04-authorization.md](./04-authorization.md) | Authorization engine RBAC + ABAC (≈ Keto) | ✅ Recreated |
| 05 | [05-session-token.md](./05-session-token.md) | Session management & token lifecycle | ✅ Recreated |
| 06 | [06-mfa.md](./06-mfa.md) | Multi-factor authentication | ✅ Recreated |
| 07 | [07-admin-api.md](./07-admin-api.md) | Admin & management API | ✅ Recreated |
| 08 | [08-data-model.md](./08-data-model.md) | PostgreSQL schema + Redis key patterns | ✅ Recreated |
| 09 | [09-integration.md](./09-integration.md) | Integrasi ke Core7 services | ✅ Recreated |
| 10 | [10-security.md](./10-security.md) | Security posture & compliance banking | ✅ Recreated |

---

## Prinsip Desain

1. **API-First, Headless** — tidak ada UI baked-in; semua via REST/gRPC
2. **Single Binary** — satu service monolitik terstruktur (bukan microservice terpisah)
3. **Banking-Grade** — comply dengan regulasi OJK, BI, dan standar keamanan perbankan Indonesia
4. **Multi-Tenant** — mendukung isolasi per-branch / per-tenant
5. **Zero External Dependencies** — tidak bergantung pada cloud auth provider

---

## Referensi Inspirasi

| Sistem | Komponen yang diadopsi |
|--------|------------------------|
| Ory Kratos | Identity flows (login, register, recovery, verification) |
| Ory Hydra | OAuth2/OIDC server, token management |
| Ory Keto | Zanzibar-inspired permission model |
| Ory Oathkeeper | Request authentication middleware (diimplementasikan sebagai Go middleware) |
| Zitadel | Event sourcing, audit trail, multi-tenancy, action hooks |
| Casbin | RBAC enforcement (sudah dipakai di service7-template) |

---

## Keputusan Desain (v1.0)

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

*Recreated: 2026-04-22 | Fase: Brainstorming → Ready for Review*
