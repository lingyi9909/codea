package app

import (
	"errors"
	"testing"

	"codea/tui/internal/checkpoint"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

func TestInterruptedRestoreInvalidatesRepoContext(t *testing.T) {
	repo := &invalidatingRepoContext{}
	m := NewModel(fakeruntime.New())
	m.SetRepoContextService(repo)

	msg := checkpointRestoreResultMsg{
		value: checkpoint.RestoreResult{
			Target:       checkpoint.Checkpoint{ID: "cp-000010"},
			Safety:       checkpoint.Checkpoint{ID: "cp-000011"},
			FilesChanged: 1,
		},
		err: errors.New("injected interrupted restore"),
	}
	_, _ = m.Update(msg)

	if repo.invalidations != 1 {
		t.Fatalf("Repo Context invalidations=%d, want 1 after restore error", repo.invalidations)
	}
	if !messagesContain(m.messages, "Restore failed") {
		t.Fatalf("missing restore failure notice: %#v", m.messages)
	}
}
