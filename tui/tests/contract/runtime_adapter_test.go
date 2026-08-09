package contract

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"codea/tui/internal/opencode"
	"codea/tui/internal/runtime"
)

func TestAgentRuntimeContract(t *testing.T) {
	var mu sync.Mutex
	sessions := make(map[string]bool)
	permissions := make(map[string]string)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		mu.Lock()
		defer mu.Unlock()

		// Handle /permission/*/reply generically
		if strings.HasPrefix(r.URL.Path, "/permission/") && strings.HasSuffix(r.URL.Path, "/reply") {
			var req opencode.OpenCodePermissionReplyRequest
			json.NewDecoder(r.Body).Decode(&req)
			permissions[r.URL.Path] = req.Reply
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(true)
			return
		}

		switch r.URL.Path {
		case "/global/health":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{"healthy": true, "version": "1.18.11"})

		case "/session":
			if r.Method == http.MethodPost {
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]string{"id": "ses_contract"})
			}

		case "/session/ses_contract/prompt_async":
			w.WriteHeader(http.StatusNoContent)

		case "/global/event":
			if r.Header.Get("Accept") != "text/event-stream" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "text/event-stream")
			flusher, _ := w.(http.Flusher)
			events := []string{
				`{"directory":"/tmp","payload":{"type":"session.status","properties":{"sessionID":"ses_contract","status":{"type":"busy"}}}}`,
				`{"directory":"/tmp","payload":{"type":"session.status","properties":{"sessionID":"ses_contract","status":{"type":"idle"}}}}`,
			}
			for _, evt := range events {
				w.Write([]byte("data: " + evt + "\n\n"))
				flusher.Flush()
			}
			<-r.Context().Done()

		case "/session/ses_contract/abort":
			delete(sessions, "ses_contract")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(true)

		case "/agent":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]map[string]string{
				{"name": "build", "mode": "build"},
			})

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	var rt runtime.AgentRuntime = opencode.NewOpenCodeAdapter(srv.URL, "", "")
	ctx := context.Background()

	// Health
	info, err := rt.Health(ctx)
	if err != nil {
		t.Fatalf("Health: %v", err)
	}
	if !info.Healthy || info.Version != "1.18.11" {
		t.Fatalf("Health: unexpected result %+v", info)
	}

	// CreateSession
	session, err := rt.CreateSession(ctx, runtime.CreateSessionRequest{Title: "contract test"})
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if session.ID != "ses_contract" {
		t.Fatalf("CreateSession: expected ses_contract, got %q", session.ID)
	}

	// Prompt
	err = rt.Prompt(ctx, runtime.SessionID(session.ID), runtime.PromptRequest{
		MessageID: "msg_1",
		Agent:     "build",
		Parts:     []runtime.PromptPart{runtime.TextPart{Text: "hello"}},
	})
	if err != nil {
		t.Fatalf("Prompt: %v", err)
	}

	// Subscribe (global SSE)
	subCtx, subCancel := context.WithTimeout(ctx, 2*time.Second)
	defer subCancel()
	ch, err := rt.Subscribe(subCtx)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}
	eventCount := 0
	for evt := range ch {
		if evt.Type == "" {
			t.Fatal("event has empty Type")
		}
		if len(evt.Raw) == 0 {
			t.Fatal("event has empty Raw")
		}
		eventCount++
		if eventCount >= 2 {
			subCancel()
		}
	}
	if eventCount != 2 {
		t.Fatalf("expected 2 events, got %d", eventCount)
	}

	// ReplyApproval (once)
	err = rt.ReplyApproval(ctx, "perm_1", runtime.ApprovalReply{
		Decision: runtime.ApprovalOnce,
		Message:  "ok",
	})
	if err != nil {
		t.Fatalf("ReplyApproval(once): %v", err)
	}

	// ReplyApproval (reject)
	err = rt.ReplyApproval(ctx, "perm_2", runtime.ApprovalReply{
		Decision: runtime.ApprovalReject,
		Message:  "denied",
	})
	if err != nil {
		t.Fatalf("ReplyApproval(reject): %v", err)
	}

	// Cancel
	err = rt.Cancel(ctx, runtime.SessionID(session.ID))
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	// ListAgents
	agents, err := rt.ListAgents(ctx)
	if err != nil {
		t.Fatalf("ListAgents: %v", err)
	}
	if len(agents) != 1 || agents[0].Name != "build" {
		t.Fatalf("ListAgents: unexpected result %+v", agents)
	}

	// Capabilities
	caps := rt.Capabilities()
	if !caps.Sessions || !caps.Streaming {
		t.Fatalf("Capabilities: missing required capabilities")
	}
}
