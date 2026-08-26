package app

import (
	"fmt"
	"strings"
	"time"

	"codea/tui/internal/runtime"
	"codea/tui/internal/theme"

	"github.com/charmbracelet/lipgloss"
)

// View returns the cached render, rebuilding only when dirty. The ~50ms tick
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
// status+input). It is only invoked on a dirty render, not on every event.
// Modal states replace normal input so keystrokes cannot leak into chat.
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

// renderCommandPalette renders a compact filtered command list above the input.
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

// renderBody assembles the chat area: messages, then the reasoning block, then
// the tool timeline.
func (m *Model) renderBody() string {
	var lines []string
	for _, msg := range m.messages {
		lines = append(lines, m.renderMessage(msg))
	}
	if r := m.renderReasoning(); r != "" {
		lines = append(lines, theme.MutedStyle().Render(r))
	}
	if tools := m.renderTools(); tools != "" {
		lines = append(lines, tools)
	}
	return strings.Join(lines, "\n")
}

// renderMessage renders a single chat turn. User turns get a "User >" prefix;
// assistant and info turns are rendered as plain content.
func (m *Model) renderMessage(msg ChatMessage) string {
	if msg.Role == RoleUser {
		return "User > " + msg.Content
	}
	return msg.Content
}

// renderTerminalTooSmall renders the minimum-size notice shown when the
// terminal is below the 70x20 floor.
func renderTerminalTooSmall(w, h int) string {
	return fmt.Sprintf("Terminal too small\nMinimum: 70x20 (current: %dx%d)", w, h)
}

// renderHeader returns the one-line header: app name plus runtime status.
func (m *Model) renderHeader() string {
	return fmt.Sprintf("Codea  %s %s", statusDot(m.runtimeStatus), statusLabel(m.runtimeStatus))
}

// renderStatusLine returns the task status line: "Ready" when idle, a working
// indicator while streaming.
func (m *Model) renderStatusLine() string {
	if m.isStreaming {
		return "◌ Working"
	}
	return "Ready"
}

// renderTools returns the tool activity timeline, or empty when no tools ran.
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

// renderInput returns the input line prompt plus the current buffer.
func (m *Model) renderInput() string {
	return "> " + m.input
}

// renderFooter returns the one-line shortcut hint.
func (m *Model) renderFooter() string {
	return "/ commands · enter submit · alt+enter newline · ctrl+t thinking · ctrl+s sessions · ctrl+k skills · ctrl+l clear · ctrl+c quit"
}

// formatDuration renders a duration compactly for the reasoning summary.
func formatDuration(d time.Duration) string {
	return fmt.Sprintf("%.1fs", d.Seconds())
}

// renderReasoning returns the plain-text reasoning block for the current
// state: active content while streaming, the full content when expanded, a
// "Spent Xs thinking" summary when collapsed, or empty when there is no
// reasoning to show.
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

// statusLabel maps a runtime lifecycle status to its display label.
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

// statusDot returns the colored-status glyph: filled for healthy/crashed,
// hollow otherwise.
func statusDot(s runtime.RuntimeStatus) string {
	switch s {
	case runtime.RuntimeHealthy, runtime.RuntimeCrashed:
		return "●"
	default:
		return "○"
	}
}

// toolSymbol returns the timeline glyph for a tool status.
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
