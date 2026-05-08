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
  echo "→ Database ready."
fi

echo "→ Running database migrations..."
./auth7 migrate up
echo "→ Migrations done."

if [ -n "$DATABASE_ADMIN_URL" ] && [ -f scripts/seed-data.sql ]; then
  echo "→ Seeding initial data..."
  psql "${DATABASE_ADMIN_URL%/postgres*}/auth7?sslmode=disable" -f scripts/seed-data.sql 2>&1 | tail -5 || true
  echo "→ Seed done."
fi

exec "$@"
