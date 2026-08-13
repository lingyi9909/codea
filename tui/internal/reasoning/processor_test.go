package reasoning

import (
	"testing"
	"time"

	"codea/tui/internal/runtime"
)

func reasoningDelta(content string) runtime.Event {
	return runtime.Event{Type: runtime.EventType("reasoning.delta"), Content: content}
}

func answerDelta(content string) runtime.Event {
	return runtime.Event{Type: runtime.EventType("answer.delta"), Content: content}
}

func stepStarted() runtime.Event {
	return runtime.Event{Type: runtime.EventType("step.started")}
}

func assertProcEvents(t *testing.T, got []Event, want ...Event) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("event count mismatch: got %d %+v, want %d %+v", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i].Kind != want[i].Kind || got[i].Content != want[i].Content {
			t.Fatalf("event %d mismatch: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func feedAll(p *Processor, events ...runtime.Event) []Event {
	var out []Event
	for _, e := range events {
		out = append(out, p.Process(e)...)
	}
	return out
}

func TestProcessorStructuredReasoning(t *testing.T) {
	p := NewProcessor()
	got := feedAll(p,
		reasoningDelta("I think"),
		reasoningDelta(" step by step"),
		answerDelta("The answer"),
	)
	got = append(got, p.Flush()...)
	assertProcEvents(t, got,
		Event{Kind: EventReasoningStart},
		Event{Kind: EventReasoningDelta, Content: "I think"},
		Event{Kind: EventReasoningDelta, Content: " step by step"},
		Event{Kind: EventReasoningEnd},
		Event{Kind: EventAnswerDelta, Content: "The answer"},
	)
}

func TestProcessorThinkFallback(t *testing.T) {
	p := NewProcessor()
	got := feedAll(p, answerDelta("<think>I think</think>The answer"))
	got = append(got, p.Flush()...)
	assertProcEvents(t, got,
		Event{Kind: EventReasoningStart},
		Event{Kind: EventReasoningDelta, Content: "I think"},
		Event{Kind: EventReasoningEnd},
		Event{Kind: EventAnswerDelta, Content: "The answer"},
	)
}

func TestProcessorStructuredAndThinkDeduplicated(t *testing.T) {
	p := NewProcessor()
	got := feedAll(p,
		reasoningDelta("I think"),
		answerDelta("<think>I think</think>The answer"),
	)
	got = append(got, p.Flush()...)
	assertProcEvents(t, got,
		Event{Kind: EventReasoningStart},
		Event{Kind: EventReasoningDelta, Content: "I think"},
		Event{Kind: EventReasoningEnd},
		Event{Kind: EventAnswerDelta, Content: "The answer"},
	)
}

func TestProcessorNoReasoningModel(t *testing.T) {
	p := NewProcessor()
	got := feedAll(p, answerDelta("Just an answer"))
	got = append(got, p.Flush()...)
	assertProcEvents(t, got,
		Event{Kind: EventAnswerDelta, Content: "Just an answer"},
	)
}

func TestProcessorReasoningDuration(t *testing.T) {
	clk := newFakeClock()
	p := NewProcessor(WithClock(clk))
	p.Process(reasoningDelta("think"))
	clk.Advance(1200 * time.Millisecond)
	events := p.Process(answerDelta("answer"))

	var end *Event
	for i := range events {
		if events[i].Kind == EventReasoningEnd {
			end = &events[i]
		}
	}
	if end == nil {
		t.Fatal("expected ReasoningEnd event")
	}
	if end.Duration != 1200*time.Millisecond {
		t.Fatalf("expected duration 1.2s, got %v", end.Duration)
	}
	if end.Interrupted {
		t.Fatal("expected normal end, got interrupted")
	}
}

func TestProcessorMultipleReasoningBlocks(t *testing.T) {
	p := NewProcessor()
	feedAll(p,
		reasoningDelta("first"),
		answerDelta("mid"),
		stepStarted(),
		answerDelta("<think>second</think>end"),
	)
	p.Flush()

	snap := p.Snapshot()
	if len(snap.Blocks) != 2 {
		t.Fatalf("expected 2 blocks, got %d", len(snap.Blocks))
	}
	if snap.Blocks[0].Content != "first" || snap.Blocks[0].State != BlockCompleted {
		t.Fatalf("block 0 unexpected: %+v", snap.Blocks[0])
	}
	if snap.Blocks[1].Content != "second" || snap.Blocks[1].State != BlockCompleted {
		t.Fatalf("block 1 unexpected: %+v", snap.Blocks[1])
	}
}

// TestProcessorStructuredPrefixThinkDedupSingleChunk guards the de-dup bug:
// a non-empty answer prefix must NOT lift structured reasoning's suppression of
// a following <think> duplicate within the same cycle.
func TestProcessorStructuredPrefixThinkDedupSingleChunk(t *testing.T) {
	p := NewProcessor()
	got := feedAll(p,
		reasoningDelta("I think"),
		answerDelta("prefix <think>duplicate</think>suffix"),
	)
	got = append(got, p.Flush()...)
	assertProcEvents(t, got,
		Event{Kind: EventReasoningStart},
		Event{Kind: EventReasoningDelta, Content: "I think"},
		Event{Kind: EventReasoningEnd},
		Event{Kind: EventAnswerDelta, Content: "prefix "},
		Event{Kind: EventAnswerDelta, Content: "suffix"},
	)
	if n := countEvents(got, EventReasoningDelta); n != 1 {
		t.Fatalf("expected reasoning emitted exactly once, got %d", n)
	}
}

// TestProcessorStructuredPrefixThinkDedupCrossChunk is the streaming variant:
// the <think>/</think> tags are split across answer chunks. Suppression must
// still hold, reasoning emitted once, and prefix + final answer fully preserved.
func TestProcessorStructuredPrefixThinkDedupCrossChunk(t *testing.T) {
	p := NewProcessor()
	got := feedAll(p,
		reasoningDelta("I think"),
		answerDelta("prefix <thi"),
		answerDelta("nk>duplicate</th"),
		answerDelta("ink>suffix"),
	)
	got = append(got, p.Flush()...)
	assertProcEvents(t, got,
		Event{Kind: EventReasoningStart},
		Event{Kind: EventReasoningDelta, Content: "I think"},
		Event{Kind: EventReasoningEnd},
		Event{Kind: EventAnswerDelta, Content: "prefix "},
		Event{Kind: EventAnswerDelta, Content: "suffix"},
	)
	if n := countEvents(got, EventReasoningDelta); n != 1 {
		t.Fatalf("expected reasoning emitted exactly once, got %d", n)
	}
}

func TestProcessorEmptyThinkNoGarbageBlock(t *testing.T) {
	p := NewProcessor()
	got := feedAll(p, answerDelta("<think></think>real answer"))
	got = append(got, p.Flush()...)

	// Empty think must produce no reasoning events, only the answer.
	assertProcEvents(t, got,
		Event{Kind: EventAnswerDelta, Content: "real answer"},
	)
	if snap := p.Snapshot(); len(snap.Blocks) != 0 {
		t.Fatalf("expected no blocks, got %d", len(snap.Blocks))
	}
}

func TestProcessorEmptyStructuredDeltaNoGarbage(t *testing.T) {
	p := NewProcessor()
	got := feedAll(p, reasoningDelta(""), reasoningDelta(""))
	got = append(got, p.Process(answerDelta("answer"))...)
	if len(got) != 1 || got[0].Kind != EventAnswerDelta || got[0].Content != "answer" {
		t.Fatalf("expected only answer delta, got %+v", got)
	}
	if snap := p.Snapshot(); len(snap.Blocks) != 0 {
		t.Fatalf("expected no blocks, got %d", len(snap.Blocks))
	}
}

func TestProcessorToolAndRawEventsIgnored(t *testing.T) {
	p := NewProcessor()
	got := feedAll(p,
		runtime.Event{Type: runtime.EventType("tool.called"), Content: "read"},
		runtime.Event{Type: runtime.EventType("raw"), Content: "plugin"},
		runtime.Event{Type: runtime.EventType("session.status")},
	)
	if len(got) != 0 {
		t.Fatalf("expected no events from tool/raw/session, got %+v", got)
	}
}

func TestProcessorErrorInterruptsReasoning(t *testing.T) {
	p := NewProcessor()
	p.Process(reasoningDelta("thinking..."))
	events := p.Process(runtime.Event{Type: runtime.EventType("session.error"), Content: "boom"})

	var end *Event
	for i := range events {
		if events[i].Kind == EventReasoningEnd {
			end = &events[i]
		}
	}
	if end == nil {
		t.Fatal("expected ReasoningEnd on error")
	}
	if !end.Interrupted {
		t.Fatal("expected interrupted=true on error end")
	}
	if snap := p.Snapshot(); len(snap.Blocks) != 1 || snap.Blocks[0].State != BlockInterrupted {
		t.Fatalf("expected interrupted block, got %+v", snap.Blocks)
	}
}

func TestProcessorUnclosedThinkFlush(t *testing.T) {
	p := NewProcessor()
	got := feedAll(p, answerDelta("<think>unfinished"))
	got = append(got, p.Flush()...)
	assertProcEvents(t, got,
		Event{Kind: EventReasoningStart},
		Event{Kind: EventReasoningDelta, Content: "unfinished"},
		Event{Kind: EventReasoningEnd},
	)
}

func TestProcessorReset(t *testing.T) {
	p := NewProcessor()
	feedAll(p, reasoningDelta("old"), answerDelta("x"))
	p.Reset()

	if snap := p.Snapshot(); len(snap.Blocks) != 0 {
		t.Fatalf("expected empty after Reset, got %d blocks", len(snap.Blocks))
	}
	got := feedAll(p, answerDelta("<think>new</think>"))
	if n := countEvents(got, EventReasoningStart); n != 1 {
		t.Fatalf("expected fresh parser after Reset, got %d starts", n)
	}
}

func countEvents(events []Event, kind EventKind) int {
	n := 0
	for _, e := range events {
		if e.Kind == kind {
			n++
		}
	}
	return n
}
