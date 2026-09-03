package app

import (
	"testing"

	"codea/tui/internal/runtime"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

func vendorAssistantMessage(sessionID, messageID, parentID string) runtime.Event {
	return runtime.Event{
		Type: "message.updated", SessionID: sessionID, MessageID: messageID,
		ParentMessageID: parentID, MessageRole: "assistant",
	}
}

func TestTaskExecutionUsesAssistantParentIdentityForRootTurn(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.sessionID = runtime.SessionID("s1")

	m.beginPromptTrace(runtime.PromptRequest{MessageID: "U0", Agent: "general"})
	m.processRuntimeEvent(vendorAssistantMessage("s1", "A0", "U0"))

	m.beginPromptTrace(runtime.PromptRequest{MessageID: "U1", Agent: "general"})
	m.processRuntimeEvent(vendorAssistantMessage("s1", "A1", "U1"))
	m.processRuntimeEvent(taskEvent(eventTypeToolSuccess, "s1", "A1", "task_plan", map[string]string{
		"codeaTaskPlan": "true", "codeaPlanTotal": "3", "codeaPlanCompleted": "0", "codeaPlanActive": "inspect",
	}))
	m.processRuntimeEvent(taskEvent(eventTypeToolCalled, "s1", "A1", "write", nil))

	if m.taskExecution.RootTurnID != "U1" || !m.taskExecution.PlanSeen || !m.taskExecution.MutationSeen {
		t.Fatalf("A1(parent=U1) did not update U1 task state: %#v", m.taskExecution)
	}
	if m.taskExecution.ActiveStep != "inspect" {
		t.Fatalf("active step=%q want inspect", m.taskExecution.ActiveStep)
	}

	m.processRuntimeEvent(taskEvent(eventTypeToolSuccess, "s1", "A0", "task_step", map[string]string{
		"codeaTaskPlan": "true", "codeaPlanTotal": "7", "codeaPlanCompleted": "6", "codeaPlanActive": "stale",
	}))
	if m.taskExecution.TotalSteps != 3 || m.taskExecution.CompletedSteps != 0 || m.taskExecution.ActiveStep != "inspect" {
		t.Fatalf("late A0(parent=U0) contaminated U1: %#v", m.taskExecution)
	}
}

func TestTask30StaleRootStepFinishedIgnored(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.sessionID = runtime.SessionID("s1")
	m.sessionModels[m.sessionID] = runtime.ModelRef{ProviderID: "private", ModelID: "coder"}

	m.beginPromptTrace(runtime.PromptRequest{MessageID: "U0", Agent: "debug"})
	m.processRuntimeEvent(vendorAssistantMessage("s1", "A0", "U0"))
	m.beginPromptTrace(runtime.PromptRequest{MessageID: "U1", Agent: "debug"})
	m.processRuntimeEvent(vendorAssistantMessage("s1", "A1", "U1"))
	m.processRuntimeEvent(taskEvent(eventTypeToolCalled, "s1", "A1", "write", nil))
	m.isStreaming = true
	m.processRuntimeEvent(runtime.Event{Type: "reasoning.delta", SessionID: "s1", MessageID: "A1", Content: "current-root-reasoning"})

	if !m.reasoningActive || !m.proc.Snapshot().HasActive() {
		t.Fatal("precondition: current root reasoning must be active")
	}
	if dirty := m.processRuntimeEvent(stepFinished("late-A0", "A0")); dirty {
		t.Fatal("stale previous-root step.finished must be ignored")
	}
	if !m.reasoningActive || !m.proc.Snapshot().HasActive() {
		t.Fatal("stale previous-root step.finished flushed current-root reasoning")
	}
	if m.taskExecution.AutoContinuation != 0 || m.pendingVerificationPrompt != nil {
		t.Fatalf("stale step drove verification loop: continuation=%d pending=%#v", m.taskExecution.AutoContinuation, m.pendingVerificationPrompt)
	}
	if !m.isStreaming {
		t.Fatal("stale step finished the active root task")
	}
	if working, ok := m.executionTrace.Entry("turn:U1:working"); !ok || working.Status != TraceRunning {
		t.Fatalf("active root working trace changed after stale step: ok=%v status=%q", ok, working.Status)
	}
	t.Log("STALE_ROOT_STEP_FINISH_IGNORED PASS")
}

func TestTask30ActiveRootStepFinishedControlsVerification(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.sessionID = runtime.SessionID("s1")
	m.sessionModels[m.sessionID] = runtime.ModelRef{ProviderID: "private", ModelID: "coder"}
	m.beginPromptTrace(runtime.PromptRequest{MessageID: "U1", Agent: "debug"})
	m.processRuntimeEvent(vendorAssistantMessage("s1", "A1", "U1"))
	m.processRuntimeEvent(taskEvent(eventTypeToolCalled, "s1", "A1", "write", nil))
	m.isStreaming = true
	m.processRuntimeEvent(runtime.Event{Type: "reasoning.delta", SessionID: "s1", MessageID: "A1", Content: "active-root-reasoning"})

	if dirty := m.processRuntimeEvent(stepFinished("active-A1", "A1")); !dirty {
		t.Fatal("active-root step.finished must retain normal processing")
	}
	if m.reasoningActive || m.proc.Snapshot().HasActive() {
		t.Fatal("active-root step.finished did not flush reasoning")
	}
	if m.taskExecution.AutoContinuation != 1 || m.pendingVerificationPrompt == nil {
		t.Fatalf("active-root step did not drive verification: continuation=%d pending=%#v", m.taskExecution.AutoContinuation, m.pendingVerificationPrompt)
	}
	if !m.isStreaming {
		t.Fatal("automatic verification continuation must keep root task streaming")
	}
	t.Log("ACTIVE_ROOT_STEP_FINISH_CONTROLS_VERIFICATION PASS")
}
