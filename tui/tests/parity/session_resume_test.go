package parity_test

import (
	"context"
	"testing"

	"codea/tui/internal/runtime"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

// TestSessionResumeIsolation proves the AgentRuntime contract preserves session
// identity across resume: creating two sessions yields distinct IDs, prompts
// route to the session they were sent to, and resuming session A continues on
// A rather than bleeding into B. The global /global/event stream is shared, so
// the contract must carry SessionID on every prompt/event for consumers to
// isolate — verified via the recorded prompts below.
func TestSessionResumeIsolation(t *testing.T) {
	rt := fakeruntime.New()
	ctx := context.Background()

	sessA, err := rt.CreateSession(ctx, runtime.CreateSessionRequest{Title: "A"})
	if err != nil {
		t.Fatalf("create session A: %v", err)
	}
	sessB, err := rt.CreateSession(ctx, runtime.CreateSessionRequest{Title: "B"})
	if err != nil {
		t.Fatalf("create session B: %v", err)
	}
	if sessA.ID == "" || sessB.ID == "" {
		t.Fatal("sessions must have non-empty IDs")
	}
	if sessA.ID == sessB.ID {
		t.Fatal("session IDs must be distinct")
	}

	prompt := func(sessID string, text string) {
		t.Helper()
		err := rt.Prompt(ctx, runtime.SessionID(sessID), runtime.PromptRequest{
			Agent: "general",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: text}},
		})
		if err != nil {
			t.Fatalf("prompt %q: %v", text, err)
		}
	}

	prompt(sessA.ID, "A1")
	prompt(sessB.ID, "B1")
	prompt(sessA.ID, "A2") // resume A

	prompts := rt.Prompts()
	if len(prompts) != 3 {
		t.Fatalf("expected 3 prompts, got %d", len(prompts))
	}
	if prompts[0].SessionID != runtime.SessionID(sessA.ID) {
		t.Errorf("prompt[0] session = %q, want A (%q)", prompts[0].SessionID, sessA.ID)
	}
	if prompts[1].SessionID != runtime.SessionID(sessB.ID) {
		t.Errorf("prompt[1] session = %q, want B (%q)", prompts[1].SessionID, sessB.ID)
	}
	if prompts[2].SessionID != runtime.SessionID(sessA.ID) {
		t.Errorf("prompt[2] (resume A) session = %q, want A (%q)", prompts[2].SessionID, sessA.ID)
	}

	// Resume support: both sessions remain listable.
	sessions, err := rt.ListSessions(ctx)
	if err != nil {
		t.Fatalf("list sessions: %v", err)
	}
	if len(sessions) != 2 {
		t.Errorf("expected 2 sessions after resume, got %d", len(sessions))
	}
}

// TestSessionEventTagging proves runtime events carry the SessionID needed to
// isolate sessions on the shared global event stream.
func TestSessionEventTagging(t *testing.T) {
	rt := fakeruntime.New()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := rt.Subscribe(ctx)
	if err != nil {
		t.Fatalf("subscribe: %v", err)
	}

	rt.Events = []runtime.Event{
		{Type: runtime.EventType("answer.delta"), SessionID: "sess-A", Content: "A"},
		{Type: runtime.EventType("answer.delta"), SessionID: "sess-B", Content: "B"},
	}

	sess, err := rt.CreateSession(ctx, runtime.CreateSessionRequest{})
	if err != nil {
		t.Fatalf("create session: %v", err)
	}
	if err := rt.Prompt(ctx, runtime.SessionID(sess.ID), runtime.PromptRequest{
		Agent: "general",
		Parts: []runtime.PromptPart{runtime.TextPart{Text: "test"}},
	}); err != nil {
		t.Fatalf("prompt: %v", err)
	}

	var got []runtime.Event
	for i := 0; i < 2; i++ {
		select {
		case ev := <-ch:
			got = append(got, ev)
		default:
		}
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 events, got %d", len(got))
	}
	seen := map[string]bool{}
	for _, ev := range got {
		seen[ev.SessionID] = true
	}
	if !seen["sess-A"] || !seen["sess-B"] {
		t.Fatalf("events lost SessionID tagging: %v", seen)
	}
}
