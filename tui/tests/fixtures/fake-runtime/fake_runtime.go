package fakeruntime

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"codea/tui/internal/runtime"
)

// ErrSimulated is a sentinel error for tests that configure simulated failures.
var ErrSimulated = errors.New("simulated error")

// PromptRecord captures a recorded Prompt call.
type PromptRecord struct {
	SessionID runtime.SessionID
	Request   runtime.PromptRequest
}

// ApprovalRecord captures a recorded ReplyApproval call.
type ApprovalRecord struct {
	ID    runtime.ApprovalID
	Reply runtime.ApprovalReply
}

// FakeRuntime is a test fake that implements runtime.AgentRuntime.
// It records all calls and allows tests to configure responses and events.
type FakeRuntime struct {
	HealthInfo  runtime.HealthInfo
	HealthError error

	CapabilitiesConfig runtime.RuntimeCapabilities

	Agents     []runtime.Agent
	AgentsErr  error
	SessionErr error

	// Events are sent to subscribers when Prompt is called.
	Events []runtime.Event

	// EventDelay, if > 0, makes Prompt sleep before publishing events.
	// Use this to test timeout / inactivity behavior with slow runtimes.
	EventDelay time.Duration

	// CancelError, if set, is returned by Cancel.
	CancelError error

	// ReplyApprovalError, if set, is returned by ReplyApproval.
	ReplyApprovalError error

	// ApprovalOnceEvents are sent to subscribers when ReplyApproval is called
	// with ApprovalOnce, simulating the runtime continuing after approval.
	ApprovalOnceEvents []runtime.Event

	mu                sync.Mutex
	sessions          map[runtime.SessionID]*runtime.Session
	prompts           []PromptRecord
	approvals         []ApprovalRecord
	cancelledSessions []runtime.SessionID
	subscribers       map[chan runtime.Event]context.Context
	nextSessionID     int
}

// New returns an initialized FakeRuntime.
func New() *FakeRuntime {
	return &FakeRuntime{
		HealthInfo:  runtime.HealthInfo{Healthy: true},
		sessions:    make(map[runtime.SessionID]*runtime.Session),
		subscribers: make(map[chan runtime.Event]context.Context),
	}
}

// compile-time interface check
var _ runtime.AgentRuntime = (*FakeRuntime)(nil)

func (f *FakeRuntime) Health(ctx context.Context) (runtime.HealthInfo, error) {
	if f.HealthError != nil {
		return runtime.HealthInfo{}, f.HealthError
	}
	return f.HealthInfo, nil
}

func (f *FakeRuntime) CreateSession(ctx context.Context, req runtime.CreateSessionRequest) (runtime.Session, error) {
	if f.SessionErr != nil {
		return runtime.Session{}, f.SessionErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.nextSessionID++
	sid := runtime.SessionID(fmt.Sprintf("fake-session-%d", f.nextSessionID))
	s := &runtime.Session{ID: string(sid)}
	f.sessions[sid] = s
	return *s, nil
}

func (f *FakeRuntime) Prompt(ctx context.Context, sessionID runtime.SessionID, req runtime.PromptRequest) error {
	f.mu.Lock()
	f.prompts = append(f.prompts, PromptRecord{SessionID: sessionID, Request: req})
	delay := f.EventDelay

	if delay > 0 {
		// Delayed publish: copy state, unlock, then sleep and send.
		events := make([]runtime.Event, len(f.Events))
		copy(events, f.Events)
		subs := make(map[chan runtime.Event]context.Context, len(f.subscribers))
		for ch, sctx := range f.subscribers {
			subs[ch] = sctx
		}
		f.mu.Unlock()

		select {
		case <-time.After(delay):
		case <-ctx.Done():
			return ctx.Err()
		}

		for _, ev := range events {
			for ch, sctx := range subs {
				select {
				case ch <- ev:
				case <-sctx.Done():
				default:
				}
			}
		}
		return nil
	}

	for _, ev := range f.Events {
		for ch, sctx := range f.subscribers {
			select {
			case ch <- ev:
			case <-sctx.Done():
			default:
			}
		}
	}
	f.mu.Unlock()
	return nil
}

func (f *FakeRuntime) Subscribe(ctx context.Context) (<-chan runtime.Event, error) {
	ch := make(chan runtime.Event, 64)
	f.mu.Lock()
	f.subscribers[ch] = ctx
	f.mu.Unlock()

	go func() {
		<-ctx.Done()
		f.mu.Lock()
		delete(f.subscribers, ch)
		close(ch)
		f.mu.Unlock()
	}()

	return ch, nil
}

func (f *FakeRuntime) ReplyApproval(ctx context.Context, approvalID runtime.ApprovalID, reply runtime.ApprovalReply) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if f.ReplyApprovalError != nil {
		return f.ReplyApprovalError
	}

	f.approvals = append(f.approvals, ApprovalRecord{ID: approvalID, Reply: reply})

	// Simulate post-approval behavior: on "once", the runtime continues and
	// emits continuation events (e.g. tool calls). On "reject", no
	// continuation events are emitted — the blocked operation does not run.
	if reply.Decision == runtime.ApprovalOnce {
		for _, ev := range f.ApprovalOnceEvents {
			for ch, sctx := range f.subscribers {
				select {
				case ch <- ev:
				case <-sctx.Done():
				default:
				}
			}
		}
	}

	return nil
}

func (f *FakeRuntime) Cancel(ctx context.Context, sessionID runtime.SessionID) error {
	if f.CancelError != nil {
		return f.CancelError
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cancelledSessions = append(f.cancelledSessions, sessionID)
	return nil
}

func (f *FakeRuntime) ListAgents(ctx context.Context) ([]runtime.Agent, error) {
	if f.AgentsErr != nil {
		return nil, f.AgentsErr
	}
	return f.Agents, nil
}

func (f *FakeRuntime) Capabilities() runtime.RuntimeCapabilities {
	return f.CapabilitiesConfig
}

// Prompts returns all recorded Prompt calls.
func (f *FakeRuntime) Prompts() []PromptRecord {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]PromptRecord, len(f.prompts))
	copy(out, f.prompts)
	return out
}

// LastPrompt returns the agent of the most recent prompt call, satisfying
// the parity.PromptRecorder interface.
func (f *FakeRuntime) LastPrompt() (agent string, ok bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.prompts) == 0 {
		return "", false
	}
	return f.prompts[len(f.prompts)-1].Request.Agent, true
}

// Approvals returns all recorded ReplyApproval calls.
func (f *FakeRuntime) Approvals() []ApprovalRecord {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]ApprovalRecord, len(f.approvals))
	copy(out, f.approvals)
	return out
}

// CancelledSessions returns all session IDs passed to Cancel.
func (f *FakeRuntime) CancelledSessions() []runtime.SessionID {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([]runtime.SessionID, len(f.cancelledSessions))
	copy(out, f.cancelledSessions)
	return out
}
