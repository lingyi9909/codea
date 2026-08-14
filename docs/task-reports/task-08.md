# Task 8 Report — Session/Resume/Tool Approval

## Overview

Checkpoint: `bfac6654ed650526b4ac5401ec76729c14c35bb6`

实现 Session 列表/恢复、Tool 权限确认弹窗、危险命令检测。核心边界：TUI/Application 层绝不接触 OpenCode Vendor DTO/Permission 类型，全部消费 Codea domain（`runtime.ApprovalRequest` / `runtime.ApprovalReply` / `runtime.ApprovalDecision` / `AgentRuntime.ReplyApproval`）；UI 不伪造 vendor "remember" 标志，once/always/reject 三种决策直接映射到 Contract，由 Adapter 负责映射到 Vendor。网络请求全部走 `tea.Cmd`，绝不阻塞 Bubble Tea Update。

## Step 1 — Session 列表组件

- Created `tui/internal/components/session.go`：`SessionItem{ID, Title, UpdatedAt, Active}`、`SessionModel{Items, Cursor, Visible}`，纯展示组件，只拥有光标/可见性/显示，绝不直接与 Runtime 通信；Application 喂入 Codea-domain SessionItem 并回读 `Selected()`。
  - `Open/Close/MoveUp/MoveDown/Selected/SetActive/View`；View 中当前 session 标记 `(active)`、光标行前缀 `>`、空列表显示 `(no sessions)`。
- Tests：`session_test.go`（OpenClose / CursorMovement / Selected / SelectedEmpty / SetActive / View / ViewEmpty）。

## Step 2 — Runtime Contract：ListSessions + 扩展 Domain

- `tui/internal/runtime/client.go`：`AgentRuntime` 新增 `ListSessions(ctx) ([]Session, error)`（原 Contract 缺失，复用已有 `GetSessionStatus` HTTP client 实现）。
- `tui/internal/runtime/models.go`：`Session` 扩展 `Title string` + `UpdatedAt time.Time`（`import "time"`）。
- `tui/internal/runtime/events.go`：`ApprovalRequest` 扩展 `Command string`。
- `tui/internal/opencode/session_mapper.go`（Create）：`MapSession(OpenCodeSessionV2Info) runtime.Session`，`UpdatedAt` 用 `time.UnixMilli(info.Time.Updated)`。
- `tui/internal/opencode/adapter.go`：`ListSessions` 实现——`GetSessionStatus` → `MapSession` 逐条映射，错误走 `classifyError("ListSessions", err)`。
- `tui/tests/fixtures/fake-runtime/fake_runtime.go`：新增 `Sessions []runtime.Session` 字段（非 nil 时原样返回）+ `ListSessions` 方法（否则从已建 session 派生）。

## Step 3 — Session Resume（重置瞬态 UI 状态）

`tui/internal/app/update.go`：
- `toggleSessions()`：打开面板（清空 Items/Cursor + 发 `ListSessionsCmd`）或关闭。
- `resumeSelectedSession()`：`Selected()` 为空或 `Active` → 直接关闭；`isStreaming` → 置 `sessionNotice` 并保留面板（Step 10 流控）；否则 `resumeSession(item.ID)` + 关闭。
- `resumeSession(id)`：切换 sessionID 并**重置全部瞬态 UI 状态**（`isStreaming=false`、`pendingPrompt=nil`、`streamBuf/reasoningBuf` Reset、reasoning 五态、`tools` 清空、`proc.Reset()`、`messages` 清空、`sessionNotice` 清空）。消息历史 rehydration 属 Task 9（需 message-part 映射），故 resume 后 chat 视图从空开始。

## Step 4 — Approval 权限组件

- Created `tui/internal/components/permission.go`：`PermissionModel{Request *runtime.ApprovalRequest}`，只消费 Codea domain，绝不触碰 Vendor Permission DTO 或 reply 端点。
  - `NewPermissionModel(req)` / `Visible()`（`Request != nil`）/ `View()`（`box()` 边框，渲染 "Tool approval required"、Tool 名、command，危险命令时追加 `⚠ Potentially dangerous command`，footer `[Y] Allow once`/`[A] Always allow`/`[N] Reject`）。
  - `box()` / `runeLen()` / `padRight()` 纯渲染 helper。
- Tests：`permission_test.go`（Visible / ViewDangerous / ViewSafeNoWarning / EmptyView）。

## Step 5 — Approval 状态进入 App Model

- `tui/internal/app/model.go`：`import "codea/tui/internal/components"`；新增 `sessionPanel components.SessionModel`、`sessionNotice string`、`permission components.PermissionModel`、`approvalErr string`。
- `tui/internal/app/events.go`：新增 `eventTypeApprovalRequested runtime.EventType = "approval.requested"`。
- `tui/internal/app/messages.go`：新增 `listSessionsResultMsg{sessions, err}` 与 `approvalResultMsg{approvalID, err}`。
- `tui/internal/app/keymap.go`：新增 `Sessions(ctrl+s)`、`Up`、`Down`、`Esc`、`AllowOnce(y)`、`AllowAlways(a,r)`、`Reject(n)` 绑定。

## Step 6 — Approval 决策映射（复用现有 ApprovalDecision）

`tui/internal/app/update.go`：
- `handleApprovalKey(msg)`：`AllowOnce → ApprovalOnce`、`AllowAlways → ApprovalAlways`、`Reject/Esc → ApprovalReject`，其余键吞掉。
- `replyApproval(decision)`：`Request == nil` 短路；否则 `ReplyApprovalCmd(client, ApprovalID(Request.ID), ApprovalReply{Decision: decision})`。**不重定义 Approval DTO，不伪造 vendor "remember"**。

## Step 7 — 异步 Approval 执行（tea.Cmd，绝不阻塞）

- `tui/internal/app/commands.go`：新增 `ListSessionsCmd` 与 `ReplyApprovalCmd`，均返回 `tea.Cmd`（goroutine 内调用 client，结果以 tea.Msg 返回），绝不阻塞 Update。
- `tui/internal/app/update.go` `Update()` 新增 `listSessionsResultMsg` 与 `approvalResultMsg` 分支：列表成功 → `sessionPanel.Open` + `SetActive`；失败 → `sessionNotice`。approval 成功 → 清空 `permission` + `approvalErr`；失败 → `approvalErr` 置错误、modal 保持打开（可重试/拒绝/关闭）。

## Step 8 — 危险命令检测

- Created `tui/internal/components/tool.go`：`dangerousCommands` 覆盖 Unix + Windows（`rm -rf`、`git reset --hard`、`git clean -fd`、`git push --force`、`chmod 777`、`> /dev/sda`、`mkfs.`、`dd if=`、fork bomb、`del /s /q`、`rmdir /s /q`、`rd /s /q`、`format`、`diskpart`、`remove-item -recurse -force`、`stop-computer`、`restart-computer`）。
- `IsDangerousCommand(input) (bool, string)`：大小写不敏感 + trim，返回匹配 fragment。
- 明确注释：仅 UI/Policy 警告，非安全引擎，不做 shell 解析/沙箱。
- Tests：`tool_test.go`（Unix / Windows / 安全命令不命中 / 大小写 / 空白）。

## Step 9 — 组合测试（Approval + Session + Global Event）

- Created `tui/internal/app/approval_test.go`（9 项）：`TestApprovalCurrentSessionAllowOnce`（模态打开 + `ReplyApprovalCmd` + fake client `Approvals()` 校验 `ApprovalOnce` + 成功后关闭）、`TestApprovalForeignSessionIgnored`、`TestApprovalAfterResumeUsesNewSession`、`TestApprovalAlwaysDecision`、`TestApprovalRejectDecision`、`TestApprovalRejectViaEsc`、`TestApprovalErrorKeepsModalOpen`（`ReplyApprovalError` → 模态保持 + `approvalErr`）、`TestApprovalModalBlocksChatKeys`（打字/enter/ctrl+s 被吞）、`TestApprovalViewRendersDangerWarning`。
- Created `tui/internal/app/session_resume_test.go`（9 项）：`TestResumeSwitchesCurrentSession`、`TestResumeSessionResetsTransientState`（直调 `resumeSession` 单测重置逻辑）、`TestOldSessionAnswerIgnoredAfterResume`、`TestOldSessionReasoningIgnoredAfterResume`、`TestOldSessionToolIgnoredAfterResume`、`TestOldSessionStepFinishedIgnoredAfterResume`、`TestResumedSessionEventsAccepted`、`TestResumeBlockedWhileStreaming`、`TestSessionPanelEscCloses`。
- 关键 helper：`openPanelAndResume()` 驱动 `ctrl+s → listSessionsResultMsg → 光标到 target → Enter` 完整恢复链路；旧 session 事件隔离测试采用「先 resume 到 s2 → 在 s2 提交新 prompt → 发 s1 旧事件」模式，使污染检查有意义。

## Step 10 — Streaming 阻塞 Resume

`resumeSelectedSession()` 在 `m.isStreaming` 时置 `sessionNotice` 且**保留面板**、不切换 session。`TestResumeBlockedWhileStreaming` 验证 sessionID 不变 + 面板保持 + notice 置位。

## Step 1-8 补充：OpenCode 事件映射（command 提取）

- `tui/internal/opencode/event_mapper.go`：`ssePermissionProps` 扩展 `Patterns []string` + `Metadata map[string]any`；`extractApproval` 填充 `Approval.Command`；新增 `permissionCommand()`（优先 `metadata.command`，回退 `strings.Join(patterns, " ")`）。
- `tui/internal/opencode/event_mapper_test.go`：新增 `TestEventMapperExtractsApprovalCommand`。

## Full Gate Verification

针对 Final Implementation Commit `bfac6654ed650526b4ac5401ec76729c14c35bb6`：

| Gate | Result |
|------|--------|
| `GOTOOLCHAIN=local go test ./... -count=1` | PASS（20 packages，无失败） |
| `GOTOOLCHAIN=local go test -race ./... -count=1` | PASS（20 packages，无竞态） |
| `GOTOOLCHAIN=local go vet ./...` | clean |
| `GOTOOLCHAIN=local go build ./...` | clean |
| `GOOS=windows GOARCH=amd64 GOTOOLCHAIN=local go build ./...` | PASS |
| `GOOS=darwin GOARCH=amd64 GOTOOLCHAIN=local go build ./...` | PASS |
| `./scripts/check-runtime-boundary.sh` | PASS（无 vendor DTO 泄漏） |
| `./scripts/check-execution-state.sh` | valid |
| `tests/execution-state/state_validator_test.sh` | valid |
| `CODEA_TUI_SMOKE=1 ./scripts/tui-smoke.sh` | PASS（真实 PTY 启动 TUI，全链路） |

## Test Summary

| Package | Tests |
|---------|-------|
| internal/components | session_test（7）/ permission_test（4）/ tool_test（5） |
| internal/app | approval_test（9）/ session_resume_test（9）+ 既有会话隔离/推理终结/submit 回归 |
| internal/opencode | event_mapper_test（approval command 提取） |
| tests/fixtures/fake-runtime | ListSessions 路径 |

覆盖清单：Session 列表光标/选中/激活标记/视图、权限模态可见性/危险警告/安全无警告/空视图、危险命令 Unix+Windows+大小写+空白、Approval once/always/reject/Esc 映射、外来 session 忽略、resume 后新 session、审批错误保持模态、模态吞掉聊天键、resume 切换 session + 瞬态重置 + 旧 session 四类事件隔离 + 新 session 事件接受 + streaming 阻塞 + Esc 关闭面板、ListSessions 契约、approval command 从 `metadata.command`/`patterns` 提取。

## Files Changed

| File | Action |
|------|--------|
| `tui/internal/components/session.go` | Create |
| `tui/internal/components/session_test.go` | Create |
| `tui/internal/components/permission.go` | Create |
| `tui/internal/components/permission_test.go` | Create |
| `tui/internal/components/tool.go` | Create |
| `tui/internal/components/tool_test.go` | Create |
| `tui/internal/opencode/session_mapper.go` | Create |
| `tui/internal/app/approval_test.go` | Create |
| `tui/internal/app/session_resume_test.go` | Create |
| `tui/internal/runtime/client.go` | Modify（AgentRuntime.ListSessions） |
| `tui/internal/runtime/models.go` | Modify（Session Title/UpdatedAt） |
| `tui/internal/runtime/events.go` | Modify（ApprovalRequest.Command） |
| `tui/internal/opencode/adapter.go` | Modify（ListSessions 实现） |
| `tui/internal/opencode/event_mapper.go` | Modify（permission command 提取） |
| `tui/internal/opencode/event_mapper_test.go` | Modify（command 提取测试） |
| `tui/internal/app/commands.go` | Modify（ListSessionsCmd/ReplyApprovalCmd） |
| `tui/internal/app/events.go` | Modify（eventTypeApprovalRequested） |
| `tui/internal/app/keymap.go` | Modify（Sessions/Up/Down/Esc/Allow*/Reject 绑定） |
| `tui/internal/app/messages.go` | Modify（listSessionsResultMsg/approvalResultMsg） |
| `tui/internal/app/model.go` | Modify（sessionPanel/sessionNotice/permission/approvalErr） |
| `tui/internal/app/update.go` | Modify（handleKey 分层 / approval / session / resume / 事件分支） |
| `tui/internal/app/view.go` | Modify（模态/面板 overlay + footer ctrl+s） |
| `tui/tests/fixtures/fake-runtime/fake_runtime.go` | Modify（Sessions + ListSessions） |
| `docs/task-reports/tui-smoke-transcript.txt` | Modify（footer 新增 ctrl+s sessions） |

## 提交记录

| Commit | Step |
|--------|------|
| `bfac665` | 全部 Step 1-10 一次性实现（Session 列表/恢复、Tool Approval 弹窗、危险命令检测、隔离测试）（Final Implementation Commit） |

## 与计划偏差

1. 计划 Task 8 只有 5 步（Session 组件 / Permission 弹窗 / 危险命令 / 集成 / Commit），且 Step 2 文案写 `[Y]Allow/[R]Remember/[N]Deny`；按五项设计原则「UI 不伪造 vendor remember 标志」及用户提供的详细 spec，将审批映射收敛为 `once/always/reject` 三态（`[Y] Allow once`/`[A] Always allow`/`[N] Reject`），`Remember` 由 Adapter 层根据 Vendor 能力处理，UI 不暴露。
2. 计划仅列 Modify `model.go`/`update.go`；实际还需扩展 Runtime Contract（`ListSessions`）、OpenCodeAdapter、event_mapper、fake-runtime，以及新建 `session_mapper.go`，故 Files Changed 超出计划列表（计划 File 列表为最小集合）。
3. 计划 `[R]Remember` 对应 UI 三键 Y/R/N；实际为 Y/A/N，`A`（always）复用 `a`/`r` 双键兼容。

## Gate 结论

- verification：pass
- Task Gate：pass
- 进入 `awaiting_acceptance`，等待人工验收；验收前不启动 Task 9。
