package skill

import (
	"os"
	"path/filepath"
	"testing"
)

// TestSyncEnabledColdStart verifies the cold-start path: with no manual toggle,
// an enabled Codea skill is materialized into the controlled config dir and a
// disabled one is not.
func TestSyncEnabledColdStart(t *testing.T) {
	root := t.TempDir()
	writeSkillDir(t, root, "code-review")
	writeSkillDir(t, root, "unit-test")

	target := filepath.Join(t.TempDir(), "skills")
	store := &memStore{m: map[string]bool{"unit-test": false}}

	if err := SyncEnabled([]Root{{Dir: root, Source: SourceCodea}}, store, target); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(target, "code-review", "SKILL.md")); err != nil {
		t.Fatalf("enabled codea skill not synced on cold start: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "unit-test")); !os.IsNotExist(err) {
		t.Fatalf("disabled codea skill must not be synced: %v", err)
	}
}

// TestSyncEnabledPreservesForeignDir guards P0-1: the Codea sync must only ever
// rewrite its own isolated target dir and must never delete a user's existing
// skills in a separate directory (e.g. ~/.config/opencode/skills).
func TestSyncEnabledPreservesForeignDir(t *testing.T) {
	root := t.TempDir()
	codeaDir := writeSkillDir(t, root, "code-review")

	foreign := filepath.Join(t.TempDir(), "opencode", "skills")
	writeSkillDir(t, foreign, "user-skill")

	target := filepath.Join(t.TempDir(), "codea", "skills")

	if err := SyncEnabled([]Root{{Dir: root, Source: SourceCodea}}, &memStore{}, target); err != nil {
		t.Fatal(err)
	}
	// Disable the Codea skill and re-sync; the foreign dir must survive.
	if err := Sync([]Skill{{Name: "code-review", Source: SourceCodea, Enabled: false, dir: codeaDir}}, target); err != nil {
		t.Fatal(err)
	}

	if _, err := os.Stat(filepath.Join(foreign, "user-skill", "SKILL.md")); err != nil {
		t.Fatalf("foreign user skill must survive Codea sync: %v", err)
	}
}
