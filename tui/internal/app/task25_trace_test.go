package app

import (
	"testing"

	"codea/tui/internal/runtime"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

func TestTask25PromptCreatesWorkingAndAgentTrace(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.sessionID = runtime.SessionID("session-a")
	m.currentAgent = "debug"
	m.input = "定位这个异常"

	if cmd := m.submit(); cmd == nil {
		t.Fatal("prompt command = nil")
	}

	working, ok := m.executionTrace.Entry("turn:msg-0:working")
	if !ok {
		t.Fatal("missing Working trace for prompt turn")
	}
	if working.Category != TraceWorking || working.Status != TraceRunning {
		t.Fatalf("working trace = %#v, want working/running", working)
	}

	agent, ok := m.executionTrace.Entry("turn:msg-0:agent")
	if !ok {
		t.Fatal("missing Agent trace for prompt turn")
	}
	if agent.Category != TraceAgent || agent.Title != "debug" || agent.Status != TraceRunning {
		t.Fatalf("agent trace = %#v, want debug/running", agent)
	}
}

func TestTask25StableToolInvocationDeduplicatesByCallID(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.sessionID = runtime.SessionID("session-a")
	m.currentAgent = "general"
	m.input = "inspect"
	_ = m.submit()

	called := runtime.Event{
		Type:      eventTypeToolCalled,
		SessionID: "session-a",
		Tool:      &runtime.ToolEvent{Name: "read", CallID: "call-1"},
	}
	m.processRuntimeEvent(called)
	m.processRuntimeEvent(called)

	if got := traceCategoryCount(m.executionTrace.Entries(), TraceTool); got != 1 {
		t.Fatalf("tool trace count = %d, want 1 for repeated callID", got)
	}
	tool, ok := m.executionTrace.Entry("tool:call-1")
	if !ok {
		t.Fatal("missing stable tool invocation key tool:call-1")
	}
	if tool.Title != "read" || tool.Status != TraceRunning {
		t.Fatalf("tool trace = %#v, want read/running", tool)
	}

	m.processRuntimeEvent(runtime.Event{
		Type:      eventTypeToolSuccess,
		SessionID: "session-a",
		Tool:      &runtime.ToolEvent{Name: "read", CallID: "call-1"},
	})
	tool, _ = m.executionTrace.Entry("tool:call-1")
	if tool.Status != TraceSuccess || tool.FinishedAt.IsZero() {
		t.Fatalf("tool trace after success = %#v, want success with finishedAt", tool)
	}
}

func TestTask25SameToolNameWithDifferentCallIDsRemainDistinct(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.sessionID = runtime.SessionID("session-a")

	m.processRuntimeEvent(runtime.Event{Type: eventTypeToolCalled, SessionID: "session-a", Tool: &runtime.ToolEvent{Name: "read", CallID: "call-1"}})
	m.processRuntimeEvent(runtime.Event{Type: eventTypeToolCalled, SessionID: "session-a", Tool: &runtime.ToolEvent{Name: "read", CallID: "call-2"}})

	if got := traceCategoryCount(m.executionTrace.Entries(), TraceTool); got != 2 {
		t.Fatalf("tool trace count = %d, want 2 distinct callIDs", got)
	}
	if _, ok := m.executionTrace.Entry("tool:call-1"); !ok {
		t.Fatal("missing call-1")
	}
	if _, ok := m.executionTrace.Entry("tool:call-2"); !ok {
		t.Fatal("missing call-2")
	}
}

func TestTask25ApprovalMakesWorkingTruthfullyWaiting(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.sessionID = runtime.SessionID("session-a")
	m.input = "fix it"
	_ = m.submit()

	m.processRuntimeEvent(runtime.Event{
		Type:      eventTypeApprovalRequested,
		SessionID: "session-a",
		Approval: &runtime.ApprovalRequest{
			ID:         "approval-1",
			Permission: "bash",
			Command:    "go test ./...",
		},
	})

	working, _ := m.executionTrace.Entry("turn:msg-0:working")
	if working.Status != TraceWaiting {
		t.Fatalf("working status = %q, want waiting while approval blocks", working.Status)
	}
	approval, ok := m.executionTrace.Entry("approval:approval-1")
	if !ok {
		t.Fatal("missing approval trace")
	}
	if approval.Category != TraceApproval || approval.Status != TraceWaiting {
		t.Fatalf("approval trace = %#v, want approval/waiting", approval)
	}
	if approval.Title != "bash" {
		t.Fatalf("approval title = %q, want structured permission bash", approval.Title)
	}
}

func TestTask25StepFinishClosesActiveWorkingTrace(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.sessionID = runtime.SessionID("session-a")
	m.input = "explain"
	_ = m.submit()

	m.processRuntimeEvent(runtime.Event{Type: eventTypeStepFinished, SessionID: "session-a"})

	working, _ := m.executionTrace.Entry("turn:msg-0:working")
	if working.Status != TraceSuccess || working.FinishedAt.IsZero() {
		t.Fatalf("working after step finish = %#v, want success with finishedAt", working)
	}
}

func TestTask25MissingStructuredMetadataNeverFabricatesSkillPluginOrSubagent(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.sessionID = runtime.SessionID("session-a")
	m.processRuntimeEvent(runtime.Event{
		Type:      eventTypeToolCalled,
		SessionID: "session-a",
		Tool:      &runtime.ToolEvent{Name: "skill-looking-tool-name", CallID: "call-1"},
	})

	entries := m.executionTrace.Entries()
	if traceCategoryCount(entries, TraceSkill) != 0 {
		t.Fatal("Skill trace must not be inferred without structured metadata")
	}
	if traceCategoryCount(entries, TracePlugin) != 0 {
		t.Fatal("Plugin trace must not be inferred without structured metadata")
	}
	if traceCategoryCount(entries, TraceSubagent) != 0 {
		t.Fatal("Subagent trace must not be inferred without structured metadata")
	}
}

func traceCategoryCount(entries []ExecutionTraceEntry, category TraceCategory) int {
	count := 0
	for _, entry := range entries {
		if entry.Category == category {
			count++
		}
	}
	return count
}
