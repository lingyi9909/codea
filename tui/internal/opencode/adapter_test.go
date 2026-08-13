package opencode

import (
	"context"
	"encoding/json"
	"errors"
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
		// Handle recovery API calls (triggered by ReconnectHook).
		if r.URL.Path == "/session" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]OpenCodeSessionV2Info{})
			return
		}
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

	// Drain recovery marker events injected by ReconnectHook.
	var evt runtime.Event
	select {
	case e, ok := <-ch:
		if !ok {
			t.Fatal("channel closed unexpectedly")
		}
		evt = e
	case <-time.After(time.Second):
		t.Fatal("timeout waiting for event")
	}

	// If first event is the recovery-connected marker, read next.
	if evt.Type == CodeaEventRuntimeConnected && evt.Metadata["recovered"] == "true" {
		select {
		case e, ok := <-ch:
			if !ok {
				t.Fatal("channel closed before receiving session.status")
			}
			evt = e
		case <-time.After(time.Second):
			t.Fatal("timeout waiting for session.status after recovery marker")
		}
	}

	if evt.Type != "session.status" {
		t.Fatalf("expected type session.status, got %q", evt.Type)
	}
	if evt.SessionID != "s1" {
		t.Fatalf("expected SessionID=s1, got %q", evt.SessionID)
	}
}

func TestAdapterBackpressureBoundedChannel(t *testing.T) {
	// Verify that the Subscribe channel is bounded and events are not dropped.
	// When the consumer is slow, backpressure propagates upstream (blocking send).

	var sent atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/session" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]OpenCodeSessionV2Info{})
			return
		}
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
	var domainEvents int
	var backpressureErrors int
	timeout := time.After(5 * time.Second)
loop:
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				break loop
			}
			// Skip recovery/infra events.
			if ev.Type == CodeaEventRuntimeConnected {
				continue
			}
			if ev.Type == CodeaEventRuntimeError {
				if ev.Error != nil && ev.Error.Kind == runtime.RuntimeErrorBackpressure {
					backpressureErrors++
				}
				continue
			}
			domainEvents++
			time.Sleep(5 * time.Millisecond)
		case <-timeout:
			cancel()
			for range ch {
			}
			break loop
		}
	}

	if domainEvents != 30 {
		t.Errorf("expected exactly 30 domain events, got %d (no silent drops)", domainEvents)
	}
	if backpressureErrors == 0 {
		t.Error("expected at least one RuntimeError(Backpressure) event when channel is full")
	}
	t.Logf("domain events: %d, backpressure errors: %d", domainEvents, backpressureErrors)
}

func TestSSE401EmitsAuthRuntimeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	adapter := NewOpenCodeAdapter(srv.URL, "", "")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ch, err := adapter.Subscribe(ctx)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	var sawAuth bool
	deadline := time.After(3 * time.Second)
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				if !sawAuth {
					t.Error("expected auth RuntimeError event before channel close on 401")
				}
				return
			}
			if ev.Type == CodeaEventRuntimeError && ev.Error != nil && ev.Error.Kind == runtime.RuntimeErrorAuth {
				sawAuth = true
			}
		case <-deadline:
			t.Fatal("channel did not close within 3s for 401")
		}
	}
}

func TestSSE400EmitsProtocolRuntimeError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	adapter := NewOpenCodeAdapter(srv.URL, "", "")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	ch, err := adapter.Subscribe(ctx)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	var sawProtocol bool
	deadline := time.After(3 * time.Second)
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				if !sawProtocol {
					t.Error("expected Protocol RuntimeError event before channel close on 400")
				}
				return
			}
			if ev.Type == CodeaEventRuntimeError && ev.Error != nil && ev.Error.Kind == runtime.RuntimeErrorProtocol {
				sawProtocol = true
			}
		case <-deadline:
			t.Fatal("channel did not close within 3s for 400")
		}
	}
}

func TestAdapterDedupsRecoveryAndLiveMessage(t *testing.T) {
	// Construct the race the reviewer flagged: message M is present in the
	// recovery REST snapshot AND its live message.updated has already entered
	// the post-reconnect SSE buffer. The Application must receive M's
	// message.updated exactly once (the recovered one), not twice.
	var sseConnects atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Recovery REST endpoints.
		if r.URL.Path == "/session" {
			json.NewEncoder(w).Encode([]OpenCodeSessionV2Info{
				{ID: "S", Title: "Test", Time: OpenCodeSessionV2InfoTime{Created: 1000}},
			})
			return
		}
		if r.URL.Path == "/session/S/message" {
			json.NewEncoder(w).Encode([]OpenCodeSessionMessage{
				map[string]any{
					"info":  map[string]any{"id": "M", "role": "assistant"},
					"parts": []map[string]any{{"id": "P1", "type": "text"}},
				},
			})
			return
		}

		// SSE stream.
		if r.URL.Path == "/global/event" {
			connNum := sseConnects.Add(1)
			flusher, _ := w.(http.Flusher)
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			flusher.Flush()

			if connNum == 1 {
				// First connection: record session S, then close → disconnect.
				fmt.Fprintf(w, "data: {\"directory\":\"/tmp\",\"payload\":{\"type\":\"session.created\",\"properties\":{\"sessionID\":\"S\"}}}\n\n")
				flusher.Flush()
				return
			}

			// Second connection: the live message.updated for M races into the
			// buffer during recovery, followed by a sentinel we can wait on.
			fmt.Fprintf(w, "data: {\"directory\":\"/tmp\",\"payload\":{\"type\":\"message.updated\",\"properties\":{\"sessionID\":\"S\",\"info\":{\"id\":\"M\"}}}}\n\n")
			flusher.Flush()
			fmt.Fprintf(w, "data: {\"directory\":\"/tmp\",\"payload\":{\"type\":\"session.status\",\"properties\":{\"sessionID\":\"S\",\"status\":{\"type\":\"idle\"}}}}\n\n")
			flusher.Flush()
			<-r.Context().Done()
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	adapter := NewOpenCodeAdapter(srv.URL, "", "")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ch, err := adapter.Subscribe(ctx)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	msgUpdates := 0
	recoveredMsg := false
	liveMsg := false
	for ev := range ch {
		if ev.Type == CodeaEventMessageUpdated && ev.MessageID == "M" {
			msgUpdates++
			if ev.Metadata["recovered"] == "true" {
				recoveredMsg = true
			} else {
				liveMsg = true
			}
		}
		// Sentinel: the adapter has already processed (and suppressed) the
		// live M by the time this arrives.
		if ev.Type == "session.status" {
			cancel()
		}
	}

	if msgUpdates != 1 {
		t.Errorf("expected exactly 1 message.updated for M (dedup), got %d", msgUpdates)
	}
	if !recoveredMsg {
		t.Error("expected the recovered message.updated for M to be delivered")
	}
	if liveMsg {
		t.Error("live message.updated for M should have been deduped against recovery")
	}
}

func TestHTTPAdapterTransportErrorClassification(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	adapter := NewOpenCodeAdapter(srv.URL, "", "")
	_, err := adapter.Health(context.Background())
	if err == nil {
		t.Fatal("expected error for HTTP 500")
	}
	if !runtime.IsTransport(err) {
		t.Fatalf("expected Transport error, got %v (kind: %v)", err, runtimeErrorKind(err))
	}
}

func TestHTTPAdapterCancelledClassification(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer srv.Close()

	adapter := NewOpenCodeAdapter(srv.URL, "", "")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := adapter.Health(ctx)
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
	if !runtime.IsCancelled(err) {
		t.Fatalf("expected Cancelled error, got %v (kind: %v)", err, runtimeErrorKind(err))
	}
}

func TestHTTPAdapterProtocolErrorClassification(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	adapter := NewOpenCodeAdapter(srv.URL, "", "")
	_, err := adapter.Health(context.Background())
	if err == nil {
		t.Fatal("expected error for HTTP 400")
	}
	if !runtime.IsProtocol(err) {
		t.Fatalf("expected Protocol error, got %v (kind: %v)", err, runtimeErrorKind(err))
	}
}

func TestHTTPAdapterErrorVendorDetails(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad request body"))
	}))
	defer srv.Close()

	adapter := NewOpenCodeAdapter(srv.URL, "", "")
	_, err := adapter.Health(context.Background())
	if err == nil {
		t.Fatal("expected error for HTTP 400")
	}
	var rerr *runtime.RuntimeError
	if !errors.As(err, &rerr) {
		t.Fatalf("expected *RuntimeError, got %T", err)
	}
	if len(rerr.VendorDetails) == 0 {
		t.Fatal("VendorDetails must be auto-populated from HTTPError")
	}
	var vd map[string]any
	if err := json.Unmarshal(rerr.VendorDetails, &vd); err != nil {
		t.Fatalf("VendorDetails is not valid JSON: %v", err)
	}
	if vd["statusCode"] != float64(http.StatusBadRequest) {
		t.Errorf("VendorDetails.statusCode = %v, want 400", vd["statusCode"])
	}
	if vd["method"] != http.MethodGet {
		t.Errorf("VendorDetails.method = %v, want GET", vd["method"])
	}
	if vd["path"] != "/global/health" {
		t.Errorf("VendorDetails.path = %v, want /global/health", vd["path"])
	}
	if vd["body"] != "bad request body" {
		t.Errorf("VendorDetails.body = %v, want 'bad request body'", vd["body"])
	}
}

func runtimeErrorKind(err error) string {
	var r *runtime.RuntimeError
	if errors.As(err, &r) {
		return string(r.Kind)
	}
	return "none"
}
