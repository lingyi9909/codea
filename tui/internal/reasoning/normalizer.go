package reasoning

import "codea/tui/internal/runtime"

// Codea domain event types recognized by the reasoning layer. These string
// values are Codea semantics, not OpenCode vendor DTOs; they mirror the
// constants in the opencode package (CodeaEventReasoningDelta and
// CodeaEventAnswerDelta) and are asserted to match in the contract test.
const (
	eventTypeReasoningDelta runtime.EventType = "reasoning.delta"
	eventTypeAnswerDelta    runtime.EventType = "answer.delta"
)

// Kind classifies a runtime event from the reasoning layer's perspective.
type Kind int

const (
	// KindReasoning is a structured reasoning delta emitted by the runtime.
	KindReasoning Kind = iota
	// KindText is a text (answer) delta, which may embed <think> tags.
	KindText
	// KindOther is any event unrelated to reasoning or answer content
	// (tool, raw, session, error, etc.).
	KindOther
)

// Normalized is the result of normalizing one runtime event.
type Normalized struct {
	Kind    Kind
	Content string
}

// Normalizer maps a runtime.Event into reasoning semantics. It consumes only
// the Codea runtime domain types and carries no vendor dependency.
type Normalizer struct{}

// NewNormalizer returns a Normalizer.
func NewNormalizer() *Normalizer {
	return &Normalizer{}
}

// Normalize classifies ev. Structured reasoning deltas map to KindReasoning,
// answer deltas to KindText, and everything else to KindOther.
func (n *Normalizer) Normalize(ev runtime.Event) Normalized {
	switch ev.Type {
	case eventTypeReasoningDelta:
		return Normalized{Kind: KindReasoning, Content: ev.Content}
	case eventTypeAnswerDelta:
		return Normalized{Kind: KindText, Content: ev.Content}
	default:
		return Normalized{Kind: KindOther}
	}
}
