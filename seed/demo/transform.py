#!/usr/bin/env python3
"""
transform.py — CSV (ibent Indonesian) -> SQL for auth7 (idempotent ON CONFLICT).

Produces:
    seed_001_organization.sql   # organizations, branch_types, permissions catalog
    seed_002_branches.sql       # branches (UUIDs MATCH enterprise.branches.id)
    seed_003_roles.sql          # roles, role_permissions, branch_default_roles
    seed_004_users.sql          # users, user_credentials, user_roles, user_branch_assignments

Cross-service UUID consistency:
    NS prefix table is at seed/demo/README.md.  Every prefix/key combo MUST
    match the enterprise/policy7/audit7 transformers.
"""

from __future__ import annotations

import csv
import os
import sys
import uuid
from pathlib import Path

ROOT = Path(__file__).parent
CSV_DIR = ROOT / "csv"

NS = uuid.UUID("11111111-1111-1111-1111-111111111111")


def deterministic_uuid(prefix: str, key: str) -> str:
    return str(uuid.uuid5(NS, f"{prefix}:{key}"))


def t(value: str | None) -> str:
    if value is None or value == "" or value.upper() == "NULL":
        return "NULL"
    return "'" + value.replace("'", "''") + "'"


def read_csv(name: str) -> list[dict[str, str]]:
    path = CSV_DIR / f"{name}.csv"
    if not path.exists():
        print(f"[warn] {path} missing — skipping", file=sys.stderr)
        return []
    with path.open() as f:
        return list(csv.DictReader(f))


def load_env() -> dict[str, str]:
    env_path = ROOT / ".env.seed"
    env: dict[str, str] = {}
    if env_path.exists():
        for line in env_path.read_text().splitlines():
            line = line.strip()
            if not line or line.startswith("#") or "=" not in line:
                continue
            k, v = line.split("=", 1)
            env[k.strip()] = v.strip()
    return env


# ─── Enum translation (subset mirrored from enterprise/transform.py) ─────────

ENUM_BRANCH_TYPE = {
    "CP": "SUB_BRANCH",
    "CU": "MAIN_BRANCH",
    "GM": "OUTLET",
    "KS": "CASH_OFFICE",
    "NO": "NON_OPERATIONAL_HQ",
    "PO": "OPERATIONAL_HQ",
    "LS": "SYARIAH_SERVICE_UNIT",
    "KF": "FUNCTIONAL_OFFICE",
    "CC": "COST_CENTER",
}

ENUM_BRANCH_CLASSIFICATION = {
    "B": "MAIN_BRANCH", "S": "SUB_BRANCH", "P": "HEAD_OFFICE", "U": "SYARIAH_SERVICE_UNIT",
}

# Auth7 branch_types metadata catalog — defines short_code/level/sort per type.
BRANCH_TYPE_CATALOG = [
    # (code, label, short_code, level, is_operational, can_have_children, sort)
    ("OPERATIONAL_HQ",       "Kantor Pusat Operasional",   "PO", 0, False, True,  1),
    ("NON_OPERATIONAL_HQ",   "Kantor Pusat Non-Op",        "NO", 0, False, True,  2),
    ("MAIN_BRANCH",          "Cabang Umum",                "CU", 1, True,  True,  3),
    ("SUB_BRANCH",           "Cabang Pembantu",            "CP", 2, True,  True,  4),
    ("OUTLET",               "Gerai",                      "GM", 3, True,  False, 5),
    ("CASH_OFFICE",          "Kantor Kas",                 "KS", 3, True,  False, 6),
    ("SYARIAH_SERVICE_UNIT", "Unit Layanan Syariah",       "LS", 3, True,  False, 7),
    ("FUNCTIONAL_OFFICE",    "Kantor Fungsional",          "KF", 2, False, True,  8),
    ("COST_CENTER",          "Cost Center",                "CC", 2, False, False, 9),
]

# Permissions catalog — minimal set covering the master entities + transactions.
# Resource type matches the bos7-* page slug for easy mapping in admin UI.
PERMISSIONS_CATALOG = [
    # (code, name, description, resource_type)
    ("branch.list",       "List Branches",        "View branches",      "branch"),
    ("branch.create",     "Create Branch",        "Create new branch",  "branch"),
    ("branch.update",     "Update Branch",        "Edit branch data",   "branch"),
    ("branch.delete",     "Delete Branch",        "Soft-delete branch", "branch"),
    ("employee.list",     "List Employees",       "View employees",     "employee"),
    ("employee.create",   "Create Employee",      "Create employee",    "employee"),
    ("employee.update",   "Update Employee",      "Edit employee",      "employee"),
    ("employee.delete",   "Delete Employee",      "Soft-delete employee", "employee"),
    ("transfer.create",   "Create Transfer",      "Initiate transfer",  "transfer"),
    ("transfer.authorize","Authorize Transfer",   "Approve transfer",   "transfer"),
    ("customer.list",     "List Customers",       "View customers",     "customer"),
    ("customer.create",   "Create Customer",      "Create customer",    "customer"),
    ("user.admin",        "User Administration",  "Manage users + roles", "user"),
    ("policy.read",       "Read Policy",          "View policy params", "policy"),
    ("policy.write",      "Write Policy",         "Edit policy params", "policy"),
    ("audit.read",        "Read Audit Log",       "View audit events",  "audit"),
]

# Default role catalog — code -> (name, description, default_permissions[])
ROLE_CATALOG = [
    ("SUPER_ADMIN",     "Super Administrator", "Full access",
     ["branch.list","branch.create","branch.update","branch.delete",
      "employee.list","employee.create","employee.update","employee.delete",
      "transfer.create","transfer.authorize","customer.list","customer.create",
      "user.admin","policy.read","policy.write","audit.read"]),
    ("BRANCH_MANAGER",  "Branch Manager", "Branch operations + authorize high-value txn",
     ["branch.list","employee.list","employee.update",
      "transfer.create","transfer.authorize","customer.list","customer.create",
      "policy.read","audit.read"]),
    ("SUPERVISOR",      "Supervisor", "Authorize teller transactions",
     ["transfer.create","transfer.authorize","customer.list","employee.list","audit.read"]),
    ("TELLER",          "Teller", "Frontline transaction handling",
     ["transfer.create","customer.list","customer.create"]),
    ("AUDITOR",         "Auditor", "Read-only across all modules",
     ["branch.list","employee.list","customer.list","audit.read","policy.read"]),
]


def write_seed_001(env: dict[str, str]) -> None:
    out = ROOT / "seed_001_organization.sql"
    print(f"[transform] writing {out}", file=sys.stderr)

    org_code = env.get("SEED_ORG_CODE", "BJBS")
    org_id_fixed = env.get("SEED_ORG_ID", "")
    org_id = org_id_fixed if org_id_fixed else deterministic_uuid("organization", org_code)

    with out.open("w") as f:
        f.write("-- seed_001_organization.sql — generated by transform.py\n\n")

        f.write("-- organization (single tenant for demo)\n")
        f.write(
            "INSERT INTO organizations (id, code, name, status, settings) VALUES ("
            f"{t(org_id)}, {t(org_code)}, {t('Bank Demo BJBS')}, 'active', '{{}}'::jsonb) "
            "ON CONFLICT (id) DO UPDATE SET code = EXCLUDED.code, name = EXCLUDED.name, status = EXCLUDED.status;\n\n"
        )

        f.write("-- branch_types catalog (one row per English branch_type enum)\n")
        for code, label, short, level, is_op, can_child, sort in BRANCH_TYPE_CATALOG:
            bt_id = deterministic_uuid("branch_type", code)
            f.write(
                "INSERT INTO branch_types (id, org_id, code, label, short_code, level, is_operational, can_have_children, sort_order) "
                f"VALUES ({t(bt_id)}, {t(org_id)}, {t(code)}, {t(label)}, {t(short)}, {level}, {is_op}, {can_child}, {sort}) "
                "ON CONFLICT (org_id, code) DO UPDATE SET label = EXCLUDED.label, level = EXCLUDED.level, "
                "is_operational = EXCLUDED.is_operational, can_have_children = EXCLUDED.can_have_children;\n"
            )

        f.write("\n-- permissions catalog (org-independent — global rows)\n")
        for code, name, desc, res in PERMISSIONS_CATALOG:
            pid = deterministic_uuid("permission", code)
            f.write(
                "INSERT INTO permissions (id, code, name, description, resource_type) "
                f"VALUES ({t(pid)}, {t(code)}, {t(name)}, {t(desc)}, {t(res)}) "
                "ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description;\n"
            )


def write_seed_002(env: dict[str, str]) -> None:
    """branches — UUIDs are deterministic per branch_code so they match enterprise."""
    out = ROOT / "seed_002_branches.sql"
    print(f"[transform] writing {out}", file=sys.stderr)

    org_code = env.get("SEED_ORG_CODE", "BJBS")
    org_id_fixed = env.get("SEED_ORG_ID", "")
    org_id = org_id_fixed if org_id_fixed else deterministic_uuid("organization", org_code)

    # Pulls from the enterprise extract's cabang.csv if it exists (preferred —
    # avoids needing Oracle here).  Falls back to ../../core7-service-enterprise/seed/demo/csv/cabang.csv
    # so auth7 can be seeded without re-running the extract.
    cabang_csv = ROOT.parent.parent.parent.parent / "appdist" / "services" / "core7-service-enterprise" / "seed" / "demo" / "csv" / "cabang.csv"
    rows: list[dict[str, str]] = []
    if cabang_csv.exists():
        with cabang_csv.open() as f:
            rows = list(csv.DictReader(f))
    else:
        print(f"[warn] {cabang_csv} missing — auth7.branches will be empty.", file=sys.stderr)
        print("       Run enterprise extract first, or extract a CABANG csv into auth7/seed/demo/csv/", file=sys.stderr)

    with out.open("w") as f:
        f.write("-- seed_002_branches.sql — generated by transform.py\n")
        f.write("-- branches.id IDENTIK dengan enterprise.branches.id (same uuid5 derivation).\n\n")

        for row in rows:
            code = row.get("kode_cabang", "")
            if not code:
                continue
            branch_id = deterministic_uuid("branch", code)
            branch_type = ENUM_BRANCH_TYPE.get((row.get("tipe_cabang") or "").upper(), "MAIN_BRANCH")
            branch_type_id = deterministic_uuid("branch_type", branch_type)
            classification = ENUM_BRANCH_CLASSIFICATION.get((row.get("status_cabang") or "").upper())
            parent_code = row.get("kode_cabang_induk", "")
            parent_id = deterministic_uuid("branch", parent_code) if parent_code else None
            status = "active" if (row.get("status_aktif","T") or "T").upper() in ("T","TRUE","1") else "inactive"

            f.write(
                "INSERT INTO branches (id, org_id, branch_type_id, code, name, status, address, phone, "
                "branch_type, parent_branch_id, branch_classification) VALUES ("
                f"{t(branch_id)}, {t(org_id)}, {t(branch_type_id)}, {t(code)}, "
                f"{t(row.get('nama_cabang',''))}, {t(status)}, "
                f"{t(row.get('kantor_alamat',''))}, {t(row.get('kantor_telepon1',''))}, "
                f"{t(branch_type)}, {t(parent_id) if parent_id else 'NULL'}, "
                f"{t(classification) if classification else 'NULL'}) "
                "ON CONFLICT (org_id, code) DO UPDATE SET name = EXCLUDED.name, status = EXCLUDED.status, "
                "branch_type = EXCLUDED.branch_type, branch_type_id = EXCLUDED.branch_type_id, "
                "parent_branch_id = EXCLUDED.parent_branch_id, "
                "branch_classification = EXCLUDED.branch_classification;\n"
            )

        f.write("\n-- branch_hierarchies (parent → child rows) — populated from parent_branch_id\n")
        f.write(
            "INSERT INTO branch_hierarchies (id, parent_id, child_id) "
            "SELECT gen_random_uuid(), parent_branch_id, id FROM branches "
            "WHERE parent_branch_id IS NOT NULL "
            "ON CONFLICT DO NOTHING;\n"
        )


def write_seed_003(env: dict[str, str]) -> None:
    """roles, role_permissions, branch_default_roles."""
    out = ROOT / "seed_003_roles.sql"
    print(f"[transform] writing {out}", file=sys.stderr)

    org_code = env.get("SEED_ORG_CODE", "BJBS")
    org_id_fixed = env.get("SEED_ORG_ID", "")
    org_id = org_id_fixed if org_id_fixed else deterministic_uuid("organization", org_code)

    with out.open("w") as f:
        f.write("-- seed_003_roles.sql — generated by transform.py\n\n")

        for code, name, desc, perms in ROLE_CATALOG:
            role_id = deterministic_uuid("role", code)
            is_default = "TRUE" if code in ("TELLER",) else "FALSE"
            f.write(
                "INSERT INTO roles (id, org_id, code, name, description, is_default) "
                f"VALUES ({t(role_id)}, {t(org_id)}, {t(code)}, {t(name)}, {t(desc)}, {is_default}) "
                "ON CONFLICT (org_id, code) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description;\n"
            )
            # role_permissions
            for p in perms:
                pid = deterministic_uuid("permission", p)
                f.write(
                    "INSERT INTO role_permissions (role_id, permission_id) "
                    f"VALUES ({t(role_id)}, {t(pid)}) "
                    "ON CONFLICT DO NOTHING;\n"
                )

        f.write("\n-- branch_default_roles: TELLER as default for every active branch\n")
        teller_id = deterministic_uuid("role", "TELLER")
        f.write(
            "INSERT INTO branch_default_roles (id, branch_id, role_id, is_default) "
            f"SELECT gen_random_uuid(), b.id, {t(teller_id)}, TRUE FROM branches b "
            f"WHERE b.org_id = {t(org_id)} AND b.status = 'active' "
            "ON CONFLICT (branch_id, role_id) DO NOTHING;\n"
        )


def write_seed_004(env: dict[str, str]) -> None:
    """users + user_credentials + user_roles + user_branch_assignments."""
    out = ROOT / "seed_004_users.sql"
    print(f"[transform] writing {out}", file=sys.stderr)

    org_code = env.get("SEED_ORG_CODE", "BJBS")
    org_id_fixed = env.get("SEED_ORG_ID", "")
    org_id = org_id_fixed if org_id_fixed else deterministic_uuid("organization", org_code)
    pwd_hash = env.get("SEED_DEFAULT_PASSWORD_HASH",
        "$argon2id$v=19$m=65536,t=3,p=4$c29tZXNhbHQ$2EgLsEMqNccY7XTG8Bxtl5Pumi4Zcs1KkJ2cspqHCiA")

    # Prefer auth7's own user.csv extract; fall back to enterprise employee.csv (employee → user 1:1).
    user_csv = CSV_DIR / "user.csv"
    if not user_csv.exists():
        # Fallback path: use enterprise employees as the user roster.
        user_csv = ROOT.parent.parent.parent.parent / "appdist" / "services" / "core7-service-enterprise" / "seed" / "demo" / "csv" / "employee.csv"
    if not user_csv.exists():
        print(f"[warn] no user source ({CSV_DIR/'user.csv'} or enterprise employee.csv missing) — seed_004 will be empty.", file=sys.stderr)
        rows = []
    else:
        with user_csv.open() as f:
            rows = list(csv.DictReader(f))

    # ibent.listperanuser: (id_peran, id_user, level_peranuser)
    # ibent.listcabangdiizinkan: (id_user, kode_cabang)
    listperanuser = {row["id_user"]: row.get("id_peran","") for row in read_csv("listperanuser") if row.get("id_user")}
    listcabang   = {}
    for r in read_csv("listcabangdiizinkan"):
        listcabang.setdefault(r.get("id_user",""), []).append(r.get("kode_cabang",""))

    # Load the branch subset (from enterprise's cabang.csv) so user_branch_assignments
    # only references branches that actually exist in this seed.  Without this filter,
    # ibent's full listcabangdiizinkan would create orphan refs to branches outside
    # the 5-branch subset.
    branch_subset: set[str] = set()
    enterprise_cabang_csv = ROOT.parent.parent.parent.parent / "appdist" / "services" / "core7-service-enterprise" / "seed" / "demo" / "csv" / "cabang.csv"
    if enterprise_cabang_csv.exists():
        with enterprise_cabang_csv.open() as f:
            branch_subset = {row["kode_cabang"] for row in csv.DictReader(f) if row.get("kode_cabang")}
    if not branch_subset:
        print(f"[warn] enterprise cabang.csv missing — user_branch_assignments will not be filtered", file=sys.stderr)

    with out.open("w") as f:
        f.write("-- seed_004_users.sql — generated by transform.py\n\n")

        for row in rows:
            # Try both ibent USER schema and enterprise employee schema.
            user_key = row.get("userid") or row.get("nomor_karyawan") or row.get("nama_lengkap","")
            if not user_key:
                continue
            user_id = deterministic_uuid("user", user_key)
            full_name = row.get("nama_lengkap") or row.get("nama") or user_key
            email = row.get("email","") or f"{user_key.lower()}@bank.local"
            username = (user_key.lower().replace(" ", ".") if user_key else "").strip()
            branch_code = row.get("kode_cabang") or ""
            branch_id = deterministic_uuid("branch", branch_code) if branch_code else None

            f.write(
                "INSERT INTO users (id, org_id, username, email, full_name, status, email_verified, mfa_enabled) "
                f"VALUES ({t(user_id)}, {t(org_id)}, {t(username)}, {t(email)}, "
                f"{t(full_name)}, 'active', TRUE, FALSE) "
                "ON CONFLICT (id) DO UPDATE SET status = EXCLUDED.status, email_verified = TRUE, "
                "full_name = EXCLUDED.full_name, email = EXCLUDED.email;\n"
            )
            cred_id = deterministic_uuid("user_credential", user_key)
            f.write(
                "INSERT INTO user_credentials (id, user_id, credential_type, secret_hash, version, is_current) "
                f"VALUES ({t(cred_id)}, {t(user_id)}, 'password', {t(pwd_hash)}, 1, TRUE) "
                "ON CONFLICT (id) DO NOTHING;\n"
            )

            # Default role assignment: TELLER unless ibent gives a different role.
            role_code = "TELLER"
            ibent_role = listperanuser.get(user_key)
            if ibent_role:
                # Map ibent role codes to English role codes when possible.
                role_map = {"ADMIN": "SUPER_ADMIN", "SUPERVISOR": "SUPERVISOR",
                            "BRANCH_MANAGER": "BRANCH_MANAGER", "TELLER": "TELLER",
                            "AUDITOR": "AUDITOR"}
                role_code = role_map.get(ibent_role.upper(), "TELLER")
            role_id = deterministic_uuid("role", role_code)
            ur_id = deterministic_uuid("user_role", f"{user_key}:{role_code}:{branch_code or 'global'}")
            granted_by = deterministic_uuid("user", "SYSTEM")
            f.write(
                "INSERT INTO user_roles (id, user_id, role_id, org_id, branch_id, granted_by) "
                f"VALUES ({t(ur_id)}, {t(user_id)}, {t(role_id)}, {t(org_id)}, "
                f"{t(branch_id) if branch_id else 'NULL'}, {t(granted_by)}) "
                "ON CONFLICT (user_id, role_id, org_id, branch_id) DO NOTHING;\n"
            )

            # user_branch_assignments: primary = home branch, plus any LISTCABANGDIIZINKAN entries.
            # Filter to branches present in the seed subset to avoid orphan refs.
            allowed = list(set([branch_code] + listcabang.get(user_key, [])))
            for i, bc in enumerate(allowed):
                if not bc:
                    continue
                if branch_subset and bc not in branch_subset:
                    continue  # skip branches outside the seed subset
                bid = deterministic_uuid("branch", bc)
                uba_id = deterministic_uuid("user_branch_assignment", f"{user_key}:{bc}")
                is_primary = "TRUE" if i == 0 and bc == branch_code else "FALSE"
                f.write(
                    "INSERT INTO user_branch_assignments (id, user_id, branch_id, is_primary) "
                    f"VALUES ({t(uba_id)}, {t(user_id)}, {t(bid)}, {is_primary}) "
                    "ON CONFLICT (user_id, branch_id) DO UPDATE SET is_primary = EXCLUDED.is_primary;\n"
                )


if __name__ == "__main__":
    env = load_env()
    write_seed_001(env)
    write_seed_002(env)
    write_seed_003(env)
    write_seed_004(env)
    print("[transform] done", file=sys.stderr)
