package parity

import (
	"context"
	"testing"
	"time"

	"codea/tui/internal/runtime"
)

func TestV1AgentSelectionUsesNativeBuildAgent(t *testing.T) {
	for _, s := range V1RequiredScenarios() {
		if s.Name != "AgentSelection" {
			continue
		}
		if s.Prompt == nil {
			t.Fatal("AgentSelection prompt is nil")
		}
		if s.Prompt.Agent != "build" || s.Assertions.RequireAgent != "build" {
			t.Fatalf("AgentSelection must exercise native build agent, got prompt=%q assertion=%q", s.Prompt.Agent, s.Assertions.RequireAgent)
		}
		return
	}
	t.Fatal("AgentSelection scenario missing")
}

type orderedEventRuntime struct {
	events    []runtime.Event
	lastAgent string
}

func (r *orderedEventRuntime) Health(context.Context) (runtime.HealthInfo, error) { return runtime.HealthInfo{Healthy: true}, nil }
func (r *orderedEventRuntime) CreateSession(context.Context, runtime.CreateSessionRequest) (runtime.Session, error) { return runtime.Session{ID: "s1"}, nil }
func (r *orderedEventRuntime) Prompt(_ context.Context, _ runtime.SessionID, req runtime.PromptRequest) error { r.lastAgent = req.Agent; return nil }
func (r *orderedEventRuntime) Subscribe(ctx context.Context) (<-chan runtime.Event, error) {
	ch := make(chan runtime.Event, len(r.events))
	for _, ev := range r.events { ch <- ev }
	close(ch)
	return ch, nil
}
func (r *orderedEventRuntime) ReplyApproval(context.Context, runtime.ApprovalID, runtime.ApprovalReply) error { return nil }
func (r *orderedEventRuntime) Cancel(context.Context, runtime.SessionID) error { return nil }
func (r *orderedEventRuntime) ListAgents(context.Context) ([]runtime.Agent, error) { return nil, nil }
func (r *orderedEventRuntime) ListModels(context.Context) ([]runtime.Model, error) { return nil, nil }
func (r *orderedEventRuntime) ListSessions(context.Context) ([]runtime.Session, error) { return nil, nil }
func (r *orderedEventRuntime) GetSessionMessages(context.Context, runtime.SessionID) ([]runtime.Message, error) { return nil, nil }
func (r *orderedEventRuntime) CompactSession(context.Context, runtime.SessionID) error { return nil }
func (r *orderedEventRuntime) Capabilities() runtime.RuntimeCapabilities { return runtime.RuntimeCapabilities{} }
func (r *orderedEventRuntime) LastPrompt() (string, bool) { return r.lastAgent, r.lastAgent != "" }

func TestRawEventHandlingDrainsEventsAfterStepFinished(t *testing.T) {
	events := []runtime.Event{
		{Type: runtime.EventType("answer.delta"), Content: "ok"},
		{Type: runtime.EventType("step.finished")},
		{Type: runtime.EventType("raw"), RawType: "session.idle", Raw: []byte(`{"payload":{"type":"session.idle"}}`)},
	}
	baseline := &orderedEventRuntime{events: events}
	candidate := &orderedEventRuntime{events: events}
	runner := Runner{Baseline: baseline, Candidate: candidate}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	result := runner.Run(ctx, Scenario{
		Name: "RawEventHandling", Required: true, RepeatCount: 1,
		Prompt: &runtime.PromptRequest{Agent: "build", Parts: []runtime.PromptPart{runtime.TextPart{Text: "raw event test"}}},
		Assertions: Assertion{RequireRaw: true},
	})
	if !result.Passed {
		t.Fatalf("tail raw event after step.finished must be observed: %+v", result.Failures)
	}
}
