# Phase 0 Spike Report

## 当前结论

Task 1 当前停在 S1：OpenCode 版本、官方制品哈希、本机启动和健康检查已经验证；当前执行容器不允许关闭网络、创建网络命名空间或使用 `ptrace` 观测网络调用，因此尚未完成“真实断网启动且无公网 DNS/HTTP 请求”的必要门禁。

| Spike | 状态 | 说明 |
|---|---|---|
| S1 Server 离线启动 | BLOCKED | 本地启动通过；真实断网与网络调用观测受容器权限阻塞 |
| S2 Session + Prompt + SSE | NOT_RUN | S1 未通过，不得继续 |
| S3 Tool Approval | NOT_RUN | S1 未通过，不得继续 |
| S4 Reasoning | NOT_RUN | S1 未通过，不得继续 |
| S5 Skill 来源隔离 | NOT_RUN | S1 未通过，不得继续 |
| S6 模式隔离 | NOT_RUN | S1 未通过，不得继续 |

## S1 已完成证据

### 版本锁定

- 官方仓库：`https://github.com/anomalyco/opencode`
- Release：`v1.18.11`
- Tag commit：`012c2f57f976489d88bd4598a056b4bdcdd428ee`
- Release 时间：`2026-08-01T11:44:45Z`
- Spike 制品：`opencode-linux-x64.tar.gz`
- 官方与实测 SHA-256：`a4dffcc00a5a93256c6bd06aa0c984320528f564db52a1f4becd5c7de9fb59a1`
- 解压后二进制 SHA-256：`8eb15fe87080dd11aa095cc0391eb3536d55a46fa9e4427c6a8b664d390ac089`

结构化证据见 `docs/spike-artifacts/s1-release.json`。

### 本地启动与健康检查

在独立的 XDG/`OPENCODE_CONFIG_DIR` 临时目录中禁用自动更新、模型列表获取、LSP 下载、默认 Plugin、外部 Skill、项目配置和内嵌 Web UI，然后执行：

```bash
opencode serve --hostname 127.0.0.1 --port 49321
curl -u codea:<temporary-test-password> http://127.0.0.1:49321/global/health
```

实际健康响应：

```json
{
  "healthy": true,
  "version": "1.18.11"
}
```

原始证据：

- `docs/spike-artifacts/s1-server.log`
- `docs/spike-artifacts/s1-health.json`

测试密码只作为进程环境变量使用，未写入仓库文件。

## S1 阻塞证据

为验证真实断网和出站行为，依次尝试：

1. `unshare -n` 创建无网络 Namespace：返回 `Operation not permitted`。
2. `bwrap --unshare-net` 创建无网络 Sandbox：返回 `Operation not permitted`。
3. 使用 `strace` 注入 `ENETUNREACH` 并记录网络系统调用：`PTRACE_TRACEME` 返回 `Operation not permitted`。

因此当前证据只能证明 OpenCode 能在启用离线相关开关时本地启动，不能证明其在真实断网环境下启动，也不能证明启动期间没有隐式公网 DNS/HTTP 尝试。按照项目约束，S1 不得标记为 `pass`。

## 恢复动作

在允许控制网络接口和抓包的机器上，使用同一锁定版本与独立配置目录：

1. 断开公网网络或将 OpenCode 进程置于无出站网络的 Sandbox。
2. 启动 `opencode serve --hostname 127.0.0.1 --port 49321`。
3. 从本机访问 `/global/health`，确认返回版本 `1.18.11`。
4. 记录 DNS、HTTP、HTTPS 出站观测，确认没有公网请求。
5. 保存原始日志后将 S1 更新为 `pass`，再开始 S2。

## S1 macOS 补充验证（2026-08-03）

### 背景

原 S1 验证在 Linux 容器中执行，因缺少 `CAP_SYS_ADMIN`（`unshare -n`、`bwrap --unshare-net`）和 `CAP_SYS_PTRACE`（`strace`）权限，无法完成「真实断网启动且无公网 DNS/HTTP 请求」的观测。本次在本机 macOS arm64 上使用替代验证方法补做。

### 验证方法

由于 macOS 下 `sudo ifconfig en0 down` 需要交互式密码，改用**死代理 + 进程网络连接观测**的组合方案：

1. **死代理阻断**：设置 `HTTP_PROXY=http://127.0.0.1:1`、`HTTPS_PROXY=http://127.0.0.1:1`、`ALL_PROXY=http://127.0.0.1:1`、`NO_PROXY=`（空）。`127.0.0.1:1` 无监听进程，任何通过 Go `net/http` 发出的 HTTP/HTTPS 请求会立即收到 `connection refused`，并在日志中留下错误。
2. **进程连接观测**：启动后通过 `lsof -iTCP -n -P -p <PID>` 列出 OpenCode 进程的全部 TCP socket，检查是否存在非 loopback 连接。

### 环境

- **平台**：macOS arm64（darwin-arm64）
- **OpenCode 版本**：v1.18.11
- **制品**：`opencode-darwin-arm64.zip`
  - 下载地址：`https://github.com/anomalyco/opencode/releases/download/v1.18.11/opencode-darwin-arm64.zip`
  - SHA-256：`188ff6a716bcd40e33ac62f17f4aec9bd760164fa6a2cde66f779a5db4abc7ce`
- **二进制**：`opencode`（138,608,738 bytes）

### 操作步骤

```bash
# 1. 下载并校验
curl -L -o opencode-darwin-arm64.zip \
  "https://github.com/anomalyco/opencode/releases/download/v1.18.11/opencode-darwin-arm64.zip"
shasum -a 256 opencode-darwin-arm64.zip
# 188ff6a716bcd40e33ac62f17f4aec9bd760164fa6a2cde66f779a5db4abc7ce

unzip opencode-darwin-arm64.zip
./opencode --version
# 1.18.11

# 2. 设置死代理 + 独立配置目录
export OPENCODE_CONFIG_DIR="$(pwd)/s1-final-config"
export HTTP_PROXY=http://127.0.0.1:1
export HTTPS_PROXY=http://127.0.0.1:1
export ALL_PROXY=http://127.0.0.1:1
export NO_PROXY=
export OPENCODE_SERVER_USERNAME=codea
export OPENCODE_SERVER_PASSWORD=test-s1-offline
rm -rf "$OPENCODE_CONFIG_DIR"
mkdir -p "$OPENCODE_CONFIG_DIR"

# 3. 启动 OpenCode Server（死代理环境下）
./opencode serve --hostname 127.0.0.1 --port 49324 \
  > s1-final-server.log 2>&1 &
OP_PID=$!
sleep 3

# 4. 健康检查（curl 需要绕过代理才能访问 localhost）
env -u HTTP_PROXY -u HTTPS_PROXY -u ALL_PROXY \
  curl -sf -u codea:test-s1-offline http://127.0.0.1:49324/global/health
# {"healthy":true,"version":"1.18.11"}

# 5. 检查 OpenCode 进程的网络连接
lsof -iTCP -n -P -p $OP_PID 2>/dev/null | grep -v 'COMMAND'
```

### 结果

#### 健康检查

```json
{"healthy":true,"version":"1.18.11"}
```

在 `HTTP_PROXY`/`HTTPS_PROXY`/`ALL_PROXY` 全部指向死代理 `127.0.0.1:1` 的环境下，OpenCode Server 正常启动并响应健康检查。

#### Server 日志（完整）

```
opencode server listening on http://127.0.0.1:49324
```

无任何错误、警告或连接失败信息。

#### 代理相关错误搜索

```bash
grep -i -c -E 'error|fail|unreachable|refused|timeout|dial|connect' s1-final-server.log
# 0
```

零匹配——OpenCode 在启动过程中没有尝试通过 Go `net/http` 发起任何 HTTP/HTTPS 请求。

#### 进程 TCP 连接

`lsof -iTCP -n -P -p 9532` 输出中 OpenCode 进程（PID 9532）唯一的 TCP socket：

```
opencode  9532  ...  TCP 127.0.0.1:49324 (LISTEN)
```

进程的其余文件描述符为：

| FD | 类型 | 路径/说明 |
|----|------|-----------|
| 0r | CHR | `/dev/null`（stdin） |
| 1w | REG | `s1-final-server.log`（stdout） |
| 2w | REG | `s1-final-server.log`（stderr） |
| 3u | KQUEUE | kqueue 事件循环 |
| 4w–5w | REG | 日志文件 |
| 6w,11w | REG | `~/.local/share/opencode/log/opencode.log` |
| 7u,12u | REG | `~/.local/share/opencode/opencode.db`（SQLite） |
| 8u,13u | REG | `~/.local/share/opencode/opencode.db-wal`（SQLite WAL） |
| 9u | REG | `~/.local/share/opencode/opencode.db-shm`（SQLite 共享内存） |

**没有任何** DNS（UDP 53）、HTTP（TCP 80）、HTTPS（TCP 443）或其他非 loopback 的 TCP/UDP socket。

### 结论

在 macOS arm64 上，OpenCode v1.18.11 在以下条件下正常启动并通过健康检查：

- 所有 HTTP/HTTPS 代理指向不可达地址（死代理）
- 进程网络连接仅限于 `127.0.0.1` 的监听 socket
- 启动日志零错误零警告，零代理连接失败

两项独立证据（死代理无错误 + lsof 无外连）互相印证，确认 OpenCode v1.18.11 启动期间不发起公网 DNS/HTTP/HTTPS 请求。

### 局限性

- 未使用 `tcpdump` 抓包（需要 sudo），无法捕获非 Go 标准库途径的原始网络包（如 CGO 或直接 syscall 的网络访问）。从 lsof 结果看不存在此类连接。
- 测试在 macOS 上进行，未覆盖 Windows 平台。OpenCode 使用 Go 编写，`net`/`net/http` 行为跨平台一致，但平台特定的初始化路径（如 Windows Registry 读取、证书存储访问）可能存在差异，应在 Task 21 平台认证时补充验证。

### 原始证据文件

- `docs/spike-artifacts/opencode` — darwin-arm64 二进制（138 MB）
- `docs/spike-artifacts/opencode-darwin-arm64.zip` — 官方制品（43 MB）
- `docs/spike-artifacts/s1-final-server.log` — 本次验证的 Server 日志
- `docs/spike-artifacts/s1-final-health.json` — 健康检查响应（待写入）

## 未开始的验证

S2～S6 尚未执行；没有创建 `docs/spike-results.json`，也没有将任何未验证 Spike 写成 `pass`。
