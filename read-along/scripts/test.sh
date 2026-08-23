#!/usr/bin/env bash
# Run the full Read-Along test suite.
set -euo pipefail

cd "$(dirname "$0")/.."

if [ ! -d .venv ]; then
    echo "Virtual environment missing. Run ./scripts/install.sh first." >&2
    exit 1
fi

# shellcheck disable=SC1091
source .venv/bin/activate

echo "================================"
echo "Read-Along Test Suite"
echo "================================"
echo ""

set +e
pytest -m "not integration" "$@"
RESULT=$?
set -e

echo ""
if [ $RESULT -eq 0 ]; then
    echo "All tests passed."
else
    echo "TESTS FAILED (exit code $RESULT)"
fi
exit $RESULT
