#!/usr/bin/env bash
set -euo pipefail

# run-real-agent-smoke.sh
#
# Task 14/15 real-runtime regression: code-reviewer + unit-test-generator are
# materialized into locked OpenCode v1.18.11, fail-closed permissions are
# enforced server-side, and Reviewer/UT workflows execute end-to-end.
# Task 16 has its own run-real-api-doc-smoke.sh so its deterministic model and
# evidence do not perturb the already accepted Task 14/15 15/15 baseline.

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
fixture_dir="$repo_root/tests/fixtures/real-parity"
evidence_file="$repo_root/tui/tests/parity/evidence/agent-evidence.json"

opencode_bin=${OPENCODE_BIN:-}
smoke_port=${SMOKE_PORT:-49322}
fake_port=${FAKE_MODEL_PORT:-49220}
username=${OPENCODE_SERVER_USERNAME:-opencode}
password=${OPENCODE_SERVER_PASSWORD:-testpass123}

run_root=$(mktemp -d "${TMPDIR:-/tmp}/codea-real-agent.XXXXXX")
run_root=$(cd "$run_root" && pwd -P)
fake_pid=""
opencode_pid=""

cleanup() {
  if [ -n "$opencode_pid" ]; then kill "$opencode_pid" 2>/dev/null || true; wait "$opencode_pid" 2>/dev/null || true; fi
  if [ -n "$fake_pid" ]; then kill "$fake_pid" 2>/dev/null || true; wait "$fake_pid" 2>/dev/null || true; fi
  rm -rf "$run_root"
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
for cmd in python3 curl git mvn; do
  command -v "$cmd" >/dev/null 2>&1 || { echo "$cmd is required" >&2; exit 2; }
done

config_dir="$run_root/config"
smoke_dir="$run_root/smoke-dir"
mkdir -p "$config_dir" "$smoke_dir"

bundle_abs="$(cd "$repo_root/distribution/plugins" && pwd)/dist/index.js"
if [ ! -f "$bundle_abs" ]; then
  echo "plugin bundle missing at $bundle_abs; run bun run build first" >&2
  exit 2
fi
python3 - "$fixture_dir/opencode.json" "$config_dir/opencode.json" "$bundle_abs" <<'PY'
import json
import pathlib
import sys
cfg = json.loads(pathlib.Path(sys.argv[1]).read_text())
cfg["plugin"] = ["file://" + sys.argv[3]]
pathlib.Path(sys.argv[2]).write_text(json.dumps(cfg, indent=2))
PY

maven_fixture="$repo_root/tui/tests/e2e/fixtures/java-maven-project"
if [ ! -f "$maven_fixture/pom.xml" ] || [ ! -f "$maven_fixture/mvnw" ]; then
  echo "java-maven-project fixture missing under $maven_fixture" >&2
  exit 2
fi
cp -R "$maven_fixture/." "$smoke_dir/"
rm -f "$smoke_dir/mvnw" "$smoke_dir/mvnw.cmd"
printf 'hello read me\n' > "$smoke_dir/read-me.txt"

(
  cd "$smoke_dir"
  git init -q
  git config user.email smoke@codea.local
  git config user.name smoke
  git add -A
  git commit -qm "fixture baseline"
  printf 'hello read me (v2)\n' > "$smoke_dir/read-me.txt"
  git add "$smoke_dir/read-me.txt"
)

(
  cd "$repo_root/tui"
  CODEA_AGENT_SYNC_ROOT="$repo_root/distribution/agents" \
  CODEA_AGENT_SYNC_DIR="$config_dir/agents" \
  GOTOOLCHAIN=local go test ./internal/agent/ -run '^TestMaterializeToConfigDir$' -count=1
)

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
  if curl -fsS --max-time 1 "http://127.0.0.1:$fake_port/v1/models" >/dev/null 2>&1; then fake_ready=1; break; fi
  sleep 0.2
done
[ "$fake_ready" -eq 1 ] || { echo "fake model did not become ready" >&2; exit 1; }

(
  cd "$smoke_dir"
  export OPENCODE_CONFIG_DIR="$config_dir"
  export OPENCODE_SERVER_USERNAME="$username"
  export OPENCODE_SERVER_PASSWORD="$password"
  export CODEA_PARITY_API_KEY="real-agent-smoke-key"
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
  if curl -fsS --max-time 1 -u "$username:$password" "http://127.0.0.1:$smoke_port/global/health" >"$health_file" 2>/dev/null; then opencode_ready=1; break; fi
  sleep 0.2
done
[ "$opencode_ready" -eq 1 ] || { echo "OpenCode did not become healthy" >&2; cat "$run_root/opencode.log" >&2; exit 1; }

python3 - "$health_file" <<'PY'
import json, pathlib, sys
payload=json.loads(pathlib.Path(sys.argv[1]).read_text())
if payload.get('healthy') is not True or payload.get('version') != '1.18.11':
    raise SystemExit(f'unexpected health payload: {payload}')
print(f"OpenCode healthy: version={payload['version']}")
PY

cd "$repo_root/tui"
OPENCODE_ENDPOINT="http://127.0.0.1:$smoke_port" \
OPENCODE_SERVER_USERNAME="$username" \
OPENCODE_SERVER_PASSWORD="$password" \
SMOKE_DIR="$smoke_dir" \
GOTOOLCHAIN=local go test ./tests/parity/ -run '^TestRealAgentEvidence$' -count=1 -v

python3 - "$evidence_file" <<'PY'
import json, pathlib, sys
path=pathlib.Path(sys.argv[1])
if not path.exists(): raise SystemExit(f'evidence artifact missing: {path}')
payload=json.loads(path.read_text())
if payload.get('available') is not True: raise SystemExit('available != true')
if payload.get('failedChecks', 1) != 0: raise SystemExit('failedChecks != 0')
if payload.get('totalChecks') != 15 or payload.get('passedChecks') != 15: raise SystemExit(f"expected 15/15, got {payload.get('passedChecks')}/{payload.get('totalChecks')}")
if payload.get('version') != '1.18.11': raise SystemExit('wrong runtime version')
print('agent-evidence.json: 15/15 PASS')
PY

echo "[PASS] Task 14/15 real OpenCode agent smoke: 15/15"
