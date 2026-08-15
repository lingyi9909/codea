// Package skill defines the Codea-owned Skill domain model and manager.
//
// Dependency rule: this package depends only on the Codea runtime contract
// (codea/tui/internal/runtime) and the standard library. It must never import
// the OpenCode vendor layer.
package skill

import "fmt"

// SkillSource identifies where a skill originates.
type SkillSource string

const (
	// SourceCodea is a skill bundled in the Codea distribution.
	SourceCodea SkillSource = "codea"
	// SourceProject is a skill discovered under the project (.opencode/skills
	// or .agents/skills).
	SourceProject SkillSource = "project"
	// SourceUser is a skill discovered under the user's OpenCode or Claude
	// config directories.
	SourceUser SkillSource = "user"
	// SourceRuntime is a skill built into the runtime itself.
	SourceRuntime SkillSource = "runtime"
)

// Valid reports whether s is a known SkillSource.
func (s SkillSource) Valid() bool {
	switch s {
	case SourceCodea, SourceProject, SourceUser, SourceRuntime:
		return true
	}
	return false
}

// Skill is the Codea-owned unified skill model. Installed, Enabled and Loaded
// are independent booleans: a skill may be installed and enabled yet fail to
// load, and Loaded is never derived from Enabled.
type Skill struct {
	Name        string
	Description string
	Source      SkillSource
	Installed   bool
	Enabled     bool
	Loaded      bool
	// LoadError carries a diagnostic when a skill is installed and enabled but
	// not reported as loaded by the runtime. Empty when there is no error.
	LoadError string

	// dir is the on-disk source directory for filesystem-discovered skills
	// (Codea/Project/User sources). It is internal to the manager and used for
	// runtime-config sync; empty for runtime-built-in skills.
	dir string
}

// SkillRequirement declares an agent's hard dependency on a named skill.
type SkillRequirement struct {
	Name string
}

// Diagnostic stages for SkillError.
const (
	StageDiscover = "discover"
	StageLoad     = "load"
	StageRequire  = "require"
)

// SkillError is a per-skill diagnostic. Stage distinguishes discovery, load and
// requirement-validation failures. It implements error.
type SkillError struct {
	Name    string
	Source  SkillSource
	Stage   string
	Message string
}

func (e SkillError) Error() string {
	return fmt.Sprintf("skill %q (source=%s) %s: %s", e.Name, e.Source, e.Stage, e.Message)
}
