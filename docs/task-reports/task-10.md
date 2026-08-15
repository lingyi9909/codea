# Task 10 Report — Skill/Plugin Manager

## Overview

Checkpoint: `421d763107d2a6aacd697cdef2b22e6f32039f3f`

建立 Codea 对 Skill 的统一能力管理，打通「发现 → 安装状态 → 启用/禁用 → Runtime 加载 → 来源识别 → Agent 依赖校验」链路。只做能力管理，TUI 仅最小可用。

核心边界（本 Task 不可违反）：

- **统一状态模型**：`Skill{Name, Description, Source, Installed, Enabled, Loaded, LoadError}`。无 `Matched` 模糊状态；`Installed`（本地可发现）/`Enabled`（配置允许）/`Loaded`（Runtime 实际加载）三者独立，`Loaded` 绝不从 `Enabled` 推导。
- **来源识别**：`SourceCodea`（Codea 发行）/`SourceProject`（项目 `.opencode/skills`、`.agents/skills`）/`SourceUser`（`~/.config/opencode/skills`、`~/.claude/skills`）/`SourceRuntime`（OpenCode 内置）。来源由扫描目录决定，不由 OpenCode DTO 透出。
- **Enable/Disable 真正生效**：Codea 持有 JSON 覆盖持久化，`Sync` 生成受控 Runtime 配置目录 `OPENCODE_CONFIG_DIR/skills/`，只落盘 enabled 的 Codea Skill——disabled 不落盘，OpenCode 不再加载，非 UI 假禁用。
- **Loaded 来自真实结果**：`runtime.SkillRuntime.ListSkills` 查询真实 OpenCode `/skill` 端点；`Loaded = name 出现在结果`。
- **requiredSkills 强失败**：启动 Agent 前逐项校验 Installed/Enabled/Loaded，任一缺失返回明确错误（含缺失 Skill 名与原因），不允许带病启动。
- **Plugin 边界**：本 Task 只管理/展示 Skill，不重新实现 OpenCode Plugin 系统（Plugin 能力仍由 OpenCode 原生承担，parity smoke 的 plugin gate 不退化）。

## Step 1 — Skill Domain Contract

`tui/internal/runtime/skills.go`（Create）+ `tui/internal/skill/domain.go`（Create）：

- `runtime.SkillRuntime`：`ListSkills(ctx, directory string) ([]LoadedSkill, error)`，与 `AgentRuntime` 分离，核心 session/event/approval 契约保持最小。`directory` 用于解析 project-scoped skills。
- `runtime.LoadedSkill{Name, Description, Location}`：Location 为 runtime 特定来源提示，Skill 域映射为自己的 `SkillSource`，不当作 vendor DTO。
- `skill.Skill`（统一模型，含内部 `dir` 字段供 sync）、`SkillSource`（四值 + `Valid()`）、`SkillRequirement`、`SkillError`（Stage 区分 discover/load/require，实现 `error`）。

## Step 2 — Discovery

`tui/internal/skill/discover.go`（Create）：扫描多个根目录下 `<name>/SKILL.md`，`name` 取 frontmatter（缺省回退目录名），`description` 取 frontmatter。单个 Skill 读取/解析失败只标记损坏并记 `SkillError`，不拖垮整个列表、不 panic、不静默吞错；结果按 name→source 确定性排序。

## Step 3 — Enable/Disable + Runtime 同步

`tui/internal/skill/store.go`（Create，JSON 持久化）+ `tui/internal/skill/sync.go`（Create）：

- `Store` 接口 + `FileStore`（`map[string]bool`，缺省空 map）。
- `Sync(skills, targetDir)` 只复制 `SourceCodea && Enabled` 的 Skill，先 `RemoveAll` 再重建，移除已不再 enabled 的 stale skill；`copyDir/copyFile` 保留可执行位。project/user skill 由 runtime 直接发现，不在此处管理。

## Step 4 — Loaded 来自真实结果

`tui/internal/skill/manager.go`（Create）+ `tui/internal/opencode/skill.go`（Create）：

- `Manager{roots, store, targetDir, projectDir, runtime}`，`List(ctx)` 串联 Discover → applyOverrides → reconcileLoaded，per-skill 失败记 `Snapshot.Errors` 不中断整体。
- `reconcileLoaded` 调用 `runtime.ListSkills(ctx, projectDir)`：`Loaded = 出现在结果`；`Installed && Enabled && !Loaded` 记 `LoadError`；runtime 内置且非文件系统的 Skill 以 `SourceRuntime + Installed + Loaded` 追加。
- `OpenCodeAdapter.ListSkills` 实现 `runtime.SkillRuntime`（`var _ runtime.SkillRuntime = (*OpenCodeAdapter)(nil)` 编译期断言），`HTTPClient.ListSkills` 走 `GET /skill?directory=<url-encoded>`，映射 `OpenCodeAppSkillsResponse` 原始数组到 `runtime.LoadedSkill`。

## Step 5 — inherit + requiredSkills 确定性合并

`tui/internal/skill/merge.go`（Create）+ `tui/internal/skill/require.go`（Create）：

- `AgentSkillConfig{Inherit, Skills, RequiredSkills}`；`MergeSkillNames(defaults, cfg)` 按「默认/全局 + Agent 显式 + requiredSkills」合并，按 Name 去重并排序（与输入顺序无关），`Inherit=false` 丢弃默认集。
- `ValidateRequirements(skills, reqs)` 逐项校验 Installed/Enabled/Loaded，任一缺失用 `errors.Join` 聚合为明确错误（含缺失 Skill 名与 Stage=require 原因）。

## Step 6 — 最小 TUI

`tui/internal/components/skill.go`（Create，纯展示表）+ `tui/internal/app/skill.go`（Create，页逻辑）+ `tui/internal/app/` 多处 Modify + `tui/cmd/codea/main.go`（Modify，装配）：

- `components.SkillModel`：cursor/可见性/展示，`View()` 每行 `name [source] installed=✓/✗ enabled=✓/✗ loaded=✓/✗`，附 `LoadError`；`Selected()`/`MoveUp`/`MoveDown` clamp。
- `app` 新增 `PageSkills`，`skillManager` 接口（`List`/`SetEnabled`，供测试替换 fake），`SetSkillManager` 注入；`ctrl+k` 切换页、`Enter` toggle、`r` refresh、`Esc` 关闭；`ListSkillsCmd`/`SetSkillEnabledCmd` 异步不阻塞事件循环，成功后重新 `List` 反映持久化 + runtime 真实状态。
- `cmd/codea` 的 `buildSkillManager` 从环境解析 roots/store/targetDir/projectDir，`adapter` 兼作 `runtime.SkillRuntime`。
- fixture：`distribution/skills/code-review/SKILL.md`、`distribution/skills/unit-test/SKILL.md`。

## Step 7 — Integration/Parity + Gate

- `tui/internal/skill/manager_integration_test.go`（Create）：全链路（真实 FileStore + 真实 targetDir + fake runtime）——discover → reconcileLoaded → SetEnabled(disable) 持久化 + sync 移除 → runtime 卸载后 re-list 反映 override + 真实 loaded。
- `tui/tests/parity/real_parity_smoke_test.go`（Modify）：新增 `skillManager` gate——真实 `adapter.ListSkills(ctx, "")` 查询 `/skill` 端点，断言返回非空且含 `smoke-skill`，证明 Loaded 来自真实 Runtime。17/17 gate 全绿。

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
| `./scripts/check-execution-state.sh` | valid |
| `tests/execution-state/state_validator_test.sh` | valid |

## Task 10 Gate Checklist

- [x] Skill Domain 不依赖 OpenCode DTO（`internal/skill/` 仅依赖 `internal/runtime/` + 标准库）
- [x] Discovery 正常；Source 正确识别；Installed 正确
- [x] Enabled 正确；Loaded 来自真实 Runtime；Enable/Disable 真正生效；Disabled 不再被 Runtime 加载
- [x] inherit=true/false 正确；requiredSkills 正确检查；缺失时明确失败
- [x] 单个坏 Skill 不拖垮全部；加载失败不静默
- [x] OpenCode Native Skill/Plugin 不退化（parity smoke skill + plugin gate 全绿）
- [x] TUI 至少可查看和管理 Skill（查看/Enable/Disable/Refresh/错误）
- [x] Runtime Boundary Gate 通过；Windows 不退化；Offline 不退化；全量测试通过

## Files Changed

| File | Action |
|------|--------|
| `tui/internal/runtime/skills.go` | Create（SkillRuntime + LoadedSkill，ListSkills 带 directory） |
| `tui/internal/skill/domain.go` | Create（Skill/SkillSource/SkillRequirement/SkillError） |
| `tui/internal/skill/discover.go` | Create（扫描 + 来源识别 + 坏 Skill 隔离） |
| `tui/internal/skill/store.go` | Create（JSON 覆盖持久化） |
| `tui/internal/skill/sync.go` | Create（受控 Runtime 配置生成） |
| `tui/internal/skill/manager.go` | Create（List/SetEnabled/reconcileLoaded） |
| `tui/internal/skill/merge.go` | Create（inherit + requiredSkills 确定性合并） |
| `tui/internal/skill/require.go` | Create（requiredSkills 强失败校验） |
| `tui/internal/skill/*_test.go` | Create（8 个测试文件，含全链路集成测试） |
| `tui/internal/opencode/skill.go` | Create（HTTPClient.ListSkills + Adapter 实现 SkillRuntime） |
| `tui/internal/opencode/http_client_test.go` | Modify（ListSkills directory 编码测试） |
| `tui/internal/components/skill.go` | Create（SkillModel 纯展示表） |
| `tui/internal/components/skill_test.go` | Create |
| `tui/internal/app/skill.go` | Create（skills 页逻辑 + 命令 + skillManager 接口） |
| `tui/internal/app/skill_test.go` | Create |
| `tui/internal/app/page.go` | Modify（PageSkills） |
| `tui/internal/app/keymap.go` | Modify（ctrl+k skills / r refresh） |
| `tui/internal/app/model.go` | Modify（skills/skillPanel/skillNotice + SetSkillManager） |
| `tui/internal/app/update.go` | Modify（skills 页消息处理 + 键路由） |
| `tui/internal/app/view.go` | Modify（skills 页渲染 + footer hint） |
| `tui/cmd/codea/main.go` | Modify（buildSkillManager 装配） |
| `tui/tests/parity/real_parity_smoke_test.go` | Modify（新增 skillManager gate） |
| `tui/tests/parity/evidence/runtime-evidence.json` | Modify（fresh evidence，17/17 全绿） |
| `distribution/skills/code-review/SKILL.md` | Create（fixture） |
| `distribution/skills/unit-test/SKILL.md` | Create（fixture） |

## 与计划偏差

1. `runtime.SkillRuntime.ListSkills` 从计划初稿的 `ListSkills(ctx)` 增加 `directory string` 参数——OpenCode `/skill` 端点需要 `directory` query param 才能正确解析 project-scoped skills，缺失会导致项目 Skill 的 Loaded 状态误判。
2. 计划 §3.5 的 `SkillRuntime` 接口注释未含 directory；实现时补齐，避免 project skill 的 loaded 判定失真。
3. `AgentSkillConfig`/`MergeSkillNames`/`ValidateRequirements` 作为独立 `merge.go`/`require.go` 落地，与 `domain.go` 分离，`SkillRequirement` 保留在 domain（plan §3.6 保持一致）。

## Gate 结论

- verification：pass
- Task Gate：pass
- 进入 `awaiting_acceptance`，等待人工验收；验收前不标记 completed，不启动 Task 11。
