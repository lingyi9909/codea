package parity_test

import (
	"context"
	"testing"

	"codea/tui/internal/runtime"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

// TestSubagentPassthrough proves that a General Agent can delegate to a
// subagent via a SubtaskPart and that the runtime contract carries the
// delegation through unmodified. Task 9 verifies passthrough — subagent
// scheduling remains the OpenCode Runtime's job, never reimplemented in the
// Go TUI/Application.
func TestSubagentPassthrough(t *testing.T) {
	rt := fakeruntime.New()
	rt.Agents = []runtime.Agent{
		{Name: "general", Mode: "primary"},
		{Name: "explore", Mode: "subagent"},
	}

	ctx := context.Background()

	agents, err := rt.ListAgents(ctx)
	if err != nil {
		t.Fatalf("list agents: %v", err)
	}
	if len(agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(agents))
	}
	var sawSubagent bool
	for _, a := range agents {
		if a.Name == "explore" && a.Mode == "subagent" {
			sawSubagent = true
		}
	}
	if !sawSubagent {
		t.Error("subagent not exposed via ListAgents")
	}

	sess, err := rt.CreateSession(ctx, runtime.CreateSessionRequest{})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}

	req := runtime.PromptRequest{
		Agent: "general",
		Parts: []runtime.PromptPart{
			runtime.SubtaskPart{
				Agent:       "explore",
				Description: "find Go files",
				Prompt:      "find all .go files under the project",
			},
		},
	}
	if err := rt.Prompt(ctx, runtime.SessionID(sess.ID), req); err != nil {
		t.Fatalf("prompt with subtask: %v", err)
	}

	prompts := rt.Prompts()
	if len(prompts) != 1 {
		t.Fatalf("expected 1 prompt, got %d", len(prompts))
	}
	got := prompts[0].Request
	if got.Agent != "general" {
		t.Errorf("expected Agent=general, got %q", got.Agent)
	}
	if len(got.Parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(got.Parts))
	}
	sub, ok := got.Parts[0].(runtime.SubtaskPart)
	if !ok {
		t.Fatalf("expected SubtaskPart, got %T", got.Parts[0])
	}
	if sub.Agent != "explore" {
		t.Errorf("expected subtask Agent=explore, got %q", sub.Agent)
	}
}
