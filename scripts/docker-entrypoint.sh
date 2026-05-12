#!/bin/sh
# docker-entrypoint.sh — Create DB/user if not exists, then exec auth7

set -e

if [ -n "$DATABASE_ADMIN_URL" ]; then
  echo "→ Ensuring database 'auth7' exists..."
  psql "$DATABASE_ADMIN_URL" -tc "SELECT 1 FROM pg_database WHERE datname='auth7'" \
    | grep -q 1 || psql "$DATABASE_ADMIN_URL" -c "CREATE DATABASE auth7;"
  psql "$DATABASE_ADMIN_URL" -tc "SELECT 1 FROM pg_roles WHERE rolname='auth7'" \
    | grep -q 1 || psql "$DATABASE_ADMIN_URL" -c "CREATE USER auth7 WITH PASSWORD 'postgres';"
  psql "$DATABASE_ADMIN_URL" -c "ALTER USER auth7 WITH PASSWORD 'postgres';" 2>/dev/null || true
  psql "$DATABASE_ADMIN_URL" -c "GRANT ALL PRIVILEGES ON DATABASE auth7 TO auth7;" 2>/dev/null || true
  # Grant schema privileges so golang-migrate can create schema_migrations table
  AUTH7_DB_URL="${DATABASE_ADMIN_URL%/postgres*}/auth7?sslmode=disable"
  psql "$AUTH7_DB_URL" -c "GRANT CREATE ON SCHEMA public TO auth7;" 2>/dev/null || true
  psql "$AUTH7_DB_URL" -c "GRANT USAGE ON SCHEMA public TO auth7;" 2>/dev/null || true
  # Force search_path to public at both database and role level.
  # Without this, PostgreSQL 15+ may resolve tables to "$user" schema (auth7)
  # instead of public, causing FK references to fail across migrations.
  psql "$AUTH7_DB_URL" -c "ALTER DATABASE auth7 SET search_path TO public;" 2>/dev/null || true
  psql "$AUTH7_DB_URL" -c "ALTER ROLE auth7 SET search_path TO public;" 2>/dev/null || true
  # Reset DB if organizations table is not in public schema (means previous migrations
  # ran with wrong search_path and created tables in auth7 schema instead of public)
  ORG_IN_PUBLIC=$(psql "$AUTH7_DB_URL" -tAc \
    "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='public' AND table_name='organizations'" \
    2>/dev/null | tr -d '[:space:]' || echo "0")
  if [ "$ORG_IN_PUBLIC" != "1" ]; then
    echo "→ Schema inconsistency detected (organizations not in public) — resetting..."
    psql "$AUTH7_DB_URL" -c "DROP SCHEMA IF EXISTS auth7 CASCADE;" 2>/dev/null || true
    psql "$AUTH7_DB_URL" -c "DELETE FROM schema_migrations;" 2>/dev/null || true
    echo "→ DB reset complete."
  else
    # Pre-populate schema_migrations for all migration files that exist on disk.
    # This handles the case where schema_migrations lost entries (e.g. after a reset)
    # but the tables themselves already exist. Without this, golang-migrate would try
    # to re-apply old migrations and fail with "relation already exists".
    echo "→ Syncing schema_migrations with migration files on disk..."
    for f in migrations/*.up.sql; do
      [ -f "$f" ] || continue
      ver=$(basename "$f" | sed 's/_.*//')
      psql "$AUTH7_DB_URL" -c "INSERT INTO schema_migrations (version, dirty) VALUES ($ver, false) ON CONFLICT DO NOTHING;" 2>/dev/null || true
    done
    # Fix any remaining dirty entries using migrate force (marks clean without re-running).
    DIRTY_VERSION=$(psql "$AUTH7_DB_URL" -tAc "SELECT version FROM schema_migrations WHERE dirty=true LIMIT 1" 2>/dev/null | tr -d '[:space:]' || echo "")
    if [ -n "$DIRTY_VERSION" ]; then
      echo "→ Repairing dirty migration state (version ${DIRTY_VERSION})..."
      ./auth7 migrate force "$DIRTY_VERSION" 2>/dev/null || \
        psql "$AUTH7_DB_URL" -c "UPDATE schema_migrations SET dirty=false WHERE version=${DIRTY_VERSION};" 2>/dev/null || true
    fi
  fi
  echo "→ Database ready."
fi

echo "→ Running database migrations..."
./auth7 migrate up
echo "→ Migrations done."

if [ -n "$DATABASE_ADMIN_URL" ] && [ -f scripts/seed-data.sql ]; then
  echo "→ Seeding initial data..."
  psql "${DATABASE_ADMIN_URL%/postgres*}/auth7?sslmode=disable" -f scripts/seed-data.sql 2>&1 || true
  echo "→ Seed done."
fi

# Ensure extra redirect URIs are present (idempotent).
# Seed-data.sql handles Railway URIs by default. This block handles:
#   1. Custom deployment domains (set EXTRA_REDIRECT_DOMAIN env var)
#   2. Legacy cleanup if old images are still running
if [ -n "$DATABASE_ADMIN_URL" ] && [ -n "$EXTRA_REDIRECT_DOMAIN" ]; then
  echo "→ Ensuring extra redirect URIs for domain: ${EXTRA_REDIRECT_DOMAIN}"
  AUTH7_DB="${DATABASE_ADMIN_URL%/postgres*}/auth7?sslmode=disable"
  for app in portal:3006 enterprise:3003 financing:3010 funding:3011 treasury:3012 smt:3013 accounting:3014 cif:3015 internalaccount:3016 remittance:3017 batchprocessing:3018; do
    client_id="bos7-${app%%:*}"
    extra_uri="https://bos7-${app%%:*}.${EXTRA_REDIRECT_DOMAIN}/api/auth/callback"
    extra_app="https://bos7-${app%%:*}.${EXTRA_REDIRECT_DOMAIN}"
    psql "$AUTH7_DB" -c "UPDATE oauth2_clients SET app_url = '${extra_app}' WHERE client_id = '${client_id}';" 2>/dev/null || true
    psql "$AUTH7_DB" -c "UPDATE oauth2_clients SET allowed_redirect_uris = array_append(allowed_redirect_uris, '${extra_uri}') WHERE client_id = '${client_id}' AND NOT ('${extra_uri}' = ANY(allowed_redirect_uris));" 2>/dev/null || true
  done
  # auth7-ui uses 'account' subdomain, not 'bos7-' prefix
  AUTH7_UI_URI="https://account.${EXTRA_REDIRECT_DOMAIN}/api/auth/callback"
  psql "$AUTH7_DB" -c "UPDATE oauth2_clients SET allowed_redirect_uris = array_append(allowed_redirect_uris, '${AUTH7_UI_URI}') WHERE client_id = 'auth7-ui-dev' AND NOT ('${AUTH7_UI_URI}' = ANY(allowed_redirect_uris));" 2>/dev/null || true
  echo "→ Extra redirect URIs ensured."
fi

exec "$@"
