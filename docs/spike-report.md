# Phase 0 Spike Report

## 当前结论

**S1～S6 判定：PASS。** 六个 Phase 0 技术假设均已通过真实 Runtime 或隔离对照验证，可以执行机器门禁。

| Spike | 状态 | 说明 |
|---|---|---|
| S1 Server 离线启动 | **PASS** | 真实断网 + 正确环境变量，内部日志无公网请求 |
| S2 Session + Prompt + SSE | **PASS** | Session 200、Prompt 204、收到目标 Session 的流式文本和 idle |
| S3 Tool Approval | **PASS** | `permission.asked` 可接收，`once/reject` 均正确执行 |
| S4 Reasoning | **PASS** | 独立 `reasoning` 与 `text` Part，无需 `<think>` 拆分 |
| S5 Skill 来源隔离 | **PASS** | 隔离模式只加载内置与批准配置目录 Skill |
| S6 模式隔离 | **PASS** | Enterprise/General Compatible/General Strict 三模式符合预期 |

**关键发现**：初版验证使用了不存在的环境变量名（`OPENCODE_SKIP_MODEL_FETCH`），导致 OpenCode 仍发起 `models.opencode.ai` 请求。v1.18.11 官方已支持 `OPENCODE_DISABLE_MODELS_FETCH=1` 禁用该请求，无需 Patch。

---

## S1 证据

### 1. 版本锁定（2026-08-04 复核）

| 项目 | 值 |
|------|-----|
| 官方仓库 | `https://github.com/anomalyco/opencode` |
| Release | `v1.18.11` |
| Tag commit | `012c2f57f976489d88bd4598a056b4bdcdd428ee` |
| Release 时间 | `2026-08-01T11:44:45Z` |
| linux-x64 `opencode-linux-x64.tar.gz` | `a4dffcc00a5a93256c6bd06aa0c984320528f564db52a1f4becd5c7de9fb59a1` |
| darwin-arm64 `opencode-darwin-arm64.zip` | `188ff6a716bcd40e33ac62f17f4aec9bd760164fa6a2cde66f779a5db4abc7ce` |
| darwin-x64 `opencode-darwin-x64.zip` | `95953ab2aca4322b90690bf34697cc9b47b6a7c72f78e7c469056fb589124d31` |
| windows-x64 `opencode-windows-x64.zip` | `f3a5ea814aecc692a4e04259d9005283f364225b38456c90f9a47b7a9d83c0e9` |

四个官方资产均已实际下载到临时目录并执行 `sha256sum`；结果逐项等于 GitHub Release API 返回的 digest。结构化证据：`runtime/version.json`、`docs/spike-artifacts/s1-release.json`、`docs/spike-artifacts/s1-release-checksums.txt`。

### 2. Linux 容器本地启动（2026-08-03）

在独立 `OPENCODE_CONFIG_DIR` 中启动，健康检查通过：

```json
{"healthy": true, "version": "1.18.11"}
```

原始证据：`docs/spike-artifacts/s1-server.log`、`docs/spike-artifacts/s1-health.json`

### 3. Linux 容器阻塞（2026-08-03）

容器拒绝网络隔离和系统调用观测：

- `unshare -n` → `Operation not permitted`
- `bwrap --unshare-net` → `Operation not permitted`
- `strace` → `PTRACE_TRACEME: Operation not permitted`

### 4. macOS 真实断网验证（2026-08-03）

#### 方法

1. `sudo ifconfig` 关闭 en0/en1-6/awdl0/llw0/bridge0/ap1，仅保留 lo0
2. `sudo tcpdump -i en0 -n -w` 持续抓包约 28 秒
3. 完全隔离沙箱：独立 `$HOME`、`XDG_*`、`OPENCODE_CONFIG_DIR`，全部指向空目录
4. 独立 `BUN_INSTALL`、`NPM_CONFIG_CACHE`

环境：

- 平台：macOS arm64
- 制品：`opencode-darwin-arm64.zip`
- SHA-256：`188ff6a716bcd40e33ac62f17f4aec9bd760164fa6a2cde66f779a5db4abc7ce`

#### Server stdout

```
opencode server listening on http://127.0.0.1:49325
```

#### 健康检查

```json
{"healthy": true, "version": "1.18.11"}
```

#### 关键证据：OpenCode 内部日志

文件：`$XDG_DATA_HOME/opencode/log/opencode.log`

```
timestamp=2026-08-03T06:58:01.313Z level=INFO  run=24ea301a message=loading config.json
timestamp=2026-08-03T06:58:01.315Z level=INFO  run=24ea301a message=loading opencode.json
timestamp=2026-08-03T06:58:01.320Z level=INFO  run=24ea301a message=loading opencode.jsonc
timestamp=2026-08-03T06:58:02.093Z level=ERROR run=24ea301a message="Failed to fetch models.dev"
  cause="Cause([Fail(HttpClientError: Transport error (GET https://models.opencode.ai/api.json)
  (cause: Error: Unable to connect. Is the computer able to access the url?))])"
timestamp=2026-08-03T06:58:03.235Z level=ERROR run=24ea301a message="Failed to fetch models.dev"
  cause="Cause([Fail(HttpClientError: Transport error (GET https://models.opencode.ai/api.json)
  (cause: Error: Was there a typo in the url or port?))])"
```

第 4、5 行是两次 `GET https://models.opencode.ai/api.json` 请求，发生在启动后约 1 秒和 2 秒。

#### tcpdump 分析

抓包仅覆盖 `en0` 接口，共捕获 4 个包，全部为断网前旧连接的入站残留 ACK。抓包期间 `en0` 已 down，因此工具上没有捕获到出站请求——但这仅仅是因为 `en0` 关闭后请求发不出去，不能说明 OpenCode 没有发起请求。

**tcpdump 的方法局限**：

- 仅监听 `en0`（主 Wi-Fi），未监听 `utun0-5`（VPN tunnel 接口），后者在断网后仍为活跃状态
- `en0` 被关闭后不产生流量是预期结果，不是「没有请求」的证据
- 出站请求的实际判定必须结合 OpenCode 内部日志

### S1 门禁对照（修正版验证）

| 门禁要求 | 状态 | 证据 |
|----------|------|------|
| 真实断网环境 | 满足 | en0-6/awdl0/llw0/bridge0/ap1 全部 down |
| 独立沙箱环境 | 满足 | 全新 `$HOME` + `XDG_*` + `OPENCODE_CONFIG_DIR` |
| 版本锁定 + SHA-256 完整 | 满足 | 四个平台的精确资产名、大小、下载 URL 和 SHA-256 均已锁定，无占位符 |
| Server 启动成功 | 满足 | `{"healthy":true,"version":"1.18.11"}` |
| 不访问公网 | **满足** | 内部日志零 `models.opencode.ai`，零 ERROR |
| 无公网 DNS 请求 | 满足 | 全接口抓包无 OpenCode 相关 DNS 查询 |
| 无公网 HTTP/HTTPS 出站 | 满足 | 全接口抓包无 OpenCode 相关 HTTP/HTTPS 流量 |

### 初版失败原因

初版验证使用以下**不存在**的环境变量，导致 OpenCode 仍发起 `models.opencode.ai` 请求：

| 错误变量（不存在） | 正确变量（v1.18.11 官方支持） |
|-------------------|------------------------------|
| `OPENCODE_SKIP_MODEL_FETCH` | `OPENCODE_DISABLE_MODELS_FETCH` |
| `OPENCODE_DISABLE_AUTO_UPDATE` | `OPENCODE_DISABLE_AUTOUPDATE` |
| `OPENCODE_SKIP_WEB_UI` | `OPENCODE_DISABLE_EMBEDDED_WEB_UI` |
| `OPENCODE_OFFLINE_MODE` | （不存在于 v1.18.11） |

官方文档参考：`packages/web/src/content/docs/cli.mdx`、`packages/core/src/flag/flag.ts`、`packages/core/src/models-dev.ts`

### 修正版验证（s1-20260803-175535）

**环境变量**：
```bash
OPENCODE_DISABLE_MODELS_FETCH=1
OPENCODE_DISABLE_AUTOUPDATE=1
OPENCODE_DISABLE_EMBEDDED_WEB_UI=1
OPENCODE_DISABLE_LSP_DOWNLOAD=1
OPENCODE_DISABLE_DEFAULT_PLUGINS=1
OPENCODE_DISABLE_EXTERNAL_SKILLS=1
OPENCODE_DISABLE_PROJECT_CONFIG=1
OPENCODE_DISABLE_CLAUDE_CODE=1
```

**内部日志**（3 行，全部 INFO）：
```
INFO loading config.json
INFO loading opencode.json
INFO loading opencode.jsonc
```
零 ERROR，零 `models.opencode.ai`。

**tcpdump**：9 个接口合计 31 包，全部在 `en0` 上，时间戳在 `ifconfig down` 前（17:55:35-36），来自 jsss/xray 等非 OpenCode 进程。其余 8 个接口（awdl0/llw0/utun0-5）零包。

**自动判定**：`scripts/check-s1-offline-evidence.sh` 统一检查健康响应、内部日志和逐接口 pcap，并只统计 Runtime 验证时间窗内的数据包。新运行由采集脚本保存 `validation-window.json`；旧证据从内部日志首条 Runtime 时间戳建立保守起点。发现 `models.opencode.ai`、任意 `ERROR`、窗口内 DNS/HTTP/HTTPS 流量或证据缺失时返回 1；`tests/phase0/check_s1_offline_evidence_test.sh` 覆盖全部失败分支、窗口前流量和干净通过分支。旧证据经内置 pcap 解析器复核，31 个 `en0` 包均在 Runtime 时间窗前，窗口内为 0 包。`docs/spike-artifacts/s1-network-test.sh` 最终以判定器的退出码退出，不再只打印 `[FAIL]`。

---

## S2 证据

### 验证范围与方法（Linux x64，2026-08-03）

本次验证使用锁定的 OpenCode v1.18.11 Linux x64 官方制品。为排除外部模型可用性对协议 Spike 的影响，模型端使用本地 OpenAI-compatible 流式协议桩；Session、Agent Loop、消息持久化和 SSE 均由真实 OpenCode Runtime 执行。

Go 客户端位于 `tui/cmd/spike-s2/`，执行顺序为：

1. `POST /session?directory=...` 创建 Session。
2. `GET /global/event` 建立 SSE 连接。
3. `POST /session/:id/prompt_async?directory=...` 发送 Prompt。
4. 过滤目标 Session 的事件，收到 `session.status` 的 `idle` 状态后结束。
5. `GET /session/:id/message?directory=...` 回读最终消息。

### 实际请求与响应

| 操作 | 实际结果 |
|---|---|
| `GET /global/health` | HTTP 200，`{"healthy":true,"version":"1.18.11"}` |
| `POST /session` | HTTP 200，Session ID `ses_038a3daa2ffepds97I1dEfA1kk` |
| `POST /session/:id/prompt_async` | HTTP 204 |
| OpenAI-compatible 模型请求 | `POST /v1/chat/completions`，`stream=true`、`include_usage=true` |
| SSE 完成条件 | 目标 Session 依次出现 `busy`，最终出现 `idle` |
| 模型流式文本 | `message.part.delta`：`hello from s2` |
| 消息回读 | 用户消息与 Assistant 文本均存在 |

Go 客户端共记录 76 个全局 SSE 事件，其中与完成链路直接相关的事件包括：

- `session.status`：4 条，最后一条为目标 Session 的 `idle`。
- `message.part.delta`：1 条，文本为 `hello from s2`。
- `message.part.updated`：5 条，包含 step-start、text、step-finish 和 patch Part。
- `message.updated`：4 条。
- `session.diff`：1 条。

未出现 `session.error`。OpenCode 内部日志记录 Session 创建、全局事件连接和 Agent Loop 启动，无 ERROR。

### S2 门禁对照

| 门禁要求 | 状态 | 证据 |
|---|---|---|
| 创建 Session | 满足 | HTTP 200 和非空 Session ID |
| 发送异步 Prompt | 满足 | HTTP 204 |
| 接收 SSE | 满足 | `events.jsonl` 共 76 条事件 |
| 接收模型流式回答 | 满足 | `message.part.delta` 为 `hello from s2` |
| 完成一轮 Agent 会话 | 满足 | 目标 Session 最终进入 `idle`，消息可回读 |
| 记录实际协议结构 | 满足 | 请求、事件、消息和日志均保存为原始证据 |

S2 验证的是 OpenCode Runtime 到 Go 客户端的完整协议与状态链路，不评价模型内容质量。模型端使用本地确定性协议桩是为了让结果可重复、无需公网和真实 API Key。

---

## S3 证据

### 验证范围与方法（Linux x64，2026-08-03）

使用真实 OpenCode v1.18.11 Runtime、本地 OpenAI-compatible Tool Call 协议桩和 Go Spike 客户端，分别创建批准与拒绝两个独立 Session。Runtime 配置 `permission.bash=ask`，模型请求执行 `touch s3-marker.txt`。

| 分支 | Permission Reply | Runtime 结果 | 文件结果 | 会话结果 |
|---|---|---|---|---|
| 批准 | `once`，HTTP 200 | Tool Part `completed`，exit 0 | marker 存在 | `idle`，无 `session.error` |
| 拒绝 | `reject`，HTTP 200 | Tool Part `error`，明确记录用户拒绝 | marker 不存在 | `idle`，无 `session.error` |

### 实际协议结构

- 申请事件：`permission.asked`
- Permission ID：`per_...`
- 关键字段：`sessionID`、`permission`、`patterns`、`metadata`、`always`、`tool.messageID`、`tool.callID`
- 推荐响应端点：`POST /permission/{requestID}/reply?directory=...`
- 请求体：`{"reply":"once"}` 或 `{"reply":"reject"}`
- 响应事件：`permission.replied`
- v1.18.11 支持的 reply 枚举：`once`、`always`、`reject`

计划中的 `tool_approval_required` 并非 v1.18.11 实际事件名；`POST /session/{sessionID}/permissions/{permissionID}` 虽仍存在，但 OpenAPI 已标记 deprecated，后续客户端应使用全局 Permission Reply API。

### S3 门禁对照

| 门禁要求 | 状态 | 证据 |
|---|---|---|
| Go 客户端接收权限申请 | 满足 | 两个 Session 均收到目标 `permission.asked` |
| 批准后执行 Tool | 满足 | Tool completed、exit 0、marker 存在 |
| 拒绝后不执行 Tool | 满足 | Tool error 为用户拒绝、marker 不存在 |
| Runtime 正确完成会话 | 满足 | 两个 Session 最终均为 idle，无 session.error |
| 记录事件、ID 与枚举 | 满足 | `docs/spike-artifacts/s3-20260803/` |

---

## S4 证据

使用真实 OpenCode v1.18.11 Runtime 和本地 OpenAI-compatible Reasoning 协议桩。模型流式 delta 同时返回 `reasoning_content` 和普通 `content`，Go 客户端监听到 87 条 SSE 后按 Part 类型完成区分。

| 项目 | 实际结果 |
|---|---|
| Reasoning Part | `type=reasoning`，文本 `considering options` |
| Answer Part | `type=text`，文本 `final answer` |
| 流式事件 | 两类 Part 均有 `message.part.delta` 与 `message.part.updated` |
| 完成条件 | 目标 Session 最终 `idle` |
| `<think>` 标签 | 不存在，不依赖标签解析 |

S4 结论：v1.18.11 能将兼容模型的 `reasoning_content` 转换为结构化 `reasoning` Part；Go 客户端应优先按 Part 类型分流，`<think>` 只能作为非结构化模型的兼容降级方案。

---

## S5 证据

在配置目录、项目目录、用户 OpenCode 目录、Claude 兼容目录和 Agents 兼容目录同时放置唯一命名的测试 Skill，并通过 `GET /skill` 读取真实 Runtime 注册结果。

隔离组使用独立 HOME/XDG，并设置：

```text
OPENCODE_CONFIG_DIR=<approved-config>
OPENCODE_DISABLE_EXTERNAL_SKILLS=1
OPENCODE_DISABLE_PROJECT_CONFIG=1
OPENCODE_DISABLE_CLAUDE_CODE=1
```

| 来源 | 隔离组 | 无隔离对照组 |
|---|---|---|
| OpenCode 内置 `customize-opencode` | 发现 | 发现 |
| 配置目录 `config-approved` | 发现 | 发现 |
| 项目 `.opencode` | 未发现 | 发现 |
| 用户 `.config/opencode` | 未发现 | 发现 |
| `~/.claude/skills` | 未发现 | 发现 |
| 项目 `.agents/skills` | 未发现 | 发现 |

结论：无需 Patch。企业隔离必须组合独立 HOME/XDG、指定 `OPENCODE_CONFIG_DIR` 和上述三个禁用开关；内置 `customize-opencode` 属于 Runtime 固有 Skill，不是外部注入。

2026-08-04 使用 `scripts/run-skill-isolation-spikes.sh` 和锁定的 Linux x64 v1.18.11 Runtime 完整重跑。脚本从 `tests/fixtures/skill-isolation/` 创建五类真实 Skill，保存未修改的 `/skill` JSON，再由 JSON 机械生成名称清单并做精确集合断言。隔离组原始响应包含 2 项，对照组包含 6 项；全部证据位于 `docs/spike-artifacts/s5-s6-20260803-rerun/s5/`。

---

## S6 证据

使用同一组 Skill 夹具和真实 `/skill` API，分别启动三个独立 Runtime Profile：

| 模式 | 实际发现的 Skill | 项目 Skill |
|---|---|---|
| Enterprise | `config-approved`、`customize-opencode` | 未注入 |
| General Compatible | 上述两项 + `project-unapproved` | 已加载 |
| General Strict（V1 默认） | `config-approved`、`customize-opencode` | 未注入 |

三种模式均继续隔离用户、Claude 和 Agents 兼容来源。General Compatible 的测试项目 Skill 具有合法目录名和 frontmatter，并被 Runtime 成功解析；后续正式实现仍需在启动前接入 Codea Skill 校验器。

S6 结论：基础双模式隔离无需 Patch，可通过独立 Runtime Profile 和启动环境变量实现；Enterprise 与 General Strict 禁用项目配置，General Compatible 仅开放通过校验的项目 Skill 来源。

同一次真实重跑依次启动 Enterprise、General Compatible、General Strict 三个独立 Profile，原始 `/skill` 响应分别包含 2、3、2 项，且名称清单均由原始 JSON 机械生成。配置夹具副本及 SHA-256 manifest 一并保存在 `docs/spike-artifacts/s5-s6-20260803-rerun/`，可独立复核。

---

## 原始证据文件

| 文件 | 说明 |
|------|------|
| `docs/spike-artifacts/s1-release.json` | 版本锁定结构化数据 |
| `docs/spike-artifacts/s1-release-checksums.txt` | 四个平台官方 CLI 资产的实际下载 SHA-256 输出 |
| `docs/spike-artifacts/s1-server.log` | Linux 容器 Server stdout |
| `docs/spike-artifacts/s1-health.json` | Linux 容器健康检查响应 |
| `docs/spike-artifacts/s1-network-test.sh` | 修正版测试脚本（trap + 全接口 + 正确环境变量） |
| `docs/spike-artifacts/s1-20260803-175535/` | 修正版验证完整证据目录 |
| `docs/spike-artifacts/s1-20260803-175535/execution.log` | 脚本完整执行日志 |
| `docs/spike-artifacts/s1-20260803-175535/health.json` | 健康检查响应 |
| `docs/spike-artifacts/s1-20260803-175535/server-stdout.log` | Server stdout |
| `docs/spike-artifacts/s1-20260803-175535/opencode-internal.log` | OpenCode 内部日志（3 行 INFO，零 ERROR） |
| `docs/spike-artifacts/s1-20260803-175535/traffic-*.pcap` | 逐接口 tcpdump 原始抓包（9 个接口） |
| `docs/spike-artifacts/s2/opencode.json` | 本地 OpenAI-compatible Provider 配置，API Key 仅引用环境变量 |
| `docs/spike-artifacts/s2/fake-openai-server.py` | 确定性流式模型协议桩 |
| `docs/spike-artifacts/s2-20260803/health.json` | OpenCode 健康响应 |
| `docs/spike-artifacts/s2-20260803/client.log` | Go 客户端 Session、Prompt 和完成摘要 |
| `docs/spike-artifacts/s2-20260803/events.jsonl` | 原始全局 SSE 事件 |
| `docs/spike-artifacts/s2-20260803/messages.json` | Session 最终消息回读 |
| `docs/spike-artifacts/s2-20260803/fake-model-requests.jsonl` | OpenCode 发给模型端的原始流式请求 |
| `docs/spike-artifacts/s2-20260803/opencode-internal.log` | OpenCode 内部日志 |
| `docs/spike-artifacts/s3/opencode.json` | 强制 Bash Tool 进入 ask 的隔离配置 |
| `docs/spike-artifacts/s3/fake-tool-server.py` | 确定性 Tool Call 协议桩 |
| `docs/spike-artifacts/s3-20260803/approve-key-events.jsonl` | 批准分支 Permission 与状态事件 |
| `docs/spike-artifacts/s3-20260803/reject-key-events.jsonl` | 拒绝分支 Permission 与状态事件 |
| `docs/spike-artifacts/s3-20260803/tool-parts.jsonl` | Tool completed/error 最终状态 |
| `docs/spike-artifacts/s3-20260803/file-results.txt` | 批准创建、拒绝不创建的文件断言 |
| `docs/spike-artifacts/s3-20260803/opencode-internal.log` | S3 OpenCode 内部日志 |
| `docs/spike-artifacts/s4/opencode.json` | Reasoning 模型隔离配置 |
| `docs/spike-artifacts/s4/fake-reasoning-server.py` | Reasoning 协议桩 |
| `docs/spike-artifacts/s4-20260803/key-events.jsonl` | Reasoning/Text Part 与状态事件 |
| `docs/spike-artifacts/s4-20260803/client.log` | 客户端分类摘要 |
| `docs/spike-artifacts/s4-20260803/opencode-internal.log` | S4 OpenCode 内部日志 |
| `scripts/run-skill-isolation-spikes.sh` | S5/S6 可重复执行入口，创建夹具、启动五个 Profile、保存原始响应并断言集合 |
| `tests/fixtures/skill-isolation/` | 配置、项目、用户、Claude、Agents 五类原始 Skill 夹具 |
| `docs/spike-artifacts/s5-s6-20260803-rerun/fixture-manifest.txt` | 本次重跑夹具 SHA-256 manifest |
| `docs/spike-artifacts/s5-s6-20260803-rerun/execution.log` | 锁定 Runtime 五 Profile 真实重跑的完整判定输出 |
| `docs/spike-artifacts/s5-s6-20260803-rerun/s5/*-skill-response.json` | S5 隔离组与对照组原始 `/skill` JSON 响应 |
| `docs/spike-artifacts/s5-s6-20260803-rerun/s6/*-skill-response.json` | S6 三 Profile 原始 `/skill` JSON 响应 |
| `docs/spike-artifacts/s5-s6-20260803-rerun/s5/` | S5 health、Runtime 日志和机械生成的名称清单 |
| `docs/spike-artifacts/s5-s6-20260803-rerun/s6/` | S6 health、Runtime 日志和机械生成的名称清单 |
| `runtime/openapi/opencode-1.18.11.json` | 锁定版本 OpenAPI 3.1 文档 |
| `runtime/openapi/golden-sse-s2.jsonl` | 76 条完整会话 Golden SSE |

---

## Phase 0 机器结果

`docs/spike-results.json` 记录 S1～S6 全部为 `pass`，由 `scripts/run-phase0-gates.sh` 唯一消费；缺失、失败或未知值均返回非零退出码。
