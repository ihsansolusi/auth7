# Auth7 — Plan 08: Admin API

> **GitHub Group Issue**: [#9](https://github.com/ihsansolusi/auth7/issues/9)  
> **Status**: 📋 Planned  
> **Total Issues**: 9 (including integration test)

---

## Goal

Implement Admin API untuk user management, role management, branch management, dan audit log.

---

## Issues

| # | GitHub Issue | Title | Est. Points | Specs |
|---|--------------|-------|-------------|-------|
| 8.1 | [#67](https://github.com/ihsansolusi/auth7/issues/67) | Admin API middleware & RBAC | 3 | 07-admin-api.md, 04-authorization.md |
| 8.2 | [#68](https://github.com/ihsansolusi/auth7/issues/68) | User management endpoints (CRUD, lock, suspend) | 5 | 07-admin-api.md |
| 8.3 | [#69](https://github.com/ihsansolusi/auth7/issues/69) | Branch type & branch management endpoints | 3 | 07-admin-api.md |
| 8.4 | [#70](https://github.com/ihsansolusi/auth7/issues/70) | Role & permission management endpoints | 3 | 07-admin-api.md |
| 8.5 | [#71](https://github.com/ihsansolusi/auth7/issues/71) | User-branch & user-role assignment endpoints | 3 | 07-admin-api.md |
| 8.6 | [#72](https://github.com/ihsansolusi/auth7/issues/72) | OAuth2 client management endpoints | 3 | 07-admin-api.md, 03-oauth2-oidc.md |
| 8.7 | [#73](https://github.com/ihsansolusi/auth7/issues/73) | Audit log query endpoint | 3 | 07-admin-api.md |
| 8.8 | [#74](https://github.com/ihsansolusi/auth7/issues/74) | Audit log implementation (all events) | 5 | 07-admin-api.md, 08-data-model.md |
| 8.9 | [#75](https://github.com/ihsansolusi/auth7/issues/75) | Plan 08 Integration Test | 3 | All Plan 08 specs |

---

## Key Deliverables

- [ ] Admin API middleware dengan RBAC
- [ ] User management: CRUD, lock, unlock, suspend, delete
- [ ] Branch type CRUD endpoints
- [ ] Branch CRUD dan hierarchy management
- [ ] Role CRUD endpoints
- [ ] Permission assignment ke roles
- [ ] User-branch assignment endpoints
- [ ] User-role assignment per branch
- [ ] OAuth2 client management
- [ ] Audit log query dengan filter
- [ ] Immutable audit log untuk semua events

---

## Dependencies

- [Plan 01-07](./PLAN-OVERVIEW.md)

---

## Next Plans

- [Plan 09: Security](./PLAN-09.md)
- [Plan 10: Integration](./PLAN-10.md)
