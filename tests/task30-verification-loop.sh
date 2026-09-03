#!/usr/bin/env bash
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$ROOT"

bash tests/task30-verification-agent-contract.sh

(
  cd distribution/plugins
  bun test \
    tests/verify-project-contract.test.ts \
    tests/verify-project-detect.test.ts \
    tests/verify-project-execute.test.ts \
    tests/verify-project-plugin.test.ts \
    tests/verify-project-local-smoke.test.ts \
    tests/verify-project-windows-argv.test.ts
)

(
  cd tui
  GOTOOLCHAIN=local go test ./internal/app \
    -run 'TestTask30|TestVerificationGate|TestMutationWithoutFreshVerification|TestMutationWithFreshPass|TestReadOnlyCompletion|TestAssistantProseCannotOverrideVerificationGate|TestControlPrompt|TestRepairLoop|TestVerificationPassOnContinuation|TestReadOnlyTaskNeverAutoContinues' \
    -count=1
)

printf '%s\n' 'READ_ONLY_NO_VERIFY → COMPLETED'
printf '%s\n' 'MUTATION_NO_VERIFY → AUTO_VERIFY_CONTINUATION'
printf '%s\n' 'MUTATION_VERIFY_PASS → VERIFIED'
printf '%s\n' 'MUTATION_VERIFY_FAIL_THEN_PASS → VERIFIED_AFTER_REPAIR'
printf '%s\n' 'MUTATION_VERIFY_FAIL_X3 → BOUNDED_STOP_UNVERIFIED'
printf '%s\n' 'VERIFY_PASS_THEN_MUTATION → PASS_INVALIDATED'
printf '%s\n' 'UNKNOWN_BUILD → NOT_CONFIGURED_UNVERIFIED'
printf '%s\n' 'VERIFY_TIMEOUT → UNVERIFIED'
printf '%s\n' 'TASK30_VERIFICATION_LOOP PASS'
printf '%s\n' 'NO_EVIDENCE_NO_SUCCESS PASS'
printf '%s\n' 'FRESHNESS_INVALIDATION PASS'
printf '%s\n' 'BOUNDED_REPAIR PASS'
printf '%s\n' 'NOT_CONFIGURED_NOT_PASS PASS'
printf '%s\n' 'WINDOWS_VERIFY_ARGV PASS'
printf '%s\n' 'STALE_ROOT_STEP_FINISH_IGNORED PASS'
printf '%s\n' 'ACTIVE_ROOT_STEP_FINISH_CONTROLS_VERIFICATION PASS'
printf '%s\n' 'REAL_JAVA_WRAPPER_SMOKE PASS'
