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

## 未开始的验证

S2～S6 尚未执行；没有创建 `docs/spike-results.json`，也没有将任何未验证 Spike 写成 `pass`。
