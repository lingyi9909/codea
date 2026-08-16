package skill

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"codea/tui/internal/runtime"
)

// TestManagerListStrictFilters verifies the manager view drops non-approved and
// non-Codea skills in strict mode, while reconcileLoaded re-adds runtime built-ins.
func TestManagerListStrictFilters(t *testing.T) {
	codeaRoot := t.TempDir()
	writeSkillDir(t, codeaRoot, "code-review")
	writeSkillDir(t, codeaRoot, "experimental")
	projectRoot := t.TempDir()
	writeSkillDir(t, projectRoot, "proj")

	m := NewManager(
		[]Root{
			{Dir: codeaRoot, Source: SourceCodea},
			{Dir: projectRoot, Source: SourceProject},
		},
		&memStore{},
		filepath.Join(t.TempDir(), "target"),
		filepath.Join(t.TempDir(), "project"),
		&fakeSkillRuntime{loaded: []runtime.LoadedSkill{
			{Name: "code-review"},
			{Name: "customize-opencode", Description: "builtin"},
		}},
		SkillPolicy{Mode: SkillModeStrict, Approved: map[string]bool{"code-review": true}},
	)

	snap, err := m.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	names := map[string]bool{}
	for _, s := range snap.Skills {
		names[s.Name] = true
	}
	if !names["code-review"] {
		t.Fatalf("approved Codea must appear: %+v", names)
	}
	if names["experimental"] || names["proj"] {
		t.Fatalf("non-approved/project must be filtered in strict: %+v", names)
	}
	if !names["customize-opencode"] {
		t.Fatalf("runtime built-in must be re-added: %+v", names)
	}
}

// TestSyncEnabledStrictExcludesUnapproved verifies strict materialization copies
// only approved+enabled Codea skills.
func TestSyncEnabledStrictExcludesUnapproved(t *testing.T) {
	root := t.TempDir()
	writeSkillDir(t, root, "code-review")
	writeSkillDir(t, root, "experimental")

	target := filepath.Join(t.TempDir(), "skills")
	p := SkillPolicy{Mode: SkillModeStrict, Approved: map[string]bool{"code-review": true}}

	if err := SyncEnabled([]Root{{Dir: root, Source: SourceCodea}}, &memStore{}, target, p); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(target, "code-review", "SKILL.md")); err != nil {
		t.Fatalf("approved skill must be synced: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "experimental")); !os.IsNotExist(err) {
		t.Fatalf("unapproved skill must not be synced: %v", err)
	}
}
