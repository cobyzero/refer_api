#!/bin/sh
# Ejecuta un comando con las vars de .env (parseo robusto: tolera quotes y
# caracteres especiales que rompen a `source`). Uso:
#   sh scripts/run-with-env.sh go run ./cmd/supabase-migrate -step=validate
exec python3 - "$@" <<'EOF'
import os, subprocess, sys
env = dict(os.environ)
with open('.env') as f:
    for line in f:
        s = line.strip()
        if not s or s.startswith('#') or '=' not in s:
            continue
        k, v = s.split('=', 1)
        env[k.strip()] = v.strip()
r = subprocess.run(sys.argv[1:], env=env)
sys.exit(r.returncode)
EOF
