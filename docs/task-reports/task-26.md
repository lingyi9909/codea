# Task 26 Report — V1.1 Integration & Acceptance

Task 26 implementation, automated verification, and human acceptance are complete.

Production implementation checkpoint: `488db3f03c6c9e11b6b5bba93ae8aa554a9cd2bb`

Final verified acceptance head: `a85ef424f463837a6bf72239923940b185069b96`

## Final automated evidence

Current-head Task 26 closeout Gate `33083017554` completed successfully on the final acceptance head.

- Linux `linux-v11-integration`: PASS
- macOS `macos-v11-integration`: PASS
- Windows `windows-v11-integration`: PASS
- Tasks 22–25 focused regressions: PASS
- architecture boundary `TUI -> Application -> AgentRuntime -> OpenCodeAdapter`: PASS
- full Linux Go regression / race / vet / build: PASS
- enterprise plugin regression/build: 247/247 PASS
- Debug Agent contract: PASS
- DLP / Tool Policy / Approval / path policy / Skill policy: PASS
- offline packaging and launcher contracts: PASS
- release builds: `darwin-arm64`, `darwin-x64`, `windows-x64` PASS
- Windows x64 cross-build: PASS
- locked OpenCode `v1.18.11` checksum/version: PASS
- Real Agent smoke G6–G7: 15/15 PASS
- API Documentation G8: 9/9 PASS
- Real Runtime G11–G13: 17/17 PASS
- Dual Runtime Parity G12.1: 12/12 PASS
- G3–G14: PASS

## Windows Native Regression false-green remediation

Human review found that the earlier Windows Gate placed `go test ./... -count=1` and `go build ./...` in one PowerShell step, allowing a successful build to overwrite a failing test exit code.

The workflow was corrected so Windows uses independent fail-closed steps:

- `Full native Windows Go regression` -> `go test ./... -count=1`
- `Full native Windows Go build` -> `go build ./...`

RED evidence: run `33081136988` correctly failed the Windows job when the independent test step failed.

The underlying Windows portability failures were then fixed without weakening production Runtime/Supervisor behavior: native user-home semantics, `.exe` fake-runtime fixtures, Windows `file://` URL expectations, generated DTO LF determinism, zero-duration immediate reasoning, executable-bit portability, and host-native XDG/OpenCode/Claude Skill-root path assertions.

Fresh GREEN evidence on the final acceptance head shows both independent Windows steps PASS. A succeeding build can no longer mask a failing test run.

## Runtime timeout / teardown remediation

Task 26 also corrected unbounded real Runtime/parity execution and teardown:

- bounded/labeled real Runtime subflows;
- bounded dual Runtime parity runner;
- OpenCode teardown changed to `TERM -> 3s grace -> SIGKILL` only after Runtime evidence has already completed.

The final head verifies G11–G13 17/17 PASS before bounded teardown and G12.1 12/12 PASS.

## Deferred intranet-only evidence

Previously deferred company-intranet-only Task 17 / G15 native-offline evidence remains deferred. Task 26 does not rewrite those deferred gates as PASS.

## Human acceptance

- Accepted: **YES**
- Acceptance source: user explicitly stated `验收通过`
- Automated verification: **PASS**
- Task Gate: **PASS**
- Final verified acceptance head: `a85ef424f463837a6bf72239923940b185069b96`
- Final current-head Gate: `33083017554` — **PASS**

Current status: **COMPLETED / HUMAN ACCEPTED**
