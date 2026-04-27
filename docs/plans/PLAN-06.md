# Auth7 — Plan 06: MFA (TOTP + Email OTP)

> **GitHub Group Issue**: [#7](https://github.com/ihsansolusi/auth7/issues/7)  
> **Status**: 📋 Planned  
> **Total Issues**: 7 (including integration test)

---

## Goal

Implement MFA dengan TOTP (QR code enrollment), Email OTP, dan backup codes.

---

## Issues

| # | GitHub Issue | Title | Est. Points | Specs |
|---|--------------|-------|-------------|-------|
| 6.1 | [#51](https://github.com/ihsansolusi/auth7/issues/51) | MFA config store (totp_secret encrypted) | 3 | 06-mfa.md, 08-data-model.md |
| 6.2 | [#52](https://github.com/ihsansolusi/auth7/issues/52) | TOTP enrollment & QR code generation | 3 | 06-mfa.md |
| 6.3 | [#53](https://github.com/ihsansolusi/auth7/issues/53) | TOTP verification with replay prevention | 3 | 06-mfa.md |
| 6.4 | [#54](https://github.com/ihsansolusi/auth7/issues/54) | Email OTP generation & verification | 3 | 06-mfa.md |
| 6.5 | [#55](https://github.com/ihsansolusi/auth7/issues/55) | Backup codes generation & usage | 3 | 06-mfa.md |
| 6.6 | [#56](https://github.com/ihsansolusi/auth7/issues/56) | MFA login flow (step-up authentication) | 5 | 06-mfa.md, 02-identity.md |
| 6.7 | [#57](https://github.com/ihsansolusi/auth7/issues/57) | Plan 06 Integration Test | 3 | All Plan 06 specs |

---

## Key Deliverables

- [ ] MFA config store dengan encrypted TOTP secret (AES-256-GCM)
- [ ] TOTP enrollment dengan QR code
- [ ] TOTP verification dengan replay prevention (Redis)
- [ ] Email OTP generation (6 digit, 10 menit expiry)
- [ ] Backup codes generation (10 codes, hashed storage)
- [ ] MFA login flow (step-up setelah password valid)
- [ ] MFA method priority: user > role > org

---

## Dependencies

- [Plan 01: Foundation](./PLAN-01.md)
- [Plan 02: Identity Core](./PLAN-02.md)
- [Plan 03: Session & Token](./PLAN-03.md)

---

## Next Plans

- None directly, but MFA affects all auth flows
