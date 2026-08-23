# Task 18 Report — Transactional Upgrade and Rollback

## Overview

Implementation checkpoint: `b0dd67d4dce8f6af0ee351dbd86ac772eef795b4`

Fresh gate evidence commit: `cc15b5c3d8f2202974eeaef53d5ded750185926c`

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

## Fresh acceptance evidence

GitHub Actions workflow:

- Workflow: `Task 18-19 Acceptance Gates`
- Run: `#24`
- Run ID: `32626853054`
- Source commit: `b0dd67d4dce8f6af0ee351dbd86ac772eef795b4`
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

## CI remediation found during final verification

The first joint Gate run exposed a real Linux CI build regression in `tui/internal/supervisor/process_unix.go`: the file was tagged `darwin` even though its process-group implementation is Unix-compatible. As a result Linux had no definitions for `configureProcess`, `attachProcess`, `detachProcess`, `terminateProcess` and `killProcess`.

This was fixed by using the non-Windows build constraint and the full Gate was rerun successfully. The Task 18/19 workflow path filters were also expanded to include supervisor/runtime/OpenCode dependencies so future changes to those dependencies cannot bypass the joint Gate.

## Current status

Task 18 automatic verification and Task Gate are **PASS**. Per the project execution-state contract, Task 18 is ready for `awaiting_acceptance`; it must not be marked `completed` until explicit human acceptance is recorded.

Task 19 has also been implemented and jointly verified, but remains formally `pending` until Task 18 is human-accepted because Task 18 is still the first incomplete task.

Task 20 remains pending and is not authorized to start.
