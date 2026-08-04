#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
fixture_source="$repo_root/tests/fixtures/skill-isolation"
opencode_bin=${OPENCODE_BIN:-}
output_dir=${OUTPUT_DIR:-$repo_root/docs/spike-artifacts/s5-s6-rerun}
port=${PORT:-49550}
run_root=$(mktemp -d "${TMPDIR:-/tmp}/codea-skill-isolation.XXXXXX")
server_pid=""

cleanup() {
  if [ -n "$server_pid" ]; then
    kill "$server_pid" 2>/dev/null || true
    wait "$server_pid" 2>/dev/null || true
  fi
  find "$run_root" -type f -delete 2>/dev/null || true
  find "$run_root" -depth -type d -exec rmdir {} \; 2>/dev/null || true
}
trap cleanup EXIT INT TERM

if [ -z "$opencode_bin" ] || [ ! -x "$opencode_bin" ]; then
  echo "OPENCODE_BIN must point to an executable OpenCode v1.18.11 binary" >&2
  exit 2
fi
if [ ! -d "$fixture_source" ]; then
  echo "Skill fixture source is missing: $fixture_source" >&2
  exit 2
fi
if ! command -v python3 >/dev/null 2>&1 || ! command -v curl >/dev/null 2>&1; then
  echo "python3 and curl are required" >&2
  exit 2
fi

mkdir -p "$output_dir/s5" "$output_dir/s6" "$output_dir/fixtures"
cp -R "$fixture_source"/. "$output_dir/fixtures/"
python3 - "$output_dir/fixtures" "$output_dir/fixture-manifest.txt" <<'PY'
import hashlib
import pathlib
import sys

root = pathlib.Path(sys.argv[1])
lines = []
for path in sorted(root.rglob("*")):
    if path.is_file():
        digest = hashlib.sha256(path.read_bytes()).hexdigest()
        lines.append(f"sha256:{digest}  {path.relative_to(root)}")
pathlib.Path(sys.argv[2]).write_text("\n".join(lines) + "\n")
PY

install_fixture() {
  local name=$1
  local destination=$2
  mkdir -p "$destination"
  cp "$fixture_source/$name/SKILL.md" "$destination/SKILL.md"
}

assert_names() {
  local response=$1
  local names_file=$2
  shift 2
  python3 - "$response" "$names_file" "$@" <<'PY'
import json
import pathlib
import sys

response = pathlib.Path(sys.argv[1])
names_file = pathlib.Path(sys.argv[2])
expected = sorted(sys.argv[3:])
payload = json.loads(response.read_text())
if not isinstance(payload, list):
    raise SystemExit(f"/skill response must be an array: {response}")
names = sorted(item["name"] for item in payload)
names_file.write_text("\n".join(names) + "\n")
if names != expected:
    raise SystemExit(f"unexpected Skill set for {response}: got={names!r} want={expected!r}")
PY
}

run_profile() {
  local profile=$1
  local group=$2
  shift 2
  local expected=("$@")
  local root="$run_root/$profile"
  local home="$root/home"
  local xdg_config="$root/xdg-config"
  local xdg_data="$root/xdg-data"
  local approved="$root/approved-config"
  local project="$root/project"
  local destination="$output_dir/$group"
  local response="$destination/$profile-skill-response.json"
  local names="$destination/$profile-skill-names.txt"
  local health="$destination/$profile-health.json"
  local stdout_log="$destination/$profile-server.log"

  if [ "$profile" = "control" ]; then
    xdg_config="$home/.config"
  fi

  mkdir -p "$home" "$xdg_config" "$xdg_data" "$root/cache" "$root/state" "$approved" "$project"
  install_fixture config-approved "$approved/skills/config-approved"
  install_fixture user-unapproved "$home/.config/opencode/skills/user-unapproved"
  install_fixture claude-unapproved "$home/.claude/skills/claude-unapproved"
  install_fixture project-unapproved "$project/.opencode/skills/project-unapproved"
  install_fixture agents-unapproved "$project/.agents/skills/agents-unapproved"

  (
    cd "$project"
    export HOME="$home"
    export XDG_CONFIG_HOME="$xdg_config"
    export XDG_DATA_HOME="$xdg_data"
    export XDG_CACHE_HOME="$root/cache"
    export XDG_STATE_HOME="$root/state"
    export OPENCODE_CONFIG_DIR="$approved"
    export OPENCODE_DISABLE_MODELS_FETCH=1
    export OPENCODE_DISABLE_AUTOUPDATE=1
    export OPENCODE_DISABLE_EMBEDDED_WEB_UI=1
    export OPENCODE_DISABLE_LSP_DOWNLOAD=1
    export OPENCODE_DISABLE_DEFAULT_PLUGINS=1
    export OPENCODE_SERVER_USERNAME=codea
    export OPENCODE_SERVER_PASSWORD=skill-isolation-fixture
    export CODEA_SKILL_PROFILE="$profile"

    case "$profile" in
      control)
        unset OPENCODE_DISABLE_EXTERNAL_SKILLS OPENCODE_DISABLE_PROJECT_CONFIG OPENCODE_DISABLE_CLAUDE_CODE
        ;;
      general-compatible)
        export OPENCODE_DISABLE_EXTERNAL_SKILLS=1
        unset OPENCODE_DISABLE_PROJECT_CONFIG
        export OPENCODE_DISABLE_CLAUDE_CODE=1
        ;;
      isolated|enterprise|general-strict)
        export OPENCODE_DISABLE_EXTERNAL_SKILLS=1
        export OPENCODE_DISABLE_PROJECT_CONFIG=1
        export OPENCODE_DISABLE_CLAUDE_CODE=1
        ;;
      *)
        echo "Unknown profile: $profile" >&2
        exit 2
        ;;
    esac

    "$opencode_bin" serve --hostname 127.0.0.1 --port "$port"
  ) >"$stdout_log" 2>&1 &
  server_pid=$!

  ready=0
  for _ in $(seq 1 50); do
    if curl -fsS --max-time 1 -u codea:skill-isolation-fixture \
      "http://127.0.0.1:$port/global/health" >"$health" 2>/dev/null; then
      ready=1
      break
    fi
    sleep 0.2
  done
  if [ "$ready" -ne 1 ]; then
    echo "OpenCode did not become healthy for profile $profile" >&2
    return 1
  fi

  curl -fsS --max-time 10 -u codea:skill-isolation-fixture --get \
    --data-urlencode "directory=$project" \
    "http://127.0.0.1:$port/skill" >"$response"

  kill "$server_pid" 2>/dev/null || true
  wait "$server_pid" 2>/dev/null || true
  server_pid=""

  internal_log="$xdg_data/opencode/log/opencode.log"
  if [ -f "$internal_log" ]; then
    cp "$internal_log" "$destination/$profile-opencode.log"
  else
    : >"$destination/$profile-opencode.log"
  fi

  assert_names "$response" "$names" "${expected[@]}"
  names_summary=$(paste -sd, "$names")
  echo "[PASS] $profile: $names_summary"
}

run_profile isolated s5 customize-opencode config-approved
run_profile control s5 customize-opencode claude-unapproved agents-unapproved user-unapproved project-unapproved config-approved
run_profile enterprise s6 config-approved customize-opencode
run_profile general-compatible s6 config-approved customize-opencode project-unapproved
run_profile general-strict s6 config-approved customize-opencode

echo "S5/S6 Skill isolation spikes passed. Evidence: $output_dir"
