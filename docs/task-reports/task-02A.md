# Task 02A Report — Codea Runtime Abstraction Rebaseline

**Task:** 2A

**Status:** completed (human accepted 2026-08-11)

**Date:** 2026-08-09

**Checkpoint:** `871110ae346b27d8629769d5a3bb9f14a7373f73`

## 完成内容

### Step 1: Codea Runtime Domain 与 AgentRuntime — PASS

按 Rebaseline 设计定义了完整的 Codea Runtime Contract，所有类型与接口均无 OpenCode DTO 依赖。

**文件:** `tui/internal/runtime/client.go`, `models.go`, `events.go`, `approval.go`, `capabilities.go`, `client_test.go`

### Step 2: OpenCode Request and Approval Mapping — PASS

Domain → Vendor DTO 映射层，含四种 PromptPart、三种 FilePartSource 和 ApprovalReply。

**文件:** `tui/internal/opencode/request_mapper.go`, `request_mapper_test.go`, `approval_mapper.go`, `approval_mapper_test.go`

### Step 3: SSE Transport and Event Mapper — PASS

SSE 协议解析 + 76 条 Golden SSE 事件全映射。EventMapper 完整语义映射：

- **Type mapping**: vendor type → Codea semantic type（`answer.delta`, `reasoning.delta`, `step.started`, `approval.requested`, `raw` 等）
- **Domain data extraction**: `ApprovalRequest{ID, Permission}`, `ToolEvent{Name, CallID}`, `RuntimeError{Code, Message}`
- **Structured error**: 支持 string 和 `{name, data: {message}}` 两种错误格式
- **Truncated stream**: SSE readLoop EOF 后残留 dataLines 不再静默丢弃，发送 truncated `runtime_error`
- **Channel safety**: 所有 `ch <- event` 使用 `select { case ch <-: case <-ctx.Done(): return }`

**文件:** `tui/internal/opencode/sse_client.go`, `sse_client_test.go`, `event_mapper.go`, `event_mapper_test.go`

### Step 4: OpenCodeAdapter and Runtime Capabilities — PASS

组合 HTTPClient、SSEClient、Mapper，完整实现 `AgentRuntime` 接口。编译期接口断言。

**文件:** `tui/internal/opencode/adapter.go`, `adapter_test.go`, `capabilities.go`, `capabilities_test.go`

### Step 5: Dependency Boundary Gate — PASS

`go list` 驱动的 import graph 检查。

**文件:** `tui/tests/architecture/vendor_boundary_test.go`, `scripts/check-runtime-boundary.sh`, `tests/runtime-boundary/runtime_boundary_test.sh`

### Step 6: Contract Test and Task Closure — PASS

端到端契约测试 + 真实 OpenCode parity smoke test。

**文件:** `tui/tests/contract/runtime_adapter_test.go`, `tui/tests/contract/real_opencode_smoke_test.go`

## Review 修复历史

### 第一轮（4 Blocking + 2 Bonus）

| # | 问题 | 修复 |
|---|------|------|
| Block 1 | EventMapper 语义映射缺失 | 重写，新增 15+ Codea 语义常量、vendorToCodea 映射表、mapVendorType() 分类 |
| Block 2 | Gate 6 deferred | 下载 OpenCode v1.18.11 运行 parity smoke |
| Block 3 | Checkpoint SHA 不一致 | 重建 checkpoint |
| Block 4 | SSE goroutine 泄漏 | select/ch ← ctx.Done() |
| Bonus 1 | 缺截断流测试 | 新增 TestSSEClientTruncatedStream |
| Bonus 2 | Non-200 error body 1KB | 改为 64KB |

### 第二轮（3 个实质问题）

| # | 问题 | 修复 |
|---|------|------|
| Issue 1 | Gate 6 smoke 只计数不验证语义事件 | 重写为完整语义断言：answer.delta, reasoning.delta, approval.requested, tool.called；检测模型未配置时 SKIP（非 PASS），含可操作提示 |
| Issue 2 | EventMapper 只做 Type 映射，未提取领域数据 | extractApproval（ID/Permission + v2 action 兼容）、extractTool（Name/CallID + fallback to part.id）、extractError（string + 结构化 format） |
| Issue 3 | Truncated stream 残留 dataLines 静默丢失 | readLoop EOF 后检测 len(dataLines) > 0，发送 truncated runtime_error；测试断言收到此事件 |

### 第三轮（3 个必须修复）

| # | 问题 | 修复 |
|---|------|------|
| Issue 1 | Gate 6 只检测事件类型，未验证 Approval once/reject 端到端流程 | 重写 phase-based 状态机：once 后 tool 执行 + reject 后 tool 阻止；两个 Prompt 串联驱动完整审批闭环 |
| Issue 2 | `quick_llm_test.go` 硬编码诊断测试，默认 `go test ./...` 必然 FAIL | 删除文件 |
| Issue 3 | `prompt_async` 请求 DTO JSON 字段 `messageID` 与 OpenCode v1.18.11 实际协议不符 | 见下方 Protocol Deviation 记录；生成器新增 `fieldJSONOverrides` + 回归测试 |

## OpenCode v1.18.11 Known Protocol Deviations

### Deviation 1: prompt_async 请求使用 `id` 而非 `messageID`

- **发现日期:** 2026-08-09
- **现象:** 使用 `messageID` 发送 prompt 后，OpenCode 返回 `session.error`：「No user message found in stream」；curl 验证 `id` 正常、`messageID` 失败
- **根因:** OpenAPI spec `prompt_async.requestBody.messageID` 与实际 Runtime 行为不一致 — Runtime 只处理 JSON 字段 `id`
- **修复:** `cmd/openapi-gen/main.go` 新增 `fieldJSONOverrides` 映射，将 `OpenCodeSessionPromptAsyncRequest.messageID` JSON tag 覆盖为 `id`
- **回归测试:** `TestPromptAsyncJSONUsesIDNotMessageID`（`go/parser` + `ast.Inspect` 验证 struct tag）
- **影响范围:** 仅 `prompt_async` 请求体；所有响应中的 `MessageID` 字段不受影响

## 完整文件变更

| 文件 | 状态 | 步骤 |
|------|------|------|
| `tui/internal/runtime/client.go` | 新增 | Step 1 |
| `tui/internal/runtime/models.go` | 新增 | Step 1 |
| `tui/internal/runtime/events.go` | 新增 | Step 1 |
| `tui/internal/runtime/approval.go` | 新增 | Step 1 |
| `tui/internal/runtime/capabilities.go` | 新增 | Step 1 |
| `tui/internal/runtime/client_test.go` | 新增 | Step 1 |
| `tui/internal/opencode/request_mapper.go` | 新增 | Step 2 |
| `tui/internal/opencode/request_mapper_test.go` | 新增 | Step 2 |
| `tui/internal/opencode/approval_mapper.go` | 新增 | Step 2 |
| `tui/internal/opencode/approval_mapper_test.go` | 新增 | Step 2 |
| `tui/internal/opencode/sse_client.go` | 新增（Review 修复） | Step 3 |
| `tui/internal/opencode/sse_client_test.go` | 新增（Review 修复） | Step 3 |
| `tui/internal/opencode/event_mapper.go` | 新增（Review 重写×2） | Step 3 |
| `tui/internal/opencode/event_mapper_test.go` | 新增（Review 重写×2） | Step 3 |
| `tui/internal/opencode/adapter.go` | 新增（Review 修复） | Step 4 |
| `tui/internal/opencode/adapter_test.go` | 新增 | Step 4 |
| `tui/internal/opencode/capabilities.go` | 新增 | Step 4 |
| `tui/internal/opencode/capabilities_test.go` | 新增 | Step 4 |
| `tui/tests/architecture/vendor_boundary_test.go` | 新增 | Step 5 |
| `scripts/check-runtime-boundary.sh` | 新增 | Step 5 |
| `tests/runtime-boundary/runtime_boundary_test.sh` | 新增 | Step 5 |
| `tui/tests/contract/runtime_adapter_test.go` | 新增 | Step 6 |
| `tui/tests/contract/real_opencode_smoke_test.go` | 新增（Review 重写×3） | Step 6 |
| `tui/cmd/openapi-gen/main.go` | 修改 — fieldJSONOverrides | Step 3/Review |
| `tui/cmd/openapi-gen/main_test.go` | 修改 — 新增回归测试 | Step 3/Review |
| `tui/internal/opencode/dto.go` | 重新生成 — messageID→id | Step 3/Review |
| `docs/execution-state.yaml` | 修改 — checkpoint + Gate 6 | — |
| `docs/task-reports/task-02A.md` | 修改 — 报告更新 | — |

**1 文件删除：** `tui/tests/contract/quick_llm_test.go`（Review Issue 2）。

## 验证结果

| 命令 | 结果 |
|------|------|
| `go test ./... -count=1` | PASS（所有包，55 个测试） |
| `go test -race ./... -count=1` | PASS |
| `go vet ./...` | PASS |
| `go build ./...` | PASS |
| `GOOS=windows GOARCH=amd64 go build ./cmd/codea ./cmd/parity-runner` | PASS |
| `go run ./cmd/openapi-gen ... \| cmp ... dto.go` | PASS（生成一致） |
| `./scripts/check-runtime-boundary.sh` | PASS（零泄漏） |
| `./scripts/check-execution-state.sh` | PASS |
| `tests/execution-state/state_validator_test.sh` | PASS |
| Real OpenCode v1.18.11 parity smoke (Gate 6) | PASS（含 Approval once/reject 端到端） |

## Task Gate 逐项

| # | 门禁项 | 状态 |
|---|--------|------|
| 1 | Task 2 无退化 | PASS |
| 2 | Contract 完整 — 编译期 OpenCodeAdapter 实现 AgentRuntime | PASS |
| 3 | DTO 零泄漏 — 边界门禁通过 | PASS |
| 4 | Event 零静默丢失 — Golden SSE 76 条全映射 + truncated EOF | PASS |
| 5 | Approval parity — once/always/reject + message，无 remember | PASS |
| 6 | Runtime parity smoke — 真实 OpenCode v1.18.11 验证通过（含 Approval once/reject 端到端流程） | PASS |
| 7 | Offline 无新增风险 | PASS |
| 8 | Windows 无新增风险 | PASS |
| 9 | 人工验收 | **PASS** |

## Gate 6 说明

`TestRealOpenCodeParitySmoke` 在真实 OpenCode v1.18.11 上运行，验证：

- Health / CreateSession / Prompt / Subscribe / ReplyApproval / Cancel / ListAgents / Capabilities — 全部 8 个方法
- 语义事件类型检测：`runtime.connected`, `answer.delta`, `reasoning.delta`, `step.started`, `step.finished`, `tool.called`, `approval.requested`, `session.error`
- **Approval 端到端流程（phase-based 状态机）：**
  - Scenario A: 第一个 Prompt 触发 `approval.requested` → `ReplyApproval(once)` → 工具放行执行 → `toolAfterOnce=true`
  - Scenario B: 第二个 Prompt 触发 `approval.requested` → `ReplyApproval(reject)` → step 终止 → `approvalRejectDone=true`
- 领域数据提取：`Approval.ID`, `Approval.Permission`, `Tool.Name`, `Tool.CallID`, `Error.Code`, `Error.Message`
- 模型未配置时自动检测 `session.error` 并 SKIP（含可操作提示），不被 `go test ./...` 的 PASS 掩盖

## 未解决问题

- Gate 9 (人工验收) pending
- 无其他阻塞项

## Gate 结论

- **Verification:** `pass`（11 项自动验证全部通过）
- **Task Gate:** `pass`（9/9 通过）
- **Human acceptance:** `true` (2026-08-11)
- **Task 2A:** `completed`
- **Task 3:** `pending`
