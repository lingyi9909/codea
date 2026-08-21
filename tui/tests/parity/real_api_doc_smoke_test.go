package parity_test

import (
    "context"
    "encoding/json"
    "os"
    "path/filepath"
    "testing"
    "time"

    "codea/tui/internal/opencode"
    "codea/tui/internal/runtime"
)

type apiDocAgentEvidence struct {
    Timestamp string `json:"timestamp"`
    Endpoint string `json:"endpoint"`
    Available bool `json:"available"`
    Version string `json:"version,omitempty"`
    AgentListed bool `json:"agentListed"`
    ReadAllowed bool `json:"readAllowed"`
    WriteDocumentAllowed bool `json:"writeDocumentAllowed"`
    CrossToolDenied bool `json:"crossToolDenied"`
    DocumentWritten bool `json:"documentWritten"`
    TotalChecks int `json:"totalChecks"`
    PassedChecks int `json:"passedChecks"`
    FailedChecks int `json:"failedChecks"`
}

func (e *apiDocAgentEvidence) check(ok bool) {
    e.TotalChecks++
    if ok { e.PassedChecks++ } else { e.FailedChecks++ }
}

// TestRealAPIDocEvidence proves Task 16's agent is materialized into the real
// locked runtime, its fail-closed whitelist is server-side, and its only write
// path is write_document. Deterministic extraction semantics are covered by the
// API Documentation contract/tool tests; this smoke proves runtime integration.
func TestRealAPIDocEvidence(t *testing.T) {
    endpoint := os.Getenv("OPENCODE_ENDPOINT")
    if endpoint == "" { endpoint = os.Getenv("OPENCODE_SERVER_URL") }
    explicit := endpoint != ""
    if endpoint == "" { endpoint = "http://127.0.0.1:4141" }
    username := os.Getenv("OPENCODE_SERVER_USERNAME")
    password := os.Getenv("OPENCODE_SERVER_PASSWORD")
    smokeDir := os.Getenv("SMOKE_DIR")

    ev := &apiDocAgentEvidence{Timestamp: time.Now().UTC().Format(time.RFC3339), Endpoint: endpoint}
    adapter := opencode.NewOpenCodeAdapter(endpoint, username, password)
    ctx, cancel := context.WithTimeout(context.Background(), 240*time.Second)
    defer cancel()

    info, err := adapter.Health(ctx)
    if err != nil {
        if explicit { writeAPIDocEvidence(t, ev) }
        t.Skipf("real runtime not reachable at %s: %v", endpoint, err)
    }
    ev.Available = true
    ev.Version = info.Version

    agents, err := adapter.ListAgents(ctx)
    listed := false
    if err == nil {
        for _, a := range agents { if a.Name == "api-documentation" { listed = true; break } }
    }
    ev.AgentListed = listed
    ev.check(listed)

    ch, err := adapter.Subscribe(ctx)
    if err != nil { t.Fatalf("Subscribe: %v", err) }
    state := &smokeState{}

    readObs := runAgentScenario(t, adapter, ch, state, "api-documentation", "READ the file please", nil)
    ev.ReadAllowed = readObs.calledOnce("read") && readObs.succeededOnce("read")
    ev.check(ev.ReadAllowed)

    approveOnce := func(id string) error {
        return adapter.ReplyApproval(ctx, runtime.ApprovalID(id), runtime.ApprovalReply{Decision: runtime.ApprovalOnce})
    }
    docObs := runAgentScenario(t, adapter, ch, state, "api-documentation", "CALL WRITE_DOCUMENT please", approveOnce)
    ev.WriteDocumentAllowed = docObs.calledOnce("write_document") && docObs.succeededOnce("write_document")
    ev.check(ev.WriteDocumentAllowed)

    deniedObs := runAgentScenario(t, adapter, ch, state, "api-documentation", "CALL WRITE_TEST_FILE please", nil)
    ev.CrossToolDenied = deniedObs.calledOnce("write_test_file") && !deniedObs.succeededOnce("write_test_file")
    ev.check(ev.CrossToolDenied)

    docPath := filepath.Join(smokeDir, "docs", "smoke.md")
    data, statErr := os.ReadFile(docPath)
    ev.DocumentWritten = statErr == nil && string(data) == "smoke\n"
    ev.check(ev.DocumentWritten)

    writeAPIDocEvidence(t, ev)
    if ev.FailedChecks != 0 || ev.PassedChecks != ev.TotalChecks {
        t.Fatalf("API doc agent evidence failed: %+v", ev)
    }
    if ev.Version != "1.18.11" {
        t.Fatalf("runtime version=%s want 1.18.11", ev.Version)
    }
    t.Logf("API doc agent evidence: %d/%d PASS", ev.PassedChecks, ev.TotalChecks)
}

func writeAPIDocEvidence(t *testing.T, ev *apiDocAgentEvidence) {
    t.Helper()
    dir := "evidence"
    if err := os.MkdirAll(dir, 0o755); err != nil { t.Fatal(err) }
    body, err := json.MarshalIndent(ev, "", "  ")
    if err != nil { t.Fatal(err) }
    body = append(body, '\n')
    if err := os.WriteFile(filepath.Join(dir, "api-doc-agent-evidence.json"), body, 0o644); err != nil {
        t.Fatalf("write api doc evidence: %v", err)
    }
}
