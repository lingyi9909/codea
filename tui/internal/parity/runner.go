package parity

import (
	"context"
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

	switch {
	case s.Name == "Health":
		r.runHealth(ctx, s, &sr)
	case s.Name == "CreateSession":
		r.runCreateSession(ctx, s, &sr)
	case s.Name == "Cancel":
		r.runCancel(ctx, s, &sr)
	case s.Prompt != nil:
		r.runPrompt(ctx, s, &sr)
	default:
		sr.Passed = true
	}

	return sr
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
	sr.Passed = true
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
	sr.Passed = true
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
	sr.Passed = true
}

func (r *Runner) runPrompt(ctx context.Context, s Scenario, sr *ScenarioResult) {
	// Baseline
	bEvents, bErr := r.collectEvents(ctx, r.Baseline, s.Prompt)
	if bErr != nil {
		sr.Failures = append(sr.Failures, Failure{Reason: "baseline prompt failed: " + bErr.Error()})
	}

	// Candidate
	cEvents, cErr := r.collectEvents(ctx, r.Candidate, s.Prompt)
	if cErr != nil {
		sr.Failures = append(sr.Failures, Failure{Reason: "candidate prompt failed: " + cErr.Error()})
	}

	// If either had a transport error, that's a hard failure.
	if len(sr.Failures) > 0 {
		return
	}

	// Compare event counts — fewer candidate events = silent loss.
	if len(cEvents) < len(bEvents) {
		sr.SilentLoss = true
		sr.Failures = append(sr.Failures, Failure{
			Reason: "silent loss: baseline produced events but candidate did not",
		})
		return
	}

	sr.Passed = true
}

func (r *Runner) collectEvents(ctx context.Context, rt runtime.AgentRuntime, req *runtime.PromptRequest) ([]runtime.Event, error) {
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
	// Collect buffered events. Fake runtimes send synchronously during
	// Prompt(), so all events are already in the channel buffer.
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				return events, nil
			}
			events = append(events, ev)
		case <-timeout:
			return events, nil
		case <-ctx.Done():
			return events, ctx.Err()
		}
	}
}
