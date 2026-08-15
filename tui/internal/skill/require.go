package skill

import (
	"errors"
	"fmt"
)

// ValidateRequirements checks that every required skill is installed, enabled
// and loaded. Any unavailable required skill is a hard failure: the returned
// error names the skill and the reason, and callers must not start degraded.
func ValidateRequirements(skills []Skill, reqs []SkillRequirement) error {
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
