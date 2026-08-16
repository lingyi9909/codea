package skill

import "fmt"

// SkillMode selects how Codea composes the runtime's effective skill set.
type SkillMode string

const (
	// SkillModeStrict is the V1 default: only Approved + Enabled Codea skills
	// enter the runtime; project and user skills are isolated out.
	SkillModeStrict SkillMode = "strict"
	// SkillModeCompatible allows Codea + Project (+ runtime built-in) skills;
	// user skills (~/.config/opencode/skills) stay isolated (Task 1 S6 baseline).
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
