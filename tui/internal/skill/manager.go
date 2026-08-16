package skill

import (
	"context"
	"fmt"
	"sort"

	"codea/tui/internal/runtime"
)

// Snapshot is a unified skill view plus per-skill diagnostics.
type Snapshot struct {
	Skills []Skill
	Errors []SkillError
}

// Manager reconciles Codea's skill configuration with the runtime's loaded
// skill set. It owns discovery, enable/disable persistence, runtime-config sync
// and loaded-state reconciliation.
type Manager struct {
	roots      []Root
	store      Store
	targetDir  string
	projectDir string
	runtime    runtime.SkillRuntime
}

// NewManager constructs a Manager. projectDir is the project directory passed to
// the runtime so it can resolve project-scoped skills; it may be empty.
func NewManager(roots []Root, store Store, targetDir string, projectDir string, rt runtime.SkillRuntime) *Manager {
	return &Manager{roots: roots, store: store, targetDir: targetDir, projectDir: projectDir, runtime: rt}
}

// List discovers skills, applies enable/disable overrides, and reconciles the
// loaded state against the runtime. A per-skill failure never aborts the whole
// list; those diagnostics are returned in Snapshot.Errors.
func (m *Manager) List(ctx context.Context) (Snapshot, error) {
	skills, errs := Discover(m.roots)

	overrides, err := m.store.Load()
	if err != nil {
		return Snapshot{}, fmt.Errorf("load skill overrides: %w", err)
	}
	skills = applyOverrides(skills, overrides)

	skills, err = m.reconcileLoaded(ctx, skills)
	if err != nil {
		return Snapshot{}, err
	}

	sortSkills(skills)
	return Snapshot{Skills: skills, Errors: errs}, nil
}

// SetEnabled persists an enable/disable override for a Codea skill and re-syncs
// the runtime config so a disabled skill is no longer materialized. Only
// SourceCodea skills can be managed; project/user/runtime skills are read-only
// and are never toggled.
func (m *Manager) SetEnabled(name string, enabled bool) error {
	skills, _ := Discover(m.roots)
	found := false
	for _, s := range skills {
		if s.Name == name && s.Source == SourceCodea {
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("codea skill %q not found", name)
	}

	overrides, err := m.store.Load()
	if err != nil {
		return fmt.Errorf("load skill overrides: %w", err)
	}
	overrides[name] = enabled
	if err := m.store.Save(overrides); err != nil {
		return fmt.Errorf("save skill overrides: %w", err)
	}
	return SyncEnabled(m.roots, m.store, m.targetDir)
}

// applyOverrides sets Enabled from explicit overrides. Overrides only ever
// apply to SourceCodea skills: project/user/runtime skills are read-only and
// remain available regardless of a same-named Codea override.
func applyOverrides(skills []Skill, overrides map[string]bool) []Skill {
	for i := range skills {
		if skills[i].Source != SourceCodea {
			skills[i].Enabled = true
			continue
		}
		if v, ok := overrides[skills[i].Name]; ok {
			skills[i].Enabled = v
		} else {
			skills[i].Enabled = true
		}
	}
	return skills
}

// reconcileLoaded marks each skill Loaded based on the runtime's reported set,
// and adds runtime-built-in skills that are loaded but not on the filesystem.
func (m *Manager) reconcileLoaded(ctx context.Context, skills []Skill) ([]Skill, error) {
	loaded, err := m.runtime.ListSkills(ctx, m.projectDir)
	if err != nil {
		return nil, fmt.Errorf("list loaded skills: %w", err)
	}

	known := make(map[string]bool, len(skills))
	for _, s := range skills {
		known[s.Name] = true
	}
	loadedNames := make(map[string]bool, len(loaded))
	for _, l := range loaded {
		loadedNames[l.Name] = true
		if !known[l.Name] {
			skills = append(skills, Skill{
				Name:        l.Name,
				Description: l.Description,
				Source:      SourceRuntime,
				Installed:   true,
				Enabled:     true,
				Loaded:      true,
			})
		}
	}

	for i := range skills {
		skills[i].Loaded = loadedNames[skills[i].Name]
		if skills[i].Installed && skills[i].Enabled && !skills[i].Loaded {
			skills[i].LoadError = "not reported as loaded by runtime"
		} else {
			skills[i].LoadError = ""
		}
	}
	return skills, nil
}

func sortSkills(skills []Skill) {
	sort.Slice(skills, func(i, j int) bool {
		if skills[i].Name != skills[j].Name {
			return skills[i].Name < skills[j].Name
		}
		return skills[i].Source < skills[j].Source
	})
}
