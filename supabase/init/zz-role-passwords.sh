#!/bin/bash
# Init script for supabase-db (runs ONLY on first init, empty data dir).
#
# The name starts with zz- ON PURPOSE: the entrypoint runs
# /docker-entrypoint-initdb.d/* in glob order, and the image's own
# migrate.sh (which creates the supabase_admin/authenticator roles and the
# auth/storage schemas) must run FIRST. A previous 01- version ran before it
# and aborted the whole init, leaving clusters without postgres role/schema.
#
# It also never aborts init: any failure only warns. An abort here leaves a
# half-initialized cluster behind (no auth schema), which is worse than a
# missing password. The password uses shell-level quote doubling instead of
# psql :'var' interpolation (which failed silently on this image).
set -u
export PGPASSWORD="$POSTGRES_PASSWORD"
escaped_pw=$(printf '%s' "$POSTGRES_PASSWORD" | sed "s/'/''/g")
if psql -U supabase_admin -d postgres -v ON_ERROR_STOP=1 \
  -c "ALTER USER authenticator WITH PASSWORD '$escaped_pw';"; then
  echo "authenticator password set"
else
  echo "WARNING: could not set authenticator password; PostgREST will reject connections until fixed with: ALTER USER authenticator WITH PASSWORD '...';" >&2
fi
