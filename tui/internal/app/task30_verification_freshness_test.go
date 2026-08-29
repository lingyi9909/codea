package app

import (
	"testing"

	"codea/tui/internal/runtime"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

func verificationEvent(eventType runtime.EventType, sessionID, turnID, callID, result string) runtime.Event {
	metadata := map[string]string(nil)
	if result != "" {
		metadata = map[string]string{
			"codeaVerification":        "true",
			"codeaVerificationResult":  result,
			"codeaVerificationProfile": "go",
		}
	}
	return runtime.Event{
		Type: eventType, SessionID: sessionID, MessageID: turnID,
		Tool: &runtime.ToolEvent{Name: "verify_project", CallID: callID, Metadata: metadata},
	}
}

func newVerificationModel() *Model {
	m := NewModel(fakeruntime.New())
	m.sessionID = runtime.SessionID("s1")
	m.beginPromptTrace(runtime.PromptRequest{MessageID: "turn-1", Agent: "general"})
	return m
}

func TestTaskExecutionCountsVerificationAttemptOncePerCallID(t *testing.T) {
	m := newVerificationModel()
	m.processRuntimeEvent(verificationEvent(eventTypeToolCalled, "s1", "turn-1", "verify-1", ""))
	m.processRuntimeEvent(verificationEvent(eventTypeToolCalled, "s1", "turn-1", "verify-1", ""))
	if m.taskExecution.VerifyAttempts != 1 {
		t.Fatalf("VerifyAttempts=%d, want 1", m.taskExecution.VerifyAttempts)
	}
}

func TestTaskExecutionTracksExactVerificationResult(t *testing.T) {
	for _, result := range []string{"pass", "fail", "timeout", "not_configured", "error"} {
		t.Run(result, func(t *testing.T) {
			m := newVerificationModel()
			m.processRuntimeEvent(taskEvent(eventTypeToolCalled, "s1", "turn-1", "write", nil))
			m.processRuntimeEvent(verificationEvent(eventTypeToolCalled, "s1", "turn-1", "verify-1", ""))
			m.processRuntimeEvent(verificationEvent(eventTypeToolSuccess, "s1", "turn-1", "verify-1", result))
			if m.taskExecution.VerifyResult != result {
				t.Fatalf("VerifyResult=%q, want %q", m.taskExecution.VerifyResult, result)
			}
			if got := m.taskExecution.VerifyPassed; got != (result == "pass") {
				t.Fatalf("VerifyPassed=%v for %q", got, result)
			}
		})
	}
}

func TestTaskExecutionVerificationFreshnessInvalidatesAfterMutation(t *testing.T) {
	m := newVerificationModel()
	m.processRuntimeEvent(taskEvent(eventTypeToolCalled, "s1", "turn-1", "write", nil))
	m.processRuntimeEvent(verificationEvent(eventTypeToolCalled, "s1", "turn-1", "verify-1", ""))
	m.processRuntimeEvent(verificationEvent(eventTypeToolSuccess, "s1", "turn-1", "verify-1", "pass"))
	if !m.taskExecution.VerifyPassed {
		t.Fatal("expected fresh PASS after mutation")
	}

	m.processRuntimeEvent(taskEvent(eventTypeToolCalled, "s1", "turn-1", "edit", nil))
	if m.taskExecution.VerifyPassed || m.taskExecution.VerifyResult != "" {
		t.Fatalf("new mutation must invalidate PASS: %#v", m.taskExecution)
	}

	m.processRuntimeEvent(verificationEvent(eventTypeToolCalled, "s1", "turn-1", "verify-2", ""))
	m.processRuntimeEvent(verificationEvent(eventTypeToolSuccess, "s1", "turn-1", "verify-2", "pass"))
	if !m.taskExecution.VerifyPassed || m.taskExecution.VerifyResult != "pass" {
		t.Fatalf("rerun after mutation must restore fresh PASS: %#v", m.taskExecution)
	}
}

func TestTaskExecutionVerificationBeforeMutationDoesNotSatisfyLaterMutation(t *testing.T) {
	m := newVerificationModel()
	m.processRuntimeEvent(verificationEvent(eventTypeToolCalled, "s1", "turn-1", "verify-1", ""))
	m.processRuntimeEvent(verificationEvent(eventTypeToolSuccess, "s1", "turn-1", "verify-1", "pass"))
	m.processRuntimeEvent(taskEvent(eventTypeToolCalled, "s1", "turn-1", "write", nil))
	if m.taskExecution.VerifyPassed || m.taskExecution.VerifyResult != "" {
		t.Fatalf("pre-mutation verification must be stale: %#v", m.taskExecution)
	}
}

func TestTaskExecutionIgnoresStaleVerificationSessionAndTurn(t *testing.T) {
	m := newVerificationModel()
	m.processRuntimeEvent(taskEvent(eventTypeToolCalled, "s1", "turn-1", "write", nil))
	m.processRuntimeEvent(verificationEvent(eventTypeToolCalled, "s2", "turn-1", "verify-x", ""))
	m.processRuntimeEvent(verificationEvent(eventTypeToolSuccess, "s2", "turn-1", "verify-x", "pass"))
	m.processRuntimeEvent(verificationEvent(eventTypeToolCalled, "s1", "old-turn", "verify-y", ""))
	m.processRuntimeEvent(verificationEvent(eventTypeToolSuccess, "s1", "old-turn", "verify-y", "pass"))
	if m.taskExecution.VerifyAttempts != 0 || m.taskExecution.VerifyPassed || m.taskExecution.VerifyResult != "" {
		t.Fatalf("stale verification changed state: %#v", m.taskExecution)
	}
}
