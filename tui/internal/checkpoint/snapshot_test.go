package checkpoint

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
)

type recordingRunner struct {
	delegate Runner
	mu       sync.Mutex
	calls    []recordedCall
}

type recordedCall struct {
	args  []string
	stdin []byte
}

func (r *recordingRunner) Run(ctx context.Context, args []string, stdin []byte) (Result, error) {
	r.mu.Lock()
	r.calls = append(r.calls, recordedCall{args: append([]string(nil), args...), stdin: append([]byte(nil), stdin...)})
	r.mu.Unlock()
	return r.delegate.Run(ctx, args, stdin)
}

func checkpointGit(t *testing.T, s *Service, args ...string) string {
	t.Helper()
	cmdArgs := append([]string{"--git-dir=" + s.repo.GitDir()}, args...)
	out, err := exec.Command("git", cmdArgs...).CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", cmdArgs, err, out)
	}
	return string(out)
}

func commitFiles(t *testing.T, s *Service, commit string) []string {
	t.Helper()
	out := checkpointGit(t, s, "ls-tree", "-r", "--name-only", commit)
	lines := strings.Fields(out)
	sort.Strings(lines)
	return lines
}

func commitFile(t *testing.T, s *Service, commit, path string) string {
	t.Helper()
	return checkpointGit(t, s, "show", commit+":"+path)
}

func TestSnapshotCapturesModifyAddDeleteAndRename(t *testing.T) {
	project := t.TempDir()
	mustWrite := func(rel, body string) {
		p := filepath.Join(project, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	mustWrite("a.txt", "a0")
	mustWrite("b.txt", "b0")
	mustWrite("old.txt", "rename-me")
	mustWrite("go.mod", "module example")
	mustWrite("docs/readme.md", "doc")
	s, err := NewService(context.Background(), t.TempDir(), project, NewGitRunner())
	if err != nil {
		t.Fatal(err)
	}
	first, err := s.Create(context.Background(), CreateRequest{TaskID: "task-31", TurnID: "turn-1", Kind: KindBaseline})
	if err != nil {
		t.Fatal(err)
	}
	if got := commitFile(t, s, first.Commit, "a.txt"); got != "a0" {
		t.Fatalf("baseline content=%q", got)
	}

	mustWrite("a.txt", "a1")
	if err := os.Remove(filepath.Join(project, "b.txt")); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(filepath.Join(project, "old.txt"), filepath.Join(project, "renamed.txt")); err != nil {
		t.Fatal(err)
	}
	mustWrite("new.txt", "new")
	second, err := s.Create(context.Background(), CreateRequest{TaskID: "task-31", TurnID: "turn-2", Kind: KindManual})
	if err != nil {
		t.Fatal(err)
	}

	gotFiles := strings.Join(commitFiles(t, s, second.Commit), ",")
	for _, want := range []string{"a.txt", "docs/readme.md", "go.mod", "new.txt", "renamed.txt"} {
		if !strings.Contains(gotFiles, want) {
			t.Fatalf("missing %s in %s", want, gotFiles)
		}
	}
	for _, gone := range []string{"b.txt", "old.txt"} {
		if strings.Contains(gotFiles, gone) {
			t.Fatalf("deleted/renamed source still present: %s", gotFiles)
		}
	}
	if got := commitFile(t, s, second.Commit, "a.txt"); got != "a1" {
		t.Fatalf("modified content=%q", got)
	}
}

func TestSnapshotSkipsSensitiveGeneratedLargeAndBinaryFilesWithEvidence(t *testing.T) {
	project := t.TempDir()
	files := map[string][]byte{
		"src/main.go":      []byte("package main\n"),
		".env":             []byte("TOKEN=secret"),
		"target/out.bin":   []byte("generated"),
		".codea/local.txt": []byte("internal"),
		"secret.pem":       []byte("pem"),
		"large.dat":        bytes.Repeat([]byte("L"), int(DefaultMaxFileSize)+1),
		"binary.dat":       append([]byte{0}, bytes.Repeat([]byte("B"), int(DefaultBinaryThreshold)+1)...),
	}
	for rel, data := range files {
		p := filepath.Join(project, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	s, err := NewService(context.Background(), t.TempDir(), project, NewGitRunner())
	if err != nil {
		t.Fatal(err)
	}
	cp, err := s.Create(context.Background(), CreateRequest{Kind: KindBaseline})
	if err != nil {
		t.Fatal(err)
	}
	got := strings.Join(commitFiles(t, s, cp.Commit), ",")
	if got != "src/main.go" {
		t.Fatalf("unexpected captured files: %s", got)
	}
	reasons := map[string]string{}
	for _, skipped := range cp.Skipped {
		reasons[skipped.Path] = skipped.Reason
	}
	for _, want := range []string{".env", "secret.pem", "large.dat", "binary.dat"} {
		if reasons[want] == "" {
			t.Fatalf("missing skip evidence for %s: %+v", want, cp.Skipped)
		}
	}
}

func TestSnapshotUsesNULPathspecInsteadOfGiantArgv(t *testing.T) {
	project := t.TempDir()
	for i := 0; i < 1200; i++ {
		rel := fmt.Sprintf("src/f-%04d.txt", i)
		p := filepath.Join(project, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	rr := &recordingRunner{delegate: NewGitRunner()}
	s, err := NewService(context.Background(), t.TempDir(), project, rr)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(context.Background(), CreateRequest{Kind: KindBaseline}); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, call := range rr.calls {
		joined := strings.Join(call.args, " ")
		if strings.Contains(joined, "--pathspec-from-file=-") {
			found = true
			if len(call.args) > 12 {
				t.Fatalf("candidate paths leaked into argv: %d args", len(call.args))
			}
			if bytes.Count(call.stdin, []byte{0}) != 1200 {
				t.Fatalf("expected 1200 NUL pathspecs, got %d", bytes.Count(call.stdin, []byte{0}))
			}
		}
	}
	if !found {
		t.Fatal("expected pathspec-from-file staging call")
	}
}

func TestSnapshotDedupeReusesCommitButRecordsEvent(t *testing.T) {
	project := t.TempDir()
	if err := os.WriteFile(filepath.Join(project, "a.txt"), []byte("same"), 0o644); err != nil {
		t.Fatal(err)
	}
	s, err := NewService(context.Background(), t.TempDir(), project, NewGitRunner())
	if err != nil {
		t.Fatal(err)
	}
	a, err := s.Create(context.Background(), CreateRequest{Kind: KindBaseline})
	if err != nil {
		t.Fatal(err)
	}
	b, err := s.Create(context.Background(), CreateRequest{Kind: KindManual})
	if err != nil {
		t.Fatal(err)
	}
	if a.Commit != b.Commit {
		t.Fatalf("unchanged tree created new commit: %s != %s", a.Commit, b.Commit)
	}
	if a.ID == b.ID {
		t.Fatalf("checkpoint event id must be unique: %s", a.ID)
	}
	list, err := s.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("expected two checkpoint records, got %d", len(list))
	}
}
