package main

import (
	"os"
	"path/filepath"
	"testing"

	"codea/tui/internal/app"
	"codea/tui/internal/checkpoint"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

func TestConfigureCheckpointCreatesShadowStoreOutsideProject(t *testing.T) {
	project := t.TempDir()
	home := filepath.Join(t.TempDir(), "Codea Home 中文")
	model := app.NewModel(fakeruntime.New())

	configureCheckpoint(model, home, project)

	id, err := checkpoint.WorkspaceID(project)
	if err != nil {
		t.Fatal(err)
	}
	gitHead := filepath.Join(home, "checkpoints", id, "git", "HEAD")
	if _, err := os.Stat(gitHead); err != nil {
		t.Fatalf("checkpoint shadow repository not initialized: %v", err)
	}
	if insideProjectForTest(project, gitHead) {
		t.Fatalf("shadow repository must stay outside project: %s", gitHead)
	}
}

func TestConfigureCheckpointGitUnavailableDoesNotBlockComposition(t *testing.T) {
	project := t.TempDir()
	home := t.TempDir()
	model := app.NewModel(fakeruntime.New())
	t.Setenv("PATH", "")

	// Composition must degrade the optional checkpoint subsystem rather than
	// failing/panicking and preventing Codea startup.
	configureCheckpoint(model, home, project)

	id, err := checkpoint.WorkspaceID(project)
	if err != nil {
		t.Fatal(err)
	}
	metadata := filepath.Join(home, "checkpoints", id, "checkpoints.json")
	if _, err := os.Stat(metadata); !os.IsNotExist(err) {
		t.Fatalf("Git-unavailable composition must not claim checkpoint metadata, err=%v", err)
	}
}

func insideProjectForTest(project, target string) bool {
	rel, err := filepath.Rel(project, target)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && len(rel) > 3 && rel[:3] != ".."+string(filepath.Separator))
}
