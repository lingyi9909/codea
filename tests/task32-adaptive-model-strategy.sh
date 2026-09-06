#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

cd "$ROOT/distribution/plugins"
bun test \
  tests/model-probes.test.ts \
  tests/model-evaluator-agent.test.ts \
  tests/model-strategy-security.test.ts

cd "$ROOT/tui"
GOTOOLCHAIN=local go test ./internal/modelprofile -run 'Task32|Store|Aggregate' -count=1
GOTOOLCHAIN=local go test ./internal/app -run 'Task32' -count=1
GOTOOLCHAIN=local go test ./internal/checkpoint -count=1

cd "$ROOT"
bash tests/task28-repo-intelligence.sh
bash tests/task29-agent-planning.sh
bash tests/task30-verification-loop.sh
bash tests/task31-checkpoint.sh
bash tests/task32-strategy-security.sh

printf '%s\n' \
  'TASK32_MODEL_QUALIFICATION PASS' \
  'STRONG_PROFILE PASS' \
  'MEDIUM_PROFILE PASS' \
  'WEAK_PROFILE PASS' \
  'MODEL_PROFILE_ISOLATION PASS' \
  'STRATEGY_SECURITY_INVARIANT PASS' \
  'V12_AGENT_RELIABILITY_REGRESSION PASS'
