package app

import (
	"fmt"
	"strings"
	"time"

	"codea/tui/internal/runtime"
	"codea/tui/internal/theme"

	"github.com/charmbracelet/lipgloss"
)

// View returns the cached render, rebuilding only when dirty. The bounded tick
// flushes buffered streaming deltas and marks the model dirty; high-frequency
// answer/reasoning deltas are buffered without marking dirty, so a token burst
// does not trigger a full re-render per token.
func (m *Model) View() string {
	if !m.dirty {
		return m.rendered
	}
	m.dirty = false
	if m.width < 70 || m.height < 20 {
		m.rendered = renderTerminalTooSmall(m.width, m.height)
		return m.rendered
	}
	m.rendered = m.renderView()
	return m.rendered
}

// renderView assembles the full three-region layout (header / chat /
// status+input). Modal states replace normal input so keystrokes cannot leak.
func (m *Model) renderView() string {
	width := m.width

	header := lipgloss.NewStyle().Foreground(theme.Primary).Bold(true).Render(m.renderHeader())
	rule := lipgloss.NewStyle().Foreground(theme.Border).Render(strings.Repeat("─", width))

	var b strings.Builder
	b.WriteString(header)
	b.WriteString("\n")
	b.WriteString(rule)
	b.WriteString("\n")

	if m.permission.Visible() {
		// Keep the semantic conversation/trace visible behind the blocking
		// approval so the user can see exactly what is waiting and why.
		if body := m.renderBody(); body != "" {
			b.WriteString(body)
			b.WriteString("\n\n")
		}
		b.WriteString(m.permission.View())
		if m.approvalErr != "" {
			b.WriteString("\n\n")
			b.WriteString(theme.ErrorStyle().Render("Approval error: " + m.approvalErr))
		}
		return b.String()
	}

	if m.feedback.Visible() {
		b.WriteString(m.renderBody())
		b.WriteString("\n\n")
		b.WriteString(theme.AccentStyle().Render(m.feedback.View()))
		return b.String()
	}

	if m.agentPicker.Visible {
		b.WriteString(m.renderBody())
		b.WriteString("\n\n")
		b.WriteString(theme.AccentStyle().Render(m.agentPickerView()))
		return b.String()
	}

	if m.modelPicker.Visible {
		b.WriteString(m.renderBody())
		b.WriteString("\n\n")
		b.WriteString(theme.AccentStyle().Render(m.modelPickerView()))
		return b.String()
	}

	if m.sessionPanel.Visible {
		b.WriteString(m.sessionPanel.View())
		if m.sessionNotice != "" {
			b.WriteString("\n")
			b.WriteString(theme.MutedStyle().Render(m.sessionNotice))
		}
		return b.String()
	}

	if m.currentPage == PageSkills {
		b.WriteString(m.skillPanel.View())
		if m.skillNotice != "" {
			b.WriteString("\n")
			b.WriteString(theme.MutedStyle().Render(m.skillNotice))
		}
		return b.String()
	}

	status := theme.MutedStyle().Render(m.renderStatusLine())
	input := theme.AccentStyle().Render(m.renderInput())
	footer := theme.MutedStyle().Render(m.renderFooter())

	b.WriteString(m.renderBody())
	b.WriteString("\n\n")
	if m.commandPalette.Visible {
		b.WriteString(m.renderCommandPalette())
		b.WriteString("\n")
	}
	b.WriteString(status)
	b.WriteString("\n")
	b.WriteString(rule)
	b.WriteString("\n")
	b.WriteString(input)
	b.WriteString("\n")
	b.WriteString(footer)
	return b.String()
}

func (m *Model) renderCommandPalette() string {
	if !m.commandPalette.Visible {
		return ""
	}
	if len(m.commandPalette.Items) == 0 {
		return theme.MutedStyle().Render("Commands  no matches")
	}
	lines := []string{theme.MutedStyle().Render("Commands")}
	for i, def := range m.commandPalette.Items {
		prefix := "  "
		if i == m.commandPalette.Cursor {
			prefix = "› "
		}
		line := prefix + "/" + def.Name
		if strings.TrimSpace(def.Description) != "" {
			line += "  " + def.Description
		}
		if i == m.commandPalette.Cursor {
			line = theme.AccentStyle().Render(line)
		} else {
			line = theme.MutedStyle().Render(line)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func (m *Model) renderBody() string {
	var lines []string
	latestUser, latestAssistant := m.latestConversationTurnIndexes()

	if m.viewMode == ViewFocus {
		// Focus is a presentation-only projection of the latest turn. It never
		// mutates history or semantic trace truth.
		if latestUser >= 0 {
			if rendered := m.renderMessage(m.messages[latestUser]); rendered != "" {
				lines = append(lines, rendered)
			}
		}
	} else {
		for i, msg := range m.messages {
			if i == latestAssistant {
				continue
			}
			if rendered := m.renderMessage(msg); rendered != "" {
				lines = append(lines, rendered)
			}
		}
	}

	if m.viewMode != ViewFocus {
		if r := m.renderReasoning(); r != "" {
			lines = append(lines, theme.MutedStyle().Render(r))
		}
	}
	if trace := m.renderExecutionTrace(); trace != "" {
		lines = append(lines, trace)
	} else if tools := m.renderTools(); tools != "" && m.viewMode != ViewFocus {
		// Legacy fallback for tests/older states that predate a semantic trace.
		lines = append(lines, tools)
	}
	if m.viewMode == ViewFocus {
		if summary := m.renderFocusActivitySummary(); summary != "" {
			lines = append(lines, theme.MutedStyle().Render(summary))
		}
	}

	if latestAssistant >= 0 {
		if rendered := m.renderMessage(m.messages[latestAssistant]); rendered != "" {
			lines = append(lines, rendered)
		}
	}
	// Focus already has one compact activity summary; do not append the normal
	// completion summary and duplicate activity beneath the latest answer.
	if !m.isStreaming && m.viewMode != ViewFocus {
		if summary := m.renderCompletionSummary(); summary != "" {
			lines = append(lines, theme.MutedStyle().Render(summary))
		}
	}
	return strings.Join(lines, "\n")
}

func (m *Model) latestConversationTurnIndexes() (int, int) {
	latestUser := -1
	for i := len(m.messages) - 1; i >= 0; i-- {
		if m.messages[i].Role == RoleUser {
			latestUser = i
			break
		}
	}
	latestAssistant := -1
	start := len(m.messages) - 1
	for i := start; i >= 0; i-- {
		if m.messages[i].Role != RoleAssistant {
			continue
		}
		if latestUser < 0 || i > latestUser {
			latestAssistant = i
			break
		}
	}
	return latestUser, latestAssistant
}

func (m *Model) renderExecutionTrace() string {
	entries := m.executionTrace.Entries()
	if len(entries) == 0 {
		return ""
	}
	turnID := m.currentTraceTurnID()
	lines := make([]string, 0, len(entries))
	for _, entry := range entries {
		if m.viewMode != ViewVerbose && turnID != "" && entry.TurnID != "" && entry.TurnID != turnID {
			continue
		}
		if !m.traceEntryVisible(entry) {
			continue
		}
		label := traceCategoryLabel(entry.Category)
		if entry.Category != TraceWorking && strings.TrimSpace(entry.Title) != "" {
			label += " · " + entry.Title
		}
		line := label
		if entry.Category == TraceTool && strings.TrimSpace(entry.Detail) != "" && m.viewMode != ViewFocus {
			line += "  " + strings.TrimSpace(entry.Detail)
		} else if m.viewMode == ViewVerbose && strings.TrimSpace(entry.Detail) != "" {
			line += "  " + strings.TrimSpace(entry.Detail)
		}
		line += fmt.Sprintf("  [%s]", entry.Status)
		if entry.Duration > 0 {
			line += "  " + formatDuration(entry.Duration)
		}
		if entry.Category == TraceApproval || entry.Category == TraceRuntime || entry.Status == TraceFailed || entry.Status == TraceDenied {
			line = theme.AccentStyle().Render(line)
		} else {
			line = theme.MutedStyle().Render(line)
		}
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}

func (m *Model) currentTraceTurnID() string {
	if m.activeTurnID != "" {
		return m.activeTurnID
	}
	if terminal, ok := m.latestTerminalWorking(); ok {
		return terminal.TurnID
	}
	return ""
}

func (m *Model) traceEntryVisible(entry ExecutionTraceEntry) bool {
	switch m.viewMode {
	case ViewVerbose:
		return true
	case ViewFocus:
		return entry.Category == TraceApproval || entry.Category == TraceRuntime || entry.Status == TraceFailed || entry.Status == TraceDenied
	default:
		// Normal is balanced: keep current execution structure with safe summaries.
		return true
	}
}

func traceCategoryLabel(category TraceCategory) string {
	switch category {
	case TraceUser:
		return "User"
	case TraceWorking:
		return "Working"
	case TraceAgent:
		return "Agent"
	case TraceTool:
		return "Tool"
	case TraceSkill:
		return "Skill"
	case TracePlugin:
		return "Plugin"
	case TraceApproval:
		return "Approval"
	case TraceAssistant:
		return "Assistant"
	case TraceSubagent:
		return "Subagent"
	case TraceRuntime:
		return "Runtime"
	default:
		return "Trace"
	}
}

func (m *Model) renderMessage(msg ChatMessage) string {
	switch msg.Role {
	case RoleUser:
		return "❯ " + msg.Content
	case RoleAssistant:
		if strings.TrimSpace(msg.Content) == "" {
			return ""
		}
		identity := assistantAgentLabel(msg.Agent)
		if model := strings.TrimSpace(msg.Model); model != "" {
			identity += " · " + model
		}
		return "● " + identity + "\n  " + msg.Content
	case RoleInfo:
		return "System · " + msg.Content
	default:
		return msg.Content
	}
}

func assistantAgentLabel(agent string) string {
	switch strings.TrimSpace(agent) {
	case "", "general":
		return "Codea"
	case "code-reviewer":
		return "Code Reviewer"
	case "unit-test-generator":
		return "Unit Test Generator"
	case "api-documentation":
		return "API Documentation"
	case "debug":
		return "Debug"
	default:
		return strings.TrimSpace(agent)
	}
}

func renderTerminalTooSmall(w, h int) string {
	return fmt.Sprintf("Terminal too small\nMinimum: 70x20 (current: %dx%d)", w, h)
}

func (m *Model) renderHeader() string {
	agent := m.currentTurnAgent()
	return fmt.Sprintf("Codea  %s %s  ·  Agent: %s  ·  Model: %s  ·  View: %s", statusDot(m.runtimeStatus), statusLabel(m.runtimeStatus), agent, m.selectedModelLabel(), m.viewMode)
}

func (m *Model) currentTurnAgent() string {
	if turnID := m.currentTraceTurnID(); turnID != "" {
		if entry, ok := m.executionTrace.Entry("turn:" + turnID + ":agent"); ok {
			if agent := strings.TrimSpace(entry.Title); agent != "" {
				return agent
			}
	}
	for i := len(m.messages) - 1; i >= 0; i-- {
		if m.messages[i].Role == RoleAssistant {
			if agent := strings.TrimSpace(m.messages[i].Agent); agent != "" {
				return agent
			}
		}
	}
	agent := strings.TrimSpace(m.currentAgent)
	if agent == "" {
		return "general"
	}
	return agent
}

func (m *Model) renderStatusLine() string {
	if m.isStreaming {
		if m.approvalWaiting() {
			return "⚠ Permission required"
		}
		return m.spinnerGlyph() + " " + m.executionStatusText()
	}
	return "Ready"
}

func (m *Model) renderTools() string {
	if len(m.tools) == 0 {
		return ""
	}
	lines := make([]string, 0, len(m.tools))
	for _, t := range m.tools {
		lines = append(lines, fmt.Sprintf("  %s %s", toolSymbol(t.Status), t.Name))
	}
	return strings.Join(lines, "\n")
}

func (m *Model) renderInput() string {
	return "> " + m.input
}

func (m *Model) renderFooter() string {
	return "/ commands · enter submit · alt+enter newline · ctrl+t thinking · ctrl+s sessions · ctrl+k skills · ctrl+l clear · ctrl+c quit"
}

func formatDuration(d time.Duration) string {
	return fmt.Sprintf("%.1fs", d.Seconds())
}

func (m *Model) renderReasoning() string {
	if m.reasoningActive {
		return m.reasoningContent
	}
	if m.reasoningContent == "" {
		return ""
	}
	if m.reasoningExpanded {
		return m.reasoningContent
	}
	return fmt.Sprintf("✓ Spent %s thinking", formatDuration(m.reasoningDuration))
}

func statusLabel(s runtime.RuntimeStatus) string {
	switch s {
	case runtime.RuntimeHealthy:
		return "Ready"
	case runtime.RuntimeStopped:
		return "Stopped"
	case runtime.RuntimeStarting:
		return "Starting"
	case runtime.RuntimeStopping:
		return "Stopping"
	case runtime.RuntimeCrashed:
		return "Crashed"
	default:
		return string(s)
	}
}

func statusDot(s runtime.RuntimeStatus) string {
	switch s {
	case runtime.RuntimeHealthy, runtime.RuntimeCrashed:
		return "●"
	default:
		return "○"
	}
}

func toolSymbol(s ToolStatus) string {
	switch s {
	case ToolSuccess:
		return "✓"
	case ToolFailed:
		return "✗"
	default:
		return "◌"
	}
}