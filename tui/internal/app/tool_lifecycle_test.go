package app

import (
	"testing"

	"codea/tui/internal/opencode"
	"codea/tui/internal/runtime"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

// toolLifecycleJSON builds the real locked-OpenCode /global/event envelope for a
// tool part update: message.part.updated with part.type=tool and a state.status,
// matching what the real runtime emits for one tool call (pending → running →
// completed | error all share one callID).
func toolLifecycleJSON(sessionID, callID, tool, status string) []byte {
	return []byte(`{"directory":"/tmp","payload":{"type":"message.part.updated","properties":{"sessionID":"` + sessionID + `","part":{"id":"prt_1","messageID":"m1","sessionID":"` + sessionID + `","type":"tool","tool":"` + tool + `","callID":"` + callID + `","state":{"status":"` + status + `"}}}}}`)
}

// feedToolEvent maps a real OpenCode-style JSON event and consumes it through
// the Model's runtime-event path, exactly the chain a live event takes
// (MapEvent → runtime.Event → processRuntimeEvent → ToolActivity).
func feedToolEvent(t *testing.T, m *Model, raw []byte) {
	t.Helper()
	ev, err := opencode.MapEvent(raw, 1)
	if err != nil {
		t.Fatalf("MapEvent: %v", err)
	}
	m.processRuntimeEvent(ev)
}

func TestToolLifecycleMergesPendingRunningIntoSingleSuccess(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.sessionID = runtime.SessionID("s1")

	feedToolEvent(t, m, toolLifecycleJSON("s1", "c1", "bash", "pending"))
	feedToolEvent(t, m, toolLifecycleJSON("s1", "c1", "bash", "running"))
	feedToolEvent(t, m, toolLifecycleJSON("s1", "c1", "bash", "completed"))

	if len(m.tools) != 1 {
		t.Fatalf("tools = %d, want 1 (pending+running must merge into one ToolActivity): %+v", len(m.tools), m.tools)
	}
	got := m.tools[0]
	if got.CallID != "c1" {
		t.Errorf("CallID = %q, want c1", got.CallID)
	}
	if got.Name != "bash" {
		t.Errorf("Name = %q, want bash", got.Name)
	}
	if got.Status != ToolSuccess {
		t.Errorf("Status = %q, want %q", got.Status, ToolSuccess)
	}
}

func TestToolLifecycleErrorLeavesSingleFailed(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.sessionID = runtime.SessionID("s1")

	feedToolEvent(t, m, toolLifecycleJSON("s1", "c2", "bash", "pending"))
	feedToolEvent(t, m, toolLifecycleJSON("s1", "c2", "bash", "running"))
	feedToolEvent(t, m, toolLifecycleJSON("s1", "c2", "bash", "error"))

	if len(m.tools) != 1 {
		t.Fatalf("tools = %d, want 1: %+v", len(m.tools), m.tools)
	}
	got := m.tools[0]
	if got.CallID != "c2" {
		t.Errorf("CallID = %q, want c2", got.CallID)
	}
	if got.Name != "bash" {
		t.Errorf("Name = %q, want bash", got.Name)
	}
	if got.Status != ToolFailed {
		t.Errorf("Status = %q, want %q", got.Status, ToolFailed)
	}
}
