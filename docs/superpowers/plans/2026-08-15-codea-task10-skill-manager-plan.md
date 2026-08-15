# Codea V1 Task 10 — Skill/Plugin Manager 实施计划

日期：2026-08-15

状态：本计划采纳人工锁定的 Task 10 边界（Skill 统一状态模型 / 来源识别 / Discovery / Enable-Disable / 真实 Loaded / inherit+requiredSkills / Runtime 同步 / 最小 TUI），取代主实施计划 `2026-07-30-codea-v1-plan.md` 中较薄的 Task 10 一节（其 `Matched` 状态与四级 `release/profile/user/project` 合并模型不再沿用）。

## 1. Goal

建立 Codea 对 Skill 的统一能力管理，打通「发现 → 安装状态 → 启用/禁用 → Runtime 加载 → 来源识别 → Agent 依赖校验」链路。只做能力管理，TUI 仅最小可用。

## 2. 架构边界（不可违反）

```text
TUI → Application → Codea Skill Domain → Skill Manager / Runtime Adapter → OpenCode
```

- `tui/internal/skill/`（Codea Skill Domain）只依赖 `tui/internal/runtime/`（Runtime Contract）与标准库，**不得 import `tui/internal/opencode/`**。
- OpenCodeAdapter 在 `tui/internal/opencode/` 内实现新增的 `runtime.SkillRuntime` 契约；composition root（`tui/cmd/codea/`）负责装配。
- 上层只认识 `Skill` / `SkillSource` / `SkillRequirement` / `SkillError` 等 Domain Model，不接触 OpenCode `/skill` DTO。
- 继续通过 `scripts/check-runtime-boundary.sh` 与 `tui/tests/architecture/vendor_boundary_test.go`。

## 3. 关键设计决策

### 3.1 统一状态模型

```go
type Skill struct {
    Name        string
    Description string
    Source      SkillSource
    Installed   bool
    Enabled     bool
    Loaded      bool
    LoadError   string // Installed&&Enabled&&!Loaded 时的诊断，空串表示无错误
}
```

无 `Matched` 状态。语义：Installed=本地可发现；Enabled=配置允许使用；Loaded=Runtime 实际已加载；三者独立，`Loaded` 绝不从 `Enabled` 推导。

### 3.2 来源识别（`SkillSource`）

```go
const (
    SourceCodea   SkillSource = "codea"   // Built-in/Codea 发行目录 distribution/skills/*
    SourceProject SkillSource = "project" // 项目 .opencode/skills、.agents/skills
    SourceUser    SkillSource = "user"    // ~/.config/opencode/skills、~/.claude/skills
    SourceRuntime SkillSource = "runtime" // OpenCode 内置（/skill location=<built-in>）
)
```

来源由扫描目录决定，不由 OpenCode DTO 字段透出。

### 3.3 Discovery（扫描 + 坏 Skill 隔离）

扫描多个根目录下的 `<name>/SKILL.md`；`name` 取自目录名，`description` 取自 SKILL.md frontmatter（无 frontmatter 时回退为空描述，仍视为 Installed）。单个 Skill 读取/解析失败只把该 Skill 标记为损坏并记录 `SkillError`，不拖垮整个列表、不 panic、不静默吞错。

### 3.4 Enable/Disable + Runtime 同步

- Codea 持有 `Store`（JSON 文件）记录 per-skill 显式 enabled/disabled 覆盖；未覆盖的 Codea 发行 Skill 默认 enabled。
- `Manager.Sync` 生成受控 Runtime 配置目录 `OPENCODE_CONFIG_DIR/skills/`，**只**包含 enabled 的 Codea Skill；disabled 不落盘。
- 由此保证「Disabled 不是 UI 假禁用」：enabled=false → 受控配置目录不含该 Skill → OpenCode 不加载。

### 3.5 Loaded 来自真实结果

`runtime.SkillRuntime` 契约：

```go
type LoadedSkill struct { Name, Description, Location string }
type SkillRuntime interface { ListSkills(ctx context.Context) ([]LoadedSkill, error) }
```

`Loaded = 该 name 出现在 ListSkills 结果`。`Installed&&Enabled&&!Loaded` → 置 `LoadError`。OpenCodeAdapter 实现该契约（`GET /skill`）。

### 3.6 inherit + requiredSkills 确定性合并

```go
type AgentSkillConfig struct {
    Inherit         bool              // 继承默认/全局 Skill 集
    Skills          []string          // Agent 显式声明
    RequiredSkills  []SkillRequirement // 强依赖
}
```

合并顺序固定（与启动顺序无关）：`Global/默认 Skill + Agent 显式 Skill + requiredSkills`，按 Name 去重。`inherit=false` 时只取 Agent 显式 + requiredSkills。结果确定、可排序。

### 3.7 requiredSkills 强失败

启动 Agent 前逐项检查 requiredSkills 的 Installed/Enabled/Loaded；任一缺失 → 返回明确错误（含缺失 Skill 名与原因），不得带病启动、不得静默降级。

## 4. 实施步骤

- [ ] **Step 1 — Skill Domain Contract**：`tui/internal/runtime/skills.go`（SkillRuntime/LoadedSkill）+ `tui/internal/skill/domain.go`（Skill/SkillSource/SkillRequirement/SkillError）。先写测试。
- [ ] **Step 2 — Discovery**：`tui/internal/skill/discover.go` 扫描 + 来源识别 + Installed + 坏 Skill 隔离。
- [ ] **Step 3 — Enable/Disable + 同步**：`tui/internal/skill/store.go`（JSON 持久化）+ `sync.go`（受控 Runtime 配置生成）。
- [ ] **Step 4 — Loaded 状态**：Manager 装配 `SkillRuntime`，Loaded 来自真实查询，enabled-not-loaded 记 LoadError。
- [ ] **Step 5 — inherit + requiredSkills**：`tui/internal/skill/merge.go` + `require.go` 确定性合并与强失败校验。
- [ ] **Step 6 — 最小 TUI**：`tui/internal/components/skill.go` 表 + `tui/internal/app/` 增加 skills 页（查看/Enable/Disable/Refresh/错误）。
- [ ] **Step 7 — Integration/Parity**：边界门禁、Fake SkillRuntime 全链路、真实 OpenCode parity smoke。

## 5. 文件清单

| File | Action |
|------|--------|
| `tui/internal/runtime/skills.go` | Create |
| `tui/internal/skill/domain.go` | Create |
| `tui/internal/skill/discover.go` | Create |
| `tui/internal/skill/store.go` | Create |
| `tui/internal/skill/sync.go` | Create |
| `tui/internal/skill/manager.go` | Create |
| `tui/internal/skill/merge.go` | Create |
| `tui/internal/skill/require.go` | Create |
| `tui/internal/opencode/skill.go` | Create（DTO + HTTP ListSkills + adapter 实现 SkillRuntime） |
| `tui/internal/components/skill.go` | Create |
| `tui/internal/app/*` | Modify（skills 页 + 命令） |
| `tui/cmd/codea/main.go` | Modify（装配 Skill Manager） |
| `distribution/skills/*` | Create（fixture：builtin/enterprise 各至少一个 SKILL.md） |

## 6. Task 10 Gate

- [ ] Skill Domain 不依赖 OpenCode DTO
- [ ] Discovery 正常；Source 正确识别；Installed 正确
- [ ] Enabled 正确；Loaded 来自真实 Runtime；Enable/Disable 真正生效；Disabled 不再被 Runtime 加载
- [ ] inherit=true/false 正确；requiredSkills 正确检查；缺失时明确失败
- [ ] 单个坏 Skill 不拖垮全部；加载失败不静默
- [ ] OpenCode Native Skill/Plugin 不退化
- [ ] TUI 至少可查看和管理 Skill
- [ ] Runtime Boundary Gate 通过；Windows 不退化；Offline 不退化；全量测试通过
