# Auth7 demo seed

Demo subset for `auth7` — organisation, branch_types, branches, roles,
permissions, users (with credentials), user_roles, user_branch_assignments.

## Cross-service ID consistency

Auth7 shares UUIDs with `core7-service-enterprise` and `policy7`/`audit7` via
deterministic UUID5 derived from business keys.  This file's
`deterministic_uuid()` MUST stay in sync with:

- `appdist/services/core7-service-enterprise/seed/demo/transform.py`
- `supported-apps/policy7/seed/demo/transform.py`
- `supported-apps/audit7/seed/demo/transform.py`

```python
NS = uuid.UUID("11111111-1111-1111-1111-111111111111")
deterministic_uuid("branch", "010")     # → same in enterprise + auth7 + policy7
deterministic_uuid("organization", "BJBS")
deterministic_uuid("user", "EMP-0001")
deterministic_uuid("role", "TELLER")
```

Prefix table (DO NOT change without coordinating across all four services):

| Prefix | Business key | Used in |
|---|---|---|
| `organization` | org code (e.g. `BJBS`) | auth7.organizations.id, policy7.org_id, audit7 (via metadata) |
| `branch` | `branch_code` (kode_cabang) | auth7.branches.id, enterprise.branches.id, audit7.branch_id |
| `branch_type` | `branch_type` enum value (e.g. `MAIN_BRANCH`) | auth7.branch_types.id |
| `position` | `kode_jabatan` | enterprise.positions.id |
| `department` | `kode_departemen` | enterprise.departments.id |
| `office` | `kode_kantor` | enterprise.offices.id |
| `user` | `nomor_karyawan` (employee_number) | auth7.users.id, enterprise.employees.id, audit7.actor_id |
| `role` | role code (e.g. `TELLER`, `SUPERVISOR`) | auth7.roles.id, policy7.applies_to_id when applies_to='role' |
| `permission` | permission code (e.g. `transfer.create`) | auth7.permissions.id |

## Prasyarat

- `psql` di PATH (install `postgresql-client` kalau belum)
- `python3` (standar Python, no external deps)
- Unified infra running (`core7-postgres` dengan `ibankdb_medium` tersedia)
- **Run enterprise extract dulu** — auth7's `seed_002_branches.sql` reuses
  enterprise's `csv/cabang.csv` (UUID consistency)

## Konfigurasi

```bash
cp .env.seed.example .env.seed
# Default sudah benar untuk core7-postgres lokal.
```

## Alur

```bash
# 0. Pastikan enterprise sudah extract dulu:
(cd ../../../../appdist/services/core7-service-enterprise && make seed-demo-extract)

# 1. Extract dari ibankdb_medium (PG) ke csv/
make seed-demo-extract     # bash extract.sh

# 2. Transform CSV → SQL (idempotent ON CONFLICT)
make seed-demo-transform

# 3. Apply ke Postgres
DATABASE_URL=postgresql://auth7:auth7secret@localhost:5432/auth7?sslmode=disable make seed-demo-apply

# Atau extract+transform+apply sekaligus
DATABASE_URL=... make seed-demo
```

## File yang dihasilkan

```
seed/demo/
├── seed_001_organization.sql   # organizations, branch_types, permissions catalog
├── seed_002_branches.sql       # branches (UUIDs IDENTIK dengan enterprise)
├── seed_003_roles.sql          # roles, role_permissions, branch_default_roles
└── seed_004_users.sql          # users, user_credentials, user_roles, user_branch_assignments
```

## Catatan branch_types

Tabel `branch_types` (auth7-internal) memetakan enum English (`MAIN_BRANCH`,
`SUB_BRANCH`, dst.) ke metadata UI (label, short_code, level).  Seed-nya
deterministik supaya `branches.branch_type_id` di FK stabil antar re-run.

## Catatan password

`SEED_DEFAULT_PASSWORD_HASH` default = argon2id dari `Password123!`.
Setiap user di-seed dengan hash yang sama.  Untuk demo only — JANGAN dipakai
di shared/staging environment tanpa rotate password.

## Catatan implementasi (PENDING)

`extract_oracle.py`, `transform.py`, dan SQL output saat ini berupa **skeleton**.
Untuk implementasi penuh, akses Oracle ibankdb_medium diperlukan untuk
verifikasi nama kolom (`IBENT.USER`, `IBENT.PERAN`, `IBENT.LISTPERANUSER`).
