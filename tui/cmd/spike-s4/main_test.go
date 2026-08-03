package main

import (
	"strings"
	"testing"
)

func TestReadReasoningAndAnswerSeparatesStructuredParts(t *testing.T) {
	stream := strings.Join([]string{
		`data: {"directory":"/tmp/project","payload":{"type":"message.part.delta","properties":{"sessionID":"ses_target","messageID":"msg_1","partID":"prt_reason","field":"text","delta":"considering options"}}}`,
		``,
		`data: {"directory":"/tmp/project","payload":{"type":"message.part.updated","properties":{"part":{"id":"prt_reason","sessionID":"ses_target","messageID":"msg_1","type":"reasoning","text":"considering options"}}}}`,
		``,
		`data: {"directory":"/tmp/project","payload":{"type":"message.part.updated","properties":{"part":{"id":"prt_text","sessionID":"ses_target","messageID":"msg_1","type":"text","text":"final answer"}}}}`,
		``,
		`data: {"directory":"/tmp/project","payload":{"type":"session.status","properties":{"sessionID":"ses_target","status":{"type":"idle"}}}}`,
		``,
	}, "\n")

	result, _, err := readReasoningAndAnswer(strings.NewReader(stream), "ses_target")
	if err != nil {
		t.Fatalf("readReasoningAndAnswer returned error: %v", err)
	}
	if result.Reasoning != "considering options" {
		t.Fatalf("reasoning = %q", result.Reasoning)
	}
	if result.Answer != "final answer" {
		t.Fatalf("answer = %q", result.Answer)
	}
	if result.HasThinkTags {
		t.Fatal("structured reasoning unexpectedly reported <think> tags")
	}
}
