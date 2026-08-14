package app

import (
	"testing"

	"codea/tui/internal/runtime"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"

	tea "github.com/charmbracelet/bubbletea"
)

func ctrlSKey() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyCtrlS} }
func upKey() tea.KeyMsg    { return tea.KeyMsg{Type: tea.KeyUp} }
func downKey() tea.KeyMsg  { return tea.KeyMsg{Type: tea.KeyDown} }
func escKey() tea.KeyMsg   { return tea.KeyMsg{Type: tea.KeyEsc} }
func yKey() tea.KeyMsg     { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}} }
func aKey() tea.KeyMsg     { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}} }
func rKey() tea.KeyMsg     { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}} }
func nKey() tea.KeyMsg     { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}} }

func twoSessions() []runtime.Session {
	return []runtime.Session{
		{ID: "s1", Title: "One"},
		{ID: "s2", Title: "Two"},
	}
}

// openPanelAndResume drives ctrl+s → list-sessions result → cursor to target →
// Enter, resuming target in one helper. It requires the model to not be
// streaming (resume is blocked while streaming).
func openPanelAndResume(t *testing.T, m *Model, target string) {
	t.Helper()
	_, cmd := m.Update(ctrlSKey())
	if cmd == nil {
		t.Fatal("ctrl+s should issue ListSessionsCmd")
	}
	msg := cmd()
	lr, ok := msg.(listSessionsResultMsg)
	if !ok {
		t.Fatalf("cmd returned %T, want listSessionsResultMsg", msg)
	}
	if lr.err != nil {
		t.Fatalf("list sessions error: %v", lr.err)
	}
	m.Update(lr)
	if !m.sessionPanel.Visible {
		t.Fatal("session panel should be visible after fetch")
	}
	for i := range m.sessionPanel.Items {
		if m.sessionPanel.Items[i].ID == runtime.SessionID(target) {
			m.sessionPanel.Cursor = i
			m.Update(enterKey())
			return
		}
	}
	t.Fatalf("target session %q not found in panel", target)
}

func TestResumeSwitchesCurrentSession(t *testing.T) {
	client := fakeruntime.New()
	client.Sessions = twoSessions()
	m := NewModel(client)
	m.sessionID = runtime.SessionID("s1")

	openPanelAndResume(t, m, "s2")

	if m.sessionID != runtime.SessionID("s2") {
		t.Errorf("sessionID = %q, want s2 after resume", m.sessionID)
	}
	if m.sessionPanel.Visible {
		t.Error("session panel should close after resume")
	}
}

func TestResumeSessionResetsTransientState(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.sessionID = runtime.SessionID("s1")
	m.isStreaming = true
	m.pendingPrompt = &runtime.PromptRequest{MessageID: "stale"}
	m.streamBuf.WriteString("partial")
	m.reasoningBuf.WriteString("think")
	m.reasoningActive = true
	m.reasoningContent = "old reasoning"
	m.reasoningDuration = 123
	m.tools = []ToolActivity{{Name: "read", CallID: "c1", Status: ToolRunning}}
	m.messages = []ChatMessage{{Role: RoleUser, Content: "hi", Finished: true}}

	m.resumeSession(runtime.SessionID("s2"))

	if m.isStreaming {
		t.Error("isStreaming should be false after resume")
	}
	if m.pendingPrompt != nil {
		t.Error("pendingPrompt should be cleared after resume")
	}
	if m.streamBuf.Len() != 0 || m.reasoningBuf.Len() != 0 {
		t.Error("streaming/reasoning buffers should be reset after resume")
	}
	if m.reasoningActive || m.reasoningContent != "" || m.reasoningDuration != 0 {
		t.Error("reasoning state should be reset after resume")
	}
	if len(m.tools) != 0 {
		t.Errorf("tools = %d, want 0 after resume", len(m.tools))
	}
	if len(m.messages) != 0 {
		t.Errorf("messages = %d, want 0 after resume", len(m.messages))
	}
}

func TestOldSessionAnswerIgnoredAfterResume(t *testing.T) {
	client := fakeruntime.New()
	client.Sessions = twoSessions()
	m := NewModel(client)
	m.sessionID = runtime.SessionID("s1")

	openPanelAndResume(t, m, "s2")

	m.input = "hello"
	m.Update(enterKey()) // start a fresh stream on s2

	m.Update(runtimeEventMsg{ev: runtime.Event{Type: "answer.delta", SessionID: "s1", Content: "stale answer"}})
	m.Update(tickMsg{})

	if m.messages[1].Content != "" {
		t.Errorf("assistant content = %q, want empty (old-session answer leaked into resumed stream)", m.messages[1].Content)
	}
}

func TestOldSessionReasoningIgnoredAfterResume(t *testing.T) {
	client := fakeruntime.New()
	client.Sessions = twoSessions()
	m := NewModel(client)
	m.sessionID = runtime.SessionID("s1")

	openPanelAndResume(t, m, "s2")

	m.input = "hello"
	m.Update(enterKey())

	m.Update(runtimeEventMsg{ev: runtime.Event{Type: "reasoning.delta", SessionID: "s1", Content: "stale think"}})
	m.Update(tickMsg{})

	if m.reasoningActive {
		t.Error("reasoningActive = true, want false (old-session reasoning leaked)")
	}
	if m.reasoningContent != "" {
		t.Errorf("reasoningContent = %q, want empty (old-session reasoning leaked)", m.reasoningContent)
	}
}

func TestOldSessionToolIgnoredAfterResume(t *testing.T) {
	client := fakeruntime.New()
	client.Sessions = twoSessions()
	m := NewModel(client)
	m.sessionID = runtime.SessionID("s1")

	openPanelAndResume(t, m, "s2")

	m.Update(runtimeEventMsg{ev: runtime.Event{
		Type:      "tool.called",
		SessionID: "s1",
		Tool:      &runtime.ToolEvent{Name: "read", CallID: "c1"},
	}})

	if len(m.tools) != 0 {
		t.Errorf("tools = %d, want 0 (old-session tool leaked after resume)", len(m.tools))
	}
}

func TestOldSessionStepFinishedIgnoredAfterResume(t *testing.T) {
	client := fakeruntime.New()
	client.Sessions = twoSessions()
	m := NewModel(client)
	m.sessionID = runtime.SessionID("s1")

	openPanelAndResume(t, m, "s2")

	m.input = "again"
	m.Update(enterKey()) // fresh stream on s2
	if !m.isStreaming {
		t.Fatal("precondition: resumed session should be streaming after submit")
	}

	m.Update(runtimeEventMsg{ev: runtime.Event{Type: "step.finished", SessionID: "s1"}})

	if !m.isStreaming {
		t.Error("old-session step.finished should not end the resumed session's stream")
	}
}

func TestResumedSessionEventsAccepted(t *testing.T) {
	client := fakeruntime.New()
	client.Sessions = twoSessions()
	m := NewModel(client)
	m.sessionID = runtime.SessionID("s1")

	openPanelAndResume(t, m, "s2")

	m.input = "hello"
	m.Update(enterKey()) // submit on s2

	m.Update(runtimeEventMsg{ev: runtime.Event{Type: "answer.delta", SessionID: "s2", Content: "Hello"}})
	m.Update(tickMsg{})

	if len(m.messages) < 2 {
		t.Fatalf("messages = %d, want at least 2 (resumed-session answer)", len(m.messages))
	}
	if m.messages[1].Content != "Hello" {
		t.Errorf("assistant content = %q, want %q (resumed-session answer dropped)", m.messages[1].Content, "Hello")
	}
}

func TestResumeBlockedWhileStreaming(t *testing.T) {
	client := fakeruntime.New()
	client.Sessions = twoSessions()
	m := NewModel(client)
	m.sessionID = runtime.SessionID("s1")
	m.isStreaming = true

	openPanelAndResume(t, m, "s2")

	if m.sessionID != runtime.SessionID("s1") {
		t.Errorf("sessionID = %q, want s1 (resume blocked while streaming)", m.sessionID)
	}
	if !m.sessionPanel.Visible {
		t.Error("session panel should stay open when resume is blocked")
	}
	if m.sessionNotice == "" {
		t.Error("blocked resume should set a session notice")
	}
}

func TestSessionPanelEscCloses(t *testing.T) {
	client := fakeruntime.New()
	client.Sessions = twoSessions()
	m := NewModel(client)
	m.sessionID = runtime.SessionID("s1")

	_, cmd := m.Update(ctrlSKey())
	lr := cmd().(listSessionsResultMsg)
	m.Update(lr)
	if !m.sessionPanel.Visible {
		t.Fatal("precondition: panel should be visible")
	}

	m.Update(escKey())
	if m.sessionPanel.Visible {
		t.Error("esc should close the session panel")
	}
}
