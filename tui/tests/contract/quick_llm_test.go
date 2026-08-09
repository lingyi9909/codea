package contract

import (
	"context"
	"testing"
	"time"

	"codea/tui/internal/opencode"
	"codea/tui/internal/runtime"
)

func TestQuickLLMCheck(t *testing.T) {
	rt := opencode.NewOpenCodeAdapter("http://127.0.0.1:14242", "testuser", "testpass")
	ctx := context.Background()

	info, err := rt.Health(ctx)
	if err != nil { t.Fatalf("Health: %v", err) }
	t.Logf("Health: healthy=%v version=%s", info.Healthy, info.Version)

	session, err := rt.CreateSession(ctx, runtime.CreateSessionRequest{Title: "llm check"})
	if err != nil { t.Fatalf("CreateSession: %v", err) }
	t.Logf("Session: %s", session.ID)

	subCtx, subCancel := context.WithTimeout(ctx, 20*time.Second)
	defer subCancel()
	ch, err := rt.Subscribe(subCtx)
	if err != nil { t.Fatalf("Subscribe: %v", err) }

	err = rt.Prompt(ctx, runtime.SessionID(session.ID), runtime.PromptRequest{
		MessageID: "msg_1",
		Agent:     "build",
		Model:     &runtime.ModelRef{ProviderID: "deepseek", ModelID: "deepseek-v4-pro"},
		Parts:     []runtime.PromptPart{runtime.TextPart{Text: "say hello"}},
	})
	if err != nil { t.Fatalf("Prompt: %v", err) }

	seenAnswer := false
	seenReasoning := false
	seenError := false
	for evt := range ch {
		switch evt.Type {
		case "answer.delta": seenAnswer = true
		case "reasoning.delta": seenReasoning = true
		case "session.error":
			seenError = true
			if evt.Error != nil { t.Logf("Error: %s", evt.Error.Message) }
		}
		t.Logf("event: type=%s", evt.Type)
		if seenAnswer { subCancel() }
	}
	t.Logf("answer=%v reasoning=%v error=%v", seenAnswer, seenReasoning, seenError)

	if seenError { t.Skip("Model error, skipping") }
	if !seenAnswer && !seenReasoning { t.Error("no answer or reasoning") }

	rt.Cancel(ctx, runtime.SessionID(session.ID))
}
