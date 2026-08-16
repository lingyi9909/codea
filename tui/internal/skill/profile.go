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
