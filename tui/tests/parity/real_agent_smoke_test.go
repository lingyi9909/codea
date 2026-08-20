package parity_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"codea/tui/internal/opencode"
	"codea/tui/internal/runtime"
)

// agentEvidence is the auditable artifact proving the enterprise agents were
// actually loaded and enforced by the real locked runtime. It is written to
// evidence/agent-evidence.json by the run-real-agent-smoke.sh harness.
type agentEvidence struct {
	Timestamp string `json:"timestamp"`
	Endpoint  string `json:"endpoint"`
	Available bool   `json:"available"`
	Version   string `json:"version,omitempty"`
	Error     string `json:"error,omitempty"`

	AgentsListed     *gateResult `json:"agentsListed"`
	ReviewerRead     *gateResult `json:"reviewerRead"`
	ReviewerWriteDen *gateResult `json:"reviewerWriteDenied"`
	UnitTestRead     *gateResult `json:"unitTestRead"`
	UnitTestWriteDen *gateResult `json:"unitTestWriteDenied"`

	TotalChecks   int `json:"totalChecks"`
	PassedChecks  int `json:"passedChecks"`
	FailedChecks  int `json:"failedChecks"`
	SkippedChecks int `json:"skippedChecks"`
}

func (ev *agentEvidence) gate(passed bool, detail string, err error) *gateResult {
	g := &gateResult{Passed: passed, Detail: detail}
	if err != nil {
		g.Error = err.Error()
	}
	ev.TotalChecks++
	switch {
	case g.Passed:
		ev.PassedChecks++
	case g.Skipped:
		ev.SkippedChecks++
	default:
		ev.FailedChecks++
	}
	return g
}

// runAgentScenario drives a single agent session to idle, mirroring runScenario
// but selecting a specific agent instead of the hardcoded "build".
func runAgentScenario(t *testing.T, adapter *opencode.OpenCodeAdapter, ch <-chan runtime.Event, state *smokeState, agent, promptText string) *toolObservation {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	sess, err := adapter.CreateSession(ctx, runtime.CreateSessionRequest{Title: "real-agent-smoke"})
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	sid := sess.ID

	if err := adapter.Prompt(ctx, runtime.SessionID(sid), runtime.PromptRequest{
		Agent: agent,
		Parts: []runtime.PromptPart{runtime.TextPart{Text: promptText}},
	}); err != nil {
		t.Fatalf("Prompt(%s, %q): %v", agent, promptText, err)
	}

	obs := newToolObservation()
	timeout := time.After(35 * time.Second)
	for !obs.idled {
		select {
		case ev, ok := <-ch:
			if !ok {
				t.Fatalf("event channel closed before idle (agent=%s prompt=%q)", agent, promptText)
			}
			if ev.RawType == "plugin.added" {
				state.pluginAdded++
			}
			if ev.SessionID != sid {
				continue
			}
			obs.collect(ev)
		case <-timeout:
			t.Fatalf("timed out waiting for idle (agent=%s prompt=%q); called=%v success=%v", agent, promptText, obs.called, obs.success)
		case <-ctx.Done():
			t.Fatalf("context done waiting for idle (agent=%s prompt=%q): %v", agent, promptText, ctx.Err())
		}
	}
	return obs
}

// TestRealAgentEvidence verifies the enterprise agents (code-reviewer,
// unit-test-generator) are materialized into the real locked OpenCode runtime
// and that their deny permissions are enforced server-side: the reviewer may
// read but its write/edit/bash are denied even when the model attempts them.
func TestRealAgentEvidence(t *testing.T) {
	evidenceDir := filepath.Join("evidence")
	_ = os.MkdirAll(evidenceDir, 0o755)

	endpoint := os.Getenv("OPENCODE_ENDPOINT")
	if endpoint == "" {
		endpoint = os.Getenv("OPENCODE_SERVER_URL")
	}
	explicitEndpoint := endpoint != ""
	if endpoint == "" {
		endpoint = "http://127.0.0.1:4141"
	}
	username := os.Getenv("OPENCODE_SERVER_USERNAME")
	password := os.Getenv("OPENCODE_SERVER_PASSWORD")

	ev := &agentEvidence{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Endpoint:  endpoint,
	}
	adapter := opencode.NewOpenCodeAdapter(endpoint, username, password)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	info, err := adapter.Health(ctx)
	if err != nil {
		ev.Available = false
		ev.Error = err.Error()
		if explicitEndpoint {
			data, _ := json.MarshalIndent(ev, "", "  ")
			_ = os.WriteFile(filepath.Join(evidenceDir, "agent-evidence.json"), data, 0o644)
		}
		t.Skipf("real runtime not reachable at %s: %v; run scripts/run-real-agent-smoke.sh", endpoint, err)
	}
	ev.Available = true
	ev.Version = info.Version

	// Gate 1: the enterprise agents are materialized and listed by the runtime.
	agents, agentErr := adapter.ListAgents(ctx)
	{
		names := map[string]bool{}
		for _, a := range agents {
			names[a.Name] = true
		}
		ev.AgentsListed = ev.gate(
			agentErr == nil && names["code-reviewer"] && names["unit-test-generator"],
			fmt.Sprintf("agents listed: code-reviewer=%v unit-test-generator=%v (total %d)",
				names["code-reviewer"], names["unit-test-generator"], len(agents)),
			agentErr)
	}

	ch, sseErr := adapter.Subscribe(ctx)
	if sseErr != nil {
		ev.AgentsListed = ev.gate(false, "subscribe", sseErr)
		writeAgentEvidence(t, evidenceDir, ev)
		t.Fatalf("Subscribe: %v", sseErr)
	}

	state := &smokeState{}

	// Gate 2: code-reviewer can read (allow) — the agent runs end-to-end.
	reviewerRead := runAgentScenario(t, adapter, ch, state, "code-reviewer", "READ the file please")
	ev.ReviewerRead = ev.gate(
		reviewerRead.calledOnce("read") && reviewerRead.succeededOnce("read") && reviewerRead.answered(),
		"code-reviewer read tool executed and agent continued", nil)

	// Gate 3: code-reviewer write is denied server-side even though the model
	// emits a write call — the permission deny is real, not a prompt hint.
	reviewerWrite := runAgentScenario(t, adapter, ch, state, "code-reviewer", "WRITE the file please")
	ev.ReviewerWriteDen = ev.gate(
		reviewerWrite.calledOnce("write") && !reviewerWrite.succeededOnce("write"),
		"code-reviewer write called but did not succeed (permission deny)", nil)

	// Gate 4: unit-test-generator can read (allow).
	unitTestRead := runAgentScenario(t, adapter, ch, state, "unit-test-generator", "READ the file please")
	ev.UnitTestRead = ev.gate(
		unitTestRead.calledOnce("read") && unitTestRead.succeededOnce("read") && unitTestRead.answered(),
		"unit-test-generator read tool executed and agent continued", nil)

	// Gate 5: unit-test-generator write is denied server-side.
	unitTestWrite := runAgentScenario(t, adapter, ch, state, "unit-test-generator", "WRITE the file please")
	ev.UnitTestWriteDen = ev.gate(
		unitTestWrite.calledOnce("write") && !unitTestWrite.succeededOnce("write"),
		"unit-test-generator write called but did not succeed (permission deny)", nil)

	writeAgentEvidence(t, evidenceDir, ev)

	t.Logf("real agent evidence: %d/%d checks passed (artifact: %s)",
		ev.PassedChecks, ev.TotalChecks, filepath.Join(evidenceDir, "agent-evidence.json"))

	if ev.FailedChecks > 0 {
		t.Errorf("%d/%d agent evidence checks failed", ev.FailedChecks, ev.TotalChecks)
	}
}

func writeAgentEvidence(t *testing.T, dir string, ev *agentEvidence) {
	t.Helper()
	data, err := json.MarshalIndent(ev, "", "  ")
	if err != nil {
		t.Errorf("marshal agent evidence: %v", err)
		return
	}
	if err := os.WriteFile(filepath.Join(dir, "agent-evidence.json"), data, 0o644); err != nil {
		t.Errorf("write agent evidence: %v", err)
	}
}
