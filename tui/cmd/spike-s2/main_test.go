package main

import (
	"strings"
	"testing"
)

func TestReadEventsStopsAtTargetSessionIdle(t *testing.T) {
	stream := strings.NewReader("data: {\"directory\":\"/tmp/project\",\"payload\":{\"type\":\"message.part.updated\",\"properties\":{\"part\":{\"sessionID\":\"ses_target\",\"type\":\"text\",\"text\":\"hello\"}}}}\n\n" +
		"data: {\"directory\":\"/tmp/project\",\"payload\":{\"type\":\"session.status\",\"properties\":{\"sessionID\":\"ses_other\",\"status\":{\"type\":\"idle\"}}}}\n\n" +
		"data: {\"directory\":\"/tmp/project\",\"payload\":{\"type\":\"session.status\",\"properties\":{\"sessionID\":\"ses_target\",\"status\":{\"type\":\"idle\"}}}}\n\n")

	events, err := readEvents(stream, "ses_target")
	if err != nil {
		t.Fatalf("readEvents returned error: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}
	if events[0].Payload.Type != "message.part.updated" {
		t.Fatalf("unexpected first event type: %s", events[0].Payload.Type)
	}
}

func TestReadEventsRejectsMalformedJSON(t *testing.T) {
	_, err := readEvents(strings.NewReader("data: not-json\n\n"), "ses_target")
	if err == nil {
		t.Fatal("expected malformed SSE data to return an error")
	}
}
