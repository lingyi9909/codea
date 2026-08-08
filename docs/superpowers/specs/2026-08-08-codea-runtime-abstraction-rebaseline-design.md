# Codea V1 Runtime Abstraction Rebaseline Design

日期：2026-08-08

状态：待人工验收；验收前不得开始 Task 2A 实现或 Task 3

## 1. 决策

Codea V1 采用：

> **OpenCode First，Runtime Agnostic**

V1 只实现并发行 OpenCode Runtime，不实现 OMP，不提供 Runtime 选择、切换或路由。Codea 上层通过自己拥有的 Runtime Contract 使用 OpenCode；OpenCode OpenAPI DTO、HTTP/SSE 协议对象只存在于 Vendor Layer。

本次调整不是更换 Runtime，也不是重做 Task 0～Task 2。它将原计划 Task 4 的核心边界提前到 Task 3 之前，避免 Capability Inventory、Parity Harness、TUI、Harness 和企业 Agent 在建立过程中直接绑定 OpenCode DTO。

未来接入第二个 Runtime 时，本边界应显著减少上层修改，但不承诺“只写 Adapter、上层零修改”。只有第二个实现接入后才能验证抽象的完备性。

## 2. 保留与禁止范围

### 2.1 完全保留

- Task 0 项目骨架和 Go Module。
- Task 1 OpenCode Phase 0 Spike、Golden SSE、OpenAPI、离线与多平台证据。
- Task 2 OpenAPI 3.1 代码生成器、472 个组件 DTO、162 条路径、HTTP Client、测试与验证证据。
- OpenCode v1.18.11 作为 V1 Reference Runtime Baseline。

Task 2 的产物重新定位为：

```text
OpenCode Vendor Client
```

而不是 Codea Domain API。

### 2.2 Task 2A 禁止范围

Task 2A 不得实现：

- OMP 或 OMP Adapter
- Multi Runtime Router、Runtime Selector、热切换或故障切换
- Swarm、Agent Graph 或多 Engine 调度
- 新 Reviewer、Tester、Debugger、Fixer
- 完整 Codea Tool Runtime、OS Sandbox、Workspace Isolation 重构
- Windows Sandbox 重构
- Tool Policy Engine；Task 2A 只定义 Approval Domain
- Harness/Application Task Event；它们不属于 Runtime Event

## 3. 依赖方向

```text
TUI / Harness / Enterprise Agent / Policy
                    │
                    ▼
             Codea Application
                    │
                    ▼
          tui/internal/runtime
             AgentRuntime API
                    ▲
                    │ implements
       tui/internal/opencode
   Adapter / Mapper / Client / DTO
                    │
                    ▼
             opencode serve
```

正确依赖：

```text
opencode adapter -> runtime contract
application      -> runtime contract
composition root -> runtime contract + opencode adapter
```

禁止依赖：

```text
runtime contract -> opencode
application      -> opencode DTO/client
TUI/Harness      -> opencode DTO/client
```

所有 Go 代码继续位于 `tui/`。根目录 `runtime/` 继续只存放 OpenCode 版本、能力清单和锁定 Spec，不放 Go Adapter 代码。

## 4. Runtime Contract

### 4.1 生命周期分离

Runtime 协议与 Runtime 进程生命周期分离：

- `AgentRuntime`：Health、Session、Prompt、全局事件、Approval、Cancel、Agent、Capabilities。
- `RuntimeSupervisor`：Start、Stop、Status；仍由原 Task 5 负责。

Task 2A 不把 `Start/Stop` 加入 `AgentRuntime`。

### 4.2 最小接口

```go
type AgentRuntime interface {
	Health(ctx context.Context) (HealthInfo, error)
	CreateSession(ctx context.Context, req CreateSessionRequest) (Session, error)
	Prompt(ctx context.Context, sessionID SessionID, req PromptRequest) error
	Subscribe(ctx context.Context) (<-chan Event, error)
	ReplyApproval(ctx context.Context, approvalID ApprovalID, reply ApprovalReply) error
	Cancel(ctx context.Context, sessionID SessionID) error
	ListAgents(ctx context.Context) ([]Agent, error)
	Capabilities() RuntimeCapabilities
}
```

`Subscribe` 必须保持全局订阅，因为锁定 OpenCode API 是 `GET /global/event`。Adapter 负责解析并在 Event 中保留 ProjectID、SessionID、MessageID 和 PartID；上层按标识过滤。不得伪装成 session-scoped 传输并静默丢失其他事件。

### 4.3 领域请求不能裁剪真实能力

`PromptRequest` 至少表达 Task 2 已确认的真实输入：

```go
type PromptRequest struct {
	MessageID string
	Agent     string
	Model     *ModelRef
	Parts     []PromptPart
}

type PromptPart interface {
	isPromptPart()
}

type TextPart struct{ Text string }
```

除 `TextPart` 外还必须提供 File、Agent、Subtask 三个领域变体。它们分别覆盖锁定 Spec 中的 `mime/url/filename/source`、`name/source`、`agent/description/prompt/command/model`，并保留各自可选 ID。实现时必须从生成 DTO 和锁定 Spec 提取确切必填/可选关系，不得照抄旧计划中的示意字段。不得把 Prompt 简化为单个 `string content`。

### 4.4 Approval

Codea Domain 使用 `ApprovalReply` 统一承载明确的决策枚举与可选消息：

```go
type ApprovalDecision string

const (
	ApprovalOnce   ApprovalDecision = "once"
	ApprovalAlways ApprovalDecision = "always"
	ApprovalReject ApprovalDecision = "reject"
)

type ApprovalReply struct {
	Decision ApprovalDecision
	Message  string // optional
}
```

`ReplyApproval` 必须接收完整的 `ApprovalReply`，不得退化为只接收 `ApprovalDecision`，否则会丢失拒绝原因等可选消息。

OpenCodeAdapter 映射到当前非废弃的 `/permission/{requestID}/reply`。OpenCode Spec 没有 `remember` 字段，Adapter 不得伪造；Codea 项目级记忆策略属于后续 Policy/Application 层。

## 5. Capabilities 双层模型

两类能力不得合并：

```text
RuntimeCapabilities = 当前 Runtime 实际声明支持的能力
Capability Inventory = Codea 产品 required/optional/deferred 要求
Parity Compare        = 实际能力与产品要求的比较结果
```

Task 2A 只提供 OpenCodeAdapter 的实际能力声明。Task 3 继续负责加载 `runtime/capabilities.yaml`、比较基线、运行 Parity Harness。

上层不得使用：

```go
if runtimeName == "opencode" { ... }
```

应使用能力判断。Capability key 必须来自已验证能力或 V1 已确定需求，不为 OMP 猜测字段。

## 6. Event Protocol

### 6.1 Runtime Event 与 Application Event 分离

Task 2A 的 Runtime Event 只覆盖：

- Session/Message/Part 生命周期
- Reasoning 与 Answer
- Tool 生命周期
- Approval 请求与结果
- Runtime 连接、错误和取消
- Raw Event

`TaskStarted`、`TaskCompleted`、`ArtifactCreated` 等属于未来 Application/Harness Event，不在 OpenCodeAdapter 中合成。

### 6.2 Event Envelope

```go
type Event struct {
	ID             string
	Type           EventType
	Sequence       int64
	ProjectID      string
	SessionID      string
	MessageID      string
	PartID         string
	CreatedAt      time.Time
	Content        string
	Tool           *ToolEvent
	Approval       *ApprovalRequest
	Error          *RuntimeError
	Metadata       map[string]string
	RawType        string
	Raw            json.RawMessage
	RawSensitivity Sensitivity
	RawTruncated   bool
	RawOriginalSize int
}
```

映射必须依据 Task 1 Golden SSE 与 Task 2 生成 DTO。旧计划中的手写 snake_case 示例不作为协议事实。

### 6.3 零静默丢失

每一条成功解析的 OpenCode 事件必须满足：

```text
Mapped Codea Event + Raw payload
OR
Unknown/Raw Event + Raw payload
```

JSON 无法解析时也必须形成 `_unparseable_` Raw Event。建立订阅前的 HTTP/认证错误由 `Subscribe` 直接返回；Channel 建立后的 Scanner/网络错误必须先发送 `runtime_error` Event 再关闭。调用方主动取消 Context 时可以正常关闭，但不得泄漏 goroutine。

### 6.4 Raw Event 安全

继承现有技术设计约束：

- 默认只进入最近 500 条的内存环形缓冲区
- 默认不写审计日志
- UI 展示前执行 DLP
- 16KB 以内保留精确 Raw；超过 16KB 时截断并设置 `RawTruncated=true`、记录 `RawOriginalSize`，不得静默截断
- 会话结束或 Runtime 重启后清理
- 仅 Debug 模式可导出，导出前再次确认并脱敏

Raw payload 是兼容性兜底，不是绕过 DLP 的后门。

## 7. 目录与依赖门禁

Task 2A 使用现有布局，避免为目录美观搬迁 6,000 行以上生成 DTO：

```text
tui/internal/runtime/
    client.go
    models.go
    events.go
    approval.go
    capabilities.go

tui/internal/opencode/
    dto.go                 # Task 2 generated
    http_client.go         # Task 2 vendor client
    sse_client.go
    adapter.go
    event_mapper.go
    approval_mapper.go
    capabilities.go
```

依赖门禁基于 `go list -deps -json` 或 Go import parser，不仅依赖字符串搜索。禁止包至少包括 Application、TUI models/components、Harness、Agent 和 Policy；允许包只包括 OpenCode Vendor Layer、对应测试和最终 composition root。

目标：Vendor DTO leakage = 0。

## 8. Task 重基线

### Task 2A

建立 Domain、AgentRuntime、OpenCodeAdapter、Mapper、Capabilities、依赖门禁和 Contract Test。不得开始上层业务能力。

### Task 3

继续实现 Capability Inventory + Parity Harness，但改为消费 `AgentRuntime` 和 `RuntimeCapabilities`。Fake Runtime 必须实现 Codea Contract；不得继续使用与锁定 Spec 不一致的手写 OpenCode JSON。

### Task 4

改为 Runtime Adapter Hardening：SSE 断线可观察性、重连、Session 状态/消息补偿、错误分类、背压与真实 Runtime 集成验证。不得重复创建 Task 2A 已完成的接口和基础 Mapper。

### Task 5

继续负责 Supervisor + Basic Auth + 跨平台进程管理。`Start/Stop` 不前移到 Task 2A。

## 9. Task 2A Required Gate

1. **Task 2 无退化**：生成一致性、HTTP Client 测试、Go test/race/vet/build 全部通过。
2. **Contract 完整**：编译期确认 OpenCodeAdapter 实现 AgentRuntime。
3. **DTO 零泄漏**：禁止包的 import graph 不含 OpenCode Vendor 包。
4. **Event 零静默丢失**：Golden SSE 中每条事件均映射或 Raw；未知和非法 JSON 有 Raw 证据。
5. **Approval parity**：once/always/reject 与可选 message 正确映射，不出现 `remember`。
6. **Runtime parity**：真实 OpenCode 的 Health、Session、Prompt、SSE、Reasoning、Answer、Approval、Deny、Abort、Agent 路径通过 Adapter 工作。
7. **Offline 无新增风险**：不得新增运行时下载或公网依赖；若未修改 Supervisor/发行包，复用 Task 1 的离线启动证据并记录理由。
8. **Windows 无新增风险**：Go 1.26.5 Windows x64 cross-build 必须通过；有 Windows Runner 时执行完整 Runtime smoke。没有 Runner 时不得伪造 PASS，完整 Windows E2E 留在发行/Task 21 Gate。
9. **人工验收**：所有自动验证和 Task Gate 通过后进入 `awaiting_acceptance`，不得自动开始 Task 3。

## 10. 执行状态

执行状态升级到 schema v2，并使用显式顺序：

```yaml
schemaVersion: 2
taskOrder: ["0", "1", "2", "2A", "3", "4", "...", "21"]
```

`tasks` 的 key 必须与 `taskOrder` 完全一致；第一个未完成 Task 由 `taskOrder` 决定。Task 2A 报告路径为 `docs/task-reports/task-02A.md`。

## 11. 验收结论

本文经人工验收后，唯一允许开始的是 Task 2A。Task 2A 人工验收完成前，Task 3 保持 `pending`。
