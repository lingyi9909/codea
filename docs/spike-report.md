# Phase 0 Spike Report

## 当前结论

**S1 判定：FAIL。** OpenCode v1.18.11 在断网环境下能够启动并通过 `/global/health`，但内部日志确认启动时两次尝试访问 `GET https://models.opencode.ai/api.json`，不满足 S1 门禁「启动成功，不访问公网」。

| Spike | 状态 | 说明 |
|---|---|---|
| S1 Server 离线启动 | **FAIL** | 内部日志确认收到 `models.opencode.ai` 请求；v1.18.11 无支持的离线配置 |
| S2 Session + Prompt + SSE | NOT_RUN | S1 未通过，不得继续 |
| S3 Tool Approval | NOT_RUN | S1 未通过，不得继续 |
| S4 Reasoning | NOT_RUN | S1 未通过，不得继续 |
| S5 Skill 来源隔离 | NOT_RUN | S1 未通过，不得继续 |
| S6 模式隔离 | NOT_RUN | S1 未通过，不得继续 |

**阻塞原因**：OpenCode v1.18.11 在启动时硬编码请求 `https://models.opencode.ai/api.json` 以获取模型列表缓存。CLI 参数、环境变量和配置文件中均未提供禁用该请求的机制。上游有同类问题报告（[anomalyco/opencode#10766](https://github.com/anomalyco/opencode/issues/10766)、[anomalyco/opencode#16117](https://github.com/anomalyco/opencode/issues/16117)），表明这是上游已知但尚未解决的限制。

**解决方向**：需要对 OpenCode v1.18.11 打最小 Patch，在启动路径中跳过 `models.dev` 的模型目录获取，或提供可配置的禁用开关。

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

### S1 门禁对照

| 门禁要求 | 状态 | 证据 |
|----------|------|------|
| 真实断网环境 | 满足 | en0-6/awdl0/llw0/bridge0/ap1 全部 down |
| 独立沙箱环境 | 满足 | 全新 `$HOME` + `XDG_*` + `OPENCODE_CONFIG_DIR` |
| 版本锁定 + SHA-256 一致 | 满足 | v1.18.11，linux/darwin 校验一致 |
| Server 启动成功 | 满足 | `{"healthy":true,"version":"1.18.11"}` |
| **不访问公网** | **不满足** | 内部日志两次 `GET https://models.opencode.ai/api.json` |
| 无公网 DNS 请求 | **无法判定** | tcpdump 仅覆盖 en0，utun 未监听 |
| 无公网 HTTP/HTTPS 出站 | **不满足** | 同上，内部日志已证明请求尝试 |

### 离线配置调查

针对「v1.18.11 是否有受支持的配置禁用模型目录获取」进行了以下调查：

1. **CLI 参数**：`opencode serve --help` 无离线相关参数，`--pure` 仅禁用外部 Plugin，不跳过模型获取。
2. **环境变量**：`OPENCODE_OFFLINE_MODE`、`OPENCODE_SKIP_MODEL_FETCH` 等变量（由我们自行尝试设置）对 v1.18.11 无效——这些变量在 OpenCode 源码中不存在。
3. **配置文件**：OpenCode 尝试加载 `config.json`、`opencode.json`、`opencode.jsonc`，但无文档说明可禁用模型目录获取的配置键。
4. **上游 Issue**：
   - [Failed to fetch models.dev](https://github.com/anomalyco/opencode/issues/10766) — 同类问题报告
   - [Offline mode proposal](https://github.com/anomalyco/opencode/issues/16117) — 离线模式提议，尚未合并

**结论**：v1.18.11 没有受支持的方式禁用 `models.opencode.ai` 请求。需要最小 Patch。

### 测试脚本已知缺陷

当前 `docs/spike-artifacts/s1-network-test.sh` 存在以下问题，需在下次验证前修复：

1. **无 `trap` 机制**：`set -e` 下任何步骤失败（如断网后 OpenCode 未启动）会直接退出，网络接口无法恢复。
2. **抓包覆盖不全**：仅监听 `en0`，未覆盖保持活跃的 `utun0-5` 接口。应监听所有接口或至少 `en0` + `utun*`。
3. **未保存完整运行日志**：脚本 stdout/stderr 未重定向到文件，无法回溯执行过程以验证步骤是否全部完成。
4. **仅保存 stdout 而非内部日志**：之前将 Server 的一行 stdout 当作「Server 日志（完整）」，遗漏了 `$XDG_DATA_HOME/opencode/log/opencode.log` 中的内部日志（含两次模型请求的 ERROR 记录）。

下次运行前需修复以上四项，并将完整执行日志和 OpenCode 内部日志一并保存为证据。

---

## 原始证据文件

| 文件 | 说明 |
|------|------|
| `docs/spike-artifacts/s1-release.json` | 版本锁定结构化数据 |
| `docs/spike-artifacts/s1-server.log` | Linux 容器 Server stdout |
| `docs/spike-artifacts/s1-health.json` | Linux 容器健康检查响应 |
| `docs/spike-artifacts/s1-offline-real.pcap` | macOS tcpdump（en0 only，4 包） |
| `docs/spike-artifacts/s1-offline-real-server.log` | macOS Server stdout（1 行） |
| `docs/spike-artifacts/s1-offline-real-health.json` | macOS 健康检查响应 |
| `docs/spike-artifacts/s1-tcpdump.log` | tcpdump 进程输出 |
| `docs/spike-artifacts/s1-network-test.sh` | 测试脚本 |

以下文件位于沙箱中，未提交到仓库（路径见报告内文）：

- `$XDG_DATA_HOME/opencode/log/opencode.log` — OpenCode 内部日志（含 models.dev 错误）

---

## 下一步

1. **确认 Patch 策略**：评估对 OpenCode v1.18.11 打最小 Patch 以跳过模型目录获取的可行性和工作量。如可行，Patch 应确保在离线配置下 `models.opencode.ai` 请求不被发起，OpenCode 内部日志不出现相关 ERROR。
2. **修复测试脚本**：补齐 trap、全接口抓包、完整执行日志保存。
3. **重新验证**：Patch 后在新隔离沙箱中运行，确认健康检查通过 + 内部日志无公网请求 + 全接口抓包无 DNS/HTTP 出站。
4. **通过后更新状态**：S1 → PASS，Task 1 → in_progress Step 2。

---

## 未开始的验证

S2～S6 尚未执行；没有创建 `docs/spike-results.json`，也没有将任何未验证 Spike 写成 `pass`。
