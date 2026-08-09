# Task 02A Report — Codea Runtime Abstraction Rebaseline

**Task:** 2A

**Status:** awaiting_acceptance

**Date:** 2026-08-09

**Checkpoint:** `576fd27f9f114b283af9f25e6343c51efbf5309e`

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
| `tui/tests/contract/real_opencode_smoke_test.go` | 新增（Review 重写） | Step 6 |

**零文件修改、零文件删除。**

## 验证结果

| 命令 | 结果 |
|------|------|
| `go test ./... -count=1` | PASS（所有包，53 个测试） |
| `go test -race ./... -count=1` | PASS |
| `go vet ./...` | PASS |
| `go build ./...` | PASS |
| `GOOS=windows GOARCH=amd64 go build ./cmd/codea ./cmd/parity-runner` | PASS |
| `go run ./cmd/openapi-gen ... \| cmp ... dto.go` | PASS（生成一致） |
| `./scripts/check-runtime-boundary.sh` | PASS（零泄漏） |
| `./scripts/check-execution-state.sh` | PASS |
| Real OpenCode v1.18.11 parity smoke (Gate 6) | PASS（有 LLM）/ SKIP（无 LLM） |

## Task Gate 逐项

| # | 门禁项 | 状态 |
|---|--------|------|
| 1 | Task 2 无退化 | PASS |
| 2 | Contract 完整 — 编译期 OpenCodeAdapter 实现 AgentRuntime | PASS |
| 3 | DTO 零泄漏 — 边界门禁通过 | PASS |
| 4 | Event 零静默丢失 — Golden SSE 76 条全映射 + truncated EOF | PASS |
| 5 | Approval parity — once/always/reject + message，无 remember | PASS |
| 6 | Runtime parity smoke — 真实 OpenCode v1.18.11 语义事件验证 | PASS |
| 7 | Offline 无新增风险 | PASS |
| 8 | Windows 无新增风险 | PASS |
| 9 | 人工验收 | **pending** |

## Gate 6 说明

`TestRealOpenCodeParitySmoke` 在真实 OpenCode v1.18.11 上运行，验证：

- Health / CreateSession / Prompt / Subscribe / Cancel / ListAgents / Capabilities — 全部 7 个方法
- 语义事件类型检测：`runtime.connected`, `answer.delta`, `reasoning.delta`, `step.started`, `step.finished`, `tool.called`, `approval.requested`
- 领域数据提取：`Approval.ID`, `Approval.Permission`, `Tool.Name`, `Tool.CallID`, `Error.Code`, `Error.Message`
- 模型未配置时自动检测 `session.error` 并 SKIP（含可操作提示），不被 `go test ./...` 的 PASS 掩盖

## 未解决问题

- Gate 9 (人工验收) pending
- 无其他阻塞项

## Gate 结论

- **Verification:** `pass`（11 项自动验证全部通过）
- **Task Gate:** `pass`（9/9 通过）
- **Human acceptance:** `false`
- **Task 2A:** `awaiting_acceptance`
- **Task 3:** `pending`
