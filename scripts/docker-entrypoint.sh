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
    # Fix dirty migration state from any previous failed deploy
    DIRTY=$(psql "$AUTH7_DB_URL" -tAc "SELECT COUNT(*) FROM schema_migrations WHERE dirty=true" 2>/dev/null || echo "0")
    if [ "$DIRTY" != "0" ] && [ -n "$DIRTY" ]; then
      echo "→ Repairing dirty migration state (${DIRTY} entry)..."
      psql "$AUTH7_DB_URL" -c "DELETE FROM schema_migrations WHERE dirty=true;" 2>/dev/null || true
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

exec "$@"
