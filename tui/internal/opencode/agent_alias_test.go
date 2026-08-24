package opencode

import (
	"testing"

	"codea/tui/internal/runtime"
)

func TestMapPromptRequestMapsCodeaGeneralToOpenCodePrimaryBuild(t *testing.T) {
	_, req, err := MapPromptRequest("sess-1", runtime.PromptRequest{
		Agent: "general",
		Parts: []runtime.PromptPart{runtime.TextPart{Text: "hello"}},
	})
	if err != nil {
		t.Fatalf("MapPromptRequest: %v", err)
	}
	if req.Agent != "build" {
		t.Fatalf("vendor Agent = %q, want build for Codea semantic general agent", req.Agent)
	}
}

func TestMapPromptRequestPreservesExplicitVendorOrCustomAgent(t *testing.T) {
	for _, agent := range []string{"build", "code-reviewer", "unit-test-generator", "api-documentation"} {
		t.Run(agent, func(t *testing.T) {
			_, req, err := MapPromptRequest("sess-1", runtime.PromptRequest{
				Agent: agent,
				Parts: []runtime.PromptPart{runtime.TextPart{Text: "hello"}},
			})
			if err != nil {
				t.Fatalf("MapPromptRequest: %v", err)
			}
			if req.Agent != agent {
				t.Fatalf("vendor Agent = %q, want %q", req.Agent, agent)
			}
		})
	}
}
