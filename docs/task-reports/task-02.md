# Task 02 Report — OpenAPI Spec 生成 DTO 与 Client

**Task:** 2

**Status:** completed

**Current step:** 5 — 人工验收通过，Task 正式结束

**Date:** 2026-08-08

**Checkpoint:** `174bb2997aa206736775c54a31f6de8ee82261f9`

## 完成内容

### Step 1：审阅锁定 OpenAPI Spec

已审阅 `runtime/openapi/opencode-1.18.11.json` 中 Task 2 要求的六类接口：

| 能力 | 方法与真实路径 | 锁定 Spec 结论 |
|---|---|---|
| 健康检查 | `GET /global/health` | 200 响应必含 `healthy: true` 与 `version` |
| 创建 Session | `POST /session` | 请求可含 `parentID`、`title`、`agent`、`model`、`metadata`、`permission`、`workspaceID`；响应为 `Session` |
| 异步 Prompt | `POST /session/{sessionID}/prompt_async` | 请求必含 `parts`；`model` 使用 `providerID` 与 `modelID`；成功响应为 204 |
| Permission 回复 | `POST /permission/{requestID}/reply` | 非废弃端点；请求必含 `reply=once|always|reject`，可含 `message`；成功响应为 boolean |
| Agent 列表 | `GET /agent` | 200 响应为 `Agent[]` |
| SSE 事件 | `GET /global/event` | 200 `text/event-stream`，Schema 为 `GlobalEvent` |

### Step 2：实现 OpenAPI 3.1 代码生成器

- 新增 `tui/cmd/openapi-gen`，命令格式为 `openapi-gen <spec.json> <output.go>`。
- 遍历全部 `components/schemas` 与 `paths`，确定性生成 Go 组件类型、请求 DTO 和成功响应 DTO。
- 支持对象、数组、标量、组件引用、`additionalProperties` 和内联对象提升；可选内联对象使用指针，避免 `omitempty` 仍发送空对象。
- 对清洗后重名的 schema 分配稳定唯一名称，并让全部 `$ref` 使用同一映射。
- 生成前校验所有 `$ref`；非本地引用或不存在的组件直接失败，不静默降级。

### Step 3：生成并人工审阅 DTO

- 从锁定的 OpenAPI 3.1.0 Spec 生成 `tui/internal/opencode/dto.go`。
- 生成范围覆盖 472 个组件 schema 与 162 条路径；生成文件通过 Go 语法与类型检查。
- 生成一致性测试会重新生成 DTO 并与已提交文件逐字节比较，防止 Spec 与 DTO 漂移。
- 抽查确认 Session 创建、Prompt、Permission、Agent、Health 与 Global Event 类型来自锁定 Spec。

### Step 4：实现使用生成 DTO 的 HTTP Client

- `Health`：`GET /global/health`。
- `CreateSession`：`POST /session`。
- `SendPrompt`：`POST /session/{sessionID}/prompt_async`，严格接受 204。
- `ApprovePermission`：使用非废弃 `POST /permission/{requestID}/reply`；请求不含 `remember`。
- `AbortSession`：`POST /session/{sessionID}/abort`。
- `ListAgents`：`GET /agent`。
- 所有请求统一处理 Basic Auth、JSON 头、精确成功状态码、有界错误体、上下文取消和生成 DTO 编解码。

### Step 5：形成代码 checkpoint

代码提交：

```text
0316ed0c3b64a9f2169a2ea11e946ae1000ae7c8
feat: generate OpenAPI DTOs and typed HTTP client
```

## 实际文件变更

- `tui/cmd/openapi-gen/main.go`：OpenAPI 3.1 DTO 生成器。
- `tui/cmd/openapi-gen/main_test.go`：fixture、完整锁定 Spec、引用校验、类型检查和生成漂移测试。
- `tui/internal/opencode/dto.go`：由锁定 v1.18.11 Spec 生成的 DTO。
- `tui/internal/opencode/http_client.go`：使用生成 DTO 的普通 HTTP Client。
- `tui/internal/opencode/http_client_test.go`：六类 HTTP 行为、认证、序列化与错误响应测试。
- `docs/execution-state.yaml`：记录 Task 2 从 `awaiting_acceptance` 到人工验收完成的状态流转。
- `docs/task-reports/task-02.md`：本报告。
- `docs/codea-v1-handoff.md`：交接点更新为 Task 2 人工验收。

OpenCode Core 修改文件数：0，符合最多 5 个文件的约束。

## 执行命令与验证结果

| 命令 | 结果 | 摘要 |
|---|---|---|
| TDD 单测逐项运行 | PASS | 每个新增行为先出现预期 RED，再由最小实现转为 GREEN |
| `cd tui && go test ./... -count=1` | PASS | 全部 Go 包测试通过 |
| `cd tui && go test -race ./... -count=1` | PASS | Race Detector 通过 |
| `cd tui && go vet ./...` | PASS | 无 vet 问题 |
| `cd tui && go build ./...` | PASS | Go 1.26.5 全量构建通过 |
| 生成器重跑后 `cmp` | PASS | 新生成 DTO 与已提交 `dto.go` 逐字节一致 |
| 锁定 Spec 结构断言 | PASS | OpenAPI 3.1.0、162 paths、472 schemas |
| `./scripts/check-execution-state.sh` | PASS | 代码收口时 Task 2 `awaiting_acceptance` 状态合法 |
| 废弃路径、占位符、凭据与 `git diff --check` 扫描 | PASS | 无命中、无格式错误 |

## 计划偏差与处理

- 主计划中的 DTO 代码是示意结构，存在 `Session.status/agent/created_at`、Prompt `model.id`、Permission `remember` 等与锁定 Spec 不一致的字段；实现严格采用 v1.18.11 Spec，而非照抄示例。
- 主计划表列出 session-scoped Permission 路径；Client 改用 Task 1 已实证且 Spec 未标记废弃的 `/permission/{requestID}/reply`。
- OpenAPI union 在 Go 中没有直接等价类型；生成器将 union 容器保留为 `any`，同时生成其全部具体组件 DTO。Prompt `parts` 因此为 `[]any`，调用方使用生成的 `OpenCodeTextPartInput`、`OpenCodeFilePartInput`、`OpenCodeAgentPartInput` 或 `OpenCodeSubtaskPartInput` 填充，不猜测字段。
- SSE 传输客户端与领域事件映射经 2026-08-08 Architecture Rebaseline 前移到 Task 2A；Task 2 只生成 `GlobalEvent` DTO，不提前实现 Adapter。

## 未解决问题与恢复建议

- **阻塞项：**无。
- **非阻塞边界：**union 容器依赖生成的具体变体 DTO；Task 2A OpenCodeAdapter 必须按锁定 Spec 做领域转换，不得直接向 TUI 暴露 OpenCode DTO。
- **恢复点：**代码 checkpoint `0316ed0c3b64a9f2169a2ea11e946ae1000ae7c8`。
- **下一步：**等待 2026-08-08 Runtime Abstraction Rebaseline 文档人工验收，再按门禁执行 Task 2A。
- **范围边界：**Task 2 本身已经结束；Task 2A 是后续独立 Task，不修改 Task 2 代码成果。

## Gate 结论

- **Verification:** `pass`
- **Task Gate:** `pass`
- **Human acceptance:** `true`（用户于 2026-08-08 明确验收通过）
- **Task 2:** `completed`
