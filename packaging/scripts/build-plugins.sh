#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
out_dir=${1:?usage: build-plugins.sh <output-dir>}
plugin_root="$repo_root/distribution/plugins"
command -v bun >/dev/null 2>&1 || { echo "bun is required" >&2; exit 2; }
command -v python3 >/dev/null 2>&1 || { echo "python3 is required" >&2; exit 2; }

(cd "$plugin_root" && bun test && bun run build)
[ -f "$plugin_root/dist/index.js" ] || { echo "plugin bundle missing" >&2; exit 1; }
mkdir -p "$out_dir"
cp "$plugin_root/dist/index.js" "$out_dir/index.js"
rm -f "$out_dir/package.json" "$out_dir/bun.lock" "$out_dir/bun.lockb"

python3 - "$out_dir/index.js" <<'PY'
import pathlib, re, sys
p=pathlib.Path(sys.argv[1])
text=p.read_text(errors='replace')
pat=re.compile(r'''(?:from\s*|import\s*\(|require\s*\()\s*["']([^"']+)["']''')
for spec in pat.findall(text):
    if not (spec.startswith('.') or spec.startswith('/') or spec.startswith('node:') or spec.startswith('bun:') or spec.startswith('data:')):
        raise SystemExit(f'external dependency import found in plugin bundle: {spec}')
print('plugin import audit passed')
PY

echo "plugin bundle ready: self-contained ESM"
