# Auth7 — Plan 13 W5 Cross-Module Audit (#128)

> **Status**: Completed  
> **Date**: 2026-05-12  
> **Umbrella**: `core7-devroot#200`  
> **Wave**: `core7-devroot#204`  
> **Stream Epic**: `auth7#114`  
> **Issue**: `auth7#128`  
> **Boundary Anchor**: `docs/architecture/auth7-policy7-enterprise-boundary.md`

---

## 1. Conformance Checklist (PASS/FAIL)

### 1.1 IAM Ownership Conformance

| Check | Result | Evidence |
|---|---|---|
| user lifecycle tetap auth7-owned | PASS | `internal/api/rest/admin_routes.go`, `internal/api/rest/admin/user.go` |
| credential lifecycle tetap auth7-owned | PASS | `internal/domain/entity.go`, `internal/service/password/*`, `internal/store/postgres/repository.go` |
| session lifecycle tetap auth7-owned | PASS | `internal/service/session/service.go`, `internal/service/session/store.go` |
| role/permission lifecycle tetap auth7-owned | PASS | `internal/api/rest/admin/role.go`, `internal/service/authz/*`, `docs/specs/04-authorization.md` |
| tidak ada fallback ke legacy authority runtime | PASS | `docs/plans/PLAN-13-ENTERPRISE-BOUNDARY.md` (Compatibility Artifact Register Wave 4 guardrail) + compatibility endpoints marked deprecated |

### 1.2 Integration Conformance

| Check | Result | Evidence |
|---|---|---|
| IAM admin contract untuk consumer berada di auth7 admin API | PASS | `internal/api/rest/admin_routes.go` (`/admin/v1`) + `internal/api/rest/admin/facade.go` (`/admin/v1/facade/*`) |
| compatibility path diberi marker transisi dan steady-state target | PASS | `internal/api/rest/admin/facade.go` (`Deprecation`, `Sunset`, `steady_state_target`) + `docs/specs/04-authorization.md` |
| ABAC parameterized decision tetap consume policy7 | PASS | `docs/specs/04-authorization.md` bagian ownership matrix dan flow (`Auth7 <-> Policy7`) |
| auth7 tidak membentuk policy business tables sebagai source of truth | PASS | `docs/specs/08-data-model.md` (policy consumption note); tidak ada migration table policy bisnis lintas domain |

---

## 2. Residual Compatibility Artifacts (W4 carry-over)

| Artifact | Status | Runtime Authority Allowed | Notes |
|---|---|---|---|
| `legacy_user_id` mapping | `facade` | no | dipakai untuk trace lineage transisi |
| `enterprise.peran` | `compatibility-only` | no | translasi ke auth7 roles |
| `enterprise.listperanuser` | `compatibility-only` | no | translasi ke auth7 user_roles |
| `enterprise.rolemenulist` | `compatibility-only` | no | translasi ke `menu:{menu_key}:access` |
| `enterprise.usermenulist` | `retire-target` | no | override user-level harus ditutup bertahap |
| legacy function/action map | `compatibility-only` | no | translasi ke `{resource}:{action}` |

---

## 3. Cutover Readiness Recommendation

**Recommendation**: `NOT READY`

Rationale:
1. Audit conformance internal auth7 sudah PASS, tetapi cutover lintas modul belum selesai.
2. Wiring final di `bos7-enterprise` ke seluruh `/admin/v1/facade/*` belum tervalidasi dari sisi stream auth7.
3. Parity matrix untuk translasi role/menu/function belum ditandatangani lintas stream.
4. Sunset execution plan compatibility path belum final di level coordinator.

---

## 4. Blockers Table

| Blocker ID | Owner | Blocker | Next Action | Target Date |
|---|---|---|---|---|
| `W5-S1-B01` | `S5` (`bos7-enterprise`) | facade IAM path belum fully wired/verified ke auth7 endpoints | S5 selesaikan route mapping + integration verification ke `/admin/v1/facade/*` | 2026-05-16 |
| `W5-S1-B02` | `S3` (`core7-service-enterprise`) | freeze final reference mapping role/menu/function source belum dinyatakan selesai | S3 publish freeze note + version tag untuk mapping source | 2026-05-19 |
| `W5-S1-B03` | `S5` + coordinator | parity test matrix translasi legacy -> permission auth7 belum disepakati | coordinator lock test matrix & acceptance thresholds lintas stream | 2026-05-21 |
| `W5-S1-B04` | coordinator (`core7-devroot`) | sunset execution date compatibility endpoints belum ditetapkan | tetapkan tanggal sunset + rollback rule + owner per milestone | 2026-05-23 |

---

## 5. Evidence Links

- Wave 3 implementation commit: `7657f6a`
- Wave 4 compatibility cleanup commit: `3bed447`
- W4 compatibility and blocker register: `docs/plans/PLAN-13-ENTERPRISE-BOUNDARY.md`
- W4 deprecation semantics: `docs/specs/04-authorization.md`
- Runtime facade compatibility endpoints: `internal/api/rest/admin/facade.go`
