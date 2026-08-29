#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd -P)"
PLUGIN="$ROOT/distribution/plugins"
TUI="$ROOT/tui"

fail() {
  echo "TASK29_AGENT_PLANNING FAIL: $*" >&2
  exit 1
}

bash "$ROOT/tests/task29-agent-planning-contract.sh"

LOG="$(mktemp)"
trap 'rm -f "$LOG"' EXIT
(
  cd "$PLUGIN"
  bun test tests/task29-acceptance.test.ts tests/task-root-epoch.test.ts 2>&1 | tee "$LOG"
)

for marker in \
  'READ_WITHOUT_PLAN PASS' \
  'WRITE_WITHOUT_PLAN PLAN_REQUIRED' \
  'EDIT_WITHOUT_PLAN PLAN_REQUIRED' \
  'BASH_WITHOUT_PLAN PLAN_REQUIRED' \
  'ENTERPRISE_WRITE_WITHOUT_PLAN PLAN_REQUIRED' \
  'PLAN_3_TO_7_STEPS PASS' \
  'SINGLE_ACTIVE_STEP PASS' \
  'CROSS_SESSION_PLAN_ISOLATION PASS' \
  'PLAN_PERSISTENCE PASS' \
  'NEW_USER_TURN_INVALIDATES_PRIOR_PLAN PASS' \
  'CONTROL_CONTINUATION_PRESERVES_ROOT_EPOCH PASS'
do
  grep -Fq "$marker" "$LOG" || fail "missing protocol marker: $marker"
done

(
  cd "$TUI"
  GOTOOLCHAIN=local go test ./internal/app ./internal/runtime ./internal/opencode \
    -run 'TaskExecution|TaskPlan|ToolMetadata|Task29MessageUpdated|Trace|Session' -count=1
  echo 'LIVE_TRACE_PLAN_STATE PASS'

  GOTOOLCHAIN=local go test ./internal/agent ./internal/app \
    -run 'Professional|Agent|Materialize|TaskStrategy' -count=1
  echo 'PROFESSIONAL_AGENT_ROUTING PASS'
)

echo 'TASK29_AGENT_PLANNING PASS'
