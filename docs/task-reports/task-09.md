# Task 9 Report — General Agent 原生能力对齐

## Overview

Checkpoint: `ee067901291e233a679b5b12e12463cd3352f0b3`

确保 General Agent 的 Shell、Edit、Subagent、Plugin 能力完整透传，事件零静默丢失。核心边界：Codea 的 AgentRuntime/Adapter/EventMapper/TUI 层不得削弱、阻断或静默丢弃 OpenCode General Agent 原生能力；Subagent 调度、Agent Loop 仍完全属于 OpenCode Runtime，Go TUI/Application 绝不重新实现；Shell 安全引擎属 Task 12，本 Task 不扩范围。

本 Task 交付两件实质工作：(1) 关闭一个真实存在的 Tool 生命周期缺口——`CodeaEventToolSuccess`/`CodeaEventToolFailed` 已定义但从未被映射（Application 层 `internal/app/update.go` 已消费这两类事件），导致 `session.next.tool.success/failed` 落在 dead code；(2) 建立 5 个 parity 门禁测试，证明零静默丢失 + 原生能力不退化。

## Step 1 — Golden SSE 覆盖盘点

对比 Task 1 S2 Spike 捕获的 Golden SSE 样本（`runtime/openapi/golden-sse-s2.jsonl`，76 条）与 `event_mapper.go` 映射表，逐一分类：

- **已语义化映射（17 条）**：`server.connected`/`session.*`/`message.updated`/`message.part.removed`/`permission.asked`/`permission.replied`/`runtime_error`，以及 `message.part.updated`（按 `part.type` 细分 step-start/step-finish/tool）与 `message.part.delta`（按 `field` 细分 reasoning/text）两条动态分类路径。
- **Raw 透传（59 条）**：`session.next.*` 现代事件族（step/text/reasoning 等除 tool 生命周期外）及未识别 vendor 类型 → `raw`，`RawType` 保留原类型、`Raw` 保留原始 payload。Golden SSE 样本本身采用 legacy `message.part.*` 表示，`session.next.*` 非 tool 事件映射为 raw 已满足零静默丢失，不强行做无样本可验证的语义映射。
- **结论**：无任何事件被静默丢弃——语义化 or raw 透传，二者必居其一。

## Step 2 — 事件零静默丢失门禁

`tui/tests/parity/event_passthrough_test.go`（Create）：

- `TestGoldenEventsNoSilentDrop`：读取全部 76 条 Golden SSE，逐条 `opencode.MapEvent`，断言 `Type`/`Raw`/`RawType` 三者非空（任一项为空即 `silentDrop++` 并报错）。输出可审计计数：`Golden SSE total=76 semantic=17 raw=59 silentDrop=0`。
- `TestUnknownVendorEventRawPassthrough`：未来/未知 vendor 事件 → `Type=raw`、`RawType` 保留原值、`Raw` 非空。
- `TestMalformedEventRawPassthrough`：损坏 JSON → `Type=_unparseable_`、`Raw` 逐字节保留。

## Step 3 — 原生 Tool 完整性

`tui/tests/parity/native_tools_test.go`（Create）：

- `nativeToolToCapability`：read/grep/glob→fileRead、write→fileWrite、edit→edit、bash→bash、agent→agents、skill→skills、plugin→plugins、abort→abort。
- `TestGeneralAgentNativeToolsRequired`：断言 15 项 General Agent 能力（sessions/streaming/reasoning/fileRead/fileWrite/edit/bash/toolApproval/agents/subagents/skills/plugins/abort/messageHistory/contextCompaction）在 `runtime/capabilities.yaml` 全部声明为 `required`，且真实 `OpenCodeAdapter.Capabilities()` 经 `inv.Compare` 无 required 缺失。
- `TestNativeToolBackedByCapability`：每个原生 Tool 名都映射到 required 能力，Codea 不悄悄剥离任何原生 Tool。

## Step 4 — Tool 生命周期 CallID 关联（含 EventMapper 修复）

`tui/tests/parity/general_agent_test.go`（Create）+ `tui/internal/opencode/event_mapper.go`（Modify）：

- **EventMapper 修复**：`sseCommonProps` 新增 `CallID`/`Tool` 字段；`mapVendorType` 新增 `session.next.tool.called→tool.called`、`session.next.tool.success→tool.success`、`session.next.tool.failed→tool.failed`；重写 `extractTool`——`tool.called` 从 `props.Tool`/`props.CallID`（回退 `part.tool`/`part.callID`/`part.id`）取 Name+CallID，`tool.success`/`tool.failed` 只取 CallID 供 Application 关联生命周期。
- `TestGeneralAgentToolLifecycleCorrelation`：`message.part.updated(tool)`→tool.called + `session.next.tool.success`→tool.success + answer.delta + step-finish，断言 `calledCallID == successCallID`，证明「tool.called 存在而 tool.success 丢失」不再发生。
- `TestGeneralAgentToolFailedCorrelation`：`session.next.tool.called`→tool.called + `session.next.tool.failed`→tool.failed，CallID 关联。
- `TestGeneralAgentNativeToolNames`：read/grep/glob/write/edit/bash 六个原生 Tool 名逐一通过 mapper 为 `tool.called`，Name/CallID 不被改名/丢弃/mangle。

对应单测 `internal/opencode/event_mapper_test.go` 新增 `TestEventMapperSessionNextToolLifecycle`（called/success/failed 三态 → Type + Tool.Name + Tool.CallID）。

## Session 隔离 + Subagent 透传

- `tui/tests/parity/session_resume_test.go`（Create）：
  - `TestSessionResumeIsolation`：创建 A/B 两 session，prompt A1/B1/A2(resume)，断言三次 prompt 分别路由到正确 sessionID，resume 后 `ListSessions` 仍返回 2 个 session。
  - `TestSessionEventTagging`：全局 `/global/event` 流上事件必须携带 `SessionID` 供消费方隔离。
- `tui/tests/parity/subagent_test.go`（Create）：
  - `TestSubagentPassthrough`：`ListAgents` 暴露 subagent（explore, mode=subagent），`Prompt` 携带 `runtime.SubtaskPart{Agent:"explore",...}` 原样透传，不做 Subagent scheduler（调度仍是 OpenCode Runtime 职责）。

## Full Gate Verification

针对 Final Implementation Commit `ee067901291e233a679b5b12e12463cd3352f0b3`：

| Gate | Result |
|------|--------|
| `GOTOOLCHAIN=local go test ./... -count=1` | PASS（20 packages，无失败） |
| `GOTOOLCHAIN=local go test -race ./... -count=1` | PASS（20 packages，无竞态） |
| `GOTOOLCHAIN=local go vet ./...` | clean |
| `GOTOOLCHAIN=local go build ./...` | clean |
| `GOOS=windows GOARCH=amd64 GOTOOLCHAIN=local go build ./cmd/codea ./cmd/parity-runner` | PASS |
| `GOOS=darwin GOARCH=amd64 GOTOOLCHAIN=local go build ./cmd/codea ./cmd/parity-runner` | PASS |
| `./scripts/check-runtime-boundary.sh` | PASS（no vendor DTO leakage） |
| `./scripts/check-execution-state.sh` | valid（Task 9 Step 1 in_progress） |
| `tests/execution-state/state_validator_test.sh` | valid |

## Parity Matrix（Todo 13）

| 能力 | 覆盖测试 | 结论 |
|------|---------|------|
| Shell（bash 透传） | `TestGeneralAgentNativeToolNames`（bash）/ `TestNativeToolBackedByCapability`（bash→bash required） | 透传，不削弱 |
| Edit（edit/write/read） | `TestGeneralAgentNativeToolNames` / `TestNativeToolBackedByCapability` | 透传 |
| Subagent | `TestSubagentPassthrough` | 透传，调度属 Runtime |
| Plugin/Skill | `TestNativeToolBackedByCapability`（plugin/skill→required） | 能力声明 required |
| Session Resume | `TestSessionResumeIsolation` / `TestSessionEventTagging` | 隔离，无串流 |
| Golden SSE 零静默丢失 | `TestGoldenEventsNoSilentDrop`（76=17 semantic+59 raw, silentDrop=0） | 零丢弃 |
| Tool 生命周期 CallID | `TestGeneralAgentToolLifecycleCorrelation` / `TestGeneralAgentToolFailedCorrelation` | 关联正确 |
| 未知/损坏事件 | `TestUnknownVendorEventRawPassthrough` / `TestMalformedEventRawPassthrough` | raw 透传 |

## Files Changed

| File | Action |
|------|--------|
| `tui/internal/opencode/event_mapper.go` | Modify（session.next.tool.* 映射 + CallID/Tool 提取） |
| `tui/internal/opencode/event_mapper_test.go` | Modify（TestEventMapperSessionNextToolLifecycle） |
| `tui/tests/parity/event_passthrough_test.go` | Create |
| `tui/tests/parity/general_agent_test.go` | Create |
| `tui/tests/parity/native_tools_test.go` | Create |
| `tui/tests/parity/session_resume_test.go` | Create |
| `tui/tests/parity/subagent_test.go` | Create |

## 与计划偏差

1. 计划 Step 1 文案建议「未映射事件类型 → 添加到 knownTypes 或确认为 Raw 透传」。经 Golden SSE 样本盘点确认 `session.next.*` 非 tool 事件在样本中不存在 legacy 等价语义，强行映射缺乏可验证样本，故保留 raw 透传——零静默丢失由 raw 透传保证，不做无依据的语义化。
2. 计划 Step 2 伪代码引用 `opencode.NewEventMapper()`（构造器）；实际 `MapEvent` 为包级函数（`opencode.MapEvent(raw, seq)`），测试直接调用该函数，无构造器。
3. 计划仅列 Modify `event_mapper.go`「确认覆盖」；实际发现并修复了 `CodeaEventToolSuccess`/`CodeaEventToolFailed` 从未映射的真实缺口（Application 层已消费、Adapter 层从未产出），属超出「确认」的必要修复。
4. `tui/tests/parity/evidence/runtime-evidence.json` 为 parity 测试运行生成的证据产物（仅 timestamp 变化），已 `git checkout` 还原，不纳入本次提交。

## Gate 结论

- verification：pass
- Task Gate：pass
- 进入 `awaiting_acceptance`，等待人工验收；验收前不启动 Task 10。
