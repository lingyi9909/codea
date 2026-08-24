package parity

import (
	"context"
	"testing"
	"time"

	"codea/tui/internal/runtime"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

func TestTaskParityIgnoresNonSemanticLifecycleEventPresence(t *testing.T) {
	baseline := fakeruntime.New()
	baseline.Events = []runtime.Event{
		{Type: runtime.EventType("session.updated")},
		{Type: runtime.EventType("message.updated")},
		{Type: runtime.EventType("session.diff")},
		{Type: runtime.EventType("answer.delta"), Content: "ok"},
	}
	candidate := fakeruntime.New()
	candidate.Events = []runtime.Event{
		{Type: runtime.EventType("answer.delta"), Content: "ok"},
	}

	runner := Runner{Baseline: baseline, Candidate: candidate}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	result := runner.Run(ctx, Scenario{
		Name: "Prompt", Required: true, RepeatCount: 1,
		Prompt: &runtime.PromptRequest{Agent: "general", Parts: []runtime.PromptPart{runtime.TextPart{Text: "hello"}}},
		Assertions: Assertion{RequireAnswer: true},
	})
	if !result.Passed {
		t.Fatalf("non-semantic OpenCode lifecycle timing must not fail task-effect parity: %+v", result.Failures)
	}
	if result.SilentLoss {
		t.Fatal("non-semantic lifecycle timing must not be classified as silent task loss")
	}
}
