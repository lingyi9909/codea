# Codea V1 项目交接说明

## 1. 当前状态

- 项目：Codea V1
- 工作分支：`develop`，后续直接在该分支按顺序修改，不另建功能分支
- 技术设计与主实施计划：已评审通过
- 执行状态机制设计与实施计划：已评审通过
- 当前执行阶段：Task E0、Task 0 已人工验收通过；Task 1 S1～S6 与机器门禁已通过，当前等待人工验收
- 唯一下一步：人工验收 Task 1
- 边界：Task 1 的 S1～S6 全部通过并进入 `awaiting_acceptance` 后停止，人工验收前不得开始 Task 2
- 已知非阻塞改进：项目全部完成时补充校验终态 `current.task = 21` 的一致性
- Go 基线：项目统一使用 Go 1.26.5

当前仓库已包含 `docs/execution-state.yaml` 及其校验器。Task 1 的六项 Spike、OpenAPI、Golden SSE、机器结果和 Phase 0 Gate 均已持久化，状态为 `awaiting_acceptance`。

## 2. 权威文档与读取顺序

接手者开始工作前必须完整阅读：

1. `AGENTS.md`
2. `CLAUDE.md`
3. `docs/superpowers/specs/2026-07-30-codea-v1-design.md`
4. `docs/superpowers/plans/2026-07-30-codea-v1-plan.md`
5. `docs/superpowers/specs/2026-08-01-codea-execution-state-design.md`
6. `docs/superpowers/plans/2026-08-01-codea-execution-state-plan.md`

文档职责：

- Codea 技术设计定义产品架构与不可违反的技术原则。
- Codea 主实施计划定义 Task 0～Task 21 的内容、顺序和验收标准。
- 执行状态设计定义状态含义、合法流转和中断恢复规则。
- 执行状态实施计划定义 Task E0 的具体步骤。
- Task E0 完成后，`docs/execution-state.yaml` 是执行位置的唯一机器可读状态源。

发现文档、Git Commit、状态文件或工作区互相矛盾时，必须停止并报告，不得猜测进度或自行修改架构绕过问题。

## 3. 中断恢复协议

### Task E0 完成前

如果 `docs/execution-state.yaml` 不存在，只允许执行 Task E0：

1. 确认当前位于 `develop`，工作区没有来源不明的修改。
2. 在修改任何 E0 文件前，用 `git rev-parse HEAD` 记录真实基线 Commit。
3. 严格执行 `docs/superpowers/plans/2026-08-01-codea-execution-state-plan.md`。
4. E0 自动验证通过后停止并等待人工验收，不得继续 Task 0。

不得从交接文档复制固定 SHA 作为 checkpoint。初始化 checkpoint 必须是执行 E0 前仓库中真实存在的完整 40 位 HEAD Commit。

### Task E0 完成后

每次开始或恢复工作时执行：

```bash
git status
./scripts/check-execution-state.sh
```

然后按 `docs/execution-state.yaml` 的 `current`、`checkpoint`、`verification`、`taskGate`、`humanAcceptance` 和 `nextAction` 恢复。状态为 `blocked` 时先处理阻塞；状态为 `awaiting_acceptance` 时停止开发并等待人工验收。

## 4. 状态含义

Task 状态：

- `pending`：尚未开始。
- `in_progress`：当前正在执行。
- `blocked`：存在阻塞，不能继续执行。
- `awaiting_acceptance`：实现、自动验证和 Task Gate 已完成，等待人工验收。
- `completed`：人工验收通过，Task 正式结束。

验证状态：

- `not_run`：尚未执行。
- `pass`：要求的验证命令全部通过。
- `fail`：验证已经执行但失败。
- `unable_to_run`：因环境、权限或依赖问题无法执行验证，此时 Task 必须为 `blocked`。

Task Gate 状态：

- `not_evaluated`：尚未按完整验收标准判断。
- `pass`：必需交付物、验证结果和专项门禁全部满足。
- `fail`：已经判断，但至少一项验收标准不满足。
- `unable_to_evaluate`：因缺少证据、环境或依赖无法判断，此时 Task 必须为 `blocked`。

三层判定不得混用：`verification` 记录命令结果，`taskGate` 记录计划验收标准的整体判断，`humanAcceptance` 记录用户明确验收。只有三者均通过，Task 才能进入 `completed`。

## 5. 项目目标与核心架构

基于 OpenCode Runtime + Go TUI 构建企业内网 AI 编码助手，支持：

- Native-Compatible Mode：General Agent，保留 OpenCode 原生能力。
- Enterprise-Controlled Mode：Code Reviewer、Unit Test Generator、API Documentation Generator。
- 私有模型接入与 Dify 企业知识查询。
- 离线安装、升级、回滚与 Doctor。
- 能力不退化的 Parity 验证。

核心架构：

- OpenCode 作为独立 Agent Runtime，通过 `opencode serve` 启动。
- Go TUI 通过 HTTP/OpenAPI + SSE 与 Runtime 通信。
- `RuntimeClient + OpenCodeAdapter` 隔离协议变化。
- Go TUI 不承担 Agent Loop、上下文管理、Tool 选择或 Subagent 调度。
- 企业能力通过 Agent + Skill + 专用 Tool 实现。
- OpenCode Core 保持最小侵入，原则上修改不超过 5 个文件。

## 6. 不可违反的约束

1. 每次只执行一个 Task；当前 Task 人工验收后才能进入下一 Task。
2. General 模式不得裁剪 OpenCode 原生能力。
3. 所有 SSE 事件必须映射或 Raw 透传，静默丢失为 0。
4. OpenCode API/DTO 必须以锁定版本 `/doc` OpenAPI 3.1 为准，不得手写猜测字段。
5. 企业 Agent 不得使用通用 Bash/Edit：Reviewer 只读；UT 使用 `write_test_file`、`run_project_test`；API Doc 使用 `write_document`。
6. 不得使用 `t.Skip`、空实现、伪造结果、删除测试或降低断言绕过 Required Gate。
7. 未真实验证的 Spike 必须标记为 `fail` 或 `blocked`，不得写成 `pass`。
8. Plugin 必须为自包含 ESM Bundle，内网启动不得触发在线安装。
9. API Key 不得写入普通配置或日志。
10. 状态文件、Task 报告、验证证据和实际 Git Commit 必须一致。

## 7. 当前执行点：Task 1 等待人工验收

Task E0 和 Task 0 已人工验收通过。Task 1 已锁定 OpenCode v1.18.11（commit `012c2f57f976489d88bd4598a056b4bdcdd428ee`），官方 Linux x64 和 macOS arm64 制品 SHA-256 校验一致。

S1（Server 离线启动）已通过。在 macOS arm64 上使用正确的 `OPENCODE_DISABLE_MODELS_FETCH=1` 环境变量完成真实断网验证（关闭全部外部接口 + 全接口 tcpdump + 隔离沙箱）。内部日志仅 3 行 INFO，零 ERROR，零 `models.opencode.ai` 请求；全接口抓包无 OpenCode 相关出站流量。详细证据见 `docs/spike-report.md` 和 `docs/spike-artifacts/s1-20260803-175535/`。

S2～S6 均已通过：Session/SSE、Tool Approval 批准与拒绝、结构化 Reasoning、Skill 来源隔离及三模式隔离均有真实 Runtime 证据。完整结果见 `docs/spike-report.md`、`docs/spike-results.json` 和 `docs/spike-artifacts/`。

`scripts/run-phase0-gates.sh` 已返回全 PASS；锁定版本 OpenAPI 和 Golden SSE 已固化。当前必须停止开发并等待人工验收，验收前不得开始 Task 2。

## 8. E0 验收后的 Task 0

只有 E0 经人工明确验收后，才能执行主实施计划中的 Task 0：项目骨架与 Go Module 结构。

Task 0 的核心范围：

- 创建计划规定的目录结构。
- 初始化 `tui/go.mod`。
- 创建 `tui/cmd/codea/main.go`、`tui/cmd/parity-runner/main.go`。
- 创建 Makefile、VERSION、基础配置。
- 创建 `runtime/version.json`、`runtime/capabilities.yaml`。
- 创建 Phase 0 门禁脚本位置。
- 更新执行状态并生成 `docs/task-reports/task-00.md`。

Task 0 完成后必须运行主计划及状态机制要求的全部验证，状态进入 `awaiting_acceptance` 后立即停止，不得自动进入 Task 1。

## 9. Phase 0 门禁

Task 0 验收后，Task 1 执行 S1～S6：

- S1：OpenCode Server 断网启动。
- S2：Session → Prompt → SSE 完整链路。
- S3：Tool Approval 批准与拒绝。
- S4：Reasoning 与 Answer 分离。
- S5：Skill 来源隔离。
- S6：General compatible / General strict / Enterprise 模式隔离。

必须产出真实的 Spike 报告、结构化结果、锁定版本 OpenAPI Spec、Golden SSE 样本、版本与制品哈希，以及 `scripts/run-phase0-gates.sh` 的真实执行结果。只有 S1～S6 全部为 `pass` 且 Task 1 经人工验收，才能进入 Task 2。

## 10. 每个 Task 的交付模板

### Task

- Task 编号与名称

### 完成内容

- 实际实现内容

### 文件变更

- 新增、修改和删除的文件

### 执行命令

- 构建、测试和验证命令

### 验证结果

- `PASS`、`FAIL` 或 `UNABLE_TO_RUN`
- 关键输出摘要与证据

### 与计划偏差

- 无，或说明偏差、原因和影响

### 未解决问题

- 阻塞项、风险项和恢复建议

### Gate 结论

- verification 结论
- Task Gate 结论
- 是否进入 `awaiting_acceptance`
- 人工验收前是否保持停止

## 11. 当前交接结论

Task E0 和 Task 0 已人工验收通过；Go 基线统一为 1.26.5。Task 1 的 S1～S6、自动验证和 Task Gate 已通过，当前为 `awaiting_acceptance`。人工验收前不得开始 Task 2。
