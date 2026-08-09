# Task 02A Report — Codea Runtime Domain and Contract

**Task:** 2A

**Status:** in_progress

**Current step:** 1 completed — 进入 Step 2

**Date:** 2026-08-09

**Checkpoint:** `31b692d8bcfc3a8e8b9677f0279b62f5b1c992af`

## 已完成内容

### Step 1: Codea Runtime Domain 与 AgentRuntime — PASS

按 Rebaseline 设计定义了完整的 Codea Runtime Contract，所有类型与接口均无 OpenCode DTO 依赖。

**文件变更:**

- 新增 `tui/internal/runtime/client.go` — `AgentRuntime` 接口（8 个方法）
- 新增 `tui/internal/runtime/models.go` — `SessionID`、`ApprovalID`、`ModelRef`、`HealthInfo`、`Session`、`CreateSessionRequest`、`PromptRequest`、四种 `PromptPart` 变体、`Agent`
- 新增 `tui/internal/runtime/events.go` — `Event`、`EventType`、`ToolEvent`、`ApprovalRequest`、`RuntimeError`、`Sensitivity`
- 新增 `tui/internal/runtime/approval.go` — `ApprovalDecision`（once/always/reject）、`ApprovalReply`
- 新增 `tui/internal/runtime/capabilities.go` — `RuntimeCapabilities`（15 个能力键）
- 新增 `tui/internal/runtime/client_test.go` — 五个契约测试（含 FilePartSource 变体和 Sensitivity 值校验）

**TDD 流程:**

- RED: `go test ./internal/runtime` → 编译失败（类型未定义）
- GREEN: `go test ./internal/runtime -count=1` → PASS

**初审修正（2026-08-09）:**

- `FilePart.Source` 从 `any` 改为 Codea-owned `FilePartSource` sealed interface（三个变体：`FileSource`、`SymbolSource`、`ResourceSource`），从锁定 OpenAPI v1.18.11 Spec 提取
- `Sensitivity` 常量从 `public/internal/private` 修正为 `public/internal/sensitive`
- 新增 `TestFilePartSourceVariantsSatisfyContract` 和 `TestSensitivityValues` 测试

**验证结果:**

| 命令 | 结果 |
|------|------|
| `cd tui && go test ./internal/runtime -count=1` | PASS（5/5） |
| `cd tui && go vet ./internal/runtime/...` | PASS |

## 计划偏差

`FilePart.Source any` → 锁定 Spec 提取的 `FilePartSource` sealed interface；`SensitivityPrivate` → `SensitivitySensitive`。其余严格按 Task 2A.1 计划和 Rebaseline Design §4 实现。

## 未解决问题

- 无阻塞项。
- 下一步：Task 2A Step 2 — Request/Approval Mapper。

## Gate 结论

- **Verification (Step 1):** `pass`
- **Task Gate:** `not_evaluated`（待 Step 1–6 全部完成）
- **Human acceptance:** `false`
- **Task 2A:** `in_progress`
