package opencode

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"codea/tui/internal/runtime"
)

// OpenCode v1.18.11 emits a reasoning part as message.part.updated with
// part.type=reasoning, then streams its text via message.part.delta with
// field=text. The adapter must retain the part kind so the delta is exposed
// as Codea reasoning.delta rather than answer.delta.
func TestOpenCodeAdapterClassifiesReasoningDeltaFromPartState(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/session" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]OpenCodeSessionV2Info{})
			return
		}
		if r.Method != http.MethodGet || r.URL.Path != "/global/event" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		flusher := w.(http.Flusher)
		updated := `{"directory":"/tmp","payload":{"type":"message.part.updated","properties":{"part":{"id":"p_reason","messageID":"m1","sessionID":"s1","type":"reasoning","text":""}}}}`
		delta := `{"directory":"/tmp","payload":{"type":"message.part.delta","properties":{"sessionID":"s1","messageID":"m1","partID":"p_reason","field":"text","delta":"parity-thinking"}}}`
		_, _ = w.Write([]byte("data: " + updated + "\n\n"))
		flusher.Flush()
		_, _ = w.Write([]byte("data: " + delta + "\n\n"))
		flusher.Flush()
		<-r.Context().Done()
	}))
	defer srv.Close()

	adapter := NewOpenCodeAdapter(srv.URL, "", "")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	ch, err := adapter.Subscribe(ctx)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	deadline := time.After(time.Second)
	for {
		select {
		case ev, ok := <-ch:
			if !ok {
				t.Fatal("event channel closed before reasoning delta")
			}
			if ev.Content != "parity-thinking" {
				continue
			}
			if ev.Type != runtime.EventType("reasoning.delta") {
				t.Fatalf("real reasoning text delta must map to reasoning.delta, got %q", ev.Type)
			}
			return
		case <-deadline:
			t.Fatal("timeout waiting for reasoning delta")
		}
	}
}
