# Auth7 — Plan 04: OAuth2/OIDC Server

> **GitHub Group Issue**: [#5](https://github.com/ihsansolusi/auth7/issues/5)  
> **Status**: 📋 Planned  
> **Total Issues**: 7 (including integration test)

---

## Goal

Implement OAuth2 server dengan PKCE, OIDC Discovery, dan M2M client credentials.

---

## Issues

| # | GitHub Issue | Title | Est. Points | Specs |
|---|--------------|-------|-------------|-------|
| 4.1 | [#38](https://github.com/ihsansolusi/auth7/issues/38) | OAuth2 clients store & management | 3 | 03-oauth2-oidc.md, 08-data-model.md |
| 4.2 | [#39](https://github.com/ihsansolusi/auth7/issues/39) | Authorization code flow with PKCE | 5 | 03-oauth2-oidc.md |
| 4.3 | [#40](https://github.com/ihsansolusi/auth7/issues/40) | Token endpoint (authorization_code, refresh_token grants) | 5 | 03-oauth2-oidc.md |
| 4.4 | [#41](https://github.com/ihsansolusi/auth7/issues/41) | OIDC Discovery & UserInfo endpoints | 3 | 03-oauth2-oidc.md |
| 4.5 | [#42](https://github.com/ihsansolusi/auth7/issues/42) | Client credentials grant (M2M) | 3 | 03-oauth2-oidc.md, 09-integration.md |
| 4.6 | [#43](https://github.com/ihsansolusi/auth7/issues/43) | Token introspection endpoint | 2 | 03-oauth2-oidc.md |
| 4.7 | [#44](https://github.com/ihsansolusi/auth7/issues/44) | Plan 04 Integration Test | 3 | All Plan 04 specs |

---

## Key Deliverables

- [ ] OAuth2 client registration dan management
- [ ] Authorization endpoint dengan PKCE support
- [ ] Token endpoint (authorization_code, refresh_token)
- [ ] OIDC Discovery endpoint (.well-known/openid-configuration)
- [ ] UserInfo endpoint
- [ ] Client credentials grant untuk M2M
- [ ] Token introspection (RFC 7662)

---

## Dependencies

- [Plan 01: Foundation](./PLAN-01.md)
- [Plan 02: Identity Core](./PLAN-02.md)
- [Plan 03: Session & Token](./PLAN-03.md)

---

## Next Plans

- [Plan 10: Integration](./PLAN-10.md) (M2M token manager)
