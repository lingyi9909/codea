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

// gateResult captures a single gate check outcome for the evidence artifact.
type gateResult struct {
	Passed  bool   `json:"passed"`
	Detail  string `json:"detail,omitempty"`
	Error   string `json:"error,omitempty"`
	Skipped bool   `json:"skipped,omitempty"`
}

// runtimeEvidence captures the result of each real-runtime gate check. It is
// written to evidence/runtime-evidence.json so a human reviewer can audit that
// the real locked OpenCode was actually exercised (available=true, failed=0),
// rather than skipped.
type runtimeEvidence struct {
	Timestamp string `json:"timestamp"`
	Endpoint  string `json:"endpoint"`
	Available bool   `json:"available"`
	Version   string `json:"version,omitempty"`
	Error     string `json:"error,omitempty"`

	Health         *gateResult `json:"health"`
	CreateSession  *gateResult `json:"createSession"`
	SSE            *gateResult `json:"sse"`
	AgentSelection *gateResult `json:"agentSelection"`
	Capabilities   *gateResult `json:"capabilities"`
	Read           *gateResult `json:"read"`
	Write          *gateResult `json:"write"`
	Edit           *gateResult `json:"edit"`
	BashOnce       *gateResult `json:"bashApprovalOnce"`
	BashAlways     *gateResult `json:"bashApprovalAlways"`
	BashReject     *gateResult `json:"bashApprovalReject"`
	Subagent       *gateResult `json:"subagent"`
	Skill          *gateResult `json:"skill"`
	SkillManager   *gateResult `json:"skillManager"`
	Plugin         *gateResult `json:"plugin"`
	SessionResume  *gateResult `json:"sessionResume"`
	Cancel         *gateResult `json:"cancel"`

	TotalChecks   int `json:"totalChecks"`
	PassedChecks  int `json:"passedChecks"`
	FailedChecks  int `json:"failedChecks"`
	SkippedChecks int `json:"skippedChecks"`
}

func (ev *runtimeEvidence) record(g *gateResult) {
	ev.TotalChecks++
	switch {
	case g.Passed:
		ev.PassedChecks++
	case g.Skipped:
		ev.SkippedChecks++
	default:
		ev.FailedChecks++
	}
}

func (ev *runtimeEvidence) gate(passed bool, detail string, err error) *gateResult {
	g := &gateResult{Passed: passed, Detail: detail}
	if err != nil {
		g.Error = err.Error()
	}
	ev.record(g)
	return g
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

// toolObservation accumulates per-scenario tool lifecycle, approval, answer,
// and idle observations for a single session.
type toolObservation struct {
	called     map[string][]string
	success    map[string][]string
	failed     map[string][]string
	approvals  []approvalObs
	answerText []string
	idled      bool
}

type approvalObs struct {
	id         string
	permission string
	command    string
}

func newToolObservation() *toolObservation {
	return &toolObservation{
		called:  map[string][]string{},
		success: map[string][]string{},
		failed:  map[string][]string{},
	}
}

func (o *toolObservation) collect(ev runtime.Event) {
	if isIdle(ev) {
		o.idled = true
	}
	switch ev.Type {
	case runtime.EventType("tool.called"):
		if ev.Tool != nil {
			o.called[ev.Tool.Name] = append(o.called[ev.Tool.Name], ev.Tool.CallID)
		}
	case runtime.EventType("tool.success"):
		if ev.Tool != nil {
			o.success[ev.Tool.Name] = append(o.success[ev.Tool.Name], ev.Tool.CallID)
		}
	case runtime.EventType("tool.failed"):
		if ev.Tool != nil {
			o.failed[ev.Tool.Name] = append(o.failed[ev.Tool.Name], ev.Tool.CallID)
		}
	case runtime.EventType("answer.delta"):
		o.answerText = append(o.answerText, ev.Content)
	case runtime.EventType("approval.requested"):
		if ev.Approval != nil {
			o.approvals = append(o.approvals, approvalObs{
				id:         ev.Approval.ID,
				permission: ev.Approval.Permission,
				command:    ev.Approval.Command,
			})
		}
	}
}

func (o *toolObservation) calledOnce(name string) bool {
	return len(o.called[name]) > 0
}

func (o *toolObservation) succeededOnce(name string) bool {
	return len(o.success[name]) > 0
}

func (o *toolObservation) answered() bool {
	return strings.TrimSpace(strings.Join(o.answerText, "")) != ""
}

func isIdle(ev runtime.Event) bool {
	// session.idle is the canonical terminal event emitted when the agent loop
	// exits. session.status with type=idle fires just before it; treating both
	// as "idle" makes the drain return early on the status event and leaves the
	// trailing session.idle in the channel, which then falsely terminates the
	// next scenario's drain. Only session.idle marks completion.
	return ev.RawType == "session.idle"
}

// smokeState carries cross-scenario observations on the global event stream,
// in particular plugin.added events (which carry no sessionID) that fire while
// the runtime finishes loading its plugin set.
type smokeState struct {
	pluginAdded int
}

// runScenario creates a fresh session, prompts the fake model, and drains the
// shared global event stream for that session until it idles. onApproval, when
// non-nil, is invoked for each permission.asked event with the approval ID so
// the smoke can exercise the real approval once/reject flow.
func runScenario(t *testing.T, adapter *opencode.OpenCodeAdapter, ch <-chan runtime.Event, state *smokeState, promptText string, onApproval func(id string) error) *toolObservation {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Second)
	defer cancel()

	sess, err := adapter.CreateSession(ctx, runtime.CreateSessionRequest{Title: "real-parity-smoke"})
	if err != nil {
		t.Fatalf("CreateSession: %v", err)
	}
	sid := sess.ID

	if err := adapter.Prompt(ctx, runtime.SessionID(sid), runtime.PromptRequest{
		Agent: "build",
		Parts: []runtime.PromptPart{runtime.TextPart{Text: promptText}},
	}); err != nil {
		t.Fatalf("Prompt(%q): %v", promptText, err)
	}

	obs := newToolObservation()
	timeout := time.After(35 * time.Second)
	for !obs.idled {
		select {
		case ev, ok := <-ch:
			if !ok {
				t.Fatalf("event channel closed before idle (prompt=%q)", promptText)
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
			t.Fatalf("timed out waiting for session %s idle (prompt=%q); called=%v success=%v", sid, promptText, obs.called, obs.success)
		case <-ctx.Done():
			t.Fatalf("context done waiting for idle (prompt=%q): %v", promptText, ctx.Err())
		}
	}
	return obs
}

// TestRealRuntimeEvidence verifies the full AgentRuntime contract AND the
// native General Agent tool surface against a running locked OpenCode server,
// recording an auditable evidence artifact. When no runtime is reachable it
// skips (recording that fact) so `go test ./...` remains green in dev; the
// scripts/run-real-parity-smoke.sh harness guarantees the runtime is up so the
// smoke actually executes and writes fresh evidence with available=true.
func TestRealRuntimeEvidence(t *testing.T) {
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
	smokeDir := os.Getenv("SMOKE_DIR")
	if smokeDir == "" {
		smokeDir = "/tmp"
	}

	ev := &runtimeEvidence{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Endpoint:  endpoint,
	}
	adapter := opencode.NewOpenCodeAdapter(endpoint, username, password)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Gate: Health (also the reachability probe).
	info, err := adapter.Health(ctx)
	if err != nil {
		ev.Available = false
		ev.Error = err.Error()
		ev.Health = &gateResult{Passed: false, Error: err.Error()}
		// Only persist a skip/available=false artifact when the endpoint was
		// explicitly requested (the smoke harness). A plain `go test ./...`
		// with no OPENCODE_ENDPOINT must not overwrite the committed green
		// evidence with a stale connection-refused snapshot.
		if explicitEndpoint {
			writeEvidence(t, evidenceDir, ev)
		}
		t.Skipf("real runtime not reachable at %s: %v; run scripts/run-real-parity-smoke.sh to exercise it", endpoint, err)
	}
	ev.Available = true
	ev.Version = info.Version
	ev.Health = ev.gate(info.Healthy, fmt.Sprintf("version=%s", info.Version), nil)
	if !info.Healthy {
		writeEvidence(t, evidenceDir, ev)
		t.Fatalf("runtime at %s reports unhealthy", endpoint)
	}

	// Gate: CreateSession.
	sess, err := adapter.CreateSession(ctx, runtime.CreateSessionRequest{Title: "real-parity-evidence"})
	ev.CreateSession = ev.gate(err == nil && sess.ID != "", func() string {
		if err != nil {
			return ""
		}
		return sess.ID
	}(), err)

	// Gate: Subscribe (global SSE) — must succeed before scenarios.
	ch, sseErr := adapter.Subscribe(ctx)
	ev.SSE = ev.gate(sseErr == nil, "global event stream", sseErr)
	if sseErr != nil {
		writeEvidence(t, evidenceDir, ev)
		t.Fatalf("Subscribe: %v", sseErr)
	}

	// Gate: AgentSelection — agents must be listed and expose the explore subagent.
	agents, agentErr := adapter.ListAgents(ctx)
	{
		var names []string
		hasExplore := false
		for _, a := range agents {
			names = append(names, a.Name)
			if a.Name == "explore" {
				hasExplore = true
			}
		}
		ev.AgentSelection = ev.gate(agentErr == nil && len(agents) > 0 && hasExplore,
			fmt.Sprintf("agents=%v", names), agentErr)
	}

	// Gate: Capabilities — required product capabilities must be declared.
	caps := adapter.Capabilities()
	ev.Capabilities = ev.gate(caps.Sessions && caps.Streaming && caps.ToolApproval &&
		caps.Skills && caps.Plugins && caps.Subagents && caps.Abort,
		"required capabilities present", nil)

	state := &smokeState{}

	// Native tool surface: Read / Write / Edit / Bash (approval) / Subagent /
	// Skill / Plugin / Session-Resume / Cancel. Each scenario drives a real
	// tool through the locked OpenCode runtime; the fake model only scripts the
	// tool selection.

	// Read: tool read called + completed.
	readObs := runScenario(t, adapter, ch, state, "READ the file please", nil)
	ev.Read = ev.gate(readObs.calledOnce("read") && readObs.succeededOnce("read") && readObs.answered(),
		"read tool executed and agent continued", nil)

	// Write: tool write called + completed, and the file landed on disk.
	writeObs := runScenario(t, adapter, ch, state, "WRITE the file please", nil)
	writeFileOK := false
	if b, rerr := os.ReadFile(filepath.Join(smokeDir, "write-out.txt")); rerr == nil && strings.TrimSpace(string(b)) == "smoke-write-ok" {
		writeFileOK = true
	}
	ev.Write = ev.gate(writeObs.calledOnce("write") && writeObs.succeededOnce("write") && writeFileOK,
		"write tool executed and write-out.txt verified", nil)

	// Edit: tool edit called + completed, and edit-me.txt changed before→after.
	editObs := runScenario(t, adapter, ch, state, "EDIT the file please", nil)
	editFileOK := false
	if b, rerr := os.ReadFile(filepath.Join(smokeDir, "edit-me.txt")); rerr == nil && strings.TrimSpace(string(b)) == "after" {
		editFileOK = true
	}
	ev.Edit = ev.gate(editObs.calledOnce("edit") && editObs.succeededOnce("edit") && editFileOK,
		"edit tool executed and edit-me.txt verified", nil)

	// Bash approval once: permission.asked → once → bash completes → agent continues.
	var onceApprovals int
	bashOnceObs := runScenario(t, adapter, ch, state, "BASH please", func(id string) error {
		onceApprovals++
		return adapter.ReplyApproval(ctx, runtime.ApprovalID(id), runtime.ApprovalReply{Decision: runtime.ApprovalOnce})
	})
	ev.BashOnce = ev.gate(
		len(bashOnceObs.approvals) == 1 && onceApprovals == 1 &&
			bashOnceObs.calledOnce("bash") && bashOnceObs.succeededOnce("bash") && bashOnceObs.answered(),
		"permission.asked → once → bash executed → agent continued", nil)

	// Bash approval reject: permission.asked → reject → bash does NOT execute.
	var rejectApprovals int
	bashRejectObs := runScenario(t, adapter, ch, state, "BASH please", func(id string) error {
		rejectApprovals++
		return adapter.ReplyApproval(ctx, runtime.ApprovalID(id), runtime.ApprovalReply{Decision: runtime.ApprovalReject})
	})
	ev.BashReject = ev.gate(
		len(bashRejectObs.approvals) == 1 && rejectApprovals == 1 &&
			bashRejectObs.calledOnce("bash") && !bashRejectObs.succeededOnce("bash"),
		"permission.asked → reject → bash did not execute", nil)

	// Subagent: the model emits a task tool call delegating to the explore
	// subagent; OpenCode Runtime performs the scheduling (Codea does not).
	subagentObs := runScenario(t, adapter, ch, state, "SUBAGENT please", nil)
	ev.Subagent = ev.gate(subagentObs.calledOnce("task") && subagentObs.succeededOnce("task"),
		"task tool delegated to subagent and completed", nil)

	// Skill: the model emits a skill tool call; the skill plugin loads and
	// returns smoke-skill content.
	skillObs := runScenario(t, adapter, ch, state, "SKILL please", nil)
	ev.Skill = ev.gate(skillObs.calledOnce("skill") && skillObs.succeededOnce("skill"),
		"skill tool loaded smoke-skill and completed", nil)

	// SkillManager: the Codea Skill Manager's real-runtime contract — ListSkills
	// queries the real /skill endpoint and reports the runtime's loaded skills,
	// including the smoke-skill materialized in the controlled config dir.
	skillMgrSkills, skillMgrErr := adapter.ListSkills(ctx, "")
	{
		var names []string
		hasSmokeSkill := false
		for _, s := range skillMgrSkills {
			names = append(names, s.Name)
			if s.Name == "smoke-skill" {
				hasSmokeSkill = true
			}
		}
		ev.SkillManager = ev.gate(skillMgrErr == nil && len(skillMgrSkills) > 0 && hasSmokeSkill,
			fmt.Sprintf("skills=%v", names), skillMgrErr)
	}

	// Plugin: plugin.added events observed on the global stream (plugins load);
	// the skill tool above already proves a plugin is invocable.
	ev.Plugin = ev.gate(state.pluginAdded > 0,
		fmt.Sprintf("%d plugin.added events observed", state.pluginAdded), nil)

	// Session/Resume: two sessions, events tagged to the right session, resume
	// routes back to the same session without cross-streaming.
	ev.SessionResume = runSessionResume(t, adapter, ch, state)
	ev.record(ev.SessionResume)

	// Cancel: abort a session mid-stream (waiting for approval).
	ev.Cancel = runCancel(t, adapter, ch, state)
	ev.record(ev.Cancel)

	// Bash approval always: permission.asked → always → bash completes → agent
	// continues. Run LAST because an "always" reply persists a bash permission
	// rule for the project, which would auto-approve subsequent bash calls and
	// starve the reject/cancel scenarios of their permission.asked.
	var alwaysApprovals int
	bashAlwaysObs := runScenario(t, adapter, ch, state, "BASH please", func(id string) error {
		alwaysApprovals++
		return adapter.ReplyApproval(ctx, runtime.ApprovalID(id), runtime.ApprovalReply{Decision: runtime.ApprovalAlways})
	})
	ev.BashAlways = ev.gate(
		len(bashAlwaysObs.approvals) == 1 && alwaysApprovals == 1 &&
			bashAlwaysObs.calledOnce("bash") && bashAlwaysObs.succeededOnce("bash") && bashAlwaysObs.answered(),
		"permission.asked → always → bash executed → agent continued", nil)

	writeEvidence(t, evidenceDir, ev)

	t.Logf("real runtime evidence: %d/%d checks passed (artifact: %s)",
		ev.PassedChecks, ev.TotalChecks, filepath.Join(evidenceDir, "runtime-evidence.json"))

	if ev.FailedChecks > 0 {
		t.Errorf("%d/%d runtime evidence checks failed", ev.FailedChecks, ev.TotalChecks)
	}
}

// runSessionResume proves two sessions stay isolated and resume routes back to
// the correct session on the real runtime.
func runSessionResume(t *testing.T, adapter *opencode.OpenCodeAdapter, ch <-chan runtime.Event, state *smokeState) *gateResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	sessA, err := adapter.CreateSession(ctx, runtime.CreateSessionRequest{Title: "resume-a"})
	if err != nil {
		return &gateResult{Passed: false, Error: err.Error()}
	}
	sessB, err := adapter.CreateSession(ctx, runtime.CreateSessionRequest{Title: "resume-b"})
	if err != nil {
		return &gateResult{Passed: false, Error: err.Error()}
	}

	// Prompt A1 → read, then resume A2 → read again, both tagged to A.
	for i, prompt := range []string{"READ the file please", "READ the file please"} {
		if err := adapter.Prompt(ctx, runtime.SessionID(sessA.ID), runtime.PromptRequest{
			Agent: "build",
			Parts: []runtime.PromptPart{runtime.TextPart{Text: prompt}},
		}); err != nil {
			return &gateResult{Passed: false, Error: fmt.Errorf("prompt A%d: %w", i+1, err).Error()}
		}
		obs := drainSession(t, ch, state, sessA.ID, fmt.Sprintf("resume-a-%d", i+1))
		if !obs.calledOnce("read") || !obs.succeededOnce("read") {
			return &gateResult{Passed: false, Error: fmt.Sprintf("resume A%d: read lifecycle incomplete: called=%v success=%v", i+1, obs.called, obs.success)}
		}
	}

	// Prompt B1 → write, tagged to B (not A).
	if err := adapter.Prompt(ctx, runtime.SessionID(sessB.ID), runtime.PromptRequest{
		Agent: "build",
		Parts: []runtime.PromptPart{runtime.TextPart{Text: "WRITE the file please"}},
	}); err != nil {
		return &gateResult{Passed: false, Error: err.Error()}
	}
	obsB := drainSession(t, ch, state, sessB.ID, "resume-b")
	if !obsB.calledOnce("write") || !obsB.succeededOnce("write") {
		return &gateResult{Passed: false, Error: fmt.Sprintf("session B write incomplete: called=%v success=%v", obsB.called, obsB.success)}
	}

	// Both sessions must still be listed.
	sessions, err := adapter.ListSessions(ctx)
	if err != nil {
		return &gateResult{Passed: false, Error: err.Error()}
	}
	if len(sessions) < 2 {
		return &gateResult{Passed: false, Error: fmt.Sprintf("expected >=2 sessions, got %d", len(sessions))}
	}

	return &gateResult{Passed: true, Detail: fmt.Sprintf("A(resume)x2 + B isolated, %d sessions listed", len(sessions))}
}

// runCancel aborts a session mid-stream while it is waiting for approval, then
// proves the cancel actually took effect: the cancelled bash tool never
// succeeds, the cancelled session terminates (session.idle), and a subsequent
// fresh session still completes a full tool lifecycle (global SSE + runtime +
// adapter all survived).
func runCancel(t *testing.T, adapter *opencode.OpenCodeAdapter, ch <-chan runtime.Event, state *smokeState) *gateResult {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	sess, err := adapter.CreateSession(ctx, runtime.CreateSessionRequest{Title: "cancel-smoke"})
	if err != nil {
		return &gateResult{Passed: false, Error: err.Error()}
	}
	sid := sess.ID

	if err := adapter.Prompt(ctx, runtime.SessionID(sid), runtime.PromptRequest{
		Agent: "build",
		Parts: []runtime.PromptPart{runtime.TextPart{Text: "BASH please"}},
	}); err != nil {
		return &gateResult{Passed: false, Error: err.Error()}
	}

	// Drain the cancelled session until it idles. When the approval arrives,
	// cancel mid-stream; then keep consuming to observe the terminal state and
	// prove the cancelled tool never succeeds and no normal answer is emitted.
	obs := newToolObservation()
	cancelled := false
	deadline := time.After(25 * time.Second)
	for !obs.idled {
		select {
		case ev, ok := <-ch:
			if !ok {
				return &gateResult{Passed: false, Error: "event channel closed before cancelled session terminated"}
			}
			if ev.RawType == "plugin.added" {
				state.pluginAdded++
			}
			if ev.SessionID != sid {
				continue
			}
			obs.collect(ev)
			if !cancelled && ev.Type == runtime.EventType("approval.requested") {
				if err := adapter.Cancel(ctx, runtime.SessionID(sid)); err != nil {
					return &gateResult{Passed: false, Error: fmt.Sprintf("Cancel: %v", err)}
				}
				cancelled = true
			}
		case <-deadline:
			return &gateResult{Passed: false, Error: "timed out waiting for cancelled session to terminate"}
		case <-ctx.Done():
			return &gateResult{Passed: false, Error: ctx.Err().Error()}
		}
	}

	if !cancelled {
		return &gateResult{Passed: false, Error: "cancel was never triggered (no approval observed before idle)"}
	}
	if obs.succeededOnce("bash") {
		return &gateResult{Passed: false, Error: "cancelled bash tool still reported success after cancel"}
	}
	if obs.answered() {
		return &gateResult{Passed: false, Error: "cancelled session emitted a normal answer after cancel"}
	}

	// Prove the runtime + global SSE survived cancel: a fresh session still
	// completes a full read tool lifecycle (called → success → answer → idle).
	fresh, err := adapter.CreateSession(ctx, runtime.CreateSessionRequest{Title: "cancel-smoke-after"})
	if err != nil {
		return &gateResult{Passed: false, Error: fmt.Sprintf("post-cancel CreateSession: %v", err)}
	}
	if err := adapter.Prompt(ctx, runtime.SessionID(fresh.ID), runtime.PromptRequest{
		Agent: "build",
		Parts: []runtime.PromptPart{runtime.TextPart{Text: "READ the file please"}},
	}); err != nil {
		return &gateResult{Passed: false, Error: fmt.Sprintf("post-cancel Prompt: %v", err)}
	}
	freshObs, err := drainUntilIdle(ctx, ch, state, fresh.ID, "cancel-smoke-after", 30*time.Second)
	if err != nil {
		return &gateResult{Passed: false, Error: fmt.Sprintf("post-cancel session did not terminate cleanly: %v", err)}
	}
	if !freshObs.calledOnce("read") || !freshObs.succeededOnce("read") || !freshObs.answered() {
		return &gateResult{Passed: false, Error: fmt.Sprintf(
			"post-cancel session did not complete normally: called=%v success=%v answered=%v",
			freshObs.called, freshObs.success, freshObs.answered())}
	}

	return &gateResult{Passed: true, Detail: "cancel terminated session mid-approval; bash did not succeed; fresh session completed normally"}
}

// drainSession collects events for a single session until it idles.
func drainSession(t *testing.T, ch <-chan runtime.Event, state *smokeState, sid, label string) *toolObservation {
	t.Helper()
	obs := newToolObservation()
	timeout := time.After(30 * time.Second)
	for !obs.idled {
		select {
		case ev, ok := <-ch:
			if !ok {
				t.Fatalf("event channel closed before idle (%s)", label)
			}
			if ev.RawType == "plugin.added" {
				state.pluginAdded++
			}
			if ev.SessionID != sid {
				continue
			}
			obs.collect(ev)
		case <-timeout:
			t.Fatalf("timed out waiting for idle (%s); called=%v success=%v", label, obs.called, obs.success)
		}
	}
	return obs
}

// drainUntilIdle is the error-returning variant of drainSession: it drains the
// shared global stream for session sid until it idles, returning the
// observation and an error instead of failing the test, so gate helpers can
// fold the failure into their gateResult.
func drainUntilIdle(ctx context.Context, ch <-chan runtime.Event, state *smokeState, sid, label string, timeout time.Duration) (*toolObservation, error) {
	obs := newToolObservation()
	deadline := time.After(timeout)
	for !obs.idled {
		select {
		case ev, ok := <-ch:
			if !ok {
				return obs, fmt.Errorf("event channel closed before idle (%s)", label)
			}
			if ev.RawType == "plugin.added" {
				state.pluginAdded++
			}
			if ev.SessionID != sid {
				continue
			}
			obs.collect(ev)
		case <-deadline:
			return obs, fmt.Errorf("timed out waiting for idle (%s); called=%v success=%v", label, obs.called, obs.success)
		case <-ctx.Done():
			return obs, fmt.Errorf("context done waiting for idle (%s): %w", label, ctx.Err())
		}
	}
	return obs, nil
}
