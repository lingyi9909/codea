package opencode

import (
	"encoding/json"

	"codea/tui/internal/runtime"
)

// eventMapState retains the vendor part kind across the OpenCode event stream.
// OpenCode v1.18.11 emits message.part.updated with part.type=reasoning/text,
// then message.part.delta with field=text for both kinds. The part identity is
// therefore required to classify the later text delta correctly.
type eventMapState struct {
	partTypes map[eventPartKey]string
}

type eventPartKey struct {
	sessionID string
	messageID string
	partID    string
}

func newEventMapState() *eventMapState {
	return &eventMapState{partTypes: make(map[eventPartKey]string)}
}

func (s *eventMapState) Map(raw []byte, sequence int64) (runtime.Event, error) {
	event, err := MapEvent(raw, sequence)
	if err != nil {
		return event, err
	}

	var envelope sseEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return event, nil
	}
	var payload ssePayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		return event, nil
	}
	var props sseCommonProps
	_ = json.Unmarshal(payload.Properties, &props)

	switch payload.Type {
	case "message.part.updated":
		if props.Part == nil || props.Part.ID == "" {
			return event, nil
		}
		s.partTypes[eventPartKey{
			sessionID: props.Part.SessionID,
			messageID: props.Part.MessageID,
			partID:    props.Part.ID,
		}] = props.Part.Type

	case "message.part.delta":
		key := eventPartKey{
			sessionID: props.SessionID,
			messageID: props.MessageID,
			partID:    props.PartID,
		}
		switch s.partTypes[key] {
		case "reasoning":
			event.Type = CodeaEventReasoningDelta
		case "text":
			event.Type = CodeaEventAnswerDelta
		}

	case "message.part.removed":
		delete(s.partTypes, eventPartKey{
			sessionID: props.SessionID,
			messageID: props.MessageID,
			partID:    props.PartID,
		})
	}

	return event, nil
}
