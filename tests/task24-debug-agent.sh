#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MANIFEST="$ROOT/distribution/agents/debug/manifest.yaml"
PROMPT="$ROOT/distribution/agents/debug/agent.md"
SKILL="$ROOT/distribution/skills/debug/SKILL.md"
PERMISSIONS="$ROOT/distribution/config/opencode/permissions.json"
BUILTINS="$ROOT/tui/internal/command/builtins.go"
ENTRY="$ROOT/distribution/plugins/src/opencode/entry.ts"

fail() {
  echo "TASK24_DEBUG_AGENT FAIL: $*" >&2
  exit 1
}

for file in "$MANIFEST" "$PROMPT" "$SKILL"; do
  [[ -f "$file" ]] || fail "missing ${file#$ROOT/}"
done

grep -Eq '^name:[[:space:]]*debug$' "$MANIFEST" || fail "debug manifest name"
grep -A4 '^requiredSkills:' "$MANIFEST" | grep -Eq '^[[:space:]]*-[[:space:]]*debug$' || fail "debug requiredSkills must include debug"
for tool in read grep glob; do
  grep -Eq "^[[:space:]]+$tool:[[:space:]]+allow$" "$MANIFEST" || fail "$tool must be allow"
done
for tool in write edit bash; do
  grep -Eq "^[[:space:]]+$tool:[[:space:]]+ask$" "$MANIFEST" || fail "$tool must be ask"
done

python3 - "$PERMISSIONS" <<'PY'
import json, sys
p = sys.argv[1]
data = json.load(open(p, encoding="utf-8"))
agents = data["agents"]
debug = agents.get("debug")
if not debug:
    raise SystemExit("TASK24_DEBUG_AGENT FAIL: permissions missing debug")
for tool in ("read", "grep", "glob"):
    if debug.get(tool) != "allow":
        raise SystemExit(f"TASK24_DEBUG_AGENT FAIL: debug {tool} must allow")
for tool in ("write", "edit", "bash"):
    if debug.get(tool) != "ask":
        raise SystemExit(f"TASK24_DEBUG_AGENT FAIL: debug {tool} must ask")
for agent in ("code-reviewer", "unit-test-generator", "api-documentation"):
    policy = agents[agent]
    for tool in ("write", "edit", "bash"):
        if policy.get(tool) != "deny":
            raise SystemExit(f"TASK24_DEBUG_AGENT FAIL: regression {agent}.{tool} != deny")
PY

# Debug workflow is an evidence-driven loop, not a generic chat persona.
for marker in 'Collect evidence' 'Reproduce' 'Root cause' 'Controlled fix' 'Fresh verification'; do
  grep -Fqi "$marker" "$PROMPT" || fail "agent prompt missing workflow marker: $marker"
  grep -Fqi "$marker" "$SKILL" || fail "debug skill missing workflow marker: $marker"
done

grep -Fqi 'do not claim' "$PROMPT" || fail "agent prompt must prohibit unverified success claims"
grep -Fqi 'approval' "$PROMPT" || fail "agent prompt must preserve approval"
grep -Fqi 'DLP' "$PROMPT" || fail "agent prompt must preserve DLP"
grep -Fqi 'offline' "$PROMPT" || fail "agent prompt must preserve offline policy"

# Professional command dispatch remains deterministic and bypasses General Agent.
for pair in \
  'review.*code-reviewer' \
  'test.*unit-test-generator' \
  'api-doc.*api-documentation' \
  'debug.*Agent: "debug"'; do
  grep -Eq "$pair" "$BUILTINS" || fail "missing deterministic route: $pair"
done

# Native mutation safety must remain in front of OpenCode's approval/execution.
grep -Fq 'NATIVE_MUTATION_TOOLS' "$ENTRY" || fail "native write/edit security hook missing"
grep -Fq 'input: output.args' "$ENTRY" || fail "native mutation input must pass DLP guard"
grep -Fq 'validateNativeReadPath(input.directory, targetPath)' "$ENTRY" || fail "native mutation path guard missing"

echo "TASK24_DEBUG_AGENT PASS"
