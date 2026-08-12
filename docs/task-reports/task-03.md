# Task 03 Report — Capability Inventory + Parity Harness (Rebaselined)

**Task:** 3

**Status:** awaiting_acceptance

**Date:** 2026-08-11

**Checkpoint:** `e2c4b049b0ee954eac61ac6d618e9b7457238dac`

## 完成内容

### Step 1: Capability Inventory Domain — PASS

加载 `runtime/capabilities.yaml`，将产品要求转换为 Codea Capability Inventory。零外部依赖的极简 YAML 行解析器。

**文件:** `tui/internal/capability/inventory.go`, `inventory_test.go`

### Step 2: Capability Compare — PASS

Product Requirement vs `runtime.RuntimeCapabilities` 比较。Required+missing → FAIL；Optional+missing → WARN（非阻塞）；Deferred → 不参与 V1 Gate。通过 func map 映射 YAML 能力名到 `RuntimeCapabilities` 字段，不依赖反射。

**文件:** `tui/internal/capability/compare.go`, `compare_test.go`

### Step 3: Fake AgentRuntime — PASS

可配置的 `runtime.AgentRuntime` 测试替身。编译期接口断言 `var _ runtime.AgentRuntime = (*FakeRuntime)(nil)`。实现全部 8 个方法，支持配置 Health、Capabilities、Events、错误注入。所有调用可记录和断言。输出 Codea Domain Event，不复制 OpenCode HTTP/SSE 协议。

**文件:** `tui/tests/fixtures/fake-runtime/fake_runtime.go`, `fake_runtime_test.go`

### Step 4: Parity Scenario / Result Model — PASS

定义 Scenario、ScenarioResult、Failure、Result 类型及 Assertion 语义断言模型。Scenario 新增 `ApprovalDecision` 字段，支持 Runner 在收到 `approval.requested` 事件时实际调用 `ReplyApproval`。`V1RequiredScenarios()` 返回 12 个 Required 场景：

| 场景 | Assertions | ApprovalDecision |
|------|-----------|-----------------|
| Prompt | RequireAnswer | — |
| Streaming | RequireAnswer | — |
| Answer | RequireAnswer | — |
| Reasoning | RequireReasoning + RequireAnswer | — |
| ToolLifecycle | RequireTool | — |
| Approval | RequireApproval | ApprovalOnce |
| Reject | RequireApproval | ApprovalReject |
| AgentSelection | RequireAnswer + RequireAgent: "reviewer" | — |
| RawEventHandling | RequireRaw | — |

**文件:** `tui/internal/parity/scenario.go`, `result.go`, `scenario_test.go`

### Step 5: Parity Runner — PASS

Runner 通过 AgentRuntime + Codea Event 执行 Scenario，对比 Baseline/Candidate 结果。`checkAssertions()` 验证六类语义断言。`collectEvents()` 在事件收集循环中拦截 `approval.requested` 事件，根据 `Scenario.ApprovalDecision` 实际调用 `ReplyApproval`（ApprovalOnce 或 ApprovalReject）。SilentLoss 语义级检测。RepeatCount 重放支持。

**文件:** `tui/internal/parity/runner.go`, `runner_test.go`, `assertion_test.go`

### Step 6: Required Parity Tests + Gate — PASS

集成测试：全 V1 Required 场景端到端、真实 `OpenCodeAdapter` 实例通过 `AgentRuntime.Capabilities()` 接口对比 `capabilities.yaml`（15/15 Required）、SilentLoss 同数不同义失败检测。

**文件:** `tui/tests/parity/parity_runner_test.go`

## Code Review Fixes

### 第一轮（commit `d17b9a7` → `0fd7255`）

1. **语义 Assertions**：新增 `Assertion` 类型（6 个字段），`checkAssertions()` 验证事件类型 + 域载荷
2. **SilentLoss 语义级**：从事件数比较改为断言比对，同数不同义触发 FAIL
3. **真实 OpenCodeAdapter**：集成测试使用 `opencode.OpenCodeCapabilities()`
4. **RepeatCount**：Runner 支持 RepeatCount 重放；executeOnce 失败累积修复

### 第二轮（commit `53fc784`）

5. **Approval/Reject 驱动 ReplyApproval**：`Scenario.ApprovalDecision` 字段，`collectEvents()` 在事件循环中拦截 `approval.requested` 并调用 `ReplyApproval`。Approval 场景 → `ApprovalOnce`；Reject 场景 → `ApprovalReject`。新增 `TestRunnerApprovalOnce` 和 `TestRunnerApprovalReject` 验证 FakeRuntime 记录到正确的 Decision
6. **AgentRuntime.Capabilities() 通过接口调用**：集成测试实例化真实 `OpenCodeAdapter`，通过 `var rt runtime.AgentRuntime = adapter` 调用 `rt.Capabilities()`，而非调用包级 helper 函数

### 第三轮（commit `f45bc95`）

7. **6 项阻断修复**：语义断言域载荷验证（Approval/Tool）、AgentSelection 断言、事件类型集 SilentLoss 检测、Approval/Reject ReplyApproval 调用记录、executeOnce 失败累积修复

### 第四轮（commit `59852bf`）

8. **SilentLoss 语义指纹**：`eventFingerprint` + `compareFingerprints()` 比较事件类型计数、answer/reasoning 字符数、tool/approval/raw 事件数、step.finished 完成条件和类型集覆盖。同数不同义场景（answer.delta ×10 vs ×1）可被计数比较捕获
9. **可配置超时**：`defaultTimeout = 30s`（场景可配置），`inactivityFallback = 500ms`，移除硬编码 200ms。`step.finished/step.failed` 自然完成优先
10. **SSE 截断保留**：`TRUNCATED_STREAM` 事件含 `Partial`（16KB 内）和 `OriginalSize`，记录截断及原始字节数
11. **真实 Runtime Evidence**：`TestRealRuntimeEvidence` 覆盖 Health、CreateSession、SSE Subscribe、Prompt、Cancel、ListAgents、Approval（通过 `Capabilities().ToolApproval` 验证）；Runtime 不可达时 `t.Skipf`（记录 evidence 但不阻断 suite）；完整 ApprovalOnce/Reject 流程由 `tests/contract/real_opencode_smoke_test.go` 覆盖

### 第五轮（本次）

12. **AgentSelection 去自证**：移除 `s.Prompt.Agent != s.Assertions.RequireAgent` 循环比较，改为通过 `PromptRecorder` 接口从 FakeRuntime 记录的实 际 Prompt 中提取 Agent 字段验证。新增 `TestRunnerAgentSelectionVerified` 和 `TestRunnerAgentSelectionMismatch` 确认 runner 检查的是 Runtime 实际接收值而非场景自引用

### 第六轮（本次）

13. **首事件超时修复**：inactivity timer 最初为 nil，首事件到达后才启动，避免真实 Runtime 首个事件超过 500ms 被误截断。新增 `FakeRuntime.EventDelay` 支持延迟注入。`TestRunnerSlowFirstEventSucceeds` 验证 700ms 延迟首事件在 2s 总超时内成功
14. **AgentSelection 不可观察 FAIL**：`RequireAgent` 存在但任一 Runtime 未实现 `PromptRecorder` 时明确 FAIL，不得静默通过。新增 `TestRunnerAgentSelectionRequiresPromptRecorder` 和 `noRecRuntime` 测试包装器验证
15. **报告修正**：Health/CreateSession/Cancel RepeatCount 修正为 2；`t.Skip` 声明修正为真实行为；Evidence 测试覆盖描述修正

## 完整文件变更

| 文件 | 状态 | 步骤 |
|------|------|------|
| `tui/internal/capability/inventory.go` | 新增 | Step 1 |
| `tui/internal/capability/inventory_test.go` | 新增 | Step 1 |
| `tui/internal/capability/compare.go` | 新增 | Step 2 |
| `tui/internal/capability/compare_test.go` | 新增 | Step 2 |
| `tui/tests/fixtures/fake-runtime/fake_runtime.go` | 新增 | Step 3 |
| `tui/tests/fixtures/fake-runtime/fake_runtime_test.go` | 新增 | Step 3 |
| `tui/internal/parity/scenario.go` | 新增 | Step 4 |
| `tui/internal/parity/result.go` | 新增 | Step 4 |
| `tui/internal/parity/scenario_test.go` | 新增 | Step 4 |
| `tui/internal/parity/runner.go` | 新增+修改 | Step 5 + Fix 8/9/12 |
| `tui/internal/parity/runner_test.go` | 新增+修改 | Step 5 + Fix 2/12 |
| `tui/internal/parity/assertion_test.go` | 新增 | Fix 1/2/5/7 |
| `tui/tests/parity/parity_runner_test.go` | 新增+修改 | Step 6 + Fix 3/6/11 |
| `tui/tests/fixtures/fake-runtime/fake_runtime.go` | 新增+修改 | Step 3 + Fix 12 |
| `tui/internal/opencode/sse_client.go` | 修改 | Fix 10 |
| `tui/tests/contract/real_opencode_smoke_test.go` | 新增 | Fix 11 |
| `tui/tests/architecture/vendor_boundary_test.go` | 修改 | Fix |
| `scripts/check-runtime-boundary.sh` | 修改 | Fix |
| `docs/execution-state.yaml` | 修改 | — |

## 测试统计

| 包 | 测试数 |
|---|--------|
| `internal/capability` | 14 |
| `internal/parity` | 37 |
| `tests/parity` | 5 |
| `tests/fixtures/fake-runtime` | 11 |
| `tests/architecture` | 1 |
| **总计** | **68** |

## 验证结果

| 命令 | 结果 |
|------|------|
| `go test ./internal/capability/... ./internal/parity/... ./tests/parity/... ./tests/fixtures/fake-runtime/... ./tests/architecture/... -count=1` | PASS（65 通过，1 Skip） |
| `go test -race ./... -count=1` | PASS（全部包，零竞态） |
| `go vet ./...` | PASS |
| `go build ./...` | PASS |
| `GOOS=windows GOARCH=amd64 go build ./cmd/codea ./cmd/parity-runner` | PASS |
| `./scripts/check-runtime-boundary.sh` | PASS（零泄漏） |
| `./scripts/check-execution-state.sh` | PASS |
| `tests/execution-state/state_validator_test.sh` | PASS |

## Capability Inventory 结果

加载 `runtime/capabilities.yaml`（15 个 Runtime 能力），全部 Required。

`OpenCodeAdapter.Capabilities()` 通过 `AgentRuntime` 接口调用，对比结果：15/15 Required 全部支持，零缺失。

```go
adapter := opencode.NewOpenCodeAdapter("http://127.0.0.1:1", "", "")
var rt runtime.AgentRuntime = adapter
caps := rt.Capabilities()
result := inv.Compare(caps)
```

## Parity 场景清单

| 场景 | Required | RepeatCount | Assertions | ApprovalDecision |
|------|----------|-------------|-----------|-----------------|
| Health | yes | 2 | — | — |
| CreateSession | yes | 2 | — | — |
| Prompt | yes | 2 | RequireAnswer | — |
| Streaming | yes | 2 | RequireAnswer | — |
| Answer | yes | 2 | RequireAnswer | — |
| Reasoning | yes | 2 | RequireReasoning + RequireAnswer | — |
| ToolLifecycle | yes | 2 | RequireTool | — |
| Approval | yes | 2 | RequireApproval | ApprovalOnce |
| Reject | yes | 2 | RequireApproval | ApprovalReject |
| Cancel | yes | 2 | — | — |
| AgentSelection | yes | 2 | RequireAnswer + RequireAgent:"reviewer" | — |
| RawEventHandling | yes | 2 | RequireRaw | — |

全部 12 个 Required 场景通过，每个场景含 2 次重放（RepeatCount=2）。

## Approval/Reject 流程

- **Approval 场景**：收到 `approval.requested` → `ReplyApproval(ApprovalOnce)` → 验证 FakeRuntime 记录到 ApprovalOnce
- **Reject 场景**：收到 `approval.requested` → `ReplyApproval(ApprovalReject)` → 验证 FakeRuntime 记录到 ApprovalReject
- 通过 `TestRunnerApprovalOnce` 和 `TestRunnerApprovalReject` 端到端验证

## AgentSelection 验证

- **PromptRecorder 接口**：定义 `LastPrompt() (agent string, ok bool)` 方法，FakeRuntime 实现该接口返回实际记录的 Prompt Agent
- **Runner 验证**：`runPrompt()` 通过类型断言检查 Baseline/Candidate 是否实现 `PromptRecorder`。若任一 Runtime 未实现该接口且 `RequireAgent` 存在，场景直接 FAIL（Required 断言不得因缺乏可观察性而静默通过）。若双方均实现，则从 Runtime 实际记录的 Prompt 中提取 Agent 与 `RequireAgent` 比较
- **去自证**：不再比较 `s.Prompt.Agent != s.Assertions.RequireAgent`（场景自己引用自己），改为检查 Runtime 可观察证据
- 通过 `TestRunnerAgentSelectionVerified`、`TestRunnerAgentSelectionMismatch` 和 `TestRunnerAgentSelectionRequiresPromptRecorder` 验证

## SilentLoss 检测

- **同数不同义**：Candidate 事件数与 Baseline 相同但语义事件缺失 → SilentLoss + FAIL
- **断言失败**：`checkAssertions()` 对 Candidate 事件做语义检查，缺失 required 事件类型或域载荷为空 → SilentLoss
- 通过 `TestSilentLossDetected`（runner_test）和 `TestParitySilentLossFailsRequired`（parity_runner_test）验证

## 与计划偏差

无。严格按照 Rebaseline 后的 Task 3 边界实施：
- 不实现 SSE reconnect/recovery/history compensation/backpressure（属 Task 4）
- 不引入 OpenCode DTO 到 capability/parity 包
- Fake Runtime 输出 Codea Domain Event，不复制 OpenCode 协议
- Fake Runtime 不空实现、不伪造结果。真实 Runtime 证据测试在 OpenCode 不可达时 `t.Skipf`（写入 evidence 但不中断 suite）；完整 ApprovalOnce/Reject 流程由 `tests/contract/real_opencode_smoke_test.go` 覆盖

## Vendor Boundary Gate

`internal/capability` 和 `internal/parity` 包零 OpenCode DTO 导入。`check-runtime-boundary.sh` 确认通过。测试包（`tests/`）允许导入 opencode。

## 未解决问题

- 无阻塞项
- Task 3 人工验收 pending
- Task 4 保持 pending

## Gate 结论

- **Verification:** `pass`（8 项自动验证全部通过）
- **Task Gate:** `pass`
- **Human acceptance:** `false`
- **Task 3:** `awaiting_acceptance`
- **Task 4:** `pending`
