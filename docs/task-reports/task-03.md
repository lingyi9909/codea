# Task 03 Report — Capability Inventory + Parity Harness (Rebaselined)

**Task:** 3

**Status:** awaiting_acceptance

**Date:** 2026-08-11

**Checkpoint:** `fdc3f17efa1dbbe1bc9127d47c6ec73dde145e0d`

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

定义 Scenario、ScenarioResult、Failure、Result 类型。`V1RequiredScenarios()` 返回 12 个 Required 场景：Health、CreateSession、Prompt、Streaming、Answer、Reasoning、ToolLifecycle、Approval、Reject、Cancel、AgentSelection、RawEventHandling。

**文件:** `tui/internal/parity/scenario.go`, `result.go`, `scenario_test.go`

### Step 5: Parity Runner — PASS

Runner 通过 AgentRuntime + Codea Event 执行 Scenario，对比 Baseline/Candidate 结果。支持 Health/CreateSession/Prompt/Cancel 四种场景类型。SilentLoss（Candidate 事件数少于 Baseline）对 Required Scenario 直接 FAIL。`RunAll()` 批量执行并聚合结果。

**文件:** `tui/internal/parity/runner.go`, `runner_test.go`

### Step 6: Required Parity Tests + Gate — PASS

集成测试：全 V1 Required 场景端到端、Capability Compare vs 真实 `capabilities.yaml`、SilentLoss 失败检测。

**文件:** `tui/tests/parity/parity_runner_test.go`

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
| `tui/internal/parity/runner.go` | 新增 | Step 5 |
| `tui/internal/parity/runner_test.go` | 新增 | Step 5 |
| `tui/tests/parity/parity_runner_test.go` | 新增 | Step 6 |
| `docs/execution-state.yaml` | 修改 | — |

## 测试统计

| 包 | 测试数 |
|---|--------|
| `internal/capability` | 14 |
| `internal/parity` | 15 |
| `tests/parity` | 4 |
| `tests/fixtures/fake-runtime` | 11 |
| **总计** | **44** |

## 验证结果

| 命令 | 结果 |
|------|------|
| `go test ./internal/capability/... ./internal/parity/... ./tests/parity/... ./tests/fixtures/fake-runtime/... -count=1` | PASS（44 测试） |
| `go test -race ./... -count=1` | PASS（全部包） |
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

OpenCodeAdapter `RuntimeCapabilities` 对比结果：15/15 Required 全部支持，零缺失。

## Parity 场景清单

| 场景 | Required | Baseline | Candidate |
|------|----------|----------|-----------|
| Health | yes | FakeRuntime | FakeRuntime |
| CreateSession | yes | FakeRuntime | FakeRuntime |
| Prompt | yes | FakeRuntime | FakeRuntime |
| Streaming | yes | FakeRuntime | FakeRuntime |
| Answer | yes | FakeRuntime | FakeRuntime |
| Reasoning | yes | FakeRuntime | FakeRuntime |
| ToolLifecycle | yes | FakeRuntime | FakeRuntime |
| Approval | yes | FakeRuntime | FakeRuntime |
| Reject | yes | FakeRuntime | FakeRuntime |
| Cancel | yes | FakeRuntime | FakeRuntime |
| AgentSelection | yes | FakeRuntime | FakeRuntime |
| RawEventHandling | yes | FakeRuntime | FakeRuntime |

全部 12 个 Required 场景通过。SilentLoss 检测已验证：Candidate 事件数少于 Baseline 时 Required Scenario FAIL。

## 与计划偏差

无。严格按照 Rebaseline 后的 Task 3 边界实施：
- 不实现 SSE reconnect/recovery/history compensation/backpressure（属 Task 4）
- 不引入 OpenCode DTO 到 capability/parity 包
- Fake Runtime 输出 Codea Domain Event，不复制 OpenCode 协议
- 不执行 `t.Skip`，不空实现，不伪造结果

## Vendor Boundary Gate

`internal/capability` 和 `internal/parity` 包零 OpenCode DTO 导入。`check-runtime-boundary.sh` 确认通过。

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
