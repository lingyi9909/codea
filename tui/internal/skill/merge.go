package skill

import "sort"

// AgentSkillConfig declares how an agent composes its skill set from the
// default/global skills plus its own explicit and required skills.
type AgentSkillConfig struct {
	// Inherit includes the default/global skill names when true.
	Inherit bool
	// Skills lists explicit skill names the agent additionally uses.
	Skills []string
	// RequiredSkills lists hard dependencies that must be available at startup.
	RequiredSkills []SkillRequirement
}

// MergeSkillNames deterministically merges the default (global) skill names with
// the agent's explicit skills and required skills. When inherit is false the
// defaults are dropped. The result is deduplicated by name and sorted, so it
// does not depend on input order.
func MergeSkillNames(defaults []string, cfg AgentSkillConfig) []string {
	set := make(map[string]struct{}, len(defaults)+len(cfg.Skills)+len(cfg.RequiredSkills))
	if cfg.Inherit {
		for _, n := range defaults {
			set[n] = struct{}{}
		}
	}
	for _, n := range cfg.Skills {
		set[n] = struct{}{}
	}
	for _, r := range cfg.RequiredSkills {
		set[r.Name] = struct{}{}
	}

	out := make([]string, 0, len(set))
	for n := range set {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}
