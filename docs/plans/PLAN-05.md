# Auth7 — Plan 05: Multi-Branch & Branch Management

> **GitHub Group Issue**: [#6](https://github.com/ihsansolusi/auth7/issues/6)  
> **Status**: 📋 Planned  
> **Total Issues**: 6 (including integration test)

---

## Goal

Implement configurable branch types, branch hierarchy, user-branch assignments, dan branch switching.

---

## Issues

| # | GitHub Issue | Title | Est. Points | Specs |
|---|--------------|-------|-------------|-------|
| 5.1 | [#45](https://github.com/ihsansolusi/auth7/issues/45) | Branch type CRUD (configurable per org) | 3 | 07-admin-api.md, 08-data-model.md |
| 5.2 | [#46](https://github.com/ihsansolusi/auth7/issues/46) | Branch CRUD with branch_type_id | 3 | 07-admin-api.md, 08-data-model.md |
| 5.3 | [#47](https://github.com/ihsansolusi/auth7/issues/47) | Branch hierarchy management (parent-child) | 3 | 08-data-model.md |
| 5.4 | [#48](https://github.com/ihsansolusi/auth7/issues/48) | User-branch assignments (multi-branch access) | 5 | 02-identity.md, 08-data-model.md |
| 5.5 | [#49](https://github.com/ihsansolusi/auth7/issues/49) | Branch switching endpoint (/auth/switch-branch) | 3 | 01-architecture.md, 02-identity.md |
| 5.6 | [#50](https://github.com/ihsansolusi/auth7/issues/50) | Plan 05 Integration Test | 3 | All Plan 05 specs |

---

## Key Deliverables

- [ ] Branch types CRUD (configurable per organization)
- [ ] Branch CRUD dengan FK ke branch_types
- [ ] Branch hierarchy dengan parent-child relationships
- [ ] User-branch assignments (1 user bisa punya multiple branches)
- [ ] Primary branch flag per user
- [ ] Branch switching endpoint dengan re-authentication

---

## Dependencies

- [Plan 01: Foundation](./PLAN-01.md)
- [Plan 02: Identity Core](./PLAN-02.md)

---

## Next Plans

- [Plan 07: Authorization](./PLAN-07.md) (role per branch)
