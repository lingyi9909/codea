# Task 4 Report — Runtime Resilience & Recovery

## Overview

Checkpoint: 1423a7396484ad2f2946606bb52d5f029e6020d9

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
