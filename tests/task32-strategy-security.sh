#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT/distribution/plugins"

bun test \
  tests/model-strategy-security.test.ts \
  tests/model-evaluator-agent.test.ts \
  tests/permissions.test.ts \
  tests/command-policy.test.ts \
  tests/path-policy.test.ts \
  tests/dlp.test.ts \
  tests/runtime-security-guard.test.ts

printf '%s\n' \
  'STRATEGY_SECURITY_INVARIANT PASS' \
  'MODEL_EVALUATOR_ISOLATION PASS'
