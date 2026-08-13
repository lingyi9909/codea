package reasoning

import (
	"time"

	"codea/tui/internal/runtime"
)

// eventTypeSessionError and eventTypeRuntimeError are recognized to interrupt
// any in-progress reasoning (design rule: error/abort ends reasoning).
const (
	eventTypeSessionError runtime.EventType = "session.error"
	eventTypeRuntimeError runtime.EventType = "runtime.error"
	eventTypeStepStarted  runtime.EventType = "step.started"
)

// source tracks where the active reasoning block originated, so the processor
// can de-duplicate structured reasoning against <think> fallback content.
type source int

const (
	sourceNone source = iota
	sourceStructured
	sourceFallback
)

// EventKind classifies processor output events consumed by the TUI.
type EventKind int

const (
	// EventReasoningStart marks the beginning of a reasoning block.
	EventReasoningStart EventKind = iota
	// EventReasoningDelta carries an incremental reasoning chunk.
	EventReasoningDelta
	// EventReasoningEnd marks the end of a reasoning block.
	EventReasoningEnd
	// EventAnswerDelta carries an incremental answer chunk with reasoning tags
	// already stripped.
	EventAnswerDelta
)

// Event is one unit of processor output. The TUI renders these in order and
// may additionally read Snapshot for the full reasoning lifecycle.
type Event struct {
	Kind        EventKind
	Content     string
	// Duration is set on EventReasoningEnd to the completed block's duration.
	Duration time.Duration
	// Interrupted is true on EventReasoningEnd when the block ended due to
	// error/abort rather than normally.
	Interrupted bool
}

// Processor consumes runtime events and produces a normalized stream of
// reasoning/answer events, de-duplicating structured reasoning against
// <think> fallback content. It is the single entry point for Task 7.
type Processor struct {
	normalizer     *Normalizer
	tagParser      *TagParser
	tracker        *Tracker
	// structuredSeen suppresses <think> fallback content for the entire
	// answer/reasoning cycle once structured reasoning has been observed. It is
	// reset only on step.started (the cycle boundary), never on answer deltas.
	structuredSeen bool
	activeSource   source
}

// NewProcessor returns a Processor. Options are forwarded to the underlying
// Tracker (e.g. WithClock for deterministic duration tests).
func NewProcessor(opts ...Option) *Processor {
	return &Processor{
		normalizer: NewNormalizer(),
		tagParser:  NewTagParser(),
		tracker:    NewTracker(opts...),
	}
}

// Process consumes one runtime event and returns the events it produced.
func (p *Processor) Process(ev runtime.Event) []Event {
	switch p.normalizer.Normalize(ev).Kind {
	case KindReasoning:
		return p.handleStructuredReasoning(ev.Content)
	case KindText:
		return p.handleText(ev.Content)
	default:
		return p.handleOther(ev)
	}
}

// Flush finalizes the stream, closing any unterminated reasoning block.
func (p *Processor) Flush() []Event {
	return p.finalize(false)
}

// Snapshot returns the current full reasoning lifecycle state.
func (p *Processor) Snapshot() Snapshot {
	return p.tracker.Snapshot()
}

// Reset clears all processor state for reuse across sessions.
func (p *Processor) Reset() {
	p.tracker.Reset()
	p.tagParser = NewTagParser()
	p.structuredSeen = false
	p.activeSource = sourceNone
}

func (p *Processor) hasActive() bool {
	_, ok := p.tracker.Active()
	return ok
}

func (p *Processor) handleStructuredReasoning(content string) []Event {
	if content == "" {
		return nil
	}
	p.structuredSeen = true
	p.activeSource = sourceStructured
	return p.appendReasoning(content)
}

func (p *Processor) handleText(content string) []Event {
	var out []Event

	// A text part after structured reasoning ends that reasoning (design rule).
	if p.activeSource == sourceStructured {
		out = append(out, p.endReasoning(false)...)
	}

	for _, pe := range p.tagParser.Feed(content) {
		switch pe.Type {
		case ParserEventAnswerDelta:
			out = append(out, Event{Kind: EventAnswerDelta, Content: pe.Content})
		case ParserEventReasoningStart:
			// Deferred: a block starts on its first non-empty delta so an empty
			// <think></think> never produces a garbage block.
		case ParserEventReasoningDelta:
			if p.structuredSeen {
				// De-duplicate: structured reasoning already supplied the
				// reasoning; strip the fallback <think> without re-emitting.
				continue
			}
			p.activeSource = sourceFallback
			out = append(out, p.appendReasoning(pe.Content)...)
		case ParserEventReasoningEnd:
			if p.activeSource == sourceFallback {
				out = append(out, p.endReasoning(false)...)
			}
		}
	}
	return out
}

func (p *Processor) handleOther(ev runtime.Event) []Event {
	switch ev.Type {
	case eventTypeSessionError, eventTypeRuntimeError:
		return p.finalize(true)
	case eventTypeStepStarted:
		out := p.finalize(false)
		p.structuredSeen = false
		return out
	}
	return nil
}

func (p *Processor) appendReasoning(content string) []Event {
	if content == "" {
		return nil
	}
	var out []Event
	if !p.hasActive() {
		p.tracker.Start()
		out = append(out, Event{Kind: EventReasoningStart})
	}
	p.tracker.Append(content)
	out = append(out, Event{Kind: EventReasoningDelta, Content: content})
	return out
}

func (p *Processor) endReasoning(interrupted bool) []Event {
	if !p.hasActive() {
		return nil
	}
	var out []Event
	var b Block
	var ok bool
	if interrupted {
		b, ok = p.tracker.Interrupt()
	} else {
		b, ok = p.tracker.End()
	}
	if ok {
		out = append(out, Event{Kind: EventReasoningEnd, Duration: b.Duration, Interrupted: interrupted})
	}
	p.activeSource = sourceNone
	return out
}

func (p *Processor) finalize(interrupted bool) []Event {
	var out []Event
	for _, pe := range p.tagParser.Flush() {
		switch pe.Type {
		case ParserEventAnswerDelta:
			out = append(out, Event{Kind: EventAnswerDelta, Content: pe.Content})
		case ParserEventReasoningDelta:
			if !p.structuredSeen {
				p.activeSource = sourceFallback
				out = append(out, p.appendReasoning(pe.Content)...)
			}
		}
	}
	out = append(out, p.endReasoning(interrupted)...)
	return out
}
