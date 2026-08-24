# Task 20 Report — 试点统计与反馈

## Overview

Implementation checkpoint: `e2a97095acc58dc0c95c4a091738181504d35f42`

Fresh Task 20/21 gate:

- Workflow: `Task 20-21 Acceptance Gates`
- Run: `#71`
- Run ID: `32700612749`
- Source commit: `e2a97095acc58dc0c95c4a091738181504d35f42`
- Result: **PASS**

Task 20 delivers local, privacy-preserving pilot metrics and lightweight task feedback. It records only operational metadata and deliberately excludes user Prompt, model answer, source code, Diff, full Tool I/O, API keys and absolute project paths.

## Delivered

### Metrics lifecycle

Implemented under `tui/internal/app/metrics.go` and `metrics_lifecycle.go`:

- session/task start and completion timestamps
- Agent type
- duration
- task status
- adoption status
- optional rating
- fixed error category
- loaded Skill IDs
- anonymous project/session/event identifiers

The event schema follows the V1 design contract and uses deterministic/irreversible hashing where a stable anonymous identifier is required.

### Privacy boundary

Persisted metrics contain metadata only. The implementation does not persist:

- user Prompt
- model full answer
- source code or Diff
- full Tool input/output
- API key or credential value
- absolute project path

Metrics files use restricted local permissions and best-effort persistence so telemetry failure does not block the user task.

### Adoption and feedback

Implemented under `tui/internal/app/feedback.go` and TUI integration:

- `Yes` → `accepted_as_is`
- `Partly` → `accepted_with_minor_changes`
- `No` → `rejected`
- `Esc` → skip without forcing feedback

The design enum `accepted_with_major_changes` remains available for programmatic adoption classification even though the lightweight three-choice prompt intentionally stays Yes/Partly/No.

### Skill loading evidence

Loaded Skill IDs are collected from runtime/tool evidence, normalized, deduplicated and sorted before persistence. This records only Skill identifiers, not Skill content.

### Terminal-path hardening

Final review found two lifecycle paths that could otherwise leave an active metric incomplete. Regressions now prove:

- SSE/event-stream closure finalizes the active metric as failed with `event_stream_closed`
- user `Ctrl+C` exit finalizes the active metric as failed with `user_quit`

Both paths clear the active metric state and preserve the application's existing exit/reconnect behavior.

## Verification

The fresh Task 20/21 Gate at checkpoint `e2a97095acc58dc0c95c4a091738181504d35f42` passed:

| Check | Result |
|---|---|
| Bun plugin regression/build | PASS |
| Release evidence tool regression | PASS |
| Go full regression | PASS |
| Locked OpenCode v1.18.11 download/checksum | PASS |
| Real distinct baseline/candidate parity | PASS — 12/12 Required scenarios |
| Parity evidence upload | PASS |

Task-specific Go regressions cover metadata-only serialization, anonymous IDs, adoption mapping, feedback skip, loaded Skill normalization, successful/failed lifecycle completion, stream closure and user quit.

## Current status

Task 20 implementation and automated verification are **PASS** at checkpoint `e2a97095acc58dc0c95c4a091738181504d35f42`.

Human acceptance has **not** been recorded by this report. The execution-state transition to `awaiting_acceptance` must keep `humanAccepted: false` until explicit user acceptance.
