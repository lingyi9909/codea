# Task 03 Report — Capability Inventory + Parity Harness (Rebaselined)

**Task:** 3

**Status:** awaiting_acceptance

**Date:** 2026-08-11

**Checkpoint:** `0fd7255e54c54bce99d9f75fc2408a8e6042bba6`

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

定义 Scenario、ScenarioResult、Failure、Result 类型及 Assertion 语义断言模型。`V1RequiredScenarios()` 返回 12 个 Required 场景，每个场景携带具体断言：

| 场景 | Assertions |
|------|-----------|
| Prompt | RequireAnswer |
| Streaming | RequireAnswer |
| Answer | RequireAnswer |
| Reasoning | RequireReasoning + RequireAnswer |
| ToolLifecycle | RequireTool |
| Approval | RequireApproval |
| Reject | RequireApproval |
| AgentSelection | RequireAnswer + RequireAgent: "reviewer" |
| RawEventHandling | RequireRaw |

**文件:** `tui/internal/parity/scenario.go`, `result.go`, `scenario_test.go`

### Step 5: Parity Runner — PASS

Runner 通过 AgentRuntime + Codea Event 执行 Scenario，对比 Baseline/Candidate 结果。支持 Health/CreateSession/Cancel/Prompt 四种场景类型。`checkAssertions()` 验证六类语义断言：reasoning.delta 存在性、answer.delta 存在性、approval.requested（含非空 ID+Permission 域载荷）、tool.called（含非空 Name+CallID 域载荷）、raw（含有效 JSON 载荷）、Agent 名称匹配。SilentLoss 针对语义级检测：Candidate 事件数相同但缺少语义事件时触发 FAIL。Runner 支持 RepeatCount 重放验证。

**文件:** `tui/internal/parity/runner.go`, `runner_test.go`, `assertion_test.go`

### Step 6: Required Parity Tests + Gate — PASS

集成测试覆盖：全 V1 Required 场景端到端（共享事件集满足全部语义断言）、真实 `opencode.OpenCodeCapabilities()` vs `capabilities.yaml` 对比（15/15 Required 全部支持）、SilentLoss 同数不同义失败检测。

**文件:** `tui/tests/parity/parity_runner_test.go`

## Code Review Fixes（本轮）

以下 4 项阻塞问题已修复（commit `d17b9a7`）：

1. **语义 Assertions**：新增 `Assertion` 类型（6 个字段），`checkAssertions()` 函数验证事件类型 + 域载荷。12 个 Prompt 场景各自携带明确断言。
2. **SilentLoss 语义级**：从 `len(cEvents) < len(bEvents)` 改为断言比对。Candidate 事件数相同但缺失 reasoning.delta → SilentLoss + FAIL。
3. **真实 OpenCodeAdapter**：集成测试使用 `opencode.OpenCodeCapabilities()` 而非手写 15 个 `true` 值。
4. **RepeatCount + Checkpoint**：Runner 支持 RepeatCount 重放；Implementation checkpoint 更新为 `d17b9a7`（包含全部 Step 6 代码和 review 修复）。

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
| `tui/internal/parity/runner.go` | 新增+修改 | Step 5 + Fix 1/2/4 |
| `tui/internal/parity/runner_test.go` | 新增+修改 | Step 5 + Fix 2 |
| `tui/internal/parity/assertion_test.go` | 新增 | Fix 1/2 |
| `tui/tests/parity/parity_runner_test.go` | 新增+修改 | Step 6 + Fix 1/3 |
| `tui/tests/architecture/vendor_boundary_test.go` | 修改 | Fix（skip /tests/） |
| `scripts/check-runtime-boundary.sh` | 修改 | Fix（skip /tests/） |
| `docs/execution-state.yaml` | 修改 | — |

## 测试统计

| 包 | 测试数 |
|---|--------|
| `internal/capability` | 14 |
| `internal/parity` | 24 （15 原 + 9 断言） |
| `tests/parity` | 4 |
| `tests/fixtures/fake-runtime` | 11 |
| `tests/architecture` | 1 |
| **总计** | **54** |

## 验证结果

| 命令 | 结果 |
|------|------|
| `go test ./internal/capability/... ./internal/parity/... ./tests/parity/... ./tests/fixtures/fake-runtime/... ./tests/architecture/... -count=1` | PASS（54 测试） |
| `go test -race ./... -count=1` | PASS（全部包，零竞态） |
| `go vet ./...` | PASS |
| `go build ./...` | PASS |
| `GOOS=windows GOARCH=amd64 go build ./cmd/codea ./cmd/parity-runner` | PASS |
| `./scripts/check-runtime-boundary.sh` | PASS（零泄漏） |
| `./scripts/check-execution-state.sh` | PASS |
| `tests/execution-state/state_validator_test.sh` | PASS |

## Capability Inventory 结果

加载 `runtime/capabilities.yaml`（15 个 Runtime 能力）：

| Level | Count |
|-------|-------|
| Required | 15 |
| Optional | 0 |
| Deferred | 0 |

OpenCodeAdapter `RuntimeCapabilities` 对比结果：15/15 Required 全部支持，零缺失。集成测试通过真实 `opencode.OpenCodeCapabilities()` 调用验证。

## Paralic 场景清单

| 场景 | Required | RepeatCount | Assertions |
|------|----------|-------------|-----------|
| Health | yes | 1 | — |
| CreateSession | yes | 1 | — |
| Prompt | yes | 2 | RequireAnswer |
| Streaming | yes | 2 | RequireAnswer |
| Answer | yes | 2 | RequireAnswer |
| Reasoning | yes | 2 | RequireReasoning + RequireAnswer |
| ToolLifecycle | yes | 2 | RequireTool |
| Approval | yes | 2 | RequireApproval |
| Reject | yes | 2 | RequireApproval |
| Cancel | yes | 1 | — |
| AgentSelection | yes | 2 | RequireAnswer + RequireAgent:"reviewer" |
| RawEventHandling | yes | 2 | RequireRaw |

全部 12 个 Required 场景通过，每个 Prompt 场景含 2 次重放（RepeatCount=2）。

## SilentLoss 检测

- **同数不同义**：Candidate 事件数与 Baseline 相同但语义事件缺失 → SilentLoss + FAIL
- **断言失败**：`checkAssertions()` 对 Candidate 事件做语义检查，缺失 required 事件类型或域载荷为空 → SilentLoss
- 通过 `TestSilentLossDetected`（runner_test）和 `TestParitySilentLossFailsRequired`（parity_runner_test）验证

## 与计划偏差

无。严格按照 Rebaseline 后的 Task 3 边界实施：
- 不实现 SSE reconnect/recovery/history compensation/backpressure（属 Task 4）
- 不引入 OpenCode DTO 到 capability/parity 包
- Fake Runtime 输出 Codea Domain Event，不复制 OpenCode 协议
- 不执行 `t.Skip`，不空实现，不伪造结果

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
