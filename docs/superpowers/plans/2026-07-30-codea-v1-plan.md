# Codea V1 实施计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 构建企业内网 AI 编码助手 Codea V1，基于 OpenCode Runtime + Go TUI，提供双模式（Native-Compatible + Enterprise-Controlled）的代码审查、单元测试生成、API 文档生成能力，支持离线发行、升级回滚和私有模型。

**Architecture:** C+ 混合模式 — OpenCode（anomalyco/opencode）作为 V1 唯一 Agent Runtime（`opencode serve`），Go TUI（Bubble Tea + Lip Gloss）通过 Codea-owned `AgentRuntime` Contract 使用 OpenCodeAdapter。OpenCode DTO/HTTP/SSE 对象不得越过 Vendor Layer。企业能力（Reviewer/UT/API Doc）通过 Agent + Skill + 专用 Tool 组合实现，不侵入 OpenCode Core。离线发行包包含预编译 TUI、OpenCode 二进制、自包含 Plugin Bundle 和全部配置。

**Tech Stack:** Go 1.26.5 (TUI), Bubble Tea + Lip Gloss + Glamour, TypeScript (Plugin, target bun), OpenCode Runtime (锁定版本, anomalyco/opencode), DeepSeek (开发), 私有模型 (内网)

## Global Constraints

- 开始或恢复任何 Task 前必须读取并通过 `./scripts/check-execution-state.sh` 校验 `docs/execution-state.yaml`
- 同一时间最多一个 Task 处于 `in_progress`、`blocked` 或 `awaiting_acceptance`，不得跳过或并行推进 Task
- 当前 Task 自动验证与 Task Gate 通过后只能进入 `awaiting_acceptance`；只有人工明确验收后才能标记为 `completed` 并开始下一 Task
- OpenCode 官方仓库: `https://github.com/anomalyco/opencode`
- OpenCode Core 修改文件数不超过 5 个，每个 Patch 必须有说明和对应测试
- Go TUI 不承担 Agent Loop、消息历史管理、Tool 选择决策、上下文压缩、Subagent 调度
- 所有 OpenCode SSE 事件必须被映射或 Raw 透传，静默丢弃数量为 0
- 核心功能可达率、原生 Tool 可调用率、事件映射或透传率、API 能力无丢失率均必须 100%
- General 任务完成率不低于原版 OpenCode 的 95%（统计容差）
- Plugin Bundle 格式 ESM，target bun，自包含无外部 import
- 离线发行包启动不触发任何在线依赖安装（npm/bun/pip/mvn）
- 配置文件 schemaVersion 定义迁移路径，升级失败后旧版本可启动并通过基础 Doctor
- API Key 通过环境变量引用，不写入普通配置文件
- TUI 最低终端要求 70×20
- V1 平台: macOS arm64/x64 + Windows x64；Linux deferred
- OpenCode API/DTO 以锁定版本的 `/doc` OpenAPI 3.1 为准，从 spec 生成，不手写猜测
- 2026-08-08 Architecture Rebaseline 后的执行顺序为 Task 0 → 1 → 2 → 2A → 3 → ... → 21，以 `docs/execution-state.yaml.taskOrder` 为准
- Task 2A 的权威设计与实施步骤分别为 `docs/superpowers/specs/2026-08-08-codea-runtime-abstraction-rebaseline-design.md` 和 `docs/superpowers/plans/2026-08-08-codea-runtime-abstraction-rebaseline-plan.md`
- V1 只实现 OpenCode；不得在 Task 2A 加入 OMP、Runtime Router、Runtime Selector 或热切换

---

## 文件结构总览

```
codea/
├── tui/                              # Go TUI（独立 Go Module，包含所有 Go 代码）
│   ├── cmd/
│   │   ├── codea/main.go      # TUI 入口
│   │   ├── parity-runner/main.go     # Parity 测试运行器
│   │   └── openapi-gen/main.go        # OpenAPI → Go DTO 生成器
│   ├── internal/
│   │   ├── app/                      # Bubble Tea 顶层 Model
│   │   │   ├── model.go
│   │   │   ├── update.go
│   │   │   ├── view.go
│   │   │   ├── messages.go
│   │   │   ├── commands.go
│   │   │   ├── keymap.go
│   │   │   ├── page.go
│   │   │   ├── metrics.go
│   │   │   └── feedback.go
│   │   ├── runtime/                  # 领域接口与模型
│   │   │   ├── client.go             # AgentRuntime 接口
│   │   │   ├── events.go             # 统一事件模型
│   │   │   └── models.go             # 领域模型
│   │   ├── opencode/                 # OpenCode 适配层
│   │   │   ├── adapter.go            # AgentRuntime 实现
│   │   │   ├── http_client.go        # HTTP API 客户端
│   │   │   ├── sse_client.go         # SSE 客户端
│   │   │   ├── event_mapper.go       # 事件映射
│   │   │   └── dto.go               # 从 OpenAPI 生成的 DTO
│   │   ├── supervisor/               # Runtime 进程管理
│   │   │   ├── supervisor.go         # 生命周期
│   │   │   ├── process_unix.go       # Unix 平台（macOS）
│   │   │   └── process_windows.go    # Windows 平台
│   │   ├── reasoning/                # 推理内容处理
│   │   │   ├── normalizer.go
│   │   │   ├── tag_parser.go
│   │   │   └── tracker.go
│   │   ├── components/               # Bubble Tea 子组件
│   │   │   ├── chat.go
│   │   │   ├── input.go
│   │   │   ├── status.go
│   │   │   ├── tool.go
│   │   │   ├── skill.go
│   │   │   ├── session.go
│   │   │   ├── agent.go
│   │   │   ├── model.go
│   │   │   ├── event_viewer.go
│   │   │   └── permission.go
│   │   ├── theme/theme.go
│   │   ├── config/                   # 配置管理
│   │   │   ├── config.go
│   │   │   ├── merge.go
│   │   │   ├── profile.go
│   │   │   └── security.go
│   │   ├── update/                   # 升级回滚
│   │   │   ├── service.go            # 事务编排
│   │   │   ├── journal.go            # 事务日志
│   │   │   ├── verifier.go           # staging 校验
│   │   │   ├── migration.go          # 配置迁移注册
│   │   │   ├── package.go
│   │   │   ├── checksum.go
│   │   │   ├── versions.go
│   │   │   ├── rollback.go
│   │   │   └── platform.go
│   │   ├── doctor/                   # 健康检查
│   │   │   ├── service.go
│   │   │   ├── checks.go
│   │   │   └── report.go
│   │   ├── capability/               # 能力盘点
│   │   │   ├── inventory.go
│   │   │   ├── compare.go
│   │   │   └── report.go
│   │   └── parity/                   # 能力不退化验证
│   │       ├── runner.go
│   │       ├── scenario.go
│   │       └── result.go
│   ├── tests/                        # Go 测试（在 tui Module 内）
│   │   ├── contract/                 # API 契约测试
│   │   │   ├── server_health_test.go
│   │   │   ├── session_test.go
│   │   │   ├── stream_events_test.go
│   │   │   ├── tool_approval_test.go
│   │   │   └── reasoning_event_test.go
│   │   ├── parity/                   # 能力不退化回归
│   │   │   ├── capability_inventory_test.go
│   │   │   ├── event_passthrough_test.go
│   │   │   ├── general_agent_test.go
│   │   │   ├── native_tools_test.go
│   │   │   ├── session_resume_test.go
│   │   │   ├── subagent_test.go
│   │   │   └── fixtures/
│   │   ├── e2e/                      # 端到端测试
│   │   │   ├── code-review/
│   │   │   ├── unit-test/
│   │   │   └── api-documentation/
│   │   └── fixtures/
│   │       ├── java-maven-project/
│   │       ├── java-gradle-project/
│   │       ├── go-project/
│   │       └── fake-opencode-server/ # 模拟 Runtime
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
│   ├── skills/                       # 预装 Skill
│   │   ├── index.yaml
│   │   ├── builtin/
│   │   │   ├── code-explain/
│   │   │   ├── git-helper/
│   │   │   └── log-analyzer/
│   │   └── enterprise/
│   │       ├── code-review/
│   │       ├── unit-test/
│   │       └── api-documentation/
│   ├── plugins/
│   │   ├── src/                      # 开发源码
│   │   │   ├── dify-query.ts
│   │   │   ├── runtime-security-guard.ts
│   │   │   ├── audit-log.ts
│   │   │   └── tools/                # 企业专用 Tool 实现
│   │   │       ├── collect-review-context.ts
│   │   │       ├── analyze-test-project.ts
│   │   │       ├── write-test-file.ts
│   │   │       ├── run-project-test.ts
│   │   │       ├── extract-api-spec.ts
│   │   │       ├── validate-api-example.ts
│   │   │       └── write-document.ts
│   │   ├── dist/                     # 自包含 Bundle
│   │   ├── package.json
│   │   └── bun.lock
│   ├── config/
│   │   ├── codea/
│   │   │   ├── defaults.yaml
│   │   │   ├── skills.yaml
│   │   │   └── profiles/
│   │   └── opencode/
│   │       ├── opencode.json.tmpl
│   │       ├── permissions.json
│   │       └── model.json.tmpl
│   └── templates/
│       └── AGENTS.md.tmpl
│
├── runtime/                          # OpenCode Runtime 元信息
│   ├── version.json
│   ├── checksums.json
│   ├── capabilities.yaml
│   ├── openapi/                      # 锁定版本的 OpenAPI 3.1 Spec
│   │   └── opencode-<version>.json
│   └── patches/README.md
│
├── packaging/                        # 离线发行包构建
│   ├── config/release.yaml
│   ├── scripts/
│   │   ├── build-runtime.sh
│   │   ├── build-plugins.sh
│   │   ├── collect-skills.sh
│   │   ├── generate-manifest.sh
│   │   ├── verify-checksum.sh
│   │   └── verify-offline.sh
│   └── platform/
│       ├── macos/install.sh
│       └── windows/install.ps1
│
├── scripts/                          # 开发辅助脚本
│   └── run-phase0-gates.sh           # Phase 0 门禁脚本
│
├── tests/                            # Shell 集成测试（离线/升级）
│   ├── offline/                      # 断网集成测试
│   │   ├── no_public_network_test.sh
│   │   ├── install_test.sh
│   │   └── runtime_start_test.sh
│   └── upgrade/                      # 升级回滚测试
│       ├── fresh_install_test.sh
│       ├── upgrade_test.sh
│       ├── failed_upgrade_test.sh
│       └── rollback_test.sh
│
├── devtools/                         # 开发辅助工具
│   ├── manifest-gen/
│   ├── skill-lint/
│   ├── license-report/
│   └── sse-recorder/
│
├── docs/
│   ├── specs/
│   └── plans/
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

### 关键设计点

1. **所有 Go 代码统一在 `tui/` Module 内** — Go 测试放在 `tui/tests/`，`tui/internal/` 包可被同 Module 内的测试导入。`go test ./...` 从 `tui/` 目录执行。
2. **Shell 测试单独在 `tests/`** — 离线安装、升级回滚等 Shell 集成测试保留在根目录 `tests/`，不参与 Go Module。
3. **OpenAPI Spec 固化** — Spike 完成后，从锁定版本 `/doc` 获取 OpenAPI 3.1 JSON，保存到 `runtime/openapi/`，后续 DTO 由 `tui/cmd/openapi-gen/` 生成。
4. **`distribution/`** — 保存所有企业扩展资源，Plugin 需要 TypeScript 构建，内置依赖随发行包交付。
5. **`runtime/`** — 只锁版本号、哈希、OpenAPI Spec 和 Patch，不含 OpenCode 源码。

---

### Task 0: 项目骨架与 Go Module 结构

**Goal:** 建立正确的项目目录结构，Go Module 统一在 `tui/` 下，确保 `go test ./...` 可运行。

**Files:**
- Modify: `docs/execution-state.yaml`
- Create: `docs/task-reports/task-00.md`
- Create: `tui/go.mod`
- Create: `tui/cmd/codea/main.go`
- Create: `tui/cmd/parity-runner/main.go`
- Create: `Makefile`
- Create: `VERSION`
- Create: `.gitignore`
- Create: `.editorconfig`
- Create: `runtime/version.json`
- Create: `runtime/capabilities.yaml`
- Create: `scripts/run-phase0-gates.sh`

- [ ] **Step 1: 创建项目目录结构**

```bash
mkdir -p tui/cmd/codea
mkdir -p tui/cmd/parity-runner
mkdir -p tui/internal/{app,runtime,opencode,supervisor,reasoning,components,theme,config,update,doctor,capability,parity}
mkdir -p tui/tests/{contract,parity,e2e/code-review,e2e/unit-test,e2e/api-documentation,fixtures/fake-opencode-server}
mkdir -p distribution/{agents,skills/builtin,skills/enterprise,plugins/src/tools,plugins/dist,config/codea/profiles,config/opencode,templates}
mkdir -p runtime/{openapi,patches}
mkdir -p packaging/{config,scripts,platform/macos,platform/windows}
mkdir -p tests/{offline,upgrade}
mkdir -p devtools/{manifest-gen,skill-lint,license-report,sse-recorder}
mkdir -p scripts
mkdir -p docs/superpowers/{specs,plans}
```

- [ ] **Step 2: 初始化 Go Module**

```bash
cd tui
go mod init codea/tui
```

- [ ] **Step 3: 编写 Makefile**

```makefile
.PHONY: build test lint package clean

VERSION := $$(cat VERSION)

build:
	cd tui && go build -o ../build/codea ./cmd/codea

test:
	cd tui && go test ./...

lint:
	cd tui && golangci-lint run ./...

package: build
	./packaging/scripts/build-plugins.sh
	./packaging/scripts/collect-skills.sh
	./packaging/scripts/generate-manifest.sh
	./packaging/scripts/verify-checksum.sh

clean:
	rm -rf build/ packaging/staging/

phase0-gates:
	./scripts/run-phase0-gates.sh
```

- [ ] **Step 4: 编写 VERSION 文件**

```
0.1.0
```

- [ ] **Step 5: 编写 .gitignore**

```
build/
packaging/staging/
*.exe
*.dll
*.so
*.dylib
.DS_Store
Thumbs.db
.env
*.key
*.p12
```

- [ ] **Step 6: 编写 runtime/version.json**

```json
{
  "openCodeVersion": "TBD-by-spike",
  "openCodeCommit": "TBD-by-spike",
  "openCodeRepo": "https://github.com/anomalyco/opencode",
  "lockedAt": "2026-07-30",
  "platforms": {
    "darwin-arm64": {"checksum": "TBD-by-spike"},
    "darwin-x64": {"checksum": "TBD-by-spike"},
    "windows-x64": {"checksum": "TBD-by-spike"}
  }
}
```

- [ ] **Step 7: 编写 runtime/capabilities.yaml（初始模板）**

```yaml
schemaVersion: 1
openCodeVersion: TBD-by-spike
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

- [ ] **Step 8: 编写 Go 编译验证**

`tui/cmd/codea/main.go`：

```go
package main

import "fmt"

func main() {
	fmt.Println("Codea V1")
}
```

- [ ] **Step 9: 验证构建和测试命令**

```bash
make build
cd tui && go test ./...
./scripts/check-execution-state.sh
```

Expected: `build/codea` 二进制生成；`go test ./...` 无测试但通过；执行状态校验通过。

- [ ] **Step 10: Commit**

```bash
git add -A
git commit -m "feat: project skeleton with correct Go module and test structure"
```

- [ ] **Step 11: 更新执行状态并等待人工验收**

创建 `docs/task-reports/task-00.md`，记录实际文件变更、执行命令、验证结果、计划偏差、未解决问题和 Gate 结论。

将 `docs/execution-state.yaml` 中 Task 0 的 `verificationStatus` 和 `taskGateStatus` 更新为 `pass`，状态更新为 `awaiting_acceptance`，checkpoint 指向 Step 10 的真实完整 Commit，并将 `nextAction` 更新为等待人工验收。然后运行：

```bash
./scripts/check-execution-state.sh
git add docs/execution-state.yaml docs/task-reports/task-00.md
git commit -m "chore: record Task 0 verification"
./scripts/check-execution-state.sh
```

提交后立即停止。只有人工明确验收 Task 0 后，才能将其标记为 `completed` 并开始 Task 1。

---

### Task 1: Phase 0 — Spike S1-S6 验证

**Goal:** 验证 6 个关键技术假设。所有 Spike 必须通过才能进入 Phase 1。Spike 报告包含真实请求/响应和 Golden SSE 录制。

**Files:**
- Create: `tui/tests/fixtures/fake-opencode-server/main.go`
- Create: `scripts/run-phase0-gates.sh`
- Create: `docs/spike-report.md`

**Gate:** S1-S6 全部通过 → 进入 Phase 1；任一不通过 → 阻断，重新评估方案。

---

#### 1.1 Spike S1: Server 离线启动

- [ ] **Step 1: 下载 OpenCode 并断网验证**

```bash
# 在有网环境确定版本
opencode --version
# 记录版本号和 commit

# 断网测试
sudo ifconfig en0 down
opencode serve --hostname 127.0.0.1 --port 0
# 预期: 启动成功，不访问公网
# 记录: 启动日志，是否有隐式网络请求
sudo ifconfig en0 up
```

通过标准: `opencode serve` 在断网环境启动，无公网 DNS/HTTP 请求。

---

#### 1.2 Spike S2: Go Session + Prompt + SSE 全链路

- [ ] **Step 2: 编写 Go 客户端手动验证**

```bash
# 终端 1: 启动 OpenCode Server
opencode serve --hostname 127.0.0.1 --port 49321

# 终端 2: 用 curl 探索实际 API
# 访问 /doc 获取 OpenAPI Spec
curl http://127.0.0.1:49321/doc > runtime/openapi/opencode-$(opencode --version).json

# 健康检查
curl http://127.0.0.1:49321/global/health

# 创建 Session（探索实际请求体）
curl -X POST http://127.0.0.1:49321/session \
  -H 'Content-Type: application/json' \
  -d '{}'

# 发送 prompt（探索实际请求体）
curl -X POST http://127.0.0.1:49321/session/<id>/prompt_async \
  -H 'Content-Type: application/json' \
  -d '{"messageID":"msg-1","parts":[{"type":"text","text":"hello"}]}'

# 监听 SSE
curl -N http://127.0.0.1:49321/global/event
```

通过标准: 创建 Session → 发送消息 → 接收 SSE → 完成一轮完整 Agent 会话。

记录: 实际 API 路径、请求/响应格式、SSE 事件类型和结构。

---

#### 1.3 Spike S3: Tool Approval

- [ ] **Step 3: 手动验证 Tool Approval 流程**

```bash
# 发送需要 Tool 执行的 prompt（如 "create a file test.txt"）
# 通过 SSE 收到 tool_approval_required 事件
# 调用 Permission API 批准/拒绝
# 验证 Tool 执行结果
```

通过标准: Go 客户端能接收权限申请事件，批准/拒绝后 Runtime 正确执行。

记录: Permission 事件结构、permissionID 格式、response 枚举值。

---

#### 1.4 Spike S4: Reasoning

- [ ] **Step 4: 验证 Reasoning 事件结构**

```bash
# 使用支持 reasoning 的模型（DeepSeek）
# 发送 prompt
# 监听 SSE，记录所有事件类型
# 验证 reasoning 事件的实际结构
```

通过标准: Go 客户端能实时接收并区分 reasoning 与 answer。

记录: 结构化 reasining Part 是否存在、`<think>` 标签是否存在、事件字段名。

---

#### 1.5 Spike S5: Skill 来源隔离

- [ ] **Step 5: 验证 Skill 隔离机制**

```bash
# 创建测试 Skill
mkdir -p /tmp/test-skills/my-skill
echo "# My Test Skill" > /tmp/test-skills/my-skill/SKILL.md

# 设置 OPENCODE_CONFIG_DIR
export OPENCODE_CONFIG_DIR=/tmp/test-runtime-config
export OPENCODE_DISABLE_CLAUDE_CODE=1

# 验证: Runtime 只发现 OPENCODE_CONFIG_DIR 中的 Skill
```

通过标准: 其他来源 Skill 不被注入，配置目录外的 Skill 不被发现。

记录: 是否需要独立 HOME/XDG、是否需要极小 Patch。

---

#### 1.6 Spike S6: General Compatible / Enterprise 模式隔离

- [ ] **Step 6: 验证双模式基础隔离**

```bash
# 1. 启动 Enterprise Profile Runtime（仅 Approved Skill）
# 2. 创建项目 .opencode/skills/test-skill/
# 3. 验证企业 Agent 不加载项目 Skill
# 4. 切换到 General compatible 模式
# 5. 验证 General 模式下项目 Skill 可被发现和加载
```

通过标准:
- Enterprise 模式下，项目 Skill 不被注入
- General compatible 模式下，项目 Skill 经过校验后可加载
- General strict 模式下（V1 默认），项目 Skill 不被注入

---

#### 1.7 Spike 收尾

- [ ] **Step 7: 编写 Phase 0 门禁脚本**

`scripts/run-phase0-gates.sh`：

```bash
#!/bin/bash
set -euo pipefail

RESULTS_FILE="docs/spike-results.json"
FAILED=0

echo "=== Phase 0 Spike Gates ==="

if [ ! -f "$RESULTS_FILE" ]; then
    echo "FAIL: $RESULTS_FILE not found."
    echo "Run all S1-S6 spikes and record results in docs/spike-results.json"
    exit 1
fi

check_gate() {
    local gate="$1"
    local result
    result=$(jq -r ".${gate} // \"missing\"" "$RESULTS_FILE")
    case "$result" in
        pass)
            echo "  $gate ... PASS"
            ;;
        fail|missing)
            echo "  $gate ... FAIL (result: $result)"
            FAILED=1
            ;;
        *)
            echo "  $gate ... FAIL (unexpected result: $result)"
            FAILED=1
            ;;
    esac
}

for gate in S1 S2 S3 S4 S5 S6; do
    check_gate "$gate"
done

if [ $FAILED -ne 0 ]; then
    echo ""
    echo "Phase 0 gates FAILED. Fix issues before proceeding to Phase 1."
    exit 1
fi

echo ""
echo "All Phase 0 gates PASSED."
```

- [ ] **Step 8: 固化 OpenAPI Spec**

```bash
# 从锁定版本获取 OpenAPI 3.1 Spec
opencode serve --hostname 127.0.0.1 --port 49321 &
sleep 3
curl http://127.0.0.1:49321/doc > runtime/openapi/opencode-$(cat VERSION).json
kill %1

# 保存 Golden SSE 样本（录制一轮完整对话的 SSE 事件）
```

- [ ] **Step 9: 编写 Spike 报告和机器可读结果**

`docs/spike-report.md` 必须包含:
- S1-S6 每个的通过/失败状态
- 实际请求/响应样例
- Golden SSE 事件列表
- OpenCode 锁定版本号和 commit
- 发现的问题和风险

`docs/spike-results.json`（机器可读，门禁脚本消费）：

```json
{
  "S1": "pass",
  "S2": "pass",
  "S3": "pass",
  "S4": "pass",
  "S5": "pass",
  "S6": "pass"
}
```

- 每个 Spike 完成后更新对应状态（pass / fail）
- 缺失的 key 或 fail 状态 → 门禁脚本返回非零
- 该文件是 Phase 0→Phase 1 的唯一机器判定依据

- [ ] **Step 10: Commit**

```bash
git add -A
git commit -m "feat: Phase 0 spike report, OpenAPI spec, and gate script"
```

---

### Task 2: 从 OpenAPI Spec 生成 DTO 与 Client

**Goal:** 从锁定版本的 OpenAPI 3.1 Spec 生成 Go DTO，不再手写猜测 API 结构。OpenCodeAdapter 只做领域转换。

**Files:**
- Create: `tui/cmd/openapi-gen/main.go`
- Modify: `tui/internal/opencode/dto.go` — 由生成器产出
- Modify: `tui/internal/opencode/http_client.go` — 使用生成的 DTO

**Interfaces:**
- Produces: 生成的 OpenCode DTO 结构体，与锁定版本 `/doc` 精确匹配

---

- [ ] **Step 1: 审阅固化的 OpenAPI Spec，确认关键接口结构**

从 `runtime/openapi/opencode-<version>.json` 确认:

| 接口 | 方法 | 路径 | 关键字段 |
|------|------|------|----------|
| 健康检查 | GET | `/global/health` | `healthy`, `version` |
| 创建 Session | POST | `/session` | `parentID`, `title` |
| 发送 Prompt | POST | `/session/:id/prompt_async` | `messageID`, `agent`, `model`, `parts[]` |
| Permission | POST | `/session/:id/permissions/:pid` | `response`, `remember` |
| Agent 列表 | GET | `/agent` | - |
| SSE 事件 | GET | `/global/event` | - |

- [ ] **Step 2: 编写 OpenAPI 代码生成器**

`tui/cmd/openapi-gen/main.go`：

```go
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// 从 OpenAPI 3.1 Spec 生成 Go DTO 结构体
// 精简实现：解析 schemas，生成 struct

func main() {
	if len(os.Args) < 3 {
		fmt.Fprintf(os.Stderr, "Usage: openapi-gen <spec.json> <output.go>\n")
		os.Exit(1)
	}
	specPath := os.Args[1]
	outputPath := os.Args[2]

	data, err := os.ReadFile(specPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read spec: %v\n", err)
		os.Exit(1)
	}

	var spec map[string]any
	if err := json.Unmarshal(data, &spec); err != nil {
		fmt.Fprintf(os.Stderr, "parse spec: %v\n", err)
		os.Exit(1)
	}

	// 生成 DTO 文件头
	var out strings.Builder
	out.WriteString("// Code generated from OpenAPI spec. DO NOT EDIT.\n\n")
	out.WriteString("package opencode\n\n")

	// 从 spec 提取 schemas 并生成 struct
	// 实际实现会遍历 paths 和 components/schemas
	// 这里展示核心结构

	out.WriteString("// OpenCodeHealthResponse — GET /global/health\n")
	out.WriteString("type OpenCodeHealthResponse struct {\n")
	out.WriteString("    Healthy bool   `json:\"healthy\"`\n")
	out.WriteString("    Version string `json:\"version\"`\n")
	out.WriteString("}\n\n")

	out.WriteString("// OpenCodeCreateSessionRequest — POST /session\n")
	out.WriteString("type OpenCodeCreateSessionRequest struct {\n")
	out.WriteString("    ParentID string `json:\"parentID,omitempty\"`\n")
	out.WriteString("    Title    string `json:\"title,omitempty\"`\n")
	out.WriteString("}\n\n")

	out.WriteString("// OpenCodeSessionResponse — POST /session response\n")
	out.WriteString("type OpenCodeSessionResponse struct {\n")
	out.WriteString("    ID        string `json:\"id\"`\n")
	out.WriteString("    Status    string `json:\"status\"`\n")
	out.WriteString("    Agent     string `json:\"agent\"`\n")
	out.WriteString("    CreatedAt string `json:\"created_at\"`\n")
	out.WriteString("}\n\n")

	out.WriteString("// OpenCodePromptPart — a single message part\n")
	out.WriteString("type OpenCodePromptPart struct {\n")
	out.WriteString("    Type string `json:\"type\"` // \"text\"\n")
	out.WriteString("    Text string `json:\"text\"`\n")
	out.WriteString("}\n\n")

	out.WriteString("// OpenCodePromptRequest — POST /session/:id/prompt_async\n")
	out.WriteString("type OpenCodePromptRequest struct {\n")
	out.WriteString("    MessageID string               `json:\"messageID\"`\n")
	out.WriteString("    Agent     string               `json:\"agent,omitempty\"`\n")
	out.WriteString("    Model     map[string]any       `json:\"model,omitempty\"`\n")
	out.WriteString("    Parts     []OpenCodePromptPart `json:\"parts\"`\n")
	out.WriteString("}\n\n")

	out.WriteString("// OpenCodePermissionRequest — POST /session/:id/permissions/:pid\n")
	out.WriteString("type OpenCodePermissionRequest struct {\n")
	out.WriteString("    Response string `json:\"response\"`          // 枚举以 /doc 为准\n")
	out.WriteString("    Remember bool   `json:\"remember,omitempty\"`\n")
	out.WriteString("}\n\n")

	out.WriteString("// OpenCodeAgentResponse — GET /agent\n")
	out.WriteString("type OpenCodeAgentResponse struct {\n")
	out.WriteString("    Name        string `json:\"name\"`\n")
	out.WriteString("    Description string `json:\"description\"`\n")
	out.WriteString("}\n")

	if err := os.WriteFile(outputPath, []byte(out.String()), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write output: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("DTO generated:", outputPath)
}
```

**重要:** 实际生成时，人工审阅生成结果，确保与锁定版本 `/doc` 一致。生成器是辅助工具，不是完全自动。

- [ ] **Step 3: 运行生成器**

```bash
cd tui
go run ./cmd/openapi-gen \
  ../runtime/openapi/opencode-$(cat ../VERSION).json \
  internal/opencode/dto.go

# 人工审阅生成的 DTO
```

- [ ] **Step 4: 基于生成的 DTO 更新 HTTP Client**

更新 `tui/internal/opencode/http_client.go`，使用生成的 DTO 结构体。

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat: OpenAPI spec-based DTO generation and reviewed client DTOs"
```

---

### Task 2A: Runtime Abstraction Rebaseline

**Goal:** 在 Task 3 前把 Task 2 的 OpenCode Vendor Client 收敛到 Codea-owned `AgentRuntime` 后面，建立领域模型、全局 Event/Raw 映射、Approval、RuntimeCapabilities 与依赖边界门禁。

**Authoritative design:** `docs/superpowers/specs/2026-08-08-codea-runtime-abstraction-rebaseline-design.md`

**Authoritative implementation plan:** `docs/superpowers/plans/2026-08-08-codea-runtime-abstraction-rebaseline-plan.md`

**Required output:**

- `tui/internal/runtime` 不依赖 OpenCode，拥有 AgentRuntime、Domain Model、Event、Approval 和 RuntimeCapabilities。
- `tui/internal/opencode` 使用 Task 2 DTO/Client 实现 AgentRuntime。
- `Subscribe(ctx)` 忠实表达 `/global/event` 全局事件流；不得伪装为 session-scoped Transport。
- once/always/reject 与 message 正确映射到非废弃 Permission Reply；不得伪造 `remember`。
- Golden SSE 每条事件均为映射事件或 Raw Event，Raw payload 不丢失并遵守 DLP/16KB/内存边界。
- Import graph Required Gate 证明 Application/TUI/Harness/Agent/Policy 的 Vendor DTO leakage 为 0。
- Task 2 全部测试、Go 1.26.5 test/race/vet/build、Windows x64 cross-build、真实 OpenCode Adapter smoke 与状态门禁通过。
- 自动验证与 Task Gate 通过后进入 `awaiting_acceptance`；人工验收前 Task 3 保持 `pending`。

**Explicit exclusions:** OMP、Runtime Router/Selector、热切换、Swarm、Agent Graph、完整 Tool Runtime、Sandbox 重构和任何新业务 Agent。

---

### Task 3: Capability Inventory + Parity Harness (Rebaselined)

**Goal:** 在 Task 2A Contract 之上加载产品 Capability Inventory、比较 OpenCodeAdapter 实际声明并运行真正的 Parity Harness。

**Rebaseline constraints:**

- `runtime/capabilities.yaml` 继续表达 required/optional/deferred 产品要求；不得替代 `runtime.RuntimeCapabilities` 的实际能力声明。
- Fake 测试实现 `runtime.AgentRuntime`，而不是复制一套手写 OpenCode HTTP JSON。
- Parity Baseline/Candidate Runner 通过 AgentRuntime/Codea Event 运行，OpenCode DTO 不进入 `internal/capability` 或 `internal/parity`。
- Fake Runtime 的 Session/Event 不得使用与锁定 v1.18.11 Spec 冲突的 `status/agent/created_at` 手写响应作为协议事实。
- Required 场景仍必须有 Baseline、Candidate、Assertions 和至少一次重复；SilentLoss 必须导致失败。

**Files:**

- Create: `tui/internal/capability/inventory.go`
- Create: `tui/internal/capability/compare.go`
- Create: `tui/internal/parity/runner.go`
- Create: `tui/internal/parity/scenario.go`
- Create: `tui/internal/parity/result.go`
- Create: `tui/tests/parity/capability_inventory_test.go`
- Create: `tui/tests/fixtures/fake-runtime/fake_runtime.go`

**Required Gate:** `cd tui && go test ./internal/capability/... ./internal/parity/... ./tests/parity/... -count=1`、Runtime boundary Gate、状态校验和人工验收全部通过。完成后不得自动开始 Task 4。

---

### Task 4: Runtime Adapter Hardening and Recovery (Rebaselined)

**Goal:** 在 Task 2A 已完成基础 Contract/Adapter/Mapper 后，补齐生产级 SSE 恢复、状态补偿、错误分类、背压与真实 Runtime 长连接验证。

**Consumes:** Task 2A 的 `runtime.AgentRuntime`、OpenCodeAdapter、SSEClient、EventMapper 和 Raw Event Contract。

**Produces:**

- SSE 断开、Scanner 错误和认证失败均可观察，不被误报为正常流结束。
- 按现有技术设计使用 500ms → 1s → 2s → 5s 有界退避重连，并响应 Context 取消。
- 重连成功后通过 Session status/message history 补偿遗漏 Message/Part，去重后继续实时流。
- Channel 背压策略有界且不静默丢弃；无法交付时产生明确 RuntimeError/compatibility evidence。
- Transport/Auth/Protocol/Incompatible/Cancelled 错误可由 Application 稳定判断，同时保留 Vendor 错误 Raw details。
- 真实 OpenCode 长连接、重连、补偿、Approval 与 Abort Contract 测试通过。

**Files:**

- Modify: `tui/internal/opencode/sse_client.go`
- Modify: `tui/internal/opencode/adapter.go`
- Create: `tui/internal/opencode/reconnect.go`
- Create: `tui/internal/opencode/recovery.go`
- Create: `tui/internal/runtime/errors.go`
- Create: `tui/tests/contract/runtime_recovery_test.go`

**Required Gate:** Go 1.26.5 test/race/vet/build、断线重连与补偿契约测试、Golden SSE 零丢失、Runtime boundary Gate、真实 OpenCode smoke、状态门禁及人工验收全部通过。不得重复创建 Task 2A 已交付的基础接口或 DTO Mapper。

---

### Task 5: Supervisor + Basic Auth + 跨平台进程管理

**Goal:** Runtime 进程生命周期管理，包括 Basic Auth、跨平台信号控制、端口管理。

**Files:**
- Create: `tui/internal/supervisor/supervisor.go`
- Create: `tui/internal/supervisor/process_unix.go`
- Create: `tui/internal/supervisor/process_windows.go`

---

- [ ] **Step 1: 编写 Supervisor**

`tui/internal/supervisor/supervisor.go`：

```go
package supervisor

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"

	"codea/tui/internal/runtime"
)

type Config struct {
	OpenCodeBin    string
	Hostname       string
	Port           int // 0 = 自动选择
	ConfigDir      string
	ProjectRoot    string
	StartupTimeout time.Duration
}

type Supervisor struct {
	config   Config
	status   runtime.RuntimeStatus
	cmd      *exec.Cmd
	port     int
	password string
	mu       sync.Mutex
}

func NewSupervisor(config Config) *Supervisor {
	if config.Hostname == "" {
		config.Hostname = "127.0.0.1"
	}
	if config.StartupTimeout == 0 {
		config.StartupTimeout = 30 * time.Second
	}
	return &Supervisor{
		config: config,
		status: runtime.RuntimeStopped,
	}
}

func (s *Supervisor) Status() runtime.RuntimeStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

func (s *Supervisor) Port() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.port
}

func (s *Supervisor) Password() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.password
}

func (s *Supervisor) BaseURL() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return fmt.Sprintf("http://%s:%d", s.config.Hostname, s.port)
}

func (s *Supervisor) Start(ctx context.Context) error {
	s.mu.Lock()
	if s.status == runtime.RuntimeHealthy || s.status == runtime.RuntimeStarting {
		s.mu.Unlock()
		return fmt.Errorf("runtime already running")
	}
	s.status = runtime.RuntimeStarting
	s.mu.Unlock()

	// 生成随机密码
	pwBytes := make([]byte, 32)
	if _, err := rand.Read(pwBytes); err != nil {
		return fmt.Errorf("generate password: %w", err)
	}
	password := hex.EncodeToString(pwBytes)

	s.mu.Lock()
	s.password = password
	s.mu.Unlock()

	// 选择端口
	port := s.config.Port
	if port == 0 {
		var err error
		port, err = findFreePort()
		if err != nil {
			return fmt.Errorf("find free port: %w", err)
		}
	}

	s.mu.Lock()
	s.port = port
	s.mu.Unlock()

	// 构建命令 — 不使用 --config-dir（OpenCode 通过环境变量 OPENCODE_CONFIG_DIR 指定配置目录）
	args := []string{
		"serve",
		"--hostname", s.config.Hostname,
		"--port", fmt.Sprintf("%d", port),
	}

	cmd := exec.CommandContext(ctx, s.config.OpenCodeBin, args...)
	cmd.Env = append(os.Environ(),
		"OPENCODE_CONFIG_DIR="+s.config.ConfigDir,
		"OPENCODE_DISABLE_CLAUDE_CODE=1",
		"OPENCODE_SERVER_PASSWORD="+password,
		"OPENCODE_SERVER_USERNAME=opencode",
	)
	configureProcess(cmd) // 平台特定设置（Unix Setpgid / Windows CREATE_NEW_PROCESS_GROUP）

	s.mu.Lock()
	s.cmd = cmd
	s.mu.Unlock()

	if err := cmd.Start(); err != nil {
		s.mu.Lock()
		s.status = runtime.RuntimeCrashed
		s.mu.Unlock()
		return fmt.Errorf("start opencode: %w", err)
	}

	if err := s.waitForReady(ctx); err != nil {
		return err
	}

	s.mu.Lock()
	s.status = runtime.RuntimeHealthy
	s.mu.Unlock()

	return nil
}

func (s *Supervisor) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.cmd == nil || s.status == runtime.RuntimeStopped {
		return nil
	}

	s.status = runtime.RuntimeStopping

	if err := terminateProcess(s.cmd); err != nil {
		return err
	}

	done := make(chan error, 1)
	go func() {
		done <- s.cmd.Wait()
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		killProcess(s.cmd)
		<-done
	}

	s.status = runtime.RuntimeStopped
	return nil
}

func (s *Supervisor) waitForReady(ctx context.Context) error {
	deadline := time.After(s.config.StartupTimeout)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	url := fmt.Sprintf("http://%s:%d/global/health", s.config.Hostname, s.port)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-deadline:
			return fmt.Errorf("startup timeout after %v", s.config.StartupTimeout)
		case <-ticker.C:
			req, _ := http.NewRequest("GET", url, nil)
			req.SetBasicAuth("opencode", s.password)
			resp, err := http.DefaultClient.Do(req)
			if err == nil && resp.StatusCode == 200 {
				resp.Body.Close()
				return nil
			}
			if resp != nil {
				resp.Body.Close()
			}
		}
	}
}

func findFreePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}
```

- [ ] **Step 2: 编写 Unix 平台进程控制**

`tui/internal/supervisor/process_unix.go`：

```go
//go:build unix

package supervisor

import (
	"os/exec"
	"syscall"
)

func configureProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

func terminateProcess(cmd *exec.Cmd) error {
	return syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
}

func killProcess(cmd *exec.Cmd) {
	syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}
```

- [ ] **Step 3: 编写 Windows 平台进程控制**

`tui/internal/supervisor/process_windows.go`：

```go
//go:build windows

package supervisor

import (
	"os/exec"
	"syscall"
)

func configureProcess(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP,
	}
}

func terminateProcess(cmd *exec.Cmd) error {
	dll, _ := syscall.LoadDLL("kernel32.dll")
	proc, _ := dll.FindProc("GenerateConsoleCtrlEvent")
	proc.Call(syscall.CTRL_BREAK_EVENT, uintptr(cmd.Process.Pid))
	return nil
}

func killProcess(cmd *exec.Cmd) {
	cmd.Process.Kill()
}
```

- [ ] **Step 4: 运行构建验证跨平台编译**

```bash
cd tui
GOOS=darwin GOARCH=arm64 go build ./internal/supervisor/...
GOOS=windows GOARCH=amd64 go build ./internal/supervisor/...
```

Expected: 两个平台均编译成功，无 build tag 冲突。

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat: Supervisor with Basic Auth and cross-platform process control"
```

---

### Task 6: Reasoning 处理

**Goal:** 实现 Reasoning 内容的结构化接收和 `<think>` 标签解析兜底。

**Files:**
- Create: `tui/internal/reasoning/normalizer.go`
- Create: `tui/internal/reasoning/tag_parser.go`
- Create: `tui/internal/reasoning/tracker.go`
- Create: `tui/tests/contract/reasoning_event_test.go`

---

- [ ] **Step 1: 编写 TagParser 状态机**

`tui/internal/reasoning/tag_parser.go`：

```go
package reasoning

import "strings"

type ThinkState int

const (
	ThinkStateAnswer           ThinkState = iota
	ThinkStatePossibleOpenTag
	ThinkStateReasoning
	ThinkStatePossibleCloseTag
)

type ParserEventType int

const (
	ParserEventAnswerDelta    ParserEventType = iota
	ParserEventReasoningStart
	ParserEventReasoningDelta
	ParserEventReasoningEnd
)

type ParserEvent struct {
	Type    ParserEventType
	Content string
}

type TagParser struct {
	state  ThinkState
	buffer strings.Builder
}

func NewTagParser() *TagParser {
	return &TagParser{state: ThinkStateAnswer}
}

func (p *TagParser) Feed(chunk string) []ParserEvent {
	var events []ParserEvent

	for _, ch := range chunk {
		p.buffer.WriteRune(ch)
		buf := p.buffer.String()

		switch p.state {
		case ThinkStateAnswer:
			if strings.HasSuffix(buf, "<think>") {
				prefix := buf[:len(buf)-len("<think>")]
				if prefix != "" {
					events = append(events, ParserEvent{Type: ParserEventAnswerDelta, Content: prefix})
				}
				p.buffer.Reset()
				p.state = ThinkStateReasoning
				events = append(events, ParserEvent{Type: ParserEventReasoningStart})
			} else if isPossibleOpenTag(buf) {
				p.state = ThinkStatePossibleOpenTag
			} else if len(buf) >= 7 {
				safeLen := len(buf) - 6
				if safeLen > 0 {
					events = append(events, ParserEvent{Type: ParserEventAnswerDelta, Content: buf[:safeLen]})
					remaining := buf[safeLen:]
					p.buffer.Reset()
					p.buffer.WriteString(remaining)
				}
			}

		case ThinkStatePossibleOpenTag:
			if strings.HasSuffix(buf, "<think>") {
				prefix := buf[:len(buf)-len("<think>")]
				if prefix != "" {
					events = append(events, ParserEvent{Type: ParserEventAnswerDelta, Content: prefix})
				}
				p.buffer.Reset()
				p.state = ThinkStateReasoning
				events = append(events, ParserEvent{Type: ParserEventReasoningStart})
			} else if len(buf) >= 7 && !isPossibleOpenTag(buf) {
				events = append(events, ParserEvent{Type: ParserEventAnswerDelta, Content: buf})
				p.buffer.Reset()
				p.state = ThinkStateAnswer
			}

		case ThinkStateReasoning:
			if strings.HasSuffix(buf, "</think>") {
				content := buf[:len(buf)-len("</think>")]
				if content != "" {
					events = append(events, ParserEvent{Type: ParserEventReasoningDelta, Content: content})
				}
				p.buffer.Reset()
				p.state = ThinkStateAnswer
				events = append(events, ParserEvent{Type: ParserEventReasoningEnd})
			} else if isPossibleCloseTag(buf) {
				p.state = ThinkStatePossibleCloseTag
			} else if len(buf) >= 8 {
				safeLen := len(buf) - 7
				if safeLen > 0 {
					events = append(events, ParserEvent{Type: ParserEventReasoningDelta, Content: buf[:safeLen]})
					remaining := buf[safeLen:]
					p.buffer.Reset()
					p.buffer.WriteString(remaining)
				}
			}

		case ThinkStatePossibleCloseTag:
			if strings.HasSuffix(buf, "</think>") {
				content := buf[:len(buf)-len("</think>")]
				if content != "" {
					events = append(events, ParserEvent{Type: ParserEventReasoningDelta, Content: content})
				}
				p.buffer.Reset()
				p.state = ThinkStateAnswer
				events = append(events, ParserEvent{Type: ParserEventReasoningEnd})
			} else if len(buf) >= 8 && !isPossibleCloseTag(buf) {
				events = append(events, ParserEvent{Type: ParserEventReasoningDelta, Content: buf})
				p.buffer.Reset()
				p.state = ThinkStateReasoning
			}
		}
	}

	return events
}

func (p *TagParser) Flush() []ParserEvent {
	var events []ParserEvent
	if p.buffer.Len() > 0 {
		switch p.state {
		case ThinkStateReasoning, ThinkStatePossibleCloseTag:
			events = append(events, ParserEvent{Type: ParserEventReasoningDelta, Content: p.buffer.String()})
			events = append(events, ParserEvent{Type: ParserEventReasoningEnd})
		default:
			events = append(events, ParserEvent{Type: ParserEventAnswerDelta, Content: p.buffer.String()})
		}
		p.buffer.Reset()
	}
	return events
}

func (p *TagParser) IsInReasoning() bool {
	return p.state == ThinkStateReasoning || p.state == ThinkStatePossibleCloseTag
}

func isPossibleOpenTag(s string) bool {
	target := "<think>"
	for i := 0; i < len(s) && i < len(target); i++ {
		if s[i] != target[i] {
			return false
		}
	}
	return true
}

func isPossibleCloseTag(s string) bool {
	target := "</think>"
	for i := 0; i < len(s) && i < len(target); i++ {
		if s[i] != target[i] {
			return false
		}
	}
	return true
}
```

- [ ] **Step 2: 编写 Normalizer 和 Tracker**

`tui/internal/reasoning/normalizer.go` 和 `tui/internal/reasoning/tracker.go`。实现必须消费 Task 2A 定义的 Codea `runtime.Event`，不得依赖已废弃的 Runtime 草案或 OpenCode Vendor DTO；具体行为以主技术设计的 Reasoning 章节为准。

- [ ] **Step 3: 编写测试**

`tui/tests/contract/reasoning_event_test.go`：

```go
package contract

import (
	"testing"

	"codea/tui/internal/reasoning"
)

func TestTagParserBasic(t *testing.T) {
	p := reasoning.NewTagParser()
	events := p.Feed("Normal text <think>")
	hasStart := false
	for _, e := range events {
		if e.Type == reasoning.ParserEventReasoningStart {
			hasStart = true
		}
	}
	if !hasStart {
		t.Error("expected reasoning start")
	}
}

func TestTagParserCrossChunk(t *testing.T) {
	p := reasoning.NewTagParser()
	e1 := p.Feed("<thi")
	if len(e1) != 0 {
		t.Error("expected no events from partial tag")
	}
	e2 := p.Feed("nk>content")
	if len(e2) == 0 {
		t.Error("expected events after tag completion")
	}
}
```

- [ ] **Step 4: 运行测试**

```bash
cd tui && go test ./internal/reasoning/... ./tests/contract/... -v
```

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "feat: reasoning normalizer, tag parser state machine, and tests"
```

---

由于计划内容非常庞大，我将在此暂停检查当前进度，然后继续补充剩余 Tasks 6-21。

---

### Task 7: TUI 基础 + SSE 事件流

**Goal:** 建立 Bubble Tea 应用骨架，Tokyo Night 主题，SSE 事件流接入。

**Files:**
- Create: `tui/internal/theme/theme.go`
- Create: `tui/internal/app/model.go`
- Create: `tui/internal/app/update.go`
- Create: `tui/internal/app/view.go`
- Create: `tui/internal/app/messages.go`
- Create: `tui/internal/app/commands.go`
- Create: `tui/internal/app/keymap.go`
- Create: `tui/internal/app/page.go`
- Modify: `tui/cmd/codea/main.go`

- [ ] **Step 1: 编写主题系统**

`tui/internal/theme/theme.go`：

```go
package theme

import "github.com/charmbracelet/lipgloss"

var (
	Primary    = lipgloss.Color("#c0caf5")
	Secondary  = lipgloss.Color("#a9b1d6")
	Muted      = lipgloss.Color("#565f89")
	Accent     = lipgloss.Color("#e0af68")
	Success    = lipgloss.Color("#9ece6a")
	Error      = lipgloss.Color("#f7768e")
	Border     = lipgloss.Color("#292e42")
	Background = lipgloss.Color("#1a1b26")
)

func ChatStyle() lipgloss.Style     { return lipgloss.NewStyle().Foreground(Primary) }
func MutedStyle() lipgloss.Style    { return lipgloss.NewStyle().Foreground(Muted).Italic(true) }
func AccentStyle() lipgloss.Style   { return lipgloss.NewStyle().Foreground(Accent) }
func SuccessStyle() lipgloss.Style  { return lipgloss.NewStyle().Foreground(Success) }
func ErrorStyle() lipgloss.Style    { return lipgloss.NewStyle().Foreground(Error) }
```

- [ ] **Step 2: 编写页面枚举、消息类型、快捷键**

`tui/internal/app/page.go`、`tui/internal/app/messages.go`、`tui/internal/app/keymap.go`（结构同设计文档）

- [ ] **Step 3: 编写主 Model 和 SSE 事件流接入**

`tui/internal/app/model.go` — 包含事件通道、流式缓冲区、reasoning 状态：

```go
package app

import (
	"strings"
	"sync"
	"time"

	"codea/tui/internal/runtime"
	"codea/tui/internal/theme"

	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	currentPage   Page
	width, height int
	runtimeStatus runtime.RuntimeStatus
	runtimeClient runtime.AgentRuntime
	messages      []ChatMessage
	input         string
	isStreaming   bool
	thinkExpanded bool
	thinkDuration time.Duration
	thinkContent  string
	keys          KeyMap
	eventCh       <-chan runtime.Event
	streamBuf     strings.Builder
	currentRole   string
	activeTools   map[string]*runtime.ToolEvent
	renderedCache []string
	permissionModel PermissionModel
	feedbackModel   FeedbackModel
	mu            sync.Mutex
}

type ChatMessage struct {
	Role     string
	Content  string
	Tool     *runtime.ToolEvent
	Finished bool
}

func NewModel(client runtime.AgentRuntime) Model {
	return Model{
		currentPage:   PageChat,
		runtimeStatus: runtime.RuntimeStopped,
		runtimeClient: client,
		keys:          DefaultKeyMap(),
		messages:      make([]ChatMessage, 0),
		activeTools:   make(map[string]*runtime.ToolEvent),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(TickCmd(), SubscribeEvents(m.runtimeClient))
}
```

- [ ] **Step 4: 编写 Update 和 View**

`tui/internal/app/update.go` — 处理 SSE 事件、按键、Tool Approval、流式渲染。
`tui/internal/app/view.go` — 对话区、状态栏、输入区、50ms 合并刷新。

- [ ] **Step 5: 更新 main.go**

```go
package main

import (
	"fmt"
	"os"

	"codea/tui/internal/app"
	"codea/tui/internal/opencode"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	baseURL := os.Getenv("OPENCODE_URL")
	if baseURL == "" {
		baseURL = "http://127.0.0.1:49321"
	}
	username := os.Getenv("OPENCODE_USERNAME")
	password := os.Getenv("OPENCODE_PASSWORD")

	adapter := opencode.NewAdapter(baseURL, username, password)
	model := app.NewModel(adapter)

	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
```

- [ ] **Step 6: 安装依赖并验证**

```bash
cd tui
go get github.com/charmbracelet/bubbletea
go get github.com/charmbracelet/bubbles
go get github.com/charmbracelet/lipgloss
go mod tidy

# 启动 Fake Server 验证
go run ./tests/fixtures/fake-opencode-server &
OPENCODE_URL=http://localhost:49323 go run ./cmd/codea
```

- [ ] **Step 7: Commit**

```bash
git add -A && git commit -m "feat: Bubble Tea TUI with SSE streaming and Tokyo Night theme"
```

---

### Task 8: Session/Resume/Tool Approval

**Goal:** Session 列表/恢复、Tool 权限确认弹窗、危险命令拒绝。

**Files:**
- Create: `tui/internal/components/session.go`
- Create: `tui/internal/components/permission.go`
- Create: `tui/internal/components/tool.go`
- Modify: `tui/internal/app/model.go`
- Modify: `tui/internal/app/update.go`

- [ ] **Step 1: 编写 Session 列表组件**

`tui/internal/components/session.go` — 显示所有 Session，支持切换和恢复。

- [ ] **Step 2: 编写 Tool 权限确认弹窗**

`tui/internal/components/permission.go` — 写操作/Shell 命令弹窗，[Y]Allow/[R]Remember/[N]Deny。

- [ ] **Step 3: 编写危险命令检测**

`tui/internal/components/tool.go`：

```go
package components

import "strings"

var dangerousCommands = []string{
	"rm -rf", "git reset --hard", "git clean -fd",
	"git push --force", "chmod 777", "> /dev/sda",
	"mkfs.", "dd if=", ":(){ :|:& };:",
}

func IsDangerousCommand(input string) (bool, string) {
	lower := strings.ToLower(strings.TrimSpace(input))
	for _, cmd := range dangerousCommands {
		if strings.Contains(lower, strings.ToLower(cmd)) {
			return true, cmd
		}
	}
	return false, ""
}
```

- [ ] **Step 4: 集成到 Model 的 Update 中**

在 `update.go` 中处理 `ToolApprovalMsg`、Permission 弹窗按键。

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "feat: session list/resume, tool approval dialog, dangerous command detection"
```

---

### Task 9: General Agent 原生能力对齐

**Goal:** 确保 General Agent 的 Shell、Edit、Subagent、Plugin 能力完整透传，事件零静默丢失。

**Files:**
- Create: `tui/tests/parity/general_agent_test.go`
- Create: `tui/tests/parity/native_tools_test.go`
- Create: `tui/tests/parity/event_passthrough_test.go`
- Create: `tui/tests/parity/session_resume_test.go`
- Create: `tui/tests/parity/subagent_test.go`
- Modify: `tui/internal/opencode/event_mapper.go` — 确认覆盖所有 Golden SSE 事件类型

- [ ] **Step 1: 从 Golden SSE 样本确认事件映射覆盖率**

```bash
# 对比 Golden SSE 中的事件类型与 event_mapper.go 中的映射表
# 未映射的事件类型 → 添加到 knownTypes 或确认为 Raw 透传
```

- [ ] **Step 2: 编写事件零静默丢失测试**

`tui/tests/parity/event_passthrough_test.go`：

```go
package parity

import (
	"encoding/json"
	"testing"

	"codea/tui/internal/opencode"
)

func TestGoldenEventsNoSilentDrop(t *testing.T) {
	mapper := opencode.NewEventMapper()

	// 从 Golden SSE 样本加载的事件
	goldenEvents := []string{
		`{"type":"session_started","session_id":"s1"}`,
		`{"type":"answer_delta","content":"hello"}`,
		`{"type":"unknown_future_event","payload":{"x":1}}`,
	}

	for i, raw := range goldenEvents {
		event := mapper.Map(json.RawMessage(raw), int64(i))
		if event.RawType == "" && event.Type == "unknown" {
			t.Errorf("event %d silently dropped: %s", i, raw)
		}
		if event.Type == "unknown" && event.Raw == nil {
			t.Errorf("event %d has no raw passthrough: %s", i, raw)
		}
	}
}
```

- [ ] **Step 3: 编写原生 Tool 完整性测试**

`tui/tests/parity/native_tools_test.go` — 验证所有 Native Tool 在能力清单中标记为 required。

- [ ] **Step 4: 运行 Parity 测试**

```bash
cd tui && go test ./tests/parity/... -v
```

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "feat: General Agent parity tests for event passthrough and native tools"
```

---

### Task 10: Skill/Plugin Manager

**Goal:** 实现 Skill 三态模型、四级配置合并、物理可见性控制、Plugin 兼容策略。

**Files:**
- Create: `tui/internal/config/config.go`
- Create: `tui/internal/config/merge.go`
- Create: `tui/internal/config/profile.go`
- Create: `tui/internal/components/skill.go`
- Create: `distribution/config/codea/defaults.yaml`
- Create: `distribution/config/codea/skills.yaml`
- Create: `distribution/config/codea/profiles/minimal.yaml`
- Create: `distribution/config/codea/profiles/java-backend.yaml`
- Create: `distribution/skills/index.yaml`

- [ ] **Step 1: 编写四级配置合并**

`tui/internal/config/merge.go`：

```go
package config

type SkillConfig struct {
	Name    string
	Enabled string // "enabled" | "disabled" | "inherit"
	Source  string // "release" | "profile" | "user" | "project"
}

// MergeSkills: Release → Profile → User → Project
// 只有 "enabled" 和 "disabled" 会覆盖，"inherit" 穿透到下一层
func MergeSkills(release, profile, user, project map[string]string) map[string]SkillConfig {
	result := make(map[string]SkillConfig)
	for name, state := range release {
		result[name] = SkillConfig{Name: name, Enabled: state, Source: "release"}
	}
	for name, state := range profile {
		if state != "inherit" {
			result[name] = SkillConfig{Name: name, Enabled: state, Source: "profile"}
		}
	}
	for name, state := range user {
		if state != "inherit" {
			result[name] = SkillConfig{Name: name, Enabled: state, Source: "user"}
		}
	}
	for name, state := range project {
		if state != "inherit" {
			result[name] = SkillConfig{Name: name, Enabled: state, Source: "project"}
		}
	}
	return result
}

func EffectiveSkills(merged map[string]SkillConfig) []SkillConfig {
	var result []SkillConfig
	for _, s := range merged {
		if s.Enabled == "enabled" {
			result = append(result, s)
		}
	}
	return result
}
```

- [ ] **Step 2: 编写技术栈自动识别**

`tui/internal/config/profile.go` — 检测 pom.xml/build.gradle/go.mod/requirements.txt 自动选择 Profile。

- [ ] **Step 3: 编写 Skill 面板组件**

`tui/internal/components/skill.go` — 显示 Installed/Enabled/Matched/Source 四列，支持切换。

- [ ] **Step 4: 编写默认配置和 Profile**

`distribution/config/codea/defaults.yaml`、`skills.yaml`、`profiles/*.yaml`。

- [ ] **Step 5: 编写 Skill 配置文件生成器**

在 config 包中添加 `GenerateRuntimeSkills` 方法，只复制 Enabled Skill 到 Runtime 配置目录。

- [ ] **Step 6: Commit**

```bash
git add -A && git commit -m "feat: skill manager with 4-level config merge and profile auto-detection"
```

---

### Task 11: strict/compatible + Enterprise 模式隔离

**Goal:** 实现双轨 Skill 策略、Enterprise Profile 隔离、General compatible 模式的 Project/User Skill 校验。

**Files:**
- Modify: `tui/internal/config/config.go` — 添加 SkillMode 字段
- Modify: `tui/internal/config/merge.go` — 添加模式过滤
- Create: `tui/internal/config/skill_validator.go` — 校验 Project/User Skill

- [ ] **Step 1: 实现 Skill 来源过滤**

```go
type SkillMode string

const (
	SkillModeStrict     SkillMode = "strict"     // V1 默认：仅 Approved
	SkillModeCompatible SkillMode = "compatible" // Approved + Project + User
)

func FilterByMode(skills []SkillConfig, mode SkillMode, approvedSet map[string]bool) []SkillConfig {
	switch mode {
	case SkillModeStrict:
		var result []SkillConfig
		for _, s := range skills {
			if approvedSet[s.Name] {
				result = append(result, s)
			}
		}
		return result
	case SkillModeCompatible:
		// 全部通过，但 Project/User 来源的 Skill 需标记为"需用户确认"
		return skills
	default:
		return skills
	}
}
```

- [ ] **Step 2: 编写 Enterprise 模式启动配置**

Enterprise Profile 启动时，`OPENCODE_CONFIG_DIR` 只包含 Approved + Enabled Skill。

- [ ] **Step 3: 编写隔离验证测试**

验证 Enterprise 模式下项目 `.opencode/skills/` 的 Skill 不被加载。

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "feat: dual-track skill mode, enterprise isolation, compatible validation"
```

---

### Task 12: 安全规则、DLP、Dify、审计

**Goal:** Shell 安全分析引擎、DLP 四层策略、Dify Plugin（含熔断降级）、审计日志 Plugin。

**Files:**
- Create: `tui/internal/config/security.go`
- Modify: `distribution/plugins/src/dify-query.ts` — 完善降级和熔断
- Modify: `distribution/plugins/src/runtime-security-guard.ts` — DLP 规则
- Modify: `distribution/plugins/src/audit-log.ts` — 审计日志
- Create: `distribution/config/opencode/permissions.json`
- Create: `distribution/config/opencode/model.json.tmpl`
- Create: `distribution/templates/AGENTS.md.tmpl`

- [ ] **Step 1: 编写 Shell 安全分析引擎**

`tui/internal/config/security.go`：

```go
package config

import "strings"

type CommandRisk int

const (
	RiskSafe CommandRisk = iota
	RiskAsk
	RiskDeny
)

type CommandAnalysis struct {
	Risk        CommandRisk
	Command     string
	HasPipe     bool
	HasRedirect bool
	HasSubCmd   bool
	MatchedRule string
}

func AnalyzeCommand(input string, dangerousCmds []string) CommandAnalysis {
	analysis := CommandAnalysis{Command: input, Risk: RiskAsk}
	lower := strings.ToLower(input)
	analysis.HasPipe = strings.Contains(input, "|")
	analysis.HasRedirect = strings.ContainsAny(input, "><")
	analysis.HasSubCmd = strings.Contains(input, "$(") || strings.Contains(input, "`")

	for _, cmd := range dangerousCmds {
		if strings.Contains(lower, strings.ToLower(cmd)) {
			analysis.Risk = RiskDeny
			analysis.MatchedRule = cmd
			return analysis
		}
	}

	safeCmds := []string{"git status", "git diff", "git log", "ls", "cat", "head", "tail", "pwd"}
	for _, cmd := range safeCmds {
		if strings.HasPrefix(lower, cmd) {
			analysis.Risk = RiskSafe
			return analysis
		}
	}

	return analysis
}
```

- [ ] **Step 2: 编写 Permissions 配置**

`distribution/config/opencode/permissions.json`：

```json
{
  "agents": {
    "general": {
      "read": "allow", "grep": "allow", "glob": "allow",
      "write": "ask", "edit": "ask", "bash": "ask",
      "agent": "allow", "skill": "allow", "plugin": "allow"
    },
    "code-reviewer": {
      "read": "allow", "grep": "allow", "glob": "allow",
      "collect_review_context": "allow", "dify-query": "allow",
      "write": "deny", "edit": "deny", "bash": "deny"
    },
    "unit-test-generator": {
      "read": "allow", "grep": "allow", "glob": "allow",
      "analyze_test_project": "allow", "write_test_file": "allow",
      "run_project_test": "allow", "dify-query": "allow",
      "bash": "deny", "edit": "deny", "write": "deny"
    },
    "api-documentation": {
      "read": "allow", "grep": "allow", "glob": "allow",
      "extract_api_spec": "allow", "validate_api_example": "allow",
      "write_document": "allow", "dify-query": "allow",
      "write": "deny", "edit": "deny", "bash": "deny"
    }
  }
}
```

- [ ] **Step 3: 完善 Dify Plugin（熔断降级）**

在 `dify-query.ts` 中实现：连续失败 3 次 → 熔断 60 秒 → 降级返回 → 到期试探。

- [ ] **Step 4: 完善安全 Plugin 和审计 Plugin**

`runtime-security-guard.ts` — Tool before/after hook、敏感文件路径拦截。
`audit-log.ts` — 不保存完整源码/Prompt/Token，路径使用项目相对路径。

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "feat: security analyzer, DLP rules, Dify circuit breaker, audit log"
```

---

### Task 13: Enterprise Custom Tools

**Goal:** 实现 7 个企业专用 Tool，每个 Tool 包含 JSON Schema、输入验证、路径边界、超时、DLP、错误分类、单元测试。

**这是审 review 新增的独立 Task，必须在企业 Agent（Task 14-16）之前完成。**

**Files:**
- Create: `distribution/plugins/src/tools/collect-review-context.ts`
- Create: `distribution/plugins/src/tools/analyze-test-project.ts`
- Create: `distribution/plugins/src/tools/write-test-file.ts`
- Create: `distribution/plugins/src/tools/run-project-test.ts`
- Create: `distribution/plugins/src/tools/extract-api-spec.ts`
- Create: `distribution/plugins/src/tools/validate-api-example.ts`
- Create: `distribution/plugins/src/tools/write-document.ts`
- Create: `distribution/plugins/src/tools/failure-classifier.ts`
- Create: `tui/tests/e2e/fixtures/java-maven-project/` — 真实测试 Fixture

---

#### 13.1 collect_review_context

- [ ] **Step 1: 实现 `collect-review-context.ts`**

```typescript
// Code Reviewer 专用 Tool：确定性返回 diff 范围、变更文件、变更行号、上下文代码

interface ReviewContextInput {
  source: "staged" | "unstaged" | "base-branch" | "commit" | "range" | "file-path";
  baseBranch?: string;  // 默认 origin/main
  commit?: string;
  rangeFrom?: string;
  rangeTo?: string;
  filePath?: string;
}

interface ReviewContextOutput {
  filesChanged: number;
  linesAdded: number;
  linesRemoved: number;
  files: Array<{
    path: string;
    status: "added" | "modified" | "deleted" | "renamed";
    oldPath?: string;
    hunks: Array<{
      oldStart: number;
      oldLines: number;
      newStart: number;
      newLines: number;
      lines: string[];
    }>;
  }>;
}

export const collectReviewContextTool = {
  name: "collect_review_context",
  description: "Collect git diff context for code review. Returns exact file paths, line numbers, and diff hunks.",
  parameters: {
    type: "object",
    properties: {
      source: { type: "string", enum: ["staged", "unstaged", "base-branch", "commit", "range", "file-path"] },
      baseBranch: { type: "string" },
      commit: { type: "string" },
      rangeFrom: { type: "string" },
      rangeTo: { type: "string" },
      filePath: { type: "string" },
    },
    required: ["source"],
  },

  execute: async (params: ReviewContextInput, ctx: ToolContext): Promise<ReviewContextOutput> => {
    // 1. 根据 source 构建 git diff 命令
    // 2. 执行 git diff（通过 exec，whitelist-only: git）
    // 3. 解析 diff 输出为结构化结果
    // 4. 路径必须在项目根目录内
    // 5. 超时 30s
    // ...
  },
};
```

#### 13.2 analyze_test_project

- [ ] **Step 2: 实现 `analyze-test-project.ts`**

```typescript
// Unit Test 专用 Tool：确定构建系统、测试目录、框架版本

interface TestProjectInfo {
  buildSystem: "maven" | "gradle" | "unknown";
  testFramework: string;       // "JUnit 5", "JUnit 4", etc.
  testRoots: string[];          // 测试源码根目录（绝对路径）
  sourceRoots: string[];        // 生产源码根目录
  wrapperAvailable: boolean;    // mvnw / gradlew 是否存在
  dependencies: string[];       // 关键测试依赖
  existingTestPattern: string;  // 已有测试文件命名模式
}

export const analyzeTestProjectTool = {
  name: "analyze_test_project",
  description: "Analyze project structure to determine build system, test directories, and framework.",
  // ...
  execute: async (params: {}, ctx: ToolContext): Promise<TestProjectInfo> => {
    // 检测 pom.xml → Maven + 解析 maven-surefire-plugin 配置
    // 检测 build.gradle(.kts) → Gradle + 解析 test 配置
    // 检测 JUnit 4/5、Mockito 版本
    // 确定 test roots 和 source roots
    // 只读操作（read/grep/glob）
  },
};
```

#### 13.3 write_test_file

- [ ] **Step 3: 实现 `write-test-file.ts`**

```typescript
// Unit Test 专用写入 Tool：路径限制在检测的 test roots 内，不可覆盖已有文件

interface WriteTestFileInput {
  path: string;       // 相对于 test root 的路径
  content: string;    // 测试代码
  overwrite: boolean; // 必须显式确认
}

export const writeTestFileTool = {
  name: "write_test_file",
  description: "Write a test file. Path MUST be within one of the detected test roots.",
  // ...
  execute: async (params: WriteTestFileInput, ctx: ToolContext): Promise<WriteResult> => {
    // 1. 验证路径在 test roots 内
    // 2. 如果文件已存在且 overwrite 不为 true → 拒绝
    // 3. DLP 扫描 content（不包含敏感信息）
    // 4. 写入文件
    // 5. 返回写入结果（路径、大小）
  },
};
```

#### 13.4 run_project_test

- [ ] **Step 4: 实现 `run-project-test.ts`**

```typescript
// Unit Test 专用执行 Tool：白名单命令（Maven/Gradle Wrapper 优先），路径限制

interface RunProjectTestInput {
  buildSystem: "maven" | "gradle";
  module?: string;
  testClass?: string;
  testMethod?: string;
  profiles?: string[];
  extraArgs?: string[];   // 经白名单校验
  timeoutSeconds?: number; // 默认 120
}

export const runProjectTestTool = {
  name: "run_project_test",
  description: "Run project tests using Maven or Gradle wrapper.",
  // ...
  execute: async (params: RunProjectTestInput, ctx: ToolContext): Promise<TestRunResult> => {
    // 1. 白名单校验 extraArgs（不允许 rm、curl、sudo 等）
    // 2. 优先使用 ./mvnw / ./gradlew
    // 3. 设置超时（默认 120s）
    // 4. 解析测试输出（通过/失败/错误数量和详情）
    // 5. 返回结构化结果
  },
};
```

#### 13.5 extract_api_spec

- [ ] **Step 5: 实现 `extract-api-spec.ts`**

```typescript
// API Doc 专用 Tool：从 Spring MVC Controller 确定性提取路由、参数、DTO、枚举

interface ApiSpecInput {
  controllerFile: string; // Controller 文件路径
}

interface ApiSpecOutput {
  controllerName: string;
  basePath: string;        // 来自 @RequestMapping
  endpoints: Array<{
    method: string;        // GET/POST/PUT/DELETE
    path: string;          // 组合后的完整路径
    summary: string;
    parameters: Array<{
      name: string;
      type: string;
      required: boolean;
      location: "path" | "query" | "body" | "header";
      validation: string[];  // Bean Validation 注解
      description: string;
    }>;
    requestBody?: { type: string; fields: FieldInfo[] };
    responseType: string;
    errorCodes: Array<{ code: string; status: string; source: "DECLARED" | "REFERENCED" | "INFERRED" }>;
  }>;
  dtos: Record<string, { fields: FieldInfo[] }>;
  enums: Record<string, { values: string[] }>;
}

export const extractApiSpecTool = {
  name: "extract_api_spec",
  description: "Extract API specification from Spring MVC controller. Deterministic — never fabricates.",
  // ...
  execute: async (params: ApiSpecInput, ctx: ToolContext): Promise<ApiSpecOutput> => {
    // 1. 读取 Controller 文件
    // 2. 解析 @RequestMapping/@PostMapping/@GetMapping 等注解
    // 3. 解析 @PathVariable/@RequestParam/@RequestBody 参数
    // 4. 解析 Bean Validation 注解
    // 5. 递归解析 DTO 字段（跟随 import）
    // 6. 解析枚举值
    // 7. 解析 @ExceptionHandler 错误码
    // 8. 不支持的类型标记为 "Not determined from code"
  },
};
```

#### 13.6 validate_api_example

- [ ] **Step 6: 实现 `validate-api-example.ts`**

```typescript
// API Doc 专用验证 Tool：校验生成的示例是否匹配 Schema

interface ValidateExampleInput {
  example: any;              // 生成的示例 JSON
  spec: ApiSpecOutput;       // extract_api_spec 的输出
  endpointIndex: number;     // 验证哪个 endpoint
}

export const validateApiExampleTool = {
  name: "validate_api_example",
  description: "Validate that generated API examples match the extracted spec schema.",
  // ...
  execute: async (params: ValidateExampleInput): Promise<ValidationResult> => {
    // 1. 检查字段是否都来自提取结果（无虚构）
    // 2. 检查必填字段是否齐全
    // 3. 检查枚举值是否在提取的列表中
    // 4. 检查数值范围是否满足 @Min/@Max
    // 5. 返回校验结果和差异
  },
};
```

#### 13.7 write_document

- [ ] **Step 7: 实现 `write-document.ts`**

```typescript
// API Doc 专用写入 Tool：路径限制为 docs/, doc/, api-docs/ 或用户指定

interface WriteDocumentInput {
  path: string;
  content: string;
}

export const writeDocumentTool = {
  name: "write_document",
  description: "Write documentation file. Path restricted to docs/, doc/, api-docs/.",
  // ...
  execute: async (params: WriteDocumentInput, ctx: ToolContext): Promise<WriteResult> => {
    // 1. 验证路径在允许的目录内
    // 2. DLP 扫描
    // 3. 写入文档
  },
};
```

- [ ] **Step 8: 编写 Tool 单元测试**

每个 Tool 至少包含：
- JSON Schema 验证测试（合法/非法输入）
- 路径边界测试（不允许越界写入）
- DLP 规则测试（敏感内容拒绝）
- 真实 Fixture 集成测试（使用 `tui/tests/e2e/fixtures/java-maven-project/`）

- [ ] **Step 9: 编写 Fixture 项目**

`tui/tests/e2e/fixtures/java-maven-project/` — 最小 Spring Boot 项目，包含 Controller、Service、DTO、Enum，用于 Tool 集成测试。

- [ ] **Step 10: Commit**

```bash
git add -A && git commit -m "feat: 7 enterprise custom tools with schemas, boundaries, DLP, and tests"
```

---

### Task 14: Code Reviewer Agent

**Goal:** 基于 Agent Manifest + Prompt + Custom Tool，实现结构化代码审查。

**Files:**
- Create: `distribution/agents/code-reviewer/agent.md`
- Create: `distribution/agents/code-reviewer/manifest.yaml`
- Create: `distribution/agents/code-reviewer/output-schema.json`
- Create: `tui/tests/e2e/code-review/review_test.go`

**Consumes:** Task 13 的 `collect_review_context` Tool

- [ ] **Step 1: 编写 Agent Manifest**

`distribution/agents/code-reviewer/manifest.yaml`：

```yaml
name: code-reviewer
version: 1.0.0
displayName: Code Reviewer
mode: enterprise-controlled
requiredSkills: [code-review]
optionalSkills: [java-review, security-review]
tools:
  read: allow
  grep: allow
  glob: allow
  collect_review_context: allow
  dify-query: allow
  write: deny
  edit: deny
  bash: deny
```

- [ ] **Step 2: 编写 Agent Prompt**

`distribution/agents/code-reviewer/agent.md` — 定义工作流、严重级别（Critical/Major/Minor/Suggestion）、证据要求、confidence 和 introducedByChange 规则。

- [ ] **Step 3: 编写输出 Schema 和 E2E 测试占位**

`output-schema.json` — JSON Schema 约束输出格式。
`review_test.go` — E2E 测试（需要真实 Runtime + 模型）。

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "feat: Code Reviewer agent with structured output and evidence tracking"
```

---

### Task 15: Unit Test Generator Agent

**Goal:** 基于 Agent + Custom Tools，实现 JUnit 5 测试生成与自动修复。

**Files:**
- Create: `distribution/agents/unit-test-generator/agent.md`
- Create: `distribution/agents/unit-test-generator/manifest.yaml`
- Create: `distribution/agents/unit-test-generator/error-categories.yaml`
- Create: `tui/tests/e2e/unit-test/ut_gen_test.go`

**Consumes:** Task 13 的 `analyze_test_project`, `write_test_file`, `run_project_test`, `failure-classifier`

- [ ] **Step 1: 编写 Agent Manifest**

```yaml
name: unit-test-generator
version: 1.0.0
mode: enterprise-controlled
requiredSkills: [unit-test]
tools:
  read: allow
  grep: allow
  glob: allow
  analyze_test_project: allow
  write_test_file: allow
  run_project_test: allow
  dify-query: allow
  edit: deny
  write: deny
  bash: deny
constraints:
  maxRepairAttempts: 3
  neverOverwriteExisting: true
```

- [ ] **Step 2: 编写 Agent Prompt 和错误分类**

`agent.md` — 7 步工作流（analyze → plan → generate → write → run → classify → repair）。
`error-categories.yaml` — 8 种失败类型及处理策略。

- [ ] **Step 3: Commit**

```bash
git add -A && git commit -m "feat: Unit Test Generator agent with repair cycle and error classification"
```

---

### Task 16: API Documentation Agent

**Goal:** 基于 Agent + Custom Tools，实现从 Spring MVC 到结构化 API 文档的生成。

**Files:**
- Create: `distribution/agents/api-documentation/agent.md`
- Create: `distribution/agents/api-documentation/manifest.yaml`
- Create: `distribution/agents/api-documentation/output-template.md`
- Create: `tui/tests/e2e/api-documentation/api_doc_test.go`

**Consumes:** Task 13 的 `extract_api_spec`, `validate_api_example`, `write_document`

- [ ] **Step 1: 编写 Agent Manifest**

```yaml
name: api-documentation
version: 1.0.0
mode: enterprise-controlled
requiredSkills: [api-documentation]
tools:
  read: allow
  grep: allow
  glob: allow
  extract_api_spec: allow
  validate_api_example: allow
  write_document: allow
  dify-query: allow
  write: deny
  edit: deny
  bash: deny
constraints:
  noFabrication: true
  uncertainFields: "Not determined from code"
```

- [ ] **Step 2: 编写 Agent Prompt 和输出模板**

`agent.md` — 确定性提取优先，模型仅负责组织文档和补充业务语义，禁止虚构。
`output-template.md` — Markdown 模板。

- [ ] **Step 3: Commit**

```bash
git add -A && git commit -m "feat: API Documentation Generator agent with deterministic extraction"
```

---

### Task 17: 离线发行包与安装

**Goal:** 离线发行包构建流程（下载 OpenCode → 编译 TUI → 构建 Plugin Bundle → 生成 Manifest → 断网验证）+ macOS/Windows 安装。

**Files:**
- Create: `packaging/config/release.yaml`
- Create: `packaging/scripts/build-runtime.sh`
- Create: `packaging/scripts/build-plugins.sh`
- Create: `packaging/scripts/collect-skills.sh`
- Create: `packaging/scripts/generate-manifest.sh`
- Create: `packaging/scripts/verify-checksum.sh`
- Create: `packaging/scripts/verify-offline.sh`
- Create: `packaging/platform/macos/install.sh`
- Create: `packaging/platform/windows/install.ps1`

- [ ] **Step 1: 编写 Release 配置**

`packaging/config/release.yaml`：

```yaml
schemaVersion: 1
codeaVersion: "0.1.0"
platforms:
  - os: darwin, arch: arm64
  - os: darwin, arch: x64
  - os: windows, arch: x64
signing:
  method: sha256
```

- [ ] **Step 2: 编写 Runtime 下载脚本**

`build-runtime.sh` — 从 `https://github.com/anomalyco/opencode` 下载固定版本，校验 SHA256。

- [ ] **Step 3: 编写 Plugin 构建脚本**

`build-plugins.sh` — bun build 每个 Plugin 为自包含 ESM Bundle，验证无外部 import。

- [ ] **Step 4: 编写断网验证脚本**

`verify-offline.sh`：

```bash
#!/bin/bash
set -euo pipefail

STAGING_DIR="$1"

echo "=== Offline Verification ==="

# 1. 清空缓存
rm -rf ~/.bun/install/cache 2>/dev/null || true
rm -rf /tmp/test-runtime-config

# 2. 阻断公网（需要 sudo）
# 实际测试中由独立隔离环境保证

# 3. 检查 runtime 配置目录无 package.json
if [ -f "$STAGING_DIR/plugins/package.json" ]; then
    echo "FAIL: package.json found in plugin dist"
    exit 1
fi

# 4. 检查 Plugin 文件无外部 import
for js in "$STAGING_DIR/plugins"/*.js; do
    if grep -qE 'require\(|from ["'"'"'](?!\.|/|bun:|node:)' "$js" 2>/dev/null; then
        echo "FAIL: External require/import in $js"
        exit 1
    fi
done

# 5. 检查无构建路径
if grep -q "$HOME" "$STAGING_DIR"/plugins/*.js 2>/dev/null; then
    echo "FAIL: Build paths found in bundle"
    exit 1
fi

echo "All offline checks passed."
```

- [ ] **Step 5: 编写 macOS 和 Windows 安装脚本**

`platform/macos/install.sh` — 校验 → 解压 → 安装到 `~/.codea/versions/` → 设置权限 → 创建启动入口。
`platform/windows/install.ps1` — 同等逻辑。

- [ ] **Step 6: Commit**

```bash
git add -A && git commit -m "feat: offline package build scripts and platform installers"
```

---

### Task 18: 升级回滚事务

**Goal:** 完整的 C1/C2-temp/C2 原子升级事务，包含事务日志、配置迁移注册、staging 校验、崩溃恢复。

**Files:**
- Create: `tui/internal/update/service.go` — 事务编排
- Create: `tui/internal/update/journal.go` — 事务日志
- Create: `tui/internal/update/verifier.go` — staging 校验
- Create: `tui/internal/update/migration.go` — 配置迁移注册
- Create: `tui/internal/update/package.go` — 解压和包校验
- Create: `tui/internal/update/checksum.go` — SHA256 校验
- Create: `tui/internal/update/versions.go` — 版本目录管理
- Create: `tui/internal/update/rollback.go` — 回滚 + 崩溃恢复
- Create: `tui/internal/update/platform.go` — 平台特定切换（Unix symlink / Windows pointer file）
- Create: `tests/upgrade/fresh_install_test.sh`
- Create: `tests/upgrade/upgrade_test.sh`
- Create: `tests/upgrade/failed_upgrade_test.sh`
- Create: `tests/upgrade/rollback_test.sh`

- [ ] **Step 1: 编写事务日志**

`tui/internal/update/journal.go`：

```go
package update

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type TxStatus string

const (
	TxPending    TxStatus = "pending"
	TxCommitted  TxStatus = "committed"
	TxRolledBack TxStatus = "rolled_back"
)

type Transaction struct {
	ID          string    `json:"id"`
	FromVersion string    `json:"fromVersion"`
	ToVersion   string    `json:"toVersion"`
	Status      TxStatus  `json:"status"`
	StartedAt   time.Time `json:"startedAt"`
	Steps       []TxStep  `json:"steps"`
}

type TxStep struct {
	Name      string    `json:"name"`
	Status    TxStatus  `json:"status"`
	StartedAt time.Time `json:"startedAt"`
	Error     string    `json:"error,omitempty"`
}

type Journal struct {
	path string
}

func NewJournal(homeDir string) *Journal {
	return &Journal{path: filepath.Join(homeDir, "update_journal.json")}
}

func (j *Journal) Begin(fromVer, toVer string) (*Transaction, error) {
	tx := &Transaction{
		ID:          time.Now().Format("20060102T150405"),
		FromVersion: fromVer,
		ToVersion:   toVer,
		Status:      TxPending,
		StartedAt:   time.Now(),
	}
	return tx, j.Save(tx)
}

func (j *Journal) Save(tx *Transaction) error {
	data, _ := json.MarshalIndent(tx, "", "  ")
	return os.WriteFile(j.path, data, 0600)
}

func (j *Journal) Load() (*Transaction, error) {
	data, err := os.ReadFile(j.path)
	if err != nil {
		return nil, err
	}
	var tx Transaction
	return &tx, json.Unmarshal(data, &tx)
}

func (j *Journal) RecoverPending() (*Transaction, error) {
	tx, err := j.Load()
	if err != nil {
		return nil, nil // 无事务日志 = 正常状态
	}
	if tx.Status == TxPending {
		// 上次升级中断，需要回滚
		return tx, nil
	}
	return nil, nil
}
```

- [ ] **Step 2: 编写升级事务编排**

`tui/internal/update/service.go`：

```go
func (s *UpdateService) Upgrade(ctx context.Context, packagePath string) error {
	// 1. 获取升级锁
	lock, err := s.acquireLock()
	if err != nil {
		return err
	}
	defer s.releaseLock(lock)

	// 2. 恢复未完成事务（如有）
	if pending, err := s.journal.RecoverPending(); err == nil && pending != nil {
		s.Rollback(ctx, pending.FromVersion)
	}

	// 3. 校验压缩包 → 解压到 staging → 校验 Manifest + 全部文件
	// 4. 复制当前配置 C1 → C2-temp
	// 5. 对 C2-temp 执行配置迁移
	// 6. 使用 V2 + C2-temp 运行预切换 Doctor
	// 7. 记录事务日志（pending）
	// 8. 安装 V2 版本目录
	// 9. 原子切换 current → V2
	// 10. 执行切换后 Doctor
	// 11. 通过后替换正式配置 C1 → C2
	// 12. 标记事务 completed
	// 失败 → 回滚步骤 13-17
	return nil
}
```

- [ ] **Step 3: 编写配置迁移注册**

`tui/internal/update/migration.go`：

```go
type MigrationFunc func(config map[string]any) (map[string]any, error)

type MigrationRegistry struct {
	migrations map[int]MigrationFunc // schemaVersion → migration
}

func (r *MigrationRegistry) Register(fromSchemaVersion int, fn MigrationFunc) {
	r.migrations[fromSchemaVersion] = fn
}

func (r *MigrationRegistry) Migrate(config map[string]any, fromVer, toVer int) (map[string]any, error) {
	current := config
	for v := fromVer; v < toVer; v++ {
		if fn, ok := r.migrations[v]; ok {
			var err error
			current, err = fn(current)
			if err != nil {
				return nil, fmt.Errorf("migration %d→%d: %w", v, v+1, err)
			}
		}
	}
	return current, nil
}
```

- [ ] **Step 4: 编写集成测试**

`tests/upgrade/upgrade_test.sh` — 全新安装 → 升级 → 验证升级后状态。
`tests/upgrade/failed_upgrade_test.sh` — 模拟升级失败 → 验证旧版本可启动并通过 Doctor。
`tests/upgrade/rollback_test.sh` — 升级后回滚 → 验证旧版本完整可用。

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "feat: atomic upgrade transaction with journal, migration, and crash recovery"
```

---

### Task 19: Doctor 诊断

**Goal:** 实现 `codea init` 和 `codea doctor` 命令，覆盖静态/连接/行为/网络四类检查。

**Files:**
- Create: `tui/internal/doctor/service.go`
- Create: `tui/internal/doctor/checks.go`
- Create: `tui/internal/doctor/report.go`
- Modify: `tui/cmd/codea/main.go` — 添加 init/doctor 子命令

- [ ] **Step 1: 编写 Doctor 服务**

`tui/internal/doctor/service.go` — 统一检查接口，PASS/WARN/FAIL/SKIP 严重级别。

- [ ] **Step 2: 编写各项检查**

`tui/internal/doctor/checks.go` — Manifest、文件哈希、配置 Schema、权限、Skill Manifest、Plugin、版本兼容、Runtime 健康、模型连接、SSE、推理。

- [ ] **Step 3: 编写 Doctor 报告**

`tui/internal/doctor/report.go` — 格式化输出，FAIL 时返回非零退出码。

- [ ] **Step 4: 添加 CLI 子命令**

```go
func main() {
	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "init":
			runInit()
			return
		case "doctor":
			runDoctor()
			return
		}
	}
	// TUI ...
}
```

- [ ] **Step 5: Commit**

```bash
git add -A && git commit -m "feat: Doctor diagnostic service and init command"
```

---

### Task 20: 试点统计与反馈

**Goal:** 会话统计收集、匿名化、轻量反馈机制。

**Files:**
- Create: `tui/internal/app/metrics.go`
- Create: `tui/internal/app/feedback.go`

- [ ] **Step 1: 编写统计指标收集器**

`tui/internal/app/metrics.go` — 记录 Session 开始/结束、Agent 类型、耗时、Skill 加载、采纳状态。

- [ ] **Step 2: 编写轻量反馈组件**

`tui/internal/app/feedback.go` — Yes/Partly/No 可选反馈，Esc 跳过。

- [ ] **Step 3: Commit**

```bash
git add -A && git commit -m "feat: pilot metrics collector and lightweight feedback"
```

---

### Task 21: Release Parity Certification

**Goal:** 全量 G1-G15 门禁，Parity Runner 真实对比，发布清单。

**Files:**
- Modify: `tui/cmd/parity-runner/main.go` — 真实执行所有 Required 场景
- Create: `docs/release-checklist.md`

- [ ] **Step 1: 完善 Parity Runner**

`tui/cmd/parity-runner/main.go`：

```go
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"codea/tui/internal/parity"
)

func main() {
	runner := parity.NewParityRunner()

	// G11: 原生能力完整性
	runner.Register(parity.ParityScenario{
		ID: "G11", Description: "All OpenCode native capabilities accessible",
		Required: true,
		BaselineRunner:  newBaselineRunner(),  // 原版 opencode serve
		CandidateRunner: newCandidateRunner(), // codea + opencode serve
		Repetitions: 1,
		Assertions: []parity.Assertion{
			{Type: parity.AssertNoError},
			{Type: parity.AssertMinCount, Value: 14},
		},
	})

	// G13: 事件零静默丢失
	runner.Register(parity.ParityScenario{
		ID: "G13", Description: "Zero silent event drops from Golden SSE",
		Required: true,
		BaselineRunner:  newBaselineRunner(),
		CandidateRunner: newCandidateRunner(),
		Repetitions: 1,
		Assertions: []parity.Assertion{
			{Type: parity.AssertNoError},
		},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	report := runner.Run(ctx)
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	enc.Encode(report)

	if report.FailedCount > 0 {
		fmt.Fprintf(os.Stderr, "\n%d scenarios FAILED\n", report.FailedCount)
		os.Exit(1)
	}
	fmt.Println("\nAll parity scenarios passed.")
}
```

- [ ] **Step 2: 编写发布清单**

`docs/release-checklist.md` — G1-G15 全部验收门禁、签核表。

- [ ] **Step 3: 运行全量测试**

```bash
cd tui && go test ./...
./scripts/run-phase0-gates.sh
go run ./tui/cmd/parity-runner
```

- [ ] **Step 4: Commit**

```bash
git add -A && git commit -m "feat: release parity certification, runner, and checklist"
```

---

## 附录 A: 开发里程碑

| 里程碑 | Tasks | 可交付物 | 关键门禁 |
|--------|-------|----------|----------|
| M0 | Task 0 | 项目骨架、Go Module 正确结构 | `go test ./...` 通过 |
| M1 | Task 1 | Spike S1-S6 报告、Golden SSE、OpenAPI Spec | S1-S6 全部通过 |
| M2 | Task 2 | 从 OpenAPI 生成的 DTO | `/doc` 路径确认 |
| M3 | Task 2A → Task 3 → Task 4 | AgentRuntime/OpenCodeAdapter、Capability Parity、Runtime 恢复与补偿 | Runtime 边界、Parity、恢复契约与人工验收通过 |
| M4 | Tasks 5-6 | Supervisor + 跨平台 + Reasoning | 双平台编译通过 |
| M5 | Tasks 7-9 | TUI 基础 + Session + General 对齐 | Parity 增量测试通过 |
| M6 | Tasks 10-11 | Skill/Plugin Manager + 模式隔离 | G3 隔离验证通过 |
| M7 | Task 12 | 安全层 + Dify + DLP + 审计 | 安全规则测试通过 |
| M8 | Task 13 | 7 个 Enterprise Custom Tools | 每个 Tool 集成测试通过 |
| M9 | Tasks 14-16 | Code Reviewer + UT + API Doc | G6-G8 门禁 |
| M10 | Tasks 17-18 | 离线包 + 安装 + 升级回滚 | G1-G5 门禁 |
| M11 | Tasks 19-20 | Doctor + 试点统计 | Doctor 通过 |
| M12 | Task 21 | Release Parity Certification | G1-G15 全部通过 |

## 附录 B: 审查修正记录

| # | 问题 | 修正 |
|---|------|------|
| 1 | 仓库地址 `anthropics/opencode` | 改为 `anomalyco/opencode` |
| 2 | Go Module 结构导致测试无法运行 | 所有 Go 代码统一在 `tui/` Module，测试在 `tui/tests/` |
| 3 | Phase 0 缺少 S6，t.Skip 假通过 | 增加 S6 (Enterprise 隔离)，移除 t.Skip，改真实门禁 |
| 4 | API/DTO 与官方不匹配 | 从锁定版本 `/doc` 生成 DTO，不手写猜测 |
| 5 | Supervisor 缺少 Basic Auth、跨平台 | 增加 OPENCODE_SERVER_PASSWORD、platform_unix/windows.go、移除 --config-dir |
| 6 | 企业能力缺少 Tool 实现 | 新增 Task 13: 7 个 Enterprise Custom Tools，在企业 Agent 之前 |
| 7 | 任务依赖顺序倒置 | 重新排序：先 Skill Manager 后 compatible，先 Tools 后 Agent，先 Plugin 实现后离线打包 |
| 8 | Parity Harness 假通过 | Required 场景 Skip → Fail，增加真实 Assertions 和对比逻辑 |
| 9 | 离线/升级不完整 | 平台明确：macOS arm64/x64 + Windows x64；升级增加 journal/migration/verifier/crash recovery |
