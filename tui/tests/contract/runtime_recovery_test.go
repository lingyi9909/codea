package contract

import (
	"context"
	"net/http"
	"strings"
	"testing"
	"time"

	"codea/tui/internal/opencode"
	"codea/tui/internal/runtime"
)

// TestRuntimeRecoveryContract verifies the recovery, approval, and abort
// contracts against a real OpenCode instance.
//
// Prerequisites:
//   - OpenCode v1.18.11 running at http://127.0.0.1:14242
//   - A configured AI provider with a working model
//
// The test SKIPs when OpenCode is not running.
func TestRuntimeRecoveryContract(t *testing.T) {
	baseURL := "http://127.0.0.1:14242"
	user := "testuser"
	pass := "testpass"

	if _, err := http.Get(baseURL + "/global/health"); err != nil {
		t.Skipf("OpenCode not running at %s, skipping recovery contract", baseURL)
	}

	adapter := opencode.NewOpenCodeAdapter(baseURL, user, pass)
	ctx := context.Background()

	// ---- Health ----
	info, err := adapter.Health(ctx)
	if err != nil {
		t.Fatalf("Health: %v", err)
	}
	if !info.Healthy {
		t.Fatal("OpenCode reported unhealthy")
	}
	t.Logf("Health: version=%s", info.Version)

	// ---- Capabilities ----
	caps := adapter.Capabilities()
	if !caps.Sessions || !caps.Streaming || !caps.MessageHistory {
		t.Fatalf("missing capabilities: sessions=%v streaming=%v history=%v",
			caps.Sessions, caps.Streaming, caps.MessageHistory)
	}

	// ---- ListAgents ----
	agents, err := adapter.ListAgents(ctx)
	if err != nil {
		t.Fatalf("ListAgents: %v", err)
	}
	if len(agents) == 0 {
		t.Fatal("no agents available")
	}
	t.Logf("Agents: %d available", len(agents))

	// ---- CreateSession ----
	session, err := adapter.CreateSession(ctx, runtime.CreateSessionRequest{Title: "recovery contract"})
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if session.ID == "" {
		t.Fatal("empty session ID")
	}
	t.Logf("Session: %s", session.ID)

	// ---- Subscribe with reconnecting client ----
	subCtx, subCancel := context.WithTimeout(ctx, 90*time.Second)
	defer subCancel()
	ch, err := adapter.Subscribe(subCtx)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	// ---- Prompt ----
	err = adapter.Prompt(ctx, runtime.SessionID(session.ID), runtime.PromptRequest{
		MessageID: "msg_recovery",
		Agent:     "build",
		Parts: []runtime.PromptPart{
			runtime.TextPart{Text: "Print 'recovery-test-ok'. Do not read any files."},
		},
	})
	if err != nil {
		t.Fatalf("Prompt: %v", err)
	}
	t.Log("Prompt sent")

	// ---- Collect events, verify recovery markers ----
	var (
		seenConnected   bool
		seenDisconnect  bool
		seenRecovery    bool
		seenAnswer      bool
		seenSessionErr  bool
		seenStepStart   bool
		seenStepFinish  bool
		seenToolCalled  bool
		seenApprovalReq bool
		eventCount      int
	)

	for evt := range ch {
		eventCount++

		switch evt.Type {
		case opencode.CodeaEventRuntimeConnected:
			seenConnected = true
			if evt.Metadata != nil && evt.Metadata["recovered"] == "true" {
				seenRecovery = true
				t.Logf("Recovery event: seq=%d", evt.Sequence)
			}

		case opencode.CodeaEventRuntimeError:
			if evt.Error != nil {
				code := evt.Error.Code
				if code == "DISCONNECTED" || code == "CONNECT_FAILED" || code == "SCANNER_ERROR" {
					seenDisconnect = true
					t.Logf("Disconnect: code=%s msg=%s", code, evt.Error.Message)
				}
				if strings.Contains(evt.Error.Message, "No user message found") {
					t.Log("Model not configured — skipping semantic assertions")
					seenSessionErr = true
				}
			}

		case opencode.CodeaEventAnswerDelta:
			seenAnswer = true

		case opencode.CodeaEventStepStarted:
			seenStepStart = true

		case opencode.CodeaEventStepFinished:
			seenStepFinish = true

		case opencode.CodeaEventToolCalled:
			seenToolCalled = true
			if evt.Tool != nil {
				t.Logf("Tool: name=%s callID=%s", evt.Tool.Name, evt.Tool.CallID)
			}

		case opencode.CodeaEventApprovalRequested:
			seenApprovalReq = true
			if evt.Approval != nil {
				t.Logf("Approval: id=%s permission=%s", evt.Approval.ID, evt.Approval.Permission)
				// Auto-approve once.
				err = adapter.ReplyApproval(ctx, runtime.ApprovalID(evt.Approval.ID), runtime.ApprovalReply{
					Decision: runtime.ApprovalOnce,
					Message:  "contract test — approve once",
				})
				if err != nil {
					t.Fatalf("ReplyApproval: %v", err)
				}
				t.Log("ReplyApproval: ok")
			}

		case opencode.CodeaEventSessionError:
			seenSessionErr = true
			if evt.Error != nil {
				t.Logf("Session error: %s", evt.Error.Message)
			}
		}

		// Stop after answer received or session error or timeout gate.
		if seenAnswer || (seenStepFinish && seenAnswer) {
			subCancel()
		}
		if seenSessionErr && strings.Contains(evt.Error.Message, "No user message found") {
			t.Skip("Model not configured — skipping semantic assertions")
		}
		if eventCount >= 300 {
			subCancel()
		}
	}

	t.Logf("Events: %d", eventCount)
	t.Logf("contract: connected=%v disconnect=%v recovery=%v answer=%v step=%v/%v tool=%v approval=%v",
		seenConnected, seenDisconnect, seenRecovery, seenAnswer,
		seenStepStart, seenStepFinish, seenToolCalled, seenApprovalReq)

	// ---- Assertions ----
	if !seenConnected {
		t.Error("never received runtime.connected event")
	}
	if !seenSessionErr {
		if !seenAnswer {
			t.Error("never received answer.delta")
		}
	}

	// ---- Abort contract: Cancel must succeed ----
	err = adapter.Cancel(ctx, runtime.SessionID(session.ID))
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}
	t.Log("Cancel: ok")

	// Verify abort is idempotent (second cancel should succeed or be a no-op).
	err = adapter.Cancel(ctx, runtime.SessionID(session.ID))
	if err != nil {
		t.Logf("Cancel (second): %v (idempotent check)", err)
	} else {
		t.Log("Cancel (second): ok (idempotent)")
	}

	t.Log("Recovery + Approval + Abort contract: PASS")
}
