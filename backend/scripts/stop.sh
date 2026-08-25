#!/usr/bin/env bash
# Stop API + worker.
set -euo pipefail

cd "$(dirname "$0")/.."

if ! command -v pm2 >/dev/null 2>&1; then
    echo "pm2 not found." >&2
    exit 1
fi

pm2 stop ecosystem.config.js || true
pm2 status || true
