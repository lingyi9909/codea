package parity

import "codea/tui/internal/runtime"

// Scenario defines a single parity test scenario.
type Scenario struct {
	Name      string
	Required  bool
	Prompt    *runtime.PromptRequest
	RepeatCount int
}

// V1RequiredScenarios returns the minimal set of required parity scenarios
// for V1. Each scenario is required; SilentLoss for any of them must result
// in FAIL.
func V1RequiredScenarios() []Scenario {
	return []Scenario{
		{Name: "Health", Required: true, RepeatCount: 1},
		{Name: "CreateSession", Required: true, RepeatCount: 1},
		{Name: "Prompt", Required: true, RepeatCount: 1, Prompt: &runtime.PromptRequest{
			Agent: "general",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "hello"}},
		}},
		{Name: "Streaming", Required: true, RepeatCount: 1, Prompt: &runtime.PromptRequest{
			Agent: "general",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "stream test"}},
		}},
		{Name: "Answer", Required: true, RepeatCount: 1, Prompt: &runtime.PromptRequest{
			Agent: "general",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "answer test"}},
		}},
		{Name: "Reasoning", Required: true, RepeatCount: 1, Prompt: &runtime.PromptRequest{
			Agent: "general",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "reasoning test"}},
		}},
		{Name: "ToolLifecycle", Required: true, RepeatCount: 1, Prompt: &runtime.PromptRequest{
			Agent: "general",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "tool test"}},
		}},
		{Name: "Approval", Required: true, RepeatCount: 1, Prompt: &runtime.PromptRequest{
			Agent: "general",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "approval test"}},
		}},
		{Name: "Reject", Required: true, RepeatCount: 1, Prompt: &runtime.PromptRequest{
			Agent: "general",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "reject test"}},
		}},
		{Name: "Cancel", Required: true, RepeatCount: 1},
		{Name: "AgentSelection", Required: true, RepeatCount: 1, Prompt: &runtime.PromptRequest{
			Agent: "reviewer",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "review"}},
		}},
		{Name: "RawEventHandling", Required: true, RepeatCount: 1, Prompt: &runtime.PromptRequest{
			Agent: "general",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "raw event test"}},
		}},
	}
}
