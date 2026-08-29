package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"codea/tui/internal/runtime"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

func continuationModel() (*Model, *fakeruntime.FakeRuntime) {
	fake := fakeruntime.New()
	m := NewModel(fake)
	m.sessionID = runtime.SessionID("s1")
	m.sessionModels[m.sessionID] = runtime.ModelRef{ProviderID: "private", ModelID: "coder"}
	m.beginPromptTrace(runtime.PromptRequest{MessageID: "u1", Agent: "debug"})
	m.isStreaming = true
	m.messages = append(m.messages,
		ChatMessage{Role: RoleUser, Content: "fix it", Finished: true, TurnID: "u1"},
		ChatMessage{Role: RoleAssistant, Agent: "debug", Model: "coder", TurnID: "u1"},
	)
	m.taskExecution.MutationSeen = true
	return m, fake
}

func runCmd(t *testing.T, cmd tea.Cmd) tea.Msg {
	t.Helper()
	if cmd == nil {
		t.Fatal("expected command")
	}
	return cmd()
}

func stepFinished(partID, messageID string) runtime.Event {
	return runtime.Event{Type: eventTypeStepFinished, SessionID: "s1", PartID: partID, MessageID: messageID}
}

func TestControlPromptMissingVerificationPreservesRootAgentAndModel(t *testing.T) {
	m, fake := continuationModel()
	cmd := m.handleVerificationStepFinished(stepFinished("step-1", "u1"))
	_ = runCmd(t, cmd)
	prompts := fake.Prompts()
	if len(prompts) != 1 {
		t.Fatalf("prompts=%d, want 1", len(prompts))
	}
	req := prompts[0].Request
	if req.MessageID == "u1" || req.MessageID == "" {
		t.Fatalf("control MessageID=%q, want unique non-root ID", req.MessageID)
	}
	if req.Agent != "debug" {
		t.Fatalf("Agent=%q, want root debug", req.Agent)
	}
	if req.Model == nil || req.Model.ProviderID != "private" || req.Model.ModelID != "coder" {
		t.Fatalf("Model=%#v, want selected root session model", req.Model)
	}
	if len(req.Parts) != 1 {
		t.Fatalf("parts=%d, want 1", len(req.Parts))
	}
	part, ok := req.Parts[0].(runtime.TextPart)
	if !ok || !part.Synthetic {
		t.Fatalf("part=%#v, want synthetic TextPart", req.Parts[0])
	}
	if part.Metadata["codea.kind"] != "verification-control" || part.Metadata["codea.rootTurn"] != "u1" || part.Metadata["codea.attempt"] != 1 {
		t.Fatalf("metadata=%#v", part.Metadata)
	}
	if got := m.rootTurnForMessage(req.MessageID); got != "u1" {
		t.Fatalf("control root mapping=%q, want u1", got)
	}
	if m.taskExecution.AutoContinuation != 1 {
		t.Fatalf("AutoContinuation=%d, want 1", m.taskExecution.AutoContinuation)
	}
	if len(m.messages) != 2 || m.messages[0].Role != RoleUser {
		t.Fatalf("control continuation created visible fake user message: %#v", m.messages)
	}
	if !m.isStreaming {
		t.Fatal("continuation must keep root task streaming")
	}
}

func TestControlPromptFailedVerificationUsesRepairPrompt(t *testing.T) {
	m, fake := continuationModel()
	m.taskExecution.LastVerificationResult = "fail"
	cmd := m.handleVerificationStepFinished(stepFinished("step-1", "u1"))
	_ = runCmd(t, cmd)
	part := fake.Prompts()[0].Request.Parts[0].(runtime.TextPart)
	if part.Text != failedVerificationControlPrompt {
		t.Fatalf("repair prompt mismatch: %q", part.Text)
	}
}

func TestRepairLoopIsBoundedToTwoAndReplayDoesNotDuplicate(t *testing.T) {
	m, fake := continuationModel()
	first := stepFinished("step-1", "u1")
	cmd1 := m.handleVerificationStepFinished(first)
	_ = runCmd(t, cmd1)
	if replay := m.handleVerificationStepFinished(first); replay != nil {
		t.Fatal("replayed same step.finished scheduled duplicate continuation")
	}
	if got := len(fake.Prompts()); got != 1 {
		t.Fatalf("prompts after replay=%d, want 1", got)
	}

	cmd2 := m.handleVerificationStepFinished(stepFinished("step-2", fake.Prompts()[0].Request.MessageID))
	_ = runCmd(t, cmd2)
	if m.taskExecution.AutoContinuation != 2 || len(fake.Prompts()) != 2 {
		t.Fatalf("after second continuation state=%#v prompts=%d", m.taskExecution, len(fake.Prompts()))
	}

	cmd3 := m.handleVerificationStepFinished(stepFinished("step-3", fake.Prompts()[1].Request.MessageID))
	if cmd3 != nil {
		t.Fatal("third automatic continuation must not exist")
	}
	working, _ := m.executionTrace.Entry("turn:u1:working")
	if working.Status != TraceUnverified || m.isStreaming {
		t.Fatalf("bounded exhaustion must terminal Unverified: status=%q streaming=%v", working.Status, m.isStreaming)
	}
	if len(fake.Prompts()) != 2 {
		t.Fatalf("prompts=%d, want bounded 2", len(fake.Prompts()))
	}
}

func TestVerificationPassOnContinuationCompletesVerified(t *testing.T) {
	m, fake := continuationModel()
	cmd := m.handleVerificationStepFinished(stepFinished("step-1", "u1"))
	_ = runCmd(t, cmd)
	controlID := fake.Prompts()[0].Request.MessageID
	m.taskExecution.VerifyPassed = true
	m.taskExecution.LastVerificationResult = "pass"
	m.taskExecution.LastVerificationProfile = "go"
	if next := m.handleVerificationStepFinished(stepFinished("step-2", controlID)); next != nil {
		t.Fatal("fresh PASS must not schedule another continuation")
	}
	working, _ := m.executionTrace.Entry("turn:u1:working")
	if working.Status != TraceSuccess || m.isStreaming {
		t.Fatalf("PASS continuation did not complete: status=%q streaming=%v", working.Status, m.isStreaming)
	}
}

func TestReadOnlyTaskNeverAutoContinues(t *testing.T) {
	m, fake := continuationModel()
	m.taskExecution.MutationSeen = false
	if cmd := m.handleVerificationStepFinished(stepFinished("step-1", "u1")); cmd != nil {
		t.Fatal("read-only task scheduled verification continuation")
	}
	if len(fake.Prompts()) != 0 {
		t.Fatal("read-only task sent prompt")
	}
	working, _ := m.executionTrace.Entry("turn:u1:working")
	if working.Status != TraceSuccess || m.isStreaming {
		t.Fatalf("read-only completion changed: status=%q streaming=%v", working.Status, m.isStreaming)
	}
}
