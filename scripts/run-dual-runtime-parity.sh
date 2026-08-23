#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
opencode_bin=${OPENCODE_BIN:-}
evidence_file=${CODEA_PARITY_EVIDENCE:-"$repo_root/tui/tests/parity/evidence/release-parity.json"}
fake_port=${FAKE_MODEL_PORT:-49240}
baseline_port=${BASELINE_PORT:-49340}
candidate_port=${CANDIDATE_PORT:-49341}
username=${OPENCODE_SERVER_USERNAME:-codea-parity}
password=${OPENCODE_SERVER_PASSWORD:-codea-parity-pass}

if [ -z "$opencode_bin" ] || [ ! -x "$opencode_bin" ]; then
  echo "OPENCODE_BIN must point to executable OpenCode v1.18.11" >&2
  exit 2
fi
version=$($opencode_bin --version 2>&1)
printf '%s\n' "$version" | grep -q '1.18.11' || { echo "OpenCode version mismatch: $version" >&2; exit 1; }

fixture="$repo_root/tests/fixtures/release-parity/fake_model.py"
plugin="$repo_root/distribution/plugins/dist/index.js"
[ -f "$fixture" ] || { echo "release parity fake model missing" >&2; exit 2; }
[ -f "$plugin" ] || { echo "candidate Codea plugin bundle missing; build distribution/plugins first" >&2; exit 2; }
for cmd in python3 curl; do command -v "$cmd" >/dev/null 2>&1 || { echo "$cmd is required" >&2; exit 2; }; done

run_root=$(mktemp -d "${TMPDIR:-/tmp}/codea-dual-parity.XXXXXX")
run_root=$(cd "$run_root" && pwd -P)
smoke_dir="$run_root/workspace"
baseline_cfg="$run_root/baseline-config"
candidate_cfg="$run_root/candidate-config"
mkdir -p "$smoke_dir" "$baseline_cfg" "$candidate_cfg" "$(dirname "$evidence_file")"
printf 'release parity read\n' > "$smoke_dir/read-me.txt"

fake_pid=""; baseline_pid=""; candidate_pid=""
cleanup() {
  for pid in "$candidate_pid" "$baseline_pid" "$fake_pid"; do
    if [ -n "$pid" ]; then kill "$pid" 2>/dev/null || true; wait "$pid" 2>/dev/null || true; fi
  done
  rm -rf "$run_root"
}
trap cleanup EXIT INT TERM

python3 - "$baseline_cfg/opencode.json" "$candidate_cfg/opencode.json" "$fake_port" "$plugin" <<'PY'
import copy, json, pathlib, sys
base_out=pathlib.Path(sys.argv[1]); cand_out=pathlib.Path(sys.argv[2]); port=int(sys.argv[3]); plugin=pathlib.Path(sys.argv[4]).resolve()
base={
  "$schema":"https://opencode.ai/config.json",
  "model":"codea-release-parity/release-parity",
  "small_model":"codea-release-parity/release-parity",
  "enabled_providers":["codea-release-parity"],
  "permission":{"bash":"ask"},
  "provider":{
    "codea-release-parity":{
      "npm":"@ai-sdk/openai-compatible",
      "name":"Codea Release Parity Local Provider",
      "options":{"baseURL":f"http://127.0.0.1:{port}/v1","apiKey":"{env:CODEA_PARITY_API_KEY}"},
      "models":{"release-parity":{"name":"Release Parity Fake Model","limit":{"context":32768,"output":4096}}}
    }
  }
}
candidate=copy.deepcopy(base)
candidate["plugin"]=[plugin.as_uri()]
base_out.write_text(json.dumps(base, indent=2)+"\n")
cand_out.write_text(json.dumps(candidate, indent=2)+"\n")
PY

SMOKE_DIR="$smoke_dir" FAKE_MODEL_PORT="$fake_port" python3 "$fixture" >"$run_root/fake.log" 2>&1 &
fake_pid=$!
for _ in $(seq 1 80); do
  if ! kill -0 "$fake_pid" 2>/dev/null; then cat "$run_root/fake.log" >&2; exit 1; fi
  if curl -fsS --max-time 1 "http://127.0.0.1:$fake_port/v1/models" >/dev/null 2>&1; then break; fi
  sleep 0.1
done
curl -fsS --max-time 1 "http://127.0.0.1:$fake_port/v1/models" >/dev/null || { echo "fake model not ready" >&2; exit 1; }

start_runtime() {
  local cfg=$1 port=$2 log=$3
  (
    cd "$smoke_dir"
    export OPENCODE_CONFIG_DIR="$cfg"
    export OPENCODE_SERVER_USERNAME="$username"
    export OPENCODE_SERVER_PASSWORD="$password"
    export CODEA_PARITY_API_KEY=release-parity-key
    export OPENCODE_DISABLE_MODELS_FETCH=1
    export OPENCODE_DISABLE_AUTOUPDATE=1
    export OPENCODE_DISABLE_EMBEDDED_WEB_UI=1
    export OPENCODE_DISABLE_LSP_DOWNLOAD=1
    export OPENCODE_DISABLE_DEFAULT_PLUGINS=1
    "$opencode_bin" serve --hostname 127.0.0.1 --port "$port"
  ) >"$log" 2>&1 &
  echo $!
}

baseline_pid=$(start_runtime "$baseline_cfg" "$baseline_port" "$run_root/baseline.log")
candidate_pid=$(start_runtime "$candidate_cfg" "$candidate_port" "$run_root/candidate.log")

wait_runtime() {
  local port=$1 pid=$2 log=$3 label=$4
  for _ in $(seq 1 160); do
    if ! kill -0 "$pid" 2>/dev/null; then echo "$label runtime exited" >&2; cat "$log" >&2; exit 1; fi
    if curl -fsS --max-time 1 -u "$username:$password" "http://127.0.0.1:$port/global/health" >"$run_root/$label-health.json" 2>/dev/null; then
      python3 - "$run_root/$label-health.json" <<'PY'
import json, pathlib, sys
p=json.loads(pathlib.Path(sys.argv[1]).read_text())
if p.get('healthy') is not True or p.get('version') != '1.18.11': raise SystemExit(f"bad health: {p}")
PY
      return
    fi
    sleep 0.15
  done
  echo "$label runtime did not become healthy" >&2; cat "$log" >&2; exit 1
}
wait_runtime "$baseline_port" "$baseline_pid" "$run_root/baseline.log" baseline
wait_runtime "$candidate_port" "$candidate_pid" "$run_root/candidate.log" candidate

cd "$repo_root/tui"
CODEA_PARITY_BASELINE_URL="http://127.0.0.1:$baseline_port" \
CODEA_PARITY_CANDIDATE_URL="http://127.0.0.1:$candidate_port" \
CODEA_PARITY_BASELINE_USERNAME="$username" \
CODEA_PARITY_BASELINE_PASSWORD="$password" \
CODEA_PARITY_CANDIDATE_USERNAME="$username" \
CODEA_PARITY_CANDIDATE_PASSWORD="$password" \
CODEA_PARITY_EVIDENCE="$evidence_file" \
GOTOOLCHAIN=local go run ./cmd/parity-runner

python3 - "$evidence_file" <<'PY'
import json, pathlib, sys
p=json.loads(pathlib.Path(sys.argv[1]).read_text())
if p.get('passed') is not True: raise SystemExit(f"parity evidence not passed: {p}")
r=p.get('result') or {}
if r.get('RequiredFailed', r.get('requiredFailed', 1)) != 0: raise SystemExit(f"required parity failures: {r}")
if not p.get('baselineUrl') or not p.get('candidateUrl') or p['baselineUrl']==p['candidateUrl']:
    raise SystemExit('baseline/candidate endpoints were not distinct')
print(f"dual runtime parity green: {r.get('Passed', r.get('passed'))}/{r.get('Total', r.get('total'))} scenarios")
PY

echo "[PASS] distinct baseline/candidate OpenCode v1.18.11 parity"
