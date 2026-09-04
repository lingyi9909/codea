package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	"codea/tui/internal/checkpoint"
	"codea/tui/internal/repoctx"
	"codea/tui/internal/runtime"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

type fakeCheckpointService struct {
	creates  []checkpoint.CreateRequest
	list     []checkpoint.Checkpoint
	restore  checkpoint.RestoreResult
	createErr error
	listErr   error
	restoreErr error
	restores []string
}

func (f *fakeCheckpointService) Create(_ context.Context, req checkpoint.CreateRequest) (checkpoint.Checkpoint, error) {
	f.creates = append(f.creates, req)
	if f.createErr != nil {
		return checkpoint.Checkpoint{}, f.createErr
	}
	return checkpoint.Checkpoint{ID: "cp-000001", Kind: req.Kind, TaskID: req.TaskID, TurnID: req.TurnID, Label: req.Label}, nil
}

func (f *fakeCheckpointService) List(context.Context) ([]checkpoint.Checkpoint, error) {
	return append([]checkpoint.Checkpoint(nil), f.list...), f.listErr
}

func (f *fakeCheckpointService) Restore(_ context.Context, id string) (checkpoint.RestoreResult, error) {
	f.restores = append(f.restores, id)
	return f.restore, f.restoreErr
}

type invalidatingRepoContext struct{ invalidations int }

func (r *invalidatingRepoContext) BuildMap(context.Context, repoctx.Query) (repoctx.RepoMap, error) {
	return repoctx.RepoMap{}, nil
}
func (r *invalidatingRepoContext) Invalidate() { r.invalidations++ }

// fakeRepoContextService is declared in repo_context_test.go and is used by
// existing Task 28 tests. Task 31 extends the production interface with a
// restore invalidation boundary, so the old fixture implements the no-op form.
func (f *fakeRepoContextService) Invalidate() {}

func TestBaselineCheckpointRunsBeforeRuntimePrompt(t *testing.T) {
	fakeRuntime := fakeruntime.New()
	service := &fakeCheckpointService{}
	m := NewModel(fakeRuntime)
	m.sessionID = runtime.SessionID("active")
	m.SetCheckpointService(service)

	cmd := m.startPrompt("fix it", "fix it", "general")
	if cmd == nil {
		t.Fatal("expected async baseline command")
	}
	if len(service.creates) != 0 || len(fakeRuntime.Prompts()) != 0 {
		t.Fatal("checkpoint/runtime I/O must not run synchronously in submit")
	}
	msg := cmd()
	if len(service.creates) != 1 || service.creates[0].Kind != checkpoint.KindBaseline {
		t.Fatalf("baseline create=%+v", service.creates)
	}
	if len(fakeRuntime.Prompts()) != 0 {
		t.Fatal("Runtime prompt ran before baseline result reached Update")
	}
	_, next := m.Update(msg)
	if next == nil {
		t.Fatal("baseline result must release prompt")
	}
	_ = next()
	if len(fakeRuntime.Prompts()) != 1 {
		t.Fatalf("Runtime prompts=%d, want 1", len(fakeRuntime.Prompts()))
	}
	if m.lastBaselineCheckpoint != "cp-000001" {
		t.Fatalf("baseline id=%q", m.lastBaselineCheckpoint)
	}
}

func TestBaselineFailureIsVisibleAndPromptContinues(t *testing.T) {
	fakeRuntime := fakeruntime.New()
	service := &fakeCheckpointService{createErr: errors.New("git missing")}
	m := NewModel(fakeRuntime)
	m.sessionID = runtime.SessionID("active")
	m.SetCheckpointService(service)

	msg := m.startPrompt("fix", "fix", "general")()
	_, next := m.Update(msg)
	if next == nil {
		t.Fatal("baseline failure must not block prompt")
	}
	_ = next()
	if len(fakeRuntime.Prompts()) != 1 {
		t.Fatalf("Runtime prompts=%d", len(fakeRuntime.Prompts()))
	}
	if !messagesContain(m.messages, "Checkpoint unavailable") {
		t.Fatalf("missing degradation notice: %#v", m.messages)
	}
}

func TestFreshMutatingVerificationQueuesFinalCheckpointWithoutChangingTruth(t *testing.T) {
	service := &fakeCheckpointService{createErr: errors.New("disk full")}
	m := NewModel(fakeruntime.New())
	m.SetCheckpointService(service)
	m.activeTurnID = "turn-1"
	m.isStreaming = true
	m.taskExecution = TaskExecutionState{RootTurnID: "turn-1", MutationSeen: true, VerifyPassed: true, LastVerificationResult: "pass"}

	m.finishStepWithVerification()
	if !m.taskExecution.VerifyPassed || m.taskExecution.LastVerificationResult != "pass" {
		t.Fatalf("checkpoint scheduling rewrote verification truth: %+v", m.taskExecution)
	}
	cmd := m.takePendingCheckpointCmd()
	if cmd == nil {
		t.Fatal("fresh mutating PASS should queue final checkpoint")
	}
	msg := cmd()
	_, _ = m.Update(msg)
	if !m.taskExecution.VerifyPassed || m.taskExecution.LastVerificationResult != "pass" {
		t.Fatalf("final checkpoint failure rewrote verification truth: %+v", m.taskExecution)
	}
	if !messagesContain(m.messages, "verification remains accepted") {
		t.Fatalf("missing final failure notice: %#v", m.messages)
	}
}

func TestReadOnlyCompletionDoesNotQueueFinalCheckpoint(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.SetCheckpointService(&fakeCheckpointService{})
	m.activeTurnID = "turn-read"
	m.isStreaming = true
	m.taskExecution = TaskExecutionState{RootTurnID: "turn-read", MutationSeen: false}
	m.finishStepWithVerification()
	if cmd := m.takePendingCheckpointCmd(); cmd != nil {
		t.Fatal("read-only completion must not create final checkpoint")
	}
}

func TestRestoreBusyStatesAreBlocked(t *testing.T) {
	for _, tc := range []struct {
		name  string
		setup func(*Model)
	}{
		{name: "streaming", setup: func(m *Model) { m.isStreaming = true }},
		{name: "approval", setup: func(m *Model) { m.approvalPending = true }},
		{name: "checkpoint", setup: func(m *Model) { m.checkpointInFlight = true }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service := &fakeCheckpointService{}
			m := NewModel(fakeruntime.New())
			m.SetCheckpointService(service)
			tc.setup(m)
			if cmd := m.executeWorkspaceAction("restore", "cp-000001"); cmd != nil {
				t.Fatal("busy restore should be rejected synchronously")
			}
			if len(service.restores) != 0 {
				t.Fatalf("restore invoked while busy: %+v", service.restores)
			}
		})
	}
}

func TestRestoreSuccessInvalidatesRepoContext(t *testing.T) {
	service := &fakeCheckpointService{restore: checkpoint.RestoreResult{
		Target: checkpoint.Checkpoint{ID: "cp-000010"},
		Safety: checkpoint.Checkpoint{ID: "cp-000011"},
		FilesChanged: 3,
	}}
	repo := &invalidatingRepoContext{}
	m := NewModel(fakeruntime.New())
	m.SetCheckpointService(service)
	m.SetRepoContextService(repo)

	cmd := m.executeWorkspaceAction("restore", "cp-000010")
	if cmd == nil {
		t.Fatal("restore should run asynchronously")
	}
	msg := cmd()
	_, _ = m.Update(msg)
	if repo.invalidations != 1 {
		t.Fatalf("Repo Context invalidations=%d, want 1", repo.invalidations)
	}
	if !messagesContain(m.messages, "Restored cp-000010 · safety cp-000011 · 3 files changed") {
		t.Fatalf("missing restore result: %#v", m.messages)
	}
}

func messagesContain(messages []ChatMessage, needle string) bool {
	for _, msg := range messages {
		if strings.Contains(msg.Content, needle) {
			return true
		}
	}
	return false
}
