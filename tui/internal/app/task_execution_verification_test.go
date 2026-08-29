package app

import (
	"testing"

	"codea/tui/internal/runtime"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

func verificationEvent(eventType runtime.EventType, sessionID, turnID, callID, result string) runtime.Event {
	metadata := map[string]string{}
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

func TestVerificationAttemptCountsOncePerCallIDAndPassIsMachineOwned(t *testing.T) {
	m := newVerificationModel()
	m.processRuntimeEvent(verificationEvent(eventTypeToolCalled, "s1", "turn-1", "verify-1", ""))
	m.processRuntimeEvent(verificationEvent(eventTypeToolCalled, "s1", "turn-1", "verify-1", ""))
	if m.taskExecution.VerifyAttempts != 1 {
		t.Fatalf("VerifyAttempts=%d, want 1", m.taskExecution.VerifyAttempts)
	}
	m.processRuntimeEvent(verificationEvent(eventTypeToolSuccess, "s1", "turn-1", "verify-1", "pass"))
	if !m.taskExecution.VerifyPassed || m.taskExecution.LastVerificationResult != "pass" {
		t.Fatalf("verification state=%#v", m.taskExecution)
	}
}

func TestVerificationFailureCategoriesNeverPass(t *testing.T) {
	for _, result := range []string{"fail", "timeout", "not_configured", "error"} {
		t.Run(result, func(t *testing.T) {
			m := newVerificationModel()
			m.processRuntimeEvent(verificationEvent(eventTypeToolCalled, "s1", "turn-1", "verify-1", ""))
			m.processRuntimeEvent(verificationEvent(eventTypeToolSuccess, "s1", "turn-1", "verify-1", result))
			if m.taskExecution.VerifyPassed {
				t.Fatalf("%s must not set VerifyPassed", result)
			}
			if m.taskExecution.LastVerificationResult != result {
				t.Fatalf("LastVerificationResult=%q, want %q", m.taskExecution.LastVerificationResult, result)
			}
		})
	}
}

func TestVerificationFreshnessIsInvalidatedByLaterMutation(t *testing.T) {
	m := newVerificationModel()
	m.processRuntimeEvent(taskEvent(eventTypeToolCalled, "s1", "turn-1", "write", nil))
	m.processRuntimeEvent(verificationEvent(eventTypeToolCalled, "s1", "turn-1", "verify-1", ""))
	m.processRuntimeEvent(verificationEvent(eventTypeToolSuccess, "s1", "turn-1", "verify-1", "pass"))
	if !m.taskExecution.VerifyPassed {
		t.Fatal("fresh PASS should satisfy current mutation")
	}
	m.processRuntimeEvent(taskEvent(eventTypeToolCalled, "s1", "turn-1", "edit", nil))
	if m.taskExecution.VerifyPassed {
		t.Fatal("mutation after PASS must invalidate verification freshness")
	}
	if m.taskExecution.LastVerificationResult != "" {
		t.Fatalf("stale result must be cleared after mutation, got %q", m.taskExecution.LastVerificationResult)
	}
}

func TestVerificationBeforeMutationDoesNotSatisfyLaterMutation(t *testing.T) {
	m := newVerificationModel()
	m.processRuntimeEvent(verificationEvent(eventTypeToolCalled, "s1", "turn-1", "verify-1", ""))
	m.processRuntimeEvent(verificationEvent(eventTypeToolSuccess, "s1", "turn-1", "verify-1", "pass"))
	m.processRuntimeEvent(taskEvent(eventTypeToolCalled, "s1", "turn-1", "write", nil))
	if !m.taskExecution.MutationSeen || m.taskExecution.VerifyPassed {
		t.Fatalf("pre-mutation PASS must be stale: %#v", m.taskExecution)
	}
}

func TestVerificationIgnoresStaleSessionAndRootEvents(t *testing.T) {
	m := newVerificationModel()
	m.processRuntimeEvent(verificationEvent(eventTypeToolCalled, "s2", "turn-1", "verify-foreign", ""))
	m.processRuntimeEvent(verificationEvent(eventTypeToolSuccess, "s2", "turn-1", "verify-foreign", "pass"))
	m.processRuntimeEvent(verificationEvent(eventTypeToolCalled, "s1", "turn-old", "verify-old", ""))
	m.processRuntimeEvent(verificationEvent(eventTypeToolSuccess, "s1", "turn-old", "verify-old", "pass"))
	if m.taskExecution.VerifyAttempts != 0 || m.taskExecution.VerifyPassed || m.taskExecution.LastVerificationResult != "" {
		t.Fatalf("stale verification changed current state: %#v", m.taskExecution)
	}
}
