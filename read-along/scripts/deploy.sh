#!/usr/bin/env bash
# Deploy: pull, install, test, restart PM2 processes, verify health.
# Tests failing => production is NOT restarted.
set -euo pipefail

cd "$(dirname "$0")/.."

API_PORT="${API_PORT:-8000}"

echo "==> 1/6 git pull"
if [ "${SKIP_PULL:-0}" != "1" ] && git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
    git pull --ff-only || echo "WARN: git pull failed (detached or dirty tree); continuing with local state."
else
    echo "Skipping git pull."
fi

echo "==> 2/6 install/update dependencies"
./scripts/install.sh

echo "==> 3/6 run tests"
if ! ./scripts/test.sh; then
    echo ""
    echo "!!! Tests failed — DO NOT restart production. Aborting deploy." >&2
    exit 1
fi

echo "==> 4/6 restart API + worker"
if command -v pm2 >/dev/null 2>&1; then
    pm2 reload ecosystem.config.js --update-env \
        || pm2 start ecosystem.config.js
else
    echo "pm2 not found; cannot restart managed processes." >&2
    exit 1
fi

echo "==> 5/6 verify health"
HEALTHY=0
for _ in $(seq 1 20); do
    if curl -sf "http://localhost:${API_PORT}/health" >/dev/null 2>&1; then
        HEALTHY=1
        break
    fi
    sleep 1
done

if [ "$HEALTHY" != "1" ]; then
    echo "Health check failed after restart. Check 'pm2 logs audiobook-api'." >&2
    exit 1
fi

echo "==> 6/6 done"
curl -s "http://localhost:${API_PORT}/health"
echo ""
echo "Deployment successful."
