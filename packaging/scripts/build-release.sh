#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
platform=${1:?usage: build-release.sh <darwin-arm64|darwin-x64|windows-x64> <output-dir>}
out_root=${2:?usage: build-release.sh <platform> <output-dir>}
version=$(tr -d '[:space:]' < "$repo_root/VERSION")
[ -n "$version" ] || { echo "VERSION is empty" >&2; exit 1; }
case "$platform" in
  darwin-arm64) goos=darwin; goarch=arm64; codea_name=codea; archive_ext=tar.gz ;;
  darwin-x64) goos=darwin; goarch=amd64; codea_name=codea; archive_ext=tar.gz ;;
  windows-x64) goos=windows; goarch=amd64; codea_name=codea.exe; archive_ext=zip ;;
  *) echo "unsupported V1 platform: $platform" >&2; exit 2 ;;
esac

name="codea-$version-$platform"
stage="$out_root/$name"
rm -rf "$stage"
mkdir -p "$stage/bin" "$stage/plugins" "$stage/skills" "$stage/agents" "$stage/config" "$stage/install"

# TUI is cross-compiled at build time. The target never needs a Go toolchain.
(
  cd "$repo_root/tui"
  CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" GOTOOLCHAIN=local \
    go build -trimpath -o "$stage/bin/$codea_name" ./cmd/codea
)

"$repo_root/packaging/scripts/build-runtime.sh" "$platform" "$stage/bin"
"$repo_root/packaging/scripts/build-plugins.sh" "$stage/plugins"
"$repo_root/packaging/scripts/collect-skills.sh" "$stage/skills"
cp -R "$repo_root/distribution/agents/." "$stage/agents/"
cp -R "$repo_root/distribution/config/." "$stage/config/"
cp "$repo_root/VERSION" "$stage/VERSION"
cp "$repo_root/runtime/version.json" "$stage/runtime-version.json"

case "$platform" in
  darwin-*)
    cp "$repo_root/packaging/platform/macos/install.sh" "$stage/install/install.sh"
    cp "$repo_root/packaging/scripts/verify-checksum.sh" "$stage/install/verify-checksum.sh"
    cp "$repo_root/packaging/scripts/verify-offline.sh" "$stage/install/verify-offline.sh"
    chmod +x "$stage/install/"*.sh
    ;;
  windows-x64)
    cp "$repo_root/packaging/platform/windows/install.ps1" "$stage/install/install.ps1"
    ;;
esac

"$repo_root/packaging/scripts/generate-manifest.sh" "$stage"
"$repo_root/packaging/scripts/verify-checksum.sh" "$stage"
"$repo_root/packaging/scripts/verify-offline.sh" "$stage"

mkdir -p "$out_root"
archive="$out_root/$name.$archive_ext"
rm -f "$archive" "$archive.sha256"
if [ "$archive_ext" = "tar.gz" ]; then
  (cd "$out_root" && tar -czf "$archive" "$name")
else
  command -v python3 >/dev/null 2>&1 || { echo "python3 is required for zip packaging" >&2; exit 2; }
  python3 - "$stage" "$archive" <<'PY'
import pathlib, sys, zipfile
root=pathlib.Path(sys.argv[1]).resolve(); target=pathlib.Path(sys.argv[2]).resolve()
with zipfile.ZipFile(target, 'w', compression=zipfile.ZIP_DEFLATED) as z:
    prefix=root.name
    for p in sorted(root.rglob('*')):
        if p.is_file(): z.write(p, pathlib.PurePosixPath(prefix, p.relative_to(root).as_posix()))
PY
fi
python3 - "$archive" <<'PY' > "$archive.sha256"
import hashlib, pathlib, sys
p=pathlib.Path(sys.argv[1]); print(hashlib.sha256(p.read_bytes()).hexdigest()+"  "+p.name)
PY

echo "release ready: $archive"
echo "checksum: $archive.sha256"
