# Task 11: strict/compatible + Enterprise 模式隔离 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 Task 10 的 Skill Manager 之上增加双轨 Skill 策略（strict/compatible），保证 Enterprise Agent 只加载 Approved+Enabled 的 Codea Skill（真实隔离），同时 General Agent 不丢失 OpenCode 原生 Project/User Skill 能力。

**Architecture:** 在 `tui/internal/skill/` 增加 Mode/Policy/Validator/Profile 四个领域文件；`SkillPolicy`（Mode + Approved set）驱动两处真实行为——(1) `SyncEnabled` 只物化 strict 下 Approved+Enabled 的 Codea Skill，(2) `supervisor.buildEnv` 在 strict 下追加 `OPENCODE_DISABLE_EXTERNAL_SKILLS=1` + `OPENCODE_DISABLE_PROJECT_CONFIG=1` 关闭 Project/User 扫描。requiredSkills 校验增加 approval 维度（strict 缺失即 fail-closed）。真实隔离由两个 OpenCode 1.18.11 Smoke（A compatible / B strict）证明。

**Tech Stack:** Go 1.26.5（`tui/` 模块）、Bubble Tea、OpenCode v1.18.11（`opencode serve`）、Bash + curl + python3（Smoke）。

**Spec:** `docs/superpowers/plans/2026-07-30-codea-v1-plan.md`（Task 11 章节）＋ 2026-08-16 人工提供的《Codea V1｜Task 11 开发任务》17 节详细规格。本计划逐条落实该规格并锁定以下人为决定：

- **Mode 来源 = 环境变量**：`CODEA_SKILL_MODE=strict|compatible`，**默认 strict**（不是 compatible），保持 Task 1 S6 已验收的 V1 默认行为。
- **Approved 来源 = 环境变量**：`CODEA_APPROVED_SKILLS`（逗号分隔，仅 strict 使用）；未配置时默认「distribution 内全部 Codea Skills approved」。
- **Profile Contract 本 Task 只定义并预留** `general`/`enterprise` 类型，**不接入启动链路，不解析 distribution profile YAML**；Enterprise Profile 后续接线时必须强制 strict，禁止降级 compatible。

## Global Constraints

- 依赖方向：`tui/internal/skill` 只 import `tui/internal/runtime` + 标准库；禁止 import `opencode` vendor DTO。`tui/internal/supervisor` 不 import `skill`（用 `Config.CodeaSkillsOnly bool` 表达 strict 隔离）。
- 不推翻 Task 10：`Skill`/`SkillSource`/Installed-Enabled-Loaded 独立/Source 语义/Project-User-Runtime 只读 全部保持；Task 11 只加 Mode/Policy 层。
- 未知 `CODEA_SKILL_MODE` 值必须报错，**不得 silently fallback 到 compatible**；空值 = 默认 strict。
- strict 只允许 `Approved && Enabled` 的 Codea Skill；Project/User/Runtime（含 `customize-opencode` 原生内置）**不作为 strict 有效 requiredSkill**。
- **已知边界（记录于 task-11.md，非本 Task 修复）**：`~/.claude/skills` 在两种模式下都保持禁用——它由 Task 1 离线锁 `OPENCODE_DISABLE_CLAUDE_CODE=1` 关闭（规格 §3 列表中的 `~/.claude/skills` 被 Task 1 既有默认覆盖，用户要求「保持 Task 1 S6 已验收行为」）。`customize-opencode` 是 OpenCode 原生内置 Skill，strict 只隔离 Codea/Project/User，不（也无法）移除原生内置，符合「OpenCode 原生能力不退化」原则。
- 全量门禁命令（每 Task 完成都要绿）：
  - `GOTOOLCHAIN=local go test ./... -count=1`
  - `GOTOOLCHAIN=local go test -race ./... -count=1`
  - `GOTOOLCHAIN=local go vet ./...`
  - `GOTOOLCHAIN=local go build ./...`
  - `GOOS=windows GOARCH=amd64 GOTOOLCHAIN=local go build ./cmd/codea ./cmd/parity-runner`
  - `GOOS=darwin GOARCH=amd64 GOTOOLCHAIN=local go build ./cmd/codea ./cmd/parity-runner`
  - `./scripts/check-runtime-boundary.sh`
  - `./scripts/check-execution-state.sh`
  - `tests/execution-state/state_validator_test.sh`

---

### Task 11.1: SkillMode 类型 + 解析（mode.go）

**Files:**
- Create: `tui/internal/skill/mode.go`
- Test: `tui/internal/skill/mode_test.go`

**Interfaces:**
- Consumes: 无（纯领域类型，仅标准库 `fmt`）。
- Produces: `SkillMode`、`SkillModeStrict`、`SkillModeCompatible`、`(SkillMode).Valid()`、`ParseSkillMode(string) (SkillMode, error)`、`ResolveSkillMode(string) (SkillMode, error)`。

- [ ] **Step 1: 写失败测试**

`tui/internal/skill/mode_test.go`：

```go
package skill

import (
	"strings"
	"testing"
)

func TestParseSkillModeKnown(t *testing.T) {
	for _, in := range []string{"strict", "compatible"} {
		m, err := ParseSkillMode(in)
		if err != nil {
			t.Fatalf("ParseSkillMode(%q): %v", in, err)
		}
		if string(m) != in {
			t.Fatalf("ParseSkillMode(%q) = %q", in, m)
		}
	}
}

func TestParseSkillModeUnknownErrors(t *testing.T) {
	if _, err := ParseSkillMode("bogus"); err == nil {
		t.Fatal("unknown mode must error, not silently compatible")
	} else if !strings.Contains(err.Error(), "bogus") {
		t.Fatalf("error should name the bad value: %v", err)
	}
}

func TestResolveSkillModeDefaultsStrict(t *testing.T) {
	m, err := ResolveSkillMode("")
	if err != nil {
		t.Fatalf("empty should default, not error: %v", err)
	}
	if m != SkillModeStrict {
		t.Fatalf("default = %q, want strict", m)
	}
}

func TestSkillModeValid(t *testing.T) {
	if !SkillModeStrict.Valid() || !SkillModeCompatible.Valid() {
		t.Fatal("strict and compatible must be valid")
	}
	if SkillMode("").Valid() || SkillMode("nope").Valid() {
		t.Fatal("empty/unknown must be invalid")
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd tui && GOTOOLCHAIN=local go test ./internal/skill/ -run 'TestParseSkillMode|TestResolveSkillMode|TestSkillModeValid' -count=1`
Expected: FAIL（`undefined: ParseSkillMode` 等）。

- [ ] **Step 3: 实现**

`tui/internal/skill/mode.go`：

```go
package skill

import "fmt"

// SkillMode selects how Codea composes the runtime's effective skill set.
type SkillMode string

const (
	// SkillModeStrict is the V1 default: only Approved + Enabled Codea skills
	// enter the runtime; project and user skills are isolated out.
	SkillModeStrict SkillMode = "strict"
	// SkillModeCompatible preserves OpenCode native skill discovery: Codea +
	// Project + User (+ runtime built-in) skills are all available.
	SkillModeCompatible SkillMode = "compatible"
)

// Valid reports whether m is a known SkillMode.
func (m SkillMode) Valid() bool {
	return m == SkillModeStrict || m == SkillModeCompatible
}

// ParseSkillMode maps a raw string to a SkillMode. An unrecognised non-empty
// value is an error; it is never silently coerced to compatible.
func ParseSkillMode(s string) (SkillMode, error) {
	m := SkillMode(s)
	if m.Valid() {
		return m, nil
	}
	return "", fmt.Errorf("unknown skill mode %q (want %q or %q)", s, SkillModeStrict, SkillModeCompatible)
}

// ResolveSkillMode returns the configured mode, defaulting to strict when the
// raw value is empty (the V1 default). A non-empty unrecognised value errors.
func ResolveSkillMode(s string) (SkillMode, error) {
	if s == "" {
		return SkillModeStrict, nil
	}
	return ParseSkillMode(s)
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd tui && GOTOOLCHAIN=local go test ./internal/skill/ -run 'TestParseSkillMode|TestResolveSkillMode|TestSkillModeValid' -count=1`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add tui/internal/skill/mode.go tui/internal/skill/mode_test.go
git commit -m "feat: task 11 — SkillMode type and strict-by-default parsing"
```

---

### Task 11.2: SkillPolicy + FilterForMode + approved 解析（policy.go）

**Files:**
- Create: `tui/internal/skill/policy.go`
- Test: `tui/internal/skill/policy_test.go`

**Interfaces:**
- Consumes: `SkillMode`/`SkillModeStrict`/`SkillModeCompatible`（Task 11.1）、`Skill`/`SourceCodea`（domain.go，已存在）。
- Produces: `SkillPolicy{Mode, Approved}`、`DefaultPolicy`、`(SkillPolicy).Approves(string) bool`、`(SkillPolicy).StrictAllowed(Skill) bool`、`ParseApprovedSkills(string) map[string]bool`、`FilterForMode([]Skill, SkillPolicy) []Skill`。

- [ ] **Step 1: 写失败测试**

`tui/internal/skill/policy_test.go`：

```go
package skill

import "testing"

func codea(name string, enabled bool) Skill {
	return Skill{Name: name, Source: SourceCodea, Installed: true, Enabled: enabled}
}

func TestFilterForModeCompatibleKeepsAll(t *testing.T) {
	skills := []Skill{
		codea("code-review", true),
		{Name: "proj", Source: SourceProject, Installed: true, Enabled: true},
		{Name: "user", Source: SourceUser, Installed: true, Enabled: true},
	}
	got := FilterForMode(skills, SkillPolicy{Mode: SkillModeCompatible})
	if len(got) != 3 {
		t.Fatalf("compatible must keep all, got %d", len(got))
	}
}

func TestFilterForModeStrictKeepsApprovedEnabledCodea(t *testing.T) {
	skills := []Skill{
		codea("code-review", true),
		codea("experimental", true),
		codea("unit-test", false),
		{Name: "proj", Source: SourceProject, Installed: true, Enabled: true},
		{Name: "user", Source: SourceUser, Installed: true, Enabled: true},
	}
	got := FilterForMode(skills, SkillPolicy{
		Mode:     SkillModeStrict,
		Approved: map[string]bool{"code-review": true},
	})
	names := map[string]bool{}
	for _, s := range got {
		names[s.Name] = true
	}
	if len(got) != 1 || !names["code-review"] {
		t.Fatalf("strict should keep only approved+enabled Codea: %+v", got)
	}
}

func TestFilterForModeStrictEmptyApprovedMeansAll(t *testing.T) {
	skills := []Skill{codea("code-review", true), codea("unit-test", true)}
	got := FilterForMode(skills, SkillPolicy{Mode: SkillModeStrict})
	if len(got) != 2 {
		t.Fatalf("empty approved set approves all Codea, got %d", len(got))
	}
}

func TestParseApprovedSkills(t *testing.T) {
	got := ParseApprovedSkills(" code-review , unit-test , ")
	if len(got) != 2 || !got["code-review"] || !got["unit-test"] {
		t.Fatalf("parse comma list: %v", got)
	}
	if len(ParseApprovedSkills("")) != 0 || len(ParseApprovedSkills("  ")) != 0 {
		t.Fatal("empty/whitespace must yield empty set")
	}
}

func TestStrictAllowed(t *testing.T) {
	p := SkillPolicy{Mode: SkillModeStrict, Approved: map[string]bool{"code-review": true}}
	if !p.StrictAllowed(codea("code-review", true)) {
		t.Fatal("approved Codea must be strict-allowed")
	}
	if p.StrictAllowed(codea("experimental", true)) {
		t.Fatal("unapproved Codea must not be strict-allowed")
	}
	if p.StrictAllowed(Skill{Name: "code-review", Source: SourceProject}) {
		t.Fatal("project skill must never be strict-allowed")
	}
	if p.StrictAllowed(Skill{Name: "customize-opencode", Source: SourceRuntime}) {
		t.Fatal("runtime built-in must never be strict-allowed")
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd tui && GOTOOLCHAIN=local go test ./internal/skill/ -run 'TestFilterForMode|TestParseApprovedSkills|TestStrictAllowed' -count=1`
Expected: FAIL。

- [ ] **Step 3: 实现**

`tui/internal/skill/policy.go`：

```go
package skill

import "strings"

// SkillPolicy bundles a mode with the set of approved skill names. Approved
// only ever gates SourceCodea skills: project/user skills are isolated by the
// runtime env flags (strict) or left alone (compatible), and runtime built-ins
// are never gated.
type SkillPolicy struct {
	Mode     SkillMode
	Approved map[string]bool // empty means "all Codea skills approved"
}

// DefaultPolicy is the V1 default: strict mode, all Codea skills approved.
var DefaultPolicy = SkillPolicy{Mode: SkillModeStrict}

// Approves reports whether name is approved. An empty approved set approves
// every name (the distribution default).
func (p SkillPolicy) Approves(name string) bool {
	if len(p.Approved) == 0 {
		return true
	}
	return p.Approved[name]
}

// StrictAllowed reports whether a skill may be used under strict mode: it must
// be an approved Codea skill. Project/user/runtime skills are never allowed.
func (p SkillPolicy) StrictAllowed(s Skill) bool {
	return s.Source == SourceCodea && p.Approves(s.Name)
}

// ParseApprovedSkills parses a comma-separated skill list into an approval set.
// An empty or whitespace-only value returns an empty map, which Approves treats
// as "approve all".
func ParseApprovedSkills(s string) map[string]bool {
	if strings.TrimSpace(s) == "" {
		return map[string]bool{}
	}
	out := map[string]bool{}
	for _, part := range strings.Split(s, ",") {
		if name := strings.TrimSpace(part); name != "" {
			out[name] = true
		}
	}
	return out
}

// FilterForMode returns the skills visible under the policy. Compatible keeps
// everything. Strict keeps only enabled Codea skills that are approved; project,
// user and runtime skills are dropped from the Codea-managed view (the runtime
// independently hides project/user via env flags, and re-adds built-ins during
// loaded reconciliation).
func FilterForMode(skills []Skill, p SkillPolicy) []Skill {
	if p.Mode != SkillModeStrict {
		return skills
	}
	out := make([]Skill, 0, len(skills))
	for _, s := range skills {
		if !s.Enabled || !p.StrictAllowed(s) {
			continue
		}
		out = append(out, s)
	}
	return out
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd tui && GOTOOLCHAIN=local go test ./internal/skill/ -run 'TestFilterForMode|TestParseApprovedSkills|TestStrictAllowed' -count=1`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add tui/internal/skill/policy.go tui/internal/skill/policy_test.go
git commit -m "feat: task 11 — SkillPolicy, FilterForMode and approved parsing"
```

---

### Task 11.3: Validator（validator.go）

**Files:**
- Create: `tui/internal/skill/validator.go`
- Test: `tui/internal/skill/validator_test.go`

**Interfaces:**
- Consumes: `Skill`、`SkillSource.Valid()`、`SkillError`/`StageDiscover`（domain.go，已存在）。
- Produces: `ValidateSkill(Skill) error`。

说明：Task 10 的 `Discover` 已负责「SKILL.md 存在 / 读失败隔离 / frontmatter 解析 / 目录名回退 / 越界防护（`entry.Name()` 无分隔符）」。本 Task 的 `ValidateSkill` 只补 Discover 未显式强制的一项基本合法性（name 非空 + source 可识别），并在 Manager.List 管线中调用（Task 11.4 接线）。文件级 breakage 仍由 Discover 报 `StageDiscover` 错误。

- [ ] **Step 1: 写失败测试**

`tui/internal/skill/validator_test.go`：

```go
package skill

import (
	"strings"
	"testing"
)

func TestValidateSkillOK(t *testing.T) {
	if err := ValidateSkill(Skill{Name: "code-review", Source: SourceCodea}); err != nil {
		t.Fatalf("valid skill rejected: %v", err)
	}
}

func TestValidateSkillEmptyName(t *testing.T) {
	err := ValidateSkill(Skill{Name: "  ", Source: SourceProject})
	if err == nil {
		t.Fatal("empty name must error")
	}
	var se SkillError
	if !asSkillError(err, &se) || se.Stage != StageDiscover {
		t.Fatalf("expected a discover-stage SkillError, got %v", err)
	}
}

func TestValidateSkillUnknownSource(t *testing.T) {
	err := ValidateSkill(Skill{Name: "x", Source: SkillSource("mars")})
	if err == nil || !strings.Contains(err.Error(), "source") {
		t.Fatalf("unknown source must error clearly: %v", err)
	}
}

func asSkillError(err error, out *SkillError) bool {
	se, ok := err.(SkillError)
	if ok {
		*out = se
	}
	return ok
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd tui && GOTOOLCHAIN=local go test ./internal/skill/ -run TestValidateSkill -count=1`
Expected: FAIL。

- [ ] **Step 3: 实现**

`tui/internal/skill/validator.go`：

```go
package skill

import "strings"

// ValidateSkill reports basic legality problems for a discovered skill. It
// checks only what discovery does not already guarantee (name validity and
// source identity); file-level breakage (missing/unreadable SKILL.md) is
// reported by Discover. Task 12 owns shell/DLP/secret/audit scanning.
func ValidateSkill(s Skill) error {
	if strings.TrimSpace(s.Name) == "" {
		return SkillError{Name: s.Name, Source: s.Source, Stage: StageDiscover, Message: "skill name is empty"}
	}
	if !s.Source.Valid() {
		return SkillError{Name: s.Name, Source: s.Source, Stage: StageDiscover, Message: "unknown skill source"}
	}
	return nil
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd tui && GOTOOLCHAIN=local go test ./internal/skill/ -run TestValidateSkill -count=1`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add tui/internal/skill/validator.go tui/internal/skill/validator_test.go
git commit -m "feat: task 11 — basic skill legality validator"
```

---

### Task 11.4: Profile Contract（profile.go）

**Files:**
- Create: `tui/internal/skill/profile.go`
- Test: `tui/internal/skill/profile_test.go`

**Interfaces:**
- Consumes: `SkillMode`/`ParseSkillMode`（Task 11.1）。
- Produces: `Profile{Name, SkillMode, ApprovedSkills}`、`ProfileGeneral`、`ProfileEnterprise`、`(Profile).Validate() error`。

- [ ] **Step 1: 写失败测试**

`tui/internal/skill/profile_test.go`：

```go
package skill

import "testing"

func TestReservedProfiles(t *testing.T) {
	if ProfileGeneral.SkillMode != SkillModeCompatible {
		t.Fatalf("general must be compatible, got %q", ProfileGeneral.SkillMode)
	}
	if ProfileEnterprise.SkillMode != SkillModeStrict {
		t.Fatalf("enterprise must be strict, got %q", ProfileEnterprise.SkillMode)
	}
}

func TestProfileValidateEnterpriseDowngradeRejected(t *testing.T) {
	p := Profile{Name: "enterprise", SkillMode: SkillModeCompatible}
	if err := p.Validate(); err == nil {
		t.Fatal("enterprise must not be downgradable to compatible")
	}
}

func TestProfileValidateOK(t *testing.T) {
	if err := ProfileEnterprise.Validate(); err != nil {
		t.Fatalf("enterprise profile invalid: %v", err)
	}
	if err := ProfileGeneral.Validate(); err != nil {
		t.Fatalf("general profile invalid: %v", err)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd tui && GOTOOLCHAIN=local go test ./internal/skill/ -run TestProfile -count=1`
Expected: FAIL。

- [ ] **Step 3: 实现**

`tui/internal/skill/profile.go`：

```go
package skill

import "fmt"

// Profile is a reserved enterprise profile contract. Task 11 only defines and
// reserves the type and the two V1 profiles; it is NOT wired into startup and
// no distribution profile YAML is parsed yet. Tasks 14-16 will consume it.
type Profile struct {
	Name           string
	SkillMode      SkillMode
	ApprovedSkills []string
}

// Reserved V1 profiles. Enterprise always uses strict mode (never compatible).
var (
	ProfileGeneral    = Profile{Name: "general", SkillMode: SkillModeCompatible}
	ProfileEnterprise = Profile{Name: "enterprise", SkillMode: SkillModeStrict}
)

// Validate enforces the profile contract. An enterprise profile must use strict
// mode and cannot be downgraded to compatible.
func (p Profile) Validate() error {
	if _, err := ParseSkillMode(string(p.SkillMode)); err != nil {
		return fmt.Errorf("profile %q: %w", p.Name, err)
	}
	if p.Name == ProfileEnterprise.Name && p.SkillMode != SkillModeStrict {
		return fmt.Errorf("enterprise profile must use strict skill mode, got %q", p.SkillMode)
	}
	return nil
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd tui && GOTOOLCHAIN=local go test ./internal/skill/ -run TestProfile -count=1`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add tui/internal/skill/profile.go tui/internal/skill/profile_test.go
git commit -m "feat: task 11 — reserved general/enterprise profile contract"
```

---

### Task 11.5: requiredSkills + strict 联动（require.go）

**Files:**
- Modify: `tui/internal/skill/require.go`
- Test: `tui/internal/skill/require_test.go`（追加，保留现有 4 个测试不动）

**Interfaces:**
- Consumes: `SkillPolicy`/`StrictAllowed`（Task 11.2）、`Skill`/`SkillRequirement`/`SkillError`/`StageRequire`（domain.go）。
- Produces: `ValidateRequiredSkills([]Skill, []SkillRequirement, SkillPolicy) error`；保留 `ValidateRequirements([]Skill, []SkillRequirement) error`（Task 10 兼容，语义不变）。

- [ ] **Step 1: 写失败测试**

在 `tui/internal/skill/require_test.go` 末尾追加：

```go
func TestValidateRequiredSkillsStrictNotApproved(t *testing.T) {
	skills := []Skill{
		{Name: "enterprise-review", Source: SourceCodea, Installed: true, Enabled: true, Loaded: true},
	}
	p := SkillPolicy{Mode: SkillModeStrict, Approved: map[string]bool{"other": true}}
	err := ValidateRequiredSkills(skills, []SkillRequirement{{Name: "enterprise-review"}}, p)
	if err == nil {
		t.Fatal("required skill not approved must fail in strict mode")
	}
	if !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("error should say not-allowed: %v", err)
	}
}

func TestValidateRequiredSkillsStrictProjectSkillRejected(t *testing.T) {
	skills := []Skill{
		{Name: "proj", Source: SourceProject, Installed: true, Enabled: true, Loaded: true},
	}
	p := SkillPolicy{Mode: SkillModeStrict} // empty approved = all Codea approved, but proj is not Codea
	err := ValidateRequiredSkills(skills, []SkillRequirement{{Name: "proj"}}, p)
	if err == nil || !strings.Contains(err.Error(), "not allowed") {
		t.Fatalf("project skill must fail strict requiredSkills: %v", err)
	}
}

func TestValidateRequiredSkillsStrictPass(t *testing.T) {
	skills := []Skill{
		{Name: "code-review", Source: SourceCodea, Installed: true, Enabled: true, Loaded: true},
	}
	p := SkillPolicy{Mode: SkillModeStrict, Approved: map[string]bool{"code-review": true}}
	if err := ValidateRequiredSkills(skills, []SkillRequirement{{Name: "code-review"}}, p); err != nil {
		t.Fatalf("approved+enabled+loaded must pass: %v", err)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd tui && GOTOOLCHAIN=local go test ./internal/skill/ -run TestValidateRequiredSkills -count=1`
Expected: FAIL（`undefined: ValidateRequiredSkills`）。

- [ ] **Step 3: 实现（重构 require.go）**

`tui/internal/skill/require.go` 整体替换为：

```go
package skill

import (
	"errors"
	"fmt"
)

// ValidateRequirements checks that every required skill is installed, enabled
// and loaded under compatible semantics (no approval policy). Retained for Task
// 10 compatibility.
func ValidateRequirements(skills []Skill, reqs []SkillRequirement) error {
	return validateRequiredSkills(skills, reqs, SkillPolicy{Mode: SkillModeCompatible})
}

// ValidateRequiredSkills checks required skills against the reconciled skill set
// under the active policy. In strict mode a required skill must also be an
// approved Codea skill, otherwise it fails closed with a distinct "not allowed"
// error rather than a misleading "not installed". Order is fixed:
// installed -> approved -> enabled -> loaded.
func ValidateRequiredSkills(skills []Skill, reqs []SkillRequirement, p SkillPolicy) error {
	return validateRequiredSkills(skills, reqs, p)
}

func validateRequiredSkills(skills []Skill, reqs []SkillRequirement, p SkillPolicy) error {
	byName := make(map[string]Skill, len(skills))
	for _, s := range skills {
		byName[s.Name] = s
	}

	var errs []error
	for _, r := range reqs {
		s, ok := byName[r.Name]
		switch {
		case !ok || !s.Installed:
			errs = append(errs, SkillError{Name: r.Name, Stage: StageRequire, Message: "skill not installed"})
		case p.Mode == SkillModeStrict && !p.StrictAllowed(s):
			errs = append(errs, SkillError{Name: r.Name, Source: s.Source, Stage: StageRequire, Message: "not allowed by strict policy"})
		case !s.Enabled:
			errs = append(errs, SkillError{Name: r.Name, Source: s.Source, Stage: StageRequire, Message: "skill is disabled"})
		case !s.Loaded:
			msg := s.LoadError
			if msg == "" {
				msg = "skill is not loaded by runtime"
			}
			errs = append(errs, SkillError{Name: r.Name, Source: s.Source, Stage: StageRequire, Message: msg})
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("required skills unavailable: %w", errors.Join(errs...))
	}
	return nil
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd tui && GOTOOLCHAIN=local go test ./internal/skill/ -run 'TestValidateRequirements|TestValidateRequiredSkills' -count=1`
Expected: PASS（原 4 个 `TestValidateRequirements*` 仍绿）。

- [ ] **Step 5: Commit**

```bash
git add tui/internal/skill/require.go tui/internal/skill/require_test.go
git commit -m "feat: task 11 — requiredSkills strict fail-closed approval check"
```

---

### Task 11.6: Manager + Sync 接入 Mode（sync.go / manager.go）

**Files:**
- Modify: `tui/internal/skill/sync.go`（`SyncEnabled` 增加 `policy` 参数）
- Modify: `tui/internal/skill/manager.go`（`Manager.policy`、`NewManager` 增加参数、`List` 过滤、`SetEnabled` 传 policy）
- Test: `tui/internal/skill/manager_mode_test.go`（Create）、`tui/internal/skill/sync_test.go`（追加 strict 物化测试）
- Modify（迁移现有调用点）：`tui/internal/skill/coldstart_test.go`、`manager_test.go`、`readonly_test.go`、`manager_integration_test.go`

**Interfaces:**
- Consumes: `SkillPolicy`/`DefaultPolicy`/`FilterForMode`（Task 11.2）、`ValidateSkill`（Task 11.3）。
- Produces: `SyncEnabled([]Root, Store, string, SkillPolicy) error`、`NewManager([]Root, Store, string, string, runtime.SkillRuntime, SkillPolicy) *Manager`。`Sync([]Skill, string) error` 签名不变。

- [ ] **Step 1: 写失败测试**

`tui/internal/skill/manager_mode_test.go`：

```go
package skill

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"codea/tui/internal/runtime"
)

// TestManagerListStrictFilters verifies the manager view drops non-approved and
// non-Codea skills in strict mode, while reconcileLoaded re-adds runtime built-ins.
func TestManagerListStrictFilters(t *testing.T) {
	codeaRoot := t.TempDir()
	writeSkillDir(t, codeaRoot, "code-review")
	writeSkillDir(t, codeaRoot, "experimental")
	projectRoot := t.TempDir()
	writeSkillDir(t, projectRoot, "proj")

	m := NewManager(
		[]Root{
			{Dir: codeaRoot, Source: SourceCodea},
			{Dir: projectRoot, Source: SourceProject},
		},
		&memStore{},
		filepath.Join(t.TempDir(), "target"),
		filepath.Join(t.TempDir(), "project"),
		&fakeSkillRuntime{loaded: []runtime.LoadedSkill{
			{Name: "code-review"},
			{Name: "customize-opencode", Description: "builtin"},
		}},
		SkillPolicy{Mode: SkillModeStrict, Approved: map[string]bool{"code-review": true}},
	)

	snap, err := m.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, s := range snap.Skills {
		names[s.Name] = true
	}
	if !names["code-review"] {
		t.Fatalf("approved Codea must appear: %+v", names)
	}
	if names["experimental"] || names["proj"] {
		t.Fatalf("non-approved/project must be filtered in strict: %+v", names)
	}
	if !names["customize-opencode"] {
		t.Fatalf("runtime built-in must be re-added: %+v", names)
	}
}

// TestSyncEnabledStrictExcludesUnapproved verifies strict materialization copies
// only approved+enabled Codea skills.
func TestSyncEnabledStrictExcludesUnapproved(t *testing.T) {
	root := t.TempDir()
	writeSkillDir(t, root, "code-review")
	writeSkillDir(t, root, "experimental")

	target := filepath.Join(t.TempDir(), "skills")
	p := SkillPolicy{Mode: SkillModeStrict, Approved: map[string]bool{"code-review": true}}

	if err := SyncEnabled([]Root{{Dir: root, Source: SourceCodea}}, &memStore{}, target, p); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(target, "code-review", "SKILL.md")); err != nil {
		t.Fatalf("approved skill must be synced: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "experimental")); !os.IsNotExist(err) {
		t.Fatalf("unapproved skill must not be synced: %v", err)
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd tui && GOTOOLCHAIN=local go test ./internal/skill/ -run 'TestManagerListStrictFilters|TestSyncEnabledStrictExcludesUnapproved' -count=1`
Expected: FAIL（编译错误：`SyncEnabled`/`NewManager` 参数不匹配）。

- [ ] **Step 3: 实现**

`tui/internal/skill/sync.go` 修改 `SyncEnabled`（只改这一处）：

```go
// SyncEnabled discovers skills, applies the persisted overrides and the mode
// policy, and materializes the resulting enabled Codea skills into targetDir.
// It is the cold-start equivalent of Manager.SetEnabled's sync path and
// deliberately does not query the runtime, so it can run before the runtime
// process has started.
func SyncEnabled(roots []Root, store Store, targetDir string, p SkillPolicy) error {
	skills, _ := Discover(roots)
	overrides, err := store.Load()
	if err != nil {
		return fmt.Errorf("load skill overrides: %w", err)
	}
	skills = applyOverrides(skills, overrides)
	skills = FilterForMode(skills, p)
	return Sync(skills, targetDir)
}
```

`tui/internal/skill/manager.go` 修改（`Manager` 结构体、`NewManager`、`List`、`SetEnabled`）：

```go
type Manager struct {
	roots      []Root
	store      Store
	targetDir  string
	projectDir string
	runtime    runtime.SkillRuntime
	policy     SkillPolicy
}

func NewManager(roots []Root, store Store, targetDir string, projectDir string, rt runtime.SkillRuntime, p SkillPolicy) *Manager {
	return &Manager{roots: roots, store: store, targetDir: targetDir, projectDir: projectDir, runtime: rt, policy: p}
}

func (m *Manager) List(ctx context.Context) (Snapshot, error) {
	skills, errs := Discover(m.roots)

	overrides, err := m.store.Load()
	if err != nil {
		return Snapshot{}, fmt.Errorf("load skill overrides: %w", err)
	}
	skills = applyOverrides(skills, overrides)

	skills = FilterForMode(skills, m.policy)

	for _, s := range skills {
		if err := ValidateSkill(s); err != nil {
			errs = append(errs, err.(SkillError))
		}
	}

	skills, err = m.reconcileLoaded(ctx, skills)
	if err != nil {
		return Snapshot{}, err
	}

	sortSkills(skills)
	return Snapshot{Skills: skills, Errors: errs}, nil
}
```

`SetEnabled` 末尾的 sync 调用改为：

```go
	return SyncEnabled(m.roots, m.store, m.targetDir, m.policy)
```

- [ ] **Step 4: 迁移现有测试调用点**

以下 4 个文件里的 `SyncEnabled`/`NewManager` 调用补 `DefaultPolicy` 参数：

- `tui/internal/skill/coldstart_test.go`：两处 `SyncEnabled(..., store, target)` → `SyncEnabled(..., store, target, DefaultPolicy)`。
- `tui/internal/skill/manager_test.go`：`newTestManager` 里 `NewManager(...)` → 追加 `DefaultPolicy`。
- `tui/internal/skill/readonly_test.go`：`NewManager(...)` → 追加 `DefaultPolicy`。
- `tui/internal/skill/manager_integration_test.go`：`NewManager(...)` → 追加 `DefaultPolicy`。

（`DefaultPolicy` = strict + 空 approved = 全部 Codea approved，行为与迁移前一致。）

- [ ] **Step 5: 运行测试确认通过**

Run: `cd tui && GOTOOLCHAIN=local go test ./internal/skill/ -count=1`
Expected: PASS（含新增 2 个 mode 测试与迁移后的全部既有测试）。

- [ ] **Step 6: Commit**

```bash
git add tui/internal/skill/sync.go tui/internal/skill/manager.go \
  tui/internal/skill/manager_mode_test.go tui/internal/skill/sync_test.go \
  tui/internal/skill/coldstart_test.go tui/internal/skill/manager_test.go \
  tui/internal/skill/readonly_test.go tui/internal/skill/manager_integration_test.go
git commit -m "feat: task 11 — thread SkillPolicy through Manager and sync"
```

---

### Task 11.7: Supervisor 环境隔离（supervisor.go）

**Files:**
- Modify: `tui/internal/supervisor/supervisor.go`（`Config` 加字段、`buildEnv` 条件追加 disable 标志）
- Test: `tui/internal/supervisor/supervisor_test.go`（追加 `TestBuildEnvIsolation`）

**Interfaces:**
- Consumes: 无新依赖（`Config` 已有；supervisor 不 import `skill`）。
- Produces: `supervisor.Config.CodeaSkillsOnly bool`。

- [ ] **Step 1: 写失败测试**

在 `tui/internal/supervisor/supervisor_test.go` 末尾追加：

```go
func hasEnv(env []string, k string) bool {
	for _, e := range env {
		if e == k {
			return true
		}
	}
	return false
}

func TestBuildEnvIsolation(t *testing.T) {
	base := buildEnv(Config{ConfigDir: "/c", CodeaSkillsOnly: false}, "u", "p")
	if hasEnv(base, "OPENCODE_DISABLE_EXTERNAL_SKILLS=1") || hasEnv(base, "OPENCODE_DISABLE_PROJECT_CONFIG=1") {
		t.Fatal("compatible mode must not disable external/project skills")
	}
	if !hasEnv(base, "OPENCODE_DISABLE_CLAUDE_CODE=1") {
		t.Fatal("Task 1 offline lock must remain")
	}

	strict := buildEnv(Config{ConfigDir: "/c", CodeaSkillsOnly: true}, "u", "p")
	if !hasEnv(strict, "OPENCODE_DISABLE_EXTERNAL_SKILLS=1") || !hasEnv(strict, "OPENCODE_DISABLE_PROJECT_CONFIG=1") {
		t.Fatal("strict mode must disable external + project skills")
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

Run: `cd tui && GOTOOLCHAIN=local go test ./internal/supervisor/ -run TestBuildEnvIsolation -count=1`
Expected: FAIL（`Config` 无 `CodeaSkillsOnly` 字段）。

- [ ] **Step 3: 实现**

`tui/internal/supervisor/supervisor.go`：

`Config` 结构体增加字段：

```go
type Config struct {
	OpenCodeBin    string
	Hostname       string // forced to 127.0.0.1 (loopback-only; V1 has no remote runtime)
	Port           int    // 0 selects a free local port
	ConfigDir      string
	ProjectRoot    string
	StartupTimeout time.Duration
	StopTimeout    time.Duration
	// CodeaSkillsOnly disables the runtime's discovery of external (user) and
	// project skills so only Codea-controlled skills load. Set for strict mode.
	CodeaSkillsOnly bool
}
```

`buildEnv` 改为：

```go
func buildEnv(config Config, username, password string) []string {
	env := append(os.Environ(),
		"OPENCODE_CONFIG_DIR="+config.ConfigDir,
		"OPENCODE_SERVER_USERNAME="+username,
		"OPENCODE_SERVER_PASSWORD="+password,
		// Offline env locked by Task 1: prevent models fetch, autoupdate, web
		// UI, LSP download and default plugin install during startup.
		"OPENCODE_DISABLE_CLAUDE_CODE=1",
		"OPENCODE_DISABLE_MODELS_FETCH=1",
		"OPENCODE_DISABLE_AUTOUPDATE=1",
		"OPENCODE_DISABLE_EMBEDDED_WEB_UI=1",
		"OPENCODE_DISABLE_LSP_DOWNLOAD=1",
		"OPENCODE_DISABLE_DEFAULT_PLUGINS=1",
	)
	if config.CodeaSkillsOnly {
		env = append(env,
			"OPENCODE_DISABLE_EXTERNAL_SKILLS=1",
			"OPENCODE_DISABLE_PROJECT_CONFIG=1",
		)
	}
	return env
}
```

- [ ] **Step 4: 运行测试确认通过**

Run: `cd tui && GOTOOLCHAIN=local go test ./internal/supervisor/ -count=1`
Expected: PASS。

- [ ] **Step 5: Commit**

```bash
git add tui/internal/supervisor/supervisor.go tui/internal/supervisor/supervisor_test.go
git commit -m "feat: task 11 — supervisor strict skill isolation env flags"
```

---

### Task 11.8: 组合根接线（cmd/codea/main.go）

**Files:**
- Modify: `tui/cmd/codea/main.go`
- Test: `tui/cmd/codea/main_test.go`（更新 `bootstrapRuntime` 调用；追加 mode 解析测试）

**Interfaces:**
- Consumes: `skill.ResolveSkillMode`/`ParseApprovedSkills`/`SkillPolicy`（Task 11.1/11.2）、`supervisor.Config.CodeaSkillsOnly`（Task 11.7）。
- Produces: `run()` 内解析 `CODEA_SKILL_MODE`/`CODEA_APPROVED_SKILLS` 并贯穿 SyncEnabled/bootstrapRuntime/NewManager；`bootstrapRuntime(cfgDir string, mode skill.SkillMode)`、`supervisorConfig(cfgDir string, mode skill.SkillMode)`。

- [ ] **Step 1: 实现 main.go**

`tui/cmd/codea/main.go` 的 `run`、`bootstrapRuntime`、`supervisorConfig` 改为：

```go
func run() error {
	cfgDir := codeaConfigDir()

	mode, err := skill.ResolveSkillMode(os.Getenv("CODEA_SKILL_MODE"))
	if err != nil {
		return err
	}
	policy := skill.SkillPolicy{
		Mode:     mode,
		Approved: skill.ParseApprovedSkills(os.Getenv("CODEA_APPROVED_SKILLS")),
	}

	// Cold-start sync: materialize the mode-policy-approved enabled Codea skills
	// into the controlled runtime config dir BEFORE the runtime starts so they
	// are actually loaded by OpenCode on first launch.
	roots := skillRoots()
	store := skill.NewFileStore(filepath.Join(cfgDir, "codea", "skills.json"))
	targetDir := filepath.Join(cfgDir, "skills")
	if err := skill.SyncEnabled(roots, store, targetDir, policy); err != nil {
		return fmt.Errorf("sync skills: %w", err)
	}

	adapter, cleanup, err := bootstrapRuntime(cfgDir, mode)
	if err != nil {
		return err
	}
	defer cleanup()

	projectDir, _ := os.Getwd()
	model := app.NewModel(adapter)
	model.SetSkillManager(skill.NewManager(roots, store, targetDir, projectDir, adapter, policy))
	p := tea.NewProgram(model, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		return fmt.Errorf("run TUI: %w", err)
	}
	return nil
}

func bootstrapRuntime(cfgDir string, mode skill.SkillMode) (adapter *opencode.OpenCodeAdapter, cleanup func(), err error) {
	if baseURL := os.Getenv("OPENCODE_URL"); baseURL != "" {
		adapter = opencode.NewOpenCodeAdapter(
			baseURL,
			os.Getenv("OPENCODE_USERNAME"),
			os.Getenv("OPENCODE_PASSWORD"),
		)
		return adapter, func() {}, nil
	}

	sup := supervisor.NewSupervisor(supervisorConfig(cfgDir, mode))
	if err := sup.Start(context.Background()); err != nil {
		return nil, nil, fmt.Errorf("start runtime: %w", err)
	}

	adapter = opencode.NewOpenCodeAdapter(sup.BaseURL(), sup.Username(), sup.Password())
	return adapter, func() { _ = sup.Stop() }, nil
}

func supervisorConfig(cfgDir string, mode skill.SkillMode) supervisor.Config {
	bin := os.Getenv("OPENCODE_BIN")
	if bin == "" {
		bin = "opencode"
	}
	projectRoot, _ := os.Getwd()
	return supervisor.Config{
		OpenCodeBin:     bin,
		ConfigDir:       cfgDir,
		ProjectRoot:     projectRoot,
		CodeaSkillsOnly: mode == skill.SkillModeStrict,
	}
}
```

- [ ] **Step 2: 更新 main_test.go 现有调用 + 追加测试**

`tui/cmd/codea/main_test.go`：

1. `TestBootstrapRuntimeSupervisedChain` 里 `bootstrapRuntime(t.TempDir())` → `bootstrapRuntime(t.TempDir(), skill.SkillModeStrict)`。
2. `TestBootstrapRuntimeStartupFailure` 里 `bootstrapRuntime(t.TempDir())` → `bootstrapRuntime(t.TempDir(), skill.SkillModeStrict)`。
3. 末尾追加：

```go
// TestResolveModeFromEnv guards the strict-by-default entrypoint wiring.
func TestSupervisorConfigMapsStrictToIsolation(t *testing.T) {
	strict := supervisorConfig(t.TempDir(), skill.SkillModeStrict)
	if !strict.CodeaSkillsOnly {
		t.Fatal("strict mode must set CodeaSkillsOnly")
	}
	compat := supervisorConfig(t.TempDir(), skill.SkillModeCompatible)
	if compat.CodeaSkillsOnly {
		t.Fatal("compatible mode must not set CodeaSkillsOnly")
	}
}
```

- [ ] **Step 3: 运行测试确认通过**

Run: `cd tui && GOTOOLCHAIN=local go test ./cmd/codea/ ./internal/skill/ ./internal/supervisor/ -count=1`
Expected: PASS。

- [ ] **Step 4: Commit**

```bash
git add tui/cmd/codea/main.go tui/cmd/codea/main_test.go
git commit -m "feat: task 11 — resolve skill mode/approved from env and wire through composition root"
```

---

### Task 11.9: 真实 OpenCode Smoke（compatible + strict）

**Files:**
- Create: `scripts/run-skill-mode-smoke.sh`

**Interfaces:**
- Consumes: `OPENCODE_BIN`（必须指向 OpenCode v1.18.11 可执行文件）、`curl`、`python3`。
- Produces: 两个 profile 的真实 `/skill` 断言，stdout 打印 `[PASS]`，全部通过 exit 0。

- [ ] **Step 1: 写脚本**

`scripts/run-skill-mode-smoke.sh`：

```bash
#!/usr/bin/env bash
set -euo pipefail

# run-skill-mode-smoke.sh
#
# Proves Task 11's two skill modes against the real locked OpenCode v1.18.11:
#   Smoke A (compatible): Codea + Project + User skills are ALL loadable — the
#                         isolated Codea config dir must not shadow native skills.
#   Smoke B (strict):     only the approved Codea skill is loadable; project,
#                         user and unapproved-Codea skills are isolated out via
#                         OPENCODE_DISABLE_EXTERNAL_SKILLS + PROJECT_CONFIG and
#                         materialization of approved-only skills.
#
# Exits 0 only when both profiles pass.

repo_root=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
opencode_bin=${OPENCODE_BIN:-}
port=${PORT:-49552}
username=${OPENCODE_SERVER_USERNAME:-opencode}
password=${OPENCODE_SERVER_PASSWORD:-skill-mode-smoke}

run_root=$(mktemp -d "${TMPDIR:-/tmp}/codea-skill-mode.XXXXXX")
server_pid=""

cleanup() {
  if [ -n "$server_pid" ]; then
    kill "$server_pid" 2>/dev/null || true
    wait "$server_pid" 2>/dev/null || true
  fi
  rm -rf "$run_root"
}
trap cleanup EXIT INT TERM

if [ -z "$opencode_bin" ] || [ ! -x "$opencode_bin" ]; then
  echo "OPENCODE_BIN must point to an executable OpenCode v1.18.11 binary" >&2
  exit 2
fi
for cmd in python3 curl; do
  command -v "$cmd" >/dev/null 2>&1 || { echo "$cmd is required" >&2; exit 2; }
done

write_skill() { # $1=dir $2=name
  mkdir -p "$1/$2"
  cat > "$1/$2/SKILL.md" <<EOF
---
name: $2
description: smoke skill $2
---
EOF
}

start_server() { # $1=project_dir $2=config_dir $3=home $4=mode
  local project=$1 config=$2 home=$3 mode=$4
  (
    cd "$project"
    export HOME="$home"
    export OPENCODE_CONFIG_DIR="$config"
    export OPENCODE_SERVER_USERNAME="$username"
    export OPENCODE_SERVER_PASSWORD="$password"
    export OPENCODE_DISABLE_CLAUDE_CODE=1
    export OPENCODE_DISABLE_MODELS_FETCH=1
    export OPENCODE_DISABLE_AUTOUPDATE=1
    export OPENCODE_DISABLE_EMBEDDED_WEB_UI=1
    export OPENCODE_DISABLE_LSP_DOWNLOAD=1
    export OPENCODE_DISABLE_DEFAULT_PLUGINS=1
    if [ "$mode" = "strict" ]; then
      export OPENCODE_DISABLE_EXTERNAL_SKILLS=1
      export OPENCODE_DISABLE_PROJECT_CONFIG=1
    fi
    "$opencode_bin" serve --hostname 127.0.0.1 --port "$port"
  ) >"$run_root/opencode-$mode.log" 2>&1 &
  server_pid=$!

  local ready=0
  for _ in $(seq 1 150); do
    if ! kill -0 "$server_pid" 2>/dev/null; then
      wait "$server_pid" 2>/dev/null || true
      server_pid=""
      echo "OpenCode exited before healthy ($mode)" >&2
      cat "$run_root/opencode-$mode.log" >&2
      exit 1
    fi
    if curl -fsS --max-time 1 -u "$username:$password" \
      "http://127.0.0.1:$port/global/health" >"$run_root/health-$mode.json" 2>/dev/null; then
      if python3 - "$run_root/health-$mode.json" <<'PY'
import json, pathlib, sys
p = json.loads(pathlib.Path(sys.argv[1]).read_text())
raise SystemExit(0 if p.get("healthy") is True and p.get("version") == "1.18.11" else 1)
PY
      then
        ready=1
        break
      fi
    fi
    sleep 0.2
  done
  if [ "$ready" -ne 1 ]; then
    echo "OpenCode did not become healthy ($mode)" >&2
    exit 1
  fi
}

stop_server() {
  if [ -n "$server_pid" ]; then
    kill "$server_pid" 2>/dev/null || true
    wait "$server_pid" 2>/dev/null || true
    server_pid=""
  fi
}

fetch_skills() { # $1=project_dir $2=outfile
  curl -fsS --max-time 10 -u "$username:$password" --get \
    --data-urlencode "directory=$1" \
    "http://127.0.0.1:$port/skill" > "$2"
}

# --- Smoke A: compatible ------------------------------------------------------
home="$run_root/homeA"
config="$home/.codea/runtime-config"
project="$run_root/projectA"
mkdir -p "$project" "$config/skills"
write_skill "$config/skills" "codea-skill"
write_skill "$home/.config/opencode/skills" "user-skill"
write_skill "$project/.opencode/skills" "project-skill"

start_server "$project" "$config" "$home" "compatible"
resp="$run_root/skill-compatible.json"
fetch_skills "$project" "$resp"
stop_server

python3 - "$resp" <<'PY'
import json, pathlib, sys
payload = json.loads(pathlib.Path(sys.argv[1]).read_text())
names = {item["name"] for item in payload}
required = {"codea-skill", "project-skill", "user-skill"}
missing = sorted(required - names)
if missing:
    raise SystemExit(f"compatible missing {missing}; got {sorted(names)}")
print(f"[PASS] compatible: Codea+Project+User all loadable ({sorted(names)})")
PY

# --- Smoke B: strict ----------------------------------------------------------
home="$run_root/homeB"
config="$home/.codea/runtime-config"
project="$run_root/projectB"
mkdir -p "$project" "$config/skills"
write_skill "$config/skills" "codea-approved"
# codea-unapproved exists only in a "distribution" dir OpenCode does not scan,
# mirroring how Codea's strict sync never materializes it.
write_skill "$run_root/distribution" "codea-unapproved"
write_skill "$home/.config/opencode/skills" "user-skill"
write_skill "$project/.opencode/skills" "project-skill"

start_server "$project" "$config" "$home" "strict"
resp="$run_root/skill-strict.json"
fetch_skills "$project" "$resp"
stop_server

python3 - "$resp" <<'PY'
import json, pathlib, sys
payload = json.loads(pathlib.Path(sys.argv[1]).read_text())
names = {item["name"] for item in payload}
if "codea-approved" not in names:
    raise SystemExit(f"strict missing approved skill; got {sorted(names)}")
forbidden = {"project-skill", "user-skill", "codea-unapproved"}
present = sorted(forbidden & names)
if present:
    raise SystemExit(f"strict leaked non-approved skills {present}; got {sorted(names)}")
print(f"[PASS] strict: only approved Codea loadable ({sorted(names)})")
PY

echo "[PASS] skill mode smoke: compatible + strict both verified"
```

- [ ] **Step 2: 运行 Smoke**

Run:
```bash
OPENCODE_BIN=/Users/zhangzhanhui/Documents/job/codea/docs/spike-artifacts/opencode \
  ./scripts/run-skill-mode-smoke.sh
```
Expected: 输出两个 `[PASS]` 行，exit 0。（`customize-opencode` 原生内置可能出现在两个 profile 中，属预期；断言用 contains/not-contains，不要求精确相等。）

- [ ] **Step 3: Commit**

```bash
git add scripts/run-skill-mode-smoke.sh
git commit -m "test: task 11 — real OpenCode compatible/strict skill mode smoke"
```

---

### Task 11.10: 全量门禁 + Task 报告 + 执行状态

**Files:**
- Create: `docs/task-reports/task-11.md`
- Modify: `docs/execution-state.yaml`（Task 11 → awaiting_acceptance，checkpoint 指向最新 HEAD）

- [ ] **Step 1: 运行全量门禁**

```bash
cd tui && GOTOOLCHAIN=local go test ./... -count=1
cd tui && GOTOOLCHAIN=local go test -race ./... -count=1
cd tui && GOTOOLCHAIN=local go vet ./...
cd tui && GOTOOLCHAIN=local go build ./...
cd tui && GOOS=windows GOARCH=amd64 GOTOOLCHAIN=local go build ./cmd/codea ./cmd/parity-runner
cd tui && GOOS=darwin GOARCH=amd64 GOTOOLCHAIN=local go build ./cmd/codea ./cmd/parity-runner
./scripts/check-runtime-boundary.sh
OPENCODE_BIN=/Users/zhangzhanhui/Documents/job/codea/docs/spike-artifacts/opencode ./scripts/run-real-parity-smoke.sh
OPENCODE_BIN=/Users/zhangzhanhui/Documents/job/codea/docs/spike-artifacts/opencode ./scripts/run-skill-native-smoke.sh
OPENCODE_BIN=/Users/zhangzhanhui/Documents/job/codea/docs/spike-artifacts/opencode ./scripts/run-skill-mode-smoke.sh
./scripts/check-execution-state.sh
tests/execution-state/state_validator_test.sh
```
Expected: 全部 PASS；parity smoke 17/17 全绿（Task 10 不退化），skill-mode smoke 两 profile 全绿。

- [ ] **Step 2: 写 Task 报告**

`docs/task-reports/task-11.md` 记录：目标、文件变更、Mode/Policy/FilterForMode/Validator/Profile/requiredSkills 语义、两个真实 Smoke 输出、Gate 表（Gate A–H 逐条对照规格 §十六）、已知边界（`~/.claude/skills` 被 Task 1 离线锁保持禁用；`customize-opencode` 原生内置在 strict 下仍存在）。

- [ ] **Step 3: 更新执行状态并提交**

`docs/execution-state.yaml`：`tasks["11"]` → `{status: awaiting_acceptance, completedSteps: [1..10], verificationStatus: pass, taskGateStatus: pass, humanAccepted: false, checkpoint: <最新 HEAD>, report: docs/task-reports/task-11.md}`；`current.status: awaiting_acceptance`。

```bash
git add docs/task-reports/task-11.md docs/execution-state.yaml
git commit -m "docs: Task 11 — strict/compatible + Enterprise 模式隔离, awaiting acceptance"
```

- [ ] **Step 4: 停止，等待人工验收**

不得自行将 Task 11 标记 completed，不得开始 Task 12。

---

## Self-Review

**Spec 覆盖（对照人工规格 §一~§十七）：**
- §四/§五 Step 1-2（SkillMode + Approved Policy）→ Task 11.1 + 11.2。
- §六 Step 3（Compatible 不退化）→ Task 11.7（不追加 disable 标志）+ Smoke A。
- §七/§八 Step 4-5（Strict 真隔离）→ Task 11.6（物化过滤）+ Task 11.7（env 隔离）+ Smoke B。
- §九 Step 6（Validator）→ Task 11.3。
- §十 Step 7（Profile contract）→ Task 11.4。
- §十一 Step 8（requiredSkills 联动）→ Task 11.5。
- §十二 测试清单 1-5 → mode_test / policy_test / require_test / manager_mode_test / validator_test。
- §十三 Smoke A/B → Task 11.9。
- §十四 Runtime Boundary → `skill` 仍不 import vendor DTO；`supervisor` 用 bool 不 import `skill`。
- §十五 不越界 → 未引入 DLP/安全/Agent 等。

**Placeholder 扫描：** 无 TBD/TODO；每个代码步骤含完整代码。

**类型一致性：** `SkillPolicy`/`DefaultPolicy`/`FilterForMode`/`StrictAllowed`/`ParseApprovedSkills`/`ValidateSkill`/`ValidateRequiredSkills`/`SyncEnabled(…, SkillPolicy)`/`NewManager(…, SkillPolicy)`/`Config.CodeaSkillsOnly` 在 Task 11.2→11.8 之间签名一致；`bootstrapRuntime`/`supervisorConfig` 的 mode 参数在 main.go 与 main_test.go 一致。
