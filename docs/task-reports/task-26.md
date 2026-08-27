# Task 26 Report — V1.1 Integration & Acceptance

Task 26 implementation and automated verification are complete and remain **awaiting human acceptance**. `humanAccepted` stays `false` until the user explicitly accepts Task 26.

Production implementation checkpoint: `488db3f03c6c9e11b6b5bba93ae8aa554a9cd2bb`

Current acceptance checkpoint: `3b77ed0f05b51c4cbcb870ceab8afd8e66a73d57`

Fresh Task 26 verification:

- Task 26 V1.1 Integration and Acceptance Gates — run `33082124762` — **PASS**
- Source: `3b77ed0f05b51c4cbcb870ceab8afd8e66a73d57`
- Linux `linux-v11-integration`: **PASS**
- macOS `macos-v11-integration`: **PASS**
- Windows `windows-v11-integration`: **PASS**

The acceptance checkpoint contains only Task 26 CI/test portability corrections after the production checkpoint; it does not change production Runtime, Supervisor, Agent, Application, policy, or product behavior.

## Integrated acceptance coverage

The final Gate verifies the approved Task 26 scope without adding product features:

- Tasks 22–25 focused regressions: PASS;
- architecture boundary `TUI -> Application -> AgentRuntime -> OpenCodeAdapter`: PASS;
- full Linux Go regression, race detector, vet, and build: PASS;
- enterprise plugin regression/build: PASS;
- Debug Agent contract: PASS;
- DLP / Tool Policy / Approval / path policy / Skill policy regressions: PASS;
- offline packaging and launcher contracts: PASS;
- `darwin-arm64`, `darwin-x64`, `windows-x64` release builds: PASS;
- Windows x64 cross-build: PASS;
- native macOS workspace/full Go regression: PASS;
- native Windows focused/full Go regression/build: PASS;
- locked OpenCode `v1.18.11` checksum/version: PASS;
- release parity G3–G14: PASS;
- distinct baseline/candidate dual Runtime parity: **12/12 PASS**.

## Windows Native Regression false-green remediation

### Finding

Human acceptance review found that the earlier Task 26 Windows job was falsely green. The workflow had placed both commands in the same PowerShell step:

```powershell
go test ./... -count=1
go build ./...
```

The Windows `go test` process really failed, but the subsequent successful `go build` became the PowerShell step's final exit status, so GitHub marked `windows-v11-integration` successful.

The affected earlier result must therefore **not** be used as Windows native regression evidence.

### Fail-closed CI RED evidence

Commit `7661f0a8271fdb4c9e96363ebcb2eb0b4a303d0a` split the Windows commands into independent steps:

- `Full native Windows Go regression` -> `go test ./... -count=1`
- `Full native Windows Go build` -> `go build ./...`

Fresh run `33081136988` then behaved correctly:

- Windows focused regression: PASS;
- `Full native Windows Go regression`: **FAIL**;
- `Full native Windows Go build`: SKIPPED;
- overall Windows job: **FAIL**.

This is the RED evidence proving the outer-job false-green path was removed before fixing the underlying Windows tests.

### Root causes fixed

The failures collapsed into cross-platform test/fixture assumptions rather than independent production Runtime defects:

1. **Windows user-home semantics**
   - tests had changed `HOME` and assumed `os.UserHomeDir()` would follow it on Windows;
   - test setup now uses native Windows `USERPROFILE` semantics while retaining `HOME` on Unix.

2. **Windows fake executable suffix**
   - fake OpenCode binaries were built/copied without `.exe`;
   - Windows test fixtures now preserve the required `.exe` suffix, including paths containing spaces.

3. **Windows `file://` URL semantics**
   - production correctly emits local-drive URLs as `file:///C:/...`;
   - the test now expects the standards-compliant `/C:/...` URL path on Windows rather than treating the leading slash as a product bug.

4. **Generated OpenAPI DTO determinism**
   - Windows checkout line-ending conversion could make the committed DTO differ byte-for-byte from `go/format` LF output;
   - `.gitattributes` now locks `tui/internal/opencode/dto.go` to LF on every platform, retaining the strict byte-for-byte stale-generated-file check.

5. **Immediate reasoning duration**
   - an immediate start/end can legitimately measure `0` on Windows wall-clock resolution;
   - the app lifecycle test now accepts zero while the reasoning tracker keeps its deterministic positive-duration test (`2.8s`) through the injected clock. No fake minimum duration or sleep was introduced.

6. **Unix-only executable-bit / path assumptions**
   - Windows does not expose Unix executable-bit preservation;
   - the Skill test still verifies whole-directory copy on Windows and verifies executable-bit preservation on Unix;
   - XDG isolation tests now compare against host-native `filepath.Join` paths instead of hard-coded Unix separators.

No production Supervisor/Runtime behavior was weakened to make these tests pass.

### Fresh Windows GREEN evidence

Run `33082124762`, Windows job `98551935288`, on `windows-2025` / Go `1.26.5`:

```text
Run $env:GOTOOLCHAIN='local'
go test ./... -count=1
...
ok  codea/tui/cmd/codea
ok  codea/tui/cmd/openapi-gen
...
ok  codea/tui/internal/app
ok  codea/tui/internal/opencode
ok  codea/tui/internal/reasoning
ok  codea/tui/internal/skill
ok  codea/tui/internal/supervisor
...
ok  codea/tui/tests/parity
```

The regression step completed **SUCCESS**. The workflow then entered a separate step:

```text
Run $env:GOTOOLCHAIN='local'
go build ./...
```

The build step also completed **SUCCESS**.

Therefore Windows native evidence is now based on two independent successful exit codes; a succeeding build can no longer mask a failing test run.

## Real Runtime evidence

### G6–G7

Real OpenCode Agent smoke: **15/15 PASS**. Production-source mutation protection remained intact.

### G8

Real API Documentation Agent flow (`extract -> validate -> write`): **9/9 PASS**. No fabricated schema fields accepted.

### G11–G13

Real OpenCode Runtime evidence: **17/17 PASS**.

The prior G11–G13 hang was teardown-only: Runtime checks had already passed but OpenCode could ignore TERM after cancel-in-streaming. The bounded cleanup is `TERM -> 3s grace -> SIGKILL` and runs only after the Runtime test result has completed, preventing an unbounded shell `wait` without converting Runtime failures into PASS.

### G12.1

Distinct baseline/candidate OpenCode `v1.18.11` parity: **12/12 PASS**.

### G14

Application handoff regression: PASS.

## Earlier timeout / cleanup TDD evidence

Task 26 also corrected two bounded-execution defects without expanding product scope:

- `752071b50cdaeff31cd828962e9b076620e89a9f` / run `33074632094`: RED contract for missing Runtime/parity deadlines;
- `2eecb22b3e8c42b9a8dd365dac81a1f55ece149e`: bounded/labeled real Runtime subflows;
- `4cc8cfd32b7b48cceec607ed4f81501fe0a4abac`: bounded dual Runtime parity runner;
- `f84daf612ea8c8d9199d7b85f2bacdadc81367b3` / run `33077534770`: RED bounded-cleanup contract;
- `488db3f03c6c9e11b6b5bba93ae8aa554a9cd2bb`: bounded TERM -> SIGKILL cleanup;
- run `33077649260`: original production-checkpoint Linux/macOS/Windows integration Gate PASS, with G11–G13 17/17 and G12.1 12/12.

## Existing V1 intranet-only release evidence

Task 26 does **not** rewrite previously deferred company-intranet-only V1 release gates as PASS. Those gates remain deferred until executed in the required company network/mirror environment.

## Human acceptance

- Accepted: **NO — pending**
- Automated verification: **PASS**
- Task Gate: **PASS**
- Production implementation checkpoint: `488db3f03c6c9e11b6b5bba93ae8aa554a9cd2bb`
- Current acceptance checkpoint: `3b77ed0f05b51c4cbcb870ceab8afd8e66a73d57`
- Fresh automated Gate: `33082124762` — **PASS**
- Current state: `awaiting_acceptance`

Current status: **IMPLEMENTATION COMPLETE / AUTOMATED GATE PASS / AWAITING HUMAN ACCEPTANCE**
