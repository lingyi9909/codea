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
