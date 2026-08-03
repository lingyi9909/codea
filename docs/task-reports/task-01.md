# Task 01 Report — Phase 0 Spike S1–S6

**Task:** 1

**Status:** in_progress

**Current step:** 2 — S2 Session + Prompt + SSE 全链路

**Date:** 2026-08-03

**Checkpoint:** `06bb850e6876014e4024fdd98d6744ba2b0626a7`

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

## 下一步

S2：Go Session + Prompt + SSE 全链路验证。

## Gate 结论

- **Verification (S1):** `pass`
- **Task Gate:** `not_evaluated`（待 S1–S6 全部通过）
- **Human acceptance:** `false`
- **Task 1:** `in_progress`
