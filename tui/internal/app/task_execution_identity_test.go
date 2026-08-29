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
