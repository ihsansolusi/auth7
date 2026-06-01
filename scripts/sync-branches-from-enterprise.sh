#!/usr/bin/env bash
# sync-branches-from-enterprise.sh
#
# One-shot dev sync: pull ALL active branches from core7_enterprise.branches
# into auth7.branches (id, branch_code, name, is_active), then remap any
# user_branch_assignments that pointed at legacy (non-enterprise) branch
# UUIDs to the canonical enterprise UUID for the same branch_code.
#
# Production will replace this with a NATS-based push subscription
# (enterprise emits branch.{created,updated,deleted}; auth7 subscribes).
# Until that lands, rerun this script after enterprise branch changes.
#
# Both DBs must be on the same Postgres instance reachable as the role
# given by AUTH7_DSN / ENTERPRISE_DSN below. Local default uses the
# postgres superuser for the enterprise side since enterprise's branches
# table sits in a separate database with its own owner.

set -euo pipefail

AUTH7_DSN="${AUTH7_DSN:-postgres://auth7:auth7secret@localhost:5432/auth7?sslmode=disable}"
ENTERPRISE_DSN="${ENTERPRISE_DSN:-postgres://postgres:postgres@localhost:5432/core7_enterprise?sslmode=disable}"
ORG_ID="${ORG_ID:-00000000-0000-0000-0000-000000000001}"

echo "─── source (enterprise) ─────────────────────────────────────"
ENT_COUNT=$(psql "$ENTERPRISE_DSN" -tA -c "SELECT COUNT(*) FROM branches WHERE org_id='$ORG_ID' AND is_active=true;")
echo "  active branches in core7_enterprise (org $ORG_ID): $ENT_COUNT"

echo "─── dump enterprise rows to staging file ─────────────────────"
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT
DUMP="$TMPDIR/branches.csv"
psql "$ENTERPRISE_DSN" -c "\COPY (SELECT id, org_id, branch_code, branch_name, is_active FROM branches WHERE org_id='$ORG_ID') TO '$DUMP' WITH CSV"
echo "  rows dumped: $(wc -l < "$DUMP")"

echo "─── sync auth7.branches + remap assignments ──────────────────"
psql "$AUTH7_DSN" -v ON_ERROR_STOP=1 <<SQL
BEGIN;

-- (a) Stage: enterprise rows (canonical UUIDs).
CREATE TEMP TABLE stage (
    id          UUID,
    org_id      UUID,
    branch_code VARCHAR(20),
    branch_name VARCHAR(255),
    is_active   BOOLEAN
) ON COMMIT DROP;

\COPY stage FROM '$DUMP' WITH CSV

-- (b) Snapshot the legacy branches that collide with enterprise on
-- (org_id, branch_code) — same code, different id. We need to remap
-- assignments away from these before we can delete them.
CREATE TEMP TABLE legacy_collisions ON COMMIT DROP AS
SELECT old.id AS legacy_id, canonical.id AS canonical_id, old.org_id, old.branch_code
FROM branches old
JOIN stage canonical
  ON canonical.org_id = old.org_id
 AND canonical.branch_code = old.branch_code
 AND canonical.id <> old.id;

-- (c) Park legacy branch_code so the (org_id, branch_code) UNIQUE
-- constraint frees up for the canonical row. Use the first 18 chars of the
-- legacy UUID with a "_L:" prefix — fits VARCHAR(20).
UPDATE branches
SET branch_code = '_L:' || substr(branches.id::text, 1, 17)
WHERE id IN (SELECT legacy_id FROM legacy_collisions);

-- (d) Upsert enterprise canonical rows.
INSERT INTO branches (id, org_id, branch_code, is_active, name)
SELECT id, org_id, branch_code, is_active, branch_name FROM stage
ON CONFLICT (id) DO UPDATE SET
    branch_code = EXCLUDED.branch_code,
    name        = EXCLUDED.name,
    is_active   = EXCLUDED.is_active,
    updated_at  = NOW();

-- (e) Remap assignments: legacy_id → canonical_id.
UPDATE user_branch_assignments uba
SET branch_id = lc.canonical_id
FROM legacy_collisions lc
WHERE uba.branch_id = lc.legacy_id;

-- (f) Delete the now-orphan parked legacy rows. Their assignments have
-- been remapped in (e), so cascade is a no-op.
DELETE FROM branches WHERE branch_code LIKE '_L:%';

-- (g) Delete orphan branches that have no enterprise counterpart at all
-- (no row in stage with the same id AND no code-collision either).
-- FK ON DELETE CASCADE handles any leftover assignments.
DELETE FROM branches
WHERE id NOT IN (SELECT id FROM stage)
  AND org_id = '$ORG_ID';

COMMIT;
SQL

echo "─── verify ───────────────────────────────────────────────────"
AUTH7_COUNT=$(psql "$AUTH7_DSN" -tA -c "SELECT COUNT(*) FROM branches WHERE org_id='$ORG_ID';")
echo "  auth7.branches now (org $ORG_ID): $AUTH7_COUNT"
psql "$AUTH7_DSN" -c "SELECT branch_code, name FROM branches WHERE org_id='$ORG_ID' ORDER BY branch_code LIMIT 5;"
echo ""
echo "  user_branch_assignments for known demo users:"
psql "$AUTH7_DSN" -c "SELECT u.username, b.branch_code, b.name FROM users u JOIN user_branch_assignments uba ON uba.user_id=u.id JOIN branches b ON b.id=uba.branch_id WHERE u.username IN ('admin','teller','bosdemo','manager','spv','auditor') ORDER BY u.username;"

echo "─── done ──────────────────────────────────────────────────────"
