# Auth7 — Plans Overview

> Dokumen ini adalah index dari semua implementation plans auth7.
> 
> **Fase saat ini**: ✅ Implementation Planning Complete
> **Root Issue**: [#1 — Auth7 v1.0 Implementation](https://github.com/ihsansolusi/auth7/issues/1)
> **Project Board**: [Core7 v2026.1](https://github.com/orgs/ihsansolusi/projects/8)

---

## Status

| # | Plan | Topik | Status | GitHub Group | Issues |
|---|------|-------|--------|--------------|--------|
| 01 | [Foundation](./PLAN-01.md) | Repo setup, CI/CD, Docker, DB migrations | 📋 Planned | [#2](https://github.com/ihsansolusi/auth7/issues/2) | #12-21 (10) |
| 02 | [Identity Core](./PLAN-02.md) | User CRUD, credential management | 📋 Planned | [#3](https://github.com/ihsansolusi/auth7/issues/3) | #22-29 (8) |
| 03 | [Session & Token](./PLAN-03.md) | JWT signing, session lifecycle, revocation | 📋 Planned | [#4](https://github.com/ihsansolusi/auth7/issues/4) | #30-37 (8) |
| 04 | [OAuth2 Server](./PLAN-04.md) | Authorization code, client credentials | 📋 Planned | [#5](https://github.com/ihsansolusi/auth7/issues/5) | #38-44 (7) |
| 05 | [Multi-Branch](./PLAN-05.md) | Branch types, hierarchy, user-branch assignments | 📋 Planned | [#6](https://github.com/ihsansolusi/auth7/issues/6) | #45-50 (6) |
| 06 | [MFA](./PLAN-06.md) | TOTP enrollment, verification, backup codes | 📋 Planned | [#7](https://github.com/ihsansolusi/auth7/issues/7) | #51-57 (7) |
| 07 | [Authorization](./PLAN-07.md) | RBAC, Casbin integration, field masking | 📋 Planned | [#8](https://github.com/ihsansolusi/auth7/issues/8) | #58-66 (9) |
| 08 | [Admin API](./PLAN-08.md) | User mgmt, role mgmt, branch mgmt | 📋 Planned | [#9](https://github.com/ihsansolusi/auth7/issues/9) | #67-75 (9) |
| 09 | [Security](./PLAN-09.md) | Rate limiting, brute force protection | 📋 Planned | [#10](https://github.com/ihsansolusi/auth7/issues/10) | #76-81 (6) |
| 10 | [Integration](./PLAN-10.md) | lib7-auth-go, gRPC service, notif7 integration | 📋 Planned | [#11](https://github.com/ihsansolusi/auth7/issues/11) | #82-91 (10) |
| 11 | [Service Migration](./PLAN-11.md) | Migrate existing services to auth7 security | 📋 Planned | [#92](https://github.com/ihsansolusi/auth7/issues/92) | #93-104 (12) |
| 12 | **[NATS Integration](./PLAN-12.md)** | **Event streaming, service communication** | 📋 Planned | [#105](https://github.com/ihsansolusi/auth7/issues/105) | #106-112 (7) |

**Total Issues**: 111 issues (1 root + 12 groups + 98 individual)

---

## Hierarchy Structure

```
ihsansolusi/core7-devroot#35 (105 - Supported Apps)
│
└── ihsansolusi/auth7#1 (ROOT: Auth7 v1.0)
    │
    ├── ihsansolusi/auth7#2 (Plan 01 — Foundation)
    │   └── 10 child issues (#12-21)
    │
    ├── ihsansolusi/auth7#3 (Plan 02 — Identity)
    │   └── 8 child issues (#22-29)
    │
    ├── ihsansolusi/auth7#4 (Plan 03 — Session)
    │   └── 8 child issues (#30-37)
    │
    ├── ihsansolusi/auth7#5 (Plan 04 — OAuth2)
    │   └── 7 child issues (#38-44)
    │
    ├── ihsansolusi/auth7#6 (Plan 05 — Multi-Branch)
    │   └── 6 child issues (#45-50)
    │
    ├── ihsansolusi/auth7#7 (Plan 06 — MFA)
    │   └── 7 child issues (#51-57)
    │
    ├── ihsansolusi/auth7#8 (Plan 07 — Authorization)
    │   └── 9 child issues (#58-66)
    │
    ├── ihsansolusi/auth7#9 (Plan 08 — Admin API)
    │   └── 9 child issues (#67-75)
    │
    ├── ihsansolusi/auth7#10 (Plan 09 — Security)
    │   └── 6 child issues (#76-81)
    │
    ├── ihsansolusi/auth7#11 (Plan 10 — Integration)
    │   └── 10 child issues (#82-91)
    │
    ├── ihsansolusi/auth7#12 (Plan 11 — Service Migration)
    │   └── 12 child issues (#93-104)
    │
    ├── ihsansolusi/auth7#105 (Plan 12 — NATS Integration) ✅ NEW
    │   └── 7 child issues (#106-112)
    │
```

---

## Dependency Graph

```
Plan 01 (Foundation)
  └── Plan 02 (Identity Core)
        ├── Plan 03 (Session & Token)
        │     ├── Plan 04 (OAuth2)
        │     └── Plan 06 (MFA)
        └── Plan 05 (Multi-Branch)
              └── Plan 07 (Authorization)
                    └── Plan 08 (Admin API)
                          ├── Plan 09 (Security)
                          ├── Plan 10 (Integration)
                          ├── Plan 11 (Service Migration) ← After Plan 10
                          └── Plan 12 (NATS Integration) ← Parallel dengan Plan 10
```

**Execution Order**: 01 → 02 → (03, 05) → (04, 06, 07) → 08 → (09, 10, **12**) → **11**

---

## Specs Reference

All plans are derived from:
- [00-overview.md](../specs/00-overview.md)
- [01-architecture.md](../specs/01-architecture.md)
- [02-identity.md](../specs/02-identity.md)
- [03-oauth2-oidc.md](../specs/03-oauth2-oidc.md)
- [04-authorization.md](../specs/04-authorization.md)
- [05-session-token.md](../specs/05-session-token.md)
- [06-mfa.md](../specs/06-mfa.md)
- [07-admin-api.md](../specs/07-admin-api.md)
- [08-data-model.md](../specs/08-data-model.md)
- [09-integration.md](../specs/09-integration.md)
- [10-security.md](../specs/10-security.md)

## Additional References

- [Internal Service Security Analysis](../../docs/security/INTERNAL-SERVICE-SECURITY.md) — Security requirements for Plan 11
- [Hybrid Messaging Model](../../docs/infra/HYBRID-MESSAGING-MODEL.md) — NATS integration requirements for Plan 12

---

*Dibuat: 2026-04-22 | Updated: 2026-04-27 (Plan 12 — NATS Integration Added)*
