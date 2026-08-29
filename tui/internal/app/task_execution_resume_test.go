package app

import (
	"testing"

	"codea/tui/internal/runtime"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

func TestResumeSessionMakesPreviousTaskExecutionInactive(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.sessionID = runtime.SessionID("s1")
	m.activeTurnID = "turn-old"
	m.taskExecution = TaskExecutionState{
		RootTurnID: "turn-old", PlanSeen: true, ActiveStep: "change",
		CompletedSteps: 1, TotalSteps: 3, MutationSeen: true,
	}

	m.resumeSession(runtime.SessionID("s2"), nil)

	if m.activeTurnID != "" {
		t.Fatalf("activeTurnID after resume=%q, want empty", m.activeTurnID)
	}
	if got := m.renderTaskProgress(); got != "" {
		t.Fatalf("stale plan rendered after resume: %q", got)
	}
}

func TestOldSessionPlanEventIgnoredAfterResume(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.sessionID = runtime.SessionID("s1")
	m.activeTurnID = "turn-old"
	m.taskExecution = TaskExecutionState{RootTurnID: "turn-old", PlanSeen: true, TotalSteps: 3}
	m.resumeSession(runtime.SessionID("s2"), nil)
	m.beginPromptTrace(runtime.PromptRequest{MessageID: "turn-new", Agent: "general"})

	m.processRuntimeEvent(taskEvent(eventTypeToolSuccess, "s1", "turn-old", "task_step", map[string]string{
		"codeaTaskPlan": "true", "codeaPlanTotal": "3", "codeaPlanCompleted": "2", "codeaPlanActive": "verify",
	}))
	if m.taskExecution.PlanSeen || m.taskExecution.CompletedSteps != 0 || m.taskExecution.ActiveStep != "" {
		t.Fatalf("old-session plan event leaked after resume: %#v", m.taskExecution)
	}
}
