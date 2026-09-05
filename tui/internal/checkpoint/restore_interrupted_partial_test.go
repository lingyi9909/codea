package checkpoint

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
)

type failNthBlobRunner struct {
	delegate Runner
	mu       sync.Mutex
	failAt   int
	seen     int
}

func (r *failNthBlobRunner) Run(ctx context.Context, args []string, stdin []byte) (Result, error) {
	r.mu.Lock()
	isBlobRead := strings.Contains(strings.Join(args, " "), "cat-file blob")
	shouldFail := false
	if isBlobRead && r.failAt > 0 {
		r.seen++
		shouldFail = r.seen == r.failAt
	}
	r.mu.Unlock()
	if shouldFail {
		return Result{ExitCode: 89}, errors.New("injected second restore blob failure")
	}
	return r.delegate.Run(ctx, args, stdin)
}

func TestInterruptedRestoreReportsPartialFilesChanged(t *testing.T) {
	project := t.TempDir()
	writeText(t, project, "A.txt", "A0")
	writeText(t, project, "B.txt", "B0")
	runner := &failNthBlobRunner{delegate: NewGitRunner()}
	s, err := NewService(context.Background(), t.TempDir(), project, runner)
	if err != nil {
		t.Fatal(err)
	}
	baseline, err := s.Create(context.Background(), CreateRequest{Kind: KindBaseline})
	if err != nil {
		t.Fatal(err)
	}
	writeText(t, project, "A.txt", "A1")
	writeText(t, project, "B.txt", "B1")
	runner.failAt = 2

	result, err := s.Restore(context.Background(), baseline.ID)
	if err == nil || !IsCode(err, CodeRestoreInterrupted) {
		t.Fatalf("expected interrupted restore, result=%+v err=%v", result, err)
	}
	if result.FilesChanged != 1 {
		t.Fatalf("FilesChanged=%d, want 1 after first file was restored", result.FilesChanged)
	}
	if got := readText(t, project, "A.txt"); got != "A0" {
		t.Fatalf("A=%q, want restored A0", got)
	}
	if got := readText(t, project, "B.txt"); got != "B1" {
		t.Fatalf("B=%q, want unchanged B1 after injected failure", got)
	}
}
