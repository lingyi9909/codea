# Task 5 Report — Runtime Supervisor + Basic Auth + 跨平台进程管理

## Overview

Checkpoint: cbe692aaff6700c257d3e0806801dcf8d3396870

Runtime 进程生命周期管理：`RuntimeSupervisor` 负责 Start/Stop/Status（不进入 `AgentRuntime`）、每次启动随机 Basic Auth、`127.0.0.1` 只监听、跨平台（darwin/Windows）进程组信号控制、readiness 探测、Crash 检测，以及 Supervisor 启动的 Runtime 可被 `OpenCodeAdapter` 直接驱动的集成契约。

## Step 1 — RuntimeStatus Domain + Supervisor 状态机

- Created `tui/internal/runtime/status.go`：
  - `RuntimeStatus` 字符串类型 + 5 个状态常量：`RuntimeStopped` / `RuntimeStarting` / `RuntimeHealthy` / `RuntimeStopping` / `RuntimeCrashed`
- Created `tui/internal/supervisor/supervisor.go`：
  - `Config{OpenCodeBin, Hostname, Port, ConfigDir, ProjectRoot, StartupTimeout, StopTimeout}`
  - `Supervisor` 结构：`mu` 保护 `status/cmd/port/username/password/lastErr/exitCh`
  - `NewSupervisor` 默认值：hostname=`127.0.0.1`、username=`opencode`、StartupTimeout=30s、StopTimeout=5s
  - `Start` → `startProcess`（单次加锁完成状态迁移 + 进程 spawn，杜绝并发 Start 竞态）→ `monitor` goroutine → `waitForReady` → `Healthy`
  - `monitor` 是**唯一** `cmd.Wait()` 调用方，进程退出后关闭 `exitCh`；非 Stop 流程退出 → `Crashed` + `lastErr`
  - `Stop` 幂等；`Stopped/Crashed` 上为安全 no-op；`Stopping` 上等待 `exitCh`
  - `cleanupFailedStart` 处理「已 spawn 但未 ready」的失败启动
  - 使用 `exec.Command`（**非** `CommandContext`，避免 Start ctx 取消误杀进程）
- Tests（`supervisor_test.go`）：`TestDefaultStatusStopped` / `TestStartReachesHealthy` / `TestStartWhileHealthyErrors` / `TestStopIdempotent` / `TestUnexpectedExitCrashes` / `TestConcurrentStartSingleProcess` / `TestRestartAfterStop`

## Step 2 — Basic Auth + 启动参数/环境变量

- `generatePassword()`：`crypto/rand` 32 字节 → 64 hex 字符（每次启动随机）
- `buildEnv`：注入 `OPENCODE_SERVER_USERNAME` / `OPENCODE_SERVER_PASSWORD` / `OPENCODE_CONFIG_DIR` + 6 个离线变量（`OPENCODE_DISABLE_CLAUDE_CODE` / `MODELS_FETCH` / `AUTOUPDATE` / `EMBEDDED_WEB_UI` / `LSP_DOWNLOAD` / `DEFAULT_PLUGINS`）
- `buildArgs`：`serve --hostname <host> --port <port>`，密码**绝不**进入 args（`TestBuildArgsNeverContainsPassword`）；只绑定 `127.0.0.1`（`TestBuildArgsBindsLocalhostNotWildcard`）
- Tests（`auth_test.go`）：密码非空/长度 64/唯一性、username 默认 opencode、hostname 默认 127.0.0.1、env 携带凭据/ConfigDir/离线变量、args 形状与不含密码

## Step 3 — Port 自动分配 + Ready 探测

- `findFreePort()`：`net.Listen("tcp", "127.0.0.1:0")` 自动选口；`Config.Port=0` 时使用
- `probeReady()`：GET `/global/health` + Basic Auth，要求 200 **且** `healthy:true`；`probeClient` 2s 超时防挂起；`readyInterval` 200ms 轮询
- `waitForReady` 监听 ctx.Done / deadline / exitCh / ticker 四路
- Tests（`readiness_test.go`）：healthy/带 BasicAuth/401/500/`healthy:false` 拒绝、端口有效性、固定端口、BaseURL、auth-required 启动、startup timeout → Crashed、ctx cancel 中断、进程即退 fail-fast

## Step 4 — darwin 进程组控制

- Created `tui/internal/supervisor/process_unix.go`（`//go:build darwin`）：
  - `configureProcess`：`SysProcAttr{Setpgid: true}`（新进程组，整树可信号）
  - `terminateProcess`：`syscall.Kill(-pid, SIGTERM)`（整组优雅退出）
  - `killProcess`：`syscall.Kill(-pid, SIGKILL)`（整组强杀）
- Tests（`process_unix_test.go`，`//go:build darwin`）：`TestStopGraceful` / `TestStopForceKillFallback`（忽略 SIGTERM → 强杀兜底）/ `TestChildProcessNotLeftBehind`（子进程一并清理，无孤儿）/ `TestStopAfterCrashNoPanic`

## Step 5 — Windows 进程组控制（cross-build 验证）

- Created `tui/internal/supervisor/process_windows.go`（`//go:build windows`）：
  - `configureProcess`：`CREATE_NEW_PROCESS_GROUP`
  - `terminateProcess`：`GenerateConsoleCtrlEvent(CTRL_BREAK_EVENT)`（优雅请求，best-effort）
  - `killProcess`：`TerminateProcess` 兜底
- Cross-build 验证：`GOOS=windows GOARCH=amd64 go build ./internal/supervisor/...` 与 `go build ./cmd/codea ./cmd/parity-runner` 均通过

## Step 6 — Supervisor ↔ OpenCodeAdapter 集成契约

- Created `tui/tests/contract/supervisor_adapter_contract_test.go`（位于 `tests/contract`，通过 Runtime 边界门禁）：
  - `Supervisor.Start()` → `Healthy` → 取 `BaseURL/Username/Password` → `NewOpenCodeAdapter(...)` → `Health()` 成功 → 错密码 `IsAuth` 失败 → `CreateSession()` → `Subscribe()` 收到 `runtime.connected` → `Stop()` 后端口不可达 + `Stopped` → 重启获得**新密码** + 有效新端口 → Adapter 用新凭证工作、旧凭证失效
  - `isolateOpenCodeEnv` 把 `HOME`/`XDG_*` 指向临时目录，真实 OpenCode 不触碰开发者真实配置/缓存，保持纯离线
- 真实 OpenCode v1.18.11 冒烟通过（`docs/spike-artifacts/opencode`，darwin-arm64）：`TestSupervisorAdapterContract` PASS（1.17s，未 skip）

## Full Gate Verification

针对 Final Implementation Commit `bf6fd3e6ada5e1b2b97df61cb8eadafc3da9ef38`：

| Gate | Result |
|------|--------|
| `go test ./... -count=1` | PASS（15 packages） |
| `go test -race ./... -count=1` | PASS（无竞态） |
| `go vet ./...` | clean |
| `go build ./...` | clean |
| `GOOS=windows GOARCH=amd64 go build ./cmd/codea ./cmd/parity-runner` | PASS |
| `GOOS=darwin GOARCH=amd64 go build ./cmd/codea ./cmd/parity-runner` | PASS |
| `check-runtime-boundary.sh` | PASS（无 vendor DTO 泄漏） |
| `check-execution-state.sh` | valid |
| `state_validator_test.sh` | valid |
| `check-opencode-available.sh` | available |

Task 5 专项契约：Supervisor lifecycle / Basic Auth / readiness / crash detection / graceful shutdown / force-kill fallback / real OpenCode supervisor smoke / offline regression 全部通过。

## Windows 真实 lifecycle smoke — unable_to_run

本机无 Windows 真机，`Windows x64 real lifecycle smoke`（真机进程树清理验证）无法执行。Windows 侧已通过 **x64 cross-build** 验证编译正确性；真机行为按清单第 15 节标记 `unable_to_run`，补齐 Windows 环境后再验。

## 遗留说明

本机存在一个 Task 4 遗留的 OpenCode 进程（`/tmp/opencode serve --port 14242`，Task 4 人工 smoke 遗留，非 Task 5 Supervisor 产生）。Task 5 全部测试使用随机端口并显式 `Stop()`，`TestChildProcessNotLeftBehind` 证明 Supervisor 无孤儿进程。

## Round 1 Review — 人工验收反馈与修复

人工验收结论：**暂不通过，Task 5 保持 blocked**，2 个阻塞问题。

### Blocking 1 — Windows force-kill 未清理整棵进程树

原 `killProcess()` = `cmd.Process.Kill()` 只杀主进程，遗留子进程孤儿。修复为 **Windows Job Object**：

- `attachProcess(cmd)`：`CreateJobObjectW` 创建 Job → `SetInformationJobObject` 设置 `JOB_OBJECT_LIMIT_KILL_ON_JOB_CLOSE` → `AssignProcessToJobObject` 把启动后的 opencode 挂入 Job，句柄存入 `sync.Map`（`*exec.Cmd` → `syscall.Handle`）
- `killProcess(cmd)`：`TerminateJobObject` 整树强杀 + `CloseHandle`
- `detachProcess(cmd)`（`monitor` 在 `cmd.Wait()` 后调用）：`LoadAndDelete` 关闭句柄，`KILL_ON_JOB_CLOSE` 保证 opencode 崩溃后无孤儿
- 全部 Win32 调用经 `syscall.NewLazyDLL`（无 x/sys 依赖，保持纯离线）

新增 `process_windows_test.go`（`//go:build windows`，真机运行）：

- `TestStopTerminatesProcessTree`：spawn 子进程 → Stop → 父/子进程均消失（0 orphan）
- `TestStopForceKillFallback`：优雅退出无效 → StopTimeout 后 `TerminateJobObject` 强杀

### Blocking 2 — Start→Healthy 与 monitor→Crashed 竞态

原 `Start` 无条件 `s.status = runtime.RuntimeHealthy`，可能覆盖 monitor 已写入的 `Crashed`，遗留 stale Healthy。修复：

- 新增 `runID uint64`（每次启动自增的 run generation），`startProcess` 返回 runID
- 新增 `markHealthy(runID) bool`：锁内 CAS 校验 `s.runID == runID && s.status == Starting` 才置 `Healthy`，否则返回 false（Start 返回错误）
- `monitor`/`cleanupFailedStart` 以 runID 守卫，陈旧 monitor 不再污染新实例

新增白盒单测（确定性验证 CAS 不变量）：

- `TestMarkHealthyAcceptsCurrentStarting` / `TestMarkHealthyRejectsAfterCrash` / `TestMarkHealthyRejectsStaleRun`

新增集成回归测试 `TestHealthyThenExitSettlesCrashed`：fake 返回 `healthy:true` 后立即退出（`FAKE_OPENCODE_MODE=healthy-then-exit`，固定 body + Content-Length + Flush 避免截断）→ 最终状态必为 `Crashed`，永不为 `Healthy`。

另补「path/configDir with spaces」跨平台测试（`paths_test.go`，test-only commit `de4e9db`）：`TestBuildEnvConfigDirWithSpaces`（env 携带空格路径逐字透传）+ `TestStartWithSpacesInPaths`（OpenCodeBin/ConfigDir/ProjectRoot 三处含空格仍 Start→Healthy），补齐 Windows Required Gate 的空格路径项。

### Round 1 验证

| Gate | Result |
|------|--------|
| `go test ./... -count=1` | PASS（15 packages） |
| `go test -race ./internal/supervisor/... -count=1` | PASS（无竞态） |
| `go vet ./...` | clean |
| `go build ./...` | clean |
| `GOOS=windows GOARCH=amd64 go build ./cmd/codea ./cmd/parity-runner` | PASS |
| `GOOS=darwin GOARCH=amd64 go build ./cmd/codea ./cmd/parity-runner` | PASS |
| `GOOS=windows GOARCH=amd64 go test -c ./internal/supervisor/` | PASS（Windows 测试编译通过） |
| `check-runtime-boundary.sh` | PASS |
| `check-execution-state.sh` | valid |

Windows 真机 lifecycle smoke（`TestStopTerminatesProcessTree` / `TestStopForceKillFallback`）仍 `unable_to_run`（本机无 Windows 真机），Task 5 保持 blocked。

## Round 2 Review — localhost-only 强制 + 最终收口

人工验收反馈：Task 5 的「127.0.0.1 only」声明与实际实现不一致——`NewSupervisor(Config{Hostname: "0.0.0.0"})` 仍会启动 `opencode serve --hostname 0.0.0.0`，违背「never 0.0.0.0」的注释与安全约束。修复为**硬锁 loopback**：

- `NewSupervisor`：无条件 `config.Hostname = defaultHostname`（忽略调用方传入的任意 hostname），V1 无 remote-runtime 需求，杜绝 LAN 暴露
- `buildArgs`：`--hostname` 直接硬编码 `defaultHostname`（`127.0.0.1`），参数签名改为 `_ Config` 明确表明忽略 config.Hostname，防止通配符绑定
- `Config.Hostname` 注释更新为 `forced to 127.0.0.1 (loopback-only; V1 has no remote runtime)`

新增两个**非循环**回归测试（`auth_test.go`）：

- `TestSupervisorForcesLoopback`：`NewSupervisor(Config{Hostname: "0.0.0.0"})` → 断言 `config.Hostname == "127.0.0.1"`，且 `buildArgs` 产物不含 `0.0.0.0`
- `TestBuildArgsCannotExposeRuntime`：直接向 `buildArgs` 传入 `Config{Hostname: "0.0.0.0"}` → 断言产物不含 `0.0.0.0` 且必含 `127.0.0.1`（不再让测试自己传 `127.0.0.1` 来自证）

### Round 2 验证

| Gate | Result |
|------|--------|
| `go test ./... -count=1` | PASS |
| `go test -race ./internal/supervisor/... -count=1` | PASS |
| `go vet ./...` | clean |
| `go build ./...` | clean |
| `GOOS=windows GOARCH=amd64 go build ./cmd/codea ./cmd/parity-runner` | PASS |
| `GOOS=darwin GOARCH=amd64 go build ./cmd/codea ./cmd/parity-runner` | PASS |
| `check-runtime-boundary.sh` | PASS |
| `check-execution-state.sh` | valid |

## Windows x64 real lifecycle smoke — accepted residual risk

Windows x64 real lifecycle smoke（真机进程树清理验证）**未独立执行**。真实 Windows 进程树清理行为（Job Object 整树强杀 + 0 orphan）延期到 Windows 集成环境 / 发行验收阶段验证。该风险经人工接受，**不阻塞 Task 6**。

这是 accepted residual risk / 未独立验证，**不是验证通过**。Windows 侧当前已通过 x64 cross-build 验证编译正确性；真机行为在 Windows 集成 / 发行验收阶段补齐。

## Test Summary

| Package | Tests |
|---------|-------|
| internal/runtime | status.go（5 状态常量） |
| internal/supervisor | 41 darwin tests + 2 windows tests（状态机 + auth + readiness + 进程控制 + Healthy/CAS + 空格路径 + loopback 硬锁） |
| internal/supervisor/fakeopencode | 测试专用 fake binary（main 包） |
| tests/contract | 1（`TestSupervisorAdapterContract`，真实 OpenCode） |

## Files Changed

| File | Action |
|------|--------|
| `tui/internal/runtime/status.go` | Create |
| `tui/internal/supervisor/supervisor.go` | Create |
| `tui/internal/supervisor/process_unix.go` | Create |
| `tui/internal/supervisor/process_windows.go` | Create |
| `tui/internal/supervisor/supervisor_test.go` | Create |
| `tui/internal/supervisor/auth_test.go` | Create |
| `tui/internal/supervisor/readiness_test.go` | Create |
| `tui/internal/supervisor/process_unix_test.go` | Create |
| `tui/internal/supervisor/process_windows_test.go` | Create（Round 1） |
| `tui/internal/supervisor/paths_test.go` | Create（Round 1，test-only，空格路径） |
| `tui/internal/supervisor/supervisor.go` | Modify（Round 2 — loopback 硬锁） |
| `tui/internal/supervisor/auth_test.go` | Modify（Round 2 — 非循环 loopback 测试） |
| `tui/internal/supervisor/fake_runtime_test.go` | Create |
| `tui/internal/supervisor/fakeopencode/main.go` | Create |
| `tui/tests/contract/supervisor_adapter_contract_test.go` | Create |

## 提交记录

| Commit | Step |
|--------|------|
| `dba4e28` | Step 1 — Supervisor domain + 状态机 |
| `959943f` | Step 2 — Basic Auth + 启动 args/env |
| `5c84c2a` | Step 3 — Port 分配 + Ready 探测 |
| `3026bf5` | Step 4 — darwin 进程组控制 |
| `9eeba53` | Step 5 — Windows 进程组控制（cross-build） |
| `bf6fd3e` | Step 6 — Supervisor↔Adapter 集成契约（Final Implementation Commit） |
| `0b86714` | Round 1 — Windows Job Object 整树强杀 + Healthy/Crashed CAS（新 Final Implementation Commit） |
| `de4e9db` | Round 1 — 空格路径跨平台测试（test-only，不改实现） |
| `cbe692a` | Round 2 — localhost-only 硬锁（新 Final Implementation Commit） |
