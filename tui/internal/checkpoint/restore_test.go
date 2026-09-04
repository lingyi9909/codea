package checkpoint

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func writeText(t *testing.T, root, rel, body string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readText(t *testing.T, root, rel string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(rel)))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func TestRestoreExactAndSafetyCheckpointCanUndoRestore(t *testing.T) {
	project := t.TempDir()
	writeText(t, project, "A.txt", "A0")
	writeText(t, project, "B.txt", "B0")
	s, err := NewService(context.Background(), t.TempDir(), project, NewGitRunner())
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := s.Create(context.Background(), CreateRequest{TaskID: "task", TurnID: "turn", Kind: KindBaseline})
	if err != nil {
		t.Fatal(err)
	}

	writeText(t, project, "A.txt", "A1")
	if err := os.Remove(filepath.Join(project, "B.txt")); err != nil {
		t.Fatal(err)
	}
	writeText(t, project, "C.txt", "C1")

	result, err := s.Restore(context.Background(), baseline.ID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Target.ID != baseline.ID {
		t.Fatalf("target=%s", result.Target.ID)
	}
	if result.Safety.Kind != KindSafety || result.Safety.ID == "" {
		t.Fatalf("missing safety checkpoint: %+v", result.Safety)
	}
	if result.FilesChanged != 3 {
		t.Fatalf("files changed=%d, want 3", result.FilesChanged)
	}
	if got := readText(t, project, "A.txt"); got != "A0" {
		t.Fatalf("A=%q", got)
	}
	if got := readText(t, project, "B.txt"); got != "B0" {
		t.Fatalf("B=%q", got)
	}
	if _, err := os.Stat(filepath.Join(project, "C.txt")); !os.IsNotExist(err) {
		t.Fatalf("C should be removed, err=%v", err)
	}

	if _, err := s.Restore(context.Background(), result.Safety.ID); err != nil {
		t.Fatal(err)
	}
	if got := readText(t, project, "A.txt"); got != "A1" {
		t.Fatalf("undo A=%q", got)
	}
	if _, err := os.Stat(filepath.Join(project, "B.txt")); !os.IsNotExist(err) {
		t.Fatalf("undo should delete B, err=%v", err)
	}
	if got := readText(t, project, "C.txt"); got != "C1" {
		t.Fatalf("undo C=%q", got)
	}
}

func TestRestoreDoesNotTouchProjectHeadBranchIndexOrRefs(t *testing.T) {
	project := t.TempDir()
	initProjectGit(t, project)
	home := t.TempDir()
	s, err := NewService(context.Background(), home, project, NewGitRunner())
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := s.Create(context.Background(), CreateRequest{Kind: KindBaseline})
	if err != nil {
		t.Fatal(err)
	}
	writeText(t, project, "tracked.txt", "after restore target")
	before := captureGitState(t, project)
	if _, err := s.Restore(context.Background(), baseline.ID); err != nil {
		t.Fatal(err)
	}
	after := captureGitState(t, project)
	if before.head != after.head || before.branch != after.branch || before.index != after.index || before.refs != after.refs {
		t.Fatalf("project git metadata changed\nbefore=%+v\nafter=%+v", before, after)
	}
}

func TestRestoreRejectsMalformedAndUnknownCheckpointIDs(t *testing.T) {
	project := t.TempDir()
	writeText(t, project, "a.txt", "x")
	s, err := NewService(context.Background(), t.TempDir(), project, NewGitRunner())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.Create(context.Background(), CreateRequest{Kind: KindBaseline}); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"HEAD~1", "../cp-000001", "cp-999999"} {
		_, err := s.Restore(context.Background(), id)
		if err == nil || !IsCode(err, CodeInvalidCheckpoint) {
			t.Fatalf("id %q: expected invalid checkpoint, got %v", id, err)
		}
	}
}

type failAfterSafetyRunner struct {
	delegate Runner
	mu       sync.Mutex
	failBlob bool
}

func (r *failAfterSafetyRunner) Run(ctx context.Context, args []string, stdin []byte) (Result, error) {
	r.mu.Lock()
	shouldFail := r.failBlob && strings.Contains(strings.Join(args, " "), "cat-file blob")
	if shouldFail {
		r.failBlob = false
	}
	r.mu.Unlock()
	if shouldFail {
		return Result{ExitCode: 88}, errors.New("injected restore write failure")
	}
	return r.delegate.Run(ctx, args, stdin)
}

func TestInterruptedRestorePersistsRecoveryStateAndSurfacesOnReopen(t *testing.T) {
	project := t.TempDir()
	writeText(t, project, "a.txt", "base")
	home := t.TempDir()
	rr := &failAfterSafetyRunner{delegate: NewGitRunner()}
	s, err := NewService(context.Background(), home, project, rr)
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := s.Create(context.Background(), CreateRequest{Kind: KindBaseline})
	if err != nil {
		t.Fatal(err)
	}
	writeText(t, project, "a.txt", "mutated")
	rr.failBlob = true
	_, err = s.Restore(context.Background(), baseline.ID)
	if err == nil || !IsCode(err, CodeRestoreInterrupted) {
		t.Fatalf("expected interrupted restore, got %v", err)
	}
	if _, err := os.Stat(s.paths.RestoreState); err != nil {
		t.Fatalf("restore-state missing: %v", err)
	}

	reopened, err := NewService(context.Background(), home, project, NewGitRunner())
	if err != nil {
		t.Fatal(err)
	}
	recovery := reopened.Recovery()
	if recovery == nil || recovery.TargetID != baseline.ID || recovery.SafetyID == "" {
		t.Fatalf("bad recovery state: %+v", recovery)
	}
	if !strings.Contains(reopened.RecoveryGuidance(), recovery.SafetyID) {
		t.Fatalf("guidance missing safety id: %q", reopened.RecoveryGuidance())
	}
}
