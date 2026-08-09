#!/bin/bash
set -euo pipefail

# Check that no package outside the OpenCode vendor layer imports opencode.
# Allowed consumers: opencode itself, opencode tests, cmd/ composition roots.
# Rejected consumers: application, TUI models/components, harness, agent, policy.

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TUI_DIR="${1:-$PROJECT_ROOT/tui}"

cd "$TUI_DIR"

FORBIDDEN_PKG="codea/tui/internal/opencode"

# Collect all packages excluding the vendor layer itself
ALL_PKGS=$(go list ./... 2>/dev/null | grep -v 'internal/opencode' || true)

if [ -z "$ALL_PKGS" ]; then
    echo "PASS: no packages outside opencode vendor layer"
    exit 0
fi

VIOLATIONS=""
while IFS= read -r pkg; do
    IMPORTS=$(go list -f '{{join .Imports "\n"}}' "$pkg" 2>/dev/null || true)
    if echo "$IMPORTS" | grep -qF "$FORBIDDEN_PKG"; then
        VIOLATIONS="$VIOLATIONS$pkg imports $FORBIDDEN_PKG"$'\n'
    fi
done <<< "$ALL_PKGS"

if [ -n "$VIOLATIONS" ]; then
    echo "FAIL: forbidden opencode imports detected:"
    echo "$VIOLATIONS"
    exit 1
fi

echo "PASS: no vendor DTO leakage"
