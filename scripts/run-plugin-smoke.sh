#!/usr/bin/env bash
set -euo pipefail

# run-plugin-smoke.sh
#
# Builds the plugin bundle then exercises it with zero public-network activity:
#   - bundle-smoke.ts: guard + audit + Dify degradation on the built bundle
#   - plugin-smoke.ts: the OpenCode plugin adapter — loads the bundle's DEFAULT
#     export (the readV1Plugin contract), invokes server() to obtain Hooks.tool
#     (the fromPlugin contract), and drives all 8 tools (7 enterprise + dify-query)
#     through the guard: path deny, DLP input deny, write permission ask, output
#     DLP block, and Dify degradation.

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
bun run tests/plugin-smoke.ts
