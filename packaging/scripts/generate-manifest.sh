#!/usr/bin/env bash
set -euo pipefail

stage=${1:?usage: generate-manifest.sh <staging-dir>}
command -v python3 >/dev/null 2>&1 || { echo "python3 is required" >&2; exit 2; }
python3 - "$stage" <<'PY'
import hashlib, json, pathlib, sys
root=pathlib.Path(sys.argv[1]).resolve()
if not root.is_dir(): raise SystemExit(f'staging dir missing: {root}')
files=[]
for p in sorted(root.rglob('*')):
    if not p.is_file() or p.name == 'manifest.json':
        continue
    rel=p.relative_to(root).as_posix()
    h=hashlib.sha256(p.read_bytes()).hexdigest()
    files.append({'path': rel, 'sha256': h, 'size': p.stat().st_size})
manifest={'schemaVersion':1,'algorithm':'sha256','files':files}
(root/'manifest.json').write_text(json.dumps(manifest, indent=2)+'\n')
print(f'manifest generated: {len(files)} files')
PY
