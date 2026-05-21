#!/usr/bin/env bash
# extract.sh — pull auth7-relevant subset from local ibankdb_medium (PostgreSQL).
#
# Source DB: postgresql://ibank@localhost:5432/ibankdb_medium  (schema: ibent)
# Target:    ./csv/*.csv
#
# Subset criterion: employees whose home branch (kode_cabang) belongs to the
# enterprise extract's top-N branches.  Run this AFTER the enterprise extract
# (so the same branch subset is honoured), or set SEED_BRANCH_LIMIT identically.

set -euo pipefail
cd "$(dirname "$0")"

# shellcheck disable=SC1091
[ -f .env.seed ] && source .env.seed

PGHOST="${PGHOST:-localhost}"
PGPORT="${PGPORT:-5432}"
PGUSER="${SOURCE_PG_USER:-ibank}"
PGPASSWORD="${SOURCE_PG_PASSWORD:-}"
SOURCE_DB="${SOURCE_PG_DB:-ibankdb_medium}"
SOURCE_SCHEMA="${SOURCE_PG_SCHEMA:-ibent}"
BRANCH_LIMIT="${SEED_BRANCH_LIMIT:-5}"
USER_LIMIT="${SEED_USER_LIMIT:-50}"

export PGHOST PGPORT PGUSER PGPASSWORD

mkdir -p csv

psql_copy() {
  local out="$1"
  local sql="$2"
  echo "[extract] $out"
  psql "$SOURCE_DB" -v ON_ERROR_STOP=1 -c "\\copy ($sql) TO '$PWD/$out' WITH (FORMAT csv, HEADER, NULL '')"
}

BRANCH_FILTER="SELECT kode_cabang FROM $SOURCE_SCHEMA.cabang WHERE status_aktif='T' ORDER BY kode_cabang LIMIT $BRANCH_LIMIT"

# --- Roles & permissions metadata (full) --------------------------------------
psql_copy csv/peran.csv             "SELECT * FROM $SOURCE_SCHEMA.peran"
psql_copy csv/aplikasi.csv          "SELECT * FROM $SOURCE_SCHEMA.aplikasi"
psql_copy csv/fungsi.csv            "SELECT * FROM $SOURCE_SCHEMA.fungsi"
psql_copy csv/listaplikasiperan.csv "SELECT * FROM $SOURCE_SCHEMA.listaplikasiperan"
psql_copy csv/listfungsiaplikasiperan.csv "SELECT * FROM $SOURCE_SCHEMA.listfungsiaplikasiperan"

# --- Users from usertbl (passwords) + biodata for full name -----------------
# usertbl in this snapshot is sparse — extract anyway; transform.py falls back
# to enterprise employee.csv when this is empty.
psql_copy csv/usertbl.csv "SELECT * FROM $SOURCE_SCHEMA.usertbl"

# --- User -> role mapping (filtered to subset) ------------------------------
# The subset is "users present in usertbl OR users referenced by employees in the
# branch subset".  We pull all mappings for users in the branch subset:
psql_copy csv/listperanuser.csv "
  SELECT lpu.*
  FROM $SOURCE_SCHEMA.listperanuser lpu
  WHERE lpu.id_user IN (
    SELECT DISTINCT nomor_karyawan FROM (
      SELECT *, ROW_NUMBER() OVER (PARTITION BY kode_cabang ORDER BY nomor_karyawan) rn
      FROM $SOURCE_SCHEMA.employee
      WHERE kode_cabang IN ($BRANCH_FILTER)
    ) e WHERE rn <= $USER_LIMIT
  )
"

# --- User -> allowed branches (multi-branch access) -------------------------
psql_copy csv/listcabangdiizinkan.csv "
  SELECT lcd.*
  FROM $SOURCE_SCHEMA.listcabangdiizinkan lcd
  WHERE lcd.id_user IN (
    SELECT DISTINCT nomor_karyawan FROM (
      SELECT *, ROW_NUMBER() OVER (PARTITION BY kode_cabang ORDER BY nomor_karyawan) rn
      FROM $SOURCE_SCHEMA.employee
      WHERE kode_cabang IN ($BRANCH_FILTER)
    ) e WHERE rn <= $USER_LIMIT
  )
"

echo "[extract] done — see csv/"
