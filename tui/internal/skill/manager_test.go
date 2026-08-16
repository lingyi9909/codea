package skill

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"codea/tui/internal/runtime"
)

type memStore struct{ m map[string]bool }

func (s *memStore) Load() (map[string]bool, error) {
	out := map[string]bool{}
	for k, v := range s.m {
		out[k] = v
	}
	return out, nil
}

func (s *memStore) Save(m map[string]bool) error {
	s.m = m
	return nil
}

type fakeSkillRuntime struct {
	loaded []runtime.LoadedSkill
	err    error
}

func (f *fakeSkillRuntime) ListSkills(context.Context, string) ([]runtime.LoadedSkill, error) {
	return f.loaded, f.err
}

func newTestManager(t *testing.T, root string, rt runtime.SkillRuntime, overrides map[string]bool) *Manager {
	t.Helper()
	return NewManager(
		[]Root{{Dir: root, Source: SourceCodea}},
		&memStore{m: overrides},
		filepath.Join(t.TempDir(), "target"),
		filepath.Join(t.TempDir(), "project"),
		rt,
		DefaultPolicy,
	)
}

func TestManagerListReconcilesLoaded(t *testing.T) {
	root := t.TempDir()
	writeSkillDir(t, root, "git")
	writeSkillDir(t, root, "code-review")

	m := newTestManager(t, root, &fakeSkillRuntime{
		loaded: []runtime.LoadedSkill{{Name: "git", Description: "Git helpers."}},
	}, nil)

	snap, err := m.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]Skill{}
	for _, s := range snap.Skills {
		byName[s.Name] = s
	}
	if !byName["git"].Loaded {
		t.Error("git should be loaded")
	}
	if byName["code-review"].Loaded || byName["code-review"].LoadError == "" {
		t.Errorf("code-review should be enabled-but-not-loaded with an error: %+v", byName["code-review"])
	}
}

func TestManagerListAddsRuntimeSkills(t *testing.T) {
	root := t.TempDir()
	writeSkillDir(t, root, "git")

	m := newTestManager(t, root, &fakeSkillRuntime{
		loaded: []runtime.LoadedSkill{
			{Name: "git"},
			{Name: "customize-opencode", Description: "builtin"},
		},
	}, nil)

	snap, err := m.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]Skill{}
	for _, s := range snap.Skills {
		byName[s.Name] = s
	}
	if !byName["customize-opencode"].Installed || byName["customize-opencode"].Source != SourceRuntime {
		t.Errorf("runtime skill should be installed with SourceRuntime: %+v", byName["customize-opencode"])
	}
	if !byName["customize-opencode"].Loaded {
		t.Error("runtime skill should be loaded")
	}
}

func TestManagerListAppliesOverride(t *testing.T) {
	root := t.TempDir()
	writeSkillDir(t, root, "git")

	m := newTestManager(t, root, &fakeSkillRuntime{}, map[string]bool{"git": false})

	snap, err := m.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	// Under strict DefaultPolicy a disabled Codea skill is filtered from the view.
	if len(snap.Skills) != 0 {
		t.Errorf("disabled Codea skill must be filtered from strict view: %+v", snap.Skills)
	}
}

func TestManagerSetEnabledDisablesAndSyncs(t *testing.T) {
	root := t.TempDir()
	writeSkillDir(t, root, "git")

	m := newTestManager(t, root, &fakeSkillRuntime{}, nil)
	if err := m.SetEnabled("git", false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(m.targetDir, "git")); !os.IsNotExist(err) {
		t.Fatalf("disabled skill must not be synced: %v", err)
	}
	// The override must be persisted for the next List: a disabled Codea skill is
	// filtered from the strict view, so the re-list is empty.
	snap, err := m.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(snap.Skills) != 0 {
		t.Errorf("disabled Codea skill must be filtered from strict view: %+v", snap.Skills)
	}
}

func TestManagerSetEnabledUnknownSkill(t *testing.T) {
	root := t.TempDir()
	m := newTestManager(t, root, &fakeSkillRuntime{}, nil)
	if err := m.SetEnabled("nope", false); err == nil {
		t.Fatal("expected error for unknown skill")
	}
}

func TestManagerListRuntimeError(t *testing.T) {
	root := t.TempDir()
	writeSkillDir(t, root, "git")
	m := newTestManager(t, root, &fakeSkillRuntime{err: os.ErrClosed}, nil)
	if _, err := m.List(context.Background()); err == nil {
		t.Fatal("expected error when runtime ListSkills fails")
	}
}
