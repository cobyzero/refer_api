#!/bin/bash
# Init script for supabase-db (runs ONLY on first init, empty data dir).
# The supabase/postgres image leaves the `authenticator` role (used by
# PostgREST) without password; set it from POSTGRES_PASSWORD so
# PGRST_DB_URI works out of the box. Uses :'pw' quoting (safe for any chars,
# though alphanumeric passwords are still recommended to avoid URL issues).
set -e
export PGPASSWORD="$POSTGRES_PASSWORD"
psql -U supabase_admin -d postgres -v ON_ERROR_STOP=1 -v pw="$POSTGRES_PASSWORD" \
  -c "ALTER USER authenticator WITH PASSWORD :'pw';"
