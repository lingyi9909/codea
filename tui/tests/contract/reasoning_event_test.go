package contract

import (
	"testing"

	"codea/tui/internal/opencode"
	"codea/tui/internal/reasoning"
	"codea/tui/internal/runtime"
)

func mapSSE(t *testing.T, raw string) runtime.Event {
	t.Helper()
	ev, err := opencode.MapEvent([]byte(raw), 1)
	if err != nil {
		t.Fatalf("MapEvent: %v", err)
	}
	return ev
}

func collectProc(p *reasoning.Processor, events ...runtime.Event) []reasoning.Event {
	var out []reasoning.Event
	for _, e := range events {
		out = append(out, p.Process(e)...)
	}
	return out
}

// assertStream compares Kind+Content sequences, ignoring timing-dependent
// Duration/Interrupted fields so structured and fallback paths are comparable.
func assertStream(t *testing.T, got []reasoning.Event, want ...reasoning.Event) {
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

func structuredStream(t *testing.T) []reasoning.Event {
	p := reasoning.NewProcessor()
	return collectProc(p,
		mapSSE(t, `{"directory":"/tmp","payload":{"type":"message.part.delta","properties":{"sessionID":"s1","messageID":"m1","partID":"p1","field":"reasoning","delta":"I think"}}}`),
		mapSSE(t, `{"directory":"/tmp","payload":{"type":"message.part.delta","properties":{"sessionID":"s1","messageID":"m1","partID":"p1","field":"text","delta":"The answer"}}}`),
	)
}

func fallbackStream(t *testing.T) []reasoning.Event {
	p := reasoning.NewProcessor()
	return collectProc(p,
		mapSSE(t, `{"directory":"/tmp","payload":{"type":"message.part.delta","properties":{"sessionID":"s1","messageID":"m1","partID":"p1","field":"text","delta":"<think>I think</think>The answer"}}}`),
	)
}

func TestContractCaseAStructuredReasoning(t *testing.T) {
	got := structuredStream(t)
	assertStream(t, got,
		reasoning.Event{Kind: reasoning.EventReasoningStart},
		reasoning.Event{Kind: reasoning.EventReasoningDelta, Content: "I think"},
		reasoning.Event{Kind: reasoning.EventReasoningEnd},
		reasoning.Event{Kind: reasoning.EventAnswerDelta, Content: "The answer"},
	)
}

func TestContractCaseBFallbackMatchesStructured(t *testing.T) {
	structured := structuredStream(t)
	fallback := fallbackStream(t)
	// Case B must be user-semantically identical to Case A.
	assertStream(t, fallback, structured...)
}

func TestContractCaseCStructuredAndThinkDeduplicated(t *testing.T) {
	p := reasoning.NewProcessor()
	got := collectProc(p,
		mapSSE(t, `{"directory":"/tmp","payload":{"type":"message.part.delta","properties":{"sessionID":"s1","messageID":"m1","partID":"p1","field":"reasoning","delta":"I think"}}}`),
		mapSSE(t, `{"directory":"/tmp","payload":{"type":"message.part.delta","properties":{"sessionID":"s1","messageID":"m1","partID":"p1","field":"text","delta":"<think>I think</think>The answer"}}}`),
	)
	// Reasoning appears exactly once; the duplicate <think> is stripped.
	assertStream(t, got,
		reasoning.Event{Kind: reasoning.EventReasoningStart},
		reasoning.Event{Kind: reasoning.EventReasoningDelta, Content: "I think"},
		reasoning.Event{Kind: reasoning.EventReasoningEnd},
		reasoning.Event{Kind: reasoning.EventAnswerDelta, Content: "The answer"},
	)
}

func TestContractCaseDNoReasoningModel(t *testing.T) {
	p := reasoning.NewProcessor()
	got := collectProc(p,
		mapSSE(t, `{"directory":"/tmp","payload":{"type":"message.part.delta","properties":{"sessionID":"s1","messageID":"m1","partID":"p1","field":"text","delta":"Just an answer"}}}`),
	)
	assertStream(t, got,
		reasoning.Event{Kind: reasoning.EventAnswerDelta, Content: "Just an answer"},
	)
}

func TestContractReasoningDeltaTypeMappedCorrectly(t *testing.T) {
	ev := mapSSE(t, `{"directory":"/tmp","payload":{"type":"message.part.delta","properties":{"sessionID":"s1","messageID":"m1","partID":"p1","field":"reasoning","delta":"x"}}}`)
	if ev.Type != opencode.CodeaEventReasoningDelta {
		t.Fatalf("expected reasoning.delta, got %q", ev.Type)
	}
	// The processor must recognize the mapped event as structured reasoning.
	p := reasoning.NewProcessor()
	got := p.Process(ev)
	if len(got) == 0 || got[0].Kind != reasoning.EventReasoningStart {
		t.Fatalf("expected processor to recognize structured reasoning, got %+v", got)
	}
}
