package parity

import (
	"time"

	"codea/tui/internal/runtime"
)

var (
	approvalOnce   = runtime.ApprovalOnce
	approvalReject = runtime.ApprovalReject
)

// Assertion defines the semantic checks for a parity scenario.
// Each bool flag enables a specific check against collected events.
type Assertion struct {
	RequireReasoning bool
	RequireAnswer    bool
	RequireApproval  bool
	RequireTool      bool
	RequireRaw       bool
	RequireAgent     string // expected agent name, checked against Prompt request
}

// Scenario defines a single parity test scenario.
type Scenario struct {
	Name             string
	Required         bool
	Prompt           *runtime.PromptRequest
	RepeatCount      int
	Timeout          time.Duration          // collection timeout per execution; 0 = default 30s
	Assertions       Assertion
	ApprovalDecision *runtime.ApprovalDecision // if set, ReplyApproval is called when approval.requested is received
}

// V1RequiredScenarios returns the minimal set of required parity scenarios
// for V1. Each scenario is required; SilentLoss for any of them must result
// in FAIL.
func V1RequiredScenarios() []Scenario {
	return []Scenario{
		{Name: "Health", Required: true, RepeatCount: 2},
		{Name: "CreateSession", Required: true, RepeatCount: 2},
		{Name: "Prompt", Required: true, RepeatCount: 2, Prompt: &runtime.PromptRequest{
			Agent: "general",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "hello"}},
		}, Assertions: Assertion{RequireAnswer: true}},
		{Name: "Streaming", Required: true, RepeatCount: 2, Prompt: &runtime.PromptRequest{
			Agent: "general",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "stream test"}},
		}, Assertions: Assertion{RequireAnswer: true}},
		{Name: "Answer", Required: true, RepeatCount: 2, Prompt: &runtime.PromptRequest{
			Agent: "general",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "answer test"}},
		}, Assertions: Assertion{RequireAnswer: true}},
		{Name: "Reasoning", Required: true, RepeatCount: 2, Prompt: &runtime.PromptRequest{
			Agent: "general",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "reasoning test"}},
		}, Assertions: Assertion{RequireReasoning: true, RequireAnswer: true}},
		{Name: "ToolLifecycle", Required: true, RepeatCount: 2, Prompt: &runtime.PromptRequest{
			Agent: "general",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "tool test"}},
		}, Assertions: Assertion{RequireTool: true}},
		{Name: "Approval", Required: true, RepeatCount: 2, Prompt: &runtime.PromptRequest{
			Agent: "general",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "approval test"}},
		}, Assertions: Assertion{RequireApproval: true}, ApprovalDecision: &approvalOnce},
		{Name: "Reject", Required: true, RepeatCount: 2, Prompt: &runtime.PromptRequest{
			Agent: "general",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "reject test"}},
		}, Assertions: Assertion{RequireApproval: true}, ApprovalDecision: &approvalReject},
		{Name: "Cancel", Required: true, RepeatCount: 2},
		{Name: "AgentSelection", Required: true, RepeatCount: 2, Prompt: &runtime.PromptRequest{
			Agent: "reviewer",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "review"}},
		}, Assertions: Assertion{RequireAnswer: true, RequireAgent: "reviewer"}},
		{Name: "RawEventHandling", Required: true, RepeatCount: 2, Prompt: &runtime.PromptRequest{
			Agent: "general",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "raw event test"}},
		}, Assertions: Assertion{RequireRaw: true}},
	}
}
