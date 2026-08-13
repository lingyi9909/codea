package reasoning

import "strings"

const (
	openTag  = "<think>"
	closeTag = "</think>"
)

// ThinkState is the streaming state of the tag parser.
type ThinkState int

const (
	// ThinkStateAnswer means we are consuming answer text, watching for <think>.
	ThinkStateAnswer ThinkState = iota
	// ThinkStateReasoning means we are consuming reasoning text, watching for
	// </think>.
	ThinkStateReasoning
)

// ParserEventType classifies events emitted by the tag parser.
type ParserEventType int

const (
	// ParserEventAnswerDelta is answer text with no reasoning tags.
	ParserEventAnswerDelta ParserEventType = iota
	// ParserEventReasoningStart marks the beginning of a <think> block.
	ParserEventReasoningStart
	// ParserEventReasoningDelta is reasoning text inside a <think> block.
	ParserEventReasoningDelta
	// ParserEventReasoningEnd marks the end of a <think> block.
	ParserEventReasoningEnd
)

// ParserEvent is a single unit of output from the tag parser.
type ParserEvent struct {
	Type    ParserEventType
	Content string
}

// TagParser is a stateful streaming parser that separates <think>…</think>
// reasoning blocks from answer text. It handles tags split across chunk
// boundaries and only ever buffers a partial tag prefix, never an unbounded
// answer.
type TagParser struct {
	state   ThinkState
	pending string
}

// NewTagParser returns a TagParser in the answer state.
func NewTagParser() *TagParser {
	return &TagParser{state: ThinkStateAnswer}
}

// Feed consumes a chunk and returns the events it produced.
func (p *TagParser) Feed(chunk string) []ParserEvent {
	if chunk == "" {
		return nil
	}
	p.pending += chunk

	var events []ParserEvent
	for p.pending != "" {
		var advanced bool
		if p.state == ThinkStateAnswer {
			advanced = p.stepAnswer(&events)
		} else {
			advanced = p.stepReasoning(&events)
		}
		if !advanced {
			break
		}
	}
	return events
}

// Flush finalizes the stream. Any unterminated <think> block is closed safely
// so a truncated stream never panics or leaves reasoning open.
func (p *TagParser) Flush() []ParserEvent {
	var events []ParserEvent
	switch p.state {
	case ThinkStateReasoning:
		if p.pending != "" {
			events = append(events, ParserEvent{Type: ParserEventReasoningDelta, Content: p.pending})
		}
		events = append(events, ParserEvent{Type: ParserEventReasoningEnd})
		p.state = ThinkStateAnswer
	default:
		if p.pending != "" {
			events = append(events, ParserEvent{Type: ParserEventAnswerDelta, Content: p.pending})
		}
	}
	p.pending = ""
	return events
}

// IsInReasoning reports whether the parser is currently inside a <think> block.
func (p *TagParser) IsInReasoning() bool {
	return p.state == ThinkStateReasoning
}

// stepAnswer processes pending in the answer state. It returns false when more
// input is needed because pending ends with a partial <think> prefix.
func (p *TagParser) stepAnswer(events *[]ParserEvent) bool {
	if idx := strings.Index(p.pending, openTag); idx >= 0 {
		if idx > 0 {
			*events = append(*events, ParserEvent{Type: ParserEventAnswerDelta, Content: p.pending[:idx]})
		}
		*events = append(*events, ParserEvent{Type: ParserEventReasoningStart})
		p.pending = p.pending[idx+len(openTag):]
		p.state = ThinkStateReasoning
		return true
	}

	if keep := longestPrefixSuffix(p.pending, openTag); keep > 0 {
		if cut := len(p.pending) - keep; cut > 0 {
			*events = append(*events, ParserEvent{Type: ParserEventAnswerDelta, Content: p.pending[:cut]})
			p.pending = p.pending[cut:]
		}
		return false
	}

	*events = append(*events, ParserEvent{Type: ParserEventAnswerDelta, Content: p.pending})
	p.pending = ""
	return false
}

// stepReasoning processes pending in the reasoning state. It returns false when
// more input is needed because pending ends with a partial </think> prefix.
func (p *TagParser) stepReasoning(events *[]ParserEvent) bool {
	if idx := strings.Index(p.pending, closeTag); idx >= 0 {
		if idx > 0 {
			*events = append(*events, ParserEvent{Type: ParserEventReasoningDelta, Content: p.pending[:idx]})
		}
		*events = append(*events, ParserEvent{Type: ParserEventReasoningEnd})
		p.pending = p.pending[idx+len(closeTag):]
		p.state = ThinkStateAnswer
		return true
	}

	if keep := longestPrefixSuffix(p.pending, closeTag); keep > 0 {
		if cut := len(p.pending) - keep; cut > 0 {
			*events = append(*events, ParserEvent{Type: ParserEventReasoningDelta, Content: p.pending[:cut]})
			p.pending = p.pending[cut:]
		}
		return false
	}

	*events = append(*events, ParserEvent{Type: ParserEventReasoningDelta, Content: p.pending})
	p.pending = ""
	return false
}

// longestPrefixSuffix returns the length of the longest non-empty suffix of s
// that is a proper prefix of tag, or 0 if none. A full tag is handled by the
// callers via strings.Index, so only lengths < len(tag) are considered.
func longestPrefixSuffix(s, tag string) int {
	max := len(tag) - 1
	if len(s) < max {
		max = len(s)
	}
	for l := max; l >= 1; l-- {
		if strings.HasPrefix(tag, s[len(s)-l:]) {
			return l
		}
	}
	return 0
}
