package app

import (
	"errors"
	"strings"
	"testing"

	"codea/tui/internal/runtime"
)

func TestTaskStrategyGeneralEmitsCodeaOwnedControl(t *testing.T) {
	part, ok := taskStrategyPart("general")
	if !ok {
		t.Fatal("general agent should receive Task 29 synthetic planning control")
	}
	if !part.Synthetic {
		t.Fatal("task strategy must be synthetic")
	}
	if part.Metadata["codea.kind"] != "task-strategy" {
		t.Fatalf("metadata=%#v", part.Metadata)
	}
	for _, required := range []string{
		"For read-only/explanatory work, do not create a plan.",
		"Before your first project mutation or project command execution, create task_plan.",
		"3–7 steps",
		"update task_step with evidence",
	} {
		if !strings.Contains(part.Text, required) {
			t.Fatalf("task strategy missing %q: %q", required, part.Text)
		}
	}
}

func TestTaskStrategyProfessionalAgentsDoNotReceiveGeneralControl(t *testing.T) {
	for _, agent := range []string{"code-reviewer", "debug", "unit-test-generator", "api-documentation"} {
		if _, ok := taskStrategyPart(agent); ok {
			t.Fatalf("professional agent %q must keep its own deterministic contract", agent)
		}
	}
}

func TestBuildRepoAwarePromptInjectsGeneralTaskStrategyWithoutChangingUserText(t *testing.T) {
	intent := repoPromptIntent{
		request: runtime.PromptRequest{MessageID: "m1", Agent: "general"},
		promptText: "Fix OrderService",
	}
	req := buildRepoAwarePrompt(intent, repoMapFixture(), nil)
	if req.Agent != "general" {
		t.Fatalf("agent=%q, want general", req.Agent)
	}
	if len(req.Parts) != 3 {
		t.Fatalf("parts=%#v, want repo-map + task-strategy + user", req.Parts)
	}
	strategy, ok := req.Parts[1].(runtime.TextPart)
	if !ok || !strategy.Synthetic || strategy.Metadata["codea.kind"] != "task-strategy" {
		t.Fatalf("strategy part=%#v", req.Parts[1])
	}
	user, ok := req.Parts[2].(runtime.TextPart)
	if !ok || user.Synthetic || user.Text != "Fix OrderService" {
		t.Fatalf("user part=%#v", req.Parts[2])
	}
}

func TestBuildRepoAwarePromptKeepsProfessionalRoutingAndNoGeneralStrategy(t *testing.T) {
	intent := repoPromptIntent{
		request: runtime.PromptRequest{MessageID: "m2", Agent: "code-reviewer"},
		promptText: "Review OrderService",
	}
	req := buildRepoAwarePrompt(intent, repoMapFixture(), nil)
	if req.Agent != "code-reviewer" {
		t.Fatalf("agent=%q, professional route must be preserved", req.Agent)
	}
	if len(req.Parts) != 2 {
		t.Fatalf("parts=%#v, want repo-map + user only", req.Parts)
	}
	for _, raw := range req.Parts {
		part, ok := raw.(runtime.TextPart)
		if ok && part.Metadata["codea.kind"] == "task-strategy" {
			t.Fatal("professional prompt must not receive General task strategy")
		}
	}
}

func TestTaskStrategySurvivesRepoContextFailure(t *testing.T) {
	intent := repoPromptIntent{
		request: runtime.PromptRequest{MessageID: "m3", Agent: "general"},
		promptText: "Fix OrderService",
	}
	req := buildRepoAwarePrompt(intent, repoMapFixture(), errors.New("index unavailable"))
	if len(req.Parts) != 2 {
		t.Fatalf("parts=%#v, want task-strategy + user when repo context fails", req.Parts)
	}
	strategy, ok := req.Parts[0].(runtime.TextPart)
	if !ok || strategy.Metadata["codea.kind"] != "task-strategy" {
		t.Fatalf("strategy=%#v", req.Parts[0])
	}
}
