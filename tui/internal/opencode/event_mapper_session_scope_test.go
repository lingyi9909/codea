package opencode

import "testing"

func TestMapEventBackfillsSessionIDFromNestedPart(t *testing.T) {
	raw := []byte(`{"directory":"/tmp","payload":{"type":"message.part.updated","properties":{"part":{"id":"p1","messageID":"m1","sessionID":"session-nested","type":"step-finish","reason":"stop"}}}}`)
	event, err := MapEvent(raw, 1)
	if err != nil {
		t.Fatalf("MapEvent: %v", err)
	}
	if event.SessionID != "session-nested" {
		t.Fatalf("SessionID = %q, want session-nested", event.SessionID)
	}
}

func TestMapEventKeepsExplicitTopLevelSessionID(t *testing.T) {
	raw := []byte(`{"directory":"/tmp","payload":{"type":"message.part.updated","properties":{"sessionID":"session-top","part":{"id":"p1","messageID":"m1","sessionID":"session-nested","type":"step-finish","reason":"stop"}}}}`)
	event, err := MapEvent(raw, 1)
	if err != nil {
		t.Fatalf("MapEvent: %v", err)
	}
	if event.SessionID != "session-top" {
		t.Fatalf("SessionID = %q, want session-top", event.SessionID)
	}
}
