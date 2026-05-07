#!/bin/sh
# docker-entrypoint.sh — Create DB/user if not exists, then exec auth7

set -e

if [ -n "$DATABASE_ADMIN_URL" ]; then
  echo "→ Ensuring database 'auth7' exists..."
  psql "$DATABASE_ADMIN_URL" -tc "SELECT 1 FROM pg_database WHERE datname='auth7'" \
    | grep -q 1 || psql "$DATABASE_ADMIN_URL" -c "CREATE DATABASE auth7;"
  psql "$DATABASE_ADMIN_URL" -tc "SELECT 1 FROM pg_roles WHERE rolname='auth7'" \
    | grep -q 1 || psql "$DATABASE_ADMIN_URL" -c "CREATE USER auth7 WITH PASSWORD 'postgres';"
  psql "$DATABASE_ADMIN_URL" -c "GRANT ALL PRIVILEGES ON DATABASE auth7 TO auth7;" 2>/dev/null || true
  echo "→ Database ready."
fi

exec "$@"
