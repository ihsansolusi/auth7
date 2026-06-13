#!/usr/bin/env python3
"""
gen_ibankdb_employees.py — Generate demo seed dari ibankdb_medium untuk auth7.

Menghasilkan migrations-seed/demo/000006_seed_ibankdb_employees.up.sql berisi:
  1. INSERT ibent.peran (81 roles) → auth7.roles
  2. INSERT employees (anonymized) → auth7.users + credentials
  3. INSERT user_roles menggunakan ibent role UUID yang sebenarnya
  4. INSERT user_branch_assignments berdasarkan kode_cabang

Jalankan dari supported-apps/auth7/:
    python3 migrations-seed/scripts/gen_ibankdb_employees.py
"""

from __future__ import annotations
import csv
import hashlib
import io
import os
import subprocess
import sys
import uuid
from pathlib import Path

# ─── Config ────────────────────────────────────────────────────────────────────
SOURCE_DSN = os.getenv("IBANKDB_DSN", "postgresql://ibank:solusi@localhost:5432/ibankdb_medium")
SCHEMA     = "ibent"
ORG_ID     = "00000000-0000-0000-0000-000000000001"

DEFAULT_PASSWORD_HASH = (
    "$argon2id$v=19$m=65536,t=3,p=4"
    "$c2VlZF9zYWx0X2ZpeGVkIQ"
    "$N+pMwLuOqjb62N8jRGpZTng1AJkGETP4yvjfe6CWPRI"
)

# UUID namespace — konsisten dengan transform.py original
NS = uuid.UUID("11111111-1111-1111-1111-111111111111")

# Branch codes yang ada di 000002_seed_branches.up.sql
SEEDED_BRANCHES = {
    "000", "001", "002", "003", "004", "005", "006", "007", "008",
    "501", "502", "503", "504", "505", "506", "507", "508", "509", "510",
    "511", "512", "513", "514", "515", "516", "517", "518", "519", "520",
    "521", "522", "523", "524", "525", "526", "527", "528", "529", "530",
    "531", "532", "533", "534", "535", "536", "537", "538", "539", "540",
    "541", "542", "543", "544", "545", "546", "547", "548", "549", "552",
    "554", "555",
    "701", "702", "703", "704", "705", "706", "707", "708", "709",
}

# ─── Anonymization pool ────────────────────────────────────────────────────────
FIRST_NAMES = [
    "Andi", "Budi", "Citra", "Dewi", "Eko", "Fitri", "Galih", "Hendra",
    "Indah", "Joko", "Kartika", "Lutfi", "Maya", "Nanda", "Oscar", "Putri",
    "Reza", "Sari", "Teddy", "Ulfa", "Vino", "Wulan", "Yanuar", "Zaki",
    "Ahmad", "Bella", "Candra", "Dimas", "Ela", "Ferry", "Gita", "Hadi",
    "Ivan", "Jasmine", "Krisna", "Lina", "Miftah", "Nina", "Omar", "Panji",
    "Rahmat", "Sandi", "Tika", "Umar", "Vandi", "Wahyu", "Yasmin", "Zahra",
    "Bagas", "Cantika", "Dian", "Erik", "Farida", "Gilang", "Hasna", "Irfan",
    "Kirana", "Lukman", "Maman", "Nabil", "Oktavia", "Pandu", "Rini", "Surya",
    "Taufik", "Usman", "Wawan", "Yogi", "Arief", "Bunga", "Chandra", "Dedi",
    "Edy", "Faisal", "Guntur", "Hesti", "Ilham", "Jaka", "Karim", "Laras",
    "Mulia", "Nisa", "Opik", "Rangga", "Sheila", "Tri", "Udin", "Vika",
    "Widi", "Yana", "Zulfa", "Andhika", "Bambang", "Cindi", "Denny", "Erwin",
]

LAST_NAMES = [
    "Santoso", "Wijaya", "Kusuma", "Pratama", "Hidayat", "Rahayu", "Putra",
    "Gunawan", "Setiawan", "Nugroho", "Suryana", "Andika", "Firmansyah",
    "Hakim", "Ismail", "Jaya", "Kurniawan", "Lestari", "Muharam", "Nuraeni",
    "Oktavia", "Permana", "Ramdhani", "Sudirman", "Triyono", "Utama",
    "Wulandari", "Yuliana", "Zainal", "Arief", "Basuki", "Cahyadi", "Darmawan",
    "Effendi", "Fuadi", "Ginting", "Handoko", "Ibrahim", "Juliana", "Khoirul",
    "Lubis", "Mansyur", "Nasution", "Prasetyo", "Rohman", "Suharto", "Tanjung",
    "Utomo", "Wahyudi", "Yuniar", "Zarkasyi", "Abdurrahman", "Bahri", "Chaniago",
    "Darussalam", "Firdaus", "Ginanjar", "Husaini", "Imron", "Jauhari", "Kasim",
    "Latief", "Maulana", "Nurdin", "Pramono", "Rustam", "Saputra", "Thamrin",
    "Akbar", "Bakri", "Cempaka", "Dahlan", "Ekawati", "Fadilah", "Gumilar",
    "Haryanto", "Iskandar", "Jaelani", "Kartini", "Mariani", "Nurcholis",
    "Parwoto", "Qomariyah", "Saleh", "Triono", "Untung", "Vatimah", "Wahab",
    "Yaqin", "Zubair", "Ansori", "Budiman", "Cahyana", "Darwis", "Ediyanto",
]


def _hash_int(s: str) -> int:
    return int(hashlib.md5(s.encode()).hexdigest(), 16)


def fake_full_name(nomor: str) -> str:
    h = _hash_int(nomor)
    first = FIRST_NAMES[h % len(FIRST_NAMES)]
    last  = LAST_NAMES[(h >> 16) % len(LAST_NAMES)]
    return f"{first} {last}"


def fake_username(nomor: str, used: set[str]) -> str:
    h    = _hash_int(nomor)
    name = fake_full_name(nomor)
    parts = name.lower().split()
    base  = f"{parts[0]}.{parts[1][:5]}" if len(parts) > 1 else parts[0]
    # Ensure uniqueness: append 2-digit suffix from hash if collision
    candidate = base
    suffix = h % 100
    while candidate in used:
        candidate = f"{base}{suffix:02d}"
        suffix = (suffix + 1) % 100
    used.add(candidate)
    return candidate


# ─── Helpers ───────────────────────────────────────────────────────────────────
def duuid(prefix: str, key: str) -> str:
    return str(uuid.uuid5(NS, f"{prefix}:{key}"))


def ibent_role_uuid(id_peran: str) -> str:
    return duuid("ibent_role", id_peran)


def branch_uuid(code: str) -> str:
    return f"d0000000-0000-0000-0000-{int(code, 10):012d}"


def q(v: str | None) -> str:
    if v is None:
        return "NULL"
    return "'" + str(v).replace("'", "''") + "'"


def psql_csv(sql: str) -> list[dict[str, str]]:
    cmd = ["psql", SOURCE_DSN, "--csv", "-c", sql]
    env = {**os.environ, "PGPASSWORD": "solusi"}
    r = subprocess.run(cmd, capture_output=True, text=True, env=env)
    if r.returncode != 0:
        print(r.stderr, file=sys.stderr)
        sys.exit(1)
    return list(csv.DictReader(io.StringIO(r.stdout.strip())))


# ─── Main ──────────────────────────────────────────────────────────────────────
def main() -> None:
    out_path  = Path(__file__).parent.parent / "demo" / "000006_seed_ibankdb_employees.up.sql"
    down_path = Path(__file__).parent.parent / "demo" / "000006_seed_ibankdb_employees.down.sql"

    # Load ibent.peran (81 roles)
    print("[gen] loading ibent.peran...", file=sys.stderr)
    peran_rows = psql_csv(f"SELECT id_peran, nama_peran FROM {SCHEMA}.peran ORDER BY id_peran")

    # Load employee → ibent roles mapping
    print("[gen] loading ibent.listperanuser...", file=sys.stderr)
    role_rows = psql_csv(
        f"SELECT id_user, id_peran FROM {SCHEMA}.listperanuser ORDER BY id_user, id_peran"
    )
    valid_peran = {p["id_peran"] for p in peran_rows}
    employee_roles: dict[str, list[str]] = {}
    for r in role_rows:
        if r["id_peran"] in valid_peran:  # skip orphan codes (IBOPR, KADSO, etc.)
            employee_roles.setdefault(r["id_user"], []).append(r["id_peran"])

    # Load active employees in seeded branches
    print("[gen] loading ibent.employee...", file=sys.stderr)
    in_clause = ", ".join(f"'{b}'" for b in sorted(SEEDED_BRANCHES))
    employees = psql_csv(
        f"SELECT nomor_karyawan, kode_cabang, status_aktif "
        f"FROM {SCHEMA}.employee "
        f"WHERE kode_cabang IN ({in_clause}) "
        f"ORDER BY kode_cabang, nomor_karyawan"
    )
    print(f"[gen] {len(employees)} employees, {len(peran_rows)} ibent roles", file=sys.stderr)

    # Build sections
    roles_vals:  list[str] = []
    users_vals:  list[str] = []
    creds_vals:  list[str] = []
    ur_vals:     list[str] = []
    branch_vals: list[str] = []

    # 1 — ibent.peran → auth7.roles
    # is_default: TLR dan TLRMKR (teller)
    default_codes = {"TLR", "TLRMKR"}
    for p in peran_rows:
        rid   = ibent_role_uuid(p["id_peran"])
        is_def = "true" if p["id_peran"] in default_codes else "false"
        roles_vals.append(
            f"    ({q(rid)}, {q(ORG_ID)}, {q(p['id_peran'])},\n"
            f"     {q(p['nama_peran'])}, {q(p['nama_peran'])}, {is_def})"
        )

    # 2 — Employees as anonymized users
    used_usernames: set[str] = set()
    for row in employees:
        nomor = (row.get("nomor_karyawan") or "").strip()
        kode  = (row.get("kode_cabang") or "").strip().zfill(3)
        aktif = (row.get("status_aktif") or "F").strip().upper() == "T"

        if not nomor or kode not in SEEDED_BRANCHES:
            continue

        user_id  = duuid("employee", nomor)
        username = fake_username(nomor, used_usernames)
        nama     = fake_full_name(nomor)
        email    = f"{username}@bank.local"
        status   = "active" if aktif else "inactive"
        bid      = branch_uuid(kode)

        users_vals.append(
            f"    ({q(user_id)}, {q(ORG_ID)}, {q(username)}, {q(email)},\n"
            f"     {q(nama)}, {q(status)}, true, false, '', false, false, 0, 'id')"
        )
        creds_vals.append(
            f"    (gen_random_uuid(), {q(user_id)}, 'password',\n"
            f"     {q(DEFAULT_PASSWORD_HASH)}, NOW())"
        )
        branch_vals.append(
            f"    (gen_random_uuid(), {q(user_id)}, {q(bid)},\n"
            f"     {q(ORG_ID)}, true, 'system', NOW())"
        )

        # User roles: semua ibent roles yang dimiliki employee ini
        ibent_list = employee_roles.get(nomor, [])
        if not ibent_list:
            # Fallback: TLR jika tidak ada di listperanuser
            ibent_list = ["TLR"]
        for id_peran in ibent_list:
            rid = ibent_role_uuid(id_peran)
            ur_vals.append(
                f"    (gen_random_uuid(), {q(user_id)}, {q(rid)},\n"
                f"     {q(ORG_ID)}, 'system', NOW())"
            )

    # ─── Write up.sql ──────────────────────────────────────────────────────────
    out: list[str] = [
        "-- 000006_seed_ibankdb_employees.up.sql",
        "-- Auto-generated by migrations-seed/scripts/gen_ibankdb_employees.py",
        f"-- Source: ibankdb_medium.ibent — {len(employees)} employees, {len(peran_rows)} roles",
        "-- Nama dan username telah dianonimkan (bukan data asli karyawan).",
        "-- Default password: password123",
        "",
        f"-- ─── 1. Ibent Roles ({len(roles_vals)}) ───────────────────────────────────────────",
        "-- Semua peran dari ibent.peran disalin sebagai auth7.roles.",
        "INSERT INTO roles (id, org_id, code, name, description, is_default)",
        "VALUES",
        ",\n".join(roles_vals),
        "ON CONFLICT (org_id, code) DO UPDATE",
        "    SET name = EXCLUDED.name, description = EXCLUDED.description;",
        "",
        f"-- ─── 2. Users ({len(users_vals)}) ──────────────────────────────────────────────────",
        "INSERT INTO users (id, org_id, username, email, full_name, status, email_verified,",
        "                   mfa_enabled, mfa_method, mfa_reset_required, require_password_change,",
        "                   failed_login_attempts, preferred_locale)",
        "VALUES",
        ",\n".join(users_vals),
        "ON CONFLICT (username) DO UPDATE",
        "    SET full_name = EXCLUDED.full_name, status = EXCLUDED.status;",
        "",
        "-- ─── 3. Credentials ────────────────────────────────────────────────────────────",
        "INSERT INTO user_credentials (id, user_id, credential_type, secret_hash, created_at)",
        "VALUES",
        ",\n".join(creds_vals),
        "ON CONFLICT (user_id, credential_type) DO NOTHING;",
        "",
        f"-- ─── 4. User Roles ({len(ur_vals)} assignments) ───────────────────────────────────",
        "-- Menggunakan ibent role UUID langsung (bukan simplified 5-role model).",
        "INSERT INTO user_roles (id, user_id, role_id, org_id, granted_by, granted_at)",
        "VALUES",
        ",\n".join(ur_vals),
        "ON CONFLICT DO NOTHING;",
        "",
        f"-- ─── 5. Branch Assignments ({len(branch_vals)}) ────────────────────────────────────",
        "INSERT INTO user_branch_assignments (id, user_id, branch_id, org_id, is_primary, assigned_by, assigned_at)",
        "VALUES",
        ",\n".join(branch_vals),
        "ON CONFLICT (user_id, branch_id) DO NOTHING;",
        "",
    ]
    out_path.write_text("\n".join(out))
    print(f"[gen] wrote {out_path}", file=sys.stderr)

    # ─── Write down.sql ────────────────────────────────────────────────────────
    down: list[str] = [
        "-- 000006_seed_ibankdb_employees.down.sql",
        "-- Rollback: hapus ibankdb employees + ibent roles (jaga 6 demo users hardcoded)",
        "",
        "-- Hapus assignments untuk ibankdb users (UUID bukan 00000000-0000-0000-0001-*)",
        "DELETE FROM user_branch_assignments",
        "WHERE user_id IN (",
        "    SELECT id FROM users",
        "    WHERE org_id = '00000000-0000-0000-0000-000000000001'",
        "      AND id NOT LIKE '00000000-0000-0000-0001-%'",
        ");",
        "DELETE FROM user_roles",
        "WHERE user_id IN (",
        "    SELECT id FROM users",
        "    WHERE org_id = '00000000-0000-0000-0000-000000000001'",
        "      AND id NOT LIKE '00000000-0000-0000-0001-%'",
        ");",
        "DELETE FROM user_credentials",
        "WHERE user_id IN (",
        "    SELECT id FROM users",
        "    WHERE org_id = '00000000-0000-0000-0000-000000000001'",
        "      AND id NOT LIKE '00000000-0000-0000-0001-%'",
        ");",
        "DELETE FROM users",
        "WHERE org_id = '00000000-0000-0000-0000-000000000001'",
        "  AND id NOT LIKE '00000000-0000-0000-0001-%';",
        "",
        "-- Hapus ibent roles (code tidak termasuk 5 simplified roles)",
        "DELETE FROM roles",
        f"WHERE org_id = '00000000-0000-0000-0000-000000000001'",
        "  AND code NOT IN ('SUPER_ADMIN', 'BRANCH_MANAGER', 'SUPERVISOR', 'TELLER', 'AUDITOR');",
        "",
    ]
    down_path.write_text("\n".join(down))
    print(f"[gen] wrote {down_path}", file=sys.stderr)

    # Summary
    roles_assigned = len([r for r in employees if (r.get("nomor_karyawan") or "").strip() in employee_roles])
    print(f"[gen] summary: {len(roles_vals)} ibent roles | {len(users_vals)} users | "
          f"{len(ur_vals)} role-assignments | {roles_assigned}/{len(employees)} from listperanuser",
          file=sys.stderr)


if __name__ == "__main__":
    main()
