package checkpoint

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestSpacesUnicodeWorkspaceLifecycle deliberately uses a path shape that is
// troublesome for shell-based Git integration. On Windows CI this exercises
// native git.exe with spaces and non-ASCII path components; on POSIX it remains
// a useful cross-platform direct-argv regression.
func TestSpacesUnicodeWorkspaceLifecycle(t *testing.T) {
	base := t.TempDir()
	project := filepath.Join(base, "Codea Checkpoint 中文", "Sample Project")
	home := filepath.Join(base, "Codea Home 中文")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	initProjectGit(t, project)
	before := captureGitState(t, project)

	s, err := NewService(context.Background(), home, project, NewGitRunner())
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := s.Create(context.Background(), CreateRequest{TaskID: "task-31", TurnID: "unicode-turn", Kind: KindBaseline})
	if err != nil {
		t.Fatal(err)
	}

	writeText(t, project, "tracked.txt", "mutated in unicode workspace\n")
	writeText(t, project, "新增 文件.txt", "new file\n")
	result, err := s.Restore(context.Background(), baseline.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Safety.ID == "" || result.Safety.Kind != KindSafety {
		t.Fatalf("missing safety checkpoint: %+v", result.Safety)
	}
	if got := readText(t, project, "tracked.txt"); got != "modified\n" {
		t.Fatalf("baseline bytes not restored: %q", got)
	}
	if _, err := os.Stat(filepath.Join(project, "新增 文件.txt")); !os.IsNotExist(err) {
		t.Fatalf("new file should be removed, err=%v", err)
	}
	after := captureGitState(t, project)
	if before.head != after.head || before.branch != after.branch || before.index != after.index || before.refs != after.refs {
		t.Fatalf("project Git metadata changed\nbefore=%+v\nafter=%+v", before, after)
	}

	if _, err := s.Restore(context.Background(), result.Safety.ID); err != nil {
		t.Fatal(err)
	}
	if got := readText(t, project, "tracked.txt"); got != "mutated in unicode workspace\n" {
		t.Fatalf("safety bytes not restored: %q", got)
	}
	if got := readText(t, project, "新增 文件.txt"); got != "new file\n" {
		t.Fatalf("safety new file not restored: %q", got)
	}
}
