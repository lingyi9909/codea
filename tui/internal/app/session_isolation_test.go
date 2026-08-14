package app

import (
	"testing"

	"codea/tui/internal/runtime"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

// The /global/event stream delivers events for every session in the runtime.
// The TUI must only consume session-scoped events that belong to its current
// session; foreign-session events must not leak into chat, reasoning, tools, or
// stream lifecycle.

func TestForeignSessionAnswerIgnored(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.sessionID = runtime.SessionID("current")
	m.input = "hi"
	m.Update(enterKey())

	m.Update(runtimeEventMsg{ev: runtime.Event{Type: "answer.delta", SessionID: "other", Content: "foreign answer"}})
	m.Update(tickMsg{})

	if got := m.messages[1].Content; got != "" {
		t.Errorf("assistant content = %q, want empty (foreign answer leaked)", got)
	}
}

func TestForeignSessionReasoningIgnored(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.sessionID = runtime.SessionID("current")
	m.input = "hi"
	m.Update(enterKey())

	m.Update(runtimeEventMsg{ev: runtime.Event{Type: "reasoning.delta", SessionID: "other", Content: "foreign think"}})
	m.Update(tickMsg{})

	if m.reasoningActive {
		t.Error("reasoningActive = true, want false (foreign reasoning leaked)")
	}
	if m.reasoningContent != "" {
		t.Errorf("reasoningContent = %q, want empty (foreign reasoning leaked)", m.reasoningContent)
	}
}

func TestForeignSessionToolIgnored(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.sessionID = runtime.SessionID("current")
	m.input = "hi"
	m.Update(enterKey())

	m.Update(runtimeEventMsg{ev: runtime.Event{
		Type:      "tool.called",
		SessionID: "other",
		Tool:      &runtime.ToolEvent{Name: "read", CallID: "c1"},
	}})

	if len(m.tools) != 0 {
		t.Errorf("tools = %d, want 0 (foreign tool leaked)", len(m.tools))
	}
}

func TestForeignSessionStepFinishedIgnored(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.sessionID = runtime.SessionID("current")
	m.input = "hi"
	m.Update(enterKey())

	m.Update(runtimeEventMsg{ev: runtime.Event{Type: "step.finished", SessionID: "other"}})

	if !m.isStreaming {
		t.Error("isStreaming = false, want true (foreign step.finished ended current stream)")
	}
	if m.messages[1].Finished {
		t.Error("assistant message finished by foreign step.finished")
	}
}

func TestCurrentSessionEventsAccepted(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.sessionID = runtime.SessionID("current")
	m.input = "hi"
	m.Update(enterKey())

	m.Update(runtimeEventMsg{ev: runtime.Event{Type: "answer.delta", SessionID: "current", Content: "Hello"}})
	m.Update(tickMsg{})

	if got := m.messages[1].Content; got != "Hello" {
		t.Errorf("assistant content = %q, want %q (current-session answer dropped)", got, "Hello")
	}
}
