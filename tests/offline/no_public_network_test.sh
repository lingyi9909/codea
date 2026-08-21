#!/usr/bin/env bash
set -euo pipefail

stage=${1:?usage: no_public_network_test.sh <staging-dir>}
repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)

# This test must run inside the release isolation environment. It fails closed
# if public HTTPS is reachable; it does not mutate host firewall rules itself.
if command -v curl >/dev/null 2>&1; then
  if curl -fsS --connect-timeout 2 --max-time 3 https://github.com/ >/dev/null 2>&1; then
    echo "FAIL: public HTTPS is reachable; run this gate in the isolated/offline environment" >&2
    exit 1
  fi
fi

"$repo_root/packaging/scripts/verify-offline.sh" "$stage"

runtime="$stage/bin/opencode"
[ -x "$runtime" ] || { echo "FAIL: packaged OpenCode binary missing/not executable" >&2; exit 1; }
export OPENCODE_DISABLE_MODELS_FETCH=1
export OPENCODE_DISABLE_AUTOUPDATE=1
export OPENCODE_DISABLE_EMBEDDED_WEB_UI=1
export OPENCODE_DISABLE_LSP_DOWNLOAD=1
export OPENCODE_DISABLE_DEFAULT_PLUGINS=1

if command -v timeout >/dev/null 2>&1; then
  version=$(timeout 10 "$runtime" --version 2>&1)
else
  version=$("$runtime" --version 2>&1)
fi
printf '%s\n' "$version" | grep -q '1.18.11' || { echo "FAIL: packaged runtime version mismatch: $version" >&2; exit 1; }

echo "offline runtime gate passed: public HTTPS blocked, static package clean, OpenCode 1.18.11 executable"
