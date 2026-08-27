package app

import (
	"fmt"
	"strings"

	"codea/tui/internal/runtime"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// agentPickerModel is Codea-owned presentation state over runtime.Agent. Agent
// choice is an Application/session preference and deliberately does not add a
// SetAgent method to AgentRuntime.
type agentPickerModel struct {
	Visible bool
	Items   []runtime.Agent
	Cursor  int
}

func (p *agentPickerModel) Open(items []runtime.Agent, current string) {
	p.Visible = len(items) > 0
	p.Items = append([]runtime.Agent(nil), items...)
	p.Cursor = 0
	current = strings.TrimSpace(current)
	for i, item := range p.Items {
		if item.Name == current {
			p.Cursor = i
			break
		}
	}
}

func (p *agentPickerModel) Close() {
	p.Visible = false
	p.Items = nil
	p.Cursor = 0
}

func (p *agentPickerModel) MoveUp() {
	if p.Cursor > 0 {
		p.Cursor--
	}
}

func (p *agentPickerModel) MoveDown() {
	if p.Cursor+1 < len(p.Items) {
		p.Cursor++
	}
}

func (p *agentPickerModel) Selected() (runtime.Agent, bool) {
	if !p.Visible || p.Cursor < 0 || p.Cursor >= len(p.Items) {
		return runtime.Agent{}, false
	}
	return p.Items[p.Cursor], true
}

// handleProfessionalWorkspaceMessage runs before the legacy Task 22 message
// switch. It upgrades /agents from a read-only list to a real picker while
// keeping all values Codea-owned. Returning false for session-history results
// lets the existing resume flow continue after closing stale picker state.
func (m *Model) handleProfessionalWorkspaceMessage(msg tea.Msg) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case listAgentsResultMsg:
		if msg.err != nil {
			m.appendInfo("Failed to load agents: " + msg.err.Error())
			return true, nil
		}
		if len(msg.agents) == 0 {
			m.agentPicker.Close()
			m.appendInfo("No runtime agents are available.")
			return true, nil
		}
		m.agentPicker.Open(msg.agents, m.currentAgent)
		m.markDirty()
		return true, nil

	case loadHistoryResultMsg:
		if m.pendingResumeID != "" && msg.sessionID == m.pendingResumeID {
			m.agentPicker.Close()
		}
		return false, nil

	case tea.KeyMsg:
		if m.agentPicker.Visible {
			return true, m.handleAgentPickerKey(msg)
		}
	}
	return false, nil
}

func (m *Model) handleAgentPickerKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.Up):
		m.agentPicker.MoveUp()
	case key.Matches(msg, m.keys.Down):
		m.agentPicker.MoveDown()
	case key.Matches(msg, m.keys.Esc):
		m.agentPicker.Close()
	case key.Matches(msg, m.keys.Submit):
		selected, ok := m.agentPicker.Selected()
		m.agentPicker.Close()
		if !ok || strings.TrimSpace(selected.Name) == "" {
			m.appendInfo("Agent selection expired.")
			return nil
		}
		m.currentAgent = selected.Name
		m.appendInfo("Agent selected for this session: " + selected.Name)
	}
	m.markDirty()
	return nil
}

func (m *Model) agentPickerView() string {
	if !m.agentPicker.Visible {
		return ""
	}
	var b strings.Builder
	b.WriteString("Agents (runtime)\n")
	for i, agent := range m.agentPicker.Items {
		prefix := "  "
		if i == m.agentPicker.Cursor {
			prefix = "> "
		}
		mode := ""
		if strings.TrimSpace(agent.Mode) != "" {
			mode = " (" + agent.Mode + ")"
		}
		fmt.Fprintf(&b, "%s%s%s\n", prefix, agent.Name, mode)
	}
	b.WriteString("↑/↓ select · Enter apply · Esc close")
	return b.String()
}
