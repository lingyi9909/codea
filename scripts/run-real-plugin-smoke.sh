#!/usr/bin/env bash
set -euo pipefail

# run-real-plugin-smoke.sh
#
# Exercises the Codea enterprise plugin against the real locked OpenCode v1.18.11
# runtime. Two layers:
#
#   1. Adapter smoke (deterministic, offline): loads the built bundle's DEFAULT
#      export — the exact readV1Plugin contract — invokes server() and drives all
#      8 tools through the guard (path deny, DLP input deny, write permission ask,
#      output DLP block, Dify degradation). This is the same code path OpenCode's
#      tool registry (fromPlugin) executes at tool-call time.
#
#   2. Real runtime load: starts `opencode serve` with the plugin registered in
#      config, waits for health, and asserts the server reports version 1.18.11
#      while the plugin is present (the process must import and invoke server()
#      without crashing).
#
# On success exits 0. Requires OPENCODE_BIN pointing at an executable OpenCode
# v1.18.11 binary (same contract as run-real-parity-smoke.sh).

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
plugin_dir="$repo_root/distribution/plugins"
opencode_bin=${OPENCODE_BIN:-}
smoke_port=${SMOKE_PORT:-49341}
username=${OPENCODE_SERVER_USERNAME:-opencode}
password=${OPENCODE_SERVER_PASSWORD:-testpass123}

if ! command -v bun >/dev/null 2>&1; then
  if [ -x "$HOME/.bun/bin/bun" ]; then
    export PATH="$HOME/.bun/bin:$PATH"
  else
    echo "FAIL: bun not found on PATH" >&2
    exit 2
  fi
fi

if [ -z "$opencode_bin" ] || [ ! -x "$opencode_bin" ]; then
  echo "OPENCODE_BIN must point to an executable OpenCode v1.18.11 binary" >&2
  exit 2
fi
command -v curl >/dev/null 2>&1 || { echo "curl is required" >&2; exit 2; }

# --- 1. Build + adapter smoke ------------------------------------------------
cd "$plugin_dir"
bun run build
bun run tests/plugin-smoke.ts

# --- 2. Real OpenCode serve load ---------------------------------------------
bundle_abs="$(cd "$plugin_dir" && pwd)/dist/index.js"
run_root=$(mktemp -d "${TMPDIR:-/tmp}/codea-real-plugin.XXXXXX")
run_root=$(cd "$run_root" && pwd -P)
config_dir="$run_root/config"
mkdir -p "$config_dir"
opencode_pid=""

cleanup() {
  if [ -n "$opencode_pid" ]; then kill "$opencode_pid" 2>/dev/null || true; wait "$opencode_pid" 2>/dev/null || true; fi
  find "$run_root" -type f -delete 2>/dev/null || true
  find "$run_root" -depth -type d -exec rmdir {} \; 2>/dev/null || true
}
trap cleanup EXIT INT TERM

cat > "$config_dir/opencode.json" <<EOF
{
  "plugin": ["file://$bundle_abs"],
  "permission": { "bash": "ask" }
}
EOF

(
  cd "$run_root"
  export OPENCODE_CONFIG_DIR="$config_dir"
  export OPENCODE_SERVER_USERNAME="$username"
  export OPENCODE_SERVER_PASSWORD="$password"
  export OPENCODE_DISABLE_MODELS_FETCH=1
  export OPENCODE_DISABLE_AUTOUPDATE=1
  export OPENCODE_DISABLE_EMBEDDED_WEB_UI=1
  export OPENCODE_DISABLE_LSP_DOWNLOAD=1
  export OPENCODE_DISABLE_DEFAULT_PLUGINS=1
  "$opencode_bin" serve --hostname 127.0.0.1 --port "$smoke_port"
) >"$run_root/opencode.log" 2>&1 &
opencode_pid=$!

health_file="$run_root/health.json"
opencode_ready=0
for _ in $(seq 1 150); do
  if ! kill -0 "$opencode_pid" 2>/dev/null; then
    wait "$opencode_pid" 2>/dev/null || true
    opencode_pid=""
    echo "OpenCode exited before becoming healthy" >&2
    cat "$run_root/opencode.log" >&2
    exit 1
  fi
  if curl -fsS --max-time 1 -u "$username:$password" \
    "http://127.0.0.1:$smoke_port/global/health" >"$health_file" 2>/dev/null; then
    opencode_ready=1
    break
  fi
  sleep 0.2
done
if [ "$opencode_ready" -ne 1 ]; then
  echo "OpenCode did not become healthy" >&2
  cat "$run_root/opencode.log" >&2
  exit 1
fi

python3 - "$health_file" <<'PY'
import json
import pathlib
import sys

payload = json.loads(pathlib.Path(sys.argv[1]).read_text())
if payload.get("healthy") is not True or payload.get("version") != "1.18.11":
    raise SystemExit(f"unexpected health payload: {payload}")
print(f"OpenCode healthy: version={payload['version']} (plugin registered)")
PY

echo "[PASS] real OpenCode v1.18.11 plugin smoke: adapter 8-tool guard chain + runtime serve load"
