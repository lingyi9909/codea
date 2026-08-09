#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CHECKER="$PROJECT_ROOT/scripts/check-runtime-boundary.sh"

if [ ! -x "$CHECKER" ]; then
    echo "SKIP: check-runtime-boundary.sh not found or not executable"
    exit 0
fi

# --- Test 1: Repository should have zero vendor DTO leakage ---
echo "=== Test 1: Zero vendor DTO leakage in repository ==="
if "$CHECKER"; then
    echo "PASS: no vendor DTO leakage"
else
    echo "FAIL: vendor DTO leakage detected"
    exit 1
fi

# --- Test 2: Fixture with forbidden import should be rejected ---
echo ""
echo "=== Test 2: Fixture with forbidden import ==="

LEAK_DIR="$PROJECT_ROOT/tui/internal/application/leak"
trap 'rm -rf "$PROJECT_ROOT/tui/internal/application"' EXIT

mkdir -p "$LEAK_DIR"
cat > "$LEAK_DIR/leak.go" <<'GOEOF'
package leak

import "codea/tui/internal/opencode"

var _ = opencode.OpenCodeCapabilities
GOEOF

if "$CHECKER" 2>&1; then
    echo "FAIL: checker should have detected the forbidden import"
    exit 1
else
    echo "PASS: checker rejected the forbidden import"
fi

echo ""
echo "All runtime boundary tests passed."
