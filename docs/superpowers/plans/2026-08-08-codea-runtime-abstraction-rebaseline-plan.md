# Codea Runtime Abstraction Rebaseline Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Put the Task 2 OpenCode vendor client behind a Codea-owned Runtime Contract before Task 3 starts, with zero vendor DTO leakage and zero silent SSE event loss.

**Architecture:** `tui/internal/runtime` owns domain types and the `AgentRuntime` interface. `tui/internal/opencode` implements the interface using the generated DTO and HTTP client from Task 2. Runtime process lifecycle remains in Task 5 Supervisor; Task 3 consumes the new contract for Capability Inventory and Parity Harness.

**Tech Stack:** Go 1.26.5, standard library HTTP/SSE/JSON, locked OpenCode v1.18.11 OpenAPI 3.1 Spec, Bash/Python state validation.

## Global Constraints

- V1 implements only OpenCode; do not add OMP, routing, selection or switching.
- All Go code and Go tests remain under `tui/`.
- Generated OpenCode DTOs remain Spec-driven and are never copied into Domain types by embedding or aliases.
- `Start`, `Stop` and process state remain outside `AgentRuntime` and belong to Task 5 Supervisor.
- Every OpenCode SSE event is mapped or returned as Raw; silent drop count must remain zero.
- Raw payload follows the existing 500-event memory, 16KB display, DLP and no-default-audit rules.
- Use TDD for every behavior: observe RED, add the minimum implementation, then observe GREEN.
- Do not modify OpenCode Core.
- Task 3 remains `pending` until Task 2A passes verification, Task Gate and human acceptance.

---

### Task 2A.1: Codea Runtime Domain and Contract

**Files:**
- Create: `tui/internal/runtime/models.go`
- Create: `tui/internal/runtime/events.go`
- Create: `tui/internal/runtime/approval.go`
- Create: `tui/internal/runtime/capabilities.go`
- Create: `tui/internal/runtime/client.go`
- Create: `tui/internal/runtime/client_test.go`

**Interfaces:**
- Consumes: no OpenCode packages or DTOs
- Produces: `AgentRuntime`, `SessionID`, `ApprovalID`, `HealthInfo`, `Session`, `CreateSessionRequest`, `PromptRequest`, four Prompt Part variants, `Event`, `ApprovalDecision`, `RuntimeCapabilities`, `RuntimeError`

- [ ] **Step 1: Write compile-time and enum tests**

Create tests in package `runtime` that assert:

```go
func TestApprovalDecisionValues(t *testing.T) {
	cases := map[ApprovalDecision]string{
		ApprovalOnce: "once", ApprovalAlways: "always", ApprovalReject: "reject",
	}
	for decision, want := range cases {
		if string(decision) != want { t.Fatalf("%q != %q", decision, want) }
	}
}

func TestPromptPartVariantsSatisfyContract(t *testing.T) {
	var parts = []PromptPart{TextPart{}, FilePart{}, AgentPart{}, SubtaskPart{}}
	if len(parts) != 4 { t.Fatal("expected four prompt part variants") }
}
```

- [ ] **Step 2: Run the focused tests and confirm RED**

Run: `cd tui && go test ./internal/runtime -run 'TestApprovalDecisionValues|TestPromptPartVariantsSatisfyContract' -count=1`

Expected: FAIL because the runtime package/types do not exist.

- [ ] **Step 3: Implement the domain types and interface**

Use this exact interface:

```go
type AgentRuntime interface {
	Health(context.Context) (HealthInfo, error)
	CreateSession(context.Context, CreateSessionRequest) (Session, error)
	Prompt(context.Context, SessionID, PromptRequest) error
	Subscribe(context.Context) (<-chan Event, error)
	ReplyApproval(context.Context, ApprovalID, ApprovalReply) error
	Cancel(context.Context, SessionID) error
	ListAgents(context.Context) ([]Agent, error)
	Capabilities() RuntimeCapabilities
}
```

Model Prompt variants from the locked generated inputs:

- Text: ID, Text, Synthetic, Ignored, Metadata
- File: ID, MIME, URL, Filename and typed source data
- Agent: ID, Name and optional source range
- Subtask: ID, Agent, Description, Prompt, Command and optional ModelRef

Do not add Start/Stop, Runtime name checks, OMP fields or application task events.

- [ ] **Step 4: Run runtime tests and confirm GREEN**

Run: `cd tui && go test ./internal/runtime -count=1`

Expected: PASS.

- [ ] **Step 5: Commit the domain contract**

```bash
git add tui/internal/runtime
git commit -m "feat: define Codea runtime contract"
```

---

### Task 2A.2: OpenCode Request and Approval Mapping

**Files:**
- Create: `tui/internal/opencode/request_mapper.go`
- Create: `tui/internal/opencode/request_mapper_test.go`
- Create: `tui/internal/opencode/approval_mapper.go`
- Create: `tui/internal/opencode/approval_mapper_test.go`

**Interfaces:**
- Consumes: Codea Domain requests and Task 2 generated OpenCode request DTOs
- Produces: pure mapping functions used by `OpenCodeAdapter`

- [ ] **Step 1: Write request mapping tests**

Cover all four Prompt Part variants, model provider/model IDs, optional IDs, CreateSession fields and metadata. Assert JSON after mapping to generated DTOs so required field names come from `dto.go`, not a duplicate fixture struct.

- [ ] **Step 2: Write Approval mapping tests**

Use a table for once, always and reject. Assert:

```go
got := mapApprovalReply(runtime.ApprovalReply{
	Decision: runtime.ApprovalReject,
	Message:  "denied by user",
})
if got.Reply != "reject" || got.Message != "denied by user" { t.Fatalf("%+v", got) }
```

Also marshal the DTO and assert the JSON has no `remember` field.

- [ ] **Step 3: Run mapping tests and confirm RED**

Run: `cd tui && go test ./internal/opencode -run 'TestMap(CreateSession|Prompt|Approval)' -count=1`

Expected: FAIL because mapper functions do not exist.

- [ ] **Step 4: Implement the minimum pure mappers**

Use exhaustive type switches over `runtime.PromptPart`. Return a typed error for unsupported/nil variants; never silently omit a part. Set OpenCode discriminators (`text`, `file`, `agent`, `subtask`) in the Vendor DTO only.

- [ ] **Step 5: Run mapping tests and confirm GREEN**

Run: `cd tui && go test ./internal/opencode -run 'TestMap(CreateSession|Prompt|Approval)' -count=1`

Expected: PASS.

- [ ] **Step 6: Commit the request mappers**

```bash
git add tui/internal/opencode/request_mapper.go tui/internal/opencode/request_mapper_test.go tui/internal/opencode/approval_mapper.go tui/internal/opencode/approval_mapper_test.go
git commit -m "feat: map Codea runtime requests to OpenCode"
```

---

### Task 2A.3: SSE Transport and Event Mapper

**Files:**
- Create: `tui/internal/opencode/sse_client.go`
- Create: `tui/internal/opencode/sse_client_test.go`
- Create: `tui/internal/opencode/event_mapper.go`
- Create: `tui/internal/opencode/event_mapper_test.go`
- Consume: `runtime/openapi/golden-sse-s2.jsonl`

**Interfaces:**
- Consumes: authenticated `/global/event` SSE and generated OpenCode event DTOs
- Produces: ordered `runtime.Event` values with Raw data preserved

- [ ] **Step 1: Write SSE parser tests**

Use `httptest.Server` and cover multiple `data:` lines, comments, blank event separators, payloads larger than Scanner's default buffer, non-200 responses, context cancellation and a server-side truncated stream.

- [ ] **Step 2: Write Golden SSE mapping tests**

For every non-empty Golden SSE record, assert exactly one result:

```go
event, err := mapper.Map(raw, sequence)
if err != nil { t.Fatalf("map event %d: %v", sequence, err) }
if len(event.Raw) == 0 { t.Fatalf("event %d lost raw payload", sequence) }
if event.Type == "" { t.Fatalf("event %d has no mapped or unknown type", sequence) }
```

Add explicit unknown-type, malformed-JSON and over-16KB cases. Unknown must preserve `RawType`; malformed JSON must use `_unparseable_` and preserve exact bytes within the limit. Oversized Raw must set `RawTruncated=true` and the original byte count.

- [ ] **Step 3: Run SSE tests and confirm RED**

Run: `cd tui && go test ./internal/opencode -run 'Test(SSE|EventMapper|Golden)' -count=1`

Expected: FAIL because SSEClient and EventMapper do not exist.

- [ ] **Step 4: Implement SSEClient and EventMapper**

Requirements:

- Request `GET /global/event` with Basic Auth and `Accept: text/event-stream`.
- Reject non-200 status with the same 64KB error-body bound as the HTTP client.
- Keep Subscription global; do not accept a SessionID parameter.
- Preserve monotonically increasing sequence numbers per connection.
- Return pre-subscription HTTP/auth failures directly. After the channel is established, emit one `runtime_error` Event for Scanner/network failure before closing. Context cancellation closes cleanly without leaking a goroutine.
- Preserve Raw bytes for both known and unknown events.
- Map from actual v1.18.11 event envelopes/Goten SSE, not the old hand-written snake_case sample.

- [ ] **Step 5: Run SSE tests and confirm GREEN**

Run: `cd tui && go test ./internal/opencode -run 'Test(SSE|EventMapper|Golden)' -count=1`

Expected: PASS with one output event per Golden SSE input event.

- [ ] **Step 6: Commit SSE and mapping**

```bash
git add tui/internal/opencode/sse_client.go tui/internal/opencode/sse_client_test.go tui/internal/opencode/event_mapper.go tui/internal/opencode/event_mapper_test.go
git commit -m "feat: map OpenCode SSE to Codea events"
```

---

### Task 2A.4: OpenCodeAdapter and Runtime Capabilities

**Files:**
- Create: `tui/internal/opencode/adapter.go`
- Create: `tui/internal/opencode/adapter_test.go`
- Create: `tui/internal/opencode/capabilities.go`
- Create: `tui/internal/opencode/capabilities_test.go`

**Interfaces:**
- Consumes: Task 2 `HTTPClient`, Task 2A mappers/SSEClient and `runtime.AgentRuntime`
- Produces: `OpenCodeAdapter` implementing the complete Codea contract

- [ ] **Step 1: Write a compile-time interface assertion and adapter behavior tests**

```go
var _ runtime.AgentRuntime = (*OpenCodeAdapter)(nil)
```

Use `httptest.Server` to cover Health, CreateSession, Prompt, global Subscribe, ReplyApproval, Cancel and ListAgents. Assert endpoint paths and exact success status codes from Task 2.

- [ ] **Step 2: Write capability declaration tests**

Assert OpenCode declares only the verified V1 keys in `runtime/capabilities.yaml`: sessions, streaming, reasoning, fileRead, fileWrite, edit, bash, toolApproval, agents, subagents, skills, plugins, abort, messageHistory and contextCompaction.

- [ ] **Step 3: Run adapter tests and confirm RED**

Run: `cd tui && go test ./internal/opencode -run 'Test(OpenCodeAdapter|OpenCodeCapabilities)' -count=1`

Expected: FAIL because the adapter and declaration do not exist.

- [ ] **Step 4: Implement OpenCodeAdapter**

Compose existing HTTPClient, SSEClient and pure mappers. Do not duplicate HTTP transport logic and do not expose generated DTOs from public adapter methods.

- [ ] **Step 5: Run all OpenCode tests and confirm GREEN**

Run: `cd tui && go test ./internal/opencode -count=1`

Expected: PASS, including all unchanged Task 2 client and generator-drift tests.

- [ ] **Step 6: Commit the adapter**

```bash
git add tui/internal/opencode/adapter.go tui/internal/opencode/adapter_test.go tui/internal/opencode/capabilities.go tui/internal/opencode/capabilities_test.go
git commit -m "feat: implement OpenCode runtime adapter"
```

---

### Task 2A.5: Dependency Boundary Gate

**Files:**
- Create: `tui/tests/architecture/vendor_boundary_test.go`
- Create: `scripts/check-runtime-boundary.sh`
- Create: `tests/runtime-boundary/runtime_boundary_test.sh`

**Interfaces:**
- Consumes: Go package import graph
- Produces: a Required Gate that rejects Vendor package imports from upper layers

- [ ] **Step 1: Write a failing boundary fixture test**

The shell test creates a temporary Go package representing `internal/application/leak` that imports `codea/tui/internal/opencode`, runs the checker against the fixture and requires a non-zero exit containing the importer and forbidden import path.

- [ ] **Step 2: Run the boundary test and confirm RED**

Run: `tests/runtime-boundary/runtime_boundary_test.sh`

Expected: FAIL because `check-runtime-boundary.sh` does not exist.

- [ ] **Step 3: Implement the import-graph checker**

Use `go list -deps -json ./...` or Go's parser. Do not rely only on `rg`. Reject OpenCode imports from application, TUI model/component, harness, agent and policy packages. Allow OpenCode's own packages, their tests and explicit composition roots under `cmd/`.

- [ ] **Step 4: Run positive and negative boundary tests**

Run:

```bash
tests/runtime-boundary/runtime_boundary_test.sh
./scripts/check-runtime-boundary.sh
```

Expected: PASS; the injected leak is rejected and the repository has zero Vendor DTO leakage.

- [ ] **Step 5: Commit the boundary gate**

```bash
git add tui/tests/architecture/vendor_boundary_test.go scripts/check-runtime-boundary.sh tests/runtime-boundary/runtime_boundary_test.sh
git commit -m "test: enforce runtime vendor boundary"
```

---

### Task 2A.6: Contract, Parity Smoke and Task Closure

**Files:**
- Create: `tui/tests/contract/runtime_adapter_test.go`
- Create: `docs/task-reports/task-02A.md`
- Modify: `docs/execution-state.yaml`
- Modify: `docs/codea-v1-handoff.md`

**Interfaces:**
- Consumes: complete OpenCodeAdapter and Task 1/2 evidence
- Produces: Task 2A verification evidence and `awaiting_acceptance` state

- [ ] **Step 1: Write the end-to-end contract test against a Spec-correct fake Runtime**

Exercise Health → CreateSession → Prompt → global SSE → once/reject Approval → Cancel → ListAgents using only `runtime.AgentRuntime` references in the test body.

- [ ] **Step 2: Run the contract test and confirm RED where wiring is incomplete**

Run: `cd tui && go test ./tests/contract -run TestAgentRuntimeContract -count=1`

Expected before final wiring: FAIL at the first missing adapter behavior. Complete only that wiring and repeat until GREEN.

- [ ] **Step 3: Run the full deterministic gate**

```bash
cd tui
GOTOOLCHAIN=local go test ./... -count=1
GOTOOLCHAIN=local go test -race ./... -count=1
GOTOOLCHAIN=local go vet ./...
GOTOOLCHAIN=local go build ./...
GOOS=windows GOARCH=amd64 GOTOOLCHAIN=local go build ./cmd/codea ./cmd/parity-runner
TASK2A_DTO="$(mktemp)"
trap 'rm -f "$TASK2A_DTO"' EXIT
go run ./cmd/openapi-gen ../runtime/openapi/opencode-1.18.11.json "$TASK2A_DTO"
cmp "$TASK2A_DTO" internal/opencode/dto.go
cd ..
./scripts/check-runtime-boundary.sh
./scripts/check-execution-state.sh
tests/execution-state/state_validator_test.sh
```

Expected: every command exits 0 under Go 1.26.5.

- [ ] **Step 4: Run real OpenCode parity smoke**

Using the locked v1.18.11 asset and internal/private model configuration, exercise Health, Session, Prompt, SSE Reasoning/Answer, Approval once, Approval reject, Abort and Agent list through `OpenCodeAdapter`. Save raw request/event evidence under the Task 2A report or its artifact directory. Do not claim PASS when the Runtime, model or platform is unavailable; set `blocked` with recovery advice.

- [ ] **Step 5: Evaluate Offline and Windows impact**

- Confirm Task 2A added no Runtime download/install/network path.
- If Supervisor and packaging are unchanged, cite Task 1 offline evidence instead of rerunning unrelated S1 capture.
- Record Windows x64 cross-build output.
- Run Windows Runtime smoke when a Windows Runner is available; otherwise explicitly defer the platform E2E to the existing release Gate without fabricating a result.

- [ ] **Step 6: Write the Task 2A report and enter human gate**

Record files, commits, RED/GREEN evidence, full commands, real Runtime evidence, deviations and remaining risks in `docs/task-reports/task-02A.md`. Set Task 2A to `awaiting_acceptance` only when Verification and Task Gate are both `pass`. Keep Task 3 `pending`.

- [ ] **Step 7: Commit and stop**

```bash
git add docs/task-reports/task-02A.md docs/execution-state.yaml docs/codea-v1-handoff.md
git commit -m "docs: record Task 2A verification"
```

Stop for human acceptance. Do not start Task 3.
