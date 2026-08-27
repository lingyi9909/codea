#!/usr/bin/env bash
set -euo pipefail

# run-real-parity-smoke.sh
#
# Exercises the real locked OpenCode v1.18.11 runtime through the Codea
# OpenCodeAdapter, driving every native General Agent tool (read/write/edit/
# bash/subagent/skill/plugin) plus real approval once/reject and cancel-in-
# streaming. Only the model is a deterministic stub (fake_model.py); the
# runtime, agent loop, permission gating, SSE and persistence are all real.
#
# On success it writes fresh evidence to
#   tui/tests/parity/evidence/runtime-evidence.json
# with available=true and failedChecks=0, and exits 0. On any failure it exits
# non-zero so the parity gate cannot be satisfied by a silent skip.

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
fixture_dir="$repo_root/tests/fixtures/real-parity"
evidence_file="$repo_root/tui/tests/parity/evidence/runtime-evidence.json"

opencode_bin=${OPENCODE_BIN:-}
smoke_port=${SMOKE_PORT:-49321}
fake_port=${FAKE_MODEL_PORT:-49220}
username=${OPENCODE_SERVER_USERNAME:-opencode}
password=${OPENCODE_SERVER_PASSWORD:-testpass123}
cleanup_term_grace_seconds=${CLEANUP_TERM_GRACE_SECONDS:-3}

if ! [[ "$cleanup_term_grace_seconds" =~ ^[0-9]+$ ]]; then
  echo "CLEANUP_TERM_GRACE_SECONDS must be a non-negative integer" >&2
  exit 2
fi

run_root=$(mktemp -d "${TMPDIR:-/tmp}/codea-real-parity.XXXXXX")
# Resolve the temp dir to its canonical path. On macOS both /tmp and /var are
# symlinks (/tmp -> /private/tmp, /var -> /private/var); OpenCode resolves the
# session directory to the canonical path but the fake model builds file paths
# from the unresolved SMOKE_DIR. If they disagree, OpenCode treats the read/
# write/edit target as an external_directory and asks for permission, stalling
# the tool at "running". Canonicalising here keeps the two paths identical.
run_root=$(cd "$run_root" && pwd -P)
fake_pid=""
opencode_pid=""

terminate_pid() {
  local pid=${1:-}
  local label=${2:-process}
  local ticks=$((cleanup_term_grace_seconds * 10))

  [ -n "$pid" ] || return 0

  if ! kill -0 "$pid" 2>/dev/null; then
    wait "$pid" 2>/dev/null || true
    return 0
  fi

  kill "$pid" 2>/dev/null || true
  for ((i=0; i<ticks; i++)); do
    if ! kill -0 "$pid" 2>/dev/null; then
      wait "$pid" 2>/dev/null || true
      return 0
    fi
    sleep 0.1
  done

  if kill -0 "$pid" 2>/dev/null; then
    echo "cleanup: $label pid=$pid did not exit after ${cleanup_term_grace_seconds}s; forcing SIGKILL" >&2
    kill -KILL "$pid" 2>/dev/null || true
  fi

  # Do not perform an unconditional wait here. cancel-in-streaming can leave a
  # runtime in a shutdown state where wait blocks indefinitely; after SIGKILL
  # the parent shell must be allowed to exit so CI remains bounded.
  for ((i=0; i<20; i++)); do
    if ! kill -0 "$pid" 2>/dev/null; then
      wait "$pid" 2>/dev/null || true
      return 0
    fi
    sleep 0.05
  done
  echo "cleanup: $label pid=$pid still present after SIGKILL; continuing without wait" >&2
}

cleanup() {
  terminate_pid "$opencode_pid" "OpenCode"
  terminate_pid "$fake_pid" "fake model"
  find "$run_root" -type f -delete 2>/dev/null || true
  find "$run_root" -depth -type d -exec rmdir {} \; 2>/dev/null || true
}
trap cleanup EXIT INT TERM

if [ -z "$opencode_bin" ] || [ ! -x "$opencode_bin" ]; then
  echo "OPENCODE_BIN must point to an executable OpenCode v1.18.11 binary" >&2
  exit 2
fi
if [ ! -f "$fixture_dir/fake_model.py" ] || [ ! -f "$fixture_dir/opencode.json" ]; then
  echo "real-parity fixture missing under $fixture_dir" >&2
  exit 2
fi
for cmd in python3 curl; do
  command -v "$cmd" >/dev/null 2>&1 || { echo "$cmd is required" >&2; exit 2; }
done

# --- Prepare isolated config dir + smoke dir ---------------------------------
config_dir="$run_root/config"
smoke_dir="$run_root/smoke-dir"
mkdir -p "$config_dir/skills/smoke-skill" "$smoke_dir"
cp "$fixture_dir/opencode.json" "$config_dir/opencode.json"
cp "$fixture_dir/skills/smoke-skill/SKILL.md" "$config_dir/skills/smoke-skill/SKILL.md"
printf 'hello read me\n' > "$smoke_dir/read-me.txt"
printf 'before\n' > "$smoke_dir/edit-me.txt"

# --- Start the deterministic fake model -------------------------------------
SMOKE_DIR="$smoke_dir" FAKE_MODEL_PORT="$fake_port" \
  python3 "$fixture_dir/fake_model.py" >"$run_root/fake-model.log" 2>&1 &
fake_pid=$!

fake_ready=0
for _ in $(seq 1 50); do
  if ! kill -0 "$fake_pid" 2>/dev/null; then
    wait "$fake_pid" 2>/dev/null || true
    fake_pid=""
    echo "fake model exited before becoming ready" >&2
    cat "$run_root/fake-model.log" >&2
    exit 1
  fi
  if curl -fsS --max-time 1 "http://127.0.0.1:$fake_port/v1/models" >/dev/null 2>&1; then
    fake_ready=1
    break
  fi
  sleep 0.2
done
if [ "$fake_ready" -ne 1 ]; then
  echo "fake model did not become ready" >&2
  exit 1
fi

# --- Start the real OpenCode runtime ----------------------------------------
(
  cd "$smoke_dir"
  export OPENCODE_CONFIG_DIR="$config_dir"
  export OPENCODE_SERVER_USERNAME="$username"
  export OPENCODE_SERVER_PASSWORD="$password"
  export CODEA_PARITY_API_KEY="real-parity-smoke-key"
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

try:
    payload = json.loads(pathlib.Path(sys.argv[1]).read_text())
except Exception as exc:
    raise SystemExit(f"invalid health JSON: {exc}")
if payload.get("healthy") is not True or payload.get("version") != "1.18.11":
    raise SystemExit(f"unexpected health payload: {payload}")
print(f"OpenCode healthy: version={payload['version']}")
PY

# --- Run the real-parity smoke through the Go adapter -----------------------
cd "$repo_root/tui"
OPENCODE_ENDPOINT="http://127.0.0.1:$smoke_port" \
OPENCODE_SERVER_USERNAME="$username" \
OPENCODE_SERVER_PASSWORD="$password" \
SMOKE_DIR="$smoke_dir" \
GOTOOLCHAIN=local go test ./tests/parity/ -run '^TestRealRuntimeEvidence$' -count=1 -v

# --- Assert fresh evidence is green -----------------------------------------
python3 - "$evidence_file" <<'PY'
import json
import pathlib
import sys

path = pathlib.Path(sys.argv[1])
if not path.exists():
    raise SystemExit(f"evidence artifact missing: {path}")
payload = json.loads(path.read_text())
if payload.get("available") is not True:
    raise SystemExit(f"evidence available != true: {payload.get('available')}")
if payload.get("failedChecks", 1) != 0:
    raise SystemExit(f"evidence failedChecks != 0: {payload.get('failedChecks')}")
if payload.get("totalChecks", 0) <= 0 or payload.get("passedChecks", 0) != payload.get("totalChecks"):
    raise SystemExit(f"evidence not fully green: total={payload.get('totalChecks')} passed={payload.get('passedChecks')}")
print(f"evidence green: available=true totalChecks={payload['totalChecks']} passedChecks={payload['passedChecks']} failedChecks=0")
print(f"  version={payload.get('version')} timestamp={payload.get('timestamp')}")
PY

echo "[PASS] real OpenCode parity smoke: all native capability + approval + cancel gates green"
