package app

import (
	"testing"

	"codea/tui/internal/runtime"
)

func task30VerificationEvent(eventType runtime.EventType, callID, result string) runtime.Event {
	metadata := map[string]string{}
	if eventType == eventTypeToolSuccess {
		metadata = map[string]string{
			"codeaVerification":        "true",
			"codeaVerificationResult":  result,
			"codeaVerificationProfile": "go",
		}
	}
	return runtime.Event{
		Type:      eventType,
		SessionID: "s1",
		MessageID: "u1",
		Tool: &runtime.ToolEvent{
			Name:     "verify_project",
			CallID:   callID,
			Metadata: metadata,
		},
	}
}

func task30ObserveVerification(m *Model, callID, result string) {
	m.processRuntimeEvent(task30VerificationEvent(eventTypeToolCalled, callID, ""))
	m.processRuntimeEvent(task30VerificationEvent(eventTypeToolSuccess, callID, result))
}

func TestTask30ReadOnlyNoVerifyCompleted(t *testing.T) {
	m, fake := continuationModel()
	m.taskExecution.MutationSeen = false
	if cmd := m.handleVerificationStepFinished(stepFinished("step-read", "u1")); cmd != nil {
		t.Fatal("read-only task scheduled verification continuation")
	}
	working, _ := m.executionTrace.Entry("turn:u1:working")
	if working.Status != TraceSuccess || m.isStreaming {
		t.Fatalf("read-only completion status=%q streaming=%v", working.Status, m.isStreaming)
	}
	if len(fake.Prompts()) != 0 {
		t.Fatalf("read-only task emitted %d control prompts", len(fake.Prompts()))
	}
}

func TestTask30MutationNoVerifyAutoVerifyContinuation(t *testing.T) {
	m, fake := continuationModel()
	cmd := m.handleVerificationStepFinished(stepFinished("step-missing", "u1"))
	_ = runCmd(t, cmd)
	if len(fake.Prompts()) != 1 || m.taskExecution.AutoContinuation != 1 {
		t.Fatalf("prompts=%d continuation=%d", len(fake.Prompts()), m.taskExecution.AutoContinuation)
	}
	if !m.isStreaming {
		t.Fatal("root task must remain streaming during automatic verification continuation")
	}
}

func TestTask30MutationVerifyPassVerified(t *testing.T) {
	m, _ := continuationModel()
	task30ObserveVerification(m, "verify-pass", "pass")
	if cmd := m.handleVerificationStepFinished(stepFinished("step-pass", "u1")); cmd != nil {
		t.Fatal("fresh verification PASS scheduled another continuation")
	}
	working, _ := m.executionTrace.Entry("turn:u1:working")
	if working.Status != TraceSuccess || m.isStreaming {
		t.Fatalf("verified completion status=%q streaming=%v", working.Status, m.isStreaming)
	}
}

func TestTask30MutationVerifyFailThenPassVerifiedAfterRepair(t *testing.T) {
	m, fake := continuationModel()
	task30ObserveVerification(m, "verify-fail-1", "fail")
	_ = runCmd(t, m.handleVerificationStepFinished(stepFinished("step-fail-1", "u1")))
	if len(fake.Prompts()) != 1 {
		t.Fatalf("repair prompt count=%d, want 1", len(fake.Prompts()))
	}
	task30ObserveVerification(m, "verify-pass-2", "pass")
	if cmd := m.handleVerificationStepFinished(stepFinished("step-pass-2", "u1")); cmd != nil {
		t.Fatal("PASS after bounded repair scheduled another continuation")
	}
	working, _ := m.executionTrace.Entry("turn:u1:working")
	if working.Status != TraceSuccess || m.taskExecution.AutoContinuation != 1 {
		t.Fatalf("status=%q continuation=%d", working.Status, m.taskExecution.AutoContinuation)
	}
}

func TestTask30MutationVerifyFailX3BoundedStopUnverified(t *testing.T) {
	m, fake := continuationModel()
	for attempt := 1; attempt <= 3; attempt++ {
		callID := "verify-fail-" + string(rune('0'+attempt))
		task30ObserveVerification(m, callID, "fail")
		cmd := m.handleVerificationStepFinished(stepFinished("step-fail-"+string(rune('0'+attempt)), "u1"))
		if attempt <= 2 {
			_ = runCmd(t, cmd)
		} else if cmd != nil {
			t.Fatal("third failed verification exceeded bounded continuation budget")
		}
	}
	working, _ := m.executionTrace.Entry("turn:u1:working")
	if working.Status != TraceUnverified || m.isStreaming {
		t.Fatalf("bounded stop status=%q streaming=%v", working.Status, m.isStreaming)
	}
	if len(fake.Prompts()) != 2 || m.taskExecution.AutoContinuation != 2 {
		t.Fatalf("prompts=%d continuation=%d", len(fake.Prompts()), m.taskExecution.AutoContinuation)
	}
}

func TestTask30VerifyPassThenMutationPassInvalidated(t *testing.T) {
	m, _ := continuationModel()
	task30ObserveVerification(m, "verify-before-mutation", "pass")
	if !m.taskExecution.VerifyPassed {
		t.Fatal("expected fresh verification PASS before later mutation")
	}
	m.processRuntimeEvent(runtime.Event{
		Type:      eventTypeToolCalled,
		SessionID: "s1",
		MessageID: "u1",
		Tool:      &runtime.ToolEvent{Name: "write", CallID: "write-after-pass"},
	})
	if m.taskExecution.VerifyPassed || m.taskExecution.LastVerificationResult != "" {
		t.Fatalf("later mutation did not invalidate PASS: %#v", m.taskExecution)
	}
	if cmd := m.handleVerificationStepFinished(stepFinished("step-after-mutation", "u1")); cmd == nil {
		t.Fatal("stale PASS incorrectly allowed terminal success instead of fresh verification continuation")
	}
}
