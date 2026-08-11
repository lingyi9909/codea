package parity

import (
	"context"
	"testing"
	"time"

	"codea/tui/internal/runtime"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

// TestSemanticAssertionReasoning verifies that a scenario with
// RequireReasoning=true detects when reasoning is missing.
func TestSemanticAssertionReasoningRequired(t *testing.T) {
	baseline := fakeruntime.New()
	baseline.Events = []runtime.Event{
		{Type: runtime.EventType("reasoning.delta"), Content: "let me think"},
		{Type: runtime.EventType("answer.delta"), Content: "ok"},
	}

	candidate := fakeruntime.New()
	// Candidate has same event count but missing reasoning.delta.
	candidate.Events = []runtime.Event{
		{Type: runtime.EventType("answer.delta"), Content: "ok"},
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
		t.Error("should fail: candidate missing reasoning.delta even though same event count")
	}
	if !result.SilentLoss {
		t.Error("should detect silent loss: reasoning.delta present in baseline, missing in candidate")
	}
}

// TestSemanticAssertionApproval verifies that RequireApproval detects missing
// approval.requested events.
func TestSemanticAssertionApprovalRequired(t *testing.T) {
	baseline := fakeruntime.New()
	baseline.Events = []runtime.Event{
		{Type: runtime.EventType("approval.requested"), Approval: &runtime.ApprovalRequest{
			ID: "approval-1", Permission: "bash",
		}},
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

	result := runner.Run(ctx, Scenario{
		Name:     "Approval",
		Required: true,
		Prompt: &runtime.PromptRequest{
			Agent: "general",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "test"}},
		},
		Assertions: Assertion{
			RequireApproval: true,
		},
	})

	if result.Passed {
		t.Error("should fail: candidate missing approval.requested")
	}
	if !result.SilentLoss {
		t.Error("should detect silent loss: approval.requested present in baseline, missing in candidate")
	}
}

// TestSemanticAssertionTool verifies RequireTool detects missing tool.called.
func TestSemanticAssertionToolRequired(t *testing.T) {
	baseline := fakeruntime.New()
	baseline.Events = []runtime.Event{
		{Type: runtime.EventType("tool.called"), Tool: &runtime.ToolEvent{
			Name: "read", CallID: "call-1",
		}},
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

	result := runner.Run(ctx, Scenario{
		Name:     "ToolLifecycle",
		Required: true,
		Prompt: &runtime.PromptRequest{
			Agent: "general",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "test"}},
		},
		Assertions: Assertion{
			RequireTool: true,
		},
	})

	if result.Passed {
		t.Error("should fail: candidate missing tool.called")
	}
	if !result.SilentLoss {
		t.Error("should detect silent loss: tool.called present in baseline, missing in candidate")
	}
}

// TestSemanticAssertionRaw verifies RequireRaw detects missing raw events.
func TestSemanticAssertionRawRequired(t *testing.T) {
	baseline := fakeruntime.New()
	baseline.Events = []runtime.Event{
		{Type: runtime.EventType("raw"), Raw: []byte(`{"foo":"bar"}`)},
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

	result := runner.Run(ctx, Scenario{
		Name:     "RawEventHandling",
		Required: true,
		Prompt: &runtime.PromptRequest{
			Agent: "general",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "test"}},
		},
		Assertions: Assertion{
			RequireRaw: true,
		},
	})

	if result.Passed {
		t.Error("should fail: candidate missing raw event")
	}
	if !result.SilentLoss {
		t.Error("should detect silent loss: raw event present in baseline, missing in candidate")
	}
}

// TestSemanticAssertionSameCountDifferentSemantics is the critical Fix 2 test:
// Baseline and Candidate have same event count but different event types.
// This must FAIL with SilentLoss.
func TestSemanticAssertionSameCountDifferentSemantics(t *testing.T) {
	baseline := fakeruntime.New()
	baseline.Events = []runtime.Event{
		{Type: runtime.EventType("reasoning.delta"), Content: "thinking..."},
		{Type: runtime.EventType("answer.delta"), Content: "ok"},
	}

	candidate := fakeruntime.New()
	// Same count (2) but completely different semantics — no reasoning.
	candidate.Events = []runtime.Event{
		{Type: runtime.EventType("tool.called"), Tool: &runtime.ToolEvent{Name: "x", CallID: "1"}},
		{Type: runtime.EventType("answer.delta"), Content: "ok"},
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
		},
	})

	if result.Passed {
		t.Error("Fix 2 FAIL: same event count but different semantics must FAIL")
	}
	if !result.SilentLoss {
		t.Error("Fix 2 FAIL: must detect silent loss even when len(bEvents)==len(cEvents)")
	}
}

// TestSemanticAssertionAllSatisfied verifies that when both runtimes satisfy
// all assertions, the scenario passes.
func TestSemanticAssertionAllSatisfied(t *testing.T) {
	baseline := fakeruntime.New()
	baseline.Events = []runtime.Event{
		{Type: runtime.EventType("reasoning.delta"), Content: "think"},
		{Type: runtime.EventType("answer.delta"), Content: "ok"},
		{Type: runtime.EventType("tool.called"), Tool: &runtime.ToolEvent{Name: "read", CallID: "1"}},
		{Type: runtime.EventType("step.finished")},
	}

	candidate := fakeruntime.New()
	candidate.Events = []runtime.Event{
		{Type: runtime.EventType("reasoning.delta"), Content: "think"},
		{Type: runtime.EventType("answer.delta"), Content: "ok"},
		{Type: runtime.EventType("tool.called"), Tool: &runtime.ToolEvent{Name: "read", CallID: "2"}},
		{Type: runtime.EventType("step.finished")},
	}

	runner := Runner{Baseline: baseline, Candidate: candidate}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := runner.Run(ctx, Scenario{
		Name:     "FullParity",
		Required: true,
		Prompt: &runtime.PromptRequest{
			Agent: "general",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "test"}},
		},
		Assertions: Assertion{
			RequireReasoning: true,
			RequireAnswer:    true,
			RequireTool:      true,
		},
	})

	if !result.Passed {
		t.Errorf("should pass: both satisfy all assertions, failures: %v", result.Failures)
	}
	if result.SilentLoss {
		t.Error("should not report silent loss when both satisfy assertions")
	}
}

// TestAssertionDomainPayload verifies that when RequireApproval is set,
// the assertion also checks that Approval.ID and Approval.Permission are
// non-empty.
func TestAssertionDomainPayloadApproval(t *testing.T) {
	baseline := fakeruntime.New()
	baseline.Events = []runtime.Event{
		{Type: runtime.EventType("approval.requested"), Approval: &runtime.ApprovalRequest{
			ID: "approval-1", Permission: "bash",
		}},
	}

	candidate := fakeruntime.New()
	// Has approval event but with empty ID — domain payload incomplete.
	candidate.Events = []runtime.Event{
		{Type: runtime.EventType("approval.requested"), Approval: &runtime.ApprovalRequest{
			ID: "", Permission: "",
		}},
	}

	runner := Runner{Baseline: baseline, Candidate: candidate}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := runner.Run(ctx, Scenario{
		Name:     "Approval",
		Required: true,
		Prompt: &runtime.PromptRequest{
			Agent: "general",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "test"}},
		},
		Assertions: Assertion{
			RequireApproval: true,
		},
	})

	if result.Passed {
		t.Error("should fail: candidate has approval event but empty domain payload")
	}
}

// TestAssertionDomainPayloadTool verifies RequireTool checks Tool.Name and Tool.CallID.
func TestAssertionDomainPayloadTool(t *testing.T) {
	baseline := fakeruntime.New()
	baseline.Events = []runtime.Event{
		{Type: runtime.EventType("tool.called"), Tool: &runtime.ToolEvent{
			Name: "read", CallID: "call-1",
		}},
	}

	candidate := fakeruntime.New()
	candidate.Events = []runtime.Event{
		{Type: runtime.EventType("tool.called"), Tool: &runtime.ToolEvent{
			Name: "", CallID: "",
		}},
	}

	runner := Runner{Baseline: baseline, Candidate: candidate}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := runner.Run(ctx, Scenario{
		Name:     "ToolLifecycle",
		Required: true,
		Prompt: &runtime.PromptRequest{
			Agent: "general",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "test"}},
		},
		Assertions: Assertion{
			RequireTool: true,
		},
	})

	if result.Passed {
		t.Error("should fail: candidate has tool event but empty domain payload")
	}
}

// TestAgentSelectionAssertion verifies agent selection is checked.
func TestAgentSelectionAssertion(t *testing.T) {
	baseline := fakeruntime.New()
	baseline.Events = []runtime.Event{
		{Type: runtime.EventType("answer.delta"), Content: "ok"},
	}

	candidate := fakeruntime.New()
	candidate.Events = []runtime.Event{
		{Type: runtime.EventType("answer.delta"), Content: "ok"},
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
		Assertions: Assertion{
			RequireAgent: "reviewer",
		},
	})

	if !result.Passed {
		t.Errorf("should pass: both used reviewer agent, failures: %v", result.Failures)
	}

	// Now test with wrong agent.
	result2 := runner.Run(ctx, Scenario{
		Name:     "AgentSelection",
		Required: true,
		Prompt: &runtime.PromptRequest{
			Agent: "general",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "review"}},
		},
		Assertions: Assertion{
			RequireAgent: "reviewer",
		},
	})

	if result2.Passed {
		t.Error("should fail: prompt sent to general but requires reviewer agent")
	}
}

func TestRunnerApprovalOnce(t *testing.T) {
	once := runtime.ApprovalOnce

	baseline := fakeruntime.New()
	baseline.Events = []runtime.Event{
		{Type: runtime.EventType("approval.requested"), Approval: &runtime.ApprovalRequest{
			ID: "approval-1", Permission: "bash",
		}},
	}
	// After ReplyApproval(once), the runtime continues and emits tool + answer.
	baseline.ApprovalOnceEvents = []runtime.Event{
		{Type: runtime.EventType("tool.called"), Tool: &runtime.ToolEvent{
			Name: "read", CallID: "call-1",
		}},
		{Type: runtime.EventType("answer.delta"), Content: "done after approval"},
	}

	candidate := fakeruntime.New()
	candidate.Events = []runtime.Event{
		{Type: runtime.EventType("approval.requested"), Approval: &runtime.ApprovalRequest{
			ID: "approval-1", Permission: "bash",
		}},
	}
	candidate.ApprovalOnceEvents = []runtime.Event{
		{Type: runtime.EventType("tool.called"), Tool: &runtime.ToolEvent{
			Name: "read", CallID: "call-1",
		}},
		{Type: runtime.EventType("answer.delta"), Content: "done after approval"},
	}

	runner := Runner{Baseline: baseline, Candidate: candidate}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := runner.Run(ctx, Scenario{
		Name:     "Approval",
		Required: true,
		Prompt: &runtime.PromptRequest{
			Agent: "general",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "approve this"}},
		},
		Assertions:       Assertion{RequireApproval: true, RequireTool: true, RequireAnswer: true},
		ApprovalDecision: &once,
	})

	if !result.Passed {
		t.Errorf("approval once should pass, failures: %v", result.Failures)
	}

	// Verify ReplyApproval was called with ApprovalOnce.
	cRecords := candidate.Approvals()
	if len(cRecords) != 1 {
		t.Fatalf("expected 1 ReplyApproval call, got %d", len(cRecords))
	}
	if cRecords[0].Reply.Decision != runtime.ApprovalOnce {
		t.Errorf("expected decision ApprovalOnce, got %s", cRecords[0].Reply.Decision)
	}
	if cRecords[0].ID != runtime.ApprovalID("approval-1") {
		t.Errorf("expected approval ID approval-1, got %s", cRecords[0].ID)
	}
}

func TestRunnerApprovalReject(t *testing.T) {
	reject := runtime.ApprovalReject

	baseline := fakeruntime.New()
	baseline.Events = []runtime.Event{
		{Type: runtime.EventType("approval.requested"), Approval: &runtime.ApprovalRequest{
			ID: "approval-1", Permission: "bash",
		}},
		{Type: runtime.EventType("step.finished")},
	}

	candidate := fakeruntime.New()
	candidate.Events = []runtime.Event{
		{Type: runtime.EventType("approval.requested"), Approval: &runtime.ApprovalRequest{
			ID: "approval-1", Permission: "bash",
		}},
		{Type: runtime.EventType("step.finished")},
	}

	runner := Runner{Baseline: baseline, Candidate: candidate}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := runner.Run(ctx, Scenario{
		Name:     "Reject",
		Required: true,
		Prompt: &runtime.PromptRequest{
			Agent: "general",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "reject this"}},
		},
		Assertions:       Assertion{RequireApproval: true},
		ApprovalDecision: &reject,
	})

	if !result.Passed {
		t.Errorf("reject should pass, failures: %v", result.Failures)
	}

	// Verify ReplyApproval was called with ApprovalReject.
	cRecords := candidate.Approvals()
	if len(cRecords) != 1 {
		t.Fatalf("expected 1 ReplyApproval call, got %d", len(cRecords))
	}
	if cRecords[0].Reply.Decision != runtime.ApprovalReject {
		t.Errorf("expected decision ApprovalReject, got %s", cRecords[0].Reply.Decision)
	}
	if cRecords[0].ID != runtime.ApprovalID("approval-1") {
		t.Errorf("expected approval ID approval-1, got %s", cRecords[0].ID)
	}

	// Verify that after reject, no tool execution was performed.
	// ApprovalOnceEvents were not configured, and no tool.called should appear.
	for _, ev := range candidate.Events {
		if ev.Type == "tool.called" {
			t.Error("reject scenario must not produce tool.called events — rejected operation must not execute")
		}
	}
}

func TestRunnerApprovalReplyError(t *testing.T) {
	once := runtime.ApprovalOnce

	baseline := fakeruntime.New()
	baseline.Events = []runtime.Event{
		{Type: runtime.EventType("approval.requested"), Approval: &runtime.ApprovalRequest{
			ID: "approval-1", Permission: "bash",
		}},
	}

	candidate := fakeruntime.New()
	candidate.Events = []runtime.Event{
		{Type: runtime.EventType("approval.requested"), Approval: &runtime.ApprovalRequest{
			ID: "approval-1", Permission: "bash",
		}},
	}
	candidate.ReplyApprovalError = fakeruntime.ErrSimulated

	runner := Runner{Baseline: baseline, Candidate: candidate}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := runner.Run(ctx, Scenario{
		Name:     "Approval",
		Required: true,
		Prompt: &runtime.PromptRequest{
			Agent: "general",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "approve this"}},
		},
		Assertions:       Assertion{RequireApproval: true},
		ApprovalDecision: &once,
	})

	if result.Passed {
		t.Error("ReplyApproval error must cause scenario FAIL")
	}
}
