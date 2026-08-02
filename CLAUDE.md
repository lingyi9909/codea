# Codea V1

企业内网 AI 编码助手，基于 OpenCode Runtime + Go TUI 的 C+ 混合架构。

## 项目概述

- **目标**：为内网开发环境提供 AI 编码助手，支持代码审查、单元测试生成、API 文档生成
- **架构**：C+ 混合模式 — OpenCode（anomalyco/opencode）作为独立 Agent Runtime，Go TUI（Bubble Tea + Lip Gloss）通过 HTTP/SSE 通信
- **平台**：V1 支持 macOS arm64/x64 + Windows x64；Linux 延迟
- **开发模型**：AI 辅助开发，团队无 TypeScript/Go 专业人员

## 关键仓库

- OpenCode 官方：`https://github.com/anomalyco/opencode`（非 anthropics/opencode）

## 设计文档

- **项目交接**：`docs/codea-v1-handoff.md`
- **技术设计**：`docs/superpowers/specs/2026-07-30-codea-v1-design.md`
- **实施计划**：`docs/superpowers/plans/2026-07-30-codea-v1-plan.md`
- **执行状态设计**：`docs/superpowers/specs/2026-08-01-codea-execution-state-design.md`
- **执行状态计划**：`docs/superpowers/plans/2026-08-01-codea-execution-state-plan.md`

## 五项设计原则

1. **OpenCode Core 最小侵入** — Patch 不超过 5 个文件，每个 Patch 有说明和测试
2. **Go TUI 不承担 Agent 逻辑** — TUI 只负责交互、流式展示、进程管理、Tool 审批；Agent Loop/Session/上下文管理由 OpenCode Runtime 负责
3. **企业能力使用原生扩展** — Reviewer/UT/API Doc 通过 Agent + Skill + Tool 组合实现
4. **RuntimeClient 隔离协议** — Go TUI 不直接依赖 OpenCode API 数据结构
5. **OpenCode 原生能力不退化** — General 模式完整保留 OpenCode 能力，事件零静默丢弃

## 双模式架构

- **Native-Compatible Mode（General Agent）**：保留 OpenCode 完整能力
- **Enterprise-Controlled Mode（3 个企业 Agent）**：Code Reviewer / Unit Test Generator / API Documentation Generator，专用 Tool + 严格路径限制

## 开发流程

### 执行顺序

先执行 Task E0 建立执行状态机制；E0 经人工验收后，再按主实施计划中的 Task 0 → Task 21 逐个执行：

| 阶段 | Tasks | 内容 |
|------|-------|------|
| Bootstrap | Task E0 | 执行状态、校验器、中断恢复与人工验收协议 |
| Skeleton | Task 0 | 项目骨架与 Go Module 结构 |
| Phase 0 | Task 1 | Spike S1-S6 验证 |
| Phase 1 | Task 2-3 | OpenAPI 固化 + 能力盘点 + Parity Harness |
| Phase 2 | Task 4-5 | RuntimeClient + Supervisor |
| Phase 3 | Task 6-9 | Reasoning + TUI + Session + General 对齐 |
| Phase 4 | Task 10-11 | Skill/Plugin Manager + 模式隔离 |
| Phase 5 | Task 12-13 | 安全/DLP/Dify + Enterprise Custom Tools |
| Phase 6 | Task 14-16 | 三个企业 Agent |
| Phase 7 | Task 17-18 | 离线发行包 + 升级回滚 |
| Phase 8 | Task 19-20 | Doctor + 试点统计 |
| Phase 9 | Task 21 | Release Parity Certification |

当前阶段以 `docs/codea-v1-handoff.md` 和 `docs/execution-state.yaml` 为准。状态机制初始化前，唯一允许执行的是 Task E0；不得直接开始 Task 0。

### 执行状态恢复（最高优先级）

开始或恢复任何 Task 前：

1. 阅读 `docs/codea-v1-handoff.md`、技术设计、主实施计划、执行状态设计和执行状态计划。
2. 如果 `docs/execution-state.yaml` 不存在，只允许按执行状态计划完成 Task E0；不得运行尚不存在的校验器，不得开始 Task 0。
3. 如果 `docs/execution-state.yaml` 已存在，先运行：

   ```bash
   git status
   ./scripts/check-execution-state.sh
   ```

4. 校验通过后，只执行 `current.task`、`current.step` 和 `nextAction` 指定的工作。
5. 状态为 `blocked` 时先处理阻塞；状态为 `awaiting_acceptance` 时立即停止并等待人工验收。
6. 状态文件、Git Commit、Task 报告或工作区互相矛盾时停止报告，不得猜测、覆盖或跳过。

初始化 Task E0 时，checkpoint 必须通过 `git rev-parse HEAD` 获取 E0 修改前的真实完整 Commit，并用 `git cat-file` 验证；不得从文档复制固定 SHA。

### 每个 Task 的工作流

以下流程适用于 Task 0～Task 21；Task E0 按执行状态计划单独完成，并在人工确认后结束 Bootstrap 阶段。

1. 读取并校验执行状态，确认当前 Task、Step 和 checkpoint
2. 阅读计划中对应 Task 的完整内容
3. 开始 Step 前将状态更新为 `in_progress`
4. 编写代码（TDD：先测试，再实现）
5. 运行当前 Task 要求的全部构建、测试和校验命令
6. 更新状态并生成 `docs/task-reports/task-XX.md`
7. 自动验证及 Task Gate 通过后进入 `awaiting_acceptance`，提交并停止
8. 只有用户明确验收后才能标记 `completed` 并开始下一 Task

### Git 提交规范

```
feat: <task description>
```

每个 Task 至少一个 commit，不跨 Task 混合提交。

后续修改直接提交到 `develop`，不新建功能分支；禁止强制推送。每个 Task 和验收状态变更仍须使用独立、可追溯的 Commit。

## 技术栈

- **TUI**：Go 1.22+ / Bubble Tea / Lip Gloss / Glamour
- **Runtime**：OpenCode（锁定版本，`opencode serve`）
- **Plugin**：TypeScript（ESM，target bun，自包含 Bundle）
- **企业 Tool**：TypeScript Plugin（7 个专用 Tool）
- **模型**：开发用 DeepSeek，内网用私有模型（OpenAI-compatible API）

## 关键约束

- `go.mod` 在 `tui/` 目录下，所有 Go 代码（含测试）在 `tui/` 内
- 运行命令统一从 `tui/` 执行：`cd tui && go test ./...`
- OpenCode API/DTO 以锁定版本 `/doc` OpenAPI 3.1 为准，从 spec 生成
- OpenCode 启动认证通过 `OPENCODE_SERVER_PASSWORD` + `OPENCODE_SERVER_USERNAME` 环境变量
- OpenCode 配置目录通过 `OPENCODE_CONFIG_DIR` 环境变量指定（非 `--config-dir` 参数）
- Plugin Bundle 自包含，发行包中不包含 `package.json`、`bun.lock`、`node_modules`
- API Key 通过环境变量引用，不写入配置文件
- TUI 最低终端 70×20

## Phase 0 门禁

Spike S1-S6 全部通过才能进入 Phase 1。结果记录在 `docs/spike-results.json`：

```json
{"S1": "pass", "S2": "pass", "S3": "pass", "S4": "pass", "S5": "pass", "S6": "pass"}
```

运行 `scripts/run-phase0-gates.sh` 验证。

## 关键接口

```go
type RuntimeClient interface {
    Health(ctx context.Context) (HealthInfo, error)
    CreateSession(ctx context.Context, request CreateSessionRequest) (Session, error)
    SendPromptAsync(ctx context.Context, sessionID string, req PromptRequest) error
    Subscribe(ctx context.Context) (<-chan RuntimeEvent, error)
    ApprovePermission(ctx context.Context, sessionID string, permissionID string, decision PermissionDecision) error
    AbortSession(ctx context.Context, sessionID string) error
    ListAgents(ctx context.Context) ([]Agent, error)
}
```

## 项目结构

```
codea/
├── tui/                    # Go Module（所有 Go 代码）
│   ├── cmd/                # 入口程序
│   ├── internal/           # 内部包
│   └── tests/              # Go 测试
├── distribution/           # 企业发行层（Agent/Skill/Plugin/Config）
├── runtime/                # OpenCode 版本锁定 + OpenAPI Spec
├── packaging/              # 离线包构建脚本
├── tests/                  # Shell 集成测试（离线/升级）
├── scripts/                # 开发辅助脚本
├── devtools/               # 开发工具
└── docs/                   # 设计文档和实施计划
```
