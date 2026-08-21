#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
out_dir=${1:?usage: build-plugins.sh <output-dir>}
plugin_root="$repo_root/distribution/plugins"
command -v bun >/dev/null 2>&1 || { echo "bun is required" >&2; exit 2; }

(cd "$plugin_root" && bun test && bun run build)
[ -f "$plugin_root/dist/index.js" ] || { echo "plugin bundle missing" >&2; exit 1; }
mkdir -p "$out_dir"
cp "$plugin_root/dist/index.js" "$out_dir/index.js"

if grep -Eq 'from ["'"'](?!\.|/|bun:|node:)|require\(["'"'][^./]' "$out_dir/index.js" 2>/dev/null; then
  echo "external dependency import found in plugin bundle" >&2
  exit 1
fi
rm -f "$out_dir/package.json" "$out_dir/bun.lock" "$out_dir/bun.lockb"
echo "plugin bundle ready: self-contained ESM"
