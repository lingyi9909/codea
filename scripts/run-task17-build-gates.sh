#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
out_root=${1:-"$repo_root/.task17-release"}
evidence_file=${2:-"$repo_root/tests/offline/evidence/task17-build-evidence.json"}
rm -rf "$out_root"
mkdir -p "$out_root" "$(dirname "$evidence_file")"

for cmd in go bun python3 curl; do
  command -v "$cmd" >/dev/null 2>&1 || { echo "BLOCKED: $cmd is required" >&2; exit 2; }
done

go_version=$(go version)
printf '%s\n' "$go_version" | grep -q 'go1.26.5' || { echo "BLOCKED: Go 1.26.5 required, got: $go_version" >&2; exit 2; }

platforms=(darwin-arm64 darwin-x64 windows-x64)
for platform in "${platforms[@]}"; do
  echo "=== build $platform ==="
  "$repo_root/packaging/scripts/build-release.sh" "$platform" "$out_root"
done

python3 - "$out_root" "$evidence_file" <<'PY'
import hashlib, json, pathlib, sys, time
root=pathlib.Path(sys.argv[1]).resolve(); out=pathlib.Path(sys.argv[2])
expected={
  'darwin-arm64': '.tar.gz',
  'darwin-x64': '.tar.gz',
  'windows-x64': '.zip',
}
releases={}
for platform, suffix in expected.items():
    matches=sorted(root.glob(f'codea-*-{platform}{suffix}'))
    if len(matches) != 1: raise SystemExit(f'{platform}: expected one archive, found {len(matches)}')
    archive=matches[0]; checksum_file=pathlib.Path(str(archive)+'.sha256')
    if not checksum_file.is_file(): raise SystemExit(f'{platform}: archive.sha256 missing')
    actual=hashlib.sha256(archive.read_bytes()).hexdigest()
    declared=checksum_file.read_text().split()[0]
    if actual != declared: raise SystemExit(f'{platform}: archive sha mismatch')
    stage=root/archive.name.removesuffix(suffix)
    if not stage.is_dir(): raise SystemExit(f'{platform}: staging directory missing')
    required=['VERSION','runtime-version.json','manifest.json','plugins/index.js']
    for rel in required:
        if not (stage/rel).is_file(): raise SystemExit(f'{platform}: missing {rel}')
    for rel in ['agents','skills','config','install','bin']:
        if not (stage/rel).is_dir(): raise SystemExit(f'{platform}: missing directory {rel}')
    releases[platform]={
      'archive': archive.name,
      'sha256': actual,
      'bytes': archive.stat().st_size,
      'staticPackageVerified': True,
    }
payload={
  'timestamp': time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime()),
  'openCodeVersion': '1.18.11',
  'platforms': releases,
  'passedChecks': 3,
  'totalChecks': 3,
}
out.parent.mkdir(parents=True, exist_ok=True)
out.write_text(json.dumps(payload, indent=2)+'\n')
print('Task 17 three-platform build evidence: 3/3 PASS')
PY

echo "[PASS] Task 17 release archives built for darwin-arm64, darwin-x64, windows-x64"
echo "evidence: $evidence_file"
