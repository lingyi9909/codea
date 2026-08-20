package parity_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

	// Custom Tool whitelist: proves the fail-closed manifest permission is
	// enforced for enterprise custom tools, not just native read/write.
	ReviewerCollectAllowed *gateResult `json:"reviewerCollectReviewAllowed"`
	ReviewerWriteTestDen   *gateResult `json:"reviewerWriteTestDenied"`
	ReviewerRunTestDen     *gateResult `json:"reviewerRunTestDenied"`
	ReviewerWriteDocDen    *gateResult `json:"reviewerWriteDocDenied"`
	UnitTestWriteTestAllow *gateResult `json:"unitTestWriteTestAllowed"`
	UnitTestWriteDocDen    *gateResult `json:"unitTestWriteDocDenied"`
	UnitTestCollectDen     *gateResult `json:"unitTestCollectReviewDenied"`

	// Real agent workflow E2E.
	ReviewerWorkflow        *gateResult `json:"reviewerWorkflow"`
	UnitTestWorkflow        *gateResult `json:"unitTestWorkflow"`
	UnitTestFailureWorkflow *gateResult `json:"unitTestFailureWorkflow"`

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
// but selecting a specific agent instead of the hardcoded "build". onApproval,
// when non-nil, is invoked for each permission.asked event so custom tools that
// carry write/execute actions (write_test_file, run_project_test) can be
// approved through the real approval flow instead of hanging.
func runAgentScenario(t *testing.T, adapter *opencode.OpenCodeAdapter, ch <-chan runtime.Event, state *smokeState, agent, promptText string, onApproval func(id string) error) *toolObservation {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
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
	timeout := time.After(145 * time.Second)
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
			if ev.Type == runtime.EventType("approval.requested") && onApproval != nil {
				if err := onApproval(ev.Approval.ID); err != nil {
					t.Fatalf("approval reply: %v", err)
				}
			}
		case <-timeout:
			t.Fatalf("timed out waiting for idle (agent=%s prompt=%q); called=%v success=%v", agent, promptText, obs.called, obs.success)
		case <-ctx.Done():
			t.Fatalf("context done waiting for idle (agent=%s prompt=%q): %v", agent, promptText, ctx.Err())
		}
	}
	return obs
}

// reviewerJSONValid reports whether a code-reviewer scenario's final answer is
// a valid output-schema JSON object: a PASS summary, a findings array and a
// reviewStats object. It tolerates the runtime wrapping the JSON in a code fence.
func reviewerJSONValid(obs *toolObservation) bool {
	text := strings.TrimSpace(strings.Join(obs.answerText, ""))
	m, ok := extractJSONObject(text)
	if !ok {
		return false
	}
	summary, ok := m["summary"].(map[string]any)
	if !ok || summary["result"] != "PASS" {
		return false
	}
	if _, ok := m["findings"].([]any); !ok {
		return false
	}
	if _, ok := m["reviewStats"].(map[string]any); !ok {
		return false
	}
	if _, ok := m["businessKnowledgeUnavailable"].(bool); !ok {
		return false
	}
	return true
}

func extractJSONObject(text string) (map[string]any, bool) {
	t := strings.TrimSpace(text)
	if strings.HasPrefix(t, "```") {
		if i := strings.IndexByte(t, '\n'); i >= 0 {
			t = strings.TrimSpace(t[i+1:])
		}
		t = strings.TrimSuffix(t, "```")
		t = strings.TrimSpace(t)
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(t), &m); err == nil {
		return m, true
	}
	start := strings.IndexByte(t, '{')
	end := strings.LastIndexByte(t, '}')
	if start < 0 || end <= start {
		return nil, false
	}
	if err := json.Unmarshal([]byte(t[start:end+1]), &m); err != nil {
		return nil, false
	}
	return m, true
}

func jsonNumber(m map[string]any, key string) (float64, bool) {
	v, ok := m[key].(float64)
	return v, ok
}

// unitTestConclusionValid proves the final agent conclusion is derived from the
// structured run_project_test result instead of a hard-coded workflow message.
func unitTestConclusionValid(obs *toolObservation, want string) bool {
	text := strings.TrimSpace(strings.Join(obs.answerText, ""))
	m, ok := extractJSONObject(text)
	if !ok || m["result"] != want || m["source"] != "run_project_test" {
		return false
	}
	category, _ := m["category"].(string)
	exitCode, exitOK := jsonNumber(m, "exitCode")
	passed, passedOK := jsonNumber(m, "passed")
	failed, failedOK := jsonNumber(m, "failed")
	errors, errorsOK := jsonNumber(m, "errors")
	if !(exitOK && passedOK && failedOK && errorsOK) {
		return false
	}
	if want == "PASS" {
		return category == "PASS" && exitCode == 0 && passed >= 1 && failed == 0 && errors == 0
	}
	return want == "FAIL" && (category != "PASS" || exitCode != 0 || failed > 0 || errors > 0)
}

// verifyGeneratedJUnit proves write_test_file created the expected source and
// that real Maven/Surefire actually produced a report for that generated class.
func verifyGeneratedJUnit(smokeDir, sourceName, className string, wantFailure bool) (bool, string) {
	if smokeDir == "" {
		return false, "SMOKE_DIR is empty"
	}
	source := filepath.Join(smokeDir, "src", "test", "java", "com", "example", "demo", sourceName+".java")
	if _, err := os.Stat(source); err != nil {
		return false, fmt.Sprintf("generated source missing: %v", err)
	}
	report := filepath.Join(smokeDir, "target", "surefire-reports", className+".txt")
	data, err := os.ReadFile(report)
	if err != nil {
		return false, fmt.Sprintf("Surefire report missing: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "Test set: "+className) {
		return false, "Surefire report does not identify generated test class"
	}
	if wantFailure {
		if !strings.Contains(text, "Tests run: 1, Failures: 1, Errors: 0") {
			return false, "Surefire did not execute deterministic failing generated test"
		}
		return true, "generated failing JUnit executed by real Maven/Surefire"
	}
	if !strings.Contains(text, "Tests run: 1, Failures: 0, Errors: 0") {
		return false, "Surefire did not execute green generated test"
	}
	return true, "GeneratedFlowTest.java exists and real Maven/Surefire executed it green"
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

	ctx, cancel := context.WithTimeout(context.Background(), 360*time.Second)
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
	reviewerRead := runAgentScenario(t, adapter, ch, state, "code-reviewer", "READ the file please", nil)
	ev.ReviewerRead = ev.gate(
		reviewerRead.calledOnce("read") && reviewerRead.succeededOnce("read") && reviewerRead.answered(),
		"code-reviewer read tool executed and agent continued", nil)

	// Gate 3: code-reviewer write is denied server-side even though the model
	// emits a write call — the permission deny is real, not a prompt hint.
	reviewerWrite := runAgentScenario(t, adapter, ch, state, "code-reviewer", "WRITE the file please", nil)
	ev.ReviewerWriteDen = ev.gate(
		reviewerWrite.calledOnce("write") && !reviewerWrite.succeededOnce("write"),
		"code-reviewer write called but did not succeed (permission deny)", nil)

	// Gate 4: unit-test-generator can read (allow).
	unitTestRead := runAgentScenario(t, adapter, ch, state, "unit-test-generator", "READ the file please", nil)
	ev.UnitTestRead = ev.gate(
		unitTestRead.calledOnce("read") && unitTestRead.succeededOnce("read") && unitTestRead.answered(),
		"unit-test-generator read tool executed and agent continued", nil)

	// Gate 5: unit-test-generator write is denied server-side.
	unitTestWrite := runAgentScenario(t, adapter, ch, state, "unit-test-generator", "WRITE the file please", nil)
	ev.UnitTestWriteDen = ev.gate(
		unitTestWrite.calledOnce("write") && !unitTestWrite.succeededOnce("write"),
		"unit-test-generator write called but did not succeed (permission deny)", nil)

	// approveOnce grants the real approval flow for custom write/execute tools so
	// an ALLOWED tool can run to completion rather than hang waiting for a user.
	approveOnce := func(id string) error {
		return adapter.ReplyApproval(ctx, runtime.ApprovalID(id), runtime.ApprovalReply{Decision: runtime.ApprovalOnce})
	}

	// Custom Tool whitelist: the fail-closed manifest permission must hold for
	// enterprise custom tools, not just native read/write.

	// code-reviewer whitelist: collect_review_context allow.
	reviewerCollect := runAgentScenario(t, adapter, ch, state, "code-reviewer", "CALL collect_review_context please", nil)
	ev.ReviewerCollectAllowed = ev.gate(
		reviewerCollect.calledOnce("collect_review_context") && reviewerCollect.succeededOnce("collect_review_context"),
		"code-reviewer collect_review_context allowed and executed", nil)

	// code-reviewer whitelist: write_test_file deny (Reviewer is read-only).
	reviewerWt := runAgentScenario(t, adapter, ch, state, "code-reviewer", "CALL write_test_file please", nil)
	ev.ReviewerWriteTestDen = ev.gate(
		reviewerWt.calledOnce("write_test_file") && !reviewerWt.succeededOnce("write_test_file"),
		"code-reviewer write_test_file denied by whitelist", nil)

	// code-reviewer whitelist: run_project_test deny.
	reviewerRt := runAgentScenario(t, adapter, ch, state, "code-reviewer", "CALL run_project_test please", nil)
	ev.ReviewerRunTestDen = ev.gate(
		reviewerRt.calledOnce("run_project_test") && !reviewerRt.succeededOnce("run_project_test"),
		"code-reviewer run_project_test denied by whitelist", nil)

	// code-reviewer whitelist: write_document deny.
	reviewerWd := runAgentScenario(t, adapter, ch, state, "code-reviewer", "CALL write_document please", nil)
	ev.ReviewerWriteDocDen = ev.gate(
		reviewerWd.calledOnce("write_document") && !reviewerWd.succeededOnce("write_document"),
		"code-reviewer write_document denied by whitelist", nil)

	// unit-test-generator whitelist: write_test_file allow (approve the write).
	utWt := runAgentScenario(t, adapter, ch, state, "unit-test-generator", "CALL write_test_file please", approveOnce)
	ev.UnitTestWriteTestAllow = ev.gate(
		utWt.calledOnce("write_test_file") && utWt.succeededOnce("write_test_file"),
		"unit-test-generator write_test_file allowed and executed", nil)

	// unit-test-generator whitelist: write_document deny (not in its tool set).
	utWd := runAgentScenario(t, adapter, ch, state, "unit-test-generator", "CALL write_document please", nil)
	ev.UnitTestWriteDocDen = ev.gate(
		utWd.calledOnce("write_document") && !utWd.succeededOnce("write_document"),
		"unit-test-generator write_document denied by whitelist", nil)

	// unit-test-generator whitelist: collect_review_context deny.
	utCollect := runAgentScenario(t, adapter, ch, state, "unit-test-generator", "CALL collect_review_context please", nil)
	ev.UnitTestCollectDen = ev.gate(
		utCollect.calledOnce("collect_review_context") && !utCollect.succeededOnce("collect_review_context"),
		"unit-test-generator collect_review_context denied by whitelist", nil)

	// Reviewer workflow E2E: collect_review_context → output-schema JSON.
	reviewerFlow := runAgentScenario(t, adapter, ch, state, "code-reviewer", "RUN the REVIEWFLOW please", nil)
	ev.ReviewerWorkflow = ev.gate(
		reviewerFlow.calledOnce("collect_review_context") && reviewerFlow.succeededOnce("collect_review_context") && reviewerJSONValid(reviewerFlow),
		"code-reviewer workflow: collect_review_context + valid output-schema JSON", nil)

	// Unit Test workflow E2E: analyze_test_project → write_test_file → run_project_test
	// → final PASS derived from the real run_project_test result. The fixture's
	// mvnw stub is removed by the shell harness, so this must be real Maven/JUnit.
	utFlow := runAgentScenario(t, adapter, ch, state, "unit-test-generator", "RUN the UTFLOW please", approveOnce)
	flowExecuted, flowDetail := verifyGeneratedJUnit(os.Getenv("SMOKE_DIR"), "GeneratedFlowTest", "com.example.demo.GeneratedFlowTest", false)
	ev.UnitTestWorkflow = ev.gate(
		utFlow.calledOnce("analyze_test_project") && utFlow.succeededOnce("analyze_test_project") &&
			utFlow.calledOnce("write_test_file") && utFlow.succeededOnce("write_test_file") &&
			utFlow.calledOnce("run_project_test") && utFlow.succeededOnce("run_project_test") &&
			unitTestConclusionValid(utFlow, "PASS") && flowExecuted,
		"unit-test workflow: analyze -> write -> real Maven run -> final PASS; "+flowDetail, nil)

	// Deterministic failure E2E: the generated JUnit fails, run_project_test
	// returns a structured FAIL, and the agent's final conclusion must be FAIL.
	utFailFlow := runAgentScenario(t, adapter, ch, state, "unit-test-generator", "RUN the UTFLOW_FAIL please", approveOnce)
	failExecuted, failDetail := verifyGeneratedJUnit(os.Getenv("SMOKE_DIR"), "GeneratedFailureFlowTest", "com.example.demo.GeneratedFailureFlowTest", true)
	ev.UnitTestFailureWorkflow = ev.gate(
		utFailFlow.calledOnce("analyze_test_project") && utFailFlow.succeededOnce("analyze_test_project") &&
			utFailFlow.calledOnce("write_test_file") && utFailFlow.succeededOnce("write_test_file") &&
			utFailFlow.calledOnce("run_project_test") && utFailFlow.succeededOnce("run_project_test") &&
			unitTestConclusionValid(utFailFlow, "FAIL") && failExecuted,
		"unit-test failure workflow: run_project_test failure -> final FAIL; "+failDetail, nil)

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
