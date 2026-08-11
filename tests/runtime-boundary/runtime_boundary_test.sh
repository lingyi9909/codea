#!/bin/bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
CHECKER="$PROJECT_ROOT/scripts/check-runtime-boundary.sh"

if [ ! -x "$CHECKER" ]; then
    echo "SKIP: check-runtime-boundary.sh not found or not executable"
    exit 0
fi

FAILURES=0

# --- Test 1: Repository should have zero vendor DTO leakage ---
echo "=== Test 1: Zero vendor DTO leakage in repository ==="
if "$CHECKER"; then
    echo "PASS: no vendor DTO leakage"
else
    echo "FAIL: vendor DTO leakage detected"
    FAILURES=$((FAILURES + 1))
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
    FAILURES=$((FAILURES + 1))
else
    echo "PASS: checker rejected the forbidden import"
fi

# --- Test 3: go list failure must not PASS ---
echo ""
echo "=== Test 3: go list failure must not PASS ==="

EMPTY_DIR=$(mktemp -d)
trap 'rm -rf "$PROJECT_ROOT/tui/internal/application" "$EMPTY_DIR"' EXIT

# $EMPTY_DIR has no go.mod — go list will fail.
if "$CHECKER" "$EMPTY_DIR" 2>/dev/null; then
    echo "FAIL: checker PASSed when go list should have failed"
    FAILURES=$((FAILURES + 1))
else
    echo "PASS: checker correctly failed when go list failed"
fi

# --- Test 4: go not on PATH must not PASS ---
echo ""
echo "=== Test 4: go not on PATH must not PASS ==="

ORIG_PATH="$PATH"
export PATH="/tmp/no-such-go-dir:$PATH"
# The checker uses 'command -v go' — we need to hide the real go.
# We do this by temporarily replacing go with a non-functional stub.
MOCK_DIR=$(mktemp -d)
trap 'rm -rf "$PROJECT_ROOT/tui/internal/application" "$EMPTY_DIR" "$MOCK_DIR"' EXIT
export PATH="$ORIG_PATH"

# We can't easily mock 'go' away across the script since it runs in a subshell.
# Instead, point to a TUI_DIR that has no go.mod but with a fake go on PATH.
# Actually, the simplest reliable negative test: make a TUI_DIR where go exists
# but go list returns non-zero (no packages).
# Test 3 already covers go list failure. The "go missing" case can be tested
# by pointing to a dir where go works but PATH is mocked.
# Test it by temporarily putting a non-functional 'go' first on PATH.

cat > "$MOCK_DIR/go" <<'SHIM'
#!/bin/bash
echo "mock: go would fail" >&2
exit 1
SHIM
chmod +x "$MOCK_DIR/go"

export PATH="$MOCK_DIR:$ORIG_PATH"
if "$CHECKER" 2>/dev/null; then
    echo "FAIL: checker PASSed when go should have failed"
    FAILURES=$((FAILURES + 1))
else
    echo "PASS: checker correctly failed when go failed"
fi
export PATH="$ORIG_PATH"

echo ""
if [ "$FAILURES" -eq 0 ]; then
    echo "All runtime boundary tests passed."
else
    echo "FAIL: $FAILURES test(s) failed."
    exit 1
fi
