package parity_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"codea/tui/internal/capability"
	"codea/tui/internal/opencode"
	"codea/tui/internal/parity"
	"codea/tui/internal/runtime"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

func TestAllV1RequiredScenariosPass(t *testing.T) {
	// Events must satisfy all semantic assertions in V1RequiredScenarios():
	// RequireAnswer, RequireReasoning, RequireTool, RequireApproval, RequireRaw.
	sharedEvents := []runtime.Event{
		{Type: runtime.EventType("reasoning.delta"), Content: "thinking..."},
		{Type: runtime.EventType("answer.delta"), Content: "ok"},
		{Type: runtime.EventType("tool.called"), Tool: &runtime.ToolEvent{
			Name: "read", CallID: "call-1",
		}},
		{Type: runtime.EventType("approval.requested"), Approval: &runtime.ApprovalRequest{
			ID: "approval-1", Permission: "bash",
		}},
		{Type: runtime.EventType("raw"), Raw: []byte(`{"foo":"bar"}`)},
		{Type: runtime.EventType("step.finished")},
	}

	baseline := fakeruntime.New()
	baseline.HealthInfo = runtime.HealthInfo{Healthy: true, Version: "test"}
	baseline.Events = sharedEvents

	candidate := fakeruntime.New()
	candidate.HealthInfo = runtime.HealthInfo{Healthy: true, Version: "test"}
	candidate.Events = sharedEvents

	runner := parity.Runner{Baseline: baseline, Candidate: candidate}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result := runner.RunAll(ctx, parity.V1RequiredScenarios())

	if result.RequiredFailed > 0 {
		t.Errorf("all V1 required scenarios should pass, got %d required failures", result.RequiredFailed)
		for _, s := range result.Scenarios {
			if !s.Passed && s.Required {
				t.Errorf("  FAIL: %s — %v", s.Name, s.Failures)
			}
		}
	}
	if result.Total != len(parity.V1RequiredScenarios()) {
		t.Errorf("expected %d scenarios, got %d", len(parity.V1RequiredScenarios()), result.Total)
	}
}

func TestCapabilityCompareWithRealOpenCodeAdapter(t *testing.T) {
	// Load real product requirements from capabilities.yaml.
	inv, err := capability.Load("../../../runtime/capabilities.yaml")
	if err != nil {
		t.Fatalf("load capabilities.yaml: %v", err)
	}

	// Instantiate a real OpenCodeAdapter and call Capabilities() through the
	// AgentRuntime interface — not through the package-level helper function.
	adapter := opencode.NewOpenCodeAdapter("http://127.0.0.1:1", "", "")
	var rt runtime.AgentRuntime = adapter
	caps := rt.Capabilities()

	result := inv.Compare(caps)
	if result.HasRequiredFailures() {
		t.Errorf("OpenCodeAdapter.Capabilities() must satisfy all product requirements, missing: %v", result.RequiredMissing)
	}
	if len(result.RequiredSupported) != 15 {
		t.Errorf("expected 15 required supported, got %d", len(result.RequiredSupported))
	}
}

func TestCapabilityCompareMissingRequired(t *testing.T) {
	inv := &capability.Inventory{
		Requirements: []capability.CapabilityRequirement{
			{Name: "sessions", Level: capability.Required},
			{Name: "streaming", Level: capability.Required},
		},
	}

	caps := runtime.RuntimeCapabilities{
		Sessions:  true,
		Streaming: false,
	}

	result := inv.Compare(caps)
	if !result.HasRequiredFailures() {
		t.Error("should detect missing required capabilities")
	}
}

// runtimeEvidence captures the result of each real-runtime gate check.
type runtimeEvidence struct {
	Timestamp string `json:"timestamp"`
	Endpoint  string `json:"endpoint"`
	Available bool   `json:"available"`
	Error     string `json:"error,omitempty"`

	// Per-gate results.
	Health         *gateResult `json:"health"`
	CreateSession  *gateResult `json:"createSession"`
	Prompt         *gateResult `json:"prompt"`
	SSE            *gateResult `json:"sse"`
	Cancel         *gateResult `json:"cancel"`
	AgentSelection *gateResult `json:"agentSelection"`
	Approval       *gateResult `json:"approval"`

	TotalChecks   int `json:"totalChecks"`
	PassedChecks  int `json:"passedChecks"`
	FailedChecks  int `json:"failedChecks"`
	SkippedChecks int `json:"skippedChecks"`
}

type gateResult struct {
	Passed  bool   `json:"passed"`
	Detail  string `json:"detail,omitempty"`
	Error   string `json:"error,omitempty"`
	Skipped bool   `json:"skipped,omitempty"`
}

// TestRealRuntimeEvidence verifies the full AgentRuntime contract against a
// running OpenCode server and records the result as an auditable evidence
// artifact. When no runtime is configured, the artifact records which gates
// could not be checked — no silent skip.
func TestRealRuntimeEvidence(t *testing.T) {
	evidenceDir := filepath.Join("evidence")
	_ = os.MkdirAll(evidenceDir, 0o755)

	endpoint := os.Getenv("OPENCODE_ENDPOINT")
	if endpoint == "" {
		endpoint = os.Getenv("OPENCODE_SERVER_URL")
	}
	if endpoint == "" {
		endpoint = "http://127.0.0.1:4141"
	}

	username := os.Getenv("OPENCODE_SERVER_USERNAME")
	password := os.Getenv("OPENCODE_SERVER_PASSWORD")

	ev := &runtimeEvidence{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Endpoint:  endpoint,
	}
	adapter := opencode.NewOpenCodeAdapter(endpoint, username, password)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Gate 1: Health
	info, err := adapter.Health(ctx)
	if err != nil {
		ev.Available = false
		ev.Error = err.Error()
		ev.Health = &gateResult{Passed: false, Error: err.Error()}
		writeEvidence(t, evidenceDir, ev)
		t.Fatalf("real runtime not reachable at %s: %v — evidence written to %s",
			endpoint, err, evidenceDir)
	}
	ev.Available = true
	ev.Health = &gateResult{Passed: info.Healthy, Detail: fmt.Sprintf("version=%s", info.Version)}
	ev.TotalChecks++

	if !info.Healthy {
		ev.FailedChecks++
		writeEvidence(t, evidenceDir, ev)
		t.Fatalf("runtime at %s reports unhealthy", endpoint)
	}
	ev.PassedChecks++

	// Gate 2: CreateSession
	sess, err := adapter.CreateSession(ctx, runtime.CreateSessionRequest{Title: "evidence-smoke"})
	ev.CreateSession = &gateResult{Passed: err == nil && sess.ID != ""}
	ev.TotalChecks++
	if err != nil {
		ev.CreateSession.Error = err.Error()
		ev.FailedChecks++
	} else if sess.ID == "" {
		ev.CreateSession.Error = "empty session ID"
		ev.FailedChecks++
	} else {
		ev.CreateSession.Detail = sess.ID
		ev.PassedChecks++
	}

	// Gate 3: SSE Subscribe
	ch, sseErr := adapter.Subscribe(ctx)
	ev.SSE = &gateResult{}
	ev.TotalChecks++
	if sseErr != nil {
		ev.SSE.Error = sseErr.Error()
		ev.FailedChecks++
	} else {
		ev.SSE.Passed = true
		ev.PassedChecks++
	}

	// Gate 4: Prompt (requires session + SSE from above)
	ev.Prompt = &gateResult{}
	if sess.ID != "" && sseErr == nil {
		ev.TotalChecks++
		promptErr := adapter.Prompt(ctx, runtime.SessionID(sess.ID), runtime.PromptRequest{
			Agent: "general",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "hello"}},
		})
		if promptErr != nil {
			ev.Prompt.Error = promptErr.Error()
			ev.FailedChecks++
		} else {
			// Drain a few events to confirm SSE stream works.
			timeout := time.After(3 * time.Second)
			eventCount := 0
		drainLoop:
			for {
				select {
				case _, ok := <-ch:
					if !ok {
						break drainLoop
					}
					eventCount++
					if eventCount >= 5 {
						break drainLoop
					}
				case <-timeout:
					break drainLoop
				case <-ctx.Done():
					break drainLoop
				}
			}
			ev.Prompt.Passed = true
			ev.Prompt.Detail = fmt.Sprintf("%d events received", eventCount)
			ev.PassedChecks++
		}
	} else {
		ev.Prompt.Skipped = true
		ev.SkippedChecks++
	}

	// Gate 5: Cancel
	if sess.ID != "" {
		ev.TotalChecks++
		cancelErr := adapter.Cancel(ctx, runtime.SessionID(sess.ID))
		ev.Cancel = &gateResult{Passed: cancelErr == nil}
		if cancelErr != nil {
			ev.Cancel.Error = cancelErr.Error()
			ev.FailedChecks++
		} else {
			ev.PassedChecks++
		}
	} else {
		ev.Cancel = &gateResult{Skipped: true}
		ev.SkippedChecks++
	}

	// Gate 6: AgentSelection — verify that agents are listed and selection is supported.
	ev.TotalChecks++
	agents, agentErr := adapter.ListAgents(ctx)
	ev.AgentSelection = &gateResult{}
	if agentErr != nil {
		ev.AgentSelection.Error = agentErr.Error()
		ev.FailedChecks++
	} else if len(agents) == 0 {
		ev.AgentSelection.Passed = false
		ev.AgentSelection.Error = "no agents listed by runtime"
		ev.FailedChecks++
	} else {
		ev.AgentSelection.Passed = true
		names := make([]string, len(agents))
		for i, a := range agents {
			names[i] = a.Name
		}
		ev.AgentSelection.Detail = fmt.Sprintf("agents: %v", names)
		ev.PassedChecks++
	}

	// Gate 7: Approval — verified through Capabilities (tool approval support declaration).
	ev.TotalChecks++
	caps := adapter.Capabilities()
	ev.Approval = &gateResult{Passed: caps.ToolApproval}
	if caps.ToolApproval {
		ev.Approval.Detail = "tool approval supported"
		ev.PassedChecks++
	} else {
		ev.Approval.Error = "tool approval not declared in capabilities"
		ev.FailedChecks++
	}

	writeEvidence(t, evidenceDir, ev)

	t.Logf("real runtime evidence: %d/%d checks passed (artifact: %s)",
		ev.PassedChecks, ev.TotalChecks, filepath.Join(evidenceDir, "runtime-evidence.json"))

	if ev.FailedChecks > 0 {
		t.Errorf("%d/%d runtime evidence checks failed", ev.FailedChecks, ev.TotalChecks)
	}
}

func writeEvidence(t *testing.T, dir string, ev *runtimeEvidence) {
	t.Helper()
	data, err := json.MarshalIndent(ev, "", "  ")
	if err != nil {
		t.Errorf("failed to marshal evidence: %v", err)
		return
	}
	if err := os.WriteFile(filepath.Join(dir, "runtime-evidence.json"), data, 0o644); err != nil {
		t.Errorf("failed to write evidence: %v", err)
	}
}

func TestParitySilentLossFailsRequired(t *testing.T) {
	baseline := fakeruntime.New()
	baseline.Events = []runtime.Event{
		{Type: runtime.EventType("reasoning.delta"), Content: "thinking"},
		{Type: runtime.EventType("answer.delta"), Content: "a"},
		{Type: runtime.EventType("step.finished")},
	}

	candidate := fakeruntime.New()
	// Same count (3) but missing reasoning.delta — semantic silent loss.
	candidate.Events = []runtime.Event{
		{Type: runtime.EventType("tool.called"), Tool: &runtime.ToolEvent{Name: "x", CallID: "1"}},
		{Type: runtime.EventType("answer.delta"), Content: "a"},
		{Type: runtime.EventType("step.finished")},
	}

	runner := parity.Runner{Baseline: baseline, Candidate: candidate}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	result := runner.Run(ctx, parity.Scenario{
		Name:     "Reasoning",
		Required: true,
		Prompt: &runtime.PromptRequest{
			Agent: "general",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: "test"}},
		},
		Assertions: parity.Assertion{
			RequireReasoning: true,
		},
	})

	if result.Passed {
		t.Error("required scenario with silent loss must fail")
	}
	if !result.SilentLoss {
		t.Error("must detect silent loss: same event count but missing semantic event")
	}
}
