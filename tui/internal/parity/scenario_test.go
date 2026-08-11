package parity

import (
	"testing"

	"codea/tui/internal/runtime"
)

func TestScenarioDefaults(t *testing.T) {
	s := Scenario{
		Name:     "test-scenario",
		Required: true,
		// RepeatCount 0 means "use runner default" (1).
	}

	if s.Name != "test-scenario" {
		t.Errorf("Name mismatch")
	}
	if !s.Required {
		t.Error("should be required")
	}
	// Runner treats 0 as 1.
	if s.RepeatCount != 0 {
		t.Errorf("zero-value RepeatCount expected, got %d", s.RepeatCount)
	}
}

func TestScenarioWithRepeat(t *testing.T) {
	s := Scenario{
		Name:        "repeat-test",
		RepeatCount: 3,
	}
	if s.RepeatCount != 3 {
		t.Errorf("expected RepeatCount 3, got %d", s.RepeatCount)
	}
}

func TestScenarioResultPassed(t *testing.T) {
	sr := ScenarioResult{
		Name:     "test",
		Required: true,
		Passed:   true,
		Runs:     1,
	}
	if !sr.Passed {
		t.Error("should be passed")
	}
	if sr.Runs != 1 {
		t.Errorf("expected 1 run, got %d", sr.Runs)
	}
}

func TestScenarioResultSilentLoss(t *testing.T) {
	sr := ScenarioResult{
		Name:       "test",
		Required:   true,
		Passed:     false,
		SilentLoss: true,
		Failures:   []Failure{{Reason: "expected event not received"}},
	}
	if sr.Passed {
		t.Error("should not pass with silent loss")
	}
	if len(sr.Failures) != 1 {
		t.Errorf("expected 1 failure, got %d", len(sr.Failures))
	}
}

func TestResultTotalCounts(t *testing.T) {
	r := Result{
		Scenarios: []ScenarioResult{
			{Name: "a", Passed: true, Required: true},
			{Name: "b", Passed: false, Required: true},
			{Name: "c", Passed: true, Required: false},
			{Name: "d", Passed: false, Required: false},
		},
	}
	r.compute()

	if r.Total != 4 {
		t.Errorf("expected Total 4, got %d", r.Total)
	}
	if r.Passed != 2 {
		t.Errorf("expected Passed 2, got %d", r.Passed)
	}
	if r.Failed != 2 {
		t.Errorf("expected Failed 2, got %d", r.Failed)
	}
	if r.RequiredFailed != 1 {
		t.Errorf("expected RequiredFailed 1, got %d", r.RequiredFailed)
	}
}

func TestV1RequiredScenarios(t *testing.T) {
	scenarios := V1RequiredScenarios()

	if len(scenarios) < 10 {
		t.Fatalf("expected at least 10 V1 required scenarios, got %d", len(scenarios))
	}

	for _, s := range scenarios {
		if !s.Required {
			t.Errorf("V1 scenario %q must be required", s.Name)
		}
		if s.Name == "" {
			t.Error("scenario name must not be empty")
		}
		if s.RepeatCount < 1 {
			t.Errorf("scenario %q RepeatCount must be >= 1", s.Name)
		}
	}

	// Verify key scenarios are present.
	names := make(map[string]bool)
	for _, s := range scenarios {
		names[s.Name] = true
	}
	for _, name := range []string{
		"Health",
		"CreateSession",
		"Prompt",
		"Answer",
		"Reasoning",
		"ToolLifecycle",
		"Approval",
		"Reject",
		"Cancel",
		"RawEventHandling",
	} {
		if !names[name] {
			t.Errorf("missing V1 required scenario: %s", name)
		}
	}
}

func TestScenarioNilPrompt(t *testing.T) {
	// Scenarios like Health don't need a prompt.
	s := Scenario{
		Name:     "Health",
		Required: true,
	}
	if s.Prompt != nil {
		t.Error("prompt should be nil for Health scenario")
	}
}

func TestScenarioWithPrompt(t *testing.T) {
	req := &runtime.PromptRequest{
		MessageID: "msg-1",
		Agent:     "general",
		Parts:     []runtime.PromptPart{runtime.TextPart{ID: "1", Text: "test"}},
	}
	s := Scenario{
		Name:     "Prompt",
		Required: true,
		Prompt:   req,
	}
	if s.Prompt == nil {
		t.Fatal("prompt should not be nil")
	}
	if s.Prompt.MessageID != "msg-1" {
		t.Errorf("expected msg-1, got %s", s.Prompt.MessageID)
	}
}
