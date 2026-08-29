package opencode

import "testing"

func TestTask29MessageUpdatedMapsAssistantParentIdentity(t *testing.T) {
	raw := []byte(`{"directory":"/repo","payload":{"type":"message.updated","properties":{"info":{"id":"A1","sessionID":"s1","role":"assistant","parentID":"U1"}}}}`)
	ev, err := MapEvent(raw, 1)
	if err != nil {
		t.Fatal(err)
	}
	if ev.MessageID != "A1" || ev.SessionID != "s1" || ev.MessageRole != "assistant" || ev.ParentMessageID != "U1" {
		t.Fatalf("mapped message identity=%#v", ev)
	}
}
