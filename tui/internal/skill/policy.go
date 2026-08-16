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
