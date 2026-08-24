package parity

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"codea/tui/internal/runtime"
)

type PromptRecorder interface {
	LastPrompt() (agent string, ok bool)
}

type Runner struct {
	Baseline  runtime.AgentRuntime
	Candidate runtime.AgentRuntime
}

const (
	defaultTimeout     = 30 * time.Second
	inactivityFallback = 500 * time.Millisecond
	terminalGrace      = 1 * time.Second
	semanticGrace      = 2 * time.Second
)

func (r *Runner) Run(ctx context.Context, s Scenario) ScenarioResult {
	sr := ScenarioResult{Name: s.Name, Required: s.Required, Runs: 1}
	repeats := s.RepeatCount
	if repeats < 1 {
		repeats = 1
	}
	passes := make([]bool, 0, repeats)
	for i := 0; i < repeats; i++ {
		passes = append(passes, r.executeOnce(ctx, s, &sr))
	}
	sr.Runs = repeats
	for _, pass := range passes {
		if !pass {
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
		r.runHealth(ctx, sr)
	case s.Name == "CreateSession":
		r.runCreateSession(ctx, sr)
	case s.Name == "Cancel":
		r.runCancel(ctx, sr)
	case s.Prompt != nil:
		r.runPrompt(ctx, s, sr)
	}
	return len(sr.Failures) == prev
}

func (r *Runner) RunAll(ctx context.Context, scenarios []Scenario) *Result {
	result := &Result{}
	for _, s := range scenarios {
		result.Scenarios = append(result.Scenarios, r.Run(ctx, s))
	}
	result.compute()
	return result
}

func (r *Runner) runHealth(ctx context.Context, sr *ScenarioResult) {
	bInfo, bErr := r.Baseline.Health(ctx)
	cInfo, cErr := r.Candidate.Health(ctx)
	switch {
	case bErr != nil && cErr != nil:
		sr.Failures = append(sr.Failures, Failure{Reason: "both baseline and candidate Health failed: " + bErr.Error() + " / " + cErr.Error()})
	case bErr != nil:
		sr.Failures = append(sr.Failures, Failure{Reason: "baseline Health failed but candidate succeeded"})
	case cErr != nil:
		sr.Failures = append(sr.Failures, Failure{Reason: "candidate Health failed but baseline succeeded: " + cErr.Error()})
	case bInfo.Healthy != cInfo.Healthy:
		sr.Failures = append(sr.Failures, Failure{Reason: "health mismatch"})
	}
}

func (r *Runner) runCreateSession(ctx context.Context, sr *ScenarioResult) {
	bSess, bErr := r.Baseline.CreateSession(ctx, runtime.CreateSessionRequest{Title: "parity-test"})
	cSess, cErr := r.Candidate.CreateSession(ctx, runtime.CreateSessionRequest{Title: "parity-test"})
	switch {
	case bErr != nil && cErr != nil:
		sr.Failures = append(sr.Failures, Failure{Reason: "both baseline and candidate CreateSession failed: " + bErr.Error() + " / " + cErr.Error()})
	case bErr != nil:
		sr.Failures = append(sr.Failures, Failure{Reason: "baseline CreateSession failed but candidate succeeded"})
	case cErr != nil:
		sr.Failures = append(sr.Failures, Failure{Reason: "candidate CreateSession failed but baseline succeeded: " + cErr.Error()})
	case bSess.ID == "":
		sr.Failures = append(sr.Failures, Failure{Reason: "baseline returned empty session ID"})
	case cSess.ID == "":
		sr.Failures = append(sr.Failures, Failure{Reason: "candidate returned empty session ID"})
	}
}

func (r *Runner) runCancel(ctx context.Context, sr *ScenarioResult) {
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
	bErr = r.Baseline.Cancel(ctx, runtime.SessionID(bSess.ID))
	cErr = r.Candidate.Cancel(ctx, runtime.SessionID(cSess.ID))
	switch {
	case bErr != nil && cErr != nil:
		sr.Failures = append(sr.Failures, Failure{Reason: "both baseline and candidate Cancel failed: " + bErr.Error() + " / " + cErr.Error()})
	case bErr != nil:
		sr.Failures = append(sr.Failures, Failure{Reason: "baseline Cancel failed but candidate succeeded"})
	case cErr != nil:
		sr.Failures = append(sr.Failures, Failure{Reason: "candidate Cancel failed but baseline succeeded: " + cErr.Error()})
	}
}

func (r *Runner) runPrompt(ctx context.Context, s Scenario, sr *ScenarioResult) {
	timeout := s.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	drainRawTail := s.Assertions.RequireRaw
	bEvents, bErr := r.collectEvents(ctx, r.Baseline, s.Prompt, s.ApprovalDecision, s.Assertions, timeout, drainRawTail)
	if bErr != nil {
		sr.Failures = append(sr.Failures, Failure{Reason: "baseline prompt failed: " + bErr.Error()})
	}
	cEvents, cErr := r.collectEvents(ctx, r.Candidate, s.Prompt, s.ApprovalDecision, s.Assertions, timeout, drainRawTail)
	if cErr != nil {
		sr.Failures = append(sr.Failures, Failure{Reason: "candidate prompt failed: " + cErr.Error()})
	}
	if len(sr.Failures) > 0 {
		return
	}

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
			sr.Failures = append(sr.Failures, Failure{Reason: fmt.Sprintf("agent assertion requires PromptRecorder but %s does not implement it", strings.Join(missing, " and "))})
			return
		}
		bAgent, bHas := bRec.LastPrompt()
		if !bHas {
			sr.Failures = append(sr.Failures, Failure{Reason: "baseline PromptRecorder has no recorded prompt — cannot verify agent"})
			return
		}
		if bAgent != s.Assertions.RequireAgent {
			sr.Failures = append(sr.Failures, Failure{Reason: fmt.Sprintf("baseline agent mismatch: expected %q, runtime received %q", s.Assertions.RequireAgent, bAgent)})
			return
		}
		cAgent, cHas := cRec.LastPrompt()
		if !cHas {
			sr.SilentLoss = true
			sr.Failures = append(sr.Failures, Failure{Reason: "candidate PromptRecorder has no recorded prompt — cannot verify agent"})
			return
		}
		if cAgent != s.Assertions.RequireAgent {
			sr.SilentLoss = true
			sr.Failures = append(sr.Failures, Failure{Reason: fmt.Sprintf("silent loss — candidate agent mismatch: expected %q, runtime received %q", s.Assertions.RequireAgent, cAgent)})
			return
		}
	}

	bSat := checkAssertions(bEvents, s.Assertions)
	cSat := checkAssertions(cEvents, s.Assertions)
	if !bSat.ok {
		sr.Failures = append(sr.Failures, Failure{Reason: "baseline failed assertion: " + bSat.reason})
		return
	}
	if !cSat.ok {
		sr.SilentLoss = true
		sr.Failures = append(sr.Failures, Failure{Reason: "silent loss — candidate failed assertion: " + cSat.reason})
		return
	}
	for _, issue := range compareFingerprints(bEvents, cEvents, s.Assertions) {
		sr.SilentLoss = true
		sr.Failures = append(sr.Failures, Failure{Reason: "silent loss — " + issue})
	}
}

type assertResult struct {
	ok     bool
	reason string
}

func checkAssertions(events []runtime.Event, a Assertion) assertResult {
	if a.RequireReasoning && !hasEventType(events, "reasoning.delta") {
		return assertResult{false, "missing reasoning.delta event"}
	}
	if a.RequireAnswer && !hasEventType(events, "answer.delta") {
		return assertResult{false, "missing answer.delta event"}
	}
	if a.RequireApproval {
		found := false
		for _, e := range events {
			if e.Type == "approval.requested" && e.Approval != nil && e.Approval.ID != "" && e.Approval.Permission != "" {
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
			if e.Type == "tool.called" && e.Tool != nil && e.Tool.Name != "" && e.Tool.CallID != "" {
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

type eventFingerprint struct {
	counts         map[runtime.EventType]int
	answerChars    int
	reasoningChars int
	rawEvents      int
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
		case "raw":
			fp.rawEvents++
		}
	}
	return fp
}

// compareFingerprints is deliberately scoped to task-effect semantics. OpenCode
// emits lifecycle metadata (session.updated, message.updated, session.diff, etc.)
// on timing-dependent paths and may split the same semantic content into a
// different number of streaming deltas. Those transport/lifecycle details are
// covered by G13's zero-silent-drop mapper gate, not by G12.1 task parity.
//
// Task parity still fails closed when a semantic capability class that existed
// in the baseline disappears entirely from the candidate.
func compareFingerprints(bEvents, cEvents []runtime.Event, a Assertion) []string {
	bFP := computeFingerprint(bEvents)
	cFP := computeFingerprint(cEvents)
	var issues []string

	if a.RequireAnswer && bFP.answerChars > 0 && cFP.answerChars == 0 {
		issues = append(issues, fmt.Sprintf("answer.delta content: baseline has %d chars, candidate has 0", bFP.answerChars))
	}
	if a.RequireReasoning && bFP.reasoningChars > 0 && cFP.reasoningChars == 0 {
		issues = append(issues, fmt.Sprintf("reasoning.delta content: baseline has %d chars, candidate has 0", bFP.reasoningChars))
	}
	if a.RequireRaw && bFP.rawEvents > 0 && cFP.rawEvents == 0 {
		issues = append(issues, "candidate missing raw event required by scenario")
	}

	semanticTypes := []runtime.EventType{
		"answer.delta",
		"reasoning.delta",
		"tool.called",
		"tool.success",
		"tool.failed",
		"approval.requested",
		"approval.resolved",
	}
	for _, typ := range semanticTypes {
		if bFP.counts[typ] > 0 && cFP.counts[typ] == 0 {
			issues = append(issues, fmt.Sprintf("candidate missing semantic event type %q present in baseline", typ))
		}
	}
	return issues
}

func hasEventType(events []runtime.Event, typ runtime.EventType) bool {
	for _, e := range events {
		if e.Type == typ {
			return true
		}
	}
	return false
}

func (r *Runner) collectEvents(ctx context.Context, rt runtime.AgentRuntime, req *runtime.PromptRequest, approvalDecision *runtime.ApprovalDecision, assertions Assertion, timeout time.Duration, drainRawTail bool) ([]runtime.Event, error) {
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
	var inactivity <-chan time.Time
	var terminalWait <-chan time.Time
	var semanticWait <-chan time.Time
	terminalSeen := false
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				return events, nil
			}
			if ev.SessionID != "" && ev.SessionID != session.ID {
				continue
			}
			events = append(events, ev)

			if approvalDecision != nil && ev.Type == "approval.requested" && ev.Approval != nil && ev.Approval.ID != "" {
				if err := rt.ReplyApproval(ctx, runtime.ApprovalID(ev.Approval.ID), runtime.ApprovalReply{Decision: *approvalDecision}); err != nil {
					return events, fmt.Errorf("ReplyApproval(%s): %w", ev.Approval.ID, err)
				}
			}

			terminal := ev.RawType == "session.idle" || ev.Type == "step.finished" || ev.Type == "step.failed"
			requiredSatisfied := checkAssertions(events, assertions).ok
			if !requiredSatisfied {
				inactivity = nil
				if terminal {
					terminalSeen = true
					if terminalWait == nil {
						terminalWait = time.After(terminalGrace)
					}
				}
				if isSemanticParityEvent(ev) && semanticWait == nil {
					semanticWait = time.After(semanticGrace)
				}
				continue
			}

			semanticWait = nil
			if terminalSeen {
				if drainRawTail {
					terminalWait = nil
					inactivity = time.After(inactivityFallback)
					continue
				}
				return events, nil
			}
			if ev.RawType == "session.idle" {
				return events, nil
			}
			if (ev.Type == "step.finished" || ev.Type == "step.failed") && !drainRawTail {
				return events, nil
			}
			if isSemanticParityEvent(ev) || (drainRawTail && terminal) {
				inactivity = time.After(inactivityFallback)
			}
		case <-terminalWait:
			return events, nil
		case <-semanticWait:
			return events, nil
		case <-inactivity:
			return events, nil
		case <-deadline:
			return events, nil
		case <-ctx.Done():
			return events, ctx.Err()
		}
	}
}

func isSemanticParityEvent(ev runtime.Event) bool {
	switch ev.Type {
	case "answer.delta", "reasoning.delta", "tool.called", "tool.success", "tool.failed", "approval.requested", "approval.resolved", "raw":
		return true
	default:
		return false
	}
}

func ConcatContent(events []runtime.Event, typ runtime.EventType) string {
	var b strings.Builder
	for _, e := range events {
		if e.Type == typ {
			b.WriteString(e.Content)
		}
	}
	return b.String()
}
