package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"codea/tui/internal/runtime"
	fakeruntime "codea/tui/tests/fixtures/fake-runtime"
)

func TestTask24ProfessionalCommandsReachExactRuntimeAgents(t *testing.T) {
	tests := []struct {
		command string
		agent   string
		want    string
	}{
		{command: "/review OrderService  --changed-only", agent: "code-reviewer", want: "OrderService  --changed-only"},
		{command: "/test OrderServiceTest --repair", agent: "unit-test-generator", want: "OrderServiceTest --repair"},
		{command: "/api-doc /orders/{id}", agent: "api-documentation", want: "/orders/{id}"},
		{command: "/debug NullPointerException at OrderService:42", agent: "debug", want: "NullPointerException at OrderService:42"},
	}

	for _, tt := range tests {
		t.Run(tt.agent, func(t *testing.T) {
			fake := fakeruntime.New()
			m := NewModel(fake)
			m.sessionID = runtime.SessionID("active-session")
			m.input = tt.command

			cmd := m.submit()
			if cmd == nil {
				t.Fatalf("%s returned no Runtime prompt command", tt.command)
			}
			_ = cmd()

			prompts := fake.Prompts()
			if len(prompts) != 1 {
				t.Fatalf("Runtime prompts = %d, want 1", len(prompts))
			}
			if prompts[0].Request.Agent != tt.agent {
				t.Fatalf("agent = %q, want %q", prompts[0].Request.Agent, tt.agent)
			}
			if prompts[0].Request.Agent == "general" {
				t.Fatal("professional command must never fall back to general")
			}
			part, ok := prompts[0].Request.Parts[0].(runtime.TextPart)
			if !ok {
				t.Fatalf("part type = %T, want runtime.TextPart", prompts[0].Request.Parts[0])
			}
			if part.Text != tt.want {
				t.Fatalf("prompt = %q, want %q", part.Text, tt.want)
			}
			if m.currentAgent != tt.agent {
				t.Fatalf("currentAgent = %q, want %q", m.currentAgent, tt.agent)
			}
		})
	}
}

func TestTask24NaturalLanguageContinuesWithSelectedAgent(t *testing.T) {
	fake := fakeruntime.New()
	m := NewModel(fake)
	m.sessionID = runtime.SessionID("active-session")
	m.currentAgent = "code-reviewer"
	m.input = "再看一下这个方法的并发问题"

	cmd := m.submit()
	if cmd == nil {
		t.Fatal("natural-language continuation should prompt Runtime")
	}
	_ = cmd()

	prompts := fake.Prompts()
	if len(prompts) != 1 {
		t.Fatalf("Runtime prompts = %d, want 1", len(prompts))
	}
	if prompts[0].Request.Agent != "code-reviewer" {
		t.Fatalf("agent = %q, want selected code-reviewer", prompts[0].Request.Agent)
	}
}

func TestTask24AgentWorkspaceSelectsFromRuntimeList(t *testing.T) {
	m := NewModel(fakeruntime.New())
	_, _ = m.Update(listAgentsResultMsg{agents: []runtime.Agent{
		{Name: "general", Mode: "primary"},
		{Name: "code-reviewer", Mode: "enterprise-controlled"},
	}})

	m.handleKey(tea.KeyMsg{Type: tea.KeyDown})
	m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})

	if m.currentAgent != "code-reviewer" {
		t.Fatalf("currentAgent = %q, want runtime-selected code-reviewer", m.currentAgent)
	}
}

func TestTask24AgentWorkspaceEscapeKeepsCurrentAgent(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.currentAgent = "unit-test-generator"
	_, _ = m.Update(listAgentsResultMsg{agents: []runtime.Agent{
		{Name: "general", Mode: "primary"},
		{Name: "unit-test-generator", Mode: "enterprise-controlled"},
		{Name: "api-documentation", Mode: "enterprise-controlled"},
	}})

	m.handleKey(tea.KeyMsg{Type: tea.KeyDown})
	m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})

	if m.currentAgent != "unit-test-generator" {
		t.Fatalf("currentAgent = %q, Esc must keep unit-test-generator", m.currentAgent)
	}
}

func TestTask24AgentSwitchIsBlockedWhileStreaming(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.isStreaming = true
	m.input = "/agents"

	if cmd := m.submit(); cmd != nil {
		t.Fatal("/agents must not call Runtime while a response is in flight")
	}
	if len(m.messages) == 0 || m.messages[len(m.messages)-1].Role != RoleInfo {
		t.Fatalf("messages = %#v, want explicit blocked-switch notice", m.messages)
	}
}

func TestTask24SessionResumeResetsProfessionalAgent(t *testing.T) {
	m := NewModel(fakeruntime.New())
	m.currentAgent = "debug"
	m.resumeSession(runtime.SessionID("session-b"), nil)
	if m.currentAgent != "general" {
		t.Fatalf("currentAgent = %q, want general after session resume", m.currentAgent)
	}
}
