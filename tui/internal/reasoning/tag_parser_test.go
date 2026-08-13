package reasoning

import (
	"strings"
	"testing"
)

func assertEvents(t *testing.T, got []ParserEvent, want ...ParserEvent) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("event count mismatch: got %d %+v, want %d %+v", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i].Type != want[i].Type || got[i].Content != want[i].Content {
			t.Fatalf("event %d mismatch: got %+v, want %+v", i, got[i], want[i])
		}
	}
}

func joinContent(events []ParserEvent, typ ParserEventType) string {
	var b strings.Builder
	for _, e := range events {
		if e.Type == typ {
			b.WriteString(e.Content)
		}
	}
	return b.String()
}

func countType(events []ParserEvent, typ ParserEventType) int {
	n := 0
	for _, e := range events {
		if e.Type == typ {
			n++
		}
	}
	return n
}

func TestTagParserStandard(t *testing.T) {
	p := NewTagParser()
	events := append(p.Feed("<think>I think</think>"), p.Flush()...)
	assertEvents(t, events,
		ParserEvent{Type: ParserEventReasoningStart},
		ParserEvent{Type: ParserEventReasoningDelta, Content: "I think"},
		ParserEvent{Type: ParserEventReasoningEnd},
	)
}

func TestTagParserCrossChunk(t *testing.T) {
	cases := []struct {
		name   string
		chunks []string
	}{
		{"open split", []string{"<thi", "nk>content</think>"}},
		{"close split", []string{"<think>content</th", "ink>"}},
		{"both split", []string{"<th", "ink>content</th", "ink>"}},
		{"char by char", []string{"<", "t", "h", "i", "n", "k", ">", "c", "o", "n", "t", "e", "n", "t", "<", "/", "t", "h", "i", "n", "k", ">"}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			p := NewTagParser()
			var events []ParserEvent
			for _, chunk := range c.chunks {
				events = append(events, p.Feed(chunk)...)
			}
			events = append(events, p.Flush()...)
			if got := joinContent(events, ParserEventReasoningDelta); got != "content" {
				t.Fatalf("expected reasoning 'content', got %q", got)
			}
			if got := joinContent(events, ParserEventAnswerDelta); got != "" {
				t.Fatalf("expected no answer delta, got %q", got)
			}
			if n := countType(events, ParserEventReasoningStart); n != 1 {
				t.Fatalf("expected 1 start, got %d", n)
			}
			if n := countType(events, ParserEventReasoningEnd); n != 1 {
				t.Fatalf("expected 1 end, got %d", n)
			}
		})
	}
}

func TestTagParserAnswerBeforeThink(t *testing.T) {
	p := NewTagParser()
	events := append(p.Feed("hello <think>why</think>world"), p.Flush()...)
	assertEvents(t, events,
		ParserEvent{Type: ParserEventAnswerDelta, Content: "hello "},
		ParserEvent{Type: ParserEventReasoningStart},
		ParserEvent{Type: ParserEventReasoningDelta, Content: "why"},
		ParserEvent{Type: ParserEventReasoningEnd},
		ParserEvent{Type: ParserEventAnswerDelta, Content: "world"},
	)
}

func TestTagParserNoThink(t *testing.T) {
	p := NewTagParser()
	events := append(p.Feed("just an answer, no tags"), p.Flush()...)
	assertEvents(t, events,
		ParserEvent{Type: ParserEventAnswerDelta, Content: "just an answer, no tags"},
	)
}

func TestTagParserUnclosedThinkFlush(t *testing.T) {
	p := NewTagParser()
	events := append(p.Feed("<think>unfinished"), p.Flush()...)
	assertEvents(t, events,
		ParserEvent{Type: ParserEventReasoningStart},
		ParserEvent{Type: ParserEventReasoningDelta, Content: "unfinished"},
		ParserEvent{Type: ParserEventReasoningEnd},
	)
}

func TestTagParserUnclosedThinkAcrossChunk(t *testing.T) {
	p := NewTagParser()
	var events []ParserEvent
	events = append(events, p.Feed("<think>part")...)
	events = append(events, p.Feed("ial")...)
	events = append(events, p.Flush()...)
	assertEvents(t, events,
		ParserEvent{Type: ParserEventReasoningStart},
		ParserEvent{Type: ParserEventReasoningDelta, Content: "part"},
		ParserEvent{Type: ParserEventReasoningDelta, Content: "ial"},
		ParserEvent{Type: ParserEventReasoningEnd},
	)
}

func TestTagParserEmptyThink(t *testing.T) {
	p := NewTagParser()
	events := append(p.Feed("<think></think>"), p.Flush()...)
	// The parser faithfully reports start+end with no delta; the processor is
	// responsible for dropping the resulting empty block.
	assertEvents(t, events,
		ParserEvent{Type: ParserEventReasoningStart},
		ParserEvent{Type: ParserEventReasoningEnd},
	)
}

func TestTagParserMultipleBlocks(t *testing.T) {
	p := NewTagParser()
	events := append(p.Feed("<think>a</think><think>b</think>"), p.Flush()...)
	assertEvents(t, events,
		ParserEvent{Type: ParserEventReasoningStart},
		ParserEvent{Type: ParserEventReasoningDelta, Content: "a"},
		ParserEvent{Type: ParserEventReasoningEnd},
		ParserEvent{Type: ParserEventReasoningStart},
		ParserEvent{Type: ParserEventReasoningDelta, Content: "b"},
		ParserEvent{Type: ParserEventReasoningEnd},
	)
}

func TestTagParserNestedThinkIsReasoningContent(t *testing.T) {
	p := NewTagParser()
	events := append(p.Feed("<think>a<think>b</think>c</think>"), p.Flush()...)
	assertEvents(t, events,
		ParserEvent{Type: ParserEventReasoningStart},
		ParserEvent{Type: ParserEventReasoningDelta, Content: "a<think>b"},
		ParserEvent{Type: ParserEventReasoningEnd},
		ParserEvent{Type: ParserEventAnswerDelta, Content: "c</think>"},
	)
}

func TestTagParserDoubleOpenBracket(t *testing.T) {
	p := NewTagParser()
	events := append(p.Feed("<<think>x</think>"), p.Flush()...)
	assertEvents(t, events,
		ParserEvent{Type: ParserEventAnswerDelta, Content: "<"},
		ParserEvent{Type: ParserEventReasoningStart},
		ParserEvent{Type: ParserEventReasoningDelta, Content: "x"},
		ParserEvent{Type: ParserEventReasoningEnd},
	)
}

func TestTagParserMalformedOpenTagIsAnswer(t *testing.T) {
	p := NewTagParser()
	events := append(p.Feed("<thinkx>text</think>"), p.Flush()...)
	assertEvents(t, events,
		ParserEvent{Type: ParserEventAnswerDelta, Content: "<thinkx>text</think>"},
	)
}

func TestTagParserLessThanNotTag(t *testing.T) {
	p := NewTagParser()
	events := append(p.Feed("5 < 3"), p.Flush()...)
	assertEvents(t, events,
		ParserEvent{Type: ParserEventAnswerDelta, Content: "5 < 3"},
	)
}

func TestTagParserLessThanAcrossChunkPreservesAnswer(t *testing.T) {
	p := NewTagParser()
	var events []ParserEvent
	events = append(events, p.Feed("a <")...)
	events = append(events, p.Feed(" b")...)
	events = append(events, p.Flush()...)
	if got := joinContent(events, ParserEventAnswerDelta); got != "a < b" {
		t.Fatalf("expected answer 'a < b', got %q", got)
	}
	if got := joinContent(events, ParserEventReasoningDelta); got != "" {
		t.Fatalf("expected no reasoning, got %q", got)
	}
}

func TestTagParserIsInReasoning(t *testing.T) {
	p := NewTagParser()
	if p.IsInReasoning() {
		t.Fatal("expected not in reasoning initially")
	}
	p.Feed("<think>")
	if !p.IsInReasoning() {
		t.Fatal("expected in reasoning after open tag")
	}
	p.Feed("x</think>")
	if p.IsInReasoning() {
		t.Fatal("expected not in reasoning after close tag")
	}
}
