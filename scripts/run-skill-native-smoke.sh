#!/usr/bin/env bash
set -euo pipefail

# run-skill-native-smoke.sh
#
# Proves Codea's isolated OPENCODE_CONFIG_DIR does not shadow a user's native
# OpenCode skills. Under an isolated HOME it pre-seeds:
#   - a native user skill  at ~/.config/opencode/skills/native-user-skill
#   - a Codea skill        at ~/.codea/runtime-config/skills/codea-smoke
# then launches the real OpenCode v1.18.11 with the same environment Codea's
# supervisor sets (OPENCODE_CONFIG_DIR pointing at the isolated dir, and the Task
# 1 offline locks — crucially WITHOUT OPENCODE_DISABLE_EXTERNAL_SKILLS), and
# asserts GET /skill still reports both. If native-user-skill disappears, Codea
# would have swapped "deleting user files" for "hiding user skills".
#
# Exits 0 only when both skills are present.

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
opencode_bin=${OPENCODE_BIN:-}
port=${PORT:-49551}
username=${OPENCODE_SERVER_USERNAME:-opencode}
password=${OPENCODE_SERVER_PASSWORD:-native-smoke}

run_root=$(mktemp -d "${TMPDIR:-/tmp}/codea-skill-native.XXXXXX")
home="$run_root/home"
isolated="$home/.codea/runtime-config"
project="$run_root/project"
server_pid=""

cleanup() {
  if [ -n "$server_pid" ]; then
    kill "$server_pid" 2>/dev/null || true
    wait "$server_pid" 2>/dev/null || true
  fi
  rm -rf "$run_root"
}
trap cleanup EXIT INT TERM

if [ -z "$opencode_bin" ] || [ ! -x "$opencode_bin" ]; then
  echo "OPENCODE_BIN must point to an executable OpenCode v1.18.11 binary" >&2
  exit 2
fi
for cmd in python3 curl; do
  command -v "$cmd" >/dev/null 2>&1 || { echo "$cmd is required" >&2; exit 2; }
done

# --- Pre-seed native user skill + Codea skill ---------------------------------
mkdir -p "$home/.config/opencode/skills/native-user-skill"
mkdir -p "$isolated/skills/codea-smoke"
mkdir -p "$project"

cat > "$home/.config/opencode/skills/native-user-skill/SKILL.md" <<'EOF'
---
name: native-user-skill
description: native user skill
---
EOF

cat > "$isolated/skills/codea-smoke/SKILL.md" <<'EOF'
---
name: codea-smoke
description: codea smoke skill
---
EOF

# --- Start real OpenCode with Codea supervisor-equivalent env -----------------
(
  cd "$project"
  export HOME="$home"
  export OPENCODE_CONFIG_DIR="$isolated"
  export OPENCODE_SERVER_USERNAME="$username"
  export OPENCODE_SERVER_PASSWORD="$password"
  export OPENCODE_DISABLE_CLAUDE_CODE=1
  export OPENCODE_DISABLE_MODELS_FETCH=1
  export OPENCODE_DISABLE_AUTOUPDATE=1
  export OPENCODE_DISABLE_EMBEDDED_WEB_UI=1
  export OPENCODE_DISABLE_LSP_DOWNLOAD=1
  export OPENCODE_DISABLE_DEFAULT_PLUGINS=1
  # Deliberately NOT setting OPENCODE_DISABLE_EXTERNAL_SKILLS: that flag would
  # hide ~/.config/opencode/skills, which is exactly what this smoke proves is
  # still scanned.
  "$opencode_bin" serve --hostname 127.0.0.1 --port "$port"
) >"$run_root/opencode.log" 2>&1 &
server_pid=$!

# --- Wait for health ----------------------------------------------------------
health_file="$run_root/health.json"
ready=0
for _ in $(seq 1 150); do
  if ! kill -0 "$server_pid" 2>/dev/null; then
    wait "$server_pid" 2>/dev/null || true
    server_pid=""
    echo "OpenCode exited before becoming healthy" >&2
    cat "$run_root/opencode.log" >&2
    exit 1
  fi
  if curl -fsS --max-time 1 -u "$username:$password" \
    "http://127.0.0.1:$port/global/health" >"$health_file" 2>/dev/null; then
    if python3 - "$health_file" <<'PY'
import json, pathlib, sys
p = json.loads(pathlib.Path(sys.argv[1]).read_text())
raise SystemExit(0 if p.get("healthy") is True and p.get("version") == "1.18.11" else 1)
PY
    then
      ready=1
      break
    fi
  fi
  sleep 0.2
done
if [ "$ready" -ne 1 ]; then
  echo "OpenCode did not become healthy" >&2
  cat "$run_root/opencode.log" >&2
  exit 1
fi

# --- Assert /skill reports both ----------------------------------------------
response="$run_root/skill.json"
curl -fsS --max-time 10 -u "$username:$password" --get \
  --data-urlencode "directory=$project" \
  "http://127.0.0.1:$port/skill" >"$response"

python3 - "$response" <<'PY'
import json, pathlib, sys

payload = json.loads(pathlib.Path(sys.argv[1]).read_text())
if not isinstance(payload, list):
    raise SystemExit(f"/skill response must be an array: {sys.argv[1]}")
names = {item["name"] for item in payload}
required = {"codea-smoke", "native-user-skill"}
missing = sorted(required - names)
if missing:
    raise SystemExit(f"missing skills {missing}; got {sorted(names)}")
print(f"[PASS] isolated config keeps native user skill loadable: {sorted(names)}")
PY

echo "[PASS] skill native smoke: codea-smoke + native-user-skill both present in /skill"
