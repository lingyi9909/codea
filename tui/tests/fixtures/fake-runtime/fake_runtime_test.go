package fakeruntime_test

import (
	"context"
	"testing"

	"codea/tui/internal/runtime"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

func TestImplementsAgentRuntime(t *testing.T) {
	// Compile-time check.
	var _ runtime.AgentRuntime = fakeruntime.New()
}

func TestHealth(t *testing.T) {
	rt := fakeruntime.New()
	rt.HealthInfo = runtime.HealthInfo{Healthy: true, Version: "test-v1"}

	info, err := rt.Health(context.Background())
	if err != nil {
		t.Fatalf("Health failed: %v", err)
	}
	if !info.Healthy {
		t.Error("should be healthy")
	}
	if info.Version != "test-v1" {
		t.Errorf("expected version test-v1, got %s", info.Version)
	}
}

func TestHealthError(t *testing.T) {
	rt := fakeruntime.New()
	rt.HealthError = fakeruntime.ErrSimulated

	_, err := rt.Health(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCreateSession(t *testing.T) {
	rt := fakeruntime.New()

	session, err := rt.CreateSession(context.Background(), runtime.CreateSessionRequest{Title: "test"})
	if err != nil {
		t.Fatalf("CreateSession failed: %v", err)
	}
	if session.ID == "" {
		t.Error("session ID should not be empty")
	}
}

func TestPromptRecordsRequest(t *testing.T) {
	rt := fakeruntime.New()

	session, _ := rt.CreateSession(context.Background(), runtime.CreateSessionRequest{Title: "test"})
	req := runtime.PromptRequest{
		MessageID: "msg-1",
		Agent:     "general",
		Parts:     []runtime.PromptPart{runtime.TextPart{ID: "p1", Text: "hello"}},
	}

	err := rt.Prompt(context.Background(), runtime.SessionID(session.ID), req)
	if err != nil {
		t.Fatalf("Prompt failed: %v", err)
	}

	records := rt.Prompts()
	if len(records) != 1 {
		t.Fatalf("expected 1 prompt record, got %d", len(records))
	}
	if records[0].Request.MessageID != "msg-1" {
		t.Errorf("expected msg-1, got %s", records[0].Request.MessageID)
	}
	if records[0].SessionID != runtime.SessionID(session.ID) {
		t.Errorf("session ID mismatch")
	}
}

func TestSubscribeReceivesEvents(t *testing.T) {
	rt := fakeruntime.New()

	session, _ := rt.CreateSession(context.Background(), runtime.CreateSessionRequest{})

	// Pre-configure events to emit when Prompt is called.
	rt.Events = []runtime.Event{
		{Type: runtime.EventType("answer.delta"), Content: "hello"},
		{Type: runtime.EventType("step.finished"), Content: "done"},
	}

	ch, err := rt.Subscribe(context.Background())
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	go func() {
		_ = rt.Prompt(context.Background(), runtime.SessionID(session.ID), runtime.PromptRequest{
			MessageID: "msg-1",
		})
	}()

	var events []runtime.Event
	for e := range ch {
		events = append(events, e)
		if len(events) >= 2 {
			break
		}
	}

	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
	if events[0].Content != "hello" {
		t.Errorf("expected 'hello', got %q", events[0].Content)
	}
}

func TestSubscribeContextCancellation(t *testing.T) {
	rt := fakeruntime.New()

	ctx, cancel := context.WithCancel(context.Background())
	ch, err := rt.Subscribe(ctx)
	if err != nil {
		t.Fatalf("Subscribe failed: %v", err)
	}

	cancel()

	// Channel should close after context cancellation.
	_, ok := <-ch
	if ok {
		// Drain and check
		for range ch {
		}
	}
}

func TestReplyApproval(t *testing.T) {
	rt := fakeruntime.New()

	err := rt.ReplyApproval(context.Background(), runtime.ApprovalID("approval-1"), runtime.ApprovalReply{
		Decision: runtime.ApprovalOnce,
		Message:  "ok",
	})
	if err != nil {
		t.Fatalf("ReplyApproval failed: %v", err)
	}

	records := rt.Approvals()
	if len(records) != 1 {
		t.Fatalf("expected 1 approval record, got %d", len(records))
	}
	if records[0].ID != runtime.ApprovalID("approval-1") {
		t.Errorf("expected approval-1, got %s", records[0].ID)
	}
	if records[0].Reply.Decision != runtime.ApprovalOnce {
		t.Errorf("expected once, got %s", records[0].Reply.Decision)
	}
}

func TestCancel(t *testing.T) {
	rt := fakeruntime.New()

	session, _ := rt.CreateSession(context.Background(), runtime.CreateSessionRequest{})
	err := rt.Cancel(context.Background(), runtime.SessionID(session.ID))
	if err != nil {
		t.Fatalf("Cancel failed: %v", err)
	}

	cancelled := rt.CancelledSessions()
	if len(cancelled) != 1 {
		t.Fatalf("expected 1 cancelled session, got %d", len(cancelled))
	}
}

func TestListAgents(t *testing.T) {
	rt := fakeruntime.New()
	rt.Agents = []runtime.Agent{
		{Name: "general", Mode: "native"},
		{Name: "reviewer", Mode: "enterprise"},
	}

	agents, err := rt.ListAgents(context.Background())
	if err != nil {
		t.Fatalf("ListAgents failed: %v", err)
	}
	if len(agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(agents))
	}
}

func TestCapabilities(t *testing.T) {
	rt := fakeruntime.New()
	rt.CapabilitiesConfig = runtime.RuntimeCapabilities{
		Sessions:     true,
		Streaming:    true,
		ToolApproval: true,
	}

	caps := rt.Capabilities()
	if !caps.Sessions {
		t.Error("sessions should be enabled")
	}
	if !caps.Streaming {
		t.Error("streaming should be enabled")
	}
	if caps.Reasoning {
		t.Error("reasoning should be disabled by default")
	}
}
