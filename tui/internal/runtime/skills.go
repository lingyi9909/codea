package runtime

import "context"

// LoadedSkill is a skill the runtime reports as currently loaded. Location is
// an opaque, runtime-specific source hint (e.g. a path or "<built-in>"); the
// Codea Skill domain maps it to its own SkillSource and never treats it as a
// vendor DTO.
type LoadedSkill struct {
	Name        string
	Description string
	Location    string
}

// SkillRuntime is implemented by runtime adapters that can report the set of
// skills currently loaded by the runtime. It is kept separate from AgentRuntime
// so the core session/event/approval contract stays minimal.
type SkillRuntime interface {
	// ListSkills reports the skills the runtime currently has loaded. directory
	// is the project directory used to resolve project-scoped skills.
	ListSkills(ctx context.Context, directory string) ([]LoadedSkill, error)
}
