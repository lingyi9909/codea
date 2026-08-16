package skill

import "strings"

// SkillPolicy bundles a mode with the set of approved skill names. Approved
// only ever gates SourceCodea skills. The runtime env isolates user and external
// (.claude/.agents) skills in BOTH modes; project skills are isolated only in
// strict mode. Runtime built-ins are never gated.
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

// CompatibleAllowed reports whether a skill is in the effective set under
// compatible mode: Codea, project and runtime built-ins are allowed; user
// skills (~/.config/opencode/skills) stay isolated in both modes and are never
// part of the compatible set.
func (p SkillPolicy) CompatibleAllowed(s Skill) bool {
	switch s.Source {
	case SourceCodea, SourceProject, SourceRuntime:
		return true
	default:
		return false
	}
}

// Allowed reports whether s is in the effective skill set under p's mode.
func (p SkillPolicy) Allowed(s Skill) bool {
	if p.Mode == SkillModeStrict {
		return p.StrictAllowed(s)
	}
	return p.CompatibleAllowed(s)
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

// FilterForMode returns the effective skill set under p's mode. Strict keeps
// only approved Codea skills; compatible keeps Codea, project and runtime
// built-ins but drops user skills. The enabled dimension is deliberately NOT
// gated here: it is orthogonal to mode (Installed/Enabled/Loaded stay
// independent, per Task 10), and materialization is gated by Sync, which
// already skips disabled skills.
func FilterForMode(skills []Skill, p SkillPolicy) []Skill {
	out := make([]Skill, 0, len(skills))
	for _, s := range skills {
		if p.Allowed(s) {
			out = append(out, s)
		}
	}
	return out
}
