package opencode

import (
	"encoding/json"

	"codea/tui/internal/runtime"
)

const maxRawSize = 16 * 1024

// sseEnvelope is the top-level SSE event envelope from OpenCode.
type sseEnvelope struct {
	Directory string          `json:"directory"`
	Payload   json.RawMessage `json:"payload"`
}

// ssePayload is the payload within the envelope.
type ssePayload struct {
	Type       string          `json:"type"`
	Properties json.RawMessage `json:"properties"`
}

// sseProperties is a partial parse of common event properties.
type sseProperties struct {
	SessionID string `json:"sessionID"`
	MessageID string `json:"messageID"`
	PartID    string `json:"partID"`
	Field     string `json:"field"`
	Delta     string `json:"delta"`
	Text      string `json:"text"`
}

// MapEvent maps a raw OpenCode SSE event to a Codea runtime.Event.
// Sequence is the monotonically increasing connection-level sequence number.
func MapEvent(raw []byte, sequence int64) (runtime.Event, error) {
	rawSize := len(raw)

	var envelope sseEnvelope
	if err := json.Unmarshal(raw, &envelope); err != nil {
		rawPayload := trimRaw(raw)
		return runtime.Event{
			Type:            "_unparseable_",
			Sequence:        sequence,
			Raw:             rawPayload,
			RawType:         "_unparseable_",
			RawTruncated:    len(rawPayload) < rawSize,
			RawOriginalSize: rawSize,
		}, nil
	}

	var payload ssePayload
	if err := json.Unmarshal(envelope.Payload, &payload); err != nil {
		rawPayload := trimRaw(raw)
		return runtime.Event{
			Type:            "_unparseable_",
			Sequence:        sequence,
			Raw:             rawPayload,
			RawType:         "_unparseable_",
			RawTruncated:    len(rawPayload) < rawSize,
			RawOriginalSize: rawSize,
		}, nil
	}

	var props sseProperties
	_ = json.Unmarshal(payload.Properties, &props)

	event := runtime.Event{
		Type:     runtime.EventType(payload.Type),
		Sequence: sequence,
		RawType:  payload.Type,
	}

	rawPayload := trimRaw(raw)
	event.Raw = rawPayload
	if len(rawPayload) < rawSize {
		event.RawTruncated = true
		event.RawOriginalSize = rawSize
	}

	if props.SessionID != "" {
		event.SessionID = props.SessionID
	}
	if props.MessageID != "" {
		event.MessageID = props.MessageID
	}
	if props.PartID != "" {
		event.PartID = props.PartID
	}
	if props.Delta != "" {
		event.Content = props.Delta
	} else if props.Text != "" {
		event.Content = props.Text
	}

	return event, nil
}

func trimRaw(raw []byte) []byte {
	if len(raw) <= maxRawSize {
		return raw
	}
	return raw[:maxRawSize]
}
