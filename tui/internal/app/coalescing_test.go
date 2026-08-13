package app

import (
	"strings"
	"testing"

	"codea/tui/internal/runtime"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

// TestHighFrequencyDeltasCoalesceToOneFlush proves that a burst of answer
// deltas is buffered (not appended per delta) and committed by a single tick
// flush. This is the acceptance semantics: N deltas -> 1 flush.
func TestHighFrequencyDeltasCoalesceToOneFlush(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.input = "hi"
	m.Update(enterKey())

	const n = 100
	for i := 0; i < n; i++ {
		m.Update(runtimeEventMsg{ev: runtime.Event{Type: "answer.delta", Content: "x"}})
	}

	if m.messages[1].Content != "" {
		t.Fatalf("deltas must be buffered, not appended per-delta (content=%q)", m.messages[1].Content)
	}

	m.Update(tickMsg{})

	want := strings.Repeat("x", n)
	if m.messages[1].Content != want {
		t.Fatalf("after flush content length = %d, want %d", len(m.messages[1].Content), n)
	}
}

// TestHighFrequencyDeltasDoNotRerenderPerToken proves that during a token
// flood the cached View is returned unchanged, and only the tick triggers a
// re-render. This is the "not one full redraw per token" guarantee.
func TestHighFrequencyDeltasDoNotRerenderPerToken(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.width, m.height = 100, 30
	m.input = "hi"
	m.Update(enterKey())
	before := m.View()

	const n = 100
	for i := 0; i < n; i++ {
		m.Update(runtimeEventMsg{ev: runtime.Event{Type: "answer.delta", Content: "x"}})
	}

	if got := m.View(); got != before {
		t.Fatal("View re-rendered during token flood without a tick")
	}

	m.Update(tickMsg{})
	after := m.View()
	if after == before {
		t.Fatal("View did not update after tick flush")
	}
	if !strings.Contains(after, strings.Repeat("x", n)) {
		t.Fatal("flushed answer not present in rendered view")
	}
}
