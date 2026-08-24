package opencode

import "encoding/json"

// UnmarshalJSON preserves the locked OpenCode v1.18.11 event shape while
// normalizing session correlation. Some message.part.updated events carry the
// session only under part.sessionID; parity and application consumers need the
// same Event.SessionID regardless of whether OpenCode emitted the top-level
// sessionID field.
func (p *sseCommonProps) UnmarshalJSON(data []byte) error {
	type rawSSECommonProps sseCommonProps
	var decoded rawSSECommonProps
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}
	*p = sseCommonProps(decoded)
	if p.SessionID == "" && p.Part != nil && p.Part.SessionID != "" {
		p.SessionID = p.Part.SessionID
	}
	return nil
}
