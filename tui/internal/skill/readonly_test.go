package skill

import (
	"path/filepath"
	"testing"
)

// TestApplyOverridesOnlyAffectsCodea guards P1-3: a same-named override must
// disable only the Codea skill and leave project/user skills read-only.
func TestApplyOverridesOnlyAffectsCodea(t *testing.T) {
	skills := []Skill{
		{Name: "java-review", Source: SourceCodea},
		{Name: "java-review", Source: SourceProject},
		{Name: "java-review", Source: SourceUser},
	}
	out := applyOverrides(skills, map[string]bool{"java-review": false})

	got := map[SkillSource]bool{}
	for _, s := range out {
		got[s.Source] = s.Enabled
	}
	if got[SourceCodea] {
		t.Error("Codea skill should honor the disable override")
	}
	if !got[SourceProject] || !got[SourceUser] {
		t.Errorf("non-Codea skills must stay enabled (read-only): %+v", got)
	}
}

// TestSetEnabledRejectsNonCodea guards P1-3: the manager must refuse to toggle a
// project/user/runtime skill.
func TestSetEnabledRejectsNonCodea(t *testing.T) {
	root := t.TempDir()
	writeSkillDir(t, root, "proj")

	m := NewManager(
		[]Root{{Dir: root, Source: SourceProject}},
		&memStore{},
		filepath.Join(t.TempDir(), "target"),
		filepath.Join(t.TempDir(), "project"),
		&fakeSkillRuntime{},
		DefaultPolicy,
	)
	if err := m.SetEnabled("proj", false); err == nil {
		t.Fatal("expected error when toggling a non-Codea skill")
	}
}
