# Auth7 — Plan 03: Session & Token Management

> **GitHub Group Issue**: [#4](https://github.com/ihsansolusi/auth7/issues/4)  
> **Status**: 📋 Planned  
> **Total Issues**: 8 (including integration test)

---

## Goal

Implement JWT signing (RS256), session lifecycle management, token revocation, dan hard IP binding.

---

## Issues

| # | GitHub Issue | Title | Est. Points | Specs |
|---|--------------|-------|-------------|-------|
| 3.1 | [#30](https://github.com/ihsansolusi/auth7/issues/30) | JWT key management (RSA key pair generation, rotation) | 5 | 03-oauth2-oidc.md, 05-session-token.md |
| 3.2 | [#31](https://github.com/ihsansolusi/auth7/issues/31) | JWT signing & verification service | 5 | 03-oauth2-oidc.md, 05-session-token.md |
| 3.3 | [#32](https://github.com/ihsansolusi/auth7/issues/32) | Session store (Redis + PostgreSQL) | 3 | 05-session-token.md, 08-data-model.md |
| 3.4 | [#33](https://github.com/ihsansolusi/auth7/issues/33) | Refresh token implementation (family, reuse detection) | 5 | 05-session-token.md, 03-oauth2-oidc.md |
| 3.5 | [#34](https://github.com/ihsansolusi/auth7/issues/34) | Token revocation & blacklist | 3 | 05-session-token.md |
| 3.6 | [#35](https://github.com/ihsansolusi/auth7/issues/35) | Session timeout & hard IP binding | 3 | 05-session-token.md |
| 3.7 | [#36](https://github.com/ihsansolusi/auth7/issues/36) | JWKS endpoint implementation | 2 | 03-oauth2-oidc.md |
| 3.8 | [#37](https://github.com/ihsansolusi/auth7/issues/37) | Plan 03 Integration Test | 3 | All Plan 03 specs |

---

## Key Deliverables

- [ ] RSA key pair generation and rotation
- [ ] JWT signing service (RS256)
- [ ] Session storage (Redis primary, PostgreSQL backup)
- [ ] Refresh token dengan family tracking
- [ ] Token reuse detection
- [ ] Token blacklist mechanism
- [ ] Session timeout (idle + absolute)
- [ ] Hard IP binding enforcement
- [ ] JWKS endpoint untuk public key distribution

---

## Dependencies

- [Plan 01: Foundation](./PLAN-01.md)
- [Plan 02: Identity Core](./PLAN-02.md)

---

## Next Plans

- [Plan 04: OAuth2 Server](./PLAN-04.md)
- [Plan 06: MFA](./PLAN-06.md)
