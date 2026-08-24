package parity

import (
	"context"
	"sync"
	"testing"
	"time"

	"codea/tui/internal/runtime"
)

type terminalOrderRuntime struct {
	mu       sync.Mutex
	subs     []chan runtime.Event
	schedule func(chan runtime.Event)
}

func (r *terminalOrderRuntime) Health(context.Context) (runtime.HealthInfo, error) {
	return runtime.HealthInfo{Healthy: true}, nil
}
func (r *terminalOrderRuntime) CreateSession(context.Context, runtime.CreateSessionRequest) (runtime.Session, error) {
	return runtime.Session{ID: "terminal-session"}, nil
}
func (r *terminalOrderRuntime) Subscribe(context.Context) (<-chan runtime.Event, error) {
	ch := make(chan runtime.Event, 16)
	r.mu.Lock()
	r.subs = append(r.subs, ch)
	r.mu.Unlock()
	return ch, nil
}
func (r *terminalOrderRuntime) Prompt(context.Context, runtime.SessionID, runtime.PromptRequest) error {
	r.mu.Lock()
	subs := append([]chan runtime.Event(nil), r.subs...)
	r.mu.Unlock()
	for _, ch := range subs {
		go r.schedule(ch)
	}
	return nil
}
func (r *terminalOrderRuntime) ReplyApproval(context.Context, runtime.ApprovalID, runtime.ApprovalReply) error {
	return nil
}
func (r *terminalOrderRuntime) Cancel(context.Context, runtime.SessionID) error { return nil }
func (r *terminalOrderRuntime) ListAgents(context.Context) ([]runtime.Agent, error) { return nil, nil }
func (r *terminalOrderRuntime) ListSessions(context.Context) ([]runtime.Session, error) { return nil, nil }
func (r *terminalOrderRuntime) GetSessionMessages(context.Context, runtime.SessionID) ([]runtime.Message, error) {
	return nil, nil
}
func (r *terminalOrderRuntime) Capabilities() runtime.RuntimeCapabilities { return runtime.RuntimeCapabilities{} }

func newTerminalOrderRuntime(schedule func(chan runtime.Event)) *terminalOrderRuntime {
	return &terminalOrderRuntime{schedule: schedule}
}

func TestRunnerDoesNotAcceptSessionIdleBeforeRequiredAnswer(t *testing.T) {
	schedule := func(ch chan runtime.Event) {
		ch <- runtime.Event{Type: runtime.EventType("raw"), RawType: "session.idle", SessionID: "terminal-session", Raw: []byte(`{"idle":true}`)}
		time.Sleep(700 * time.Millisecond)
		ch <- runtime.Event{Type: runtime.EventType("answer.delta"), SessionID: "terminal-session", Content: "late answer"}
		ch <- runtime.Event{Type: runtime.EventType("step.finished"), SessionID: "terminal-session"}
	}
	runner := Runner{Baseline: newTerminalOrderRuntime(schedule), Candidate: newTerminalOrderRuntime(schedule)}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := runner.Run(ctx, Scenario{
		Name: "Prompt", Required: true, RepeatCount: 1, Timeout: 2 * time.Second,
		Prompt: &runtime.PromptRequest{Agent: "general", Parts: []runtime.PromptPart{runtime.TextPart{Text: "hello"}}},
		Assertions: Assertion{RequireAnswer: true},
	})
	if !result.Passed {
		t.Fatalf("session.idle must not end collection before required answer semantics: %+v", result.Failures)
	}
}

func TestRunnerDoesNotStartInactivityUntilAllRequiredSemanticsArrive(t *testing.T) {
	schedule := func(ch chan runtime.Event) {
		ch <- runtime.Event{Type: runtime.EventType("answer.delta"), SessionID: "terminal-session", Content: "answer first"}
		time.Sleep(700 * time.Millisecond)
		ch <- runtime.Event{Type: runtime.EventType("reasoning.delta"), SessionID: "terminal-session", Content: "late reasoning"}
		ch <- runtime.Event{Type: runtime.EventType("step.finished"), SessionID: "terminal-session"}
	}
	runner := Runner{Baseline: newTerminalOrderRuntime(schedule), Candidate: newTerminalOrderRuntime(schedule)}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := runner.Run(ctx, Scenario{
		Name: "Reasoning", Required: true, RepeatCount: 1, Timeout: 2 * time.Second,
		Prompt: &runtime.PromptRequest{Agent: "general", Parts: []runtime.PromptPart{runtime.TextPart{Text: "reasoning test"}}},
		Assertions: Assertion{RequireAnswer: true, RequireReasoning: true},
	})
	if !result.Passed {
		t.Fatalf("500ms inactivity must not end collection until every required semantic is present: %+v", result.Failures)
	}
}
