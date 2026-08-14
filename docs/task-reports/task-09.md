# Task 9 Report — General Agent 原生能力对齐

## Overview

Checkpoint: `5093f2d91da3fd772a2a1dc1825563f3843efd75`

确保 General Agent 的 Shell、Edit、Subagent、Plugin 能力完整透传，事件零静默丢失。核心边界：Codea 的 AgentRuntime/Adapter/EventMapper/TUI 层不得削弱、阻断或静默丢弃 OpenCode General Agent 原生能力；Subagent 调度、Agent Loop 仍完全属于 OpenCode Runtime，Go TUI/Application 绝不重新实现；Shell 安全引擎属 Task 12，本 Task 不扩范围。

本 Task 交付三件实质工作：(1) 关闭一个真实存在的 Tool 生命周期缺口——`CodeaEventToolSuccess`/`CodeaEventToolFailed` 已定义但从未被映射（Application 层 `internal/app/update.go` 已消费这两类事件），导致 `session.next.tool.success/failed` 落在 dead code；(2) 建立 5 个 parity 门禁测试，证明零静默丢失 + 原生能力不退化；(3) **补真实锁定版本 OpenCode parity smoke**（Step 5），用真实 OpenCode v1.18.11 通过 Codea Adapter 端到端驱动原生 Tool + 审批 + Cancel，生成 fresh evidence 证明「真实 OpenCode General Agent 经过 Codea 后能力不退化」。

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

## Step 5 — 真实 OpenCode parity smoke（本轮补强）

针对人工验收反馈「Contract/EventMapper 证明没丢能力，但未证明真实 OpenCode General Agent 经 Codea 后不退化」，补真实锁定版本 OpenCode v1.18.11 parity smoke。方法论沿用 S2/S3 Phase 0 spike：**只有模型是确定性 stub，OpenCode Runtime、Agent Loop、消息持久化、权限门禁、SSE 全部真实**。

`tests/fixtures/real-parity/fake_model.py` 是一个 OpenAI-compatible 流式 stub，按最后一条 user 消息脚本化一个固定 tool-call 生命周期（READ→read / WRITE→write / EDIT→edit / BASH→bash / SUBAGENT→task(explore) / SKILL→skill / 否则纯文本）。`scripts/run-real-parity-smoke.sh` 启动 fake model（port 49220）+ 真实 `opencode serve`（port 49321），经 Codea `OpenCodeAdapter` 跑 `TestRealRuntimeEvidence`，逐 gate 落盘 fresh evidence 到 `tui/tests/parity/evidence/runtime-evidence.json`，并断言 `available=true && failedChecks=0`。

### 16 个 gate 全绿

| Gate | 结果 |
|------|------|
| health（`/global/health` version=1.18.11） | PASS |
| createSession（`/session`） | PASS |
| sse（`/global/event` 全局订阅） | PASS |
| agentSelection（`ListAgents` + general agent） | PASS |
| capabilities（15 项 required） | PASS |
| read（真实 read tool 执行） | PASS |
| write（真实 write tool 执行） | PASS |
| edit（真实 edit tool 执行） | PASS |
| bashApprovalOnce（`permission.asked` → once → tool 执行 → agent 继续） | PASS |
| bashApprovalAlways（`permission.asked` → always → tool 执行 → agent 继续） | PASS |
| bashApprovalReject（`permission.asked` → reject → tool 不执行） | PASS |
| subagent（`task` 委派 explore subagent 端到端） | PASS |
| skill（skill 发现 + 调用） | PASS |
| plugin（`plugin.added` ×45 启动事件） | PASS |
| sessionResume（session 恢复 + 历史 rehydration） | PASS |
| cancel（真实 streaming session 中 cancel/abort + 后续新 session 正常） | PASS |

`totalChecks=16 passedChecks=16 failedChecks=0 available=true version=1.18.11`。

### smoke 暴露并修复的真实 Bug（非重新实现能力，仅修 Adapter/Harness）

1. **真实 Tool 生命周期映射缺口（event_mapper.go）**：真实 `/global/event` 的 tool 生命周期是 `message.part.updated` 且 `part.type=tool`、`state.status` 依次 `pending→running→completed|error`（共享同一 callID）。原 mapper 只认 `session.next.tool.*` 通道，真实路径的 `completed/error` 会落回 `tool.called`，导致「tool.called 存在而 tool.success 丢失」缺口在真实 runtime 上重开。修复：`ssePart` 新增 `State`，`mapVendorType` 将 `completed→tool.success`、`error→tool.failed`、`pending/running→tool.called`；`extractTool` 从 `props.Part` 回退链取 Name/CallID。单测 `TestEventMapperRealToolLifecycleViaPartUpdated` / `TestEventMapperRealToolLifecycleCallIDCorrelation` 覆盖四态 + CallID 关联。
2. **macOS symlink 路径不一致（run-real-parity-smoke.sh）**：`/tmp→/private/tmp`、`/var→/private/var`。fake model 用未解析 `SMOKE_DIR` 拼文件路径，OpenCode 把 session 目录解析成 canonical 路径，二者不一致时 read/write/edit 目标被当作 `external_directory` 请求权限，tool 卡死在 `running`。修复：`run_root=$(cd "$run_root" && pwd -P)` canonical 化。
3. **Resume 脚本化错误（fake_model.py）**：原 `has_tool_result = any(role=="tool")` 会命中历史里任意 tool result，resume 会话本应脚本新 tool call 却误答纯文本。修复：仅判 `messages[-1].get("role")=="tool"`（紧邻前一条才关闭 tool 循环）。
4. **证据静默丢失（real_parity_smoke_test.go）**：`isIdle()` 被定义却从未调用，session 永不判 idle；`sessionResume`/`cancel` 结果直接赋值绕过 `record()`，失败不进 `failedChecks`。修复：`collect()` 增加 idle 判定，`sessionResume`/`cancel` 走 `record()`；`isIdle` 只认 `session.idle`（去掉 `session.status idle` 分支，避免 status 事件提前 drain 导致下个 scenario 误终止）。
5. **`go test ./...` 覆盖已提交证据（real_parity_smoke_test.go，本轮 Gate 复跑暴露）**：smoke 测试在 runtime 不可达（无 `OPENCODE_ENDPOINT`）时仍 `writeEvidence` 写 `available=false`，导致全量 `go test ./...` 把 harness 刚写入的 green evidence 覆盖成 connection-refused 快照。修复：仅当 `OPENCODE_ENDPOINT`/`OPENCODE_SERVER_URL` 显式设置（即 harness 驱动）时才落盘 skip 证据；普通 `go test` 静默 skip、不触碰已提交证据。已验证：`go test ./tests/parity/ -run TestRealRuntimeEvidence`（无 endpoint）skip 后 evidence 仍 `available=true`。

### 本轮补强（Blocking 1/2 闭环）

1. **ApprovalAlways 真实 Runtime 验证（real_parity_smoke_test.go）**：新增 `bashApprovalAlways` gate，真实走 `permission.asked → ReplyApproval(ApprovalAlways) → /permission/{id}/reply(reply="always") → bash 执行 → agent 继续 → session.idle`，不再只依赖 FakeRuntime/UI 单测。once / always / reject 三种 Domain decision 全部真实闭环。
2. **Cancel 加强（real_parity_smoke_test.go）**：原 `runCancel` 只验证 `Cancel()` API 返回成功即 PASS；现验证「approval pending → Cancel → 该 session 真正 idle → 被取消的 bash 未报 success → 未产出正常 answer → 再新建 session 完整走 read tool 生命周期」。证明 Cancel 生效、全局 SSE 未坏、Runtime 未坏、Adapter 后续仍可工作。
3. **`always` 持久化语义的测试排序（关键发现）**：真实 OpenCode 的 `always` 决策会在 project 级持久化一条 bash 权限规则，导致后续任何 session 的 bash 都自动放行、不再发 `permission.asked`。因此 `bashApprovalAlways` 必须放在 reject/cancel **之后**（作为最后一个 approval scenario），否则会饿死 reject/cancel 的 `permission.asked`。这是真实 runtime 行为，非 stub 假设。

## Full Gate Verification

针对 Final Implementation Commit `5093f2d91da3fd772a2a1dc1825563f3843efd75`：

| Gate | Result |
|------|--------|
| `GOTOOLCHAIN=local go test ./... -count=1` | PASS（20 packages，无失败） |
| `GOTOOLCHAIN=local go test -race ./... -count=1` | PASS（20 packages，无竞态） |
| `GOTOOLCHAIN=local go vet ./...` | clean |
| `GOTOOLCHAIN=local go build ./...` | clean |
| `GOOS=windows GOARCH=amd64 GOTOOLCHAIN=local go build ./cmd/codea ./cmd/parity-runner` | PASS |
| `GOOS=darwin GOARCH=amd64 GOTOOLCHAIN=local go build ./cmd/codea ./cmd/parity-runner` | PASS |
| `./scripts/check-runtime-boundary.sh` | PASS（no vendor DTO leakage） |
| `OPENCODE_BIN=<v1.18.11> ./scripts/run-real-parity-smoke.sh` | PASS（16/16 gates，failedChecks=0） |
| `./scripts/check-execution-state.sh` | valid（Task 9 Step 5 awaiting_acceptance） |
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
| `tui/internal/opencode/event_mapper.go` | Modify（session.next.tool.* 映射 + CallID/Tool 提取；本轮补真实 `message.part.updated` state.status 生命周期映射） |
| `tui/internal/opencode/event_mapper_test.go` | Modify（TestEventMapperSessionNextToolLifecycle；本轮补 RealToolLifecycle 四态 + CallID 关联） |
| `tui/tests/parity/event_passthrough_test.go` | Create |
| `tui/tests/parity/general_agent_test.go` | Create |
| `tui/tests/parity/native_tools_test.go` | Create |
| `tui/tests/parity/session_resume_test.go` | Create |
| `tui/tests/parity/subagent_test.go` | Create |
| `tui/tests/parity/real_parity_smoke_test.go` | Create（真实 runtime 16-gate smoke；本验收轮补 bashApprovalAlways + 加强 runCancel + drainUntilIdle） |
| `tui/tests/parity/parity_runner_test.go` | Modify（real-runtime evidence 结构迁至 real_parity_smoke_test.go） |
| `scripts/run-real-parity-smoke.sh` | Create（真实 OpenCode smoke harness） |
| `tests/fixtures/real-parity/fake_model.py` | Create（确定性流式模型 stub） |
| `tests/fixtures/real-parity/opencode.json` | Create（fake provider 配置，permission.bash=ask） |
| `tests/fixtures/real-parity/skills/smoke-skill/SKILL.md` | Create（smoke skill fixture） |
| `tui/tests/parity/evidence/runtime-evidence.json` | Modify（fresh evidence，16/16 全绿） |

## 与计划偏差

1. 计划 Step 1 文案建议「未映射事件类型 → 添加到 knownTypes 或确认为 Raw 透传」。经 Golden SSE 样本盘点确认 `session.next.*` 非 tool 事件在样本中不存在 legacy 等价语义，强行映射缺乏可验证样本，故保留 raw 透传——零静默丢失由 raw 透传保证，不做无依据的语义化。
2. 计划 Step 2 伪代码引用 `opencode.NewEventMapper()`（构造器）；实际 `MapEvent` 为包级函数（`opencode.MapEvent(raw, seq)`），测试直接调用该函数，无构造器。
3. 计划仅列 Modify `event_mapper.go`「确认覆盖」；实际发现并修复了 `CodeaEventToolSuccess`/`CodeaEventToolFailed` 从未映射的真实缺口（Application 层已消费、Adapter 层从未产出），属超出「确认」的必要修复。
4. `tui/tests/parity/evidence/runtime-evidence.json` 为本轮真实 parity smoke 的 fresh evidence 产物（16/16 全绿），**本次正式纳入提交**，供人工验收直接审阅。

## Gate 结论

- verification：pass
- Task Gate：pass
- 进入 `awaiting_acceptance`，等待人工验收；验收前不启动 Task 10。
