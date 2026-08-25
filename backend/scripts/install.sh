#!/usr/bin/env bash
# Read-Along Audiobook System installer.
# Fails clearly when a required dependency is missing.
set -euo pipefail

cd "$(dirname "$0")/.."
ROOT_DIR="$(pwd)"
VENV_DIR="$ROOT_DIR/.venv"

log()  { printf '\033[1;34m[install]\033[0m %s\n' "$*"; }
ok()   { printf '\033[1;32m  ok\033[0m %s\n' "$*"; }
fail() { printf '\033[1;31mFAIL:\033[0m %s\n' "$*" >&2; exit 1; }

log "Checking system dependencies..."

# 1. Python 3.11+
if command -v python3 >/dev/null 2>&1; then
    PY_VERSION=$(python3 -c 'import sys; print(f"{sys.version_info.major}.{sys.version_info.minor}")')
    PY_OK=$(python3 -c 'import sys; print(int(sys.version_info >= (3, 11)))')
    [ "$PY_OK" = "1" ] || fail "Python 3.11+ required, found $PY_VERSION at $(command -v python3)"
    ok "Python $PY_VERSION ($(command -v python3))"
else
    fail "python3 not found on PATH. Install Python 3.11+ first."
fi

# 2-3. Node/npm/PM2 (needed for process management)
if command -v node >/dev/null 2>&1; then ok "Node $(node -v)"; else
    echo "WARN: node not found (only needed to run PM2)"; fi

PM2_CMD=""
if command -v pm2 >/dev/null 2>&1; then
    PM2_CMD="pm2"; ok "PM2 v$(pm2 -v)"
elif command -v npm >/dev/null 2>&1 && [ "${INSTALL_PM2:-0}" = "1" ]; then
    log "Installing PM2 globally via npm..."
    npm install -g pm2 >/dev/null 2>&1 && PM2_CMD="pm2" \
        && ok "PM2 v$(pm2 -v)" || echo "WARN: PM2 install failed; you can still run without it."
else
    echo "WARN: pm2 not found. Install with 'npm install -g pm2' to use scripts/start.sh / deploy.sh."
fi

# FFmpeg is needed for MP3 output; WAV works without it.
if command -v ffmpeg >/dev/null 2>&1; then
    ok "FFmpeg ($(ffmpeg -version | head -n1 | cut -d' ' -f3))"
else
    echo "WARN: ffmpeg not found; audio will be stored as WAV instead of MP3."
fi

# 4. Virtual environment
if [ ! -d "$VENV_DIR" ]; then
    log "Creating virtual environment..."
    python3 -m venv "$VENV_DIR" || fail "Could not create virtualenv (is python3-venv installed?)"
fi
# shellcheck disable=SC1091
source "$VENV_DIR/bin/activate"
log "Upgrading pip..."
pip install --quiet --upgrade pip

# 5-6. Python dependencies + Kokoro engine
log "Installing Python dependencies (this includes torch/kokoro and can take a while)..."
if ! pip install -r "$ROOT_DIR/requirements.txt"; then
    log "Full install failed; retrying without heavy TTS stack (tests will still run with the fake provider)."
    grep -v -E '^kokoro' "$ROOT_DIR/requirements.txt" > "$ROOT_DIR/.requirements-slim.txt"
    pip install -r "$ROOT_DIR/.requirements-slim.txt" || fail "pip install failed"
fi
ok "Python dependencies installed"

# 7-8. Model download (non-fatal: downloads automatically on first synthesis)
log "Prefetching Kokoro model (optional)..."
python -m app.tts.prefetch && ok "Kokoro model ready" \
    || echo "WARN: model prefetch skipped; it will download on first use."

# 9. Data directories + SQLite schema
log "Creating data directories and initialising SQLite..."
python -m app.database.init_db || fail "database init failed"
ok "Database initialised"

# 10. Basic health test
log "Running basic health test..."
python - <<'EOF'
from app.api.main import app  # noqa: F401
print("API import OK:", app.title)
EOF

log "Install complete."
echo ""
echo "Next steps:"
echo "  source .venv/bin/activate"
echo "  ./scripts/test.sh          # run the test suite"
[ -n "$PM2_CMD" ] && echo "  ./scripts/start.sh         # start API + worker under PM2" \
    || echo "  uvicorn app.api.main:app   # start the API manually"
