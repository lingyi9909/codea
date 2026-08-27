package app

import (
	"testing"
	"time"

	"codea/tui/internal/runtime"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

func TestTask23ResumeSessionResetsTransientStateAndKeepsModelIsolation(t *testing.T) {
	m := NewModel(fakeruntime.New())
	oldID := runtime.SessionID("s1")
	newID := runtime.SessionID("s2")
	m.sessionID = oldID
	m.sessionModels[oldID] = runtime.ModelRef{ProviderID: "company", ModelID: "kimi"}
	m.streamBuf.WriteString("stale answer")
	m.reasoningBuf.WriteString("stale reasoning")
	m.reasoningActive = true
	m.reasoningContent = "stale reasoning"
	m.reasoningDuration = 3 * time.Second
	m.reasoningExpanded = true
	m.tools = []ToolActivity{{Name: "Read", CallID: "old-call", Status: ToolRunning}}
	m.modelPicker = modelPickerModel{Visible: true, SessionID: oldID, Items: []runtime.Model{{Ref: runtime.ModelRef{ProviderID: "company", ModelID: "kimi"}}}}
	m.currentAgent = "code-reviewer"

	m.resumeSession(newID, []runtime.Message{{ID: "m1", Role: "user", Content: "rehydrated"}})

	if m.sessionID != newID {
		t.Fatalf("sessionID = %q, want %q", m.sessionID, newID)
	}
	if m.isStreaming || m.streamBuf.Len() != 0 || m.reasoningBuf.Len() != 0 || m.reasoningActive || m.reasoningContent != "" || m.reasoningDuration != 0 || m.reasoningExpanded {
		t.Fatalf("transient streaming/reasoning state leaked across resume")
	}
	if len(m.tools) != 0 {
		t.Fatalf("tool state leaked across resume: %#v", m.tools)
	}
	if m.modelPicker.Visible {
		t.Fatal("model picker leaked across resume")
	}
	if m.currentAgent != "general" {
		t.Fatalf("currentAgent = %q, want general after resume", m.currentAgent)
	}
	if len(m.messages) != 1 || m.messages[0].Content != "rehydrated" {
		t.Fatalf("rehydrated messages = %#v", m.messages)
	}
	if _, ok := m.sessionModels[newID]; ok {
		t.Fatal("new session inherited another session's explicit model")
	}
	if got := m.sessionModels[oldID]; got.ModelID != "kimi" {
		t.Fatalf("old session model selection was lost: %#v", got)
	}
}
