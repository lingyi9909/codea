#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
platform=${1:?usage: build-runtime.sh <darwin-arm64|darwin-x64|windows-x64> <output-dir>}
out_dir=${2:?usage: build-runtime.sh <platform> <output-dir>}
metadata="$repo_root/runtime/version.json"

command -v python3 >/dev/null 2>&1 || { echo "python3 is required" >&2; exit 2; }
command -v curl >/dev/null 2>&1 || { echo "curl is required" >&2; exit 2; }

IFS=$'\t' read -r version url expected asset < <(python3 - "$metadata" "$platform" <<'PY'
import json, pathlib, sys
p=json.loads(pathlib.Path(sys.argv[1]).read_text())
plat=sys.argv[2]
if p.get('openCodeVersion') != '1.18.11':
    raise SystemExit('runtime/version.json is not locked to OpenCode 1.18.11')
x=p.get('platforms',{}).get(plat)
if not x: raise SystemExit(f'unsupported platform: {plat}')
checksum=x['checksum']
if checksum.startswith('sha256:'):
    checksum=checksum[len('sha256:'):]
print('\t'.join([p['openCodeVersion'], x['url'], checksum, x['asset']]))
PY
)
[ -n "$version" ] && [ -n "$url" ] && [ -n "$expected" ] && [ -n "$asset" ] || { echo "invalid runtime metadata" >&2; exit 1; }
mkdir -p "$out_dir"
tmp=$(mktemp -d)
trap 'rm -rf "$tmp"' EXIT
archive="$tmp/$asset"
curl --fail --location --proto '=https' --tlsv1.2 "$url" -o "$archive"
actual=$(python3 - "$archive" <<'PY'
import hashlib, pathlib, sys
h=hashlib.sha256(); h.update(pathlib.Path(sys.argv[1]).read_bytes()); print(h.hexdigest())
PY
)
[ "$actual" = "$expected" ] || { echo "OpenCode archive checksum mismatch" >&2; exit 1; }
case "$asset" in
  *.zip) command -v unzip >/dev/null 2>&1 || { echo "unzip is required" >&2; exit 2; }; mkdir -p "$tmp/extract"; unzip -q "$archive" -d "$tmp/extract" ;;
  *.tar.gz|*.tgz) mkdir -p "$tmp/extract"; tar -xzf "$archive" -C "$tmp/extract" ;;
  *) echo "unsupported runtime archive: $asset" >&2; exit 2 ;;
esac
bin=$(find "$tmp/extract" -type f \( -name opencode -o -name opencode.exe \) -print -quit)
[ -n "$bin" ] || { echo "OpenCode binary not found in $asset" >&2; exit 1; }
cp "$bin" "$out_dir/$(basename "$bin")"
chmod +x "$out_dir/$(basename "$bin")" 2>/dev/null || true
printf '%s\n' "$version" > "$out_dir/OPENCODE_VERSION"
echo "runtime ready: $platform OpenCode v$version"
