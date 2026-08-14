package parity_test

import (
	"context"
	"testing"
	"time"

	"codea/tui/internal/capability"
	"codea/tui/internal/opencode"
	"codea/tui/internal/parity"
	"codea/tui/internal/runtime"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

func TestAllV1RequiredScenariosPass(t *testing.T) {
	// Events must satisfy all semantic assertions in V1RequiredScenarios():
	// RequireAnswer, RequireReasoning, RequireTool, RequireApproval, RequireRaw.
	sharedEvents := []runtime.Event{
		{Type: runtime.EventType("reasoning.delta"), Content: "thinking..."},
		{Type: runtime.EventType("answer.delta"), Content: "ok"},
		{Type: runtime.EventType("tool.called"), Tool: &runtime.ToolEvent{
			Name: "read", CallID: "call-1",
		}},
		{Type: runtime.EventType("approval.requested"), Approval: &runtime.ApprovalRequest{
			ID: "approval-1", Permission: "bash",
		}},
		{Type: runtime.EventType("raw"), Raw: []byte(`{"foo":"bar"}`)},
		{Type: runtime.EventType("step.finished")},
	}

	baseline := fakeruntime.New()
	baseline.HealthInfo = runtime.HealthInfo{Healthy: true, Version: "test"}
	baseline.Events = sharedEvents

	candidate := fakeruntime.New()
	candidate.HealthInfo = runtime.HealthInfo{Healthy: true, Version: "test"}
	candidate.Events = sharedEvents

	runner := parity.Runner{Baseline: baseline, Candidate: candidate}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := runner.RunAll(ctx, parity.V1RequiredScenarios())

	if result.RequiredFailed > 0 {
		t.Errorf("all V1 required scenarios should pass, got %d required failures", result.RequiredFailed)
		for _, s := range result.Scenarios {
			if !s.Passed && s.Required {
				t.Errorf("  FAIL: %s — %v", s.Name, s.Failures)
			}
		}
	}
	if result.Total != len(parity.V1RequiredScenarios()) {
		t.Errorf("expected %d scenarios, got %d", len(parity.V1RequiredScenarios()), result.Total)
	}
}

func TestCapabilityCompareWithRealOpenCodeAdapter(t *testing.T) {
	// Load real product requirements from capabilities.yaml.
	inv, err := capability.Load("../../../runtime/capabilities.yaml")
	if err != nil {
		t.Fatalf("load capabilities.yaml: %v", err)
	}

	// Instantiate a real OpenCodeAdapter and call Capabilities() through the
	// AgentRuntime interface — not through the package-level helper function.
	adapter := opencode.NewOpenCodeAdapter("http://127.0.0.1:1", "", "")
	var rt runtime.AgentRuntime = adapter
	caps := rt.Capabilities()

	result := inv.Compare(caps)
	if result.HasRequiredFailures() {
		t.Errorf("OpenCodeAdapter.Capabilities() must satisfy all product requirements, missing: %v", result.RequiredMissing)
	}
	if len(result.RequiredSupported) != 15 {
		t.Errorf("expected 15 required supported, got %d", len(result.RequiredSupported))
	}
}

func TestCapabilityCompareMissingRequired(t *testing.T) {
	inv := &capability.Inventory{
		Requirements: []capability.CapabilityRequirement{
			{Name: "sessions", Level: capability.Required},
			{Name: "streaming", Level: capability.Required},
		},
	}

	caps := runtime.RuntimeCapabilities{
		Sessions:  true,
		Streaming: false,
	}

	result := inv.Compare(caps)
	if !result.HasRequiredFailures() {
		t.Error("should detect missing required capabilities")
	}
}

func TestParitySilentLossFailsRequired(t *testing.T) {
	baseline := fakeruntime.New()
	baseline.Events = []runtime.Event{
		{Type: runtime.EventType("reasoning.delta"), Content: "thinking"},
		{Type: runtime.EventType("answer.delta"), Content: "a"},
		{Type: runtime.EventType("step.finished")},
	}

	candidate := fakeruntime.New()
	// Same count (3) but missing reasoning.delta — semantic silent loss.
	candidate.Events = []runtime.Event{
		{Type: runtime.EventType("tool.called"), Tool: &runtime.ToolEvent{Name: "x", CallID: "1"}},
		{Type: runtime.EventType("answer.delta"), Content: "a"},
		{Type: runtime.EventType("step.finished")},
	}

	runner := parity.Runner{Baseline: baseline, Candidate: candidate}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := runner.Run(ctx, parity.Scenario{
		Name:     "Reasoning",
		Required: true,
		Prompt: &runtime.PromptRequest{
			Agent: "general",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "test"}},
		},
		Assertions: parity.Assertion{
			RequireReasoning: true,
		},
	})

	if result.Passed {
		t.Error("required scenario with silent loss must fail")
	}
	if !result.SilentLoss {
		t.Error("must detect silent loss: same event count but missing semantic event")
	}
}
