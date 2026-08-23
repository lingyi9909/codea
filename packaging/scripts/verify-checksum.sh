#!/usr/bin/env bash
set -euo pipefail

stage=${1:?usage: verify-checksum.sh <staging-dir>}
manifest="$stage/manifest.json"
[ -f "$manifest" ] || { echo "manifest.json missing" >&2; exit 1; }
command -v python3 >/dev/null 2>&1 || { echo "python3 is required" >&2; exit 2; }
python3 - "$stage" "$manifest" <<'PY'
import hashlib, json, pathlib, sys
root=pathlib.Path(sys.argv[1]).resolve()
m=json.loads(pathlib.Path(sys.argv[2]).read_text())
if m.get('schemaVersion') != 1 or m.get('algorithm') != 'sha256':
    raise SystemExit('unsupported manifest schema or algorithm')
seen=set()
for entry in m.get('files',[]):
    rel=entry['path']
    if rel.startswith('/') or '..' in pathlib.PurePosixPath(rel).parts:
        raise SystemExit(f'unsafe manifest path: {rel}')
    p=root/rel
    if not p.is_file(): raise SystemExit(f'manifest file missing: {rel}')
    actual=hashlib.sha256(p.read_bytes()).hexdigest()
    if actual != entry.get('sha256'): raise SystemExit(f'checksum mismatch: {rel}')
    if p.stat().st_size != entry.get('size'): raise SystemExit(f'size mismatch: {rel}')
    seen.add(rel)
actual_files={p.relative_to(root).as_posix() for p in root.rglob('*') if p.is_file() and p.name!='manifest.json'}
extra=actual_files-seen
if extra: raise SystemExit('unmanifested files: '+', '.join(sorted(extra)))
print(f'checksums verified: {len(seen)} files')
PY
