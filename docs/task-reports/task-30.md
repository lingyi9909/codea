# Task 30 — Deterministic Verification Loop Verification Report

## Status

- Task: **30 — Deterministic Verification Loop**
- Production baseline: `85ab175f04a6642db73714f42048bfa5c73ce89c`
- Fresh Task 30 Gate: **GitHub Actions `33774541543` — PASS**
- Linux: **PASS**
- Windows: **PASS**
- macOS: **PASS**
- Automated verification: **PASS**
- Task Gate: **PASS**
- Human acceptance: **PENDING**
- Task 31: **not started**

## Delivered behavior

Task 30 makes machine verification, rather than assistant prose, authoritative for verified completion after project mutation.

1. **Deterministic `verify_project` contract**
   - Supports bounded local verification profiles for Maven, Gradle, and Go.
   - Unknown/ambiguous project configuration returns `NOT_CONFIGURED`; it is never converted to PASS.
   - The tool accepts only a bounded timeout option. Arbitrary command strings, args, repository selectors, and shell extensions are not caller-controllable.
   - Maven/Gradle wrappers are preferred when present; Go uses fixed `go test ./...` and `go vet ./...` stages.
   - Static/check stages are run only when locally configured and detected; Task 30 does not claim support for every build system or every project-specific check plugin.

2. **Secure local execution**
   - Verification executes through the existing Codea command execution/security boundary.
   - Task 29 planning remains mandatory before verification execution.
   - Existing approval, project-root binding, command policy, DLP, audit, timeout, and bounded-output behavior remain in force.
   - Windows `.cmd/.bat` wrappers are routed as fixed argv; project paths containing spaces are covered by regression tests without shell interpolation.
   - Public internet is not required by the Task 30 verification contract or acceptance smoke.

3. **Explicit machine verification evidence**
   - Plugin results emit allowlisted machine metadata for verification identity/result/profile.
   - Application consumes only explicit Runtime metadata; raw tool stdout, assistant answer text, and reasoning text are not parsed to infer PASS.
   - Verification attempts and results are tracked in `TaskExecutionState` for the active root task.
   - A successful verification becomes stale immediately after a later mutation in the same root task.

4. **Truthful completion gate**
   - Read-only successful work may complete without project verification.
   - A mutating task requires a fresh successful `verify_project` result before rendering verified completion.
   - Missing verification, `FAIL`, `TIMEOUT`, `NOT_CONFIGURED`, or execution error cannot be presented as verified success.
   - Assistant prose cannot override the machine gate.

5. **Bounded synthetic verification/repair continuation**
   - Missing verification or failed verification can schedule an internal Codea control continuation.
   - Continuations keep the original engineering root turn and preserve session, Agent, model, approval, and trace identity.
   - Control metadata uses `codea.kind=verification-control` plus the original root turn.
   - The continuation is not appended as a fake visible user message.
   - Automatic continuation is bounded to at most two attempts; exhaustion stops automatically as failed/unverified and never emits a third retry.
   - Stale `step.finished` events from a previous root are ignored before they can flush current reasoning or control the active verification lifecycle.

6. **Mutation-capable Agent contracts**
   - Debug and Unit Test workflows explicitly use `verify_project` as the final machine gate.
   - On failure they inspect evidence, make the smallest justified fix, and reverify within the bounded loop.
   - `NOT_CONFIGURED` and `TIMEOUT` are reported truthfully as unverified.
   - Code Reviewer remains read-only and does not receive an unnecessary verification requirement.

## TDD / focused evidence

Task 30 was developed with RED → GREEN checks for the verification contract, profile detection, execution, Runtime metadata mapping, completion gate, bounded continuation, and mutation-capable Agent contracts.

For Step 7 specifically:

- `33282508285` — RED: mutation-capable Agent contract check failed because Debug did not yet require `verify_project`.
- `33282584648` — GREEN: Agent contract and related Task 29 / Debug regressions passed after the minimal contract update.

A three-platform proof run was used before formal merge to expose compatibility issues. Follow-up production fixes closed stale-root lifecycle handling, native Windows wrapper routing/selection, Windows path fixtures, Windows workflow fail-fast behavior, and the real Java wrapper smoke without weakening the verification contract.

## Formal mechanical acceptance

Fresh Task 30 Gate run: **GitHub Actions `33774541543`** on exact production baseline `85ab175f04a6642db73714f42048bfa5c73ce89c`.

The permanent mechanical acceptance produced these factual scenario markers:

```text
READ_ONLY_NO_VERIFY → COMPLETED
MUTATION_NO_VERIFY → AUTO_VERIFY_CONTINUATION
MUTATION_VERIFY_PASS → VERIFIED
MUTATION_VERIFY_FAIL_THEN_PASS → VERIFIED_AFTER_REPAIR
MUTATION_VERIFY_FAIL_X3 → BOUNDED_STOP_UNVERIFIED
VERIFY_PASS_THEN_MUTATION → PASS_INVALIDATED
UNKNOWN_BUILD → NOT_CONFIGURED_UNVERIFIED
VERIFY_TIMEOUT → UNVERIFIED
TASK30_VERIFICATION_LOOP PASS
NO_EVIDENCE_NO_SUCCESS PASS
FRESHNESS_INVALIDATION PASS
BOUNDED_REPAIR PASS
NOT_CONFIGURED_NOT_PASS PASS
WINDOWS_VERIFY_ARGV PASS
STALE_ROOT_STEP_FINISH_IGNORED PASS
ACTIVE_ROOT_STEP_FINISH_CONTROLS_VERIFICATION PASS
REAL_JAVA_WRAPPER_SMOKE PASS
WINDOWS_WORKFLOW_FAIL_FAST PASS
```

Focused verification passed **26 tests / 0 fail**. Full plugin regression passed **303 tests / 0 fail**.

## Full regression evidence

### Linux — Ubuntu 24.04

Exact checkout: `85ab175f04a6642db73714f42048bfa5c73ce89c`.

- execution-state validation: PASS
- Task 30 mechanical acceptance: PASS
- focused verification tests: **26 pass / 0 fail**
- plugin regression: **303 pass / 0 fail**, 33 files, 801 assertions
- plugin build: PASS
- `GOTOOLCHAIN=local go test ./... -count=1`: PASS
- `GOTOOLCHAIN=local go build ./cmd/codea`: PASS
- `STALE_ROOT_STEP_FINISH_IGNORED PASS`
- `ACTIVE_ROOT_STEP_FINISH_CONTROLS_VERIFICATION PASS`
- `REAL_JAVA_WRAPPER_SMOKE PASS`

### Windows — Windows Server 2025

Exact checkout: `85ab175f04a6642db73714f42048bfa5c73ce89c`; native `windows/amd64`, Go `1.26.5`.

- `WINDOWS_WORKFLOW_FAIL_FAST PASS`
- real Java wrapper verification smoke: `REAL_JAVA_WRAPPER_SMOKE PASS`
- native Windows wrapper/argv verification: `WINDOWS_VERIFY_ARGV PASS`
- focused verification tests: **26 pass / 0 fail**
- full plugin regression: **303 pass / 0 fail**, 33 files, 804 assertions
- plugin build: PASS
- focused Task 30 Go regression with `CGO_ENABLED=0`: PASS
- full Go regression with `CGO_ENABLED=0`: PASS
- `GOTOOLCHAIN=local go build ./cmd/codea`: PASS

### macOS — macOS 15 arm64

Exact checkout: `85ab175f04a6642db73714f42048bfa5c73ce89c`; native `darwin/arm64`, Go `1.26.5`.

- Task 30 mechanical acceptance: PASS
- focused verification tests: **26 pass / 0 fail**
- full plugin regression: **303 pass / 0 fail**, 33 files, 801 assertions
- plugin build: PASS
- full Go regression/build: PASS
- `REAL_JAVA_WRAPPER_SMOKE PASS`

## Supported scope / explicit limits

Task 30 deterministically supports the currently implemented local profiles: Maven, Gradle, and Go. Maven/Gradle verification uses detected local build configuration and wrapper/bare tooling; configured static/check stages are recognized only for the specific local patterns implemented by the detector. Projects using other build systems or unsupported/ambiguous configuration return `NOT_CONFIGURED` rather than being guessed or marked PASS.

The feature does not introduce a Verifier LLM Agent, does not fetch public dependencies as part of its control logic, and does not let assistant wording determine completion.

## Gate conclusion

All Task 30 technical done criteria are satisfied at production baseline `85ab175f04a6642db73714f42048bfa5c73ce89c`, with Fresh Task 30 Gate `33774541543` passing on Linux, Windows, and macOS. Fixed local verification profiles execute through existing security controls, safe machine metadata drives Application freshness, stale previous-root completion events cannot control the active task, mutation cannot render verified completion without a fresh PASS, failed/missing verification is bounded to two automatic continuations, retry exhaustion stops truthfully, unknown configuration never becomes PASS, real Java wrappers are exercised through the actual verification tool path, and Windows Gate native-command failures are fail-fast.

Task 30 technical verification is complete and ready for final human acceptance. Human acceptance remains separate and is intentionally not marked true in execution state. Task 31 remains pending until Task 30 is explicitly accepted.
