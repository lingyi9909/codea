#!/bin/bash
set -euo pipefail

# Check that no package outside the OpenCode vendor layer imports opencode.
# Allowed consumers: opencode itself, opencode tests, cmd/ composition roots, tests/.
# Rejected consumers: application, TUI models/components, harness, agent, policy.

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
TUI_DIR="${1:-$PROJECT_ROOT/tui}"

cd "$TUI_DIR"

if ! command -v go &>/dev/null; then
    echo "FAIL: go is not installed or not on PATH"
    exit 1
fi

FORBIDDEN_PKG="codea/tui/internal/opencode"

ALL_PKGS=$(go list ./... 2>/dev/null) || {
    echo "FAIL: go list failed — cannot enumerate packages"
    exit 1
}

ALL_PKGS=$(echo "$ALL_PKGS" | grep -v 'internal/opencode' | grep -v '/tests/' || true)

if [ -z "$ALL_PKGS" ]; then
    echo "FAIL: no non-opencode packages found — cannot verify boundary"
    exit 1
fi

VIOLATIONS=""
while IFS= read -r pkg; do
    IMPORTS=$(go list -f '{{join .Imports "\n"}}' "$pkg" 2>/dev/null) || {
        echo "FAIL: go list failed for package $pkg"
        exit 1
    }
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
