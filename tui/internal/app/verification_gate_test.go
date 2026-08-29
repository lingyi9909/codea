package app

import (
	"strings"
	"testing"

	"codea/tui/internal/runtime"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

func TestVerificationGateTruthTable(t *testing.T) {
	cases := []struct {
		name string
		state TaskExecutionState
		want VerificationDecision
	}{
		{"read-only", TaskExecutionState{}, VerifyNotRequired},
		{"mutation-pass", TaskExecutionState{MutationSeen: true, VerifyPassed: true, LastVerificationResult: "pass"}, VerifyAccepted},
		{"mutation-missing", TaskExecutionState{MutationSeen: true}, VerifyMissing},
		{"mutation-fail", TaskExecutionState{MutationSeen: true, LastVerificationResult: "fail"}, VerifyFailed},
		{"mutation-not-configured", TaskExecutionState{MutationSeen: true, LastVerificationResult: "not_configured"}, VerifyFailed},
		{"mutation-timeout", TaskExecutionState{MutationSeen: true, LastVerificationResult: "timeout"}, VerifyFailed},
		{"mutation-error", TaskExecutionState{MutationSeen: true, LastVerificationResult: "error"}, VerifyFailed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := verificationDecision(tc.state); got != tc.want {
				t.Fatalf("verificationDecision(%#v)=%q, want %q", tc.state, got, tc.want)
			}
		})
	}
}

func completionModel(state TaskExecutionState) *Model {
	m := NewModel(fakeruntime.New())
	m.sessionID = runtime.SessionID("s1")
	m.beginPromptTrace(runtime.PromptRequest{MessageID: "turn-1", Agent: "general"})
	state.RootTurnID = "turn-1"
	state.messageRoots = map[string]string{"turn-1": "turn-1"}
	m.taskExecution = state
	m.isStreaming = true
	m.messages = append(m.messages, ChatMessage{Role: RoleAssistant, Content: "done", Finished: false})
	return m
}

func TestMutationWithoutFreshVerificationCannotFinalizeSuccess(t *testing.T) {
	m := completionModel(TaskExecutionState{MutationSeen: true})
	m.processRuntimeEvent(runtime.Event{Type: eventTypeStepFinished, SessionID: "s1", MessageID: "turn-1"})
	working, ok := m.executionTrace.Entry("turn:turn-1:working")
	if !ok {
		t.Fatal("missing working trace")
	}
	if working.Status != TraceUnverified {
		t.Fatalf("working status=%q, want %q", working.Status, TraceUnverified)
	}
	if m.isStreaming {
		t.Fatal("Step 5 terminal unverified path should finish streaming; Step 6 will interpose continuation")
	}
	if summary := m.renderCompletionSummary(); !strings.Contains(summary, "⚠ Unverified") || strings.Contains(summary, "✓ Completed") {
		t.Fatalf("summary=%q, want truthful unverified status", summary)
	}
}

func TestMutationWithFreshPassFinalizesVerified(t *testing.T) {
	m := completionModel(TaskExecutionState{MutationSeen: true, VerifyPassed: true, LastVerificationResult: "pass", LastVerificationProfile: "go"})
	m.processRuntimeEvent(runtime.Event{Type: eventTypeStepFinished, SessionID: "s1", MessageID: "turn-1"})
	working, _ := m.executionTrace.Entry("turn:turn-1:working")
	if working.Status != TraceSuccess {
		t.Fatalf("working status=%q, want success", working.Status)
	}
	if summary := m.renderCompletionSummary(); !strings.Contains(summary, "✓ Verified") || strings.Contains(summary, "✓ Completed") {
		t.Fatalf("summary=%q, want verified status", summary)
	}
}

func TestReadOnlyCompletionRemainsCompleted(t *testing.T) {
	m := completionModel(TaskExecutionState{})
	m.processRuntimeEvent(runtime.Event{Type: eventTypeStepFinished, SessionID: "s1", MessageID: "turn-1"})
	working, _ := m.executionTrace.Entry("turn:turn-1:working")
	if working.Status != TraceSuccess {
		t.Fatalf("working status=%q, want success", working.Status)
	}
	if summary := m.renderCompletionSummary(); !strings.Contains(summary, "✓ Completed") {
		t.Fatalf("summary=%q, want read-only Completed", summary)
	}
}

func TestAssistantProseCannotOverrideVerificationGate(t *testing.T) {
	m := completionModel(TaskExecutionState{MutationSeen: true, LastVerificationResult: "fail"})
	m.messages[len(m.messages)-1].Content = "All tests passed and the task is complete."
	m.processRuntimeEvent(runtime.Event{Type: eventTypeStepFinished, SessionID: "s1", MessageID: "turn-1"})
	working, _ := m.executionTrace.Entry("turn:turn-1:working")
	if working.Status != TraceUnverified {
		t.Fatalf("assistant prose overrode gate: status=%q", working.Status)
	}
	if summary := m.renderCompletionSummary(); !strings.Contains(summary, "⚠ Unverified") {
		t.Fatalf("summary=%q, want Unverified", summary)
	}
}
