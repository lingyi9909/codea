#!/usr/bin/env bash
set -euo pipefail

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
source_commit=${CODEA_SOURCE_COMMIT:-$(git -C "$repo_root" rev-parse HEAD)}
evidence_dir=${CODEA_RELEASE_GATE_DIR:-"$repo_root/tests/evidence/release-gates.d"}
opencode_bin=${OPENCODE_BIN:-}
real_smoke_timeout_seconds=${CODEA_REAL_SMOKE_TIMEOUT_SECONDS:-900}

[ -n "$source_commit" ] || { echo "CODEA_SOURCE_COMMIT is required" >&2; exit 2; }
[ -n "$opencode_bin" ] && [ -x "$opencode_bin" ] || { echo "OPENCODE_BIN must point to locked OpenCode v1.18.11" >&2; exit 2; }
for cmd in python3 go bun timeout; do command -v "$cmd" >/dev/null 2>&1 || { echo "$cmd is required" >&2; exit 2; }; done

rm -rf "$evidence_dir"
mkdir -p "$evidence_dir"

write_gate() {
  local id=$1 evidence=$2
  python3 "$repo_root/scripts/write-release-gate.py" \
    --id "$id" --source-commit "$source_commit" --status pass \
    --evidence "$evidence" --out "$evidence_dir/${id//./_}.json"
  echo "[PASS] $id — $evidence"
}

run_bounded() {
  local label=$1
  shift
  local started=$SECONDS
  local status
  echo "=== START ${label} ==="
  set +e
  timeout --foreground "$real_smoke_timeout_seconds" "$@"
  status=$?
  set -e
  if [ "$status" -eq 124 ]; then
    echo "FAIL: ${label} timed out after ${real_smoke_timeout_seconds}s" >&2
    return 124
  fi
  if [ "$status" -ne 0 ]; then
    echo "FAIL: ${label} exited with status ${status}" >&2
    return "$status"
  fi
  echo "=== PASS ${label} · $((SECONDS-started))s ==="
}

# G3 — all three skill modes and source/approval policy are exercised by the
# skill package's mode/integration/policy tests.
(
  cd "$repo_root/tui"
  GOTOOLCHAIN=local go test ./internal/skill -count=1
)
write_gate G3 "tui/internal/skill full package tests: Enterprise controlled + General strict + General compatible skill source/isolation policy"

# G4/G5 — accepted upgrade subsystem contracts. The failed-upgrade suite proves
# atomic rollback of version+config; rollback suite proves one-command recovery.
bash "$repo_root/tests/upgrade/failed_upgrade_test.sh"
bash "$repo_root/tests/upgrade/upgrade_test.sh"
write_gate G4 "tests/upgrade/failed_upgrade_test.sh + upgrade_test.sh: failed upgrade restores version/config and committed upgrade remains atomic"

bash "$repo_root/tests/upgrade/rollback_test.sh"
write_gate G5 "tests/upgrade/rollback_test.sh: rollback last committed version/config and crash recovery"

# G6/G7 — one real locked-runtime smoke emits the accepted 15/15 evidence for
# Code Reviewer and Unit Test Generator.
run_bounded "G6-G7" env OPENCODE_BIN="$opencode_bin" bash "$repo_root/scripts/run-real-agent-smoke.sh"
write_gate G6 "tui/tests/parity/evidence/agent-evidence.json 15/15: Code Reviewer file/line/code-evidence workflow"
write_gate G7 "tui/tests/parity/evidence/agent-evidence.json 15/15: Unit Test Generator production-source mutation count = 0"

# G8 — real locked-runtime API Documentation smoke validates deterministic
# extraction/validation and rejects invented schema fields.
run_bounded "G8" env OPENCODE_BIN="$opencode_bin" bash "$repo_root/scripts/run-real-api-doc-smoke.sh"
write_gate G8 "tui/tests/parity/evidence/api-doc-agent-evidence.json: real API Documentation workflow with schema/example validation and no fabricated fields"

# G9 — reasoning normalization/processing plus TUI application rendering/key
# behavior must remain green.
(
  cd "$repo_root/tui"
  GOTOOLCHAIN=local go test ./internal/reasoning ./internal/app -count=1
)
write_gate G9 "go test ./internal/reasoning ./internal/app: reasoning separated from answer and TUI reasoning interaction regression"

# G10 — fail-closed command, permission, path and DLP policy.
(
  cd "$repo_root/distribution/plugins"
  bun test tests/command-policy.test.ts tests/permissions.test.ts tests/runtime-security-guard.test.ts tests/dlp.test.ts
)
write_gate G10 "Bun security suite: dangerous command deny, write/bash approval policy, path policy and DLP"

# G11/G12/G13 — run the locked real Runtime capability smoke once, then combine
# it with mapper/parity unit contracts. This evidence covers native capabilities,
# capability reachability, approval flows and unknown-event Raw fallback.
run_bounded "G11-G13" env OPENCODE_BIN="$opencode_bin" bash "$repo_root/scripts/run-real-parity-smoke.sh"
(
  cd "$repo_root/tui"
  GOTOOLCHAIN=local go test ./internal/opencode ./internal/parity ./tests/parity -count=1
)
write_gate G11 "real-parity evidence on OpenCode v1.18.11 + OpenCode adapter tests: locked native capability inventory usable through Codea; unknown SSE maps to Raw"
write_gate G12 "real-parity evidence + parity/adapter regressions: core feature/tool/event/API capability reachability remains complete"
write_gate G13 "real-parity evidence + golden/adapter event regression: SSE events mapped or Raw-transmitted with zero silent drop"

# G12.1 — baseline is vanilla locked OpenCode; candidate is same Runtime/model/
# permissions with Codea plugin. All Required scenarios are repeated by Runner.
run_bounded "G12.1" env OPENCODE_BIN="$opencode_bin" bash "$repo_root/scripts/run-dual-runtime-parity.sh"
write_gate G12.1 "tui/tests/parity/evidence/release-parity.json: distinct vanilla/candidate OpenCode v1.18.11, 12/12 Required task-effect parity"

# G14 — Application-layer handoff contract keeps one Session and transfers only
# structured state to General without repeating generated-file writes.
(
  cd "$repo_root/tui"
  GOTOOLCHAIN=local go test ./internal/app -run 'Handoff' -count=1
)
write_gate G14 "tui/internal/app handoff regression: same Session, original goal/facts/tool results/failure transferred to general, generated files protected from overwrite"

printf '[PASS] core release gates G3-G14 generated under %s\n' "$evidence_dir"
