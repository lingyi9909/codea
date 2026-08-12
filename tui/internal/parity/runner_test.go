package parity

import (
	"context"
	"strings"
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

func TestRunnerBothHealthFail(t *testing.T) {
	baseline := fakeruntime.New()
	baseline.HealthError = fakeruntime.ErrSimulated
	candidate := fakeruntime.New()
	candidate.HealthError = fakeruntime.ErrSimulated

	runner := Runner{Baseline: baseline, Candidate: candidate}
	ctx := context.Background()

	result := runner.Run(ctx, Scenario{Name: "Health", Required: true})
	if result.Passed {
		t.Error("both sides failing Health must NOT pass")
	}
	if len(result.Failures) == 0 {
		t.Error("should have failure reason")
	}
}

func TestRunnerBothCreateSessionFail(t *testing.T) {
	baseline := fakeruntime.New()
	baseline.SessionErr = fakeruntime.ErrSimulated
	candidate := fakeruntime.New()
	candidate.SessionErr = fakeruntime.ErrSimulated

	runner := Runner{Baseline: baseline, Candidate: candidate}
	ctx := context.Background()

	result := runner.Run(ctx, Scenario{Name: "CreateSession", Required: true})
	if result.Passed {
		t.Error("both sides failing CreateSession must NOT pass")
	}
}

func TestRunnerAgentSelectionVerified(t *testing.T) {
	// The runner must check the runtime's recorded prompt — not the scenario's
	// own Prompt.Agent field — to verify that the required agent was actually
	// delivered to and received by the runtime.
	baseline := fakeruntime.New()
	baseline.Events = []runtime.Event{
		{Type: runtime.EventType("answer.delta"), Content: "review"},
		{Type: runtime.EventType("step.finished")},
	}
	candidate := fakeruntime.New()
	candidate.Events = []runtime.Event{
		{Type: runtime.EventType("answer.delta"), Content: "review"},
		{Type: runtime.EventType("step.finished")},
	}

	runner := Runner{Baseline: baseline, Candidate: candidate}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := runner.Run(ctx, Scenario{
		Name:     "AgentSelection",
		Required: true,
		Prompt: &runtime.PromptRequest{
			Agent: "reviewer",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "review"}},
		},
		Assertions: Assertion{RequireAnswer: true, RequireAgent: "reviewer"},
	})

	if !result.Passed {
		t.Errorf("agent selection should pass when runtime receives correct agent, failures: %v", result.Failures)
	}

	// Verify the runtime actually received the agent in the request.
	agent, ok := baseline.LastPrompt()
	if !ok || agent != "reviewer" {
		t.Errorf("baseline should have recorded agent=reviewer, got agent=%q ok=%v", agent, ok)
	}
	agent, ok = candidate.LastPrompt()
	if !ok || agent != "reviewer" {
		t.Errorf("candidate should have recorded agent=reviewer, got agent=%q ok=%v", agent, ok)
	}
}

func TestRunnerAgentSelectionMismatch(t *testing.T) {
	// When the scenario sends agent="general" but RequireAgent="reviewer",
	// the runner checks what each runtime *actually received* (via
	// PromptRecorder) and fails — not because the scenario disagrees with
	// itself, but because the runtime-recorded agent differs from the
	// required agent.
	baseline := fakeruntime.New()
	baseline.Events = []runtime.Event{
		{Type: runtime.EventType("answer.delta"), Content: "ok"},
		{Type: runtime.EventType("step.finished")},
	}
	candidate := fakeruntime.New()
	candidate.Events = []runtime.Event{
		{Type: runtime.EventType("answer.delta"), Content: "ok"},
		{Type: runtime.EventType("step.finished")},
	}

	runner := Runner{Baseline: baseline, Candidate: candidate}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Prompt.Agent is "general" but RequireAgent expects "reviewer".
	// The old circular check compared s.Prompt.Agent == RequireAgent.
	// The new check compares runtime-recorded agent == RequireAgent,
	// so this correctly fails on the baseline (runtime received "general").
	result := runner.Run(ctx, Scenario{
		Name:     "AgentSelection",
		Required: true,
		Prompt: &runtime.PromptRequest{
			Agent: "general",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "review"}},
		},
		Assertions: Assertion{RequireAnswer: true, RequireAgent: "reviewer"},
	})

	if result.Passed {
		t.Error("agent selection should fail when runtime receives wrong agent (general != reviewer)")
	}

	// Verify baseline recorded "general" — proving we're checking the
	// runtime's actual received value, not the scenario's self-reference.
	agent, ok := baseline.LastPrompt()
	if !ok {
		t.Error("baseline should have recorded a prompt")
	}
	if agent != "general" {
		t.Errorf("baseline should have received agent=general, got %q", agent)
	}
	t.Logf("failures: %v", result.Failures)
}

func TestRunnerAgentSelectionRequiresPromptRecorder(t *testing.T) {
	// When RequireAgent is set but a runtime doesn't implement PromptRecorder,
	// the assertion cannot be verified and must FAIL — not silently pass.
	// We simulate this with a minimal runtime that wraps FakeRuntime.
	baseline := fakeruntime.New()
	baseline.Events = []runtime.Event{
		{Type: runtime.EventType("answer.delta"), Content: "ok"},
		{Type: runtime.EventType("step.finished")},
	}
	candidate := fakeruntime.New()
	candidate.Events = []runtime.Event{
		{Type: runtime.EventType("answer.delta"), Content: "ok"},
		{Type: runtime.EventType("step.finished")},
	}

	// noRecRuntime wraps AgentRuntime but deliberately does NOT implement
	// PromptRecorder, simulating a real adapter that lacks observability.
	runner := Runner{Baseline: baseline, Candidate: &noRecRuntime{AgentRuntime: candidate}}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := runner.Run(ctx, Scenario{
		Name:     "AgentSelection",
		Required: true,
		Prompt: &runtime.PromptRequest{
			Agent: "reviewer",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "review"}},
		},
		Assertions: Assertion{RequireAnswer: true, RequireAgent: "reviewer"},
	})

	if result.Passed {
		t.Error("agent selection must FAIL when PromptRecorder is not implemented")
	}
	found := false
	for _, f := range result.Failures {
		if strings.Contains(f.Reason, "PromptRecorder") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("failure reason should mention PromptRecorder, got: %v", result.Failures)
	}
}

// noRecRuntime wraps an AgentRuntime but does NOT implement PromptRecorder.
type noRecRuntime struct{ runtime.AgentRuntime }

func TestRunnerSlowFirstEventSucceeds(t *testing.T) {
	// Regression: the inactivity timer must not start until the first event
	// arrives. A runtime whose first event takes >inactivityFallback (500ms)
	// must still succeed within the total deadline.
	baseline := fakeruntime.New()
	baseline.Events = []runtime.Event{
		{Type: runtime.EventType("answer.delta"), Content: "hello"},
		{Type: runtime.EventType("step.finished")},
	}
	candidate := fakeruntime.New()
	candidate.EventDelay = 700 * time.Millisecond
	candidate.Events = []runtime.Event{
		{Type: runtime.EventType("answer.delta"), Content: "hello"},
		{Type: runtime.EventType("step.finished")},
	}

	runner := Runner{Baseline: baseline, Candidate: candidate}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := runner.Run(ctx, Scenario{
		Name:     "Streaming",
		Required: true,
		Timeout:  2 * time.Second,
		Prompt: &runtime.PromptRequest{
			Agent: "general",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "test"}},
		},
		Assertions: Assertion{RequireAnswer: true},
	})

	if !result.Passed {
		t.Errorf("slow first event (700ms) should still succeed within 2s timeout, failures: %v", result.Failures)
	}
	if result.SilentLoss {
		t.Error("should not have silent loss")
	}
}

func TestRunnerBothCancelFail(t *testing.T) {
	baseline := fakeruntime.New()
	baseline.CancelError = fakeruntime.ErrSimulated
	candidate := fakeruntime.New()
	candidate.CancelError = fakeruntime.ErrSimulated

	runner := Runner{Baseline: baseline, Candidate: candidate}
	ctx := context.Background()

	result := runner.Run(ctx, Scenario{Name: "Cancel", Required: true})
	if result.Passed {
		t.Error("both sides failing Cancel must NOT pass")
	}
	if len(result.Failures) == 0 {
		t.Error("should have failure reason")
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
