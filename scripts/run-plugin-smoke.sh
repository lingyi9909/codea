#!/usr/bin/env bash
set -euo pipefail

# run-plugin-smoke.sh
#
# Builds the plugin bundle then loads it and exercises the security foundation
# (guard + audit + Dify degradation) with zero public-network activity.

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PLUGIN_DIR="$SCRIPT_DIR/../distribution/plugins"

if ! command -v bun >/dev/null 2>&1; then
  if [ -x "$HOME/.bun/bin/bun" ]; then
    export PATH="$HOME/.bun/bin:$PATH"
  else
    echo "FAIL: bun not found on PATH" >&2
    exit 2
  fi
fi

cd "$PLUGIN_DIR"
bun run build
bun run tests/bundle-smoke.ts
