package opencode

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"codea/tui/internal/runtime"
)

func TestOpenCodeAdapterLastPromptRecordsMappedAgentAfterRuntimeAcceptsPrompt(t *testing.T) {
	var gotAgent string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/session/ses_test/prompt_async" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		var body OpenCodeSessionPromptAsyncRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("decode prompt: %v", err)
		}
		gotAgent = body.Agent
		w.WriteHeader(http.StatusNoContent)
	}))
	defer srv.Close()

	adapter := NewOpenCodeAdapter(srv.URL, "", "")
	if _, ok := adapter.LastPrompt(); ok {
		t.Fatal("LastPrompt must be empty before a successful prompt")
	}

	err := adapter.Prompt(context.Background(), "ses_test", runtime.PromptRequest{
		Agent: "reviewer",
		Parts: []runtime.PromptPart{runtime.TextPart{Text: "review"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotAgent != "reviewer" {
		t.Fatalf("runtime received agent=%q want reviewer", gotAgent)
	}
	agent, ok := adapter.LastPrompt()
	if !ok || agent != "reviewer" {
		t.Fatalf("LastPrompt=(%q,%v), want (reviewer,true)", agent, ok)
	}
}

func TestOpenCodeAdapterLastPromptDoesNotRecordRejectedPrompt(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"error":"bad prompt"}`))
	}))
	defer srv.Close()

	adapter := NewOpenCodeAdapter(srv.URL, "", "")
	err := adapter.Prompt(context.Background(), "ses_test", runtime.PromptRequest{
		Agent: "reviewer",
		Parts: []runtime.PromptPart{runtime.TextPart{Text: "review"}},
	})
	if err == nil {
		t.Fatal("expected rejected prompt error")
	}
	if agent, ok := adapter.LastPrompt(); ok {
		t.Fatalf("rejected prompt must not become parity evidence: %q", agent)
	}
}
