# Task 02A Report — Codea Runtime Domain and Contract

**Task:** 2A

**Status:** in_progress

**Current step:** 3 completed — 进入 Step 4

**Date:** 2026-08-09

**Checkpoint:** `05a26d8054999f23314d34803fccdf88c1d2fc8e`

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

### Step 2: OpenCode Request and Approval Mapping — PASS

**文件变更:**

- 新增 `tui/internal/opencode/request_mapper.go` — `MapCreateSessionRequest`、`MapPromptRequest` 及内部 Part/Source 映射
- 新增 `tui/internal/opencode/request_mapper_test.go` — 15 个测试：CreateSession、TextPart（含 JSON discriminator）、FilePart FileSource/SymbolSource/ResourceSource、AgentPart（含 nil Source）、SubtaskPart（含 nil Model）、nil Model、全 Part 组合
- 新增 `tui/internal/opencode/approval_mapper.go` — `MapApprovalReply`
- 新增 `tui/internal/opencode/approval_mapper_test.go` — 5 个测试：once/always/reject/with message/no remember field

**TDD 流程:**

- RED: `go test ./internal/opencode -run 'TestMap(CreateSession|Prompt|Approval)' -count=1` → 编译失败（函数未定义）
- GREEN: 全部 21 个新测试 + 所有 Task 2 现有测试 PASS

**映射覆盖:**

| Codea Domain | OpenCode DTO | Discriminator |
|-------------|-------------|---------------|
| `TextPart` | `OpenCodeTextPartInput` | `"text"` |
| `FilePart` + `FileSource` | `OpenCodeFilePartInput` + `OpenCodeFileSource` | `"file"` |
| `FilePart` + `SymbolSource` | `OpenCodeFilePartInput` + `OpenCodeSymbolSource` | `"file"` |
| `FilePart` + `ResourceSource` | `OpenCodeFilePartInput` + `OpenCodeResourceSource` | `"file"` |
| `AgentPart` | `OpenCodeAgentPartInput` | `"agent"` |
| `SubtaskPart` | `OpenCodeSubtaskPartInput` | `"subtask"` |
| `ApprovalReply` | `OpenCodePermissionReplyRequest` | — |
| `ModelRef` | `OpenCodeSessionPromptAsyncRequestModel` | — |

**验证结果:**

| 命令 | 结果 |
|------|------|
| `cd tui && go test ./internal/opencode -run 'TestMap(CreateSession\|Prompt\|Approval)' -count=1` | PASS（21/21，含 nil/error 路径） |
| `cd tui && go test ./internal/opencode -count=1` | PASS（全 34 tests，含 Task 2 现有测试） |

**复审修正（2026-08-09）:**

- `MapPromptRequest` 签名改为 `(string, OpenCodeSessionPromptAsyncRequest, error)`，`mapPromptPart` 和 `mapFilePartSource` 改为返回 error
- 新增 `MappingError` typed error，支持 `errors.As` 识别
- nil PromptPart、nil FilePartSource 和部分映射提前终止均返回 error，不 panic
- 新增 `TestMapPromptRequestRejectsNilPart`、`TestMapPromptRequestRejectsNilFileSource`、`TestMapPromptRequestRejectsNilPartStopsEarly`

### Step 3: SSE Transport and Event Mapper — PASS

**文件变更:**

- 新增 `tui/internal/opencode/sse_client.go` — `SSEClient`，通过 `GET /global/event` 订阅 SSE 流，Basic Auth，返回 `<-chan SSERawEvent`
- 新增 `tui/internal/opencode/sse_client_test.go` — 8 个测试：单行/多行数据、注释忽略、多事件、非 200、上下文取消、大负载（128KB）、Basic Auth
- 新增 `tui/internal/opencode/event_mapper.go` — `MapEvent`，解析 SSE envelope `{directory, payload: {type, properties}}` → `runtime.Event`
- 新增 `tui/internal/opencode/event_mapper_test.go` — 9 个测试：Golden SSE（76 条事件逐一映射）、事件类型统计、未知类型、畸形 JSON、超大 Raw（20KB 截断）、字段提取（SessionID/MessageID/PartID/Content）、Raw JSON 完整性

**TDD 流程:**

- RED: `go test ./internal/opencode -run 'Test(SSE|EventMapper|Golden)' -count=1` → 编译失败
- GREEN: 全部 17 个新测试 PASS

**关键行为:**

- SSE 协议：多行 `data:` 用 `\n` 合并，`:` 注释忽略，空行分隔事件
- Scanner buffer 128KB，最大 2MB；大负载不受默认 64KB 限制
- 非 200 响应返回 error（含截断 body）
- 上下文取消时 channel 正常关闭，无 goroutine 泄漏
- Golden SSE 76 条事件全部映射：每条均有非空 Type、RawType、Raw
- 未知事件类型：Type = RawType = 原始 OpenCode 类型字符串，Raw 保留
- 畸形 JSON：Type = `_unparseable_`，Raw 保留精确原始字节
- Raw > 16KB：截断至 16KB，`RawTruncated=true`，`RawOriginalSize` 记录原始大小
- 从 properties 提取：sessionID、messageID、partID、delta/text（Content）

**验证结果:**

| 命令 | 结果 |
|------|------|
| `cd tui && go test ./internal/opencode -run 'Test(SSE\|EventMapper\|Golden)' -count=1` | PASS（17/17） |
| `cd tui && go test ./internal/opencode -count=1` | PASS（全 51 tests） |

## 计划偏差

Step 1: `FilePart.Source any` → 锁定 Spec 提取的 `FilePartSource` sealed interface；`SensitivityPrivate` → `SensitivitySensitive`。

Step 2: 严格按计划实现。复审后修正 panic → typed error（`MappingError`），`MapPromptRequest` 返回 `error`。

## 未解决问题

- 无阻塞项。
- 下一步：Task 2A Step 4 — OpenCodeAdapter and Runtime Capabilities。

## Gate 结论

- **Verification (Step 1):** `pass`
- **Verification (Step 2):** `pass`
- **Verification (Step 3):** `pass`
- **Task Gate:** `not_evaluated`（待 Step 1–6 全部完成）
- **Human acceptance:** `false`
- **Task 2A:** `in_progress`
