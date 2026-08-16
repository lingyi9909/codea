package skill

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"codea/tui/internal/runtime"
)

// TestManagerFullChain exercises the complete lifecycle with real persistence
// and a real filesystem sync target: discover, reconcile loaded state, disable
// one skill (override persists + sync removes it), then re-list after the
// runtime stops reporting the disabled skill.
func TestManagerFullChain(t *testing.T) {
	root := t.TempDir()
	writeSkillDir(t, root, "code-review")
	writeSkillDir(t, root, "unit-test")

	targetDir := filepath.Join(t.TempDir(), "target")
	statePath := filepath.Join(t.TempDir(), "state", "skills.json")
	rt := &fakeSkillRuntime{loaded: []runtime.LoadedSkill{
		{Name: "code-review"},
		{Name: "unit-test"},
	}}

	m := NewManager(
		[]Root{{Dir: root, Source: SourceCodea}},
		NewFileStore(statePath),
		targetDir,
		filepath.Join(t.TempDir(), "project"),
		rt,
		DefaultPolicy,
	)

	// Initial list: both discovered, enabled by default, loaded from runtime.
	snap, err := m.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	byName := map[string]Skill{}
	for _, s := range snap.Skills {
		byName[s.Name] = s
	}
	if len(snap.Skills) != 2 {
		t.Fatalf("List = %d skills, want 2", len(snap.Skills))
	}
	if !byName["code-review"].Loaded || !byName["unit-test"].Loaded {
		t.Fatalf("both skills should load initially: %+v", byName)
	}

	// Disable code-review: override persists to disk and sync removes it from
	// the controlled runtime config dir.
	if err := m.SetEnabled("code-review", false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(targetDir, "code-review")); !os.IsNotExist(err) {
		t.Fatalf("disabled skill must not be synced: %v", err)
	}
	if _, err := os.Stat(filepath.Join(targetDir, "unit-test", "SKILL.md")); err != nil {
		t.Fatalf("enabled skill must be synced: %v", err)
	}

	// Simulate the runtime unloading the disabled skill after its config dir no
	// longer contains it, then re-list to confirm the persisted override and the
	// now-accurate loaded state.
	rt.loaded = []runtime.LoadedSkill{{Name: "unit-test"}}
	snap, err = m.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	byName = map[string]Skill{}
	for _, s := range snap.Skills {
		byName[s.Name] = s
	}
	if byName["code-review"].Enabled {
		t.Error("code-review override should persist as disabled")
	}
	if !byName["unit-test"].Enabled || !byName["unit-test"].Loaded {
		t.Errorf("unit-test should stay enabled+loaded: %+v", byName["unit-test"])
	}
	if byName["code-review"].Loaded {
		t.Error("code-review should no longer be loaded after runtime unload")
	}
}
