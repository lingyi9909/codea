# Task 01 Report — Phase 0 Spike S1–S6

**Task:** 1

**Status:** in_progress

**Current step:** 4 — S4 Reasoning 事件结构

**Date:** 2026-08-03

**Checkpoint:** `7d96468b415f6ef6c09206bacc59eae61b60221a`

## 已完成内容

### S1 Server 离线启动 — PASS

- 锁定 OpenCode v1.18.11，Linux x64 和 macOS arm64 制品 SHA-256 校验一致。
- 在 macOS arm64 上完成真实断网验证：
  - 关闭 en0-6/awdl0/llw0/bridge0/ap1 全部外部接口
  - 完全隔离沙箱：独立 `$HOME`、`XDG_*`、`OPENCODE_CONFIG_DIR`
  - tcpdump 9 接口持续抓包
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
  - tcpdump：31 包全在 en0 且时间戳在断网前，来自非 OpenCode 进程；其余接口零包

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

## 下一步

S4：Reasoning 事件结构验证。

## Gate 结论

- **Verification (S1–S3):** `pass`
- **Task Gate:** `not_evaluated`（待 S1–S6 全部通过）
- **Human acceptance:** `false`
- **Task 1:** `in_progress`
