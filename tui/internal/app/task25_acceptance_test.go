package app

import (
	"strings"
	"testing"
	"time"

	"codea/tui/internal/runtime"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

func TestTask25SpinnerAnimatesWhileRunningAndStopsForApproval(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.sessionID = runtime.SessionID("session-a")
	m.input = "inspect"
	_ = m.submit()

	first := m.renderStatusLine()
	_, _ = m.Update(tickMsg{t: time.Now()})
	second := m.renderStatusLine()
	if first == second {
		t.Fatalf("running spinner did not advance: %q", first)
	}
	if !strings.Contains(second, "Working") {
		t.Fatalf("running status = %q, want truthful Working fallback", second)
	}

	m.processRuntimeEvent(runtime.Event{
		Type:      eventTypeApprovalRequested,
		SessionID: "session-a",
		Approval:  &runtime.ApprovalRequest{ID: "approval-1", Permission: "bash", Command: "go test ./..."},
	})
	waiting := m.renderStatusLine()
	_, _ = m.Update(tickMsg{t: time.Now()})
	if got := m.renderStatusLine(); got != waiting {
		t.Fatalf("approval waiting indicator animated: before=%q after=%q", waiting, got)
	}
	if !strings.Contains(waiting, "Permission required") {
		t.Fatalf("approval status = %q, want explicit Permission required", waiting)
	}
}

func TestTask25KnownExecutionStageUsesTruthfulStatusText(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.sessionID = runtime.SessionID("session-a")
	m.input = "inspect"
	_ = m.submit()

	m.reasoningActive = true
	if got := m.executionStatusText(); got != "Thinking…" {
		t.Fatalf("reasoning status = %q, want Thinking…", got)
	}
	m.reasoningActive = false
	m.processRuntimeEvent(runtime.Event{Type: eventTypeToolCalled, SessionID: "session-a", Tool: &runtime.ToolEvent{Name: "read", CallID: "call-1"}})
	if got := m.executionStatusText(); got != "Running tools…" {
		t.Fatalf("tool status = %q, want Running tools…", got)
	}
}

func TestTask25ConversationRolesAreVisuallyUnambiguous(t *testing.T) {
	m := NewModel(fakeruntime.New())
	if got := m.renderMessage(ChatMessage{Role: RoleUser, Content: "hello"}); !strings.HasPrefix(got, "❯ ") {
		t.Fatalf("user rendering = %q, want ❯ prefix", got)
	}
	if got := m.renderMessage(ChatMessage{Role: RoleAssistant, Content: "answer"}); !strings.HasPrefix(got, "● Codea") {
		t.Fatalf("assistant rendering = %q, want Codea identity", got)
	}
	if got := m.renderMessage(ChatMessage{Role: RoleInfo, Content: "notice"}); !strings.HasPrefix(got, "System · ") {
		t.Fatalf("system rendering = %q, want System identity", got)
	}
}

func TestTask25SelectedModelIsVisibleInConversationContext(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.sessionID = runtime.SessionID("session-a")
	m.sessionModels[m.sessionID] = runtime.ModelRef{ProviderID: "private", ModelID: "qwen3-coder"}
	got := m.renderHeader()
	if !strings.Contains(got, "qwen3-coder") || !strings.Contains(got, "private") {
		t.Fatalf("header = %q, want selected provider/model", got)
	}
}

func TestTask25CompletionSummaryIsDerivedFromTrace(t *testing.T) {
	m := NewModel(fakeruntime.New())
	start := time.Date(2026, 8, 27, 5, 0, 0, 0, time.UTC)
	finish := start.Add(10 * time.Second)
	m.executionTrace.upsert(ExecutionTraceEntry{Category: TraceWorking, Title: "Working", Status: TraceSuccess, InvocationKey: "turn:msg-1:working", StartedAt: start, FinishedAt: finish, Duration: 10 * time.Second, TurnID: "msg-1"})
	m.executionTrace.upsert(ExecutionTraceEntry{Category: TraceTool, Title: "read", Status: TraceSuccess, InvocationKey: "tool:1", StartedAt: start, FinishedAt: finish, TurnID: "msg-1"})
	m.executionTrace.upsert(ExecutionTraceEntry{Category: TraceTool, Title: "grep", Status: TraceSuccess, InvocationKey: "tool:2", StartedAt: start, FinishedAt: finish, TurnID: "msg-1"})
	m.executionTrace.upsert(ExecutionTraceEntry{Category: TraceSkill, Title: "code-review", Status: TraceSuccess, InvocationKey: "skill:1", StartedAt: start, FinishedAt: finish, TurnID: "msg-1"})

	got := m.renderCompletionSummary()
	for _, want := range []string{"Completed in 10.0s", "2 tool calls", "1 skill"} {
		if !strings.Contains(got, want) {
			t.Fatalf("completion summary = %q, missing %q", got, want)
		}
	}
}

func TestTask25VerboseTraceNeverRendersSensitiveRawContent(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.width, m.height = 120, 40
	m.sessionID = runtime.SessionID("session-a")
	m.input = "inspect"
	_ = m.submit()
	m.processRuntimeEvent(runtime.Event{
		Type:           eventTypeToolCalled,
		SessionID:      "session-a",
		Content:        "api_key=supersecret123",
		RawSensitivity: runtime.SensitivitySensitive,
		Metadata:       map[string]string{"target": "README.md"},
		Tool:           &runtime.ToolEvent{Name: "read", CallID: "call-1"},
	})
	m.viewMode = ViewVerbose
	m.markDirty()
	view := m.View()
	if strings.Contains(view, "supersecret123") || strings.Contains(view, "api_key=") {
		t.Fatalf("verbose trace leaked sensitive content:\n%s", view)
	}
	if !strings.Contains(view, "README.md") {
		t.Fatalf("verbose trace should retain safe structured target:\n%s", view)
	}
}

func TestTask25StructuredExecutionMetadataCreatesSkillPluginAndSubagent(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.sessionID = runtime.SessionID("session-a")
	m.processRuntimeEvent(runtime.Event{
		Type:      eventTypeToolCalled,
		SessionID: "session-a",
		Metadata: map[string]string{
			"skill":                 "code-review",
			"skillInvocationID":     "skill-1",
			"plugin":                "codea-enterprise",
			"pluginInvocationID":    "plugin-1",
			"subagent":              "review-worker",
			"subagentInvocationID":  "subagent-1",
		},
		Tool: &runtime.ToolEvent{Name: "read", CallID: "call-1"},
	})

	for key, category := range map[string]TraceCategory{
		"skill:skill-1":       TraceSkill,
		"plugin:plugin-1":     TracePlugin,
		"subagent:subagent-1": TraceSubagent,
	} {
		entry, ok := m.executionTrace.Entry(key)
		if !ok || entry.Category != category {
			t.Fatalf("trace %s = %#v, ok=%v, want category %s", key, entry, ok, category)
		}
	}
}

func TestTask25ApprovalResolutionUpdatesTraceWithoutGuessing(t *testing.T) {
	for _, tc := range []struct {
		name     string
		decision runtime.ApprovalDecision
		want     TraceStatus
	}{
		{name: "allow", decision: runtime.ApprovalOnce, want: TraceSuccess},
		{name: "reject", decision: runtime.ApprovalReject, want: TraceDenied},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := NewModel(fakeruntime.New())
			m.sessionID = runtime.SessionID("session-a")
			m.input = "fix"
			_ = m.submit()
			m.processRuntimeEvent(runtime.Event{Type: eventTypeApprovalRequested, SessionID: "session-a", Approval: &runtime.ApprovalRequest{ID: "approval-1", Permission: "bash"}})
			m.pendingApprovalDecision = tc.decision
			_, _ = m.Update(approvalResultMsg{approvalID: runtime.ApprovalID("approval-1")})

			approval, _ := m.executionTrace.Entry("approval:approval-1")
			if approval.Status != tc.want {
				t.Fatalf("approval status = %q, want %q", approval.Status, tc.want)
			}
			working, _ := m.executionTrace.Entry("turn:msg-0:working")
			if working.Status != TraceRunning {
				t.Fatalf("working after approval resolution = %q, want running until terminal runtime evidence", working.Status)
			}
		})
	}
}

func TestTask25SessionResumeDoesNotLeakExecutionTrace(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.sessionID = runtime.SessionID("session-a")
	m.input = "inspect"
	_ = m.submit()
	m.processRuntimeEvent(runtime.Event{Type: eventTypeToolCalled, SessionID: "session-a", Tool: &runtime.ToolEvent{Name: "read", CallID: "call-1"}})
	if len(m.executionTrace.Entries()) == 0 {
		t.Fatal("setup: trace is empty")
	}

	m.resumeSession(runtime.SessionID("session-b"), nil)
	if got := len(m.executionTrace.Entries()); got != 0 {
		t.Fatalf("trace entries after resume = %d, want 0", got)
	}
	if m.activeTurnID != "" || m.activeApprovalTraceKey != "" {
		t.Fatalf("active trace state leaked after resume: turn=%q approval=%q", m.activeTurnID, m.activeApprovalTraceKey)
	}
}
