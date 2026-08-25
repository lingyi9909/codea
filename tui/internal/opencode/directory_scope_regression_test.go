package opencode

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHTTPClientListAgentsSendsExplicitDirectoryScope(t *testing.T) {
	const projectDir = `C:\work\codea-project`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/agent" {
			t.Fatalf("path = %q, want /agent", r.URL.Path)
		}
		if got := r.Header.Get("x-opencode-directory"); got != projectDir {
			t.Fatalf("x-opencode-directory = %q, want %q", got, projectDir)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`[]`))
	}))
	defer server.Close()

	client := NewHTTPClientForDirectory(server.URL, "", "", projectDir)
	if _, err := client.ListAgents(context.Background()); err != nil {
		t.Fatalf("ListAgents: %v", err)
	}
}
