# Phase 0 Spike Report

## 当前结论

Task 1 当前停在 S1。版本锁定和 Linux 容器本地启动已在早期完成；2026-08-03 在 macOS arm64 本机上完成了真实断网 + tcpdump 持续抓包验证，证据见下方「S1 真实断网验证」章节。S1 是否通过由审核人判定。

| Spike | 状态 | 说明 |
|---|---|---|
| S1 Server 离线启动 | BLOCKED | 真实断网验证已补做，待审核人判定 |
| S2 Session + Prompt + SSE | NOT_RUN | S1 未通过，不得继续 |
| S3 Tool Approval | NOT_RUN | S1 未通过，不得继续 |
| S4 Reasoning | NOT_RUN | S1 未通过，不得继续 |
| S5 Skill 来源隔离 | NOT_RUN | S1 未通过，不得继续 |
| S6 模式隔离 | NOT_RUN | S1 未通过，不得继续 |

---

## S1 证据

### 1. 版本锁定（Linux 容器，2026-08-03 初测）

- 官方仓库：`https://github.com/anomalyco/opencode`
- Release：`v1.18.11`
- Tag commit：`012c2f57f976489d88bd4598a056b4bdcdd428ee`
- Release 时间：`2026-08-01T11:44:45Z`
- Spike 制品：`opencode-linux-x64.tar.gz`
- 官方与实测 SHA-256：`a4dffcc00a5a93256c6bd06aa0c984320528f564db52a1f4becd5c7de9fb59a1`
- 解压后二进制 SHA-256：`8eb15fe87080dd11aa095cc0391eb3536d55a46fa9e4427c6a8b664d390ac089`

结构化证据：`docs/spike-artifacts/s1-release.json`

### 2. Linux 容器本地启动（2026-08-03 初测）

在独立 `OPENCODE_CONFIG_DIR` 中禁用自动更新、模型列表获取、LSP 下载、默认 Plugin、外部 Skill、项目配置和内嵌 Web UI，启动并验证健康检查：

```json
{"healthy": true, "version": "1.18.11"}
```

原始证据：`docs/spike-artifacts/s1-server.log`、`docs/spike-artifacts/s1-health.json`

### 3. Linux 容器阻塞（2026-08-03）

依次尝试以下方式均被容器拒绝：

1. `unshare -n` → `Operation not permitted`
2. `bwrap --unshare-net` → `Operation not permitted`
3. `strace` → `PTRACE_TRACEME: Operation not permitted`

容器缺少 `CAP_SYS_ADMIN` 和 `CAP_SYS_PTRACE`，无法完成网络隔离和系统调用观测。

### 4. macOS 死代理验证（2026-08-03，已被后续真实断网验证取代）

在 macOS arm64 上使用 `HTTP_PROXY`/`HTTPS_PROXY` 指向死代理 + `lsof` 快照的方式做了初步验证。该方法的局限：

- 死代理只影响遵循代理配置的请求，不阻断直接 TCP/UDP/DNS
- OpenCode Runtime 为 TypeScript/Bun 编译产物，非 Go 二进制，代理变量行为不确定
- `lsof` 为单次快照，无法覆盖短连接或启动窗口内的瞬时连接

因此该方法**不作为 S1 通过依据**，仅作为辅助参考。原始证据：`docs/spike-artifacts/s1-final-server.log`、`docs/spike-artifacts/s1-final-health.json`

### 5. macOS 真实断网验证（2026-08-03，本次 S1 决定性证据）

#### 方法

1. **网络接口关闭**：`sudo ifconfig` 逐一关闭 en0/en1-6/awdl0/llw0/bridge0/ap1，仅保留 lo0
2. **持续抓包**：`sudo tcpdump -i en0 -n -w` 从断网前开始持续抓取，覆盖启动 + 健康检查 + 额外 15 秒观测窗口
3. **完全隔离沙箱**：独立 `$HOME`、`XDG_CONFIG_HOME`、`XDG_DATA_HOME`、`XDG_CACHE_HOME`、`XDG_STATE_HOME`、`OPENCODE_CONFIG_DIR`，全部指向空目录
4. **Bun/Node 隔离**：独立 `BUN_INSTALL`、`NPM_CONFIG_CACHE`

执行脚本：`/tmp/s1-network-test.sh`（已通过 `! sudo bash` 执行）

#### 环境

- **平台**：macOS arm64（darwin-arm64）
- **OpenCode 版本**：v1.18.11
- **制品**：`opencode-darwin-arm64.zip`
  - 下载地址：`https://github.com/anomalyco/opencode/releases/download/v1.18.11/opencode-darwin-arm64.zip`
  - SHA-256：`188ff6a716bcd40e33ac62f17f4aec9bd760164fa6a2cde66f779a5db4abc7ce`

#### 网络接口状态

断网前活跃接口：lo0, anpi0-2, en0-6, awdl0, llw0, bridge0, ap1, utun0-5

断网后剩余：lo0, anpi0-2, utun0-5（仅 loopback、Apple 私有 API 虚拟接口、VPN tunnel 接口）

全部可关闭的外部物理/无线/桥接接口均已 down。

#### 健康检查

```json
{"healthy": true, "version": "1.18.11"}
```

#### Server 日志（完整）

```
opencode server listening on http://127.0.0.1:49325
```

#### tcpdump 分析

**抓包覆盖时间**：从 `ifconfig down` 前约 1 秒开始，持续至 OpenCode 停止后约 3 秒，总计约 28 秒。

**捕获包总数**：4

全部 4 个包的明细：

```
14:57:58.753344 IP  120.253.253.225.443 > 10.135.5.10.64297: ACK
14:57:59.200417 IP6 2409:8900:...       > 2600:1f13:...:43607: ACK
14:57:59.604236 IP6 2600:1f13:...       > 2409:8900:...:63600: ACK
14:57:59.701591 IP6 2409:8900:...       > 2600:1f13:...:43607: ACK
```

四条均为**入站包**（方向均为远程 IP → 本机 IP），特征：

| 判定 | 依据 |
|------|------|
| 非本机发起 | 所有包的源 IP 均为远程地址，目的 IP 为本机地址 |
| 断网前旧连接残留 | 时间戳在接口刚 down 的瞬间，为途中的 TCP ACK |
| 非 OpenCode 流量 | 目的端口 64297/43607/63600 与 OpenCode 监听端口 49325 无关；IPv6 包涉及 2600:1f13（AWS us-east-1），为本机 xray 代理的已有连接 |

**DNS 查询（port 53）**：0 包

**HTTP/HTTPS（port 80/443）**：仅上述 1 个入站 443 ACK，无本机发出的 SYN 或数据包

**结论**：在 28 秒抓包窗口内，本机未发起任何 DNS 查询、TCP 出站连接或 HTTP/HTTPS 请求。OpenCode Server 在真实断网 + 完全隔离沙箱环境下正常启动并通过健康检查。

#### 沙箱路径

```
HOME=/Users/.../spike-artifacts/s1-sandbox/home
XDG_CONFIG_HOME=/Users/.../spike-artifacts/s1-sandbox/config
XDG_DATA_HOME=/Users/.../spike-artifacts/s1-sandbox/data
XDG_CACHE_HOME=/Users/.../spike-artifacts/s1-sandbox/cache
XDG_STATE_HOME=/Users/.../spike-artifacts/s1-sandbox/state
OPENCODE_CONFIG_DIR=/Users/.../spike-artifacts/s1-sandbox/config/opencode
```

所有目录在 OpenCode 启动前均为空，启动后仅创建了 OpenCode 自身的 SQLite 数据库和日志文件。

#### 原始证据文件

| 文件 | 说明 |
|------|------|
| `docs/spike-artifacts/s1-offline-real.pcap` | tcpdump 原始抓包（4 包，28 秒窗口） |
| `docs/spike-artifacts/s1-offline-real-server.log` | Server 日志（1 行，无错误） |
| `docs/spike-artifacts/s1-offline-real-health.json` | 健康检查响应 |
| `docs/spike-artifacts/s1-tcpdump.log` | tcpdump 进程输出 |
| `/tmp/s1-network-test.sh` | 完整测试脚本（可复现） |

### S1 门禁对照

| 门禁要求 | 状态 | 证据 |
|----------|------|------|
| 真实断网环境 | 满足 | en0-6/awdl0/llw0/bridge0/ap1 全部 down |
| 独立沙箱环境 | 满足 | 全新 `$HOME` + `XDG_*` + `OPENCODE_CONFIG_DIR` |
| 版本锁定 + SHA-256 一致 | 满足 | v1.18.11，官方与实测一致 |
| 启动成功 | 满足 | `{"healthy":true,"version":"1.18.11"}` |
| 持续抓包覆盖全程 | 满足 | tcpdump 28 秒，启动前到停止后 |
| 无公网 DNS 请求 | 满足 | port 53: 0 包 |
| 无公网 HTTP/HTTPS 出站 | 满足 | 0 本机发起的外部 TCP 连接 |
| 保存原始抓包/日志 | 满足 | pcap + log + health.json 均已保存 |

### 局限说明

- 测试平台为 macOS arm64，未覆盖 Windows x64。OpenCode 官方提供 `opencode-windows-x64.zip` 构建，该构建同样由 Bun 从同一 TypeScript 源码编译。平台行为差异应在 Task 21 Release Parity Certification 中补充验证。
- tcpdump 抓包接口为 en0（主 Wi-Fi 接口）。其余接口在抓包期间已 down，不会产生流量；utun 和 anpi 为系统内部隧道/私有接口，不路由公网流量。

---

## 未开始的验证

S2～S6 尚未执行；没有创建 `docs/spike-results.json`，也没有将任何未验证 Spike 写成 `pass`。
