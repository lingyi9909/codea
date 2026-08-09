package contract

import (
	"context"
	"net/http"
	"testing"
	"time"

	"codea/tui/internal/opencode"
	"codea/tui/internal/runtime"
)

func TestRealOpenCodeParitySmoke(t *testing.T) {
	baseURL := "http://127.0.0.1:14242"
	if _, err := http.Get(baseURL + "/global/health"); err != nil {
		t.Skipf("OpenCode not running at %s, skipping parity smoke", baseURL)
	}

	rt := opencode.NewOpenCodeAdapter(baseURL, "testuser", "testpass")
	ctx := context.Background()

	info, err := rt.Health(ctx)
	if err != nil {
		t.Fatalf("Health: %v", err)
	}
	if !info.Healthy || info.Version != "1.18.11" {
		t.Fatalf("Health: unexpected %+v", info)
	}

	session, err := rt.CreateSession(ctx, runtime.CreateSessionRequest{Title: "parity smoke"})
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	if session.ID == "" {
		t.Fatal("CreateSession: empty ID")
	}

	subCtx, subCancel := context.WithTimeout(ctx, 10*time.Second)
	defer subCancel()
	ch, err := rt.Subscribe(subCtx)
	if err != nil {
		t.Fatalf("Subscribe: %v", err)
	}

	err = rt.Prompt(ctx, runtime.SessionID(session.ID), runtime.PromptRequest{
		MessageID: "msg_1",
		Agent:     "build",
		Parts:     []runtime.PromptPart{runtime.TextPart{Text: "say hello"}},
	})
	if err != nil {
		t.Fatalf("Prompt: %v", err)
	}

	count := 0
	for evt := range ch {
		if evt.Type == "" {
			t.Fatal("event has empty Type")
		}
		if len(evt.Raw) == 0 {
			t.Fatal("event has empty Raw")
		}
		count++
		if count >= 5 {
			subCancel()
		}
	}
	if count < 3 {
		t.Fatalf("expected >= 3 events from real OpenCode, got %d", count)
	}

	err = rt.Cancel(ctx, runtime.SessionID(session.ID))
	if err != nil {
		t.Fatalf("Cancel: %v", err)
	}

	agents, err := rt.ListAgents(ctx)
	if err != nil {
		t.Fatalf("ListAgents: %v", err)
	}
	if len(agents) == 0 {
		t.Fatal("ListAgents: no agents")
	}

	caps := rt.Capabilities()
	if !caps.Sessions || !caps.Streaming {
		t.Fatal("Capabilities: missing required")
	}
}
