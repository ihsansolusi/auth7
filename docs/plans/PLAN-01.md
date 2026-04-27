# Auth7 — Plan 01: Foundation & Infrastructure

> **GitHub Group Issue**: [#2](https://github.com/ihsansolusi/auth7/issues/2)  
> **Status**: 📋 Planned  
> **Total Issues**: 10 (including integration test)

---

## Goal

Setup repository structure, CI/CD, Docker environment, database migrations, and base infrastructure untuk Auth7 service.

---

## Issues

| # | GitHub Issue | Title | Est. Points | Specs |
|---|--------------|-------|-------------|-------|
| 1.1 | [#12](https://github.com/ihsansolusi/auth7/issues/12) | Setup repository structure (cmd/, internal/, configs/) | 3 | 01-architecture.md |
| 1.2 | [#13](https://github.com/ihsansolusi/auth7/issues/13) | Setup CI/CD pipeline (GitHub Actions: test, lint, build) | 3 | 01-architecture.md |
| 1.3 | [#14](https://github.com/ihsansolusi/auth7/issues/14) | Setup Docker & docker-compose (dev environment) | 2 | 01-architecture.md |
| 1.4 | [#15](https://github.com/ihsansolusi/auth7/issues/15) | Database migrations: organizations, branch_types, branches, branch_hierarchies | 5 | 08-data-model.md |
| 1.5 | [#16](https://github.com/ihsansolusi/auth7/issues/16) | Database migrations: users, user_credentials, user_attributes | 3 | 08-data-model.md |
| 1.6 | [#17](https://github.com/ihsansolusi/auth7/issues/17) | Redis connection & key pattern implementation | 2 | 08-data-model.md (Redis section) |
| 1.7 | [#18](https://github.com/ihsansolusi/auth7/issues/18) | Configuration management (env-based, no secrets in files) | 2 | 01-architecture.md |
| 1.8 | [#19](https://github.com/ihsansolusi/auth7/issues/19) | Logging & observability setup (zerolog, OpenTelemetry) | 3 | 01-architecture.md |
| 1.9 | [#20](https://github.com/ihsansolusi/auth7/issues/20) | Base domain errors & interfaces | 2 | 02-identity.md (domain section) |
| 1.10 | [#21](https://github.com/ihsansolusi/auth7/issues/21) | Plan 01 Integration Test | 3 | All Plan 01 specs |

---

## Key Deliverables

- [ ] Working Go repository with clean architecture structure
- [ ] CI/CD pipeline running on GitHub Actions
- [ ] Docker Compose for local development
- [ ] PostgreSQL migrations for core tables (organizations, branch_types, branches, users)
- [ ] Redis connection with proper key patterns
- [ ] Config management without secrets in files
- [ ] Structured logging and OpenTelemetry setup

---

## Dependencies

- None (foundation plan)

---

## Next Plan

- [Plan 02: Identity Core](./PLAN-02.md)
