package parity

import (
	"context"
	"testing"
	"time"

	"codea/tui/internal/runtime"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

func TestRunnerHealthParity(t *testing.T) {
	baseline := fakeruntime.New()
	baseline.HealthInfo = runtime.HealthInfo{Healthy: true, Version: "v1"}
	candidate := fakeruntime.New()
	candidate.HealthInfo = runtime.HealthInfo{Healthy: true, Version: "v1"}

	runner := Runner{Baseline: baseline, Candidate: candidate}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := runner.Run(ctx, Scenario{Name: "Health", Required: true})
	if !result.Passed {
		t.Errorf("health parity should pass, failures: %v", result.Failures)
	}
	if result.SilentLoss {
		t.Error("should not have silent loss")
	}
}

func TestRunnerHealthMismatchFails(t *testing.T) {
	baseline := fakeruntime.New()
	baseline.HealthInfo = runtime.HealthInfo{Healthy: true}
	candidate := fakeruntime.New()
	candidate.HealthError = fakeruntime.ErrSimulated

	runner := Runner{Baseline: baseline, Candidate: candidate}
	ctx := context.Background()

	result := runner.Run(ctx, Scenario{Name: "Health", Required: true})
	if result.Passed {
		t.Error("health mismatch should fail")
	}
	if len(result.Failures) == 0 {
		t.Error("should have failure reason")
	}
}

func TestRunnerCreateSessionParity(t *testing.T) {
	baseline := fakeruntime.New()
	candidate := fakeruntime.New()

	runner := Runner{Baseline: baseline, Candidate: candidate}
	ctx := context.Background()

	result := runner.Run(ctx, Scenario{Name: "CreateSession", Required: true})
	if !result.Passed {
		t.Errorf("create session parity should pass, failures: %v", result.Failures)
	}
}

func TestRunnerPromptEventParity(t *testing.T) {
	baseline := fakeruntime.New()
	baseline.Events = []runtime.Event{
		{Type: runtime.EventType("answer.delta"), Content: "hello"},
		{Type: runtime.EventType("step.finished")},
	}

	candidate := fakeruntime.New()
	candidate.Events = []runtime.Event{
		{Type: runtime.EventType("answer.delta"), Content: "hello"},
		{Type: runtime.EventType("step.finished")},
	}

	runner := Runner{Baseline: baseline, Candidate: candidate}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := runner.Run(ctx, Scenario{
		Name:     "Prompt",
		Required: true,
		Prompt: &runtime.PromptRequest{
			Agent: "general",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "test"}},
		},
	})

	if !result.Passed {
		t.Errorf("prompt event parity should pass, failures: %v", result.Failures)
	}
	if result.SilentLoss {
		t.Error("should not have silent loss")
	}
}

func TestRunnerSilentLossDetected(t *testing.T) {
	baseline := fakeruntime.New()
	baseline.Events = []runtime.Event{
		{Type: runtime.EventType("reasoning.delta"), Content: "thinking"},
		{Type: runtime.EventType("answer.delta"), Content: "hello"},
		{Type: runtime.EventType("step.finished")},
	}

	candidate := fakeruntime.New()
	// Candidate has same event count (3) but missing reasoning.delta — silent loss.
	candidate.Events = []runtime.Event{
		{Type: runtime.EventType("tool.called"), Tool: &runtime.ToolEvent{Name: "x", CallID: "1"}},
		{Type: runtime.EventType("answer.delta"), Content: "hello"},
		{Type: runtime.EventType("step.finished")},
	}

	runner := Runner{Baseline: baseline, Candidate: candidate}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := runner.Run(ctx, Scenario{
		Name:     "Reasoning",
		Required: true,
		Prompt: &runtime.PromptRequest{
			Agent: "general",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "test"}},
		},
		Assertions: Assertion{
			RequireReasoning: true,
			RequireAnswer:    true,
		},
	})

	if result.Passed {
		t.Error("silent loss should cause failure for required scenario")
	}
	if !result.SilentLoss {
		t.Error("should detect silent loss: reasoning missing in candidate despite same event count")
	}
}

func TestRunnerCancelParity(t *testing.T) {
	baseline := fakeruntime.New()
	candidate := fakeruntime.New()

	runner := Runner{Baseline: baseline, Candidate: candidate}
	ctx := context.Background()

	result := runner.Run(ctx, Scenario{Name: "Cancel", Required: true})
	if !result.Passed {
		t.Errorf("cancel parity should pass, failures: %v", result.Failures)
	}

	if len(baseline.CancelledSessions()) != 1 {
		t.Error("baseline should have 1 cancelled session")
	}
	if len(candidate.CancelledSessions()) != 1 {
		t.Error("candidate should have 1 cancelled session")
	}
}

func TestRunnerRunAll(t *testing.T) {
	baseline := fakeruntime.New()
	baseline.HealthInfo = runtime.HealthInfo{Healthy: true}
	candidate := fakeruntime.New()
	candidate.HealthInfo = runtime.HealthInfo{Healthy: true}

	runner := Runner{Baseline: baseline, Candidate: candidate}
	ctx := context.Background()

	result := runner.RunAll(ctx, []Scenario{
		{Name: "Health", Required: true},
		{Name: "CreateSession", Required: true},
		{Name: "Cancel", Required: true},
	})

	if result.Total != 3 {
		t.Errorf("expected 3 total, got %d", result.Total)
	}
	if result.RequiredFailed != 0 {
		t.Errorf("expected 0 required failed, got %d", result.RequiredFailed)
	}
}
