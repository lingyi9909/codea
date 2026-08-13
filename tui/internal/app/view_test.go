package app

import (
	"strings"
	"testing"

	"codea/tui/internal/runtime"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"

	tea "github.com/charmbracelet/bubbletea"
)

func ctrlCKey() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyCtrlC} }
func ctrlLKey() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyCtrlL} }

func TestStatusLabel(t *testing.T) {
	cases := []struct {
		s    runtime.RuntimeStatus
		want string
	}{
		{runtime.RuntimeHealthy, "Ready"},
		{runtime.RuntimeStopped, "Stopped"},
		{runtime.RuntimeStarting, "Starting"},
		{runtime.RuntimeStopping, "Stopping"},
		{runtime.RuntimeCrashed, "Crashed"},
	}
	for _, c := range cases {
		if got := statusLabel(c.s); got != c.want {
			t.Errorf("statusLabel(%q) = %q, want %q", c.s, got, c.want)
		}
	}
}

func TestStatusDot(t *testing.T) {
	cases := []struct {
		s    runtime.RuntimeStatus
		want string
	}{
		{runtime.RuntimeHealthy, "●"},
		{runtime.RuntimeCrashed, "●"},
		{runtime.RuntimeStopped, "○"},
		{runtime.RuntimeStarting, "○"},
	}
	for _, c := range cases {
		if got := statusDot(c.s); got != c.want {
			t.Errorf("statusDot(%q) = %q, want %q", c.s, got, c.want)
		}
	}
}

func TestToolSymbol(t *testing.T) {
	cases := []struct {
		s    ToolStatus
		want string
	}{
		{ToolRunning, "◌"},
		{ToolSuccess, "✓"},
		{ToolFailed, "✗"},
	}
	for _, c := range cases {
		if got := toolSymbol(c.s); got != c.want {
			t.Errorf("toolSymbol(%q) = %q, want %q", c.s, got, c.want)
		}
	}
}

func TestRenderHeaderContainsNameAndStatus(t *testing.T) {
	m := NewModel(fakeruntime.New())
	if got := m.renderHeader(); !strings.Contains(got, "Codea") || !strings.Contains(got, "Stopped") {
		t.Errorf("header = %q, want Codea + Stopped", got)
	}
	m.runtimeStatus = runtime.RuntimeHealthy
	if got := m.renderHeader(); !strings.Contains(got, "Ready") {
		t.Errorf("header = %q, want Ready", got)
	}
}

func TestRenderStatusLine(t *testing.T) {
	m := NewModel(fakeruntime.New())
	if got := m.renderStatusLine(); got != "Ready" {
		t.Errorf("idle status = %q, want Ready", got)
	}
	m.isStreaming = true
	if got := m.renderStatusLine(); !strings.Contains(got, "Working") {
		t.Errorf("streaming status = %q, want Working", got)
	}
}

func TestRenderTools(t *testing.T) {
	m := NewModel(fakeruntime.New())
	if got := m.renderTools(); got != "" {
		t.Errorf("empty tools = %q, want empty", got)
	}
	m.tools = []ToolActivity{
		{Name: "read", CallID: "c1", Status: ToolSuccess},
		{Name: "bash", CallID: "c2", Status: ToolRunning},
	}
	got := m.renderTools()
	if !strings.Contains(got, "read") || !strings.Contains(got, "✓") {
		t.Errorf("tools = %q, want read + success symbol", got)
	}
	if !strings.Contains(got, "bash") || !strings.Contains(got, "◌") {
		t.Errorf("tools = %q, want bash + running symbol", got)
	}
}

func TestViewContainsMessages(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.input = "hi"
	m.Update(enterKey())
	m.Update(runtimeEventMsg{ev: runtime.Event{Type: "answer.delta", Content: "hello there"}})

	v := m.View()
	if !strings.Contains(v, "hi") {
		t.Errorf("View missing user message: %q", v)
	}
	if !strings.Contains(v, "hello there") {
		t.Errorf("View missing assistant answer: %q", v)
	}
}

func TestWindowSizeMsgSetsDimensions(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	if m.width != 100 || m.height != 30 {
		t.Errorf("dims = %dx%d, want 100x30", m.width, m.height)
	}
}

func TestQuitKeyReturnsQuit(t *testing.T) {
	m := NewModel(fakeruntime.New())
	_, cmd := m.Update(ctrlCKey())
	if cmd == nil {
		t.Fatal("ctrl+c should issue a quit cmd")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Errorf("ctrl+c cmd returned %T, want tea.QuitMsg", cmd())
	}
}

func TestClearScreenClearsMessages(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.input = "hi"
	m.Update(enterKey())
	if len(m.messages) != 2 {
		t.Fatalf("setup: messages = %d, want 2", len(m.messages))
	}
	m.Update(ctrlLKey())
	if len(m.messages) != 0 {
		t.Errorf("messages after clear = %d, want 0", len(m.messages))
	}
}

func TestToolEventsAddAndUpdate(t *testing.T) {
	m := NewModel(fakeruntime.New())

	m.Update(runtimeEventMsg{ev: runtime.Event{
		Type: "tool.called",
		Tool: &runtime.ToolEvent{Name: "read", CallID: "c1"},
	}})
	if len(m.tools) != 1 {
		t.Fatalf("tools = %d, want 1", len(m.tools))
	}
	if m.tools[0].Status != ToolRunning || m.tools[0].Name != "read" {
		t.Errorf("tool = %+v, want running read", m.tools[0])
	}

	m.Update(runtimeEventMsg{ev: runtime.Event{
		Type: "tool.success",
		Tool: &runtime.ToolEvent{Name: "read", CallID: "c1"},
	}})
	if m.tools[0].Status != ToolSuccess {
		t.Errorf("tool status = %q, want success", m.tools[0].Status)
	}
}

func TestSubscribeSetsHealthy(t *testing.T) {
	m := NewModel(fakeruntime.New())
	ch := make(chan runtime.Event)
	m.Update(subscribedMsg{ch: ch})
	if m.runtimeStatus != runtime.RuntimeHealthy {
		t.Errorf("runtimeStatus = %q, want healthy", m.runtimeStatus)
	}
}
