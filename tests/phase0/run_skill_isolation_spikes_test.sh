#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
test_root=$(mktemp -d "${TMPDIR:-/tmp}/codea-skill-runner-test.XXXXXX")
trap 'find "$test_root" -type f -delete 2>/dev/null || true; find "$test_root" -depth -type d -exec rmdir {} \; 2>/dev/null || true' EXIT

cat >"$test_root/fake-opencode" <<'FAKE'
#!/usr/bin/env bash
set -euo pipefail
port=""
while [ "$#" -gt 0 ]; do
  if [ "$1" = "--port" ]; then
    port=$2
    shift 2
    continue
  fi
  shift
done
exec python3 "${FAKE_SKILL_SERVER:?}" "$port" "${CODEA_SKILL_PROFILE:?}"
FAKE
chmod +x "$test_root/fake-opencode"

cat >"$test_root/fake-server.py" <<'PY'
import json
import sys
from http.server import BaseHTTPRequestHandler, HTTPServer
from urllib.parse import urlparse

port = int(sys.argv[1])
profile = sys.argv[2]
import os
names = {
    "control": ["customize-opencode", "claude-unapproved", "agents-unapproved", "user-unapproved", "project-unapproved", "config-approved"],
    "isolated": ["customize-opencode", "config-approved"],
    "enterprise": ["config-approved", "customize-opencode"],
    "general-compatible": ["config-approved", "customize-opencode", "project-unapproved"],
    "general-strict": ["config-approved", "customize-opencode"],
}[profile]
if os.environ.get("FAKE_WRONG_PROFILE") == profile:
    names.append("unexpected-skill")

class Handler(BaseHTTPRequestHandler):
    def do_GET(self):
        path = urlparse(self.path).path
        if path == "/global/health":
            payload = {"healthy": True, "version": os.environ.get("FAKE_HEALTH_VERSION", "1.18.11")}
        elif path == "/skill":
            payload = [{"name": name, "description": "fixture", "location": f"/fixture/{name}/SKILL.md", "content": f"raw {name}"} for name in names]
        else:
            self.send_response(404)
            self.end_headers()
            return
        body = json.dumps(payload).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, *_):
        pass

HTTPServer(("127.0.0.1", port), Handler).serve_forever()
PY

output="$test_root/output"
FAKE_SKILL_SERVER="$test_root/fake-server.py" \
OPENCODE_BIN="$test_root/fake-opencode" \
OUTPUT_DIR="$output" \
PORT=49570 \
  "$repo_root/scripts/run-skill-isolation-spikes.sh" >/dev/null

for response in \
  s5/isolated-skill-response.json \
  s5/control-skill-response.json \
  s6/enterprise-skill-response.json \
  s6/general-compatible-skill-response.json \
  s6/general-strict-skill-response.json; do
  test -s "$output/$response"
  python3 -m json.tool "$output/$response" >/dev/null
done

test -s "$output/fixture-manifest.txt"
grep -q 'config-approved/SKILL.md' "$output/fixture-manifest.txt"
diff -u <(printf '%s\n' config-approved customize-opencode) "$output/s6/enterprise-skill-names.txt"
diff -u <(printf '%s\n' config-approved customize-opencode project-unapproved) "$output/s6/general-compatible-skill-names.txt"

if FAKE_SKILL_SERVER="$test_root/fake-server.py" \
  FAKE_HEALTH_VERSION=9.9.9 \
  OPENCODE_BIN="$test_root/fake-opencode" \
  OUTPUT_DIR="$test_root/wrong-version" \
  PORT=49571 \
    "$repo_root/scripts/run-skill-isolation-spikes.sh" >/dev/null 2>&1; then
  echo 'expected wrong Runtime version to fail' >&2
  exit 1
fi

if FAKE_SKILL_SERVER="$test_root/fake-server.py" \
  FAKE_WRONG_PROFILE=enterprise \
  OPENCODE_BIN="$test_root/fake-opencode" \
  OUTPUT_DIR="$test_root/wrong-set" \
  PORT=49572 \
    "$repo_root/scripts/run-skill-isolation-spikes.sh" >/dev/null 2>&1; then
  echo 'expected unexpected Skill set to fail' >&2
  exit 1
fi

echo 'Skill isolation spike runner tests passed'
