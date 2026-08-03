# Task 01 Report — Phase 0 Spike S1–S6

**Task:** 1

**Status:** blocked

**Current step:** 1 — S1 Server 离线启动

**Date:** 2026-08-03

**Checkpoint:** `f09d9b8262d03438fee4728e551f889d03179c93`

## 已完成内容

- 锁定 OpenCode `v1.18.11`，Tag commit 为 `012c2f57f976489d88bd4598a056b4bdcdd428ee`。
- 下载官方 `opencode-linux-x64.tar.gz`，官方与实测 SHA-256 均为 `a4dffcc00a5a93256c6bd06aa0c984320528f564db52a1f4becd5c7de9fb59a1`。
- 在独立配置目录和离线相关开关下启动 Server。
- 使用临时 Basic Auth 调用 `/global/health`，返回 `healthy: true`、`version: 1.18.11`。
- 更新 `runtime/version.json` 和 `runtime/capabilities.yaml` 的锁定版本。
- 保存原始版本、启动和健康检查证据，详见 `docs/spike-report.md`。

## 阻塞项

S1 必须证明 OpenCode 在真实断网环境启动，并确认启动期间没有公网 DNS/HTTP 请求。当前容器：

- `unshare -n`：`Operation not permitted`
- `bwrap --unshare-net`：`Operation not permitted`
- `strace`：`PTRACE_TRACEME: Operation not permitted`

因此无法完成必要的网络隔离和调用观测。仅凭本地启动成功不足以将 S1 标记为 `pass`。

## 恢复动作

在允许控制网络和抓包的环境，使用已锁定的 v1.18.11 制品补做：

1. 禁止 OpenCode 进程访问公网。
2. 启动 `opencode serve --hostname 127.0.0.1 --port 49321`。
3. 调用 `/global/health` 并保存响应。
4. 保存 DNS、HTTP、HTTPS 出站观测，确认没有公网请求。
5. 证据通过后将 S1 标记为 `pass`，再开始 S2。

## Gate 结论

- **Verification:** `unable_to_run`
- **Task Gate:** `unable_to_evaluate`
- **Human acceptance:** `false`
- **Task 1:** `blocked`
- **S2–S6:** 未开始
