package app

import (
	"strings"
	"testing"
	"time"

	"codea/tui/internal/runtime"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

func TestTask25ViewModeCommandsChangeOnlyDerivedPresentation(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.sessionID = runtime.SessionID("session-a")
	m.input = "inspect"
	_ = m.submit()
	m.processRuntimeEvent(runtime.Event{
		Type:      eventTypeToolCalled,
		SessionID: "session-a",
		Metadata:  map[string]string{"target": "README.md"},
		Tool:      &runtime.ToolEvent{Name: "read", CallID: "call-1"},
	})

	before := m.executionTrace.Entries()

	m.input = "/view verbose"
	if cmd := m.submit(); cmd != nil {
		t.Fatal("/view must be a local UI action")
	}
	if m.viewMode != ViewVerbose {
		t.Fatalf("view mode = %q, want verbose", m.viewMode)
	}
	assertTraceUnchanged(t, before, m.executionTrace.Entries())

	m.input = "/view focus"
	_ = m.submit()
	if m.viewMode != ViewFocus {
		t.Fatalf("view mode = %q, want focus", m.viewMode)
	}
	assertTraceUnchanged(t, before, m.executionTrace.Entries())

	m.input = "/view normal"
	_ = m.submit()
	if m.viewMode != ViewNormal {
		t.Fatalf("view mode = %q, want normal", m.viewMode)
	}
	assertTraceUnchanged(t, before, m.executionTrace.Entries())
}

func TestTask25InvalidViewModeDoesNotMutateModeOrTrace(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.viewMode = ViewFocus
	before := m.executionTrace.Entries()
	m.input = "/view raw-events"

	if cmd := m.submit(); cmd != nil {
		t.Fatal("invalid /view must remain a local action")
	}
	if m.viewMode != ViewFocus {
		t.Fatalf("view mode = %q, want unchanged focus", m.viewMode)
	}
	assertTraceUnchanged(t, before, m.executionTrace.Entries())
	if len(m.messages) == 0 || !strings.Contains(m.messages[len(m.messages)-1].Content, "/view normal|verbose|focus") {
		t.Fatalf("missing explicit /view usage error: %#v", m.messages)
	}
}

func TestTask25FocusHidesRoutineTraceButKeepsApprovalBlocker(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.width, m.height = 120, 40
	m.sessionID = runtime.SessionID("session-a")
	m.input = "fix it"
	_ = m.submit()
	m.processRuntimeEvent(runtime.Event{Type: eventTypeToolCalled, SessionID: "session-a", Metadata: map[string]string{"target": "go test ./..."}, Tool: &runtime.ToolEvent{Name: "bash", CallID: "call-1"}})
	m.processRuntimeEvent(runtime.Event{Type: eventTypeApprovalRequested, SessionID: "session-a", Approval: &runtime.ApprovalRequest{ID: "approval-1", Permission: "bash", Command: "go test ./..."}})

	m.viewMode = ViewFocus
	m.markDirty()
	view := m.View()
	if strings.Contains(view, "Working") {
		t.Fatalf("focus view should hide routine Working block:\n%s", view)
	}
	if strings.Contains(view, "Tool · bash") {
		t.Fatalf("focus view should hide routine tool block:\n%s", view)
	}
	if !strings.Contains(view, "Approval") || !strings.Contains(view, "waiting") {
		t.Fatalf("focus view must keep blocking approval visible:\n%s", view)
	}
}

func TestTask25VerboseShowsAvailableTraceDetail(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.width, m.height = 120, 40
	m.sessionID = runtime.SessionID("session-a")
	m.input = "inspect"
	_ = m.submit()
	m.processRuntimeEvent(runtime.Event{Type: eventTypeToolCalled, SessionID: "session-a", Metadata: map[string]string{"target": "README.md"}, Tool: &runtime.ToolEvent{Name: "read", CallID: "call-1"}})
	m.viewMode = ViewVerbose
	m.markDirty()

	view := m.View()
	if !strings.Contains(view, "Working") || !strings.Contains(view, "Agent · general") || !strings.Contains(view, "Tool · read") {
		t.Fatalf("verbose view missing semantic trace blocks:\n%s", view)
	}
	if !strings.Contains(view, "README.md") {
		t.Fatalf("verbose view missing available structured detail:\n%s", view)
	}
}

func TestTask25FocusShowsCompactActivitySummaryWithoutDiscardingTrace(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.width, m.height = 120, 40
	m.sessionID = runtime.SessionID("session-a")
	m.input = "inspect"
	_ = m.submit()
	for i, id := range []string{"call-1", "call-2"} {
		name := "read"
		if i == 1 {
			name = "grep"
		}
		m.processRuntimeEvent(runtime.Event{Type: eventTypeToolCalled, SessionID: "session-a", Tool: &runtime.ToolEvent{Name: name, CallID: id}})
		m.processRuntimeEvent(runtime.Event{Type: eventTypeToolSuccess, SessionID: "session-a", Tool: &runtime.ToolEvent{Name: name, CallID: id}})
	}
	before := m.executionTrace.Entries()
	m.viewMode = ViewFocus
	m.markDirty()
	view := m.View()
	if !strings.Contains(view, "2 tool calls") {
		t.Fatalf("focus view missing compact trace-derived activity summary:\n%s", view)
	}
	assertTraceUnchanged(t, before, m.executionTrace.Entries())
}

func TestTask25RefreshCadenceStaysWithinBoundedDesignWindow(t *testing.T) {
	if refreshInterval < 16*time.Millisecond || refreshInterval > 33*time.Millisecond {
		t.Fatalf("refreshInterval = %s, want 16ms..33ms", refreshInterval)
	}
}

func assertTraceUnchanged(t *testing.T, before, after []ExecutionTraceEntry) {
	t.Helper()
	if len(before) != len(after) {
		t.Fatalf("trace length changed from %d to %d", len(before), len(after))
	}
	for i := range before {
		if before[i].InvocationKey != after[i].InvocationKey || before[i].Category != after[i].Category || before[i].Status != after[i].Status || before[i].Title != after[i].Title || before[i].Detail != after[i].Detail {
			t.Fatalf("trace entry %d mutated by view mode: before=%#v after=%#v", i, before[i], after[i])
		}
	}
}
