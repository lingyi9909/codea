package opencode

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"codea/tui/internal/runtime"
)

// Compile-time interface check.
var _ runtime.AgentRuntime = (*OpenCodeAdapter)(nil)

func TestOpenCodeAdapterHealth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/global/health" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(OpenCodeGlobalHealthResponse{Healthy: true, Version: "1.18.11"})
	}))
	defer srv.Close()

	adapter := NewOpenCodeAdapter(srv.URL, "", "")
	ctx := context.Background()

	info, err := adapter.Health(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !info.Healthy {
		t.Fatal("expected healthy")
	}
	if info.Version != "1.18.11" {
		t.Fatalf("expected version 1.18.11, got %q", info.Version)
	}
}

func TestOpenCodeAdapterCreateSession(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/session" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(OpenCodeSession{ID: "ses_test"})
	}))
	defer srv.Close()

	adapter := NewOpenCodeAdapter(srv.URL, "", "")
	ctx := context.Background()

	session, err := adapter.CreateSession(ctx, runtime.CreateSessionRequest{Title: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if session.ID != "ses_test" {
		t.Fatalf("expected session ID ses_test, got %q", session.ID)
	}
}

func TestOpenCodeAdapterPrompt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if r.URL.Path != "/session/ses_test/prompt_async" {
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	adapter := NewOpenCodeAdapter(srv.URL, "", "")
	ctx := context.Background()

	err := adapter.Prompt(ctx, "ses_test", runtime.PromptRequest{
		Parts: []runtime.PromptPart{runtime.TextPart{Text: "hello"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenCodeAdapterReplyApproval(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/permission/req_1/reply" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(OpenCodePermissionReplyResponse(true))
	}))
	defer srv.Close()

	adapter := NewOpenCodeAdapter(srv.URL, "", "")
	ctx := context.Background()

	err := adapter.ReplyApproval(ctx, "req_1", runtime.ApprovalReply{
		Decision: runtime.ApprovalOnce,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenCodeAdapterCancel(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/session/ses_test/abort" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(OpenCodeSessionAbortResponse(true))
	}))
	defer srv.Close()

	adapter := NewOpenCodeAdapter(srv.URL, "", "")
	ctx := context.Background()

	err := adapter.Cancel(ctx, "ses_test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestOpenCodeAdapterListAgents(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/agent" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(OpenCodeAppAgentsResponse{
			{Name: "build", Mode: "build"},
		})
	}))
	defer srv.Close()

	adapter := NewOpenCodeAdapter(srv.URL, "", "")
	ctx := context.Background()

	agents, err := adapter.ListAgents(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(agents))
	}
	if agents[0].Name != "build" || agents[0].Mode != "build" {
		t.Fatalf("unexpected agent: %+v", agents[0])
	}
}

func TestOpenCodeAdapterSubscribe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/global/event" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
			return
		}
		if r.Header.Get("Accept") != "text/event-stream" {
			t.Errorf("expected Accept: text/event-stream, got %q", r.Header.Get("Accept"))
		}
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		evt := `{"directory":"/tmp","payload":{"type":"session.status","properties":{"sessionID":"s1"}}}`
		w.Write([]byte("data: " + evt + "\n\n"))
		flusher.Flush()
		<-r.Context().Done()
	}))
	defer srv.Close()

	adapter := NewOpenCodeAdapter(srv.URL, "", "")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	ch, err := adapter.Subscribe(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	select {
	case evt, ok := <-ch:
		if !ok {
			t.Fatal("channel closed unexpectedly")
		}
		if evt.Type != "session.status" {
			t.Fatalf("expected type session.status, got %q", evt.Type)
		}
		if evt.SessionID != "s1" {
			t.Fatalf("expected SessionID=s1, got %q", evt.SessionID)
		}
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}
}

func TestAdapterBackpressureBoundedChannel(t *testing.T) {
	// Verify that the Subscribe channel is bounded and events are not dropped.
	// When the consumer is slow, backpressure propagates upstream (blocking send).

	var sent atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		for i := 0; i < 30; i++ {
			select {
			case <-r.Context().Done():
				return
			default:
			}
			fmt.Fprintf(w, "data: {\"payload\":{\"type\":\"answer.delta\",\"properties\":{\"content\":\"msg%d\"}}}\n\n", i)
			flusher.Flush()
			sent.Add(1)
		}
		<-r.Context().Done()
	}))
	defer srv.Close()

	adapter := NewOpenCodeAdapter(srv.URL, "", "")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	ch, err := adapter.Subscribe(ctx)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	// Slow consumer: 5ms delay between reads ensures the channel fills up
	// and backpressure propagates.
	var received int
	var backpressureCount int
	timeout := time.After(5 * time.Second)
loop:
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				break loop
			}
			received++
			if ev.Type == CodeaEventRuntimeError && ev.Error != nil &&
				ev.Error.Kind == runtime.RuntimeErrorBackpressure {
				backpressureCount++
			}
			time.Sleep(5 * time.Millisecond)
		case <-timeout:
			cancel()
			for range ch {
			}
			break loop
		}
	}

	if received < 20 {
		t.Errorf("received only %d events, expected at least 20 (no silent drops)", received)
	}
	if backpressureCount == 0 {
		t.Error("expected at least one RuntimeError(Backpressure) event when channel is full")
	}
	t.Logf("received %d events, %d backpressure errors with slow consumer", received, backpressureCount)
}
