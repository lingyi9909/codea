# Task 30 — Deterministic Verification Loop Verification Report

## Status

- Task: **30 — Deterministic Verification Loop**
- Production checkpoint: `ba65929a9dc1d7f582fba1629acfe61e75476ea4`
- Production Gate: **GitHub Actions `33283072841` — PASS**
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

A three-platform proof run was used before formal merge to expose compatibility issues. Two proof-only issues were fixed without weakening the production contract: the existing Unit Test E2E wording was preserved, and the Windows real-Go smoke was given a test-harness timeout large enough for native Windows execution. Proof-only workflow files were excluded from `develop`.

## Formal mechanical acceptance

Production Gate run: **GitHub Actions `33283072841`** on exact production checkpoint `ba65929a9dc1d7f582fba1629acfe61e75476ea4`.

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
```

Focused plugin verification also passed **25 tests / 0 fail** covering contract aggregation, local profile detection, secure execution, plugin registration/security, Windows argv safety, and a real standard-library-only Go verification smoke.

## Full regression evidence

### Linux — Ubuntu 24.04

Exact checkout: `ba65929a9dc1d7f582fba1629acfe61e75476ea4`.

- execution-state validation: PASS
- Task 30 mechanical acceptance: PASS
- focused verification tests: **25 pass / 0 fail**
- plugin regression: **301 pass / 0 fail**, 33 files, 783 assertions
- plugin build: **101 modules bundled**
- `GOTOOLCHAIN=local go test ./... -count=1`: PASS
- `GOTOOLCHAIN=local go build ./cmd/codea`: PASS

### Windows — Windows Server 2025

Exact checkout: `ba65929a9dc1d7f582fba1629acfe61e75476ea4`; native `windows/amd64`, Go `1.26.5`.

- real local verification smoke: PASS
- path-with-spaces plus fixed `.cmd/.bat` argv regression: PASS
- full plugin regression/build: PASS
- focused Task 30 Go regression with `CGO_ENABLED=0`: PASS
- full Go regression with `CGO_ENABLED=0`: PASS

### macOS — macOS 15 arm64

Exact checkout: `ba65929a9dc1d7f582fba1629acfe61e75476ea4`; native `darwin/arm64`, Go `1.26.5`.

- Task 30 mechanical acceptance: PASS
- full plugin regression/build: PASS
- full Go regression/build: PASS

## Supported scope / explicit limits

Task 30 deterministically supports the currently implemented local profiles: Maven, Gradle, and Go. Maven/Gradle verification uses detected local build configuration and wrapper/bare tooling; configured static/check stages are recognized only for the specific local patterns implemented by the detector. Projects using other build systems or unsupported/ambiguous configuration return `NOT_CONFIGURED` rather than being guessed or marked PASS.

The feature does not introduce a Verifier LLM Agent, does not fetch public dependencies as part of its control logic, and does not let assistant wording determine completion.

## Gate conclusion

All Task 30 done criteria are satisfied at production checkpoint `ba65929a9dc1d7f582fba1629acfe61e75476ea4`: fixed local verification profiles execute through existing security controls, safe machine metadata drives Application freshness, mutation cannot render verified completion without a fresh PASS, failed/missing verification is bounded to two automatic continuations, retry exhaustion stops truthfully, unknown configuration never becomes PASS, and Linux/Windows/macOS regressions are green.

Task 30 is therefore ready for human acceptance. Human acceptance remains separate and is intentionally not marked true in execution state. Task 31 remains pending until Task 30 is explicitly accepted.