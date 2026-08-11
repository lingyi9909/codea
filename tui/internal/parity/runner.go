package parity

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"codea/tui/internal/runtime"
)

// Runner executes parity scenarios against Baseline and Candidate
// AgentRuntime implementations and compares the results.
type Runner struct {
	Baseline  runtime.AgentRuntime
	Candidate runtime.AgentRuntime
}

// Run executes a single scenario against both runtimes and returns the result.
func (r *Runner) Run(ctx context.Context, s Scenario) ScenarioResult {
	sr := ScenarioResult{
		Name:     s.Name,
		Required: s.Required,
		Runs:     1,
	}

	repeats := s.RepeatCount
	if repeats < 1 {
		repeats = 1
	}

	var allPassed []bool
	for i := 0; i < repeats; i++ {
		passed := r.executeOnce(ctx, s, &sr)
		allPassed = append(allPassed, passed)
	}
	sr.Runs = repeats

	// If any repeat failed, the scenario fails.
	for _, p := range allPassed {
		if !p {
			return sr
		}
	}
	sr.Passed = true
	return sr
}

func (r *Runner) executeOnce(ctx context.Context, s Scenario, sr *ScenarioResult) bool {
	prev := len(sr.Failures)
	switch {
	case s.Name == "Health":
		r.runHealth(ctx, s, sr)
	case s.Name == "CreateSession":
		r.runCreateSession(ctx, s, sr)
	case s.Name == "Cancel":
		r.runCancel(ctx, s, sr)
	case s.Prompt != nil:
		r.runPrompt(ctx, s, sr)
	default:
		return true
	}
	return len(sr.Failures) == prev
}

// RunAll executes multiple scenarios and returns an aggregated Result.
func (r *Runner) RunAll(ctx context.Context, scenarios []Scenario) *Result {
	result := &Result{}
	for _, s := range scenarios {
		sr := r.Run(ctx, s)
		result.Scenarios = append(result.Scenarios, sr)
	}
	result.compute()
	return result
}

func (r *Runner) runHealth(ctx context.Context, s Scenario, sr *ScenarioResult) {
	bInfo, bErr := r.Baseline.Health(ctx)
	cInfo, cErr := r.Candidate.Health(ctx)

	if bErr != nil && cErr == nil {
		sr.Failures = append(sr.Failures, Failure{
			Reason: "baseline Health failed but candidate succeeded",
		})
		return
	}
	if cErr != nil && bErr == nil {
		sr.Failures = append(sr.Failures, Failure{
			Reason: "candidate Health failed but baseline succeeded: " + cErr.Error(),
		})
		return
	}
	if bInfo.Healthy != cInfo.Healthy {
		sr.Failures = append(sr.Failures, Failure{
			Reason: "health mismatch",
		})
		return
	}
}

func (r *Runner) runCreateSession(ctx context.Context, s Scenario, sr *ScenarioResult) {
	bSess, bErr := r.Baseline.CreateSession(ctx, runtime.CreateSessionRequest{Title: "parity-test"})
	cSess, cErr := r.Candidate.CreateSession(ctx, runtime.CreateSessionRequest{Title: "parity-test"})

	if bErr != nil && cErr == nil {
		sr.Failures = append(sr.Failures, Failure{
			Reason: "baseline CreateSession failed but candidate succeeded",
		})
		return
	}
	if cErr != nil && bErr == nil {
		sr.Failures = append(sr.Failures, Failure{
			Reason: "candidate CreateSession failed but baseline succeeded: " + cErr.Error(),
		})
		return
	}
	if bErr == nil && bSess.ID == "" {
		sr.Failures = append(sr.Failures, Failure{Reason: "baseline returned empty session ID"})
		return
	}
	if cErr == nil && cSess.ID == "" {
		sr.Failures = append(sr.Failures, Failure{Reason: "candidate returned empty session ID"})
		return
	}
}

func (r *Runner) runCancel(ctx context.Context, s Scenario, sr *ScenarioResult) {
	bSess, bErr := r.Baseline.CreateSession(ctx, runtime.CreateSessionRequest{Title: "parity-cancel"})
	if bErr != nil {
		sr.Failures = append(sr.Failures, Failure{Reason: "baseline CreateSession failed: " + bErr.Error()})
		return
	}
	cSess, cErr := r.Candidate.CreateSession(ctx, runtime.CreateSessionRequest{Title: "parity-cancel"})
	if cErr != nil {
		sr.Failures = append(sr.Failures, Failure{Reason: "candidate CreateSession failed: " + cErr.Error()})
		return
	}

	bCancelErr := r.Baseline.Cancel(ctx, runtime.SessionID(bSess.ID))
	cCancelErr := r.Candidate.Cancel(ctx, runtime.SessionID(cSess.ID))

	if bCancelErr != nil && cCancelErr == nil {
		sr.Failures = append(sr.Failures, Failure{Reason: "baseline Cancel failed but candidate succeeded"})
		return
	}
	if cCancelErr != nil && bCancelErr == nil {
		sr.Failures = append(sr.Failures, Failure{Reason: "candidate Cancel failed but baseline succeeded"})
		return
	}
}

func (r *Runner) runPrompt(ctx context.Context, s Scenario, sr *ScenarioResult) {
	// Check RequireAgent assertion against the scenario's Prompt request.
	if s.Assertions.RequireAgent != "" && s.Prompt != nil {
		if s.Prompt.Agent != s.Assertions.RequireAgent {
			sr.Failures = append(sr.Failures, Failure{
				Reason: fmt.Sprintf("agent mismatch: expected %q, got %q",
					s.Assertions.RequireAgent, s.Prompt.Agent),
			})
			return
		}
	}

	// Baseline
	bEvents, bErr := r.collectEvents(ctx, r.Baseline, s.Prompt, s.ApprovalDecision)
	if bErr != nil {
		sr.Failures = append(sr.Failures, Failure{Reason: "baseline prompt failed: " + bErr.Error()})
	}

	// Candidate
	cEvents, cErr := r.collectEvents(ctx, r.Candidate, s.Prompt, s.ApprovalDecision)
	if cErr != nil {
		sr.Failures = append(sr.Failures, Failure{Reason: "candidate prompt failed: " + cErr.Error()})
	}

	if len(sr.Failures) > 0 {
		return
	}

	// Check semantic assertions against Baseline and Candidate.
	bSat := checkAssertions(bEvents, s.Assertions)
	cSat := checkAssertions(cEvents, s.Assertions)

	if !bSat.ok {
		sr.Failures = append(sr.Failures, Failure{
			Reason: "baseline failed assertion: " + bSat.reason,
		})
		return
	}

	if !cSat.ok {
		sr.SilentLoss = true
		sr.Failures = append(sr.Failures, Failure{
			Reason: "silent loss — candidate failed assertion: " + cSat.reason,
		})
		return
	}
}

type assertResult struct {
	ok     bool
	reason string
}

func checkAssertions(events []runtime.Event, a Assertion) assertResult {
	if a.RequireReasoning {
		if !hasEventType(events, "reasoning.delta") {
			return assertResult{false, "missing reasoning.delta event"}
		}
	}
	if a.RequireAnswer {
		if !hasEventType(events, "answer.delta") {
			return assertResult{false, "missing answer.delta event"}
		}
	}
	if a.RequireApproval {
		found := false
		for _, e := range events {
			if e.Type == "approval.requested" && e.Approval != nil &&
				e.Approval.ID != "" && e.Approval.Permission != "" {
				found = true
				break
			}
		}
		if !found {
			return assertResult{false, "missing approval.requested event with non-empty ID and Permission"}
		}
	}
	if a.RequireTool {
		found := false
		for _, e := range events {
			if e.Type == "tool.called" && e.Tool != nil &&
				e.Tool.Name != "" && e.Tool.CallID != "" {
				found = true
				break
			}
		}
		if !found {
			return assertResult{false, "missing tool.called event with non-empty Name and CallID"}
		}
	}
	if a.RequireRaw {
		found := false
		for _, e := range events {
			if e.Type == "raw" && len(e.Raw) > 0 {
				// Verify it's valid JSON.
				var v any
				if json.Unmarshal(e.Raw, &v) == nil {
					found = true
					break
				}
			}
		}
		if !found {
			return assertResult{false, "missing raw event with non-empty valid JSON payload"}
		}
	}
	return assertResult{ok: true}
}

func hasEventType(events []runtime.Event, t runtime.EventType) bool {
	for _, e := range events {
		if e.Type == t {
			return true
		}
	}
	return false
}

func (r *Runner) collectEvents(ctx context.Context, rt runtime.AgentRuntime, req *runtime.PromptRequest, approvalDecision *runtime.ApprovalDecision) ([]runtime.Event, error) {
	session, err := rt.CreateSession(ctx, runtime.CreateSessionRequest{Title: "parity-prompt"})
	if err != nil {
		return nil, err
	}

	ch, err := rt.Subscribe(ctx)
	if err != nil {
		return nil, err
	}

	if err := rt.Prompt(ctx, runtime.SessionID(session.ID), *req); err != nil {
		return nil, err
	}

	var events []runtime.Event
	timeout := time.After(100 * time.Millisecond)
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				return events, nil
			}
			events = append(events, ev)

			// If an approval is requested and the scenario specifies a decision,
			// reply immediately so the runtime can continue.
			if approvalDecision != nil && ev.Type == "approval.requested" && ev.Approval != nil && ev.Approval.ID != "" {
				if err := rt.ReplyApproval(ctx, runtime.ApprovalID(ev.Approval.ID), runtime.ApprovalReply{Decision: *approvalDecision}); err != nil {
					return events, fmt.Errorf("ReplyApproval(%s): %w", ev.Approval.ID, err)
				}
			}
		case <-timeout:
			return events, nil
		case <-ctx.Done():
			return events, ctx.Err()
		}
	}
}
