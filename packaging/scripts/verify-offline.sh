#!/usr/bin/env bash
set -euo pipefail

stage=${1:?usage: verify-offline.sh <staging-dir>}
[ -d "$stage" ] || { echo "staging dir missing: $stage" >&2; exit 1; }

for forbidden in "$stage/plugins/package.json" "$stage/plugins/bun.lock" "$stage/plugins/bun.lockb"; do
  [ ! -e "$forbidden" ] || { echo "FAIL: runtime plugin dependency metadata found: $forbidden" >&2; exit 1; }
done

command -v python3 >/dev/null 2>&1 || { echo "python3 is required" >&2; exit 2; }
python3 - "$stage" <<'PY'
import pathlib, re, sys
root=pathlib.Path(sys.argv[1]).resolve()
plugins=root/'plugins'
if not plugins.is_dir(): raise SystemExit('FAIL: plugins directory missing')
js=list(plugins.glob('*.js'))
if not js: raise SystemExit('FAIL: no plugin bundle found')
patterns=[
    re.compile(r'''(?:from\s*|import\s*\(|require\s*\()\s*["']([^"']+)["']'''),
    re.compile(r'''(?:^|[;\n])\s*import\s*["']([^"']+)["']'''),
]
for p in js:
    text=p.read_text(errors='replace')
    for pat in patterns:
        for spec in pat.findall(text):
            if not (spec.startswith('.') or spec.startswith('/') or spec.startswith('node:') or spec.startswith('bun:') or spec.startswith('data:')):
                raise SystemExit(f'FAIL: external import {spec!r} in {p.name}')
    if str(pathlib.Path.home()) in text:
        raise SystemExit(f'FAIL: build home path leaked into {p.name}')
for p in root.rglob('*'):
    if p.is_file() and p.name in {'package.json','bun.lock','bun.lockb'} and 'plugins' in p.parts:
        raise SystemExit(f'FAIL: plugin install metadata present: {p.relative_to(root)}')
print('offline static checks passed')
PY

if [ -f "$stage/manifest.json" ]; then
  "$(dirname "${BASH_SOURCE[0]}")/verify-checksum.sh" "$stage"
fi

echo "All offline checks passed."
