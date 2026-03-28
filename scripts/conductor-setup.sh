#!/usr/bin/env bash
set -euo pipefail

# Conductor allocates ten ports starting at CONDUCTOR_PORT.
# Map them to the dev services if running inside Conductor.
if [ -n "${CONDUCTOR_PORT:-}" ]; then
  cat > .env.local <<EOF
WEB_PORT=$((CONDUCTOR_PORT + 0))
SERVER_PORT=$((CONDUCTOR_PORT + 1))
OIDC_PORT=$((CONDUCTOR_PORT + 2))
POSTGRES_PORT=$((CONDUCTOR_PORT + 3))
TILT_PORT=$((CONDUCTOR_PORT + 4))
EOF
  echo "Wrote .env.local with ports $CONDUCTOR_PORT+0..+4"
fi

mise trust
mise install
