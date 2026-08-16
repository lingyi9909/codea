# Task 11 Report — strict/compatible + Enterprise 模式隔离

## Overview

Checkpoint: `a77cd96448894b515eb9ec5530659cd922b8b839`

在 Task 10 的 Skill Manager 之上增加双轨 Skill 策略（strict/compatible），保证 Enterprise Agent 只加载 Approved+Enabled 的 Codea Skill（**真实隔离**，作用于实际 Runtime 加载，非 UI 假过滤），同时 General Agent 保留 Project Skill 能力（User/Claude/Agents Skill 在两种模式下都隔离）。

核心边界（本 Task 不可违反）：

- **不推翻 Task 10**：`Skill`/`SkillSource`/Installed-Enabled-Loaded 独立/Source 语义/Project-User-Runtime 只读 全部保持；Task 11 只加 Mode/Policy 层。
- **Mode 来源 = 环境变量**：`CODEA_SKILL_MODE=strict|compatible`，**默认 strict**（不是 compatible）；未知非空值报错，不静默 fallback。
- **Approved 来源 = 环境变量**：`CODEA_APPROVED_SKILLS`（逗号分隔，仅 strict 使用）；未配置默认「distribution 内全部 Codea Skills approved」。
- **真实隔离**：(1) `SyncEnabled` 在 strict 下只物化 Approved+Enabled Codea Skill 到受控 `OPENCODE_CONFIG_DIR/skills/`；(2) supervisor `buildEnv` 在**两种模式**下都关闭 User/Claude/Agents Skill（恒开 `OPENCODE_DISABLE_EXTERNAL_SKILLS=1` + 重定向 XDG 隔离原生 `~/.config/opencode/skills`），strict 额外 `OPENCODE_DISABLE_PROJECT_CONFIG=1` 关闭 Project Skill。
- **requiredSkills fail-closed**：strict 下 required 的 Skill 若未 approval 或非 Codea，启动前明确失败。
- **Runtime Boundary**：`internal/skill/` 只 import `internal/runtime/` + 标准库；`internal/supervisor/` 用 `Config.CodeaSkillsOnly bool` 表达 strict，不 import `skill`。
- **Profile Contract 只预留**：定义 `general`/`enterprise` 类型并校验「enterprise 必须 strict，禁止降级 compatible」，暂不接入启动链路、不解析 distribution profile YAML。

## 语义实现

### SkillMode + 解析（`mode.go`）

`SkillMode`（`strict`/`compatible`）、`Valid()`、`ParseSkillMode(string)`（未知值报错）、`ResolveSkillMode(string)`（空值默认 strict）。

### SkillPolicy + FilterForMode + approved 解析（`policy.go`）

- `SkillPolicy{Mode, Approved}`；`DefaultPolicy` = strict + 空 approved（空 approved 语义 = 「全部 Codea approved」）。
- `Approves(name)`、`StrictAllowed(s)`（仅 `SourceCodea && approved`）、`ParseApprovedSkills(csv)`。
- `FilterForMode(skills, p)`：strict 只保留 `StrictAllowed` 的 Codea Skill；compatible 保留 `SourceCodea`/`SourceProject`/`SourceRuntime`，丢弃 `SourceUser`（与 Runtime 语义一致：User Skill 在两种模式下都不属于有效 set）。
  - **Enabled 维度刻意不进 FilterForMode**（与 mode 正交，保持 Task 10 的 Installed/Enabled/Loaded 独立）：`Enabled` 门由 `Sync` 负责（`Sync` 跳过 `!s.Enabled`），materialization 一致；TUI 管理视图因此仍能看到 disabled 的 Codea Skill 并可重新启用。

### Validator（`validator.go`）

`ValidateSkill(s)` 只补 Discover 未显式强制的基本合法性（name 非空 + source 可识别），文件级 breakage 仍由 Discover 报 `StageDiscover`。Task 12 才做 shell/DLP/secret/audit 扫描。

### Profile Contract（`profile.go`）

`Profile{Name, SkillMode, ApprovedSkills}`、`ProfileGeneral`（compatible）、`ProfileEnterprise`（strict）、`Validate()`（enterprise 降级 compatible 报错）。未接线、未解析 YAML。

### requiredSkills strict 联动（`require.go`）

`ValidateRequiredSkills(skills, reqs, p)`：顺序 installed → approved（strict 下 `!StrictAllowed` →「not allowed」）→ enabled → loaded。保留 `ValidateRequirements`（Task 10 兼容，语义不变）。

### Manager/Sync 接入 Mode（`sync.go` / `manager.go`）

`SyncEnabled(roots, store, targetDir, p)`、`NewManager(..., p)` 贯穿 `SkillPolicy`；`List` 先 `FilterForMode` 再 `ValidateSkill` 再 `reconcileLoaded`；`SetEnabled` 复用 `SyncEnabled`。

### Supervisor 环境隔离（`supervisor.go`）

`Config.CodeaSkillsOnly bool`。`buildEnv` 在**两种模式**下都恒开 `OPENCODE_DISABLE_EXTERNAL_SKILLS=1` 并重定向 `XDG_CONFIG_HOME`/`XDG_DATA_HOME`/`XDG_CACHE_HOME`/`XDG_STATE_HOME` 到 `config.ConfigDir/xdg/…`（隔离原生 `~/.config/opencode/skills`，同时关闭 `.claude`/`.agents` 外部 Skill）；strict 额外追加 `OPENCODE_DISABLE_PROJECT_CONFIG=1` 关闭 Project Skill。HOME 不重定向（`~/.claude` 已被恒开的 `OPENCODE_DISABLE_CLAUDE_CODE=1` 关闭）。

### 组合根接线（`cmd/codea/main.go`）

`run()` 解析 `CODEA_SKILL_MODE`/`CODEA_APPROVED_SKILLS` → `SkillPolicy`，冷启动 `SyncEnabled`（在 runtime 启动前）→ `bootstrapRuntime(cfgDir, mode)` → `supervisorConfig(cfgDir, mode)`（`CodeaSkillsOnly = mode==strict`）→ `NewManager(..., policy)`。

## 真实 OpenCode Smoke（1.18.11）

| Profile | `/skill` 实际返回 | 结论 |
|---------|------------------|------|
| Smoke A（compatible） | `['codea-skill', 'customize-opencode', 'project-skill']` | Codea + Project + Runtime 可加载；User/Claude/Agents 隔离 |
| Smoke B（strict） | `['codea-approved', 'customize-opencode']` | 仅 approved Codea 可加载；project/user/claude/agents/unapproved 全部隔离 |

> `customize-opencode` 是 OpenCode 原生内置 Skill，strict 只隔离 Codea/Project/User，不（也无法）移除原生内置，符合「OpenCode 原生能力不退化」原则。

## Full Gate Verification

| Gate | Result |
|------|--------|
| `GOTOOLCHAIN=local go test ./... -count=1` | PASS（22 packages，无失败） |
| `GOTOOLCHAIN=local go test -race ./... -count=1` | PASS（22 packages，无竞态） |
| `GOTOOLCHAIN=local go vet ./...` | clean |
| `GOTOOLCHAIN=local go build ./...` | clean |
| `GOOS=windows GOARCH=amd64 GOTOOLCHAIN=local go build ./cmd/codea ./cmd/parity-runner` | PASS |
| `GOOS=darwin GOARCH=amd64 GOTOOLCHAIN=local go build ./cmd/codea ./cmd/parity-runner` | PASS |
| `./scripts/check-runtime-boundary.sh` | PASS（no vendor DTO leakage） |
| `OPENCODE_BIN=docs/spike-artifacts/opencode ./scripts/run-real-parity-smoke.sh` | PASS（17/17 gates，failedChecks=0，version=1.18.11） |
| `OPENCODE_BIN=docs/spike-artifacts/opencode ./scripts/run-skill-native-smoke.sh` | PASS（/skill 返回 codea-smoke + native-user-skill + customize-opencode） |
| `OPENCODE_BIN=docs/spike-artifacts/opencode ./scripts/run-skill-mode-smoke.sh` | PASS（compatible + strict 两 profile） |
| `./scripts/check-execution-state.sh` | valid |
| `tests/execution-state/state_validator_test.sh` | valid |

> 原始输出留档：`docs/task-reports/task-11-gate.log`（12 项 gate 全 exit 0，离线缓存模式 `GOPROXY=off GOSUMDB=off` 规避内网 proxy.golang.org 60s 超时；module cache 已就绪，结果一致）。parity evidence 已随本次重跑刷新（17/17，timestamp 2026-08-16T06:38:07Z）。

## Task 11 Gate Checklist

- [x] `SkillMode` strict/compatible 解析正确；未知值报错；空值默认 strict
- [x] `SkillPolicy` + `FilterForMode` strict 只保留 approved Codea；compatible 保留 Codea+Project+Runtime、丢弃 User
- [x] strict 物化只落盘 approved+enabled Codea（`SyncEnabled` + `Sync` 跳过 disabled）
- [x] supervisor 两种模式都关闭 User/Claude/Agents Skill（恒开 EXTERNAL_SKILLS + XDG 重定向），strict 额外关闭 Project Skill（真实隔离，Smoke A/B 证明）
- [x] compatible 只额外开放 Project（Smoke A 同时验证 allow + deny：codea/project/customize 存在，user/claude/agents 不存在）
- [x] `Validator` 只做基本合法性；不越界引入 shell/DLP/secret/audit（属 Task 12）
- [x] Profile `general`/`enterprise` 预留；enterprise 禁止降级 compatible；未接线
- [x] requiredSkills strict fail-closed（not approved / 非 Codea → 明确失败）
- [x] 不推翻 Task 10（Installed/Enabled/Loaded 独立、Project/User/Runtime 只读、TUI 可重新启用 disabled Codea Skill）
- [x] Runtime Boundary 不破坏（`skill` 不 import vendor DTO；`supervisor` 用 bool）
- [x] 全量 Gate 通过；Windows/darwin 交叉编译不退化；parity 17/17 不退化

## Files Changed

| File | Action |
|------|--------|
| `tui/internal/skill/mode.go` | Create（SkillMode + 解析） |
| `tui/internal/skill/policy.go` | Create（SkillPolicy + FilterForMode + approved 解析） |
| `tui/internal/skill/validator.go` | Create（基本合法性校验） |
| `tui/internal/skill/profile.go` | Create（保留 Profile Contract） |
| `tui/internal/skill/require.go` | Modify（requiredSkills 增加 approval 维度） |
| `tui/internal/skill/sync.go` | Modify（SyncEnabled 增加 policy） |
| `tui/internal/skill/manager.go` | Modify（Manager.policy / NewManager / List / SetEnabled） |
| `tui/internal/skill/mode_test.go`、`policy_test.go`、`validator_test.go`、`profile_test.go`、`require_test.go`、`manager_mode_test.go` | Create（测试） |
| `tui/internal/skill/coldstart_test.go`、`manager_test.go`、`readonly_test.go`、`manager_integration_test.go` | Modify（迁移 `SyncEnabled`/`NewManager` 调用点补 `DefaultPolicy`） |
| `tui/internal/supervisor/supervisor.go` | Modify（`Config.CodeaSkillsOnly` + strict XDG 隔离） |
| `tui/internal/supervisor/supervisor_test.go` | Modify（`TestBuildEnvIsolation` 断言两种模式恒开 EXTERNAL_SKILLS + XDG 重定向，strict 额外 PROJECT_CONFIG） |
| `tui/cmd/codea/main.go` | Modify（mode/policy 解析 + 冷启动 Sync + 贯穿组合根） |
| `tui/cmd/codea/main_test.go` | Modify（bootstrapRuntime 调用 + `TestSupervisorConfigMapsStrictToIsolation`） |
| `scripts/run-skill-mode-smoke.sh` | Create（真实 OpenCode compatible/strict smoke） |

## 与计划偏差

1. **`FilterForMode` 不门控 `Enabled`（计划 Task 11.2 初稿含 `!s.Enabled`）**：计划初稿 `FilterForMode` 同时 `!s.Enabled`，接线后 strict 下 `List` 会丢掉 disabled Codea Skill，使 TUI 管理视图无法重新启用（推翻 Task 10 的 Installed/Enabled/Loaded 独立与 re-enable UX）。修正为 `FilterForMode` 只门控 source+approval；`Enabled` 门保留在 `Sync`（materialization），运行时不退化。
2. **strict 隔离必须重定向 XDG（计划 Task 11.7/11.9 遗漏，Smoke B 首次运行失败）**：计划假设 `OPENCODE_DISABLE_EXTERNAL_SKILLS=1` 即可关闭原生 `~/.config/opencode/skills`。实测（Smoke B）证明该标志**不**关闭原生 User Skill 目录，只有重定向 `XDG_CONFIG_HOME` 才能隔离（与 S5/S6 spike、task-01 结论一致）。修正：supervisor `buildEnv` 与 smoke 在 strict 下追加四个 XDG 目录重定向；HOME 不重定向（`~/.claude` 已被 `OPENCODE_DISABLE_CLAUDE_CODE=1` 关闭）。
3. **计划 Task 11.6 brief 的 `sync_test.go` 追加项与 Step 1 代码块不一致**：两个 strict 物化测试实际落在 `manager_mode_test.go`，`sync_test.go` 未改（行为一致，仅位置差异）。
4. **Compatible 模式误放开 User Skill（人工验收「C. 不通过」后修正）**：初版把 XDG 重定向 + `OPENCODE_DISABLE_EXTERNAL_SKILLS=1` 只挂在 strict 下，导致 compatible 恢复扫描原生 `~/.config/opencode/skills`，放开了 User Skill——与开工前拍板的语义（Compatible 只额外开放 Project，User/Claude/Agents 仍隔离）相悖。修正为：两种模式恒开 XDG 重定向 + `OPENCODE_DISABLE_EXTERNAL_SKILLS=1`（隔离 User/Claude/Agents），strict 额外 `OPENCODE_DISABLE_PROJECT_CONFIG=1`；`CompatibleAllowed` 同步丢弃 `SourceUser`；Smoke A 改为同时验证 allow + deny（codea/project/customize 存在，user/claude/agents 不存在）。

## 已知边界（如实记录）

- `~/.claude/skills` 在两种模式下都保持禁用——由 Task 1 离线锁 `OPENCODE_DISABLE_CLAUDE_CODE=1` 关闭（规格 §3 列表中的 `~/.claude/skills` 被 Task 1 既有默认覆盖，用户要求「保持 Task 1 S6 已验收行为」）。
- `customize-opencode` 是 OpenCode 原生内置 Skill，strict 只隔离 Codea/Project/User，不（也无法）移除原生内置，符合「OpenCode 原生能力不退化」原则。
- `OPENCODE_URL` dev-override 下，`mode` 会贯穿 SyncEnabled/NewManager，但外部 runtime 无 CodeaSkillsOnly 隔离（无 supervisor 构建），为 dev/test 行为。
- `reconcileLoaded` 的 `Loaded` 按 name 匹配（Task 10 既有边界）；strict 下若 env 标志未真正隐藏 project/user skill，泄漏的 project skill 会被误标 `SourceRuntime`——Smoke B 证明 strict 下 env 标志确实隐藏它们。

## Gate 结论

- verification：pass
- Task Gate：pass
- 进入 `awaiting_acceptance`，等待人工验收；验收前不标记 completed，不启动 Task 12。
