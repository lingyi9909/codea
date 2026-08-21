#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)

required=(
  packaging/config/release.yaml
  packaging/scripts/build-runtime.sh
  packaging/scripts/build-plugins.sh
  packaging/scripts/collect-skills.sh
  packaging/scripts/generate-manifest.sh
  packaging/scripts/verify-checksum.sh
  packaging/scripts/verify-offline.sh
  packaging/platform/macos/install.sh
  packaging/platform/windows/install.ps1
)
for rel in "${required[@]}"; do
  test -f "$repo_root/$rel" || { echo "missing $rel" >&2; exit 1; }
done

grep -q 'openCodeVersion: "1.18.11"' "$repo_root/packaging/config/release.yaml"
grep -q 'darwin-arm64' "$repo_root/packaging/config/release.yaml"
grep -q 'darwin-x64' "$repo_root/packaging/config/release.yaml"
grep -q 'windows-x64' "$repo_root/packaging/config/release.yaml"

stage=$(mktemp -d)
trap 'rm -rf "$stage"' EXIT
mkdir -p "$stage/plugins" "$stage/bin" "$stage/config" "$stage/skills"
printf 'export default {};\n' > "$stage/plugins/index.js"
printf '#!/bin/sh\nexit 0\n' > "$stage/bin/codea"
printf '#!/bin/sh\nexit 0\n' > "$stage/bin/opencode"
chmod +x "$stage/bin/codea" "$stage/bin/opencode"

"$repo_root/packaging/scripts/generate-manifest.sh" "$stage" >/dev/null
"$repo_root/packaging/scripts/verify-checksum.sh" "$stage" >/dev/null
"$repo_root/packaging/scripts/verify-offline.sh" "$stage" >/dev/null

touch "$stage/plugins/package.json"
if "$repo_root/packaging/scripts/verify-offline.sh" "$stage" >/dev/null 2>&1; then
  echo "verify-offline must reject package.json in plugin dist" >&2
  exit 1
fi
rm -f "$stage/plugins/package.json"

printf 'import x from "left-pad"; export default x;\n' > "$stage/plugins/index.js"
if "$repo_root/packaging/scripts/verify-offline.sh" "$stage" >/dev/null 2>&1; then
  echo "verify-offline must reject external imports" >&2
  exit 1
fi

echo "task17 packaging contracts passed"
