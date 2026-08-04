# Task 01 Report — Phase 0 Spike S1–S6

**Task:** 1

**Status:** awaiting_acceptance

**Current step:** 10 — 等待人工验收

**Date:** 2026-08-04

**Checkpoint:** `a6755287dd4ab984c47517b9acdd7320776c39cf`

## 已完成内容

### S1 Server 离线启动 — PASS

- 锁定 OpenCode v1.18.11；Linux x64、macOS arm64/x64、Windows x64 四个平台精确资产、大小、URL 和 SHA-256 均已记录并通过实际下载复核。
- 在 macOS arm64 上使用新版脚本完成真实断网重跑：
  - 动态发现并关闭全部 20 个活动外部接口，包含此前漏采的 `anpi0-2` 和 `utun0-5`，仅保留 `lo0`
  - 完全隔离沙箱：独立 `$HOME`、`XDG_*`、`OPENCODE_CONFIG_DIR`
  - 对 manifest 中全部接口持续抓包，并保存显式验证时间窗
  - trap 机制保证断网后无论成功失败都恢复网络
- 使用正确的官方环境变量：
  - `OPENCODE_DISABLE_MODELS_FETCH=1`（核心——禁用 models.opencode.ai 请求）
  - `OPENCODE_DISABLE_AUTOUPDATE=1`
  - `OPENCODE_DISABLE_EMBEDDED_WEB_UI=1`
  - `OPENCODE_DISABLE_LSP_DOWNLOAD=1`
  - `OPENCODE_DISABLE_DEFAULT_PLUGINS=1`
  - `OPENCODE_DISABLE_EXTERNAL_SKILLS=1`
  - `OPENCODE_DISABLE_PROJECT_CONFIG=1`
  - `OPENCODE_DISABLE_CLAUDE_CODE=1`
- 验证结果：
  - 健康检查：`{"healthy":true,"version":"1.18.11"}`
  - 内部日志：3 行 INFO，零 ERROR，零 `models.opencode.ai`
  - 时间窗 `1785813380.841484`～`1785813405.927638` 内，20 个接口均为零 DNS/HTTP/HTTPS 流量
  - 原始证据：`docs/spike-artifacts/s1-20260804-111618/`
  - 独立判定器返回 0：`S1 evidence validation passed.`
- 新增独立 S1 证据判定器及回归测试；检测到公网主机、ERROR、时间窗内 DNS/HTTP(S) 流量、活动接口缺少 pcap、健康失败或证据缺失时必定非零退出。
- 新脚本动态记录并抓取全部活动非 `lo0` 接口，保存显式进程验证时间窗；异常退出会终止 OpenCode/tcpdump，并只恢复脚本实际关闭的接口。
- 旧证据因缺少 `anpi0-2` 抓包和显式启动时间窗仅作为历史材料；最终 PASS 仅由 2026-08-04 新证据支撑。

### 初版问题与修正

初版使用了不存在的环境变量名（`OPENCODE_SKIP_MODEL_FETCH`、`OPENCODE_DISABLE_AUTO_UPDATE`、`OPENCODE_SKIP_WEB_UI`、`OPENCODE_OFFLINE_MODE`），导致 OpenCode 仍请求 `models.opencode.ai`。经上游源码（`flag.ts`、`models-dev.ts`）确认正确变量名后修正。

### S2 Session + Prompt + SSE — PASS

- 新增 Go Spike 客户端 `tui/cmd/spike-s2/`，按 TDD 验证 SSE JSON 解码、目标 Session 过滤和 idle 完成条件。
- 使用真实 OpenCode v1.18.11 Runtime 与本地 OpenAI-compatible 流式协议桩完成确定性验证。
- 实际链路结果：
  - `POST /session`：HTTP 200，返回非空 Session ID。
  - `GET /global/event`：建立全局 SSE。
  - `POST /session/:id/prompt_async`：HTTP 204。
  - 共记录 76 条 SSE，目标 Session 从 busy 进入 idle。
  - `message.part.delta` 返回 `hello from s2`。
  - `GET /session/:id/message` 可回读用户消息和 Assistant 回答。
- 未出现 `session.error`；OpenCode 内部日志无 ERROR。
- 原始证据：`docs/spike-artifacts/s2-20260803/`。
- 该 Spike 验证 Runtime 协议与状态链路；本地模型协议桩不用于评价模型质量。

### S3 Tool Approval — PASS

- 新增 Go Spike 客户端 `tui/cmd/spike-s3/`，按 TDD 验证目标 Session 的 Permission 过滤和 `session.error` 失败处理。
- 真实事件名为 `permission.asked`，Permission ID 使用 `per_...`；回复枚举为 `once/always/reject`。
- 使用非废弃端点 `POST /permission/{requestID}/reply`。
- 批准分支：`once` 返回 HTTP 200，Tool completed/exit 0，marker 文件存在，Session idle。
- 拒绝分支：`reject` 返回 HTTP 200，Tool error 为用户拒绝，marker 文件不存在，Session idle。
- 两条链路均无 `session.error`；原始证据位于 `docs/spike-artifacts/s3-20260803/`。

### S4 Reasoning — PASS

- 新增 `tui/cmd/spike-s4/`，按 TDD 验证结构化 Reasoning 与 Answer 分类。
- 真实 Runtime 将模型的 `reasoning_content` 转换为独立 `type=reasoning` Part。
- 普通回答为独立 `type=text` Part，两类均可流式接收。
- 最终结果：reasoning=`considering options`，answer=`final answer`，Session idle。
- 不存在 `<think>` 标签；客户端应按 Part 类型分流。

### S5 Skill 来源隔离 — PASS

- 同时构造配置目录、项目、用户、Claude 和 Agents 五类 Skill 来源。
- 隔离组只发现 OpenCode 内置 `customize-opencode` 与批准的 `config-approved`。
- 无隔离对照组重新发现四个未批准 Skill，证明夹具有效且隔离开关生效。
- 必需组合：独立 HOME/XDG、`OPENCODE_CONFIG_DIR`、`OPENCODE_DISABLE_EXTERNAL_SKILLS=1`、`OPENCODE_DISABLE_PROJECT_CONFIG=1`、`OPENCODE_DISABLE_CLAUDE_CODE=1`。
- 无需 OpenCode Patch。
- 已提交可重跑脚本、五类原始 Skill 夹具、完整原始 `/skill` JSON、health、Runtime 日志和夹具 SHA-256 manifest；隔离组 2 项、对照组 6 项的结果可从 JSON 独立推导。

### S6 双模式基础隔离 — PASS

- Enterprise：只加载批准配置 Skill 与 Runtime 内置 Skill，不注入项目 Skill。
- General Compatible：加载合法项目 Skill，仍隔离用户/Claude/Agents 来源。
- General Strict（V1 默认）：不注入项目 Skill。
- 三组均使用独立 Runtime 实例和真实 `/skill` API，结果精确匹配预期集合。
- 三组完整原始 `/skill` JSON 已保存；Enterprise/General Compatible/General Strict 分别为 2/3/2 项，并由 runner 做精确集合断言。

### Phase 0 收尾

- `docs/spike-results.json` 已记录 S1～S6 全 PASS。
- `scripts/run-phase0-gates.sh` 已按 TDD 实现，缺失 S6 会失败，真实结果会通过。
- 固化 OpenCode v1.18.11 OpenAPI 3.1 文档及 76 条 Golden SSE。
- 2026-08-04 使用 Go 1.26.5 完成全量测试与构建，并通过状态机回归、四组 Phase 0 回归、新 S1 证据复核、Shell 语法、版本锁、OpenAPI/Golden SSE 完整性、checkpoint 和凭据扫描；完整门禁以 0 退出。

## 实际文件变更

- `docs/spike-artifacts/s1-20260804-111618/`：新增最终 macOS S1 原始证据，位于 checkpoint `a6755287dd4ab984c47517b9acdd7320776c39cf`。
- `docs/spike-results.json`：S1 从 `blocked` 恢复为 `pass`，S1～S6 全部为 `pass`。
- `docs/execution-state.yaml`：Task 1 Step 10 从 `blocked` 收口为 `awaiting_acceptance`，Verification 与 Task Gate 均改为 `pass`，人工验收保持 `false`。
- `docs/spike-report.md`：补充 20 接口、显式时间窗和独立判定器的最终 S1 证据与结论。
- `docs/task-reports/task-01.md`：同步最终状态、验证命令、偏差和未解决问题。
- `docs/codea-v1-handoff.md`：唯一下一步改为人工验收 Task 1，并保留“不得开始 Task 2”边界。
- `docs/superpowers/plans/2026-08-03-task1-acceptance-blockers-plan.md`：记录完整门禁已执行。

## 执行命令与验证结果

| 命令 | 结果 | 摘要 |
|---|---|---|
| `cd tui && go test ./...` | PASS | Go 1.26.5；S2/S3/S4 测试全部通过 |
| `cd tui && go build ./cmd/codea ./cmd/parity-runner ./cmd/spike-s2 ./cmd/spike-s3 ./cmd/spike-s4` | PASS | 五个入口全部构建通过 |
| `bash tests/execution-state/state_validator_test.sh` | PASS | 当前状态及全部状态机正反例通过 |
| `bash tests/phase0/check_s1_offline_evidence_test.sh` | PASS | S1 判定器失败分支与干净证据回归通过 |
| `bash scripts/check-s1-offline-evidence.sh docs/spike-artifacts/s1-20260804-111618` | PASS | 20 接口证据独立复核通过 |
| `bash tests/phase0/run_skill_isolation_spikes_test.sh` | PASS | S5/S6 可重跑 runner 行为通过 |
| `bash tests/phase0/version_lock_test.sh` | PASS | v1.18.11 四平台资产锁完整且一致 |
| `bash tests/phase0/run_phase0_gates_test.sh` | PASS | Phase 0 Gate 正反例通过 |
| `./scripts/run-phase0-gates.sh` | PASS | S1～S6 全部 PASS |
| `find ... -name '*.sh' ... bash -n` | PASS | 仓库目标 Shell 脚本语法全部通过 |
| `python3` 完整性断言 | PASS | OpenAPI 3.1.0 共 162 paths、Golden SSE 76 条、S1～S6 JSON 全 PASS、版本锁无 TBD |
| `./scripts/check-execution-state.sh` | PASS | Task 1 Step 10 `awaiting_acceptance` 合法 |
| `git cat-file` + `git merge-base --is-ancestor` | PASS | checkpoint 存在且是当前 HEAD 祖先 |
| 凭据模式扫描 | PASS | 未发现 PAT、API Key 或 Authorization 凭据 |
| `git diff --check` | PASS | 无空白错误 |

## 计划偏差与修复

- Task 1 首次进入 `awaiting_acceptance` 后，人工复审发现 S1 退出码传播、S5/S6 可复现证据和四平台哈希三个阻断；已按整改计划补齐并通过回归。
- 后续独立复审发现旧 S1 缺少 `anpi0-2` 抓包和显式启动时间窗，因此没有沿用旧结论，而是恢复为 `blocked` 并在 macOS 使用新版脚本重新采集。
- 最终新证据覆盖全部 20 个活动外部接口并通过独立判定，未降低 S1 门禁要求。

## 未解决问题与恢复建议

- **阻塞项：**无。
- **下一步：**等待人工验收 Task 1。
- **范围边界：**人工验收前不得开始 Task 2。

## Gate 结论

- **Verification (S1–S6):** `pass`
- **Task Gate:** `pass`
- **Human acceptance:** `false`
- **Task 1:** `awaiting_acceptance`
