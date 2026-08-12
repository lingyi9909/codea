package contract

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"codea/tui/internal/opencode"
	"codea/tui/internal/runtime"
)

func TestAgentRuntimeContract(t *testing.T) {
	permissions := make(map[string]string)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if strings.HasPrefix(r.URL.Path, "/permission/") && strings.HasSuffix(r.URL.Path, "/reply") {
			var req opencode.OpenCodePermissionReplyRequest
			json.NewDecoder(r.Body).Decode(&req)
			permissions[r.URL.Path] = req.Reply
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(true)
			return
		}

		switch r.URL.Path {
		case "/session/status":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(opencode.OpenCodeSessionsResponse{Data: []opencode.OpenCodeSessionV2Info{}})
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
		// Skip recovery/infra events injected by ReconnectHook (connected markers, recovery errors).
		if evt.Type == opencode.CodeaEventRuntimeConnected || evt.Type == opencode.CodeaEventRuntimeError {
			continue
		}
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

func TestAgentRuntimeRecoveryContract(t *testing.T) {
	var sseConnects atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// Recovery: return a session NOT present in the live SSE stream.
		if r.URL.Path == "/session/status" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(opencode.OpenCodeSessionsResponse{
				Data: []opencode.OpenCodeSessionV2Info{
					{
						ID:    "recovered_session",
						Title: "Recovered Session",
						Time:  opencode.OpenCodeSessionV2InfoTime{Created: 1000},
					},
				},
			})
			return
		}

		// Recovery: return messages for the recovered session.
		if strings.Contains(r.URL.Path, "/message") {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(opencode.OpenCodeSessionMessagesResponse{
				Data: []opencode.OpenCodeSessionMessage{
					map[string]any{
						"id":   "recovered_msg",
						"type": "assistant",
						"content": []map[string]any{
							{"id": "recovered_part", "type": "text"},
						},
					},
				},
			})
			return
		}

		// Permission reply.
		if strings.HasSuffix(r.URL.Path, "/reply") {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(true)
			return
		}

		// SSE event stream.
		if r.URL.Path == "/global/event" {
			connNum := sseConnects.Add(1)
			flusher, _ := w.(http.Flusher)
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)

			if connNum == 1 {
				// First connection: send live events including an approval
				// request, then close to trigger disconnect + reconnect + recovery.
				events := []string{
					`{"directory":"/tmp","payload":{"type":"answer.delta","properties":{"sessionID":"live_session","content":"hello"}}}`,
					`{"directory":"/tmp","payload":{"type":"permission.asked","properties":{"id":"perm_recovery","permission":"write","sessionID":"live_session"}}}`,
				}
				for _, evt := range events {
					w.Write([]byte("data: " + evt + "\n\n"))
					flusher.Flush()
				}
				// Close to simulate disconnect.
				return
			}

			// Subsequent connections: keep alive so we can drain recovery events.
			<-r.Context().Done()
			return
		}

		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	adapter := opencode.NewOpenCodeAdapter(srv.URL, "", "")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ch, err := adapter.Subscribe(ctx)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	var seenDisconnect, seenRecovery, seenApprovalReq bool
	for ev := range ch {
		switch {
		case ev.Type == opencode.CodeaEventRuntimeError && ev.Error != nil &&
			ev.Error.Code == "DISCONNECTED":
			seenDisconnect = true
		case ev.Type == opencode.CodeaEventSessionCreated &&
			ev.Metadata["recovered"] == "true":
			seenRecovery = true
		case ev.Type == opencode.CodeaEventApprovalRequested:
			seenApprovalReq = true
		}

		if seenDisconnect && seenRecovery && seenApprovalReq {
			cancel()
		}
	}

	if !seenDisconnect {
		t.Error("did not observe disconnect event")
	}
	if !seenRecovery {
		t.Error("did not observe recovery event (session.created with recovered=true)")
	}
	if !seenApprovalReq {
		t.Error("did not observe approval.requested event")
	}
	t.Logf("disconnect=%v recovery=%v approval=%v", seenDisconnect, seenRecovery, seenApprovalReq)
}

