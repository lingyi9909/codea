package parity

import (
	"context"
	"testing"
	"time"

	"codea/tui/internal/runtime"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

func TestRunnerIgnoresForeignSessionTerminalEvents(t *testing.T) {
	baseline := fakeruntime.New()
	baseline.Events = []runtime.Event{
		{SessionID: "foreign-session", Type: runtime.EventType("step.finished")},
		{SessionID: "fake-session-1", Type: runtime.EventType("answer.delta"), Content: "hello"},
		{SessionID: "fake-session-1", Type: runtime.EventType("step.finished")},
	}
	candidate := fakeruntime.New()
	candidate.Events = []runtime.Event{
		{SessionID: "foreign-session", Type: runtime.EventType("step.finished")},
		{SessionID: "fake-session-1", Type: runtime.EventType("answer.delta"), Content: "hello"},
		{SessionID: "fake-session-1", Type: runtime.EventType("step.finished")},
	}

	runner := Runner{Baseline: baseline, Candidate: candidate}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := runner.Run(ctx, Scenario{
		Name:     "Prompt",
		Required: true,
		Prompt: &runtime.PromptRequest{
			Agent: "general",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "hello"}},
		},
		Assertions: Assertion{RequireAnswer: true},
	})
	if !result.Passed {
		t.Fatalf("foreign-session terminal event must not terminate current scenario: %v", result.Failures)
	}
}

func TestRunnerIgnoresForeignSessionRawNoise(t *testing.T) {
	baseline := fakeruntime.New()
	baseline.Events = []runtime.Event{
		{SessionID: "foreign-session", Type: runtime.EventType("raw"), RawType: "session.diff", Raw: []byte(`{"foreign":true}`)},
		{SessionID: "fake-session-1", Type: runtime.EventType("raw"), RawType: "session.idle", Raw: []byte(`{"current":true}`)},
	}
	candidate := fakeruntime.New()
	candidate.Events = []runtime.Event{
		{SessionID: "fake-session-1", Type: runtime.EventType("raw"), RawType: "session.idle", Raw: []byte(`{"current":true}`)},
	}

	runner := Runner{Baseline: baseline, Candidate: candidate}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := runner.Run(ctx, Scenario{
		Name:     "RawEventHandling",
		Required: true,
		Prompt: &runtime.PromptRequest{
			Agent: "general",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "raw event test"}},
		},
		Assertions: Assertion{RequireRaw: true},
	})
	if !result.Passed {
		t.Fatalf("foreign-session raw events must not affect parity fingerprint: %v", result.Failures)
	}
}
