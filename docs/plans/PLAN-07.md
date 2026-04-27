# Auth7 — Plan 07: Authorization (RBAC + ABAC)

> **GitHub Group Issue**: [#8](https://github.com/ihsansolusi/auth7/issues/8)  
> **Status**: 📋 Planned  
> **Total Issues**: 9 (including integration test)

---

## Goal

Implement authorization dengan RBAC (Casbin), ABAC (JSON rules), dan 4-layer permission model dengan field-level masking.

---

## Issues

| # | GitHub Issue | Title | Est. Points | Specs |
|---|--------------|-------|-------------|-------|
| 7.1 | [#58](https://github.com/ihsansolusi/auth7/issues/58) | Role & permission domain models | 3 | 04-authorization.md |
| 7.2 | [#59](https://github.com/ihsansolusi/auth7/issues/59) | Role & permission stores | 3 | 04-authorization.md, 08-data-model.md |
| 7.3 | [#60](https://github.com/ihsansolusi/auth7/issues/60) | User roles assignment (per branch) | 3 | 04-authorization.md, 07-admin-api.md |
| 7.4 | [#61](https://github.com/ihsansolusi/auth7/issues/61) | Casbin adapter & policy storage | 5 | 04-authorization.md |
| 7.5 | [#62](https://github.com/ihsansolusi/auth7/issues/62) | RBAC enforcement middleware | 3 | 04-authorization.md |
| 7.6 | [#63](https://github.com/ihsansolusi/auth7/issues/63) | ABAC policies (JSON rules) | 3 | 04-authorization.md |
| 7.7 | [#64](https://github.com/ihsansolusi/auth7/issues/64) | Permission field masking (4-layer auth) | 5 | 04-authorization.md |
| 7.8 | [#65](https://github.com/ihsansolusi/auth7/issues/65) | gRPC AuthCheck service | 3 | 04-authorization.md, 09-integration.md |
| 7.9 | [#66](https://github.com/ihsansolusi/auth7/issues/66) | Plan 07 Integration Test | 3 | All Plan 07 specs |

---

## Key Deliverables

- [ ] Role dan permission domain models
- [ ] Role & permission stores (PostgreSQL)
- [ ] User roles assignment per branch
- [ ] Casbin adapter dengan PostgreSQL storage
- [ ] RBAC enforcement middleware
- [ ] ABAC policy evaluation (JSON rules)
- [ ] 4-layer authorization model:
  1. Page/Menu access
  2. Data access permission
  3. Branch scope (own/assigned/all)
  4. Field-level masking
- [ ] gRPC AuthCheck service untuk real-time permission check

---

## Dependencies

- [Plan 01: Foundation](./PLAN-01.md)
- [Plan 02: Identity Core](./PLAN-02.md)
- [Plan 05: Multi-Branch](./PLAN-05.md)

---

## Next Plans

- [Plan 08: Admin API](./PLAN-08.md)
