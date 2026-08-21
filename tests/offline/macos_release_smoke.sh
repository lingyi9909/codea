#!/usr/bin/env bash
set -euo pipefail

package_dir=${1:?usage: macos_release_smoke.sh <extracted-package-dir> [evidence-file]}
evidence_file=${2:-}
repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
[ "$(uname -s)" = "Darwin" ] || { echo "BLOCKED: macOS host required" >&2; exit 2; }
arch=$(uname -m)
case "$arch" in arm64) platform=darwin-arm64 ;; x86_64) platform=darwin-x64 ;; *) echo "unsupported macOS arch: $arch" >&2; exit 2 ;; esac
if [ -z "$evidence_file" ]; then evidence_file="$repo_root/tests/offline/evidence/task17-$platform-evidence.json"; fi
mkdir -p "$(dirname "$evidence_file")"

# Fail closed unless this is a genuinely isolated/offline environment.
if curl -fsS --connect-timeout 2 --max-time 3 https://github.com/ >/dev/null 2>&1; then
  echo "FAIL: public HTTPS is reachable" >&2
  exit 1
fi

package_dir=$(cd "$package_dir" && pwd -P)
[ -x "$package_dir/install/install.sh" ] || { echo "install.sh missing" >&2; exit 1; }
work=$(mktemp -d "${TMPDIR:-/tmp}/codea-macos-release.XXXXXX")
home="$work/home"
config="$work/runtime-config"
port=${SMOKE_PORT:-49331}
username=codea-smoke
password=codea-smoke-pass
runtime_pid=""
codea_pid=""
cleanup() {
  if [ -n "$codea_pid" ]; then kill "$codea_pid" 2>/dev/null || true; wait "$codea_pid" 2>/dev/null || true; fi
  if [ -n "$runtime_pid" ]; then kill "$runtime_pid" 2>/dev/null || true; wait "$runtime_pid" 2>/dev/null || true; fi
  rm -rf "$work"
}
trap cleanup EXIT INT TERM

CODEA_HOME="$home" "$package_dir/install/install.sh" "$package_dir"
[ -L "$home/current" ] || { echo "FAIL: ~/.codea/current is not a symlink" >&2; exit 1; }
[ -x "$home/bin/codea" ] || { echo "FAIL: installed codea launcher missing" >&2; exit 1; }
current=$(cd "$home/current" && pwd -P)
[ -x "$current/bin/codea" ] || { echo "FAIL: installed codea binary missing" >&2; exit 1; }
[ -x "$current/bin/opencode" ] || { echo "FAIL: installed OpenCode binary missing" >&2; exit 1; }
[ -f "$current/plugins/index.js" ] || { echo "FAIL: installed plugin missing" >&2; exit 1; }
[ -d "$current/agents" ] && [ -d "$current/skills" ] || { echo "FAIL: installed enterprise resources missing" >&2; exit 1; }

export OPENCODE_SERVER_USERNAME="$username"
export OPENCODE_SERVER_PASSWORD="$password"
export OPENCODE_DISABLE_MODELS_FETCH=1
export OPENCODE_DISABLE_AUTOUPDATE=1
export OPENCODE_DISABLE_EMBEDDED_WEB_UI=1
export OPENCODE_DISABLE_LSP_DOWNLOAD=1
export OPENCODE_DISABLE_DEFAULT_PLUGINS=1
mkdir -p "$config"
OPENCODE_CONFIG_DIR="$config" "$current/bin/opencode" serve --hostname 127.0.0.1 --port "$port" >"$work/opencode.log" 2>&1 &
runtime_pid=$!

healthy=0
for _ in $(seq 1 100); do
  if ! kill -0 "$runtime_pid" 2>/dev/null; then cat "$work/opencode.log" >&2; exit 1; fi
  if curl -fsS --max-time 1 -u "$username:$password" "http://127.0.0.1:$port/global/health" >"$work/health.json" 2>/dev/null; then healthy=1; break; fi
  sleep 0.2
done
[ "$healthy" -eq 1 ] || { echo "FAIL: packaged OpenCode serve did not become healthy" >&2; cat "$work/opencode.log" >&2; exit 1; }
python3 - "$work/health.json" <<'PY'
import json, pathlib, sys
p=json.loads(pathlib.Path(sys.argv[1]).read_text())
if p.get('healthy') is not True or p.get('version') != '1.18.11': raise SystemExit(f'bad health: {p}')
PY

# Run the installed launcher inside a pseudo-terminal. OPENCODE_URL attaches it
# to the already verified packaged runtime; remaining resource paths come only
# from the launcher itself. If the launcher wiring is broken, Codea exits early.
command -v script >/dev/null 2>&1 || { echo "BLOCKED: macOS script(1) is required" >&2; exit 2; }
TERM=xterm-256color CODEA_HOME="$home" CODEA_RUNTIME_CONFIG_DIR="$work/codea-config" \
OPENCODE_URL="http://127.0.0.1:$port" OPENCODE_USERNAME="$username" OPENCODE_PASSWORD="$password" \
  script -q /dev/null "$home/bin/codea" >"$work/codea.log" 2>&1 &
codea_pid=$!
sleep 3
if ! kill -0 "$codea_pid" 2>/dev/null; then
  wait "$codea_pid" 2>/dev/null || true
  cat "$work/codea.log" >&2
  echo "FAIL: installed Codea launcher exited during startup" >&2
  exit 1
fi
kill "$codea_pid" 2>/dev/null || true
wait "$codea_pid" 2>/dev/null || true
codea_pid=""

python3 - "$evidence_file" "$platform" "$current" <<'PY'
import json, pathlib, sys, time
out=pathlib.Path(sys.argv[1]); out.parent.mkdir(parents=True, exist_ok=True)
p={
  'timestamp': time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime()),
  'platform': sys.argv[2],
  'installedCurrent': sys.argv[3],
  'publicHttpsBlocked': True,
  'installerPassed': True,
  'currentPointerValid': True,
  'bundledResourcesPresent': True,
  'opencodeServeHealthy': True,
  'openCodeVersion': '1.18.11',
  'codeaLauncherStarted': True,
  'passedChecks': 7,
  'totalChecks': 7,
}
out.write_text(json.dumps(p, indent=2)+'\n')
PY

echo "[PASS] $platform installed release: offline + install + OpenCode serve + Codea launcher"
