package app

import (
	"testing"
	"time"

	fakeruntime "codea/tui/tests/fixtures/fake-runtime"

	tea "github.com/charmbracelet/bubbletea"
)

func ctrlTKey() tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyCtrlT} }

func TestToggleThinkTogglesExpanded(t *testing.T) {
	m := NewModel(fakeruntime.New())

	m.Update(ctrlTKey())
	if !m.reasoningExpanded {
		t.Error("ctrl+t should expand reasoning")
	}
	m.Update(ctrlTKey())
	if m.reasoningExpanded {
		t.Error("ctrl+t again should collapse reasoning")
	}
}

func TestFormatDuration(t *testing.T) {
	cases := []struct {
		d    time.Duration
		want string
	}{
		{2800 * time.Millisecond, "2.8s"},
		{3200 * time.Millisecond, "3.2s"},
		{0, "0.0s"},
		{60 * time.Second, "60.0s"},
	}
	for _, c := range cases {
		if got := formatDuration(c.d); got != c.want {
			t.Errorf("formatDuration(%v) = %q, want %q", c.d, got, c.want)
		}
	}
}

func TestRenderReasoningActiveShowsContent(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.reasoningActive = true
	m.reasoningContent = "analyzing..."

	if got := m.renderReasoning(); got != "analyzing..." {
		t.Errorf("active reasoning = %q, want %q", got, "analyzing...")
	}
}

func TestRenderReasoningCollapsedShowsSummary(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.reasoningActive = false
	m.reasoningContent = "analysis"
	m.reasoningExpanded = false
	m.reasoningDuration = 2800 * time.Millisecond

	if got := m.renderReasoning(); got != "✓ Spent 2.8s thinking" {
		t.Errorf("collapsed reasoning = %q, want %q", got, "✓ Spent 2.8s thinking")
	}
}

func TestRenderReasoningExpandedShowsContent(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.reasoningActive = false
	m.reasoningContent = "analysis"
	m.reasoningExpanded = true
	m.reasoningDuration = 2800 * time.Millisecond

	if got := m.renderReasoning(); got != "analysis" {
		t.Errorf("expanded reasoning = %q, want %q", got, "analysis")
	}
}

func TestRenderReasoningNoneIsEmpty(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.reasoningActive = false
	m.reasoningContent = ""

	if got := m.renderReasoning(); got != "" {
		t.Errorf("no reasoning = %q, want empty", got)
	}
}
