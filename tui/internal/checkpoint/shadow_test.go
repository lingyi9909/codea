package checkpoint

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type gitState struct {
	head, branch, status, refs, index string
}

func runGit(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return string(out)
}

func initProjectGit(t *testing.T, dir string) {
	t.Helper()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.name", "Test")
	runGit(t, dir, "config", "user.email", "test@example.invalid")
	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("base\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "tracked.txt")
	runGit(t, dir, "commit", "-m", "base")
	if err := os.WriteFile(filepath.Join(dir, "tracked.txt"), []byte("modified\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "staged.txt"), []byte("staged\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "staged.txt")
}

func captureGitState(t *testing.T, dir string) gitState {
	t.Helper()
	indexPath := filepath.Join(dir, ".git", "index")
	indexBytes, err := os.ReadFile(indexPath)
	if err != nil {
		t.Fatal(err)
	}
	indexSum := sha256.Sum256(indexBytes)
	return gitState{
		head:   strings.TrimSpace(runGit(t, dir, "rev-parse", "HEAD")),
		branch: strings.TrimSpace(runGit(t, dir, "branch", "--show-current")),
		status: runGit(t, dir, "status", "--porcelain=v1", "-z"),
		refs:   runGit(t, dir, "show-ref"),
		index:  fmt.Sprintf("%x", indexSum),
	}
}

func TestShadowInitializationDoesNotTouchProjectGitState(t *testing.T) {
	project := t.TempDir()
	initProjectGit(t, project)
	before := captureGitState(t, project)
	home := t.TempDir()

	repo, err := OpenOrInit(context.Background(), home, project, NewGitRunner())
	if err != nil {
		t.Fatal(err)
	}
	if repo.GitDir() == filepath.Join(project, ".git") {
		t.Fatal("shadow git dir must not be project .git")
	}
	after := captureGitState(t, project)
	if before != after {
		t.Fatalf("project git state changed\nbefore=%+v\nafter=%+v", before, after)
	}
}

func TestShadowSupportsNonGitProject(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "main.go"), []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	repo, err := OpenOrInit(context.Background(), t.TempDir(), project, NewGitRunner())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(repo.GitDir(), "HEAD")); err != nil {
		t.Fatalf("shadow repository not initialized: %v", err)
	}
}

func TestShadowWritesDeterministicExcludes(t *testing.T) {
	repo, err := OpenOrInit(context.Background(), t.TempDir(), t.TempDir(), NewGitRunner())
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(repo.GitDir(), "info", "exclude"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, want := range []string{".git/", "target/", "build/", "dist/", "node_modules/", ".codea/", ".env", ".env.*", "*.pem", "*.key", "*.p12", "*.pfx", "credentials*", "secrets*"} {
		if !strings.Contains(text, want+"\n") {
			t.Fatalf("missing exclude %q in:\n%s", want, text)
		}
	}
}
