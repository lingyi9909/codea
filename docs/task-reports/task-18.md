# Task 18 Report — Transactional Upgrade and Rollback

## Overview

Implementation checkpoint: `372aa8387257f3018f11d4404d3888fbc5136e0c`

Fresh gate evidence commit: `46e1d31a700de8f85474c64aeeef18ef65bfa8ce`

Task 18 implements the V1 transactional upgrade, migration, rollback and crash-recovery path on top of the Task 17 installed layout. The update transaction is fail-closed: a candidate package must pass package/manifest/hash validation, configuration migration and Candidate Doctor checks before the installed version can be committed.

## Delivered

Core implementation lives under `tui/internal/update/` and includes:

- package/manifest/SHA256 verification
- version directory management
- C1 -> C2-temp configuration migration
- migration registry and missing-hop rejection
- transaction journal with pending/committed/rolled-back states
- atomic current-version switching for Unix and Windows
- cross-process update locking
- upgrade-in-progress launch marker
- crash recovery and rollback entry path
- transaction/staging scratch cleanup
- downgrade rejection
- Candidate Doctor as a mandatory pre/post switch checker

Supporting integration includes the macOS/Windows launcher contract, `current.txt` authority on Windows, `CODEA_HOME`/runtime-config wiring, upgrade scenario scripts, and recovery through `codea doctor` after an interrupted update.

## Safety properties verified

- No silent fallback to a weak/basic candidate checker.
- Invalid or tampered package contents fail closed.
- A migration chain cannot skip a required schema hop.
- Failed pre-switch or post-switch checks do not commit the candidate.
- A crash in the current-pointer transition window can be recovered from the journal and real filesystem state.
- Normal launch is blocked while an update transaction exposes an intermediate version/config state.
- Stale update markers can be recovered without interfering with a still-live updater because recovery must acquire the real update lock.
- Normal plugin materialization preserves migrated `model`, `provider` and other OpenCode configuration fields instead of rebuilding `opencode.json`.
- Invalid existing `opencode.json` fails closed and is not overwritten.

## Shared Task 18/19 blocker remediation

Final human review found one shared implementation blocker in the normal startup path: `writePluginConfig()` rebuilt `opencode.json` with only the `plugin` field. A successful C2 migration could therefore be committed correctly and then lose `model`, `provider` and other migrated configuration on the next normal Codea launch.

The remediation is intentionally bounded:

- added shared `opencode.MergePluginConfig(...)` semantics
- normal startup/`codea doctor` now read existing `opencode.json`, preserve every non-Codea field and update only the Codea-managed `plugin` field
- Candidate Doctor now reuses the same shared merge helper, removing the previous duplicate implementation
- invalid existing JSON returns an error before any write
- added regressions for model/provider/custom field preservation, invalid-JSON fail-closed behavior, and migrated C2 preservation through normal plugin materialization

TDD RED was confirmed in Task 18-19 Gate run `#25` / run ID `32628166891`: all three new regressions failed against the old implementation for the expected reasons. The final implementation then passed the full fresh Gate.

## Fresh acceptance evidence

GitHub Actions workflow:

- Workflow: `Task 18-19 Acceptance Gates`
- Run: `#28`
- Run ID: `32628284824`
- Source commit: `372aa8387257f3018f11d4404d3888fbc5136e0c`
- Result: **PASS**

Persisted evidence: `tests/evidence/task18-19-gate-evidence.json`.

Task 18 relevant gates from that run:

| Gate | Result |
|---|---|
| Go full regression | PASS |
| Go race | PASS |
| go vet | PASS |
| go build | PASS |
| macOS arm64 cross-build | PASS |
| Windows x64 cross-build | PASS |
| Upgrade scenarios | PASS — 4/4 |
| Launcher contract | PASS |
| Task 17 packaging regression | PASS |
| Execution-state validator | PASS |
| Existing Agent Runtime regression | PASS |
| API Documentation Runtime regression | PASS |
| Candidate Doctor / OpenCode v1.18.11 | PASS |

The Candidate Doctor evidence records `runtimeIsolation: true`, `candidateDoctorPassed: true`, and OpenCode version `1.18.11`.

## Earlier CI remediation

An earlier joint Gate run exposed a Linux CI build regression in `tui/internal/supervisor/process_unix.go`: the file was tagged `darwin` even though its process-group implementation is Unix-compatible. Linux therefore had no definitions for the process lifecycle helpers. This was fixed with a non-Windows build constraint, and the Task 18/19 workflow path filters were expanded to include supervisor/runtime/OpenCode dependencies.

## Current status

Task 18 automatic verification and Task Gate are **PASS** at checkpoint `372aa8387257f3018f11d4404d3888fbc5136e0c`. Per the execution-state contract, Task 18 is `awaiting_acceptance`; it must not be marked `completed` until explicit human acceptance is recorded.

Task 19 has also been implemented and jointly verified at the same checkpoint, but remains formally `pending` until Task 18 is human-accepted because Task 18 is still the first incomplete task.

Task 20 remains pending and is not authorized to start.
