package opencode

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHTTPClientHealthUsesGeneratedDTOAndBasicAuth(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/global/health" {
			t.Errorf("request = %s %s, want GET /global/health", r.Method, r.URL.Path)
		}
		username, password, ok := r.BasicAuth()
		if !ok || username != "codea" || password != "secret" {
			t.Errorf("basic auth = %q/%q/%t", username, password, ok)
		}
		if got := r.Header.Get("Accept"); got != "application/json" {
			t.Errorf("Accept = %q, want application/json", got)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"healthy":true,"version":"1.18.11"}`))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL+"/", "codea", "secret")
	health, err := client.Health(context.Background())
	if err != nil {
		t.Fatalf("Health returned error: %v", err)
	}
	if !health.Healthy || health.Version != "1.18.11" {
		t.Fatalf("Health = %#v", health)
	}
}

func TestHTTPClientCreateSessionUsesGeneratedRequestAndResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/session" {
			t.Errorf("request = %s %s, want POST /session", r.Method, r.URL.Path)
		}
		if got := r.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("Content-Type = %q, want application/json", got)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if body["title"] != "Task 2" || len(body) != 1 {
			t.Errorf("request body = %#v, want title only", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"ses_1","title":"Task 2"}`))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "", "")
	session, err := client.CreateSession(context.Background(), &OpenCodeSessionCreateRequest{Title: "Task 2"})
	if err != nil {
		t.Fatalf("CreateSession returned error: %v", err)
	}
	if session.ID != "ses_1" || session.Title != "Task 2" {
		t.Fatalf("CreateSession = %#v", session)
	}
}

func TestHTTPClientHealthReturnsStatusAndBoundedErrorBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_, _ = w.Write([]byte("runtime unavailable"))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "", "")
	_, err := client.Health(context.Background())
	if err == nil {
		t.Fatal("Health returned nil error for HTTP 503")
	}
	for _, want := range []string{"503", "runtime unavailable"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("Health error = %q, want %q", err, want)
		}
	}
}

func TestHTTPClientSendPromptUsesAsyncPathAndGeneratedPayload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/session/ses_1/prompt_async" {
			t.Errorf("request = %s %s, want POST /session/ses_1/prompt_async", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		model, _ := body["model"].(map[string]any)
		if model["providerID"] != "private" || model["modelID"] != "coder" {
			t.Errorf("model = %#v", model)
		}
		parts, _ := body["parts"].([]any)
		if len(parts) != 1 || parts[0].(map[string]any)["text"] != "hello" {
			t.Errorf("parts = %#v", parts)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "", "")
	request := &OpenCodeSessionPromptAsyncRequest{
		Model: &OpenCodeSessionPromptAsyncRequestModel{
			ProviderID: "private",
			ModelID:    "coder",
		},
		Parts: []any{OpenCodeTextPartInput{Type: "text", Text: "hello"}},
	}
	if err := client.SendPrompt(context.Background(), "ses_1", request); err != nil {
		t.Fatalf("SendPrompt returned error: %v", err)
	}
}

func TestHTTPClientApprovePermissionUsesNonDeprecatedReplyEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/permission/per_1/reply" {
			t.Errorf("request = %s %s, want POST /permission/per_1/reply", r.Method, r.URL.Path)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if body["reply"] != "once" || len(body) != 1 {
			t.Errorf("request body = %#v, want reply only", body)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("true"))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "", "")
	request := &OpenCodePermissionReplyRequest{Reply: "once"}
	if err := client.ApprovePermission(context.Background(), "per_1", request); err != nil {
		t.Fatalf("ApprovePermission returned error: %v", err)
	}
}

func TestHTTPClientAbortSessionChecksBooleanResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/session/ses_1/abort" {
			t.Errorf("request = %s %s, want POST /session/ses_1/abort", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("true"))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "", "")
	if err := client.AbortSession(context.Background(), "ses_1"); err != nil {
		t.Fatalf("AbortSession returned error: %v", err)
	}
}

func TestHTTPClientListAgentsDecodesGeneratedAgentSlice(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/agent" {
			t.Errorf("request = %s %s, want GET /agent", r.Method, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[{"name":"build","mode":"primary","options":{},"permission":[]}]`))
	}))
	defer server.Close()

	client := NewHTTPClient(server.URL, "", "")
	agents, err := client.ListAgents(context.Background())
	if err != nil {
		t.Fatalf("ListAgents returned error: %v", err)
	}
	if len(agents) != 1 || agents[0].Name != "build" || agents[0].Mode != "primary" {
		t.Fatalf("ListAgents = %#v", agents)
	}
}
