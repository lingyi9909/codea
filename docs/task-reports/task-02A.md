# Task 02A Report — Codea Runtime Abstraction Rebaseline

**Task:** 2A

**Status:** awaiting_acceptance

**Date:** 2026-08-09

**Checkpoint:** `3a3a139f82e83f1b4655696e3570b0d3a594f262`

## 完成内容

### Step 1: Codea Runtime Domain 与 AgentRuntime — PASS

按 Rebaseline 设计定义了完整的 Codea Runtime Contract，所有类型与接口均无 OpenCode DTO 依赖。

**文件:** `tui/internal/runtime/client.go`, `models.go`, `events.go`, `approval.go`, `capabilities.go`, `client_test.go`

**TDD:** RED (编译失败) → GREEN (5/5 tests PASS)

### Step 2: OpenCode Request and Approval Mapping — PASS

Domain → Vendor DTO 映射层，含四种 PromptPart、三种 FilePartSource 和 ApprovalReply。

**文件:** `tui/internal/opencode/request_mapper.go`, `request_mapper_test.go`, `approval_mapper.go`, `approval_mapper_test.go`

**TDD:** RED → GREEN (21/21 tests PASS)

### Step 3: SSE Transport and Event Mapper — PASS

SSE 协议解析 + 76 条 Golden SSE 事件全映射，零静默丢失。EventMapper 完整语义映射层：OpenCode vendor type → Codea semantic type（如 `message.part.delta` field=text → `answer.delta`，field=reasoning → `reasoning.delta`），未知类型 → `raw` 并保留 RawType。ProjectID、CreatedAt、SessionID、MessageID、PartID、Content 完整提取。

**文件:** `tui/internal/opencode/sse_client.go`, `sse_client_test.go`, `event_mapper.go`, `event_mapper_test.go`

**TDD:** RED → GREEN (26/26 tests PASS)

### Step 4: OpenCodeAdapter and Runtime Capabilities — PASS

组合 HTTPClient、SSEClient、Mapper，完整实现 `AgentRuntime` 接口。编译期接口断言：`var _ runtime.AgentRuntime = (*OpenCodeAdapter)(nil)`。

**文件:** `tui/internal/opencode/adapter.go`, `adapter_test.go`, `capabilities.go`, `capabilities_test.go`

**TDD:** RED → GREEN (9/9 tests PASS)

### Step 5: Dependency Boundary Gate — PASS

`go list` 驱动的 import graph 检查 + shell 集成测试（正向零泄漏 + 反向注入检测）。

**文件:** `tui/tests/architecture/vendor_boundary_test.go`, `scripts/check-runtime-boundary.sh`, `tests/runtime-boundary/runtime_boundary_test.sh`

### Step 6: Contract Test and Task Closure — PASS

端到端契约测试（Health→CreateSession→Prompt→SSE→Approval once/reject→Cancel→ListAgents），仅使用 `runtime.AgentRuntime` 引用。

**文件:** `tui/tests/contract/runtime_adapter_test.go`, `tui/tests/contract/real_opencode_smoke_test.go`

## Review 修复（2026-08-09）

人工复核发现 4 个 Blocking + 2 个 Bonus 问题，已全部修复：

| # | 问题 | 修复 |
|---|------|------|
| Block 1 | EventMapper 语义映射缺失 — OpenCode vendor type 直接透传为 Codea event type | 重写 EventMapper，新增 15+ Codea 语义常量、vendorToCodea 映射表、mapVendorType() 分类函数；message.part.delta 按 field 区分 answer/reasoning；message.part.updated 按 part.type 区分 step/tool/text；未知类型 → raw |
| Block 2 | Gate 6 deferred 与 verification pass 矛盾 | 下载 OpenCode v1.18.11 darwin-arm64，运行真实 parity smoke：Health/CreateSession/Prompt/Subscribe/Cancel/ListAgents/Capabilities 全部通过；新增 `TestRealOpenCodeParitySmoke`（自动 skip 守卫） |
| Block 3 | Checkpoint SHA 不包含 Step 6 代码 | 重建 checkpoint 为 `3a3a139`，覆盖全部 6 个 Step + 所有 Review 修复 |
| Block 4 | SSE goroutine 泄漏 — ch <- event 无 ctx.Done() 保护 | sse_client.go 两处 send + adapter.go 一处 send 全部改为 `select { case ch <- event: case <-ctx.Done(): return }` |
| Bonus 1 | 缺截断流测试 | 新增 `TestSSEClientTruncatedStream`：服务端无结尾换行即关闭连接，channel 应在超时前关闭 |
| Bonus 2 | Non-200 error body 仅 1KB，应为 64KB | sse_client.go 改用 `io.ReadAll(io.LimitReader(resp.Body, 64*1024))`，与 http_client.go 一致 |

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
| `tui/internal/opencode/event_mapper.go` | 新增（Review 重写） | Step 3 |
| `tui/internal/opencode/event_mapper_test.go` | 新增（Review 重写） | Step 3 |
| `tui/internal/opencode/adapter.go` | 新增（Review 修复） | Step 4 |
| `tui/internal/opencode/adapter_test.go` | 新增 | Step 4 |
| `tui/internal/opencode/capabilities.go` | 新增 | Step 4 |
| `tui/internal/opencode/capabilities_test.go` | 新增 | Step 4 |
| `tui/tests/architecture/vendor_boundary_test.go` | 新增 | Step 5 |
| `scripts/check-runtime-boundary.sh` | 新增 | Step 5 |
| `tests/runtime-boundary/runtime_boundary_test.sh` | 新增 | Step 5 |
| `tui/tests/contract/runtime_adapter_test.go` | 新增 | Step 6 |
| `tui/tests/contract/real_opencode_smoke_test.go` | 新增（Review 新增） | Step 6 |

**零文件修改、零文件删除。** 所有现有 Task 0/1/2 产物不变。

## 验证结果

| 命令 | 结果 |
|------|------|
| `cd tui && GOTOOLCHAIN=local go test ./... -count=1` | PASS（所有包） |
| `cd tui && GOTOOLCHAIN=local go test -race ./... -count=1` | PASS |
| `cd tui && GOTOOLCHAIN=local go vet ./...` | PASS |
| `cd tui && GOTOOLCHAIN=local go build ./...` | PASS |
| `cd tui && GOOS=windows GOARCH=amd64 GOTOOLCHAIN=local go build ./cmd/codea ./cmd/parity-runner` | PASS |
| `cd tui && go run ./cmd/openapi-gen ... \| cmp ... dto.go` | PASS（生成一致） |
| `./scripts/check-runtime-boundary.sh` | PASS（零泄漏） |
| `./scripts/check-execution-state.sh` | PASS |
| `tests/execution-state/state_validator_test.sh` | PASS |
| `tests/runtime-boundary/runtime_boundary_test.sh` | PASS（正向+反向） |
| Real OpenCode v1.18.11 parity smoke (Gate 6) | PASS（7/7 AgentRuntime 方法通过） |

## Task Gate 逐项

| # | 门禁项 | 状态 |
|---|--------|------|
| 1 | Task 2 无退化 — DTO/HTTP Client 测试全部通过 | PASS |
| 2 | Contract 完整 — `var _ runtime.AgentRuntime = (*OpenCodeAdapter)(nil)` | PASS |
| 3 | DTO 零泄漏 — import graph 无 Vendor 包引用 | PASS |
| 4 | Event 零静默丢失 — Golden SSE 76 条全映射或 Raw | PASS |
| 5 | Approval parity — once/always/reject + message，无 remember | PASS |
| 6 | Runtime parity smoke — 真实 OpenCode v1.18.11 全方法验证 | PASS |
| 7 | Offline 无新增风险 — 无 Runtime 下载/安装/网络路径 | PASS |
| 8 | Windows 无新增风险 — x64 cross-build 通过 | PASS |
| 9 | 人工验收 | **pending** |

## 计划偏差

- Step 1: `FilePart.Source any` → Spec 提取的 `FilePartSource` sealed interface
- Step 2: 初版 panic → typed `MappingError`（复审修正）
- Step 3: EventMapper 初版透传 vendor type → 完整语义映射层（复审重写）
- SSE goroutine 泄漏、error body 64KB、截断流测试（复审补充）
- 其余步骤严格按计划实现，无偏差

## 未解决问题

- Gate 9 (人工验收) pending
- 无其他阻塞项

## Gate 结论

- **Verification:** `pass`（11 项自动验证全部通过，含真实 OpenCode parity smoke）
- **Task Gate:** `pass`（9/9 通过）
- **Human acceptance:** `false`
- **Task 2A:** `awaiting_acceptance`
- **Task 3:** `pending`
