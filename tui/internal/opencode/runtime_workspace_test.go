package opencode

import (
    "context"
    "encoding/json"
    "net/http"
    "net/http/httptest"
    "testing"

    "codea/tui/internal/runtime"
)

func TestTask23ListModelsMapsConnectedRuntimeModels(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.URL.Path != "/provider" {
            http.NotFound(w, r)
            return
        }
        w.Header().Set("Content-Type", "application/json")
        _, _ = w.Write([]byte(`{
          "all": [
            {"id":"company","name":"Company AI","models":{"kimi":{"id":"kimi","providerID":"company","name":"Kimi"}}},
            {"id":"offline","name":"Offline","models":{"other":{"id":"other","providerID":"offline","name":"Other"}}}
          ],
          "default":{"company":"kimi"},
          "connected":["company"]
        }`))
    }))
    defer server.Close()

    adapter := NewOpenCodeAdapter(server.URL, "", "")
    models, err := adapter.ListModels(context.Background())
    if err != nil {
        t.Fatalf("ListModels: %v", err)
    }
    if len(models) != 1 {
        t.Fatalf("models = %#v, want one connected runtime model", models)
    }
    got := models[0]
    if got.Ref != (runtime.ModelRef{ProviderID: "company", ModelID: "kimi"}) {
        t.Fatalf("ref = %#v", got.Ref)
    }
    if got.Name != "Kimi" || got.ProviderName != "Company AI" || !got.Default {
        t.Fatalf("model metadata = %#v", got)
    }
}

func TestTask23CompactSessionUsesActualSessionModel(t *testing.T) {
    var summarize map[string]any
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        switch {
        case r.Method == http.MethodGet && r.URL.Path == "/session/s1/message":
            w.Header().Set("Content-Type", "application/json")
            _, _ = w.Write([]byte(`[{"info":{"role":"assistant","providerID":"company","modelID":"kimi"},"parts":[]}]`))
        case r.Method == http.MethodPost && r.URL.Path == "/session/s1/summarize":
            if err := json.NewDecoder(r.Body).Decode(&summarize); err != nil {
                t.Fatalf("decode summarize payload: %v", err)
            }
            w.Header().Set("Content-Type", "application/json")
            _, _ = w.Write([]byte(`true`))
        default:
            http.NotFound(w, r)
        }
    }))
    defer server.Close()

    adapter := NewOpenCodeAdapter(server.URL, "", "")
    if err := adapter.CompactSession(context.Background(), runtime.SessionID("s1")); err != nil {
        t.Fatalf("CompactSession: %v", err)
    }
    if summarize["providerID"] != "company" || summarize["modelID"] != "kimi" {
        t.Fatalf("summarize payload = %#v", summarize)
    }
}

func TestTask23CompactSessionFailsClosedWithoutModelEvidence(t *testing.T) {
    server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if r.Method == http.MethodGet && r.URL.Path == "/session/s1/message" {
            w.Header().Set("Content-Type", "application/json")
            _, _ = w.Write([]byte(`[]`))
            return
        }
        http.NotFound(w, r)
    }))
    defer server.Close()

    adapter := NewOpenCodeAdapter(server.URL, "", "")
    err := adapter.CompactSession(context.Background(), runtime.SessionID("s1"))
    if err == nil || !runtime.IsIncompatible(err) {
        t.Fatalf("err = %v, want incompatible fail-closed error", err)
    }
}
