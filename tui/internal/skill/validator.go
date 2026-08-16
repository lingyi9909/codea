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
