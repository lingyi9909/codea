#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
opencode_bin=${OPENCODE_BIN:-}
fake_port=${FAKE_MODEL_PORT:-49225}
evidence_file=${CODEA_CANDIDATE_EVIDENCE:-$repo_root/tui/tests/e2e/update-doctor/evidence/candidate-doctor-evidence.json}

[ -n "$opencode_bin" ] && [ -x "$opencode_bin" ] || { echo "OPENCODE_BIN must point to executable OpenCode v1.18.11" >&2; exit 2; }
for cmd in python3 curl; do command -v "$cmd" >/dev/null 2>&1 || { echo "$cmd is required" >&2; exit 2; }; done

run_root=$(mktemp -d "${TMPDIR:-/tmp}/codea-candidate-doctor.XXXXXX")
fake_pid=""
cleanup() {
  if [ -n "$fake_pid" ]; then kill "$fake_pid" 2>/dev/null || true; wait "$fake_pid" 2>/dev/null || true; fi
  rm -rf "$run_root"
}
trap cleanup EXIT INT TERM

candidate="$run_root/candidate"
config_dir="$run_root/config-c2-temp"
mkdir -p "$candidate/bin" "$config_dir/codea"

cd "$repo_root/tui"
GOTOOLCHAIN=local go build -trimpath -o "$candidate/bin/codea" ./cmd/codea
cp "$opencode_bin" "$candidate/bin/opencode"
chmod +x "$candidate/bin/codea" "$candidate/bin/opencode"
cp -R "$repo_root/distribution/agents" "$candidate/agents"
cp -R "$repo_root/distribution/skills" "$candidate/skills"
mkdir -p "$candidate/plugins" "$candidate/config/opencode"
cp "$repo_root/distribution/plugins/dist/index.js" "$candidate/plugins/index.js"
cp "$repo_root/distribution/config/opencode/permissions.json" "$candidate/config/opencode/permissions.json"
printf '0.1.0\n' > "$candidate/VERSION"
cp "$repo_root/runtime/version.json" "$candidate/runtime-version.json"

python3 - "$repo_root/tests/fixtures/real-parity/opencode.json" "$config_dir/opencode.json" "$fake_port" <<'PY'
import json, pathlib, sys
cfg=json.loads(pathlib.Path(sys.argv[1]).read_text())
cfg["provider"]["codea-parity"]["options"]["baseURL"]="http://127.0.0.1:%s/v1" % sys.argv[3]
pathlib.Path(sys.argv[2]).write_text(json.dumps(cfg, indent=2)+"\n")
PY
printf '{"schemaVersion":1}\n' > "$config_dir/codea/config.json"

"$repo_root/packaging/scripts/generate-manifest.sh" "$candidate" >/dev/null
"$repo_root/packaging/scripts/verify-checksum.sh" "$candidate" >/dev/null

FAKE_MODEL_PORT="$fake_port" SMOKE_DIR="$candidate" \
  python3 "$repo_root/tests/fixtures/real-parity/fake_model.py" >"$run_root/fake-model.log" 2>&1 &
fake_pid=$!
ready=0
for _ in $(seq 1 50); do
  if ! kill -0 "$fake_pid" 2>/dev/null; then cat "$run_root/fake-model.log" >&2; exit 1; fi
  if curl -fsS --max-time 1 "http://127.0.0.1:$fake_port/v1/models" >/dev/null 2>&1; then ready=1; break; fi
  sleep 0.2
done
[ "$ready" -eq 1 ] || { echo "fake model did not become ready" >&2; exit 1; }

rm -f "$evidence_file"
cd "$repo_root/tui"
CODEA_CANDIDATE_DIR="$candidate" \
CODEA_CANDIDATE_CONFIG_DIR="$config_dir" \
CODEA_CANDIDATE_EVIDENCE="$evidence_file" \
CODEA_PARITY_API_KEY="candidate-doctor-key" \
GOTOOLCHAIN=local go test ./tests/e2e/update-doctor -run '^TestRealCandidateDoctor$' -count=1 -v

test -s "$evidence_file" || { echo "candidate doctor evidence missing" >&2; exit 1; }
python3 - "$evidence_file" <<'PY'
import json, pathlib, sys
p=json.loads(pathlib.Path(sys.argv[1]).read_text())
if p.get("candidateDoctorPassed") is not True: raise SystemExit(p)
if p.get("openCodeVersion") != "1.18.11": raise SystemExit(p)
print("candidate-doctor-evidence: PASS")
PY

echo "[PASS] Task 18/19 candidate Doctor: V2 + C2-temp + real OpenCode v1.18.11"
