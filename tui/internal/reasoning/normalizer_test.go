package reasoning

import (
	"testing"

	"codea/tui/internal/runtime"
)

func TestNormalizerStructuredReasoningDelta(t *testing.T) {
	n := NewNormalizer()
	ev := runtime.Event{Type: runtime.EventType("reasoning.delta"), Content: "I think step by step"}
	got := n.Normalize(ev)
	if got.Kind != KindReasoning {
		t.Fatalf("expected KindReasoning, got %v", got.Kind)
	}
	if got.Content != "I think step by step" {
		t.Fatalf("expected content preserved, got %q", got.Content)
	}
}

func TestNormalizerAnswerDeltaIsText(t *testing.T) {
	n := NewNormalizer()
	ev := runtime.Event{Type: runtime.EventType("answer.delta"), Content: "The final answer"}
	got := n.Normalize(ev)
	if got.Kind != KindText {
		t.Fatalf("expected KindText, got %v", got.Kind)
	}
	if got.Content != "The final answer" {
		t.Fatalf("expected content preserved, got %q", got.Content)
	}
}

func TestNormalizerToolEventIsOther(t *testing.T) {
	n := NewNormalizer()
	for _, typ := range []runtime.EventType{
		"tool.called", "tool.success", "tool.failed",
		"approval.requested", "step.started", "step.finished",
	} {
		got := n.Normalize(runtime.Event{Type: typ, Content: "not reasoning"})
		if got.Kind != KindOther {
			t.Fatalf("type %q: expected KindOther, got %v", typ, got.Kind)
		}
		if got.Content != "" {
			t.Fatalf("type %q: expected no content leak, got %q", typ, got.Content)
		}
	}
}

func TestNormalizerRawEventIsOther(t *testing.T) {
	n := NewNormalizer()
	got := n.Normalize(runtime.Event{Type: runtime.EventType("raw"), Content: "plugin.added payload"})
	if got.Kind != KindOther {
		t.Fatalf("expected KindOther for raw, got %v", got.Kind)
	}
}

func TestNormalizerEmptyReasoningStillReasoningKind(t *testing.T) {
	// An empty reasoning delta is still classified as reasoning; the tracker
	// is responsible for not emitting a garbage block out of empty content.
	n := NewNormalizer()
	got := n.Normalize(runtime.Event{Type: runtime.EventType("reasoning.delta")})
	if got.Kind != KindReasoning {
		t.Fatalf("expected KindReasoning, got %v", got.Kind)
	}
	if got.Content != "" {
		t.Fatalf("expected empty content, got %q", got.Content)
	}
}

func TestNormalizerUnknownEventIsOther(t *testing.T) {
	n := NewNormalizer()
	got := n.Normalize(runtime.Event{Type: runtime.EventType("vendor.custom"), Content: "x"})
	if got.Kind != KindOther {
		t.Fatalf("expected KindOther, got %v", got.Kind)
	}
}
