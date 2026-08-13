# Task 4 Report — Runtime Resilience & Recovery

## Overview

Checkpoint: 3a737da675eb1f3f4752aaf150de75fbbda8bba1

补齐生产级 SSE 恢复、状态补偿、错误分类、背压与真实 Runtime 长连接验证。

## Step 1 — Runtime Error Model

- Created `tui/internal/runtime/errors.go` with 7 error kinds:
  Transport, Auth, Protocol, Incompatible, Recovery, Backpressure, Cancelled
- Each kind has factory function (NewTransportError, etc.) and predicate (IsTransport, etc.)
- `IsRetryable` predicate for application-level retry decisions
- `VendorDetails json.RawMessage` preserves vendor raw error data
- Enhanced `RuntimeError` with `Error()` and `Unwrap()` for error interface
- Created `tui/internal/runtime/errors_test.go` with 17 tests
- Updated `event_mapper.go` to set `Kind: RuntimeErrorProtocol, Operation: "EventMap"`

## Step 2 — SSE Disconnect Detection + Reconnect Policy

- Created `tui/internal/opencode/reconnect.go`:
  - `Backoff(attempt)` — 500ms, 1s, 2s, 5s (capped)
  - `IsRetryableHTTP(statusCode, err)` — auth errors non-retryable, transport/5xx retryable
  - `ReconnectingSSEClient` — wraps SSE subscribe with auto-reconnect and backoff
  - `IsSSEDisconnect(ev)` — detects DISCONNECTED, CONNECT_FAILED, SCANNER_ERROR
  - `DisconnectReason(ev)` — human-readable disconnect info
  - Monotonic global sequence counter across reconnections
  - Optional `ReconnectHook` for post-reconnect recovery
- Created `tui/internal/opencode/reconnect_test.go` with 10 tests:
  - Backoff sequence validation
  - HTTP status retryability (transport, 5xx, 401, 403, 200)
  - Normal stream (no disconnect events when context cancelled)
  - Backoff and retry (multiple reconnection attempts)
  - Context cancellation during backoff
  - 401 no-retry (single attempt, channel closes)
  - Transport error retry (fails twice, succeeds third)
  - Backoff counter reset after successful events
  - Disconnect event detection
  - Monotonic sequence across reconnections

## Step 3 — Recovery / State Compensation

- Created `tui/internal/opencode/recovery.go`:
  - `SessionTracker` — tracks local state of sessions/messages/parts from events
  - `Recover(ctx, httpClient)` — queries OpenCode HTTP API after reconnect:
    1. `GET /session/status` — detect new sessions
    2. `GET /session/:id/message` — find missed messages per session
    3. Emit compensation events (session.created, message.updated)
    4. Emit recovery-complete marker (runtime.connected with recovered=true)
  - `extractMessageIDs` — extracts message/part IDs from OpenCode message union type
  - `Record(ev)` — updates local state from observed events
- Added HTTP client methods: `GetSessionStatus`, `GetSessionMessages`, `GetSession`
- Modified `adapter.go` to use `ReconnectingSSEClient` and integrate recovery:
  - Detect disconnect events in the stream
  - After reconnect, inject recovery compensation events before live events
  - Track all events in `SessionTracker`
- Created `tui/internal/opencode/recovery_test.go` with 7 tests:
  - SessionTracker record and sequence tracking
  - Empty session skip
  - Message ID extraction (with/without content parts, missing ID)
  - Recovery detects new sessions and messages
  - Recovery skips already-known messages (dedup)
  - Recovery handles API errors gracefully
  - Adapter Subscribe uses reconnecting client with disconnect events

## Step 4 — Bounded Backpressure

- Channel buffer: 16 events — limits memory usage
- Blocking send policy — no silent event drops; backpressure propagates upstream
- Context cancellation as clean shutdown path
- Test: slow consumer receives all 30 events, proving no-drop behavior

## Step 5 — Recovery + Approval + Abort Contract

- Created `tui/tests/contract/runtime_recovery_test.go`:
  - Full contract test against real OpenCode:
    - Health, Capabilities, ListAgents verification
    - Subscribe with reconnecting client
    - Prompt and event collection with recovery markers
    - Auto-approval flow (ReplyApproval)
    - Cancel (abort) with idempotency check
  - Skips gracefully when OpenCode not running

## Step 6 — Full Gate Verification

All gates pass:
- `go test ./... -count=1` — all tests pass (13 packages)
- `go test -race ./... -count=1` — no race conditions
- `go vet ./...` — clean
- `go build ./...` — clean
- Windows cross-build — clean
- `check-runtime-boundary.sh` — PASS
- `check-execution-state.sh` — valid state
- `check-opencode-available.sh` — OpenCode available
- `state_validator_test.sh` — valid

## Round 5 Review Fixes (2026-08-12) — 4 Blocking Issues

### Blocking 1: Part-level Recovery (SilentLoss)

- Modified `Recover()` in `recovery.go` to emit `part.updated` compensation events for
  missing Parts on known Messages
- Added `partRecoveryEvent()` function
- Added tests: `TestRecoveryCompensatesMissingPart`, `TestRecoveryDedupesKnownPart`

### Blocking 2: Recovery Timing

- Moved recovery invocation to `ReconnectHook`, which fires immediately after SSE
  connection success, before draining live events
- Created recovery event wrapper (`recoveryEventWrapper` / `wrapRecoveryEvent` /
  `unwrapRecoveryEvent`) to transport `runtime.Event` through the `SSERawEvent` channel
- Added `MakeRecoveryHook` on `SessionTracker`
- Updated `adapter.go` Subscribe to unwrap recovery events directly

### Blocking 3: RuntimeError Propagation + 400/404 Fix

- Added `HTTPError` struct type in `http_client.go` with StatusCode, Method, Path, Body
- Modified `HTTPClient.do()` to return `*HTTPError` for non-expected status codes
- Modified `SSEClient.Subscribe()` to return `*HTTPError` for non-200 responses
- Added `classifyError()` in `adapter.go`: context.Canceled→Cancelled, 401/403→Auth,
  5xx→Transport, 4xx→Protocol, network→Transport
- All 7 AgentRuntime API methods now wrap errors via `classifyError`
- Updated `extractHTTPStatus()` in `reconnect.go` to use `errors.As` for `*HTTPError`
- Updated `IsRetryableHTTP()` to classify all 4xx as non-retryable
- Auth errors (401/403) now emit AUTH_ERROR event before closing channel (no more silent drops)
- Added `runtimeErrorKindFromCode()` in `event_mapper.go` to map SSE error codes to
  RuntimeErrorKind (AUTH_ERROR→Auth, DISCONNECTED/CONNECT_FAILED/SCANNER_ERROR→Transport)
- Tests added: `TestSSE400NoRetry`, `TestSSE404NoRetry`, `TestReconnectingClient401EmitsAuthEvent`,
  `TestSSE401EmitsAuthRuntimeError`, `TestHTTPAdapterTransportErrorClassification`,
  `TestHTTPAdapterCancelledClassification`, `TestHTTPAdapterProtocolErrorClassification`

### Blocking 4: Contract Test Actually Verifies Recovery

- Added `TestAgentRuntimeRecoveryContract` — forces SSE disconnect, provides recovery data
  via `/session/status` + `/session/:id/message`, asserts `seenDisconnect && seenRecovery &&
  seenApprovalReq` all true

### Same-round Fixes

- Backpressure test precision: verifies exactly 30 domain events + backpressure errors > 0
- `nextAction` in execution-state.yaml: updated to "等待人工验收"

## Round 6 Review Fixes (2026-08-13) — 2 Blocking Issues

### Blocking 1: Real OpenCode Disconnect/Reconnect/Recovery Required Gate

Real OpenCode v1.18.11 gate (not `httptest.Server`). Added
`tui/tests/contract/real_recovery_contract_test.go` which drives a
`disconnectProxy` against the live server to force SSE termination and verify
the full cycle.

Root-cause fixes required to make the real gate pass:

- **Nested ID extraction** (`event_mapper.go`): real OpenCode nests message/part
  IDs under `properties.info.id` (`message.updated`) and `properties.part.*`
  (`message.part.updated`); the mapper previously read only flat top-level
  fields, so recovered parts carried empty IDs.
- **Real API shape** (`http_client.go`, `recovery.go`): `GET /session` returns a
  raw array (not `{data:[...]}`), and `GET /session/:id/message` returns
  `{info:{id}, parts:[{id}]}`. Recovery now parses both shapes.
- **Initial-connect suppression** (`reconnect.go`): `ReconnectHook` fires only
  on reconnects, not the first connect, preventing every historical session from
  being re-emitted as new on startup.

Real evidence (run against OpenCode v1.18.11 at `127.0.0.1:14242`):

```
Phase 1: totalEvents=394 msgs=259 parts=5 recoveredMsgs=257 recoveredParts=2
         curRecoveredMsgs=1 curRecoveredPts=2 answer=true tool=true
         disconnected=true recovery=true
No-duplicate check: 259 unique message IDs tracked
Phase 2: approval=true approved=true events=14   (external_directory → ask)
Cancel (post-reconnect): ok
Cancel 2 (post-reconnect): ok (idempotent)
```

This demonstrates: real `/global/event` → forced disconnect → reconnect →
real session/message history recovery → missing message/part compensation
(`curRecoveredMsgs=1`, `curRecoveredPts=2`) → no-duplicate/no-loss (259 unique
IDs) → Approval/Cancel after reconnect.

### Blocking 2: Checkpoint Layering

Checkpoint now points to the Final Implementation Commit
`338591f0d3f78c490fe05415bba5e6fe11a26741` (the round-6 code+test commit), not
an evidence/docs/state commit. `docs/execution-state.yaml` and this report both
reference the same SHA.

## Round 7 Review Fixes (2026-08-13) — 3 Blocking Issues

### Blocking 1: Non-retryable SSE failure must emit explicit runtime.error

`reconnect.go` previously emitted an Auth RuntimeError only for 401/403 and
silently closed the channel for any other non-retryable status (400/404). The
Application therefore saw a normal stream end instead of a Protocol error.

- All non-retryable subscribe failures now emit a terminal `runtime_error`
  before closing: 401/403 → `AUTH_ERROR`, every other non-retryable status →
  `PROTOCOL_ERROR`.
- `sendEvent` now blocks (returns early on context cancel) so the terminal
  error is guaranteed delivered before the channel closes.
- Tests: `TestSSE400NoRetry` / `TestSSE404NoRetry` assert a `PROTOCOL_ERROR`
  event arrives before close; `TestSSE400EmitsProtocolRuntimeError` verifies the
  full adapter chain maps it to a `RuntimeError(Protocol)`.

### Blocking 2: Recovery vs live SSE duplicate events

Recovery's REST query runs while live SSE events enter the reconnect buffer,
so a message/part present in the snapshot could also arrive as a live event and
be delivered twice.

- `SessionTracker` now tracks the message/part IDs compensated by the most recent
  `Recover()` pass in `recoveredMsgIDs` / `recoveredPartIDs`.
- `ShouldSuppressLive` provides suppress-once semantics: the first live
  `message.updated` / `part.updated` matching a recovered ID is dropped; a
  subsequent genuine update for the same ID passes through.
- `adapter.go` consults `ShouldSuppressLive` in the live-event path.
- Tests: `TestShouldSuppressLiveDedup` (unit) and `TestAdapterDedupsRecoveryAndLiveMessage`
  (integration) — the latter constructs the exact race (message M in the recovery
  snapshot AND its live `message.updated` already in the post-reconnect buffer)
  and asserts the Application receives M exactly once.

### Blocking 3: VendorDetails actually populated from vendor errors

`RuntimeError.VendorDetails` was defined but never written.

- `classifyError` (`adapter.go`) now serializes `HTTPError` (statusCode, method,
  path, body) into `VendorDetails` for every classified HTTP error.
- `extractError` (`event_mapper.go`) now sets `VendorDetails` to the raw vendor
  error JSON for all five `RuntimeError` construction sites.
- Tests: `TestHTTPAdapterErrorVendorDetails` (real HTTP→Adapter conversion chain)
  and `TestEventMapperErrorVendorDetails` (real SSE→Mapper conversion chain) prove
  the field is auto-populated by the Adapter/Mapper, not by the test itself.

## Spec Review Fixes (2026-08-12)

Post-implementation review against the 10-section acceptance criteria:

1. **Backpressure — explicit RuntimeError(Backpressure)**: `sendRuntimeEvent` now detects channel-full via non-blocking send, block-sends a `RuntimeError(Backpressure)` event before blocking on the original event. Zero silent drops preserved. Test verifies backpressure error events are emitted.

2. **Duplicate SSEClient**: Removed unused `sseClient` field from `OpenCodeAdapter` struct. Only the `ReconnectingSSEClient` instance is needed.

3. **Contract test OpenCode gate**: Added `scripts/check-opencode-available.sh` (exit 0 = available, exit 2 = unavailable). When OpenCode is unavailable, verification should be `unable_to_run` per spec. Added to verification command list.

4. **Test semantics — backoff reset**: `TestBackoffCounterResetAfterSuccess` now verifies that backoff resets to 500ms after each successful connection by asserting at least 5 reconnect cycles within a 6-second window. Without reset, accumulated backoff (500ms → 1s → 2s → 5s) would prevent this.

## Test Summary

| Package | Tests |
|---------|-------|
| internal/runtime | 17 (errors) + existing |
| internal/opencode | 10 (reconnect) + 7 (recovery) + 1 (backpressure) + existing |
| tests/contract | 1 (recovery contract) |

## Files Changed

| File | Action |
|------|--------|
| `tui/internal/runtime/errors.go` | Create |
| `tui/internal/runtime/errors_test.go` | Create |
| `tui/internal/runtime/events.go` | Modify |
| `tui/internal/opencode/event_mapper.go` | Modify |
| `tui/internal/opencode/reconnect.go` | Create |
| `tui/internal/opencode/reconnect_test.go` | Create |
| `tui/internal/opencode/recovery.go` | Create |
| `tui/internal/opencode/recovery_test.go` | Create |
| `tui/internal/opencode/adapter.go` | Modify |
| `tui/internal/opencode/adapter_test.go` | Modify |
| `tui/internal/opencode/http_client.go` | Modify |
| `tui/tests/contract/runtime_recovery_test.go` | Create |
| `scripts/check-opencode-available.sh` | Create |
