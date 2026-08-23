#!/usr/bin/env bash
# Start API + worker under PM2 (falls back to plain processes without PM2).
set -euo pipefail

cd "$(dirname "$0")/.."

if [ ! -d .venv ]; then
    echo "Virtual environment missing. Run ./scripts/install.sh first." >&2
    exit 1
fi

if ! command -v pm2 >/dev/null 2>&1; then
    echo "pm2 not found. Install with: npm install -g pm2" >&2
    echo "Or start manually:"
    echo "  .venv/bin/uvicorn app.api.main:app --host 0.0.0.0 --port 8000"
    echo "  .venv/bin/python -m app.worker.main"
    exit 1
fi

pm2 start ecosystem.config.js
echo ""
echo "Waiting for health check..."
for _ in $(seq 1 15); do
    if curl -sf "http://localhost:${API_PORT:-8000}/health" >/dev/null 2>&1; then
        echo "API is healthy."
        pm2 status
        exit 0
    fi
    sleep 1
done

echo "WARN: health check did not pass within 15s; inspect 'pm2 logs audiobook-api'." >&2
pm2 status
exit 1
