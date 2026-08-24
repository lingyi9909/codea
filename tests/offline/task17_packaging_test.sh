#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)

required=(
  packaging/config/release.yaml
  packaging/scripts/build-runtime.sh
  packaging/scripts/build-plugins.sh
  packaging/scripts/collect-skills.sh
  packaging/scripts/build-release.sh
  packaging/scripts/generate-manifest.sh
  packaging/scripts/verify-checksum.sh
  packaging/scripts/verify-offline.sh
  packaging/platform/macos/install.sh
  packaging/platform/windows/install.ps1
  tests/offline/no_public_network_test.sh
  tests/offline/macos_release_smoke.sh
  tests/offline/windows_release_smoke.ps1
)
for rel in "${required[@]}"; do
  test -f "$repo_root/$rel" || { echo "missing $rel" >&2; exit 1; }
done

grep -q 'openCodeVersion: "1.18.11"' "$repo_root/packaging/config/release.yaml"
grep -q 'darwin-arm64' "$repo_root/packaging/config/release.yaml"
grep -q 'darwin-x64' "$repo_root/packaging/config/release.yaml"
grep -q 'windows-x64' "$repo_root/packaging/config/release.yaml"
grep -q 'go build' "$repo_root/packaging/scripts/build-release.sh"
grep -q 'GOTOOLCHAIN=local' "$repo_root/packaging/scripts/build-release.sh"
grep -q 'install/install.sh' "$repo_root/packaging/scripts/build-release.sh"
grep -q 'install/install.ps1' "$repo_root/packaging/scripts/build-release.sh"
grep -q 'archive.sha256' "$repo_root/packaging/scripts/build-release.sh"
grep -q 'Junction' "$repo_root/packaging/platform/windows/install.ps1"
grep -q 'UTF8Encoding($false)' "$repo_root/packaging/platform/windows/install.ps1"
grep -q 'CODEA_POINTER=.*current.txt' "$repo_root/packaging/platform/windows/install.ps1" || { echo "Windows launcher must resolve current.txt" >&2; exit 1; }
grep -q 'set /p "CODEA_CURRENT="' "$repo_root/packaging/platform/windows/install.ps1" || { echo "Windows launcher must read atomic current pointer" >&2; exit 1; }

# Final G2/G2.1 native smokes must prove more than packaged-file presence:
# package-manager processes are trapped, and the live locked Runtime must expose
# every enterprise plugin tool while public HTTPS is unavailable.
for smoke in tests/offline/macos_release_smoke.sh tests/offline/windows_release_smoke.ps1; do
  grep -q 'package-manager-invocations' "$repo_root/$smoke" || { echo "$smoke must trap package-manager invocation" >&2; exit 1; }
  grep -q 'experimental/tool/ids' "$repo_root/$smoke" || { echo "$smoke must query live plugin tool registry" >&2; exit 1; }
  grep -q 'collect_review_context' "$repo_root/$smoke" || { echo "$smoke must verify enterprise plugin tools" >&2; exit 1; }
done

# Installed launchers must point Codea at resources inside the selected version.
for key in OPENCODE_BIN CODEA_AGENTS_DIR CODEA_SKILLS_DIR CODEA_PLUGIN_BUNDLE; do
  grep -q "$key" "$repo_root/packaging/platform/macos/install.sh" || { echo "macOS launcher missing $key" >&2; exit 1; }
  grep -q "$key" "$repo_root/packaging/platform/windows/install.ps1" || { echo "Windows launcher missing $key" >&2; exit 1; }
done

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

printf 'tampered\n' >> "$stage/bin/codea"
if "$repo_root/packaging/scripts/verify-checksum.sh" "$stage" >/dev/null 2>&1; then
  echo "verify-checksum must reject tampered files" >&2
  exit 1
fi
printf '#!/bin/sh\nexit 0\n' > "$stage/bin/codea"
chmod +x "$stage/bin/codea"
"$repo_root/packaging/scripts/generate-manifest.sh" "$stage" >/dev/null

printf 'extra\n' > "$stage/config/unmanifested.txt"
if "$repo_root/packaging/scripts/verify-checksum.sh" "$stage" >/dev/null 2>&1; then
  echo "verify-checksum must reject unmanifested files" >&2
  exit 1
fi
rm -f "$stage/config/unmanifested.txt"

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

printf 'import "left-pad"; export default {};\n' > "$stage/plugins/index.js"
if "$repo_root/packaging/scripts/verify-offline.sh" "$stage" >/dev/null 2>&1; then
  echo "verify-offline must reject side-effect external imports" >&2
  exit 1
fi

echo "task17 packaging contracts passed"
