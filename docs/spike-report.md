# Phase 0 Spike Report

## 当前结论

**S1 判定：PASS。** 使用正确的 `OPENCODE_DISABLE_MODELS_FETCH=1` 后，OpenCode v1.18.11 在真实断网 + 完全隔离沙箱环境下启动成功，内部日志零 ERROR、零 `models.opencode.ai` 请求，全接口抓包无 OpenCode 相关出站流量。

| Spike | 状态 | 说明 |
|---|---|---|
| S1 Server 离线启动 | **PASS** | 真实断网 + 正确环境变量，内部日志无公网请求 |
| S2 Session + Prompt + SSE | NOT_RUN | 待开始 |
| S3 Tool Approval | NOT_RUN | 待开始 |
| S4 Reasoning | NOT_RUN | 待开始 |
| S5 Skill 来源隔离 | NOT_RUN | 待开始 |
| S6 模式隔离 | NOT_RUN | 待开始 |

**关键发现**：初版验证使用了不存在的环境变量名（`OPENCODE_SKIP_MODEL_FETCH`），导致 OpenCode 仍发起 `models.opencode.ai` 请求。v1.18.11 官方已支持 `OPENCODE_DISABLE_MODELS_FETCH=1` 禁用该请求，无需 Patch。

---

## S1 证据

### 1. 版本锁定（Linux 容器，2026-08-03）

| 项目 | 值 |
|------|-----|
| 官方仓库 | `https://github.com/anomalyco/opencode` |
| Release | `v1.18.11` |
| Tag commit | `012c2f57f976489d88bd4598a056b4bdcdd428ee` |
| Release 时间 | `2026-08-01T11:44:45Z` |
| linux-x64 SHA-256 | `a4dffcc00a5a93256c6bd06aa0c984320528f564db52a1f4becd5c7de9fb59a1` |
| darwin-arm64 SHA-256 | `188ff6a716bcd40e33ac62f17f4aec9bd760164fa6a2cde66f779a5db4abc7ce` |

结构化证据：`docs/spike-artifacts/s1-release.json`

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
| 版本锁定 + SHA-256 一致 | 满足 | v1.18.11，linux/darwin 校验一致 |
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

---

## 原始证据文件

| 文件 | 说明 |
|------|------|
| `docs/spike-artifacts/s1-release.json` | 版本锁定结构化数据 |
| `docs/spike-artifacts/s1-server.log` | Linux 容器 Server stdout |
| `docs/spike-artifacts/s1-health.json` | Linux 容器健康检查响应 |
| `docs/spike-artifacts/s1-network-test.sh` | 修正版测试脚本（trap + 全接口 + 正确环境变量） |
| `docs/spike-artifacts/s1-20260803-175535/` | 修正版验证完整证据目录 |
| `docs/spike-artifacts/s1-20260803-175535/execution.log` | 脚本完整执行日志 |
| `docs/spike-artifacts/s1-20260803-175535/health.json` | 健康检查响应 |
| `docs/spike-artifacts/s1-20260803-175535/server-stdout.log` | Server stdout |
| `docs/spike-artifacts/s1-20260803-175535/opencode-internal.log` | OpenCode 内部日志（3 行 INFO，零 ERROR） |
| `docs/spike-artifacts/s1-20260803-175535/traffic-*.pcap` | 逐接口 tcpdump 原始抓包（9 个接口） |

---

## 未开始的验证

S2～S6 尚未执行；没有创建 `docs/spike-results.json`。
