package app

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"codea/tui/internal/opencode"
	"codea/tui/internal/runtime"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

func TestTask25FocusRendersOnlyLatestTurnAndPreservesConversationTruth(t *testing.T) {
	m := NewModel(nil)
	m.viewMode = ViewFocus
	m.messages = []ChatMessage{
		{Role: RoleInfo, Content: "historical system notice", Finished: true},
		{Role: RoleUser, Content: "first question", Finished: true},
		{Role: RoleAssistant, Content: "first answer", Finished: true},
		{Role: RoleUser, Content: "second question", Finished: true},
		{Role: RoleAssistant, Content: "second answer", Finished: true},
		{Role: RoleUser, Content: "third question", Finished: true},
		{Role: RoleAssistant, Content: "third answer", Finished: true},
	}
	start := time.Date(2026, 8, 27, 7, 0, 0, 0, time.UTC)
	m.executionTrace.upsert(ExecutionTraceEntry{
		Category:      TraceWorking,
		Title:         "Working",
		Status:        TraceSuccess,
		InvocationKey: "turn:turn-3:working",
		StartedAt:     start,
		FinishedAt:    start.Add(time.Second),
		TurnID:        "turn-3",
	})
	m.executionTrace.upsert(ExecutionTraceEntry{
		Category:      TraceTool,
		Title:         "read",
		Status:        TraceSuccess,
		InvocationKey: "tool:turn-3-read",
		TurnID:        "turn-3",
	})
	m.executionTrace.upsert(ExecutionTraceEntry{
		Category:      TraceApproval,
		Title:         "write",
		Status:        TraceWaiting,
		InvocationKey: "approval:turn-3",
		TurnID:        "turn-3",
	})
	m.executionTrace.upsert(ExecutionTraceEntry{
		Category:      TraceRuntime,
		Title:         "Runtime",
		Detail:        "verification error",
		Status:        TraceFailed,
		InvocationKey: "runtime:turn-3",
		TurnID:        "turn-3",
	})

	beforeMessages := append([]ChatMessage(nil), m.messages...)
	beforeTrace := m.executionTrace.Entries()
	got := m.renderBody()

	for _, hidden := range []string{"historical system notice", "first question", "first answer", "second question", "second answer"} {
		if strings.Contains(got, hidden) {
			t.Fatalf("focus view leaked historical content %q:\n%s", hidden, got)
		}
	}
	for _, visible := range []string{"third question", "third answer", "Activity", "Approval", "Runtime"} {
		if !strings.Contains(got, visible) {
			t.Fatalf("focus view missing %q:\n%s", visible, got)
		}
	}
	if !reflect.DeepEqual(m.messages, beforeMessages) {
		t.Fatalf("focus rendering mutated messages:\nbefore=%#v\nafter=%#v", beforeMessages, m.messages)
	}
	if !reflect.DeepEqual(m.executionTrace.Entries(), beforeTrace) {
		t.Fatalf("focus rendering mutated execution trace")
	}
}

func TestTask25RealOpenCodeMapperEvidenceReachesExecutionTraceAndTUI(t *testing.T) {
	m := NewModel(nil)
	m.sessionID = runtime.SessionID("s1")
	m.activeTurnID = "turn-current"

	rawEvents := []struct {
		name        string
		raw         string
		metadataKey string
		metadataVal string
	}{
		{
			name: "skill tool execution",
			raw: `{"directory":"/workspace","payload":{"type":"message.part.updated","properties":{"sessionID":"s1","part":{"id":"part-skill","messageID":"m1","sessionID":"s1","type":"tool","tool":"skill","callID":"call-skill-1","state":{"status":"running","input":{"name":"code-review"},"metadata":{}}}}}}`,
			metadataKey: "skill",
			metadataVal: "code-review",
		},
		{
			name: "codea enterprise plugin execution",
			raw: `{"directory":"/workspace","payload":{"type":"message.part.updated","properties":{"sessionID":"s1","part":{"id":"part-plugin","messageID":"m1","sessionID":"s1","type":"tool","tool":"collect_review_context","callID":"call-plugin-1","state":{"status":"completed","input":{"source":"staged"},"output":"ok","metadata":{"codeaPlugin":"codea-enterprise","codeaPluginInvocationID":"call-plugin-1"}}}}}}`,
			metadataKey: "plugin",
			metadataVal: "codea-enterprise",
		},
		{
			name: "structured subtask execution",
			raw: `{"directory":"/workspace","payload":{"type":"message.part.updated","properties":{"sessionID":"s1","part":{"id":"part-subtask","messageID":"m1","sessionID":"s1","type":"subtask","agent":"explore","description":"inspect call chain","prompt":"trace it"}}}}`,
			metadataKey: "subagent",
			metadataVal: "explore",
		},
	}

	for i, tc := range rawEvents {
		t.Run(tc.name, func(t *testing.T) {
			ev, err := opencode.MapEvent([]byte(tc.raw), int64(i+1))
			if err != nil {
				t.Fatalf("MapEvent: %v", err)
			}
			if ev.Metadata[tc.metadataKey] != tc.metadataVal {
				t.Fatalf("Codea Event metadata[%q] = %q, want %q; event=%#v", tc.metadataKey, ev.Metadata[tc.metadataKey], tc.metadataVal, ev)
			}
			if !m.processRuntimeEvent(ev) && len(ev.Metadata) == 0 {
				t.Fatalf("mapped event carried no structured execution evidence")
			}
		})
	}

	got := m.renderExecutionTrace()
	for _, want := range []string{"Skill · code-review", "Plugin · codea-enterprise", "Subagent · explore"} {
		if !strings.Contains(got, want) {
			t.Fatalf("execution trace missing real mapped evidence %q:\n%s", want, got)
		}
	}

	ordinary, err := opencode.MapEvent([]byte(`{"directory":"/workspace","payload":{"type":"message.part.updated","properties":{"sessionID":"s1","part":{"id":"part-read","messageID":"m1","sessionID":"s1","type":"tool","tool":"read","callID":"call-read-1","state":{"status":"running","input":{"filePath":"README.md"},"metadata":{}}}}}}`), 99)
	if err != nil {
		t.Fatalf("map ordinary tool: %v", err)
	}
	for _, key := range []string{"skill", "plugin", "subagent"} {
		if ordinary.Metadata[key] != "" {
			t.Fatalf("ordinary tool fabricated %s evidence: %#v", key, ordinary.Metadata)
		}
	}
}

func TestTask25TransientProfessionalCommandShowsActualTurnAgentWithoutChangingPersistentAgent(t *testing.T) {
	fake := fakeruntime.New()
	m := NewModel(fake)
	m.sessionID = runtime.SessionID("active-session")
	m.currentAgent = "general"
	m.sessionModels[m.sessionID] = runtime.ModelRef{ProviderID: "qwen", ModelID: "qwen3-coder"}
	m.input = "/review OrderService"

	cmd := m.submit()
	if cmd == nil {
		t.Fatal("/review should prompt Runtime")
	}
	_ = cmd()

	prompts := fake.Prompts()
	if len(prompts) != 1 || prompts[0].Request.Agent != "code-reviewer" {
		t.Fatalf("review prompt = %#v, want code-reviewer", prompts)
	}
	if m.currentAgent != "general" {
		t.Fatalf("persistent currentAgent = %q, want general", m.currentAgent)
	}
	if got := m.renderHeader(); !strings.Contains(got, "Agent: code-reviewer") {
		t.Fatalf("current-turn header does not show actual PromptRequest.Agent:\n%s", got)
	}

	m.appendAnswer("review result")
	m.finishStreaming()
	body := m.renderBody()
	if !strings.Contains(body, "● Code Reviewer") || !strings.Contains(body, "qwen3-coder") {
		t.Fatalf("assistant turn identity missing actual Agent/Model:\n%s", body)
	}
	if m.currentAgent != "general" {
		t.Fatalf("professional turn polluted persistent currentAgent: %q", m.currentAgent)
	}

	m.input = "继续解释"
	cmd = m.submit()
	if cmd == nil {
		t.Fatal("natural-language continuation should prompt Runtime")
	}
	_ = cmd()
	prompts = fake.Prompts()
	if len(prompts) != 2 {
		t.Fatalf("Runtime prompts = %d, want 2", len(prompts))
	}
	if prompts[1].Request.Agent != "general" {
		t.Fatalf("next natural-language Agent = %q, want persistent general", prompts[1].Request.Agent)
	}
}
