# Task 7 Report — TUI 基础 + SSE 事件流

## Overview

Checkpoint: `0e523e8d70cb5c3357b9e8beb7a1dae6b2480e46`

建立 Bubble Tea 应用骨架：Tokyo Night 主题、Chat 页面、输入框、Runtime 状态、`AgentRuntime.Subscribe()` 非阻塞接入、Runtime Event→UI Message 转换、Reasoning Processor→UI 流式展示、streaming answer、reasoning 默认折叠 + duration、工具活动时间线、窗口 resize、快捷键、~50ms 合并刷新、`cmd/codea` TUI 启动。

严格边界：App 只依赖 `runtime` + `reasoning` + `theme` + Bubble Tea；不 import opencode/supervisor（在 `cmd/codea` composition root 中接线）。`Subscribe()` 通过 tea.Cmd 包装，绝不阻塞 Bubble Tea Update。Reasoning 只消费 Task 6 Processor，不自行解析 `<think>`。streaming answer 追加到单条 assistant message。

## Acceptance Review Fixes（Blocking 1-4）

首轮人工验收结论为「暂不通过」，锁定 3 个代码/产品 Blocking + 1 个证据缺口，本轮修复（不扩审）：

- **Blocking 1 — 接通 RuntimeSupervisor**：`cmd/codea/main.go` 由「仅读 `OPENCODE_URL` 直连 adapter」改为产品默认链 `main → Supervisor.Start() → Healthy → Supervisor.BaseURL()/Username()/Password() → NewOpenCodeAdapter → Bubble Tea → 退出 → Supervisor.Stop()`。`OPENCODE_URL` 仅保留为 dev/test override。新增 `main_test.go` 组合测试（Supervisor→Start→Healthy→Adapter→App→Stop，`FAKE_OPENCODE_REQUIRE_AUTH=1` 验证凭证接线）+ 启动失败测试（`FAKE_OPENCODE_MODE=exit-immediately` → `Start` 报错 → 不进入 Ready TUI）。
- **Blocking 2 — 修复假 50ms 合并刷新**：`Model` 新增 `streamBuf`/`reasoningBuf` + `rendered`/`dirty` 缓存。高频 delta 写入 buffer（不置 dirty），`~50ms tick` 的 `flushStreaming()` 提交并置 dirty；`View()` 命中缓存直接返回。`coalescing_test.go` 证明 100 个 delta 仅 1 次 flush、token 洪峰期间 `View()` 不重渲染。
- **Blocking 3 — 70×20 最小终端 Gate**：`View()` 在 `width < 70 || height < 20` 时渲染 `Terminal too small / Minimum: 70x20`。`view_test.go` 表驱动覆盖 69×20/70×19/70×20，及 resize 60×10→100×30。
- **Blocking 4 — 真实 TUI smoke 证据**：扩展 `fakeopencode` 提供 `/session`、`/session/{id}/prompt_async`、`/global/event`（脚本化 reasoning/answer/tool/step.finished 事件流）+ `FAKE_OPENCODE_PID_FILE`。新增 `tui/tests/tui-smoke/smoke_test.go`（darwin PTY）真实启动 `codea` 二进制，驱动 启动→Healthy→Prompt→reasoning/answer 流式→tool 活动→resize→ctrl+t→ctrl+c→Runtime 停止 全链路；`scripts/tui-smoke.sh` 可复现。可读证据：`docs/task-reports/tui-smoke-transcript.txt`。

## Acceptance Review Fixes（Round 2 — Blocking 1-3）

第二轮独立验收结论为「有条件通过」，锁定 3 个影响运行正确性的 Blocking，本轮一次性修复（不扩审）：

- **Blocking 1 — `/global/event` 按当前 Session 过滤**：新增 `Model.acceptsEvent(ev)` 统一入口——`SessionID` 为空（真正 global/runtime 事件）恒接受；非空则必须等于当前 session 才进入 chat/reasoning/tool 流程。同时把「CreateSession → 通知 Model sessionID → Prompt」生命周期拆开（`sessionCreatedMsg` + `CreateSessionCmd` + `Model.pendingPrompt`），保证 session 隔离在首个 Prompt 的事件到达前就已生效，消除首 Prompt 竞态。测试：`TestForeignSessionAnswerIgnored` / `TestForeignSessionReasoningIgnored` / `TestForeignSessionToolIgnored` / `TestForeignSessionStepFinishedIgnored` / `TestCurrentSessionEventsAccepted`。
- **Blocking 2 — Streaming 中禁止二次 Submit**：`submit()` 顶部新增 `if m.isStreaming { return nil }`，避免并行提交导致两个 Prompt 流互相污染（answer 串行、reasoning/tool 被二次清空、step.finished 相互干扰）。测试：`TestSubmitIgnoredWhileStreaming`。
- **Blocking 3 — `step.finished` 真正终结 Reasoning**：抽出 `Model.applyReasoningEvents(events)` 统一 apply 逻辑；`step.finished` 走 `m.proc.Flush()`（finalize 无 answer delta 时仍关闭 active reasoning 并产出 `ReasoningEnd` + duration），`session.error`/`runtime.error` 仍走 `m.proc.Process(ev)`（保留 `Interrupted=true` 语义）。测试：`TestStepFinishedFinalizesReasoningWithoutAnswer` / `TestSessionErrorKeepsInterruptedSemantics`。

## Step 1 — Theme + Page + KeyMap

- Created `tui/internal/theme/theme.go`：Tokyo Night 调色板（`#c0caf5` Primary / `#e0af68` Accent / `#9ece6a` Success / `#f7768e` Error / `#292e42` Border / `#1a1b26` Background 等）+ 5 个派生 Style（Chat/Muted/Accent/Success/Error）。仅依赖 lipgloss，零 Runtime/Vendor 知识。
- Created `tui/internal/app/page.go`：`Page` 枚举，V1 仅 `PageChat`；Session/Skill/Agent 页面留给后续 Task。
- Created `tui/internal/app/keymap.go`：`KeyMap` + `DefaultKeyMap()` — enter 提交、alt+enter/ctrl+j 换行、ctrl+c 退出、ctrl+t 切换 thinking、ctrl+l 清屏、`?` 帮助。
- Tests：`theme_test.go`、`page_test.go`、`keymap_test.go`。

## Step 2 — App Model + ChatMessage

- Created `tui/internal/app/model.go`：`Role`（user/assistant/info）、`ChatMessage{Role,Content,Finished}`、`ToolStatus`/`ToolActivity`（工具活动与消息内容分离）、`Model` 结构体 + `NewModel(client)`。Model 无 `permissionModel`/`feedbackModel`（Task 8 边界）。所有 mutation 在 Update 单 goroutine 内，无需 mutex。
- Created `tui/internal/app/messages.go`：内部 tea.Msg 类型 — `subscribedMsg`/`runtimeEventMsg`/`subscribeErrMsg`/`eventStreamClosedMsg`/`tickMsg`/`runtimeStatusMsg`/`promptResultMsg`。
- Tests：`model_test.go`。

## Step 3 — Runtime Subscribe + Tea Cmd（非阻塞）

- Created `tui/internal/app/commands.go`：
  - `SubscribeEvents(client)` → `subscribedMsg{ch}` 或 `subscribeErrMsg`，不阻塞。
  - `waitForEvent(ch)` → 每次消费一个事件并包装为 `runtimeEventMsg`，Update 处理完再重发，事件逐个消费、Update 永不阻塞在 channel 上。
  - `TickCmd()` → `tea.Tick(50ms)` 合并刷新。
  - `PromptCmd` / `CreateSessionAndPromptCmd` → 非阻塞 prompt 提交。
- Created `tui/internal/app/events.go`：Codea domain 事件类型常量（`step.finished`/`session.error`/`runtime.error`/`tool.called`/`tool.success`/`tool.failed`），非 Vendor DTO。
- Modified `tui/tests/fixtures/fake-runtime/fake_runtime.go`：新增 `SubscribeError` 字段，测试订阅失败路径。
- Tests：`commands_test.go`（Subscribe 成功/失败、waitForEvent 关闭、Tick、Prompt）。

## Step 4 — Prompt 输入 + Streaming Answer

- Created `tui/internal/app/update.go`：`Init()` = `tea.Batch(SubscribeEvents, TickCmd)`；`Update()` 处理订阅生命周期、key、prompt 提交、streaming 事件。
  - `submit()`：空白忽略；append user + assistant message；reset processor/reasoning/tools；构建 `runtime.PromptRequest{MessageID, Agent:"general", Parts:[TextPart]}`（复用现有 Domain Contract，不重简化）；首个 prompt 走 `CreateSessionAndPromptCmd`，后续复用 session。
  - `appendAnswer()`：delta 追加到单条 in-flight assistant message，绝不为每个 delta 新建消息。
  - `processRuntimeEvent()`：消费 `step.finished`/`session.error`/`runtime.error` → finishStreaming；tool 事件 → addTool/updateTool；再经 `m.proc.Process(ev)` 消费 answer/reasoning 事件。
- Tests：`submit_test.go`（11 项：空白、消息 append、有/无 session、typing/space/backspace/newline、answer delta 追加、step.finished、reasoning delta）。

## Step 5 — Reasoning UI

- Created `tui/internal/app/view.go` 中 reasoning 渲染：streaming 中显示原始内容；结束后默认折叠为 `✓ Spent Xs thinking`；`ctrl+t` 展开/折叠；duration 直接消费 Task 6 `EventReasoningEnd.Duration`，TUI 不重算。
- `update.go`：`EventReasoningStart` → active + 清空 content；`EventReasoningDelta` → 累加 content；`EventReasoningEnd` → inactive + 记录 duration + 默认折叠。
- Tests：`reasoning_test.go`（toggle、formatDuration、renderReasoning 四态）。

## Step 6 — Tool/Status/View + main.go 接线

- Created `tui/internal/app/view.go` 完整 View：header（app 名 + Runtime 状态点/标签）、状态行（Ready/◌ Working）、工具时间线（◌/✓/✗ + 名称）、输入行、footer 快捷键提示。
  - `statusLabel`/`statusDot`（Healthy/Crashed → `●`，其余 `○`）、`toolSymbol`（success/failed/running）、`formatDuration`、`renderReasoning` 折叠/展开。
- Rewrote `tui/cmd/codea/main.go`：读取 `OPENCODE_URL`/`OPENCODE_USERNAME`/`OPENCODE_PASSWORD` 环境变量，构造 `opencode.NewOpenCodeAdapter`，`app.NewModel(adapter)`，`tea.NewProgram(model, tea.WithAltScreen())` 启动。这是 composition root 唯一接触 vendor 层的位置。
- Modified `scripts/check-runtime-boundary.sh`：允许 `cmd/` composition root import opencode（对齐脚本注释与 `tests/architecture/vendor_boundary_test.go` 的 `isCmd` 排除逻辑）。修复前该脚本误把 `cmd/codea` 接线 vendor adapter 判为泄漏。
- Tests：`view_test.go`（statusLabel/statusDot/toolSymbol 表、renderHeader/StatusLine/Tools、View 含消息、WindowSizeMsg、Quit `tea.QuitMsg`、ClearScreen、tool 事件、subscribe healthy）。

## Full Gate Verification

针对 Final Implementation Commit `0e523e8d70cb5c3357b9e8beb7a1dae6b2480e46`（Round 2 验收修复后）：

| Gate | Result |
|------|--------|
| `GOTOOLCHAIN=local go test ./... -count=1` | PASS（20 packages，无失败） |
| `GOTOOLCHAIN=local go test -race ./... -count=1` | PASS（20 packages，无竞态） |
| `GOTOOLCHAIN=local go vet ./...` | clean |
| `GOTOOLCHAIN=local go build ./...` | clean |
| `GOOS=windows GOARCH=amd64 GOTOOLCHAIN=local go build ./cmd/codea ./cmd/parity-runner` | PASS |
| `GOOS=darwin GOARCH=amd64 GOTOOLCHAIN=local go build ./cmd/codea ./cmd/parity-runner` | PASS |
| `./scripts/check-runtime-boundary.sh` | PASS（无 vendor DTO 泄漏） |
| `./scripts/check-execution-state.sh` | valid |
| `tests/execution-state/state_validator_test.sh` | valid |
| `CODEA_TUI_SMOKE=1 ./scripts/tui-smoke.sh` | PASS（真实 PTY 启动 TUI，全链路） |

## Test Summary

| Package | Tests |
|---------|-------|
| internal/theme | theme_test |
| internal/app | model / keymap / page / commands / update / submit（11）/ reasoning / view（表驱动 + 行为） |
| tests/fixtures/fake-runtime | SubscribeError 路径 |

覆盖清单：Theme 调色板与派生 Style、Page 枚举、KeyMap 绑定、Model 构造与依赖隔离、Subscribe 成功/失败/关闭、非阻塞 waitForEvent、Tick 合并刷新、Prompt/CreateSessionAndPrompt 命令、空白输入忽略、消息 append、streaming answer 单消息追加、typing/space/backspace/newline、step.finished 结束、reasoning start/delta/end + duration + 默认折叠、statusLabel/statusDot/toolSymbol、WindowSizeMsg resize、Quit/ClearScreen/工具事件/subscribe healthy、cmd/codea composition root 接线。

## Files Changed

| File | Action |
|------|--------|
| `tui/internal/theme/theme.go` | Create |
| `tui/internal/theme/theme_test.go` | Create |
| `tui/internal/app/model.go` | Create |
| `tui/internal/app/model_test.go` | Create |
| `tui/internal/app/page.go` | Create |
| `tui/internal/app/page_test.go` | Create |
| `tui/internal/app/keymap.go` | Create |
| `tui/internal/app/keymap_test.go` | Create |
| `tui/internal/app/messages.go` | Create |
| `tui/internal/app/commands.go` | Create |
| `tui/internal/app/commands_test.go` | Create |
| `tui/internal/app/events.go` | Create |
| `tui/internal/app/update.go` | Create |
| `tui/internal/app/update_test.go` | Create |
| `tui/internal/app/submit_test.go` | Create |
| `tui/internal/app/reasoning_test.go` | Create |
| `tui/internal/app/view.go` | Create |
| `tui/internal/app/view_test.go` | Create |
| `tui/cmd/codea/main.go` | Modify |
| `tui/cmd/codea/main_test.go` | Create（supervisor 组合 + 启动失败） |
| `tui/internal/app/model.go` | Modify（streamBuf/reasoningBuf + rendered/dirty） |
| `tui/internal/app/update.go` | Modify（delta 缓冲 + flushStreaming） |
| `tui/internal/app/view.go` | Modify（View cache + 70×20 gate） |
| `tui/internal/app/coalescing_test.go` | Create（合并刷新） |
| `tui/internal/app/submit_test.go` | Modify（tick flush） |
| `tui/internal/app/view_test.go` | Modify（70×20 gate 表驱动） |
| `tui/internal/supervisor/fakeopencode/main.go` | Modify（完整 /session /prompt_async /global/event + PID file） |
| `tui/tests/tui-smoke/smoke_test.go` | Create（真实 TUI smoke，darwin） |
| `scripts/tui-smoke.sh` | Create（可复现 smoke runner） |
| `docs/task-reports/tui-smoke-transcript.txt` | Create（smoke 可读证据） |
| `tui/go.mod` / `tui/go.sum` | Modify（新增 bubbletea/bubbles/lipgloss 依赖） |
| `tui/tests/fixtures/fake-runtime/fake_runtime.go` | Modify（SubscribeError） |
| `scripts/check-runtime-boundary.sh` | Modify（允许 cmd/ composition root） |
| `tui/internal/app/messages.go` | Modify（新增 sessionCreatedMsg） |
| `tui/internal/app/commands.go` | Modify（CreateSessionCmd 替代 CreateSessionAndPromptCmd） |
| `tui/internal/app/model.go` | Modify（新增 pendingPrompt 字段） |
| `tui/internal/app/update.go` | Modify（acceptsEvent + applyReasoningEvents + submit 流控） |
| `tui/internal/app/session_isolation_test.go` | Create（会话隔离 4+1 测试） |
| `tui/internal/app/reasoning_finalize_test.go` | Create（step.finished 终结 reasoning + interrupted 语义） |
| `tui/internal/app/submit_test.go` | Modify（TestSubmitIgnoredWhileStreaming + 两阶段会话） |

## 提交记录

| Commit | Step |
|--------|------|
| `cdcc90d` | Step 1 — Tokyo Night theme + Page + KeyMap |
| `f8bea96` | Step 2 — App Model + ChatMessage + message types |
| `b6e8443` | Step 3 — non-blocking Runtime Subscribe + tea.Cmd |
| `80d551a` | Step 4 — prompt input + streaming answer |
| `516205c` | Step 5 — reasoning UI (toggle + summary + duration) |
| `cedf98b` | Step 6 — tool/status/view + cmd/codea wiring |
| `876b200` | 收口修复 — boundary check 允许 cmd/ composition root（Final Implementation Commit） |
| `7a8f9c2` | 验收修复 — Blocking 1-4（supervisor 接线 / 真实合并刷新 / 70×20 gate / TUI smoke）（Final Implementation Commit） |
| `0e523e8` | Round 2 验收修复 — 会话隔离 / streaming 禁止二次 submit / step.finished 终结 reasoning（Final Implementation Commit） |

## 与计划偏差

1. 计划 `Model` 含 `permissionModel PermissionModel` / `feedbackModel FeedbackModel` 字段；按 Task 7 边界（Session/Approval 属 Task 8，Permission/Feedback 字段不实现）移除，改由 `ToolActivity` 只读展示工具活动。
2. 计划用 `opencode.NewAdapter`；实际 adapter 构造器为 `opencode.NewOpenCodeAdapter`。
3. 计划 Step 6/7（安装依赖 + `git add -A`）拆分为逐 step TDD commit，避免一次 `git add -A` 混入无关 artifact（`tui/tests/parity/evidence/runtime-evidence.json` 时间戳变更被排除）。
4. 修复了 `scripts/check-runtime-boundary.sh` 与 `tests/architecture/vendor_boundary_test.go` 的不一致：shell 脚本误将 `cmd/` composition root 接线 vendor adapter 判为泄漏，现已对齐脚本注释与 Go 测试的 `isCmd` 排除逻辑。

## Gate 结论

- verification：pass
- Task Gate：pass
- 进入 `awaiting_acceptance`，等待人工验收；验收前不启动 Task 8。

## 人工验收

Task 7（TUI 基础 + SSE 事件流）已通过人工验收，正式标记为 `completed`。会话隔离（`acceptsEvent` 按当前 Session 过滤）、streaming 中禁止二次 submit、`step.finished` 终结 reasoning 三项 Round 2 Blocking 修复经复核通过，全 Gate + TUI smoke 回归通过。下一步启动 Task 8（Session/Resume/Tool Approval）。
