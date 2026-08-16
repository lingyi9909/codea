#!/usr/bin/env bash
set -euo pipefail

# run-skill-mode-smoke.sh
#
# Proves Task 11's two skill modes against the real locked OpenCode v1.18.11:
#   Smoke A (compatible): Codea + Project + User skills are ALL loadable — the
#                         isolated Codea config dir must not shadow native skills.
#   Smoke B (strict):     only the approved Codea skill is loadable; project,
#                         user and unapproved-Codea skills are isolated out via
#                         OPENCODE_DISABLE_EXTERNAL_SKILLS + PROJECT_CONFIG and
#                         materialization of approved-only skills.
#
# Exits 0 only when both profiles pass.

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
opencode_bin=${OPENCODE_BIN:-}
port=${PORT:-49552}
username=${OPENCODE_SERVER_USERNAME:-opencode}
password=${OPENCODE_SERVER_PASSWORD:-skill-mode-smoke}

run_root=$(mktemp -d "${TMPDIR:-/tmp}/codea-skill-mode.XXXXXX")
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

write_skill() { # $1=dir $2=name
  mkdir -p "$1/$2"
  cat > "$1/$2/SKILL.md" <<EOF
---
name: $2
description: smoke skill $2
---
EOF
}

start_server() { # $1=project_dir $2=config_dir $3=home $4=mode
  local project=$1 config=$2 home=$3 mode=$4
  (
    cd "$project"
    export HOME="$home"
    export OPENCODE_CONFIG_DIR="$config"
    export OPENCODE_SERVER_USERNAME="$username"
    export OPENCODE_SERVER_PASSWORD="$password"
    export OPENCODE_DISABLE_CLAUDE_CODE=1
    export OPENCODE_DISABLE_MODELS_FETCH=1
    export OPENCODE_DISABLE_AUTOUPDATE=1
    export OPENCODE_DISABLE_EMBEDDED_WEB_UI=1
    export OPENCODE_DISABLE_LSP_DOWNLOAD=1
    export OPENCODE_DISABLE_DEFAULT_PLUGINS=1
    if [ "$mode" = "strict" ]; then
      export OPENCODE_DISABLE_EXTERNAL_SKILLS=1
      export OPENCODE_DISABLE_PROJECT_CONFIG=1
      # OPENCODE_DISABLE_EXTERNAL_SKILLS does not disable the native user skills
      # dir (~/.config/opencode/skills); only redirecting XDG_CONFIG_HOME away
      # from ~/.config isolates it (S5/S6 spike). Mirrors supervisor buildEnv.
      export XDG_CONFIG_HOME="$config/xdg/config"
      export XDG_DATA_HOME="$config/xdg/data"
      export XDG_CACHE_HOME="$config/xdg/cache"
      export XDG_STATE_HOME="$config/xdg/state"
    fi
    "$opencode_bin" serve --hostname 127.0.0.1 --port "$port"
  ) >"$run_root/opencode-$mode.log" 2>&1 &
  server_pid=$!

  local ready=0
  for _ in $(seq 1 150); do
    if ! kill -0 "$server_pid" 2>/dev/null; then
      wait "$server_pid" 2>/dev/null || true
      server_pid=""
      echo "OpenCode exited before healthy ($mode)" >&2
      cat "$run_root/opencode-$mode.log" >&2
      exit 1
    fi
    if curl -fsS --max-time 1 -u "$username:$password" \
      "http://127.0.0.1:$port/global/health" >"$run_root/health-$mode.json" 2>/dev/null; then
      if python3 - "$run_root/health-$mode.json" <<'PY'
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
    echo "OpenCode did not become healthy ($mode)" >&2
    exit 1
  fi
}

stop_server() {
  if [ -n "$server_pid" ]; then
    kill "$server_pid" 2>/dev/null || true
    wait "$server_pid" 2>/dev/null || true
    server_pid=""
  fi
}

fetch_skills() { # $1=project_dir $2=outfile
  curl -fsS --max-time 10 -u "$username:$password" --get \
    --data-urlencode "directory=$1" \
    "http://127.0.0.1:$port/skill" > "$2"
}

# --- Smoke A: compatible ------------------------------------------------------
home="$run_root/homeA"
config="$home/.codea/runtime-config"
project="$run_root/projectA"
mkdir -p "$project" "$config/skills"
write_skill "$config/skills" "codea-skill"
write_skill "$home/.config/opencode/skills" "user-skill"
write_skill "$project/.opencode/skills" "project-skill"

start_server "$project" "$config" "$home" "compatible"
resp="$run_root/skill-compatible.json"
fetch_skills "$project" "$resp"
stop_server

python3 - "$resp" <<'PY'
import json, pathlib, sys
payload = json.loads(pathlib.Path(sys.argv[1]).read_text())
names = {item["name"] for item in payload}
required = {"codea-skill", "project-skill", "user-skill"}
missing = sorted(required - names)
if missing:
    raise SystemExit(f"compatible missing {missing}; got {sorted(names)}")
print(f"[PASS] compatible: Codea+Project+User all loadable ({sorted(names)})")
PY

# --- Smoke B: strict ----------------------------------------------------------
home="$run_root/homeB"
config="$home/.codea/runtime-config"
project="$run_root/projectB"
mkdir -p "$project" "$config/skills"
write_skill "$config/skills" "codea-approved"
# codea-unapproved exists only in a "distribution" dir OpenCode does not scan,
# mirroring how Codea's strict sync never materializes it.
write_skill "$run_root/distribution" "codea-unapproved"
write_skill "$home/.config/opencode/skills" "user-skill"
write_skill "$project/.opencode/skills" "project-skill"

start_server "$project" "$config" "$home" "strict"
resp="$run_root/skill-strict.json"
fetch_skills "$project" "$resp"
stop_server

python3 - "$resp" <<'PY'
import json, pathlib, sys
payload = json.loads(pathlib.Path(sys.argv[1]).read_text())
names = {item["name"] for item in payload}
if "codea-approved" not in names:
    raise SystemExit(f"strict missing approved skill; got {sorted(names)}")
forbidden = {"project-skill", "user-skill", "codea-unapproved"}
present = sorted(forbidden & names)
if present:
    raise SystemExit(f"strict leaked non-approved skills {present}; got {sorted(names)}")
print(f"[PASS] strict: only approved Codea loadable ({sorted(names)})")
PY

echo "[PASS] skill mode smoke: compatible + strict both verified"
