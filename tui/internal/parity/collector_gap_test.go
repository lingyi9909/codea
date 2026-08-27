package parity

import (
	"context"
	"sync"
	"testing"
	"time"

	"codea/tui/internal/runtime"
)

type lifecycleGapRuntime struct {
	mu   sync.Mutex
	subs []chan runtime.Event
}

func (r *lifecycleGapRuntime) Health(context.Context) (runtime.HealthInfo, error) {
	return runtime.HealthInfo{Healthy: true}, nil
}
func (r *lifecycleGapRuntime) CreateSession(context.Context, runtime.CreateSessionRequest) (runtime.Session, error) {
	return runtime.Session{ID: "gap-session"}, nil
}
func (r *lifecycleGapRuntime) Subscribe(ctx context.Context) (<-chan runtime.Event, error) {
	ch := make(chan runtime.Event, 8)
	r.mu.Lock()
	r.subs = append(r.subs, ch)
	r.mu.Unlock()
	go func() {
		<-ctx.Done()
	}()
	return ch, nil
}
func (r *lifecycleGapRuntime) Prompt(context.Context, runtime.SessionID, runtime.PromptRequest) error {
	r.mu.Lock()
	subs := append([]chan runtime.Event(nil), r.subs...)
	r.mu.Unlock()
	go func() {
		for _, ch := range subs {
			ch <- runtime.Event{Type: runtime.EventType("session.status"), SessionID: "gap-session"}
		}
		time.Sleep(700 * time.Millisecond)
		for _, ch := range subs {
			ch <- runtime.Event{Type: runtime.EventType("answer.delta"), SessionID: "gap-session", Content: "late answer"}
			ch <- runtime.Event{Type: runtime.EventType("step.finished"), SessionID: "gap-session"}
		}
	}()
	return nil
}
func (r *lifecycleGapRuntime) ReplyApproval(context.Context, runtime.ApprovalID, runtime.ApprovalReply) error {
	return nil
}
func (r *lifecycleGapRuntime) Cancel(context.Context, runtime.SessionID) error { return nil }
func (r *lifecycleGapRuntime) ListAgents(context.Context) ([]runtime.Agent, error) { return nil, nil }
func (r *lifecycleGapRuntime) ListModels(context.Context) ([]runtime.Model, error) { return nil, nil }
func (r *lifecycleGapRuntime) ListSessions(context.Context) ([]runtime.Session, error) { return nil, nil }
func (r *lifecycleGapRuntime) GetSessionMessages(context.Context, runtime.SessionID) ([]runtime.Message, error) {
	return nil, nil
}
func (r *lifecycleGapRuntime) CompactSession(context.Context, runtime.SessionID) error { return nil }
func (r *lifecycleGapRuntime) Capabilities() runtime.RuntimeCapabilities { return runtime.RuntimeCapabilities{} }

func TestRunnerDoesNotTreatPreAnswerLifecycleGapAsCompletion(t *testing.T) {
	baseline := &lifecycleGapRuntime{}
	candidate := &lifecycleGapRuntime{}
	runner := Runner{Baseline: baseline, Candidate: candidate}
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	result := runner.Run(ctx, Scenario{
		Name: "Prompt", Required: true, RepeatCount: 1, Timeout: 2 * time.Second,
		Prompt: &runtime.PromptRequest{Agent: "build", Parts: []runtime.PromptPart{runtime.TextPart{Text: "hello"}}},
		Assertions: Assertion{RequireAnswer: true},
	})
	if !result.Passed {
		t.Fatalf("lifecycle activity must not start the 500ms completion timer before required answer semantics arrive: %+v", result.Failures)
	}
}
