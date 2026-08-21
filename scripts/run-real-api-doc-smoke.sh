#!/usr/bin/env bash
set -euo pipefail

# Task 16 functional E2E. The deterministic model only selects tools; the
# OpenCode v1.18.11 runtime, Codea agent materialization, custom tools,
# permission enforcement, SSE, extraction/validation and document write are real.
repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
fixture_dir="$repo_root/tests/fixtures/real-parity"
evidence_file="$repo_root/tui/tests/parity/evidence/api-doc-agent-evidence.json"
opencode_bin=${OPENCODE_BIN:-}
smoke_port=${SMOKE_PORT:-49324}
fake_port=${FAKE_MODEL_PORT:-49221}
username=${OPENCODE_SERVER_USERNAME:-opencode}
password=${OPENCODE_SERVER_PASSWORD:-testpass123}

run_root=$(mktemp -d "${TMPDIR:-/tmp}/codea-real-api-doc.XXXXXX")
run_root=$(cd "$run_root" && pwd -P)
fake_pid=""
opencode_pid=""
cleanup() {
  if [ -n "$opencode_pid" ]; then kill "$opencode_pid" 2>/dev/null || true; wait "$opencode_pid" 2>/dev/null || true; fi
  if [ -n "$fake_pid" ]; then kill "$fake_pid" 2>/dev/null || true; wait "$fake_pid" 2>/dev/null || true; fi
  rm -rf "$run_root"
}
trap cleanup EXIT INT TERM

[ -n "$opencode_bin" ] && [ -x "$opencode_bin" ] || { echo "OPENCODE_BIN must point to executable OpenCode v1.18.11" >&2; exit 2; }
for cmd in python3 curl git; do command -v "$cmd" >/dev/null 2>&1 || { echo "$cmd is required" >&2; exit 2; }; done
[ -f "$fixture_dir/fake_api_doc_model.py" ] || { echo "fake_api_doc_model.py missing" >&2; exit 2; }
[ -f "$fixture_dir/opencode.json" ] || { echo "opencode.json missing" >&2; exit 2; }

config_dir="$run_root/config"
smoke_dir="$run_root/smoke-dir"
mkdir -p "$config_dir" "$smoke_dir"
bundle_abs="$(cd "$repo_root/distribution/plugins" && pwd)/dist/index.js"
[ -f "$bundle_abs" ] || { echo "plugin bundle missing at $bundle_abs; run bun run build first" >&2; exit 2; }

python3 - "$fixture_dir/opencode.json" "$config_dir/opencode.json" "$bundle_abs" "$fake_port" <<'PY'
import json, pathlib, sys
cfg=json.loads(pathlib.Path(sys.argv[1]).read_text())
cfg["plugin"]=["file://" + sys.argv[3]]
provider=cfg["provider"]["codea-parity"]
provider["options"]["baseURL"]="http://127.0.0.1:%s/v1" % sys.argv[4]
pathlib.Path(sys.argv[2]).write_text(json.dumps(cfg, indent=2))
PY

maven_fixture="$repo_root/tui/tests/e2e/fixtures/java-maven-project"
[ -f "$maven_fixture/pom.xml" ] || { echo "java-maven-project fixture missing" >&2; exit 2; }
cp -R "$maven_fixture/." "$smoke_dir/"
printf 'hello read me\n' > "$smoke_dir/read-me.txt"
(
  cd "$smoke_dir"
  git init -q
  git config user.email smoke@codea.local
  git config user.name smoke
  git add -A
  git commit -qm "api-doc fixture baseline"
)

(
  cd "$repo_root/tui"
  CODEA_AGENT_SYNC_ROOT="$repo_root/distribution/agents" \
  CODEA_AGENT_SYNC_DIR="$config_dir/agents" \
  GOTOOLCHAIN=local go test ./internal/agent/ -run '^TestMaterializeToConfigDir$' -count=1
)

SMOKE_DIR="$smoke_dir" FAKE_MODEL_PORT="$fake_port" \
  python3 "$fixture_dir/fake_api_doc_model.py" >"$run_root/fake-model.log" 2>&1 &
fake_pid=$!
ready=0
for _ in $(seq 1 50); do
  if ! kill -0 "$fake_pid" 2>/dev/null; then cat "$run_root/fake-model.log" >&2; exit 1; fi
  if curl -fsS --max-time 1 "http://127.0.0.1:$fake_port/v1/models" >/dev/null 2>&1; then ready=1; break; fi
  sleep 0.2
done
[ "$ready" -eq 1 ] || { echo "fake api-doc model did not become ready" >&2; exit 1; }

(
  cd "$smoke_dir"
  export OPENCODE_CONFIG_DIR="$config_dir"
  export OPENCODE_SERVER_USERNAME="$username"
  export OPENCODE_SERVER_PASSWORD="$password"
  export CODEA_PARITY_API_KEY="real-api-doc-smoke-key"
  export OPENCODE_DISABLE_MODELS_FETCH=1
  export OPENCODE_DISABLE_AUTOUPDATE=1
  export OPENCODE_DISABLE_EMBEDDED_WEB_UI=1
  export OPENCODE_DISABLE_LSP_DOWNLOAD=1
  export OPENCODE_DISABLE_DEFAULT_PLUGINS=1
  "$opencode_bin" serve --hostname 127.0.0.1 --port "$smoke_port"
) >"$run_root/opencode.log" 2>&1 &
opencode_pid=$!

health_file="$run_root/health.json"
healthy=0
for _ in $(seq 1 150); do
  if ! kill -0 "$opencode_pid" 2>/dev/null; then cat "$run_root/opencode.log" >&2; exit 1; fi
  if curl -fsS --max-time 1 -u "$username:$password" "http://127.0.0.1:$smoke_port/global/health" >"$health_file" 2>/dev/null; then healthy=1; break; fi
  sleep 0.2
done
[ "$healthy" -eq 1 ] || { echo "OpenCode did not become healthy" >&2; cat "$run_root/opencode.log" >&2; exit 1; }
python3 - "$health_file" <<'PY'
import json, pathlib, sys
p=json.loads(pathlib.Path(sys.argv[1]).read_text())
if p.get("healthy") is not True or p.get("version") != "1.18.11": raise SystemExit(f"unexpected health: {p}")
PY

cd "$repo_root/tui"
OPENCODE_ENDPOINT="http://127.0.0.1:$smoke_port" \
OPENCODE_SERVER_USERNAME="$username" \
OPENCODE_SERVER_PASSWORD="$password" \
SMOKE_DIR="$smoke_dir" \
GOTOOLCHAIN=local go test ./tests/parity/ -run '^TestRealAPIDocEvidence$' -count=1 -v

python3 - "$evidence_file" <<'PY'
import json, pathlib, sys
path=pathlib.Path(sys.argv[1])
if not path.exists(): raise SystemExit(f"evidence missing: {path}")
p=json.loads(path.read_text())
if p.get("available") is not True or p.get("version") != "1.18.11": raise SystemExit(f"invalid runtime evidence: {p}")
if p.get("failedChecks") != 0 or p.get("totalChecks") != 9 or p.get("passedChecks") != 9: raise SystemExit(f"workflow evidence not 9/9: {p}")
for key in ("workflowExtractSucceeded","workflowValidateSucceeded","workflowWriteSucceeded","workflowDocumentValid"):
    if p.get(key) is not True: raise SystemExit(f"{key} != true")
print("api-doc-agent-evidence.json: 9/9 PASS")
PY

echo "[PASS] Task 16 real API Documentation workflow: extract -> validate -> write -> Markdown"
