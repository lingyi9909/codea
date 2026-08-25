package opencode

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestListAgentsUsesDedicatedColdStartTimeout(t *testing.T) {
	const responseDelay = 120 * time.Millisecond
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(responseDelay)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/agent":
			_, _ = w.Write([]byte(`[{"name":"build","mode":"primary","options":{},"permission":[]}]`))
		case "/global/health":
			_, _ = w.Write([]byte(`{"healthy":true,"version":"1.18.11"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	client := newHTTPClientWithTimeouts(
		server.URL,
		"",
		"",
		40*time.Millisecond,
		500*time.Millisecond,
	)

	agents, err := client.ListAgents(context.Background())
	if err != nil {
		t.Fatalf("ListAgents should tolerate a slow cold-start response: %v", err)
	}
	if len(agents) != 1 || agents[0].Name != "build" {
		t.Fatalf("ListAgents = %#v", agents)
	}

	if _, err := client.Health(context.Background()); err == nil {
		t.Fatal("Health should retain the shorter default HTTP timeout")
	}
}
