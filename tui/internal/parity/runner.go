package parity

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"codea/tui/internal/runtime"
)

// PromptRecorder is an optional interface that AgentRuntime implementations
// may satisfy to expose the actual prompt request received. Parity scenarios
// use this to verify agent selection through observable evidence rather than
// self-referential scenario assertion.
type PromptRecorder interface {
	LastPrompt() (agent string, ok bool)
}

// Runner executes parity scenarios against Baseline and Candidate
// AgentRuntime implementations and compares the results.
type Runner struct {
	Baseline  runtime.AgentRuntime
	Candidate runtime.AgentRuntime
}

const (
	defaultTimeout          = 30 * time.Second
	inactivityFallback      = 500 * time.Millisecond
)

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

	if bErr != nil && cErr != nil {
		sr.Failures = append(sr.Failures, Failure{
			Reason: "both baseline and candidate Health failed: " + bErr.Error() + " / " + cErr.Error(),
		})
		return
	}
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

	if bErr != nil && cErr != nil {
		sr.Failures = append(sr.Failures, Failure{
			Reason: "both baseline and candidate CreateSession failed: " + bErr.Error() + " / " + cErr.Error(),
		})
		return
	}
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

	if bCancelErr != nil && cCancelErr != nil {
		sr.Failures = append(sr.Failures, Failure{
			Reason: "both baseline and candidate Cancel failed: " + bCancelErr.Error() + " / " + cCancelErr.Error(),
		})
		return
	}
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
	timeout := s.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	// Baseline
	bEvents, bErr := r.collectEvents(ctx, r.Baseline, s.Prompt, s.ApprovalDecision, timeout)
	if bErr != nil {
		sr.Failures = append(sr.Failures, Failure{Reason: "baseline prompt failed: " + bErr.Error()})
	}

	// Candidate
	cEvents, cErr := r.collectEvents(ctx, r.Candidate, s.Prompt, s.ApprovalDecision, timeout)
	if cErr != nil {
		sr.Failures = append(sr.Failures, Failure{Reason: "candidate prompt failed: " + cErr.Error()})
	}

	if len(sr.Failures) > 0 {
		return
	}

	// Verify agent through the actual request received by each runtime.
	// PromptRecorder lets FakeRuntime expose what was actually sent, turning
	// the assertion from self-referential into observable evidence.
	// When RequireAgent is set but a runtime doesn't implement PromptRecorder,
	// the assertion cannot be verified and the scenario must FAIL — a required
	// assertion must not silently pass due to insufficient observability.
	if s.Assertions.RequireAgent != "" {
		bRec, bOK := r.Baseline.(PromptRecorder)
		cRec, cOK := r.Candidate.(PromptRecorder)
		if !bOK || !cOK {
			missing := []string{}
			if !bOK {
				missing = append(missing, "baseline")
			}
			if !cOK {
				missing = append(missing, "candidate")
			}
			sr.Failures = append(sr.Failures, Failure{
				Reason: fmt.Sprintf("agent assertion requires PromptRecorder but %s does not implement it", strings.Join(missing, " and ")),
			})
			return
		}

		bAgent, bHas := bRec.LastPrompt()
		if !bHas {
			sr.Failures = append(sr.Failures, Failure{
				Reason: "baseline PromptRecorder has no recorded prompt — cannot verify agent",
			})
			return
		}
		if bAgent != s.Assertions.RequireAgent {
			sr.Failures = append(sr.Failures, Failure{
				Reason: fmt.Sprintf("baseline agent mismatch: expected %q, runtime received %q",
					s.Assertions.RequireAgent, bAgent),
			})
			return
		}
		cAgent, cHas := cRec.LastPrompt()
		if !cHas {
			sr.SilentLoss = true
			sr.Failures = append(sr.Failures, Failure{
				Reason: "candidate PromptRecorder has no recorded prompt — cannot verify agent",
			})
			return
		}
		if cAgent != s.Assertions.RequireAgent {
			sr.SilentLoss = true
			sr.Failures = append(sr.Failures, Failure{
				Reason: fmt.Sprintf("silent loss — candidate agent mismatch: expected %q, runtime received %q",
					s.Assertions.RequireAgent, cAgent),
			})
			return
		}
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

	// Semantic fingerprint comparison: event counts, streaming content, domain payloads.
	issues := compareFingerprints(bEvents, cEvents, s.Assertions)
	if len(issues) > 0 {
		sr.SilentLoss = true
		for _, issue := range issues {
			sr.Failures = append(sr.Failures, Failure{
				Reason: "silent loss — " + issue,
			})
		}
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

// eventFingerprint captures the semantic shape of a collected event stream.
type eventFingerprint struct {
	counts         map[runtime.EventType]int
	answerChars    int
	reasoningChars int
	toolCalls      int
	approvals      int
	rawEvents      int
	hasStepFinish  bool
}

func computeFingerprint(events []runtime.Event) eventFingerprint {
	fp := eventFingerprint{counts: make(map[runtime.EventType]int)}
	for _, e := range events {
		fp.counts[e.Type]++
		switch e.Type {
		case "answer.delta":
			fp.answerChars += len(e.Content)
		case "reasoning.delta":
			fp.reasoningChars += len(e.Content)
		case "tool.called":
			fp.toolCalls++
		case "approval.requested":
			fp.approvals++
		case "raw":
			fp.rawEvents++
		case "step.finished":
			fp.hasStepFinish = true
		}
	}
	return fp
}

// compareFingerprints checks that candidate preserves the semantic shape of baseline.
// Returns a list of specific issues found, empty if no silent loss.
func compareFingerprints(bEvents, cEvents []runtime.Event, a Assertion) []string {
	bFP := computeFingerprint(bEvents)
	cFP := computeFingerprint(cEvents)
	var issues []string

	// Streaming events: if baseline has content, candidate must have non-empty content.
	if a.RequireAnswer && bFP.answerChars > 0 && cFP.answerChars == 0 {
		issues = append(issues, fmt.Sprintf(
			"answer.delta content: baseline has %d chars, candidate has 0", bFP.answerChars))
	} else if a.RequireAnswer && cFP.counts["answer.delta"] < bFP.counts["answer.delta"] {
		issues = append(issues, fmt.Sprintf(
			"answer.delta count: baseline %d, candidate %d", bFP.counts["answer.delta"], cFP.counts["answer.delta"]))
	}

	if a.RequireReasoning && bFP.reasoningChars > 0 && cFP.reasoningChars == 0 {
		issues = append(issues, fmt.Sprintf(
			"reasoning.delta content: baseline has %d chars, candidate has 0", bFP.reasoningChars))
	} else if a.RequireReasoning && cFP.counts["reasoning.delta"] < bFP.counts["reasoning.delta"] {
		issues = append(issues, fmt.Sprintf(
			"reasoning.delta count: baseline %d, candidate %d", bFP.counts["reasoning.delta"], cFP.counts["reasoning.delta"]))
	}

	// Domain events: candidate must have at least as many as baseline.
	if a.RequireTool && cFP.toolCalls < bFP.toolCalls {
		issues = append(issues, fmt.Sprintf(
			"tool.called count: baseline %d, candidate %d", bFP.toolCalls, cFP.toolCalls))
	}
	if a.RequireApproval && cFP.approvals < bFP.approvals {
		issues = append(issues, fmt.Sprintf(
			"approval.requested count: baseline %d, candidate %d", bFP.approvals, cFP.approvals))
	}
	if a.RequireRaw && cFP.rawEvents < bFP.rawEvents {
		issues = append(issues, fmt.Sprintf(
			"raw event count: baseline %d, candidate %d", bFP.rawEvents, cFP.rawEvents))
	}

	// Step completion: candidate must complete the step.
	if bFP.hasStepFinish && !cFP.hasStepFinish {
		issues = append(issues, "candidate missing step.finished completion event")
	}

	// Type set coverage: any event type in baseline must also appear in candidate.
	// This catches losses beyond the assertion-keyed checks above.
	for t := range bFP.counts {
		if cFP.counts[t] == 0 {
			issues = append(issues, fmt.Sprintf(
				"candidate missing event type %q present in baseline (%d events)", t, bFP.counts[t]))
		}
	}

	return issues
}

func hasEventType(events []runtime.Event, t runtime.EventType) bool {
	for _, e := range events {
		if e.Type == t {
			return true
		}
	}
	return false
}

func (r *Runner) collectEvents(ctx context.Context, rt runtime.AgentRuntime, req *runtime.PromptRequest, approvalDecision *runtime.ApprovalDecision, timeout time.Duration) ([]runtime.Event, error) {
	session, err := rt.CreateSession(ctx, runtime.CreateSessionRequest{Title: "parity-prompt"})
	if err != nil {
		return nil, err
	}

	subCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	ch, err := rt.Subscribe(subCtx)
	if err != nil {
		return nil, err
	}

	if err := rt.Prompt(ctx, runtime.SessionID(session.ID), *req); err != nil {
		return nil, err
	}

	var events []runtime.Event
	deadline := time.After(timeout)

	// Inactivity starts as nil — don't start the inactivity clock until the
	// first event arrives. Otherwise a real runtime whose first event takes
	// >500ms gets truncated before it ever produces output.
	var inactivity <-chan time.Time

	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				return events, nil
			}
			events = append(events, ev)

			// Reset inactivity timer on each event after the first.
			inactivity = time.After(inactivityFallback)

			// Natural completion: step finished or failed.
			if ev.Type == "step.finished" || ev.Type == "step.failed" {
				return events, nil
			}

			if approvalDecision != nil && ev.Type == "approval.requested" && ev.Approval != nil && ev.Approval.ID != "" {
				if err := rt.ReplyApproval(ctx, runtime.ApprovalID(ev.Approval.ID), runtime.ApprovalReply{Decision: *approvalDecision}); err != nil {
					return events, fmt.Errorf("ReplyApproval(%s): %w", ev.Approval.ID, err)
				}
			}
		case <-inactivity:
			return events, nil
		case <-deadline:
			return events, nil
		case <-ctx.Done():
			return events, ctx.Err()
		}
	}
}

// ConcatContent concatenates content from events of a given type for test assertions.
func ConcatContent(events []runtime.Event, t runtime.EventType) string {
	var b strings.Builder
	for _, e := range events {
		if e.Type == t {
			b.WriteString(e.Content)
		}
	}
	return b.String()
}
