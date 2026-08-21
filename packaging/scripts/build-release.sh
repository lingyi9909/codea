#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
platform=${1:?usage: build-release.sh <darwin-arm64|darwin-x64|windows-x64> <output-dir>}
out_root=${2:?usage: build-release.sh <platform> <output-dir>}
case "$platform" in
  darwin-arm64) goos=darwin; goarch=arm64; codea_name=codea ;;
  darwin-x64) goos=darwin; goarch=amd64; codea_name=codea ;;
  windows-x64) goos=windows; goarch=amd64; codea_name=codea.exe ;;
  *) echo "unsupported V1 platform: $platform" >&2; exit 2 ;;
esac

stage="$out_root/codea-0.1.0-$platform"
rm -rf "$stage"
mkdir -p "$stage/bin" "$stage/plugins" "$stage/skills" "$stage/agents" "$stage/config"

# TUI is cross-compiled at build time. No Go toolchain is required on target.
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

"$repo_root/packaging/scripts/generate-manifest.sh" "$stage"
"$repo_root/packaging/scripts/verify-checksum.sh" "$stage"
"$repo_root/packaging/scripts/verify-offline.sh" "$stage"

echo "release staging ready: $stage"
