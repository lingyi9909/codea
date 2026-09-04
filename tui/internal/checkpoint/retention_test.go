package checkpoint

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRetentionKeepsNewestTwentyAndActiveTaskBaseline(t *testing.T) {
	items := make([]Checkpoint, 0, 26)
	items = append(items, Checkpoint{ID: "cp-000001", Kind: KindBaseline, TaskID: "task-31", Commit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"})
	for i := 2; i <= 26; i++ {
		items = append(items, Checkpoint{ID: formatCheckpointID(i), Kind: KindManual, TaskID: "task-31", Commit: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb", CreatedAt: time.Unix(int64(i), 0)})
	}
	got := pruneCheckpointRecords(items, 20, nil)
	if len(got) != 21 {
		t.Fatalf("expected 20 newest plus protected baseline, got %d", len(got))
	}
	if got[0].ID != "cp-000001" {
		t.Fatalf("active baseline pruned: first=%s", got[0].ID)
	}
	if got[len(got)-1].ID != "cp-000026" {
		t.Fatalf("newest missing: %s", got[len(got)-1].ID)
	}
}

func TestRetentionKeepsRecoveryReferences(t *testing.T) {
	items := make([]Checkpoint, 0, 30)
	for i := 1; i <= 30; i++ {
		items = append(items, Checkpoint{ID: formatCheckpointID(i), Kind: KindManual, Commit: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"})
	}
	recovery := &RestoreState{TargetID: "cp-000002", SafetyID: "cp-000003"}
	got := pruneCheckpointRecords(items, 20, recovery)
	seen := map[string]bool{}
	for _, cp := range got {
		seen[cp.ID] = true
	}
	if !seen["cp-000002"] || !seen["cp-000003"] {
		t.Fatalf("recovery checkpoints pruned: %+v", seen)
	}
	if !seen["cp-000030"] {
		t.Fatal("newest checkpoint missing")
	}
}

func formatCheckpointID(i int) string {
	const digits = "000000"
	s := []byte(digits)
	n := i
	for p := len(s) - 1; p >= 0 && n > 0; p-- {
		s[p] = byte('0' + n%10)
		n /= 10
	}
	return "cp-" + string(s)
}

type unavailableRunner struct{}

func (unavailableRunner) Run(context.Context, []string, []byte) (Result, error) {
	return Result{}, errors.New("executable file not found")
}

func TestGitUnavailableFailsVisibleWithoutMetadata(t *testing.T) {
	project := t.TempDir()
	home := t.TempDir()
	_, err := NewService(context.Background(), home, project, unavailableRunner{})
	if err == nil || !IsCode(err, CodeCheckpointUnavailable) {
		t.Fatalf("expected CHECKPOINT_UNAVAILABLE, got %v", err)
	}
	id, idErr := WorkspaceID(project)
	if idErr != nil {
		t.Fatal(idErr)
	}
	metadata := filepath.Join(home, "checkpoints", id, "checkpoints.json")
	if _, statErr := os.Stat(metadata); !os.IsNotExist(statErr) {
		t.Fatalf("metadata should not exist, err=%v", statErr)
	}
}
