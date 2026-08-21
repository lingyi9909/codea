package parity_test

import (
    "context"
    "encoding/json"
    "os"
    "path/filepath"
    "strings"
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
    WorkflowExtractSucceeded bool `json:"workflowExtractSucceeded"`
    WorkflowValidateSucceeded bool `json:"workflowValidateSucceeded"`
    WorkflowWriteSucceeded bool `json:"workflowWriteSucceeded"`
    WorkflowDocumentValid bool `json:"workflowDocumentValid"`
    TotalChecks int `json:"totalChecks"`
    PassedChecks int `json:"passedChecks"`
    FailedChecks int `json:"failedChecks"`
}

func (e *apiDocAgentEvidence) check(ok bool) {
    e.TotalChecks++
    if ok { e.PassedChecks++ } else { e.FailedChecks++ }
}

// TestRealAPIDocEvidence proves Task 16's agent is materialized into the real
// locked runtime, its fail-closed whitelist is server-side, and the complete
// API Documentation workflow executes through real OpenCode v1.18.11:
// extract_api_spec -> validate_api_example -> write_document -> persisted Markdown.
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
    ctx, cancel := context.WithTimeout(context.Background(), 300*time.Second)
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

    flowObs := runAgentScenario(t, adapter, ch, state, "api-documentation", "APIDOCFLOW generate the API documentation", approveOnce)
    ev.WorkflowExtractSucceeded = flowObs.calledOnce("extract_api_spec") && flowObs.succeededOnce("extract_api_spec")
    ev.check(ev.WorkflowExtractSucceeded)
    ev.WorkflowValidateSucceeded = flowObs.calledOnce("validate_api_example") && flowObs.succeededOnce("validate_api_example")
    ev.check(ev.WorkflowValidateSucceeded)
    ev.WorkflowWriteSucceeded = flowObs.calledOnce("write_document") && flowObs.succeededOnce("write_document")
    ev.check(ev.WorkflowWriteSucceeded)

    flowPath := filepath.Join(smokeDir, "docs", "api-demo.md")
    flowData, flowErr := os.ReadFile(flowPath)
    ev.WorkflowDocumentValid = flowErr == nil && apiDocMarkdownValid(string(flowData))
    ev.check(ev.WorkflowDocumentValid)

    writeAPIDocEvidence(t, ev)
    if ev.FailedChecks != 0 || ev.PassedChecks != ev.TotalChecks {
        t.Fatalf("API doc agent evidence failed: %+v", ev)
    }
    if ev.Version != "1.18.11" {
        t.Fatalf("runtime version=%s want 1.18.11", ev.Version)
    }
    t.Logf("API doc agent evidence: %d/%d PASS", ev.PassedChecks, ev.TotalChecks)
}

func apiDocMarkdownValid(text string) bool {
    required := []string{
        "# API Documentation",
        "DemoController",
        "GET /api/users/{id}",
        "POST /api/users",
        "CreateUserRequest",
        "@NotBlank",
        "@Email",
        "@Min(1)",
        "@Max(120)",
        "DECLARED",
        "Not determined from code",
        "Example validation: PASS",
    }
    for _, needle := range required {
        if !strings.Contains(text, needle) { return false }
    }
    if strings.Contains(text, "REFERRED") { return false }
    return true
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
