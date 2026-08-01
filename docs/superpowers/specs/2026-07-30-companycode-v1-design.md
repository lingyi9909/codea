# CompanyCode V1 — 技术设计文档

产品范围：CompanyCode V1
文档版本：2.0
日期：2026-07-30
状态：设计评审通过，允许进入 Phase 0；关键能力以 Spike 和 Parity 门禁结果为准

---

## 目录

1. [总体架构](#1-总体架构) — 选型、五项原则、双模式架构、能力基线
2. [项目结构与模块划分](#2-项目结构与模块划分)
3. [通信协议与事件模型](#3-通信协议与事件模型) — 生命周期、HTTP/SSE、Raw 透传
4. [TUI 视觉与交互设计](#4-tui-视觉与交互设计)
5. [Skill 动态开关机制](#5-skill-动态开关机制) — 三态、双轨策略、Plugin 兼容
6. [私有模型接入、离线发行包、升级回滚](#6-私有模型接入离线发行包升级回滚)
7. [企业能力](#7-企业能力) — Reviewer、UT Gen、API Doc、逃生机制
8. [安全控制、Dify、安装与 Doctor、试点统计](#8-安全控制dify安装与-doctor试点统计)
9. [附录](#附录-a开发顺序) — 开发顺序、验收标准、关键术语

---

## 1. 总体架构

### 1.1 架构选型：C+ 混合模式

OpenCode 作为独立 Agent Runtime，本地启动 Server；CompanyCode 使用 Go 开发独立 TUI，通过 HTTP/SSE 与 Runtime 通信；企业能力通过 Agent、Skill、Plugin 和配置扩展。

```
┌─────────────────────────────────────────────────────────┐
│                    CompanyCode V1                       │
│                                                         │
│  ┌───────────────────────────────────────────────────┐  │
│  │ Go TUI：company-code                              │  │
│  │                                                   │  │
│  │ Bubble Tea + Lip Gloss + Glamour                  │  │
│  │                                                   │  │
│  │ • 流式对话       • Reasoning 展示                 │  │
│  │ • Tool 审批      • Skill/Profile 管理             │  │
│  │ • Runtime 管理   • 升级回滚                       │  │
│  │ • Doctor 诊断    • 配置管理                       │  │
│  └──────────────────────┬────────────────────────────┘  │
│                         │                               │
│              RuntimeClient 抽象层                       │
│                         │                               │
│       HTTP/OpenAPI + SSE + Local Authentication         │
│                         │                               │
│  ┌──────────────────────▼────────────────────────────┐  │
│  │ OpenCode Runtime                                  │  │
│  │ opencode serve --hostname 127.0.0.1               │  │
│  │                                                   │  │
│  │ Session / Agent Loop / Provider / Tool            │  │
│  │ Agent / Skill / Plugin / Permission / AGENTS.md   │  │
│  └───────────────┬──────────────────┬────────────────┘  │
│                  │                  │                   │
│        OpenAI-Compatible API     企业扩展 Tool           │
│                  │                  │                   │
│         ┌────────▼────────┐   ┌─────▼────────────┐      │
│         │ 模型 Provider    │   │ Dify 企业知识库   │      │
│         │                 │   │ 可选、失败降级    │      │
│         │ 开发：DeepSeek   │   └──────────────────┘      │
│         │ 内网：私有模型   │                             │
│         └─────────────────┘                             │
│                                                         │
│  Distribution                                           │
│  ├── agents/       企业 Agent                            │
│  ├── skills/       预装 Skill                            │
│  ├── plugins/      Dify、安全、审计扩展                  │
│  ├── config/       模型、权限、Profile                   │
│  └── manifests/    版本、依赖、哈希                      │
└─────────────────────────────────────────────────────────┘
```

### 1.2 五项设计原则

**原则 1：OpenCode Core 最小侵入**

OpenCode 作为独立 Agent Runtime 使用。优先通过配置、Agent、Skill、Plugin 和公开 Server API 实现企业能力；不裁剪、不大范围修改 Core。必要时仅维护少量、可追踪、可回归验证的 Patch。

OpenCode Core Patch 约束：
- 修改文件数原则上不超过 5 个
- 自定义修改必须保存为独立 Patch
- 每个 Patch 必须说明原因和对应测试
- 禁止大范围删除或裁剪上游模块

**原则 2：Go TUI 不承担 Agent 核心逻辑**

Go TUI 负责用户交互、流式展示、Runtime Supervisor、Tool 审批、配置管理、升级回滚和诊断。Session、Agent Loop、Tool 调度和上下文管理由 OpenCode Runtime 负责。

Go TUI 职责：
1. Runtime 进程生命周期（启动、停止、重启、状态监控）
2. OpenCode API 访问
3. SSE 事件接收和转换
4. 用户交互及界面渲染
5. Tool 审批
6. 本地安装、升级、诊断

Go TUI 不负责：Agent Loop、消息历史管理、Tool 选择决策、上下文压缩、Subagent 调度、Skill Prompt 执行逻辑、Provider 协议实现。

**原则 3：企业能力使用原生扩展机制**

Code Reviewer、Unit Test Generator 和 API Documentation Generator 使用 Agent、Skill 和 Tool 组合实现；Dify、安全和审计能力使用 Plugin/Tool 实现，不侵入 OpenCode 核心流程。Agent、Skill、Tool 的安全扩展优先使用 OpenCode 原生机制；Go TUI 只负责配置和状态管理，不直接承载企业 Agent 执行逻辑。

**原则 4：通过 RuntimeClient 隔离协议**

Go TUI 不直接依赖 OpenCode API 数据结构。通过 `RuntimeClient + OpenCodeAdapter` 将 OpenCode HTTP/SSE 事件转换为 CompanyCode 内部统一模型，降低上游版本变化影响。

```
Go TUI
   ↓
CompanyCode RuntimeClient
   ↓
OpenCodeAdapter
   ↓
OpenCode OpenAPI/SSE
```

**原则 5：OpenCode 原生能力不退化**

CompanyCode 不以裁剪 OpenCode 通用能力为目标。General 模式应完整保留锁定版本 OpenCode 的 Agent Loop、Session、Provider、原生 Tool、文件操作、Shell、Agent/Subagent、Skill 和 Plugin 扩展能力。CompanyCode 新增的适配层、安全层和 Go TUI 不得静默丢弃或阻断原生能力。

企业 Agent 可以基于业务安全要求使用更严格的 Tool 和路径权限，但这些限制只作用于对应企业 Agent，不改变 General 模式的能力边界。

CompanyCode 不得在架构和功能入口层削弱锁定版本 OpenCode 的原生能力。核心功能可达率、原生 Tool 可调用率、事件映射或透传率和原生 API 能力无静默丢失均必须达到 100%。模型生成结果具有随机性，同模型/同 Runtime/同权限条件下，任务效果通过 Parity 测试控制在统计容差内（不低于原版的 95%，且任何核心任务不得失败）。每次升级均通过能力基线回归测试验证。

### 1.3 双模式架构

CompanyCode 提供两种能力模式，确保「安全可控」与「能力完整」不互相侵蚀：

```
┌─────────────────────────────────────────┐
│         CompanyCode V1                  │
│                                         │
│  ┌───────────────┐  ┌────────────────┐  │
│  │ Native-       │  │ Enterprise-    │  │
│  │ Compatible    │  │ Controlled     │  │
│  │ Mode          │  │ Mode           │  │
│  │               │  │                │  │
│  │ General Agent │  │ Code Reviewer  │  │
│  │               │  │ Unit Test      │  │
│  │               │  │ API Doc        │  │
│  └───────────────┘  └────────────────┘  │
│                                         │
│  保留 OpenCode 广度    增强企业场景稳定性  │
└─────────────────────────────────────────┘
```

**Native-Compatible Mode（General Agent）：**

目标：保留 OpenCode 原生能力，允许用户在内网获得与公网原版尽可能等价的体验。

- OpenCode 原生 Tool 全部保留（read/grep/glob/write/edit/bash/Agent/Subagent/Skill/Plugin）
- 不通过专用 Tool 替换原生 Tool
- 不删除 Runtime 原生能力
- 危险操作通过 Permission（Allow/Ask/Deny）控制，不由 CompanyCode 单方面删除
- 只有明确高危命令直接 Deny
- 未知但非明确危险的操作进入 Ask，不直接 Deny
- 批准的内网 Maven/npm/PyPI/Go Proxy 镜像可正常使用

**Enterprise-Controlled Mode（三个企业 Agent）：**

目标：在三个特定企业场景中，通过专用 Tool、严格路径限制和固定输出结构保证结果质量。

- 专用 Tool 替代通用 Bash/Edit
- 严格路径限制
- 固定结构化输出
- 企业规范引用
- 可审计

**最终定位：**

```
General Agent：保持 OpenCode 的广度与灵活
Enterprise Agent：增强三个企业场景的稳定性与合规
```

这两种模式不互相侵蚀：企业 Agent 的安全限制不反向作用于 General Agent；General Agent 的能力保留不削弱企业 Agent 的安全约束。

### 1.4 能力基线（Capability Inventory）

每次锁定 OpenCode 新版本时必须盘点能力清单，确保不静默丢失：

`runtime/capabilities.yaml`：

```yaml
schemaVersion: 1
openCodeVersion: x.y.z

capabilities:
  sessions: required
  streaming: required
  reasoning: required
  fileRead: required
  fileWrite: required
  edit: required
  bash: required
  toolApproval: required
  agents: required
  subagents: required
  skills: required
  plugins: required
  abort: required
  messageHistory: required
  contextCompaction: required

tui:
  sessionList: required
  sessionResume: required
  toolApproval: required
  rawEventFallback: required
```

升级流程：导出上游能力 → 与当前 baseline 对比 → 发现新能力 → 决定映射/透传/明确暂不支持 → 不能静默忽略。

---

## 2. 项目结构与模块划分

### 2.1 仓库结构

```
company-code/
├── tui/                              # Go TUI（独立 Go Module）
│   ├── cmd/company-code/
│   │   └── main.go                   # 入口
│   ├── internal/
│   │   ├── app/                      # Bubble Tea 顶层 Model
│   │   │   ├── model.go              # 主 Model 定义
│   │   │   ├── update.go             # Update 消息循环
│   │   │   ├── view.go               # View 渲染
│   │   │   ├── messages.go           # 自定义消息类型
│   │   │   ├── commands.go           # Bubble Tea 命令
│   │   │   ├── keymap.go             # 快捷键定义
│   │   │   └── page.go               # 页面状态枚举
│   │   ├── runtime/                  # CompanyCode 领域接口与模型
│   │   │   ├── client.go             # RuntimeClient 接口
│   │   │   ├── events.go             # CompanyCode 统一事件
│   │   │   └── models.go             # 领域模型
│   │   ├── opencode/                 # OpenCode 适配层
│   │   │   ├── adapter.go            # RuntimeClient 的 OpenCode 实现
│   │   │   ├── http_client.go        # 普通 HTTP API
│   │   │   ├── sse_client.go         # SSE 连接、重连和取消
│   │   │   ├── event_mapper.go       # OpenCode 事件 → 内部事件
│   │   │   └── dto.go               # OpenCode 原始请求响应结构
│   │   ├── supervisor/               # Runtime 进程管理
│   │   │   ├── supervisor.go         # 生命周期管理
│   │   │   └── process.go            # 平台相关进程控制
│   │   ├── reasoning/                # 推理内容处理
│   │   │   ├── normalizer.go         # 结构化 Part 转换
│   │   │   ├── tag_parser.go         # <think> 标签状态机（兜底）
│   │   │   └── tracker.go            # 推理状态跟踪
│   │   ├── components/               # Bubble Tea 子组件
│   │   │   ├── chat.go               # 对话列表
│   │   │   ├── input.go              # 多行输入
│   │   │   ├── status.go             # 顶部状态栏
│   │   │   ├── tool.go               # Tool 状态与确认
│   │   │   ├── skill.go              # Skill 开关管理
│   │   │   ├── session.go            # Session 列表/恢复
│   │   │   ├── agent.go              # Agent 选择器
│   │   │   ├── model.go              # 模型/Provider 状态查看
│   │   │   ├── event_viewer.go       # 原始事件查看器（Debug）
│   │   │   └── permission.go         # 权限确认弹窗
│   │   ├── theme/                    # 主题与样式
│   │   │   └── theme.go
│   │   ├── config/                   # 配置管理
│   │   │   ├── config.go             # 加载/合并配置
│   │   │   ├── merge.go              # 配置合并算法
│   │   │   └── profile.go            # Profile 解析
│   │   ├── update/                   # 升级与回滚（服务层）
│   │   │   ├── service.go            # UpdateService 接口
│   │   │   ├── package.go            # 包校验与解压
│   │   │   ├── checksum.go           # 哈希校验
│   │   │   ├── versions.go           # 版本目录管理
│   │   │   ├── rollback.go           # 回滚逻辑
│   │   │   └── platform_unix.go      # 平台差异
│   │   └── doctor/                   # 健康检查（服务层）
│   │       ├── service.go            # DoctorService 接口
│   │       ├── checks.go             # 各项检查实现
│   │       └── report.go             # 检查报告
│   │   ├── capability/               # 能力盘点（原则 5）
│   │   │   ├── inventory.go          # 能力清单加载
│   │   │   ├── compare.go            # 版本能力对比
│   │   │   └── report.go             # 能力报告
│   │   └── parity/                   # 能力不退化验证
│   │       ├── runner.go             # Parity 测试运行器
│   │       ├── scenario.go           # 测试场景定义
│   │       └── result.go             # 结果判定
│   ├── go.mod
│   └── go.sum
│
├── distribution/                     # 企业发行层
│   ├── agents/
│   │   ├── code-reviewer/
│   │   │   ├── agent.md
│   │   │   ├── manifest.yaml
│   │   │   ├── output-schema.json
│   │   │   └── fixtures/
│   │   ├── unit-test-generator/
│   │   │   ├── agent.md
│   │   │   ├── manifest.yaml
│   │   │   └── error-categories.yaml
│   │   └── api-documentation/
│   │       ├── agent.md
│   │       ├── manifest.yaml
│   │       └── output-template.md
│   ├── skills/
│   │   ├── index.yaml                # 构建时生成的总索引
│   │   ├── builtin/                  # 开源预装 Skill
│   │   │   ├── code-explain/
│   │   │   │   ├── SKILL.md
│   │   │   │   ├── companycode.yaml
│   │   │   │   ├── manifest.yaml
│   │   │   │   └── LICENSE
│   │   │   ├── git-helper/
│   │   │   └── log-analyzer/
│   │   └── enterprise/               # 企业自研 Skill
│   │       ├── code-review/
│   │       │   ├── SKILL.md
│   │       │   └── companycode.yaml
│   │       ├── unit-test/
│   │       └── api-documentation/
│   ├── plugins/
│   │   ├── src/                     # 开发源码（含 package.json）
│   │   │   ├── dify-query.ts
│   │   │   ├── runtime-security-guard.ts
│   │   │   └── audit-log.ts
│   │   ├── dist/                    # 自包含 Bundle（发行包使用）
│   │   │   ├── dify-query.js
│   │   │   ├── runtime-security-guard.js
│   │   │   └── audit-log.js
│   │   ├── package.json             # 仅开发构建使用
│   │   └── bun.lock                 # 仅开发构建使用
│   ├── config/
│   │   ├── companycode/
│   │   │   ├── defaults.yaml
│   │   │   ├── skills.yaml
│   │   │   └── profiles/
│   │   │       ├── minimal.yaml
│   │   │       ├── java-backend.yaml
│   │   │       ├── go-backend.yaml
│   │   │       └── python-backend.yaml
│   │   └── opencode/
│   │       ├── opencode.json.tmpl
│   │       ├── permissions.json
│   │       └── model.json.tmpl
│   └── templates/
│       └── AGENTS.md.tmpl
│
├── runtime/                          # OpenCode Runtime 元信息
│   ├── version.json                  # 锁定的 OpenCode 版本/Commit/哈希
│   ├── checksums.json                # 平台制品哈希
│   ├── capabilities.yaml             # 锁定版本能力清单（原则 5）
│   └── patches/                      # 极小 Patch（必要时）
│       └── README.md
│
├── packaging/                        # 离线发行包构建
│   ├── config/
│   │   └── release.yaml
│   ├── scripts/
│   │   ├── build-runtime.sh
│   │   ├── build-plugins.sh
│   │   ├── collect-skills.sh
│   │   ├── generate-manifest.sh
│   │   ├── verify-checksum.sh
│   │   └── verify-offline.sh
│   └── platform/
│       ├── linux/
│       │   └── install.sh
│       ├── macos/
│       │   └── install.sh
│       └── windows/
│           └── install.ps1
│
├── tests/                            # 测试
│   ├── contract/                     # API 契约测试
│   │   ├── server_health_test.go
│   │   ├── session_test.go
│   │   ├── stream_events_test.go
│   │   ├── tool_approval_test.go
│   │   └── reasoning_event_test.go
│   ├── offline/                      # 断网集成测试
│   │   ├── no_public_network_test.sh
│   │   ├── install_test.sh
│   │   └── runtime_start_test.sh
│   ├── e2e/                          # 端到端测试
│   │   ├── chat/
│   │   ├── code-review/
│   │   ├── unit-test/
│   │   └── api-documentation/
│   ├── upgrade/                      # 升级回滚测试
│   │   ├── fresh_install_test.sh
│   │   ├── upgrade_test.sh
│   │   ├── failed_upgrade_test.sh
│   │   └── rollback_test.sh
│   ├── parity/                        # 能力不退化回归测试
│   │   ├── capability_inventory_test.go
│   │   ├── event_passthrough_test.go
│   │   ├── general_agent_test.go
│   │   ├── native_tools_test.go
│   │   ├── session_resume_test.go
│   │   ├── subagent_test.go
│   │   └── fixtures/
│   └── fixtures/
│       ├── java-maven-project/
│       ├── java-gradle-project/
│       ├── go-project/
│       └── fake-opencode-server/     # 模拟 Runtime，用于 TUI 开发
│
├── devtools/                         # 开发辅助工具
│   ├── openapi-gen/
│   ├── manifest-gen/
│   ├── skill-lint/
│   ├── license-report/
│   ├── sse-recorder/
│   └── parity-runner/              # 能力不退化回归工具
│
├── docs/
│   ├── architecture.md
│   ├── development.md
│   └── specs/
│
├── Makefile
├── VERSION
├── CHANGELOG.md
├── SECURITY.md
├── THIRD_PARTY_LICENSES.md
├── README.md
├── .gitignore
└── .editorconfig
```

### 2.2 关键设计点

1. **tui/** — Go 模块，独立 `go.mod`。不依赖 distribution 以外的 CompanyCode 代码。
2. **distribution/** — 保存所有企业扩展资源（Agent、Skill、Plugin、配置、模板）。不包含 OpenCode Core 源码。Plugin 需要 TypeScript 构建和运行时依赖，内置依赖随发行包交付，不在内网执行 `bun install`。
3. **runtime/** — 只锁版本号和 Patch，不含 OpenCode 源码。
4. **packaging/** — 构建脚本负责下载 OpenCode、编译 TUI、组装发行包。通用逻辑只保留一份，平台目录仅放真正有差异的内容。
5. **Skill 目录** — 每个 Skill 独立保存 SKILL.md、companycode.yaml、manifest.yaml 和 LICENSE，不依赖单一全局 YAML。开源 Skill 和企业 Skill 分目录管理。
6. **Agent 目录** — 每个企业 Agent 为独立目录（agent.md + manifest.yaml + 输出约束），不采用单文件。
7. **配置分层** — CompanyCode 配置与 OpenCode 配置明确分离。运行前由 CompanyCode 生成最终 OpenCode 配置目录。
8. **测试 Fixtures** — 包含 fake-opencode-server 用于模拟 SSE/Reasoning/Tool Approval，TUI 开发时不依赖真实模型。

### 2.3 配置目录结构

```
distribution/config/
├── companycode/
│   ├── defaults.yaml
│   ├── skills.yaml
│   └── profiles/
│       ├── minimal.yaml
│       ├── java-backend.yaml
│       ├── go-backend.yaml
│       └── python-backend.yaml
│
└── opencode/
    ├── opencode.json.tmpl
    ├── permissions.json
    └── model.json.tmpl
```

运行时生成：

```
~/.companycode/runtime/<project-hash>/
└── config/
    ├── opencode.json
    ├── agents/
    ├── plugins/
    └── skills/         # 仅包含 Enabled Skill
```

---

## 3. 通信协议与事件模型

### 3.1 Runtime 进程生命周期

项目根目录识别优先级：

1. 显式 `--project /workspace/order-service` 参数
2. 从当前目录向上查找 `.git`
3. 当前工作目录

找不到 `.git` 时仍可使用 General、UT 和 API Doc，仅禁用依赖 Git Diff 的 Review 模式。

```
Go TUI 启动
  │
  ├─ 确定项目根目录：--project 参数 > 向上查找 .git > 当前工作目录
  ├─ 通过项目根路径哈希获取进程级文件锁
  ├─ 生成 Runtime Password（32 字节 crypto/rand，仅内存）
  ├─ 选择本地端口（优先 --port 0，兼容方案：先占后放）
  ├─ 根据当前配置生成临时 OPENCODE_CONFIG_DIR
  ├─ 启动 opencode serve --hostname 127.0.0.1 --port <port>
  ├─ 轮询 GET /global/health 直到就绪（最多 30s）
  ├─ 校验 Runtime 版本兼容性
  ├─ 连接 GET /global/event
  ├─ POST /session 创建 Session
  │
  └─ 进入 TUI 主循环（Ready 状态）
       │
       ├─ 用户退出 → 取消当前 Session → POST /instance/dispose
       │              → SIGTERM → 等待 5s → SIGKILL → 释放项目锁
       ├─ Runtime 崩溃 → 提示 RuntimeCrashed，允许重启
       └─ 重复启动控制 → 进程锁 + /global/health 探测已有实例

Runtime 状态枚举：
  RuntimeStopped / RuntimeStarting / RuntimeHealthy
  / RuntimeIncompatible / RuntimeCrashed / RuntimeStopping
```

**Runtime 元信息记录（`~/.companycode/run/<hash>/runtime.json`）：**

```json
{
  "projectRoot": "/workspace/project",
  "workingDirectory": "/workspace/project/module-a",
  "configDirectory": "~/.companycode/runtime/<hash>/config",
  "pid": 12345,
  "port": 49152
}
```

- `projectRoot`：用于 Git、项目锁、AGENTS.md、项目哈希
- `workingDirectory`：用于 Maven 模块、Gradle 子项目、Tool 执行的当前目录
- 区分两者避免 monorepo 中所有操作被固定到仓库根目录

### 3.2 HTTP API 调用

| CompanyCode 操作 | HTTP 接口 | 说明 |
|---|---|---|
| 健康检查 | `GET /global/health` | 启动时和 Doctor 使用 |
| 创建 Session | `POST /session` | |
| 获取所有 Session 状态 | `GET /session/status` | 重连后状态补偿 |
| 异步发送消息 | `POST /session/:id/prompt_async` | 返回 204，通过 SSE 观察执行 |
| 获取消息列表 | `GET /session/:id/message` | 重连后补偿 |
| 获取指定消息 | `GET /session/:id/message/:messageID` | |
| 中止任务 | `POST /session/:id/abort` | |
| 响应权限申请 | `POST /session/:id/permissions/:permissionID` | Tool 审批 |

Go TUI 侧自己生成 messageID，关联 prompt_async 请求与后续 SSE 事件。

RuntimeClient 接口抽象：

```go
type RuntimeClient interface {
    Health(ctx context.Context) (HealthInfo, error)
    CreateSession(ctx context.Context, request CreateSessionRequest) (Session, error)
    SendPromptAsync(ctx context.Context, sessionID string, req PromptRequest) error
    Subscribe(ctx context.Context) (<-chan RuntimeEvent, error)
    ApprovePermission(
        ctx context.Context,
        sessionID string,
        permissionID string,
        decision PermissionDecision,
    ) error
    AbortSession(ctx context.Context, sessionID string) error
    ListAgents(ctx context.Context) ([]Agent, error)
}
```

配置更新通过独立服务完成（V1 通过重启 Runtime 生效，不假设热加载）：

```go
type RuntimeConfigService interface {
    Generate(ctx context.Context, projectRoot string) (GeneratedConfig, error)
    Apply(ctx context.Context, config GeneratedConfig) error
}
```

流程为：生成配置 → 停止旧 Runtime → 启动新 Runtime → 验证配置生效。

**底层扩展接口（能力逃生通道）：**

```go
// RawRuntimeClient 仅供 OpenCodeAdapter 内部和经过注册的兼容模块使用。
// 不向 Agent、Plugin、普通 TUI 组件或用户脚本直接暴露。
type RawRuntimeClient interface {
    Do(ctx context.Context, req RawRequest) (RawResponse, error)
}

type RawRequest struct {
    CapabilityID string  // 必须在 Capability Inventory 中已注册
    Method       string
    Path         string
    Body         any
}
```

调用前必须经过：

```
Capability Inventory 校验
  → 路径策略校验（白名单）
  → Authentication
  → DLP/审计策略
  → 请求执行
```

该接口不是完全任意的 HTTP 后门。未在 Capability Inventory 中注册的能力 ID，请求被拒绝。

### 3.3 SSE 事件流

**事件映射原则：**
- `/global/event` 是全局事件流，Adapter 必须按当前项目、Session、Message 过滤
- 原始事件名称以锁定版本 OpenAPI 为准，不以设计文档中的示意为事实

**内部事件模型：**

```go
type RuntimeEvent struct {
    ID        string        // 事件去重
    Type      EventType
    Sequence  int64         // 顺序号（本连接内）
    ProjectID string
    SessionID string
    MessageID string
    PartID    string        // 关联具体 Part
    CreatedAt time.Time
    Content   string
    Tool      *ToolEvent
    Error     *RuntimeError
    Metadata  map[string]string

    // Raw passthrough for unhandled events (原则 5：能力不退化)
    RawType        string          // OpenCode 原始事件类型
    Raw            json.RawMessage  // 原始事件载荷（不解析、不丢弃）
    RawSensitivity Sensitivity     // 敏感级别：public / internal / sensitive
}
```

**Raw 事件安全约束：**

1. 默认仅保存在内存环形缓冲区（最近 N 条，默认 500）
2. 默认不写入审计日志
3. 进入 `/events` 展示前执行 DLP
4. 输出长度限制（单条 Raw 截断至 16KB）
5. 会话结束或 Runtime 重启后清理
6. 仅 Debug 模式允许导出，导出前再次确认和脱敏

**事件处理原则：**

```
已识别事件 → 转换成 CompanyCode 领域事件
未识别事件 → 不丢弃 → 保留 RawType + Raw → 标记 RawSensitivity
           → 记录兼容性告警 → 通用事件查看器可展示（经 DLP）
```

契约门禁：OpenCode 原始 SSE 中出现的每个事件必须被映射或安全透传，不允许静默丢弃。
```

**事件类型枚举：**

```go
const (
    EventSessionStarted    EventType = "session_started"
    EventSessionStatus     EventType = "session_status"
    EventReasoningStart    EventType = "reasoning_start"
    EventReasoningDelta    EventType = "reasoning_delta"
    EventReasoningEnd      EventType = "reasoning_end"
    EventAnswerStart       EventType = "answer_start"
    EventAnswerDelta       EventType = "answer_delta"
    EventAnswerEnd         EventType = "answer_end"
    EventToolStarted       EventType = "tool_started"
    EventToolUpdated       EventType = "tool_updated"
    EventToolApproval      EventType = "tool_approval_required"
    EventToolCompleted     EventType = "tool_completed"
    EventSessionCompleted  EventType = "session_completed"
    EventSessionAborted    EventType = "session_aborted"
    EventRuntimeConnected  EventType = "runtime_connected"
    EventRuntimeDisconnected EventType = "runtime_disconnected"
    EventRuntimeError      EventType = "runtime_error"
)
```

**ToolEvent 结构：**

```go
type ToolEvent struct {
    CallID        string
    PartID        string
    ToolName      string
    Input         json.RawMessage
    InputSummary  string
    Status        ToolStatus
    Output        json.RawMessage
    OutputSummary string
    PermissionID  string
    StartedAt     time.Time
    CompletedAt   *time.Time
    Duration      time.Duration
    ExitCode      *int
    ErrorMessage  string
}
```

### 3.4 Reasoning 处理

优先级：
1. **结构化 Reasoning Part**（OpenCode SSE 原生提供）→ Adapter 直接转换为 `reasoning_delta`
2. **Text Part 中的 `<think>` 标签** → tag_parser 状态机兜底
3. **Runtime 丢弃了 reasoning** → 极小 Patch，让 Runtime 正确输出 reasoning Part

Go TUI 不绕过 Runtime 直接理解不同模型协议。

**标签解析状态机：**

```
SSE 文本块到达
    │
    ├─ 结构化 reasoning part → normalizer → reasoning_delta
    ├─ <think> 标签开始 → THINKING 状态 → 累积直到 </think>
    └─ 否则 → answer_delta
```

状态：`ThinkStateAnswer | ThinkStatePossibleOpenTag | ThinkStateReasoning | ThinkStatePossibleCloseTag`

跨 Chunk 处理：支持 `<thi` + `nk>` + `</th` + `ink>` 等拆分场景。

**Reasoning 结束推导规则：**
1. 收到 reasoning part completed → `reasoning_end`
2. 已处于 reasoning 状态，首次收到 text part → 自动 `reasoning_end` + `answer_start`
3. 收到 session idle/completed 且 reasoning 未结束 → `reasoning_end`
4. 收到 error/abort 且 reasoning 未结束 → `reasoning_end`（标记 interrupted）

### 3.5 SSE 重连与状态补偿

```
SSE 断开 → 发布 runtime_disconnected → 指数退避重连（500ms→1s→2s→5s）
→ 重连成功 → GET /session/status → GET /session/:id/message
→ 对比本地状态 → 补偿遗漏的 Message/Part → 继续接收实时事件
```

---

## 4. TUI 视觉与交互设计

### 4.1 界面布局

```
┌──────────────────────────────────────────────────┐
│ CompanyCode  ● Ready   java-backend              │  ← 顶部状态栏（1行，响应式）
│━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━│
│                                                  │
│  User > review OrderService                      │  ← 对话区（可滚动）
│                                                  │
│  ◌ Thinking                                      │  ← 推理中（暗灰斜体，最近 6-10 行）
│    正在分析代码……                                 │
│                                                  │
│  ✓ Spent 2.8s thinking                           │  ← 推理折叠（回答前）
│                                                  │
│  ◆ Code Review                                   │  ← 轻量步骤时间线
│    ✓ Read OrderService.java                      │
│    ✓ Loaded project rules                        │
│    ◌ Reviewing transaction boundaries            │
│                                                  │
│  Found 3 issues:                                 │  ← 正式回答（Markdown 渲染）
│  1. **Critical** `OrderService.java:42`          │
│     NPE risk: getUser() ...                      │
│                                                  │
│━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━│
│  ◌ Reviewing · 12s                               │  ← 任务状态（1行，完成后恢复 Ready）
│━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━│
│  > _                                              │  ← 输入区（多行）
│                                                  │
│  Ctrl+G Agent  Ctrl+K Skills  /doctor  Ctrl+L    │  ← 快捷键提示
└──────────────────────────────────────────────────┘
```

设计原则：
1. **对话优先**：主要空间留给对话和代码，不长期显示复杂面板
2. **弱边框**：正常任务用时间线，不使用大量嵌套卡片；完成后折叠为 `✓ Code Review · 4 steps · 2.8s`
3. **少量强调色**：暖白为主，琥珀为核心 Accent，红绿只表达状态
4. **响应式布局**：窄终端自动隐藏非必要字段，最低要求 70×20

### 4.2 配色方案

| 用途 | 颜色 |
|---|---|
| 主文字 | `#c0caf5` |
| 次要文字 | `#a9b1d6` |
| 弱化文字 | `#565f89` |
| Accent / Tool 名称 | `#e0af68` |
| 成功 | `#9ece6a` |
| 错误 | `#f7768e` |
| 边框 | `#292e42` |
| 背景 | `#1a1b26` |

支持终端颜色能力降级。

### 4.3 流式渲染

- SSE 内容立即累积到缓冲区
- UI 默认每 50ms 合并刷新一次（可配置 33ms~100ms）
- 已完成的对话消息缓存 Glamour ANSI 渲染结果
- 仅重新渲染当前流式消息
- 终端宽度变化时重新渲染已完成消息
- 响应结束后执行一次完整 Markdown 渲染

未完成 Markdown 处理：已完成段落用 Glamour，未闭合代码围栏尾部暂用普通文本，收到闭合标志后重新渲染。

### 4.4 Reasoning 展示

- 推理中：暗灰斜体（`#565f89`），只展示最近 6~10 行
- 推理结束：折叠为 `✓ Spent 3.2s thinking`，位于正式回答**之前**
- 用户按 `Ctrl+T` 或在折叠条聚焦时按 Enter/Space 展开/折叠
- 模型不返回 reasoning 时：不显示 Thinking、不显示 `Spent 0s thinking`
- CompanyCode 只展示 Runtime 明确提供的 reasoning 内容，不自行推导

### 4.5 Tool 确认

写操作/Shell 命令弹窗：

```
Tool permission required

Command:
mvn -Dtest=OrderServiceTest test

Working directory:
~/projects/order-service

Risk:
Runs project tests. No file deletion detected.

[Y] Allow once   [R] Remember for this project   [N] Deny   [V] View full details
```

`[R] Remember for this project` 写入 `.companycode/permissions.yaml`，默认不写入用户全局配置。

关键限制：
- 只保存 CompanyCode 生成的结构化规则（`tool + module + action`），不直接保存模型提供的任意 Shell 通配符
- 规则作用域为当前项目，不同项目独立管理

危险命令直接拒绝，显示被拒命令、命中规则、危险原因和建议手动执行。

### 4.6 快捷键

| 操作 | 按键 |
|---|---|
| 提交 | `Enter` |
| 换行 | `Shift+Enter` 或 `Alt+Enter` |
| 取消当前任务 | `Ctrl+C` |
| 退出（空闲时） | 再次 `Ctrl+C` 或 `/exit` |
| Agent 选择 | `Ctrl+G` |
| Skill 管理 | `Ctrl+K` |
| Thinking 展开 | `Ctrl+T` |
| 清屏 | `Ctrl+L` |
| 帮助 | `?` 或 `F1` |

Slash Command 后备：`/agent` `/skills` `/doctor` `/review` `/ut` `/api-doc` `/help` `/exit` `/upgrade`

TUI 原生能力入口（保证不退化）：

| 命令 | 功能 |
|---|---|
| `/sessions` | Session 列表与切换 |
| `/resume <id>` | 恢复历史 Session |
| `/agents` | Agent/Subagent 选择 |
| `/models` | 模型与 Provider 状态查看 |
| `/status` | 当前 Runtime 详细状态 |
| `/events` | 原始事件查看器（调试/兜底） |

### 4.7 Agent 与 Skill 管理

Agent 命名与模式归属：

| Agent | 模式 | 说明 |
|---|---|---|
| General | Native-Compatible | 保留 OpenCode 完整能力 |
| Code Reviewer | Enterprise-Controlled | 专用 Tool + 严格只读 |
| Unit Test | Enterprise-Controlled | 专用 Tool + 路径限制 |
| API Documentation | Enterprise-Controlled | 专用 Tool + 路径限制 |

Skill 面板显示：

```
Skill                 Installed  Enabled  Matched   Source
code-review           yes        yes      yes       project
java-unit-test        yes        yes      yes       project
python-unit-test      yes        no       no        profile
log-analyzer          yes        yes      no        user (manual)
```

---

## 5. Skill 动态开关机制

### 5.1 三态模型

```
Installed  → 发行包已包含且通过完整性与兼容性检查
Enabled    → 当前项目允许暴露给 OpenCode（持久配置状态）
Loaded     → 当前 Session 中 Agent 已通过 skill Tool 加载完整 SKILL.md（运行状态，非持久）
```

### 5.2 配置层级与合并

层优先级（从低到高）：

```
1. 发行包基础默认 (release)
2. 当前 Profile (profile)
3. 用户级配置 (user)
4. 项目级配置 (project)
```

配置值：`enabled` | `disabled` | `inherit`

未出现的 Skill 等同于 `inherit`，只有 `enabled` 和 `disabled` 会覆盖。

Profile 选择优先级：项目显式指定 > 用户显式指定 > 技术栈自动识别 > minimal

### 5.3 物理可见性控制（核心机制）

Disabled Skill 不仅设置 Permission deny，而是**根本不放入 Runtime 临时配置目录**。

**关键问题：** OpenCode 除了自定义配置目录，还会读取用户全局配置目录、项目 `.opencode/skills/`、Claude Code 兼容 Skill 目录等来源。即使 CompanyCode 临时目录只放 Enabled Skill，用户机器上的其他 Skill 仍可能被 OpenCode 发现。

**因此必须启用 Runtime 配置隔离模式：**

```
启动 Runtime 时设置：
  OPENCODE_CONFIG_DIR=<generated-runtime-config>
  OPENCODE_DISABLE_CLAUDE_CODE=1

并验证以下目录不会额外注入 Skill：
  - 项目 .opencode/skills/
  - 用户 ~/.config/opencode/skills/
  - 用户 ~/.claude/skills/
```

**Skill 来源等级：**

```
1. Approved — CompanyCode 发行包预装并审核通过
2. Project  — 项目自带 Skill（.opencode/skills/ 等），经本地校验后可用
3. User     — 用户级自定义 Skill
```

**双轨策略：**

| Profile | Skill 来源 | 适用场景 |
|---|---|---|
| Enterprise Profile | 仅 Approved Skill | Code Reviewer、Unit Test、API Doc |
| General Profile (strict) | 仅 Approved Skill（V1 默认） | General Agent 默认模式 |
| General Profile (compatible) | Approved + Project + User Skill（经校验确认后启用） | 用户显式开启 |

General Profile 的 `compatible` 模式：

```yaml
general:
  skillMode: compatible
```

启用后：读取 Project/User Skill → 检查 SKILL.md 和 Manifest → 检测网络/Shell 依赖 → 展示来源 → 用户确认后启用。首次发现时提示来源和风险。Project Skill 和 User Skill 均需经过本地静态校验、权限分析和首次启用确认。

**V1 承诺：** General 模式必须支持 `compatible` Skill 模式，确保 Project 和 User 自定义 Skill 不被静默丢弃。这是「能力不减弱」的组成部分。

**Plugin 兼容策略（与 Skill 对称）：**

| 来源 | 说明 |
|---|---|
| Approved Plugin | CompanyCode 发行包预装并审核通过 |
| Project Plugin | 项目自带 Plugin（`.opencode/plugins/` 等） |
| User Plugin | 用户级自定义 Plugin |

| Profile | Plugin 来源 |
|---|---|
| Enterprise Profile | 仅 Approved Plugin |
| General Profile (strict) | 仅 Approved Plugin（V1 默认） |
| General Profile (compatible) | Approved + Project + User Plugin（经 Bundle/依赖/网络权限检查并由用户确认后启用） |

Capability Inventory 标记：

```yaml
plugins:
  approved: required
  projectCompatible: required   # compatible 模式必须支持
  userCompatible: required      # compatible 模式必须支持
```

若 V1 暂不允许第三方 Plugin，必须在 Inventory 中明确标记为 `deferred`，不能笼统标记 `required` 而实际只支持预装 Plugin。

**隔离策略（V1 严格模式，仅 Enterprise Profile）：**
- Enterprise Agent Runtime 只加载 Approved Skill
- 忽略用户自行安装的 OpenCode/Claude Skill
- 项目自定义 Skill 不向企业 Agent 暴露

**三级实现策略（由 Spike 验证后确定最终方案）：**

```
第一层：原生配置隔离
  OPENCODE_CONFIG_DIR=<generated-runtime-config>
  OPENCODE_DISABLE_CLAUDE_CODE=1

第二层：进程环境隔离
  若原生配置无法隔离项目/用户级 Skill，为 Runtime 指定独立环境：
  HOME=<companycode-runtime-home>
  XDG_CONFIG_HOME=<companycode-runtime-home>/.config
  注意：不能影响项目文件和工具链执行

第三层：极小 Patch
  若 OpenCode 仍不可避免地扫描项目 .opencode/skills/，
  修改 Skill Discovery 搜索路径，使 CompanyCode 模式仅加载生成目录。
  Patch 纳入契约测试。
```

**Spike S5 失败处置：**

```
若原生配置无法隔离：
  1. 尝试独立 HOME/XDG 环境
  2. 仍无法隔离则维护 Skill Discovery 极小 Patch
  3. Patch 纳入契约测试

最终验证：通过契约测试确认 Runtime 发现列表中只包含 Enabled Skill。
```

完整流程：

```
发行包包含所有 Installed Skill
  → CompanyCode 计算 Effective Skills
  → 生成 Runtime 配置目录
  → 只链接/复制 Enabled Skill
  → 以隔离模式启动 OpenCode Runtime
  → 通过契约测试验证发现列表中只包含 Enabled Skill
```

Permission 作为第二道防线，不是唯一开关。

### 5.4 技术栈自动识别

```yaml
match:
  any:
    - pom.xml
    - build.gradle
    - build.gradle.kts
    - src/main/java/
```

- `any`：命中一个即可
- `all`：必须全部命中
- 以 `/` 结尾表示目录，其余表示文件
- 默认使用项目根目录相对路径

**Matched ≠ Enabled：** Matched 只参与推荐和 Profile 自动选择，不强制阻断用户手动启用。混合项目不会被单个 `package.json` 错误覆盖。

### 5.5 配置变更生效

V1 通过 Runtime 重启使配置生效（不假设热加载）：

```
用户修改 Skill 配置
  → 计算新的 Effective Skills
  → 如果当前无任务执行 → 重新生成 Runtime 配置目录
  → 重启 OpenCode Runtime → 恢复或新建 Session
```

TUI 区分 `Configured State` 与 `Effective State`，未生效配置显示 `pending runtime restart`。

### 5.6 Agent 与 Skill 关系

Agent Manifest 定义 requiredSkills 和 optionalSkills：

```yaml
requiredSkills:
  - code-review
optionalSkills:
  - java-review
  - security-review
```

CompanyCode 启动前校验：required Skill 未启用 → Agent 不可用或提示用户启用。

### 5.7 按需加载流程

```
用户发送消息
  → 当前 Agent 的 skill Tool 中仅暴露 Enabled Skill 的名称和描述
  → Agent 判断 Skill 与任务相关
  → Agent 调用 skill({name:"code-review"})
  → OpenCode 执行 skill Permission 检查
  → 读取完整 SKILL.md
  → Skill 内容加入当前任务上下文
  → CompanyCode 从 Tool/SSE 事件记录 Loaded 状态
```

---

## 6. 私有模型接入、离线发行包、升级回滚

### 6.1 私有模型接入

**配置层：** CompanyCode 配置 → 生成 OpenCode `opencode.json` → `OPENCODE_CONFIG_DIR` → OpenCode Provider

CompanyCode 配置：

```yaml
schemaVersion: 1
model:
  providerId: company-private
  protocol: openai-compatible
  baseUrl: https://api.deepseek.com/v1
  modelId: deepseek-chat
  apiKeyEnv: COMPANYCODE_MODEL_API_KEY
  timeout: 120s
  maxRetries: 2
  capabilities:
    streaming: auto
    toolCall: auto
    reasoning: auto
```

API Key 默认通过环境变量引用，不写入普通配置文件。V1 如需保存本地，使用独立 secrets 文件（权限 0600），日志和 Doctor 永不打印原值。

**连接测试（`company-code init`）：** 必须测试普通对话、流式、Tool Call 和 Reasoning 四项。Tool Call 不支持时判定为配置不合格；Reasoning 不支持时只关闭思考展示。

**重试约束：** Go TUI 只负责连接 OpenCode Server 的网络重试；OpenCode Runtime 负责单次模型请求；模型 Gateway 负责底层重试。涉及 Tool Call 的请求不能无条件重试。

### 6.2 离线发行包

**构建流程：**

```
读取 runtime/version.json → 下载固定版本 OpenCode（校验 SHA256）
  → 编译各平台 Go TUI → 构建 Plugin（bun install --frozen-lockfile + build）
  → 收集 distribution/ → 生成 Manifest → 打包 → 断网验证
```

**发行包结构：**

```
company-code-1.0.0-darwin-arm64/
├── bin/
│   ├── company-code
│   └── opencode
├── distribution/
│   ├── agents/  skills/  plugins/  config/  templates/
├── runtime/
│   ├── version.json
│   └── compatibility.json
├── manifest.json
├── manifest.sha256
├── LICENSE
├── THIRD_PARTY_LICENSES.md
├── VERSION
└── install.sh
```

**Manifest 结构：**

```json
{
  "schemaVersion": 1,
  "packageId": "company-code-1.0.0-darwin-arm64+20260730.1",
  "companyCodeVersion": "1.0.0",
  "buildId": "20260730.1",
  "openCodeVersion": "x.y.z",
  "openCodeCommit": "abc1234",
  "platform": {"os": "darwin", "arch": "arm64"},
  "configSchemaVersion": 1,
  "createdAt": "2026-07-30T12:00:00Z",
  "files": [
    {"path": "bin/company-code", "sha256": "abc...", "size": 15432100, "executable": true}
  ],
  "skills": [],
  "plugins": []
}
```

构建脚本只能根据 `runtime/version.json` 下载固定版本并校验固定 SHA256，不能隐式获取 latest。

Plugin 构建后不得触发在线依赖安装。**关键约束：** OpenCode 如果检测到配置目录存在 `package.json`，会在启动时执行 `bun install`，这会破坏离线目标。因此发行包的 Runtime 配置目录**只复制自包含 Bundle JS 文件**，不复制 `package.json`、`bun.lock` 或 `node_modules`。所有第三方依赖必须被打进 Bundle。

**Plugin Bundle 构建约束：**

```yaml
plugins:
  bundle:
    format: esm
    target: bun
    external: []
    sourcemap: false
    minify: false
```

构建命令：

```bash
bun build \
  distribution/plugins/src/dify-query.ts \
  --outdir distribution/plugins/dist \
  --target bun \
  --format esm
```

Spike 中必须确认：ESM/CJS 兼容性、default/named export 要求、单文件内联依赖是否支持、Node/Bun 内置模块处理。

**Plugin Bundle Gate（G2.1）：**
- 无第三方外部 import
- 不包含绝对构建路径
- 断网启动成功
- OpenCode 能加载 Plugin
- Tool 能成功注册

构建流程：

```
bun install --frozen-lockfile
  → Bundle TypeScript Plugin 为自包含 JS（ESM, target bun）
  → 验证无未解析的外部 import
  → 验证无 npm package 动态加载
  → 验证启动时不触发 bun install
  → 复制 dist/*.js 到发行包
```

发行包中 Plugin 目录仅包含：

```
distribution/plugins/
├── dify-query.js
├── runtime-security-guard.js
├── audit-log.js
└── manifests/
```

构建输出单独放在 `packaging/staging/distribution/plugins/`，避免打包脚本误复制源码和 `package.json`。

### 6.3 升级与回滚

**版本目录：**

```
~/.companycode/
├── bin/
│   └── company-code          # 稳定启动入口
├── config/                    # 用户配置（不随版本覆盖）
├── data/  logs/  backups/
└── versions/
    ├── 1.0.0+20260701.1/      # 版本 + Build ID
    ├── 1.1.0+20260730.1/
    └── current                # 符号链接或指针文件
```

**升级流程（正确顺序）：**

设旧版本 V1，旧配置 C1，新版本 V2，迁移配置副本 C2-temp。

```
1. 获取升级锁并确认无任务执行
2. 校验压缩包整体哈希
3. 解压到临时 staging 目录
4. 校验 Manifest 和全部文件
5. 校验平台、架构和版本兼容性
6. 复制当前配置 C1 → C2-temp（不在原始配置上操作）
7. 对 C2-temp 执行配置迁移
8. 使用 V2 + C2-temp 运行预切换 Doctor
9. 安装 V2 版本目录
10. 记录 pending 事务（包含 V1/V2 版本、C1/C2-temp 状态）
11. 原子切换 current → V2
12. Runtime 使用 C2-temp 启动
13. 执行切换后 Doctor
14. 通过后原子替换正式配置 C1 → C2（迁移后的配置副本）
15. 更新事务状态为 completed
16. 失败则：停止失败的 Runtime → 切回旧 Runtime 指针 → 切回旧配置
     → 启动旧 Runtime → 执行恢复 Doctor → 确保旧版本可用
```

**关键约束：**
- 新版本切换后使用经过迁移和验证的配置副本（C2-temp）
- 切换后 Doctor 成功，才原子替换正式配置 C1 → C2
- 若失败，Runtime 和配置都回退到升级前状态
- 恢复 Doctor 必须验证旧 Runtime 可启动并通过基础检查，不仅检查 current 指针已恢复

核心原则：新版本和新配置必须先在隔离环境验证，通过后才能切换。

**同版本覆盖策略：** 默认拒绝重复安装，提供 `company-code repair <package>` 命令（解压到新临时目录 → 完整校验 → 原子替换整个版本目录）。

**回滚流程：**

```
1. 选择目标版本
2. 检查当前是否有运行任务
3. 检查目标版本与当前配置兼容性
4. 备份当前配置
5. 切换 current
6. 启动目标 Runtime
7. 执行 Doctor
8. 若配置不兼容 → 提示恢复目标版本对应快照
9. 失败 → 重新切回原版本
```

**配置迁移：** 配置 `schemaVersion` 定义迁移路径。迁移器：`FromVersion → ToVersion`，只能向副本迁移，失败退出时不覆盖。

**升级锁：** `~/.companycode/update.lock` 防止两个终端同时升级。运行中不允许升级。

---

## 7. 企业能力

### 7.1 统一设计原则

```
Agent   — 负责角色、任务流程、Tool 权限和任务边界
Skill   — 负责方法论、检查规则、输出要求
Tool    — 负责确定性的数据提取、命令执行和文件修改
模型     — 负责理解、判断、归纳和生成说明
```

统一执行流程：

```
解析用户目标 → 确定任务输入范围 → 调用确定性 Tool 收集事实
→ 加载 Agent requiredSkills → 查询 AGENTS.md / Dify
→ 模型分析或生成 → 结构化校验 → 生成最终 Markdown 报告
```

三个能力都先输出结构化中间结果，经程序校验后再转换为 Markdown。

### 7.2 Code Reviewer

**输入源：**

```
/review                        → tracked staged + unstaged changes vs HEAD（默认）
                                  collect_review_context 额外发现未跟踪源文件并提示是否纳入
/review --staged               → git diff --cached
/review --base origin/main     → git diff origin/main...HEAD
/review --commit abc123        → git diff abc123^..abc123
/review --range abc..def       → 指定区间
/review path/to/file           → 指定文件
```

**Diff Collector Tool（`collect_review_context`）：** 确定性返回 diff 范围、变更文件、变更行号、上下文代码，保证行号准确。

**问题定位类型：** `LINE` | `FILE` | `CHANGESET`

**严重级别：**

| 级别 | 定义 |
|---|---|
| Critical | 明确安全漏洞、数据损坏风险、必现故障、权限绕过 |
| Major | 功能错误、高概率异常、事务/并发/缓存风险、性能退化、缺少关键校验 |
| Minor | 低概率边界问题、可维护性下降、日志/异常/规范问题 |
| Suggestion | 可选优化、代码风格、可读性增强 |

每条问题包含：`Confidence`（high/medium/low）、`IntroducedByChange`（true/false/uncertain）。

high + medium → Issues；low → Observations。

V1 默认正式报告 `introducedByChange = true` 的问题，历史问题单独列为 Pre-existing observation。

**输出结构：** 包含 Scope、Summary、Issues、Observations、Positive Findings、Suggested Verification。

**Agent 权限：** read/grep/glob/collect_review_context/dify-query → allow；edit/write/bash → deny。

### 7.3 Unit Test Generator

**流程：**

```
1. 调用 analyze_test_project → 确定构建系统、测试目录、框架
2. 生成结构化 TestPlan（用例 ID、场景、类型、预期结果）
3. 生成测试代码（JUnit 5 + Mockito）
4. 调用 write_test_file → 写入（限检测试根目录）
5. 调用 run_project_test → 执行（优先 ./mvnw > mvn；./gradlew > gradle）
6. 分类失败类型 → 决定是否修复和如何修复
7. 重试循环（最多 3 次模型生成/修复尝试）
```

**失败分类与处理策略：**

| 类型 | 处理 |
|---|---|
| COMPILE_ERROR | 修改测试代码 |
| ASSERTION_FAILURE | 谨慎：先判断是否暴露生产代码 Bug |
| MOCK_CONFIGURATION_ERROR | 修改测试代码 |
| TEST_DISCOVERY_ERROR | 修改测试代码 |
| PRODUCTION_CODE_FAILURE | 停止，报告生产代码问题 |
| TOOLCHAIN_ERROR | 停止，不修改代码 |
| DEPENDENCY_ERROR | 停止 |
| ENVIRONMENT_ERROR | 停止，不消耗修复次数 |
| TIMEOUT | 默认停止 |

**安全约束：**
- 只允许修改 `analyze_test_project` 确定的测试源码目录
- 使用专用 `write_test_file` Tool（路径必须在检测的 test roots 内）
- 使用专用 `run_project_test` Tool（内部执行白名单命令，Maven/Gradle Wrapper 优先）
- 禁止直接开放通用 `edit` 或 `bash`
- 已有测试文件不得覆盖（分析已有结构 → 追加或生成 Patch → 用户确认）
- 断言失败不能默认修改断言迎合生产代码

**Agent 权限：** read/grep/glob/analyze_test_project/write_test_file/run_project_test/dify-query → allow；edit/bash → deny。

### 7.4 API Documentation Generator

**上下文提取（确定性）：**

使用专用 `extract_api_spec` Tool 从 Controller 提取：

- 路由路径（处理组合路由：Controller @RequestMapping + Method @PostMapping）
- HTTP Method
- 参数名、类型、必填（@PathVariable / @RequestParam / @RequestBody）
- 校验规则（Bean Validation 注解）
- DTO 字段信息
- 枚举值列表
- 错误码

**模型职责：** 基于提取的结构化结果组织文档、补充业务语义、生成示例。

**模型禁止：** 虚构字段、猜测路由、添加不存在的校验规则、编造错误码。

**示例生成约束：**
- 字段只能来自提取结果
- 枚举值只能从枚举列表选
- 数值范围满足 Validation
- 必填字段不能遗漏
- 生成后通过 `validate_api_example` 校验 Schema

**错误码状态：** `DECLARED`（直接引用或 ExceptionHandler 映射）| `REFERENCED`（调用链引用）| `INFERRED`（推断）

正式文档只输出 DECLARED 和 REFERENCED。INFERRED 放入 "Potential errors requiring confirmation"。

**V1 支持范围：** Java 17 + Spring Boot + Spring MVC + Bean Validation + 普通 DTO + Enum + 统一响应包装类 + 公司标准错误码结构。暂不保证 WebFlux、动态路由、复杂注解组合、Lombok Builder 推断、泛型多层嵌套、GraphQL、RPC。

**输出原则：** 无法确定的信息显示 `Not determined from code`，不猜测。

**Agent 权限：** read/grep/extract_api_spec/validate_api_example/dify-query → allow；write_document → allow（路径限制为 docs/、doc/、api-docs/ 或用户明确选择的输出路径）；通用 write/edit/bash → deny。

三个企业 Agent 都不使用通用 write/edit/bash：

| Agent | 写入方式 | 路径限制 |
|---|---|---|
| Code Reviewer | deny write | — |
| Unit Test | write_test_file | 检测的 test roots 内 |
| API Documentation | write_document | docs/, doc/, api-docs/, 用户指定 |

### 7.5 信息优先级

```
代码与工具执行事实 > 项目 AGENTS.md > Dify 企业规范 > 模型通用知识
```

代码事实决定「现在是什么」，企业规范决定「应该是什么」。两者冲突时都记录，但不允许为迎合规范而篡改代码事实。

### 7.6 企业 Agent 逃生机制

企业 Agent 被专用 Tool 严格限制时，不能因 Tool 参数不足就彻底失败。

每个专用 Tool 提供结构化扩展参数：

```go
type RunProjectTestRequest struct {
    BuildSystem string
    Module      string
    TestClass   string
    TestMethod  string
    Profiles    []string
    SystemProps map[string]string
    ExtraArgs   []string    // 经白名单校验
    TimeoutSeconds int
}
```

- `ExtraArgs` 经过白名单校验
- 路径必须在项目内
- 环境变量只允许安全名单
- 超出能力时返回 `UNSUPPORTED_REQUEST`，不让模型构造任意 Shell

任务失败时允许用户选择：

```
[R] Retry with General Agent
[M] Show manual command
[C] Cancel
```

企业 Agent 被专用 Tool 限制时，可以将任务交还 General Agent 继续执行，不丢失上下文。

**AgentHandoff 切换载荷：**

```go
type AgentHandoff struct {
    SourceAgent    string           // 来源 Agent
    TargetAgent    string           // 目标 Agent
    SessionID      string           // 保留同一 Session
    UserGoal       string           // 用户原始目标
    TaskSummary    string           // 已执行步骤摘要
    CollectedFacts []FactRef        // 已收集事实引用（不重复获取）
    GeneratedFiles []string         // 已生成文件列表（不重复覆盖）
    ToolResults    []ToolResultRef  // 已完成的 Tool 结果
    FailureReason  string           // 切换原因
}
```

切换原则：
- 保留同一 Session（不新建）
- 生成结构化 Handoff（不把整个历史重新塞给模型）
- 引用已完成 Tool 结果（不重复执行）
- 明确未完成步骤
- 用户原始目标保留
- 失败原因传递
- General 能继续执行或给出明确方案

---

## 8. 安全控制、Dify、安装与 Doctor、试点统计

### 8.1 安全控制

**核心原则：** 企业 Agent 通过专用 Tool 执行确定性操作；通用 Bash 仅提供给 General Agent 并默认要求确认。

**Shell 规则：** 判断维度包括命令、参数、工作目录、读写路径、重定向、管道、子命令、环境变量。不能单独按命令名自动放行。

**自动放行命令：** 使用 `exec.CommandContext(ctx, "git", "status")` (argv 数组)，不经过 `sh -c`。自由输入中出现管道、重定向、命令替换、多命令组合，统一进入 ask 或 deny。

**高危命令直接拒绝：** `git reset --hard`、`git clean -fd`、`git push --force`、`git checkout -- .` 等。

**Agent 默认权限（Native-Compatible Mode — General Agent）：**

原则：安全不等于能力删除。通过 Allow/Ask/Deny 实现安全控制，不通过删除 Tool 实现。

| 能力 | 策略 | 说明 |
|---|---|---|
| read/grep/glob | allow | 敏感路径（.env, *.key, id_rsa 等）deny |
| write/edit | ask | 每次写操作需用户确认 |
| bash 普通命令 | ask | 每次需用户确认 |
| 确定性只读命令 | allow | git status/diff/log, ls, cat（非敏感路径）等 |
| 明确高危命令 | deny | rm -rf, git reset --hard, chmod 777 等 |
| Agent/Subagent | allow | 保留原生多 Agent 能力 |
| 批准 Skill | allow | CompanyCode 预装 Skill |
| 项目 Skill | ask | 首次发现时校验、展示来源、用户确认 |
| 内网 Maven/npm/PyPI/Go Proxy | ask/allow | 访问白名单镜像时允许 |
| 公网地址 | deny | |

**Agent 默认权限（Enterprise-Controlled Mode）：**

| Agent | bash | edit | write | write_document | write_test_file | 专用 Tool |
|---|---|---|---|---|---|---|
| Code Reviewer | deny | deny | deny | deny | deny | allow |
| Unit Test | deny | deny | deny | deny | allow | allow |
| API Documentation | deny | deny | deny | allow | deny | allow |

General Agent 不在此表中，其权限按上表 Native-Compatible 策略独立管理。

**网络白名单（应用层）：** 模型地址白名单、Dify 地址白名单、Bash 禁止 curl/wget 访问未批准地址。

**内网依赖源：** 禁止访问未批准的软件源；允许访问配置白名单中的内网 Maven、npm、PyPI 和 Go Proxy 镜像。General Agent 在批准的内网镜像上可正常执行依赖解析和构建。

**Plugin 网络控制：** Plugin 默认禁止网络访问。声明网络能力的 Plugin 只能访问 Manifest 中列出的白名单 Endpoint：

```yaml
# Plugin manifest
permissions:
  network:
    allow:
      - configRef: dify.baseUrl
```

未声明网络权限的 Plugin：

```yaml
permissions:
  network: false
```

**完整网络隔离：** 由公司操作系统/网络层策略兜底（主机防火墙、出口代理、DNS 控制、网络 ACL）。

**DLP 是跨层能力，由多层共同实现，不能仅依靠 OpenCode Plugin：**

| 层级 | 职责 | 实现位置 |
|---|---|---|
| Custom Tool DLP | 专用 Tool 的输入/输出扫描 | `collect_review_context`、`query_company_knowledge`、`extract_api_spec`、`run_project_test` |
| Runtime Plugin DLP | Tool before/after hook、文件读取路径拦截、可覆盖的 Runtime 事件脱敏 | `runtime-security-guard.ts`（职责收窄：不承担完整 DLP） |
| Go 导出/审计层 DLP | 日志写入前、会话导出前 | Go TUI 侧 |
| Model Gateway DLP | 最终发往模型的所有请求 | 公司模型 Gateway（最可靠边界） |

如果必须保证所有模型请求都经过统一扫描，最可靠的位置是公司模型 Gateway，不是 OpenCode Plugin。

**高风险文件默认禁止读取：** `.env`、`*.key`、`*.p12`、`id_rsa*`、`credentials*`、`secrets.*`。

**脱敏占位符：** 使用稳定编号 `[REDACTED:TOKEN:1]`，同一值映射相同编号，映射表仅内存保存。

**DLP 动作：** Block（私钥）| Redact（Token/密码）| Ask（疑似敏感）| Allow。

**审计日志原则：** 不保存完整源码、完整 Prompt、Token、完整 Tool 输出。路径使用项目相对路径或哈希。配置 `retentionDays`、`maxFileSizeMB`、`maxFiles`。

### 8.2 Dify 知识查询

**接入方式：** OpenCode Plugin，注册为 `query_company_knowledge` Tool。

**降级策略：** 默认 `warn`（非 silent）。企业 Agent 最终报告标注知识来源：

```
Knowledge sources:
✓ AGENTS.md
! Dify enterprise knowledge unavailable
```

**参数限制：** topK 最大 10，query 最大 1000 字符，单条文档最大 2000 字符，总上下文最大 8000 字符。

**查询原则：** 发送概念和场景关键词，不发送完整代码。必须发送代码片段时先执行 DLP 并限制长度。

**熔断：** 连续失败 3 次 → 熔断 60 秒 → 期间立即降级 → 到期后允许一次试探。相同查询在 Session 内缓存。

### 8.3 安装与 Doctor

**安装流程：**

```
1. 校验压缩包 SHA256 → 校验 Manifest → 校验平台架构
2. 创建 staging 目录 → 解压并校验全部文件
3. 创建 ~/.companycode 目录 → 安装到 versions/<version+buildId>
4. 原子设置 current → 创建稳定启动入口 → 设置文件权限
5. 执行基础 Doctor → 提示用户执行 init
```

支持非交互部署：`company-code init --config company-defaults.yaml --non-interactive`

**文件权限：** `~/.companycode/` 0700、secrets 600、logs 700、bin/* 755、普通配置 600。

**稳定启动入口：** `~/.companycode/bin/company-code` 读取 current 再执行真实程序，不直接链接到可变路径。Windows 单独处理。

**Doctor 检查分类：**

| 类型 | 检查内容 |
|---|---|
| 静态检查 | Manifest、文件哈希、配置 Schema、文件权限、Skill Manifest、Plugin 文件、版本兼容、工具链路径 |
| 连接检查 | Runtime 健康、模型 Endpoint、Dify Endpoint、SSE 连接 |
| 行为检查 | 普通回答、流式、合成 Tool Call、Reasoning 检测（使用合成数据，不操作真实项目） |
| 网络检查 | 配置白名单地址、未发现非预期连接、离线模式标志、公网 DNS 被阻断 |

Doctor 严重级别：PASS / WARN / FAIL / SKIP。核心检查失败时返回非零 Exit Code。

网络检查输出：

```
✓ Network Policy
  Configured endpoints are allowlisted
  No unexpected connection observed during checks
```

不表述为「确认无公网访问」。真正的断网保证由独立隔离环境测试完成。

### 8.4 试点统计

**统计指标：** 会话次数、Agent 类型分布、任务耗时、采纳状态、评分、错误分类、Skill 加载频率。

**数据隐私：**
- 默认只记录元数据（Agent 类型、Skill ID、时间、结果、评分）
- 不记录用户 Prompt、模型完整回答、源码、Diff、Tool 完整输入输出、API Key、项目绝对路径
- 项目使用不可逆哈希或本地匿名 ID

**采纳状态枚举：** `accepted_as_is` | `accepted_with_minor_changes` | `accepted_with_major_changes` | `rejected`

**反馈机制：** 每次任务仅提供轻量可跳过反馈（Yes/Partly/No）；试点开始前收集基准耗时；试点结束时统一调查。

**数据 Schema：**

```json
{
  "schemaVersion": 1,
  "eventId": "evt_xxx",
  "sessionId": "sess_abc123",
  "projectHash": "project_8f9a13",
  "agent": "code-reviewer",
  "startedAt": "2026-07-30T15:30:00Z",
  "completedAt": "2026-07-30T15:32:48Z",
  "durationMs": 168000,
  "status": "completed",
  "adoption": "accepted_with_minor_changes",
  "rating": 4,
  "errorCategory": null,
  "skillsLoaded": ["code-review"]
}
```

---

## 附录 A：开发顺序

| 阶段 | Task | 内容 |
|---|---|---|
| Phase 0 | Spike 1-5 | 验证 Server API、SSE、Tool Approval、Reasoning、Skill 隔离 |
| Phase 1 | 能力盘点 | OpenCode 原生能力清单 + General Agent Parity Harness + Golden Event 录制 |
| Phase 2 | T1-T2 + T12-T14 | OpenCode 基线 + 私有模型 + Go TUI 基础（完成能力透传） |
| Phase 3 | General 对齐 | Session/Resume、Tool、Agent/Subagent、Skill(compatible)、Shell、Edit 全能力对齐 |
| Phase 4 | T3-T4 | 离线发行包 + 升级回滚 |
| Phase 5 | T5-T6-T7-T11 | Skill Manifest + 动态开关 + Dify + 安全控制 + AGENTS.md |
| Phase 6 | T8-T10 | Code Reviewer + Unit Test Generator + API Doc Gen |
| Phase 7 | T15 | 安装与 Doctor |
| Phase 8 | T16 | 试点与效果统计 |
| Phase 9 | Release Parity Certification | 全量 G11-G15 门禁一次性通过；发布前最终确认 |

**Parity 持续门禁原则：**

Parity 不是最后阶段补测，而是从 Phase 2 开始持续运行的回归套件：

- Phase 1：建立 Baseline 和 Harness
- Phase 2 起：每个阶段执行受影响能力的增量 Parity
- 每次提交：执行快速 Parity（核心能力冒烟测试）
- 每个版本：执行完整 Parity
- 发布前：执行 G11～G15 全量门禁

## 附录 B：验收标准

### B.1 Phase 0：Spike 验证门禁

必须在进入 Phase 1 前全部通过：

| # | 验证项 | 通过标准 |
|---|---|---|
| S1 | Server 离线启动 | `opencode serve` 在断网环境启动，不访问公网 |
| S2 | Go 调用 OpenCode | 创建 Session → 发送消息 → 接收 SSE → 完成一轮完整 Agent 会话 |
| S3 | Tool Approval | Go TUI 能接收权限申请事件，批准/拒绝后 Runtime 正确执行 |
| S4 | Reasoning 接收 | Go 客户端能实时接收并区分 reasoning 与 answer；若无法接收则定位丢失层 |
| S5 | Skill 隔离 | Enabled Skill 隔离验证通过：其他来源 Skill 不被注入 |

### B.2 V1 发布门禁

以下为核心 Release Gate，完整指标仍可引用需求文档第 7 节：

| # | 门禁项 | 判定标准 |
|---|---|---|
| G1 | 断网安装 | 新机器完全断网可安装并启动，不超过 5 分钟 |
| G2 | 无启动时依赖安装 | 启动 Runtime 不触发 npm/bun/pip/mvn 在线安装 |
| G2.1 | Plugin 离线加载 | 所有预装 Plugin 在完全断网环境中成功注册和执行 |
| G3 | 配置与 Skill 隔离 | Enterprise-Controlled Mode：只加载 Approved 且 Enabled 的 Skill。General strict Mode：只加载 Approved 且 Enabled 的 Skill。General compatible Mode：可加载 Approved Skill 和经校验、展示来源并由用户确认的 Project/User Skill；不得静默加载未批准的其他来源 Skill |
| G4 | 升级原子性 | 升级失败后旧 Runtime 可启动并通过基础 Doctor，配置不丢失（不仅检查 current 指针已恢复） |
| G5 | 回滚可恢复 | 一条命令回滚到上一版本 |
| G6 | Code Review 证据定位 | 每条问题包含文件、行号、代码证据；无空泛建议 |
| G7 | UT 生产代码误修改 | 次数为 0 |
| G8 | API Doc 虚构字段 | 数量为 0 |
| G9 | TUI Reasoning 展示 | 正确分离思考与回答，支持折叠和展开 |
| G10 | 安全控制 | 高危命令直接拒绝；写操作默认要求确认 |
| G11 | 原生能力完整性 | 锁定版本 OpenCode 核心能力清单全部可通过 CompanyCode 使用；未识别 SSE 事件不得静默丢弃 |
| G12 | 能力层完整性 | 核心功能可达率、原生 Tool 可调用率、事件映射或透传率、API 能力无丢失率均必须 100% |
| G12.1 | 任务效果 Parity | 同模型/同 Runtime/同权限条件下，General 任务完成率不低于原版 OpenCode 的 95%；任何核心任务（读写文件、Shell、Tool Approval、Session、Abort、Agent/Subagent、Skill、错误显示）不得失败 |
| G13 | 事件零静默丢失 | Golden SSE 样本中所有事件均被映射或 Raw 透传，静默丢失数量为 0 |
| G14 | 企业 Agent 可回退 | 专用 Tool 不支持时，通过 AgentHandoff 转交 General Agent。转交后：用户原始目标保留、已收集事实不重复丢失、已生成文件不重复覆盖、失败原因传递、General 能继续执行或给出明确方案 |
| G15 | 内网依赖源可用 | Maven、npm、PyPI、Go Proxy 指向批准的内网镜像时，General Agent 能正常执行依赖解析和构建 |

## 附录 C：关键术语

| 术语 | 含义 |
|---|---|
| OpenCode Runtime | 以 `opencode serve` 模式运行的 OpenCode 进程 |
| RuntimeClient | Go TUI 侧定义的 CompanyCode 内部接口 |
| OpenCodeAdapter | RuntimeClient 的 OpenCode HTTP/SSE 实现 |
| Effective Skills | 经过四级配置合并后的最终 Skill 启用状态 |
| Runtime Password | 每次启动生成的 Basic Auth 随机密码 |
| Build ID | 发行包构建标识（`20260730.1`），同一语义版本可区分多次构建 |
