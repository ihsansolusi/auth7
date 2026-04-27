# Auth7 — Plan 10: Integration & lib7-auth-go

> **GitHub Group Issue**: [#11](https://github.com/ihsansolusi/auth7/issues/11)  
> **Status**: 📋 Planned  
> **Total Issues**: 10 (including E2E test)

---

## Goal

Implement gRPC proto definitions, auth7 service gRPC server, notif7 integration, dan lib7-auth-go SDK.

---

## Issues

| # | GitHub Issue | Title | Est. Points | Specs |
|---|--------------|-------|-------------|-------|
| 10.1 | [#82](https://github.com/ihsansolusi/auth7/issues/82) | gRPC protobuf definitions | 2 | 09-integration.md |
| 10.2 | [#83](https://github.com/ihsansolusi/auth7/issues/83) | gRPC server implementation | 3 | 09-integration.md |
| 10.3 | [#84](https://github.com/ihsansolusi/auth7/issues/84) | notif7 client integration (security alerts) | 3 | 09-integration.md (notif7 section) |
| 10.4 | [#85](https://github.com/ihsansolusi/auth7/issues/85) | Setup lib7-auth-go repository | 2 | 09-integration.md |
| 10.5 | [#86](https://github.com/ihsansolusi/auth7/issues/86) | lib7-auth-go: JWT middleware (Gin) | 3 | 09-integration.md |
| 10.6 | [#87](https://github.com/ihsansolusi/auth7/issues/87) | lib7-auth-go: gRPC interceptor | 3 | 09-integration.md |
| 10.7 | [#88](https://github.com/ihsansolusi/auth7/issues/88) | lib7-auth-go: Permission check client | 3 | 09-integration.md |
| 10.8 | [#89](https://github.com/ihsansolusi/auth7/issues/89) | lib7-auth-go: M2M token manager | 3 | 09-integration.md |
| 10.9 | [#90](https://github.com/ihsansolusi/auth7/issues/90) | lib7-auth-go: Testing mocks | 2 | 09-integration.md |
| 10.10 | [#91](https://github.com/ihsansolusi/auth7/issues/91) | Plan 10 E2E Test | 5 | All Plan 10 specs |

---

## Key Deliverables

- [ ] Protobuf definitions (AuthService, VerifyToken, CheckPermission)
- [ ] gRPC server implementation
- [ ] notif7 client untuk security alerts:
  - auth.login_new_device
  - auth.account_locked
  - auth.mfa_reset
  - auth.password_changed
- [ ] lib7-auth-go repository setup
- [ ] lib7-auth-go: Gin JWT middleware
- [ ] lib7-auth-go: gRPC interceptor
- [ ] lib7-auth-go: Permission check client
- [ ] lib7-auth-go: M2M token manager dengan caching
- [ ] lib7-auth-go: Testing mocks

---

## Dependencies

- [Plan 01-09](./PLAN-OVERVIEW.md)

---

## Next Steps

- Integration dengan workflow7, bos7-portal, notif7
- Production deployment preparation
