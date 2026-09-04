package checkpoint

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestWorkspaceIDStableForNormalizedRoot(t *testing.T) {
	root := t.TempDir()
	a, err := WorkspaceID(root)
	if err != nil {
		t.Fatal(err)
	}
	b, err := WorkspaceID(filepath.Join(root, "."))
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("workspace id changed after normalization: %q != %q", a, b)
	}
	if len(a) != 64 {
		t.Fatalf("workspace id must be sha256 hex, got %q", a)
	}
}

func TestWorkspaceIDCanonicalizesWindowsSeparators(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows path semantics only")
	}
	root := t.TempDir()
	forward := filepath.ToSlash(root)
	backward := strings.ReplaceAll(forward, "/", "\\")
	a, err := WorkspaceID(forward)
	if err != nil {
		t.Fatal(err)
	}
	b, err := WorkspaceID(backward)
	if err != nil {
		t.Fatal(err)
	}
	if a != b {
		t.Fatalf("slash/backslash variants differ: %q != %q", a, b)
	}
}

func TestCheckpointPathsUseHashedWorkspaceAndUnicodeHome(t *testing.T) {
	base := t.TempDir()
	project := filepath.Join(base, "Sample Project")
	home := filepath.Join(base, "Codea Home 中文")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	p, err := ResolvePaths(home, project)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(p.Root, filepath.Base(project)) {
		t.Fatalf("raw project path leaked into checkpoint root: %s", p.Root)
	}
	if filepath.Base(filepath.Dir(p.Root)) != "checkpoints" {
		t.Fatalf("unexpected root: %s", p.Root)
	}
	if p.GitDir != filepath.Join(p.Root, "git") {
		t.Fatalf("unexpected git dir: %s", p.GitDir)
	}
	if p.Metadata != filepath.Join(p.Root, "checkpoints.json") {
		t.Fatalf("unexpected metadata: %s", p.Metadata)
	}
	if p.RestoreState != filepath.Join(p.Root, "restore-state.json") {
		t.Fatalf("unexpected restore state: %s", p.RestoreState)
	}
}

func TestCheckpointRootCannotLiveInsideProject(t *testing.T) {
	project := t.TempDir()
	_, err := ResolvePaths(filepath.Join(project, ".codea-home"), project)
	if err == nil {
		t.Fatal("expected checkpoint root inside project to be rejected")
	}
	if !IsCode(err, CodeCheckpointUnavailable) {
		t.Fatalf("expected CHECKPOINT_UNAVAILABLE, got %v", err)
	}
}
