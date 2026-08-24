package app

import (
	"context"
	"encoding/json"
	"testing"

	"codea/tui/internal/runtime"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

func TestHandoffToGeneralPreservesSessionAndStructuredContext(t *testing.T) {
	fake := fakeruntime.New()
	handoff := AgentHandoff{
		SourceAgent: "unit-test-generator",
		TargetAgent: "general",
		SessionID:   runtime.SessionID("session-123"),
		UserGoal:    "add unit tests for OrderService",
		TaskSummary: "project analyzed; dedicated test tool cannot express the required profile",
		CollectedFacts: []FactRef{
			{ID: "fact-1", Summary: "Maven + JUnit 5"},
		},
		GeneratedFiles: []string{"src/test/java/OrderServiceTest.java", "src/test/java/OrderServiceTest.java"},
		ToolResults: []ToolResultRef{
			{ID: "tool-1", Tool: "analyze_test_project", Summary: "test root resolved"},
		},
		FailureReason: "UNSUPPORTED_REQUEST",
	}

	if err := HandoffToGeneral(context.Background(), fake, handoff); err != nil {
		t.Fatalf("HandoffToGeneral: %v", err)
	}

	prompts := fake.Prompts()
	if len(prompts) != 1 {
		t.Fatalf("prompt count=%d want 1", len(prompts))
	}
	got := prompts[0]
	if got.SessionID != handoff.SessionID {
		t.Fatalf("session=%q want %q", got.SessionID, handoff.SessionID)
	}
	if got.Request.Agent != "general" {
		t.Fatalf("agent=%q want general", got.Request.Agent)
	}
	if len(got.Request.Parts) != 1 {
		t.Fatalf("parts=%d want 1", len(got.Request.Parts))
	}
	text, ok := got.Request.Parts[0].(runtime.TextPart)
	if !ok {
		t.Fatalf("part type=%T want runtime.TextPart", got.Request.Parts[0])
	}
	if !text.Synthetic {
		t.Fatal("handoff control packet must be marked synthetic")
	}

	var envelope handoffEnvelope
	if err := json.Unmarshal([]byte(text.Text), &envelope); err != nil {
		t.Fatalf("handoff payload is not JSON: %v", err)
	}
	if envelope.SchemaVersion != 1 || envelope.Kind != "agent_handoff" {
		t.Fatalf("unexpected envelope: %+v", envelope)
	}
	if envelope.Handoff.UserGoal != handoff.UserGoal || envelope.Handoff.FailureReason != handoff.FailureReason {
		t.Fatalf("goal/failure not preserved: %+v", envelope.Handoff)
	}
	if len(envelope.Handoff.CollectedFacts) != 1 || len(envelope.Handoff.ToolResults) != 1 {
		t.Fatalf("facts/tool results not preserved: %+v", envelope.Handoff)
	}
	if len(envelope.Handoff.GeneratedFiles) != 1 {
		t.Fatalf("generated files must be de-duplicated, got %v", envelope.Handoff.GeneratedFiles)
	}
	if !envelope.Policy.ReuseSession || envelope.Policy.RecollectFacts || envelope.Policy.OverwriteGeneratedFiles || !envelope.Policy.ContinueOrExplain {
		t.Fatalf("unsafe continuation policy: %+v", envelope.Policy)
	}
}

func TestHandoffToGeneralRejectsIncompleteOrWrongTarget(t *testing.T) {
	fake := fakeruntime.New()
	cases := []AgentHandoff{
		{TargetAgent: "general", SessionID: "s", UserGoal: "goal", FailureReason: "reason"},
		{SourceAgent: "unit-test-generator", TargetAgent: "reviewer", SessionID: "s", UserGoal: "goal", FailureReason: "reason"},
		{SourceAgent: "unit-test-generator", TargetAgent: "general", UserGoal: "goal", FailureReason: "reason"},
		{SourceAgent: "unit-test-generator", TargetAgent: "general", SessionID: "s", FailureReason: "reason"},
		{SourceAgent: "unit-test-generator", TargetAgent: "general", SessionID: "s", UserGoal: "goal"},
	}
	for i, tc := range cases {
		if err := HandoffToGeneral(context.Background(), fake, tc); err == nil {
			t.Fatalf("case %d: expected validation error", i)
		}
	}
	if len(fake.Prompts()) != 0 {
		t.Fatal("invalid handoff must not call runtime Prompt")
	}
}
