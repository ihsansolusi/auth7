# Auth7 — Plan 02: Identity Management Core

> **GitHub Group Issue**: [#3](https://github.com/ihsansolusi/auth7/issues/3)  
> **Status**: 📋 Planned  
> **Total Issues**: 8 (including integration test)

---

## Goal

Implement user lifecycle management, credentials, password hashing dengan Argon2id.

---

## Issues

| # | GitHub Issue | Title | Est. Points | Specs |
|---|--------------|-------|-------------|-------|
| 2.1 | [#22](https://github.com/ihsansolusi/auth7/issues/22) | User entity & domain logic | 3 | 02-identity.md |
| 2.2 | [#23](https://github.com/ihsansolusi/auth7/issues/23) | Argon2id password hashing implementation | 3 | 10-security.md |
| 2.3 | [#24](https://github.com/ihsansolusi/auth7/issues/24) | User store (CRUD operations with sqlc) | 5 | 02-identity.md, 08-data-model.md |
| 2.4 | [#25](https://github.com/ihsansolusi/auth7/issues/25) | User credentials store | 3 | 02-identity.md, 08-data-model.md |
| 2.5 | [#26](https://github.com/ihsansolusi/auth7/issues/26) | User service (registration, password change) | 5 | 02-identity.md |
| 2.6 | [#27](https://github.com/ihsansolusi/auth7/issues/27) | Email verification flow | 3 | 02-identity.md |
| 2.7 | [#28](https://github.com/ihsansolusi/auth7/issues/28) | Password recovery flow | 3 | 02-identity.md |
| 2.8 | [#29](https://github.com/ihsansolusi/auth7/issues/29) | Plan 02 Integration Test | 3 | All Plan 02 specs |

---

## Key Deliverables

- [ ] User domain entity dengan validasi
- [ ] Argon2id password hashing service
- [ ] User store dengan sqlc queries
- [ ] Credential store untuk password history
- [ ] User service: register, update, change password
- [ ] Email verification flow (token-based)
- [ ] Password recovery flow (secure token)

---

## Dependencies

- [Plan 01: Foundation](./PLAN-01.md)

---

## Next Plans

- [Plan 03: Session & Token](./PLAN-03.md)
- [Plan 05: Multi-Branch](./PLAN-05.md)
