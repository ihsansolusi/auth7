# Auth7 — Plans Overview

> Dokumen ini adalah index dari semua implementation plans auth7.
> Plans dibuat setelah specs selesai di-review dan disetujui.
>
> **Fase saat ini**: Brainstorming (Specs)
> **Fase berikutnya**: Implementation Planning

---

## Status

| # | Plan | Topik | Status | GitHub Issue |
|---|------|-------|--------|--------------|
| 01 | Foundation | Repo setup, CI/CD, Docker, DB migrations | 🔲 TODO | — |
| 02 | Identity Core | User CRUD, credential management | 🔲 TODO | — |
| 03 | Auth Flows | Login, logout, register, recovery | 🔲 TODO | — |
| 04 | OAuth2 Server | Authorization code, client credentials | 🔲 TODO | — |
| 05 | OIDC | ID token, userinfo, discovery, JWKS | 🔲 TODO | — |
| 06 | Authorization | RBAC, Casbin integration, permission API | 🔲 TODO | — |
| 07 | MFA | TOTP enrollment, verification, backup codes | 🔲 TODO | — |
| 08 | Admin API | User mgmt, role mgmt, client mgmt | 🔲 TODO | — |
| 09 | Multi-tenancy | Org, branch, tenant-scoped policies | 🔲 TODO | — |
| 10 | Audit & Security | Audit log, rate limiting, brute force | 🔲 TODO | — |
| 11 | Integration | lib7-auth-go, gRPC service, workflow7 integration | 🔲 TODO | — |
| 12 | UI Integration | auth7-ui (Next.js) OAuth2 flow | 🔲 TODO | — |

---

## Prasyarat Sebelum Mulai Plan

- [x] Specs v1.0 selesai di-recreate (semua 11 files)
- [x] Open questions dijawab (30/30)
- [ ] Specs v1.0 di-review dan disetujui (1-per-1)
- [ ] GitHub Issues dibuat di Project Board Core7 v2026.1
- [ ] Stack final dikonfirmasi (Go version, dependencies)

---

## Dependency antar Plans

```
Plan 01 (Foundation)
  └── Plan 02 (Identity Core)
        ├── Plan 03 (Auth Flows)
        │     ├── Plan 04 (OAuth2)
        │     │     └── Plan 05 (OIDC)
        │     └── Plan 07 (MFA)
        └── Plan 06 (Authorization)
              └── Plan 10 (Admin API)

Plan 09 (Multi-tenancy) → parallel dengan Plan 02-08
Plan 10 (Audit & Security) → parallel, finish before production
Plan 11 (Integration) → setelah Plan 04, 06 selesai
Plan 12 (UI Integration) → setelah Plan 03, 04 selesai
```

---

*Dibuat: 2026-04-22 | Akan diupdate setelah specs disetujui*
