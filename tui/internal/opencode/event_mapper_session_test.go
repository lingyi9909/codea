package opencode

import "testing"

func TestEventMapperUsesNestedPartSessionIDWhenTopLevelMissing(t *testing.T) {
	raw := `{"directory":"/tmp","payload":{"type":"message.part.updated","properties":{"part":{"id":"p1","messageID":"m1","sessionID":"s-nested","type":"step-finish","reason":"stop"}}}}`
	event, err := MapEvent([]byte(raw), 1)
	if err != nil {
		t.Fatal(err)
	}
	if event.SessionID != "s-nested" {
		t.Fatalf("SessionID=%q, want nested part session id", event.SessionID)
	}
}
