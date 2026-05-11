# Auth7 — Plan 13: Enterprise Boundary Alignment

> **Status**: Draft  
> **Umbrella**: `core7-devroot#200`  
> **Wave Coordinator (W1)**: `core7-devroot#202`  
> **Wave Coordinator (W2)**: `core7-devroot#203`  
> **Stream Epic**: `auth7#114`  
> **Depends on**: Plans 05, 07, 08, 10, 11  
> **Reference**:
> - [`docs/architecture/auth7-policy7-enterprise-boundary.md`](../../../../docs/architecture/auth7-policy7-enterprise-boundary.md)
> - [`docs/architecture/auth7-policy7-enterprise-change-control.md`](../../../../docs/architecture/auth7-policy7-enterprise-change-control.md)
> - [`docs/plans/integration/PLAN-12-WAVE-1-BACKEND-AUTHORITY-LOCK.md`](../../../../docs/plans/integration/PLAN-12-WAVE-1-BACKEND-AUTHORITY-LOCK.md)
> - [`docs/plans/integration/PLAN-12-enterprise-boundary-alignment.md`](../../../../docs/plans/integration/PLAN-12-enterprise-boundary-alignment.md)

---

## Goal

Menyelaraskan auth7 dengan boundary baru Core7 sehingga auth7 secara eksplisit menjadi owner IAM saja, sekaligus siap dikonsumsi sebagai backend admin oleh `bos7-enterprise`.

## Wave 1 Scope

Plan ini menurunkan `Plan 12 Wave 1` untuk stream `auth7`. Fokus wave ini hanya pada
spec/plan alignment backend authority, belum pada wiring lintas stream atau kontrak integrasi final.

Ownership matrix `Wave 1` untuk stream ini:

| Dimension | Owner |
|---|---|
| UI Owner | `bos7-enterprise` untuk admin, `auth7-ui` untuk auth-facing flow |
| API Owner | `auth7` untuk IAM admin/runtime APIs |
| Data Owner | `auth7` untuk identity/credential/session/role/permission, `core7-service-enterprise` untuk branch master dan employee master yang direferensikan |

---

## Scope

- Branch projection sync dari `core7-service-enterprise`
- Employee attribute/reference model di auth7
- Konsumsi admin API auth7 oleh `bos7-enterprise`
- Compatibility mapping dari legacy enterprise user/role/menu data
- Penegasan bahwa auth7 tidak memperkenalkan business-policy tables baru

## Explicitly Allowed

- Klarifikasi ownership IAM runtime di spec dan plan auth7
- Klarifikasi semantics branch projection auth7
- Klarifikasi employee/department/position reference semantics
- Klarifikasi admin API ownership auth7 dan policy7 consumption boundary

## Explicitly Disallowed

- Mengambil ownership branch master operasional
- Mengambil ownership employee/department/position/office master
- Mengambil ownership policy categories atau parameter truth
- Menambah schema/table yang menjadikan auth7 source of truth policy bisnis

## Wave 1 Issue Set

| Issue | Fokus | Target Artefak |
|---|---|---|
| `auth7#115` | Lock IAM ownership statement | overview, architecture, plan |
| `auth7#116` | Lock branch projection semantics | overview, data model, admin API, integration |
| `auth7#117` | Lock employee reference semantics | identity, data model, integration |
| `auth7#118` | Lock admin API ownership and policy7 consumption statement | admin API, integration, data model, plan |

## Dependencies and Blockers

Dependency lintas stream yang harus dicatat untuk wave berikutnya:
- source contract projection branch dari `core7-service-enterprise` ke `auth7`
- source contract reference employee dari `core7-service-enterprise` ke `auth7`

Status `Wave 1`:
- belum menjadi blocker untuk spec lock di repo ini
- harus dibawa ke coordinator sebagai dependency sebelum masuk wiring/cutover wave berikutnya

## Wave 2 Scope

Plan ini menurunkan `Plan 12 Wave 2` untuk stream `auth7`. Fokus wave ini hanya pada
contract dan mapping definition sebagai consumer-side readiness, belum masuk wiring runtime (Wave 3).

Deliverable inti `Wave 2`:
- definisi branch projection contract consumer-side
- definisi employee reference contract consumer-side
- definisi baseline translasi legacy role/menu/function ke permission auth7

## Wave 2 Issue Set

| Issue | Fokus | Target Artefak |
|---|---|---|
| `auth7#119` | Define branch projection contract consumer side | integration, data model, plan |
| `auth7#120` | Define employee reference contract consumer side | identity, integration, data model, plan |
| `auth7#121` | Define legacy role/menu/function -> permission baseline | authorization, admin API/integration, plan |

## Contract Owner/Consumer Matrix (Wave 2)

| Contract Area | Contract Owner | Consumer | Auth7 Position |
|---|---|---|---|
| Branch master projection feed | `core7-service-enterprise` | `auth7` | consumer |
| Employee reference feed | `core7-service-enterprise` | `auth7` | consumer |
| Policy parameter context for ABAC | `policy7` | `auth7` | consumer |
| Legacy role/menu/function translation baseline | `auth7` | `bos7-enterprise` (admin facade), migration tools | baseline owner (mapping definition only) |

## Dependencies and Blockers (Wave 2)

Dependency lintas stream yang harus tersedia sebelum masuk `Wave 3` wiring:
- `core7-service-enterprise` menyediakan source contract branch projection yang stabil (identity key, status semantics, hierarchy linkage)
- `core7-service-enterprise` menyediakan source contract employee reference yang stabil (employee identity key + org/branch linkage)
- `policy7` menyediakan kontrak parameter context yang dibutuhkan ABAC auth7 tanpa memindahkan policy truth ke auth schema

Status `Wave 2`:
- auth7 mendefinisikan consumer contract dan mapping baseline di level spec
- unresolved lintas stream harus dicatat di coordinator `core7-devroot#203` sebagai dependency, bukan diresolusikan unilateral oleh auth7

---

## Work Items

### 13.1 Branch Projection Alignment
- Definisikan contract projection `branch master -> auth7 branch`
- Lock field minimal projection: `org_id`, `code`, `name`, `status`, `parent`, `type`
- Definisikan aturan sinkronisasi dan drift handling

### 13.2 Employee Reference Model
- Standarkan penggunaan `user_attributes` atau mapping table untuk `employee_id`, `department_code`, `position_code`, `branch_code`
- Pastikan auth7 tidak menjadi owner employee master

### 13.3 Admin API Consumer Model
- Dokumentasikan `bos7-enterprise` sebagai primary admin UI consumer
- Pertahankan `auth7-ui` admin surface hanya sebagai fallback/internal
- Pastikan semua endpoint admin auth7 tetap authoritative

### 13.4 Legacy Compatibility
- Map `legacy_user_id -> auth7 user_id`
- Map `legacy_role/menu/function -> auth7 role/permission`
- Tentukan artefak compatibility yang temporary vs long-lived

### 13.5 Policy Boundary
- Dokumentasikan bahwa auth7 mengkonsumsi `policy7` hanya untuk ABAC input
- Tidak membuat source-of-truth policy tables baru di auth7

---

## Acceptance Criteria

- Spec auth7 tidak lagi mengimplikasikan ownership atas employee master atau branch operasional
- Branch di auth7 terdokumentasi sebagai auth projection
- `bos7-enterprise` muncul sebagai admin UI utama di spec dan plan auth7
- `auth7-ui` tidak lagi diposisikan sebagai admin console utama
- Tidak ada proposal tabel policy bisnis baru di auth7

## Out of Scope

- Wiring `bos7-enterprise -> auth7` lintas repo
- Contract field final untuk branch projection sync
- Contract field final untuk employee reference sync
- Implementasi cache/event untuk konsumsi `policy7`
- Implementasi runtime adapter/scheduler/sync worker untuk kontrak branch/employee
- Implementasi migrasi data legacy live untuk role/menu/function translation

---

## Output

- Update spec auth7 terkait overview, data model, admin API, dan integration
- Backlog implementasi siap diturunkan menjadi issue repo auth7
