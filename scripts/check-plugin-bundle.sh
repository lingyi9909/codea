#!/usr/bin/env bash
set -euo pipefail

# check-plugin-bundle.sh
#
# Verifies the built plugin bundle is self-contained and offline-safe:
#   - dist/index.js exists (bun build output)
#   - no external (non-bun-builtin) imports or requires
#   - no runtime node_modules / npm package resolution at load time

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PLUGIN_DIR="$(cd "$SCRIPT_DIR/../distribution/plugins" && pwd)"
BUNDLE="$PLUGIN_DIR/dist/index.js"

if [ ! -f "$BUNDLE" ]; then
  echo "FAIL: bundle missing at $BUNDLE (run: bun run build in distribution/plugins)" >&2
  exit 1
fi

# Allowed bare specifiers: bun built-ins only. Everything else must be bundled in.
ALLOWED_BUILTINS='^(fs|path|os|child_process|node:fs|node:path|node:os|node:child_process|bun:test)$'

IMPORTS=$(grep -oE 'from "[^"]+"|from '"'"'[^'"'"']+'"'"'|require\("[^"]+"\)' "$BUNDLE" 2>/dev/null || true)

VIOLATIONS=""
while IFS= read -r line; do
  [ -z "$line" ] && continue
  spec=$(printf '%s' "$line" | sed -E 's/.*from ["'"'"']?([^"'"'"']+)["'"'"']?.*/\1/; s/.*require\("([^"]+)"\).*/\1/')
  [ -z "$spec" ] && continue
  if echo "$spec" | grep -qE '^(\.|/)'; then
    continue  # relative/absolute module paths are bundled in
  fi
  if ! echo "$spec" | grep -qE "$ALLOWED_BUILTINS"; then
    VIOLATIONS="$VIOLATIONS$spec"$'\n'
  fi
done <<< "$IMPORTS"

if [ -n "$VIOLATIONS" ]; then
  echo "FAIL: bundle has non-builtin external imports:" >&2
  echo "$VIOLATIONS" >&2
  exit 1
fi

if [ -d "$PLUGIN_DIR/node_modules" ]; then
  echo "note: node_modules present (dev-only, not required at runtime)"
fi

echo "PASS: bundle self-contained and offline-safe ($BUNDLE)"
