# Task 23 Report — Runtime Workspace

## Overview

Task 23 Runtime Workspace is implementation-complete and accepted against:

`docs/superpowers/specs/2026-08-26-codea-v1.1-agent-workspace-design.md`

Implementation checkpoint:

`7dca79663bfefb45c144d7f8d03864d739d52996`

Fresh implementation Gate:

- Workflow: `Task 23 Runtime Workspace Gates`
- Run ID: `33030314432`
- Source commit: `7dca79663bfefb45c144d7f8d03864d739d52996`
- Result: **PASS**

Human acceptance:

- Accepted: **YES**
- Acceptance source: user handoff on 2026-08-27 explicitly states `Task 23：验收通过` and authorizes Task 24 start.

## Delivered scope

### Codea-owned Runtime contract

`AgentRuntime` is extended only with Codea-owned workspace capabilities:

- `ListModels(ctx context.Context) ([]runtime.Model, error)`
- `CompactSession(ctx context.Context, sessionID runtime.SessionID) error`

`PromptRequest.Model` remains the model-selection transport. No stateful vendor-style `SetModel` contract was introduced.

The model DTO crossing the Runtime boundary contains only safe routing/display metadata (`ModelRef`, name, provider display name, default flag). Provider credentials/options and OpenCode DTOs stay below the adapter.

### `/model`

`/model` is registered through the existing Command Registry and opens a current-session model picker.

Behavior:

- models come from the real Runtime `ListModels` capability;
- only connected/runtime-available OpenCode providers are exposed;
- selection is stored by `SessionID` in Codea application state;
- the selected `ModelRef` is attached to future `PromptRequest.Model` for that session;
- another/new session does not inherit the explicit selection;
- model selection is blocked during an incompatible in-flight response.

### Real OpenCode v1.18.11 model listing

The OpenCode adapter calls the real `GET /provider` endpoint and maps only Codea-owned safe metadata.

The locked OpenCode v1.18.11 source contract was cross-checked during Task 23 closeout: the provider list result contains `all`, `default`, and `connected`, matching the adapter parser used here.

### `/compact`

`/compact` performs real same-session Runtime compaction.

For OpenCode v1.18.11 the adapter:

1. resolves the current provider/model from the session's real message history;
2. calls `POST /session/:sessionID/summarize` with `providerID` and `modelID`;
3. keeps the same session;
4. returns a typed Runtime error instead of pretending success when compaction/model evidence is unavailable.

The locked OpenCode v1.18.11 source contract was cross-checked during closeout: `session.summarize` is the official summarize/AI-compaction endpoint, accepts `providerID`, `modelID`, optional `auto`, and returns a Boolean success result.

No TUI-history clear, fake summary, or silent new-session behavior is used.

### `/status`

The status command now renders Codea-owned workspace context including:

- Codea version
- Runtime provider/version/health
- Project
- Session
- Current Agent
- Current model
- Skill mode
- Loaded Skills
- Streaming support
- Reasoning support
- Tool approval support
- Compaction support

Raw provider configuration, API keys, credentials, and vendor DTOs are not rendered.

The real TUI composition root supplies project, Skill mode, Codea version, and `OpenCode` provider metadata.

### Shared `/doctor` service

CLI `codea doctor` and TUI `/doctor` now share the same `internal/doctor.Service` construction through `newDoctorService(...)`.

The TUI composition root injects that shared service into the application. There is no second health-only TUI Doctor implementation.

### Session workspace safety

Existing session APIs/history rehydration are reused.

- switching is blocked while an incompatible response is streaming;
- session resume clears transient answer/reasoning buffers, reasoning state, tool state, model picker state, pending prompt state, and active Agent state;
- rehydrated history belongs only to the selected session;
- explicit model selection remains isolated by session and does not bleed into another session.

### Existing workspace behavior reused

Task 23 does not duplicate existing implementations for:

- sessions/history;
- Agent listing;
- Skills;
- cancellation;
- Runtime health.

## Architecture boundaries preserved

- TUI/Application still depend on Codea-owned Runtime types only.
- OpenCode-specific HTTP/DTO parsing stays in `internal/opencode`.
- No second Runtime abstraction was introduced.
- OpenCode remains pinned at `v1.18.11`.
- Windows x64, pure-intranet, DLP, approval, Runtime lifecycle, and offline Plugin boundaries remain intact.
- The additions (`ListModels`, `CompactSession`) are Runtime capabilities that a future non-OpenCode adapter can implement without exposing OpenCode objects.

## Verification evidence

Fresh Gate `33030314432` on exact implementation checkpoint `7dca79663bfefb45c144d7f8d03864d739d52996`:

- `bash scripts/check-execution-state.sh` — **PASS**
- Task 23 focused Runtime/OpenCode/Command/Application tests — **PASS**
- Architecture vendor-boundary tests — **PASS**
- Full `go test ./... -count=1` — **PASS**
- `go vet ./...` — **PASS**
- `go build ./...` — **PASS**
- Windows x64 cross-build — **PASS**
- Windows Task 23 focused tests — **PASS**

TDD evidence also includes a deliberate RED Gate (`33030121625`) where the shared Doctor composition test failed only because `newDoctorService` did not yet exist. The subsequent production implementation and TUI wiring produced the green Gate above.

## Acceptance mapping

- Vendor DTO zero leakage — PASS
- `/model` lists real Runtime-available models — PASS
- model selection isolated per session — PASS
- selected model applied to later prompts — PASS
- new session uses Runtime/default model unless explicitly selected — PASS
- `/compact` performs real same-session compaction — PASS
- unsupported/indeterminate compaction fails explicitly — PASS
- session resume resets transient state — PASS
- `/doctor` and CLI Doctor share service implementation — PASS
- `/status` exposes useful sanitized workspace context — PASS
- OpenCode v1.18.11 adapter regressions — PASS
- future non-OpenCode adapter boundary preserved — PASS
- Windows Task 23 behavior/build — PASS

## Final status

**Task 23: COMPLETED / HUMAN ACCEPTED**

Task 24 Professional Agent Workspace is authorized to start. Task 23 implementation checkpoint remains `7dca79663bfefb45c144d7f8d03864d739d52996`; the acceptance closeout commit is administrative evidence only.
