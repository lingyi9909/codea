package app

import (
	"testing"

	"codea/tui/internal/runtime"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"

	tea "github.com/charmbracelet/bubbletea"
)

func enterKey() tea.KeyMsg    { return tea.KeyMsg{Type: tea.KeyEnter} }
func altEnterKey() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyEnter, Alt: true} }
func ctrlJKey() tea.KeyMsg    { return tea.KeyMsg{Type: tea.KeyCtrlJ} }
func runeKey(s string) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}
func backspaceKey() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyBackspace} }
func spaceKey() tea.KeyMsg     { return tea.KeyMsg{Type: tea.KeySpace, Runes: []rune{' '}} }

func TestSubmitEmptyPromptDoesNothing(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.input = "   "

	_, cmd := m.Update(enterKey())

	if cmd != nil {
		t.Error("empty prompt should not issue a cmd")
	}
	if len(m.messages) != 0 {
		t.Errorf("messages = %d, want 0", len(m.messages))
	}
	if m.isStreaming {
		t.Error("isStreaming should be false")
	}
}

func TestSubmitAppendsUserAndAssistantMessages(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.input = "hello"

	_, cmd := m.Update(enterKey())

	if cmd == nil {
		t.Fatal("submit should issue a prompt cmd")
	}
	if len(m.messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(m.messages))
	}
	if m.messages[0].Role != RoleUser || m.messages[0].Content != "hello" {
		t.Errorf("user message = %+v", m.messages[0])
	}
	if m.messages[1].Role != RoleAssistant || m.messages[1].Content != "" {
		t.Errorf("assistant message = %+v", m.messages[1])
	}
	if m.messages[1].Finished {
		t.Error("assistant message should start unfinished")
	}
	if !m.isStreaming {
		t.Error("isStreaming should be true")
	}
	if m.input != "" {
		t.Errorf("input = %q, want empty", m.input)
	}
}

func TestSubmitWithSessionSendsPrompt(t *testing.T) {
	client := fakeruntime.New()
	m := NewModel(client)
	m.sessionID = runtime.SessionID("s1")
	m.input = "review OrderService"

	_, cmd := m.Update(enterKey())
	if cmd == nil {
		t.Fatal("submit should issue a cmd")
	}
	msg := cmd()
	pr, ok := msg.(promptResultMsg)
	if !ok {
		t.Fatalf("cmd returned %T, want promptResultMsg", msg)
	}
	if pr.err != nil {
		t.Fatalf("prompt error: %v", pr.err)
	}

	prompts := client.Prompts()
	if len(prompts) != 1 {
		t.Fatalf("prompts = %d, want 1", len(prompts))
	}
	if prompts[0].SessionID != "s1" {
		t.Errorf("session = %q, want s1", prompts[0].SessionID)
	}
	if prompts[0].Request.Agent != "general" {
		t.Errorf("agent = %q, want general", prompts[0].Request.Agent)
	}
	if prompts[0].Request.MessageID == "" {
		t.Error("MessageID should be set")
	}
	if len(prompts[0].Request.Parts) != 1 {
		t.Fatalf("parts = %d, want 1", len(prompts[0].Request.Parts))
	}
	tp, ok := prompts[0].Request.Parts[0].(runtime.TextPart)
	if !ok {
		t.Fatalf("part = %T, want TextPart", prompts[0].Request.Parts[0])
	}
	if tp.Text != "review OrderService" {
		t.Errorf("text = %q, want %q", tp.Text, "review OrderService")
	}
}

func TestSubmitWithoutSessionCreatesSession(t *testing.T) {
	client := fakeruntime.New()
	m := NewModel(client)
	m.input = "hello"

	_, cmd := m.Update(enterKey())
	if cmd == nil {
		t.Fatal("submit should issue a cmd")
	}
	msg := cmd()
	pr, ok := msg.(promptResultMsg)
	if !ok {
		t.Fatalf("cmd returned %T, want promptResultMsg", msg)
	}
	if pr.err != nil {
		t.Fatalf("prompt error: %v", pr.err)
	}
	if pr.sessionID == "" {
		t.Error("sessionID should be set after create")
	}

	prompts := client.Prompts()
	if len(prompts) != 1 {
		t.Fatalf("prompts = %d, want 1", len(prompts))
	}
	if prompts[0].SessionID != pr.sessionID {
		t.Errorf("prompt session = %q, want %q", prompts[0].SessionID, pr.sessionID)
	}

	m.Update(pr)
	if m.sessionID != pr.sessionID {
		t.Errorf("sessionID not stored on promptResultMsg: got %q, want %q", m.sessionID, pr.sessionID)
	}
}

func TestTypingAppendsRunes(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.Update(runeKey("a"))
	m.Update(runeKey("b"))
	if m.input != "ab" {
		t.Errorf("input = %q, want ab", m.input)
	}
}

func TestSpaceKeyAppendsSpace(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.Update(runeKey("a"))
	m.Update(spaceKey())
	m.Update(runeKey("b"))
	if m.input != "a b" {
		t.Errorf("input = %q, want 'a b'", m.input)
	}
}

func TestBackspaceRemovesLastRune(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.input = "abc"
	m.Update(backspaceKey())
	if m.input != "ab" {
		t.Errorf("input = %q, want ab", m.input)
	}
	m.input = ""
	m.Update(backspaceKey())
	if m.input != "" {
		t.Errorf("backspace on empty = %q, want empty", m.input)
	}
}

func TestNewlineInsertsNewline(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.input = "a"
	m.Update(altEnterKey())
	if m.input != "a\n" {
		t.Errorf("alt+enter input = %q, want 'a\\n'", m.input)
	}
	m.Update(ctrlJKey())
	if m.input != "a\n\n" {
		t.Errorf("ctrl+j input = %q, want 'a\\n\\n'", m.input)
	}
}

func TestAnswerDeltaAppendsToAssistant(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.input = "hi"
	m.Update(enterKey())

	m.Update(runtimeEventMsg{ev: runtime.Event{Type: "answer.delta", Content: "Hello"}})
	m.Update(runtimeEventMsg{ev: runtime.Event{Type: "answer.delta", Content: " world"}})

	// Deltas are buffered and coalesced into the assistant message on tick.
	m.Update(tickMsg{})

	if len(m.messages) != 2 {
		t.Fatalf("messages = %d, want 2", len(m.messages))
	}
	if m.messages[1].Content != "Hello world" {
		t.Errorf("assistant content = %q, want 'Hello world'", m.messages[1].Content)
	}
}

func TestStepFinishedMarksStreamingComplete(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.input = "hi"
	m.Update(enterKey())

	m.Update(runtimeEventMsg{ev: runtime.Event{Type: "answer.delta", Content: "done"}})
	m.Update(runtimeEventMsg{ev: runtime.Event{Type: "step.finished"}})

	if m.isStreaming {
		t.Error("isStreaming should be false after step.finished")
	}
	if !m.messages[1].Finished {
		t.Error("assistant message should be finished")
	}
}

func TestReasoningDeltaPopulatesReasoningState(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.input = "hi"
	m.Update(enterKey())

	m.Update(runtimeEventMsg{ev: runtime.Event{Type: "reasoning.delta", Content: "think1"}})

	if !m.reasoningActive {
		t.Error("reasoningActive should be true")
	}

	// Reasoning deltas are buffered and flushed into reasoningContent on tick.
	m.Update(tickMsg{})
	if m.reasoningContent != "think1" {
		t.Errorf("reasoningContent = %q, want think1", m.reasoningContent)
	}
}
