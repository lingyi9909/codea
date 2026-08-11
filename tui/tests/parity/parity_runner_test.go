package parity_test

import (
	"context"
	"testing"
	"time"

	"codea/tui/internal/capability"
	"codea/tui/internal/parity"
	"codea/tui/internal/runtime"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

func TestAllV1RequiredScenariosPass(t *testing.T) {
	baseline := fakeruntime.New()
	baseline.HealthInfo = runtime.HealthInfo{Healthy: true, Version: "test"}
	baseline.Events = []runtime.Event{
		{Type: runtime.EventType("answer.delta"), Content: "ok"},
		{Type: runtime.EventType("step.finished")},
	}

	candidate := fakeruntime.New()
	candidate.HealthInfo = runtime.HealthInfo{Healthy: true, Version: "test"}
	candidate.Events = []runtime.Event{
		{Type: runtime.EventType("answer.delta"), Content: "ok"},
		{Type: runtime.EventType("step.finished")},
	}

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

func TestCapabilityCompareWithParity(t *testing.T) {
	// Load real product requirements.
	inv, err := capability.Load("../../../runtime/capabilities.yaml")
	if err != nil {
		t.Fatalf("load capabilities.yaml: %v", err)
	}

	// Simulate OpenCodeAdapter capabilities.
	caps := runtime.RuntimeCapabilities{
		Sessions:          true,
		Streaming:         true,
		Reasoning:         true,
		FileRead:          true,
		FileWrite:         true,
		Edit:              true,
		Bash:              true,
		ToolApproval:      true,
		Agents:            true,
		Subagents:         true,
		Skills:            true,
		Plugins:           true,
		Abort:             true,
		MessageHistory:    true,
		ContextCompaction: true,
	}

	result := inv.Compare(caps)
	if result.HasRequiredFailures() {
		t.Errorf("all required capabilities should be supported, missing: %v", result.RequiredMissing)
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
		{Type: runtime.EventType("answer.delta"), Content: "a"},
		{Type: runtime.EventType("step.finished")},
	}

	candidate := fakeruntime.New()
	// Missing one event — silent loss.
	candidate.Events = []runtime.Event{
		{Type: runtime.EventType("answer.delta"), Content: "a"},
	}

	runner := parity.Runner{Baseline: baseline, Candidate: candidate}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := runner.Run(ctx, parity.Scenario{
		Name:     "Prompt",
		Required: true,
		Prompt: &runtime.PromptRequest{
			Agent: "general",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "test"}},
		},
	})

	if result.Passed {
		t.Error("required scenario with silent loss must fail")
	}
	if !result.SilentLoss {
		t.Error("must detect silent loss")
	}
}
