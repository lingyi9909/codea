package app

import (
	"testing"

	"codea/tui/internal/runtime"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

func taskEvent(eventType runtime.EventType, sessionID, turnID, tool string, metadata map[string]string) runtime.Event {
	return runtime.Event{
		Type: eventType, SessionID: sessionID, MessageID: turnID,
		Tool: &runtime.ToolEvent{Name: tool, CallID: tool + "-1", Metadata: metadata},
	}
}

func TestTaskExecutionTracksPlanStepAndMutationFromMachineEvents(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.sessionID = runtime.SessionID("s1")
	m.beginPromptTrace(runtime.PromptRequest{MessageID: "turn-1", Agent: "general"})

	m.processRuntimeEvent(taskEvent(eventTypeToolSuccess, "s1", "turn-1", "task_plan", map[string]string{
		"codeaTaskPlan": "true", "codeaPlanTotal": "3", "codeaPlanCompleted": "0", "codeaPlanActive": "",
	}))
	if !m.taskExecution.PlanSeen || m.taskExecution.TotalSteps != 3 || m.taskExecution.CompletedSteps != 0 {
		t.Fatalf("after task_plan: %#v", m.taskExecution)
	}

	m.processRuntimeEvent(taskEvent(eventTypeToolSuccess, "s1", "turn-1", "task_step", map[string]string{
		"codeaTaskPlan": "true", "codeaPlanTotal": "3", "codeaPlanCompleted": "1", "codeaPlanActive": "change",
	}))
	if m.taskExecution.ActiveStep != "change" || m.taskExecution.CompletedSteps != 1 {
		t.Fatalf("after task_step: %#v", m.taskExecution)
	}

	m.processRuntimeEvent(taskEvent(eventTypeToolCalled, "s1", "turn-1", "read", nil))
	if m.taskExecution.MutationSeen {
		t.Fatal("read must not mark mutation")
	}
	m.processRuntimeEvent(taskEvent(eventTypeToolCalled, "s1", "turn-1", "write", nil))
	if !m.taskExecution.MutationSeen {
		t.Fatal("write must mark mutation")
	}
}

func TestTaskExecutionResetsPerRootTurnAndIgnoresStaleEvents(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.sessionID = runtime.SessionID("s1")
	m.beginPromptTrace(runtime.PromptRequest{MessageID: "turn-1", Agent: "general"})
	m.processRuntimeEvent(taskEvent(eventTypeToolSuccess, "s1", "turn-1", "task_plan", map[string]string{
		"codeaTaskPlan": "true", "codeaPlanTotal": "3", "codeaPlanCompleted": "1", "codeaPlanActive": "change",
	}))

	m.beginPromptTrace(runtime.PromptRequest{MessageID: "turn-2", Agent: "general"})
	if m.taskExecution.RootTurnID != "turn-2" || m.taskExecution.PlanSeen || m.taskExecution.MutationSeen {
		t.Fatalf("new turn did not reset task state: %#v", m.taskExecution)
	}

	m.processRuntimeEvent(taskEvent(eventTypeToolSuccess, "s1", "turn-1", "task_step", map[string]string{
		"codeaTaskPlan": "true", "codeaPlanTotal": "3", "codeaPlanCompleted": "2", "codeaPlanActive": "verify",
	}))
	m.processRuntimeEvent(taskEvent(eventTypeToolSuccess, "s2", "turn-2", "task_step", map[string]string{
		"codeaTaskPlan": "true", "codeaPlanTotal": "7", "codeaPlanCompleted": "6", "codeaPlanActive": "wrong",
	}))
	if m.taskExecution.PlanSeen || m.taskExecution.TotalSteps != 0 || m.taskExecution.ActiveStep != "" {
		t.Fatalf("stale event changed task state: %#v", m.taskExecution)
	}
}

func TestTaskExecutionProgressRenderingIsMinimal(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.taskExecution = TaskExecutionState{RootTurnID: "turn-1", PlanSeen: true, ActiveStep: "step-3", CompletedSteps: 2, TotalSteps: 5}
	m.viewMode = ViewNormal
	if got := m.renderTaskProgress(); got != "Plan · 2/5 steps · step-3" {
		t.Fatalf("normal progress=%q", got)
	}
	m.viewMode = ViewFocus
	if got := m.renderTaskProgress(); got != "Plan · 2/5" {
		t.Fatalf("focus progress=%q", got)
	}
}
