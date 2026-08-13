package app

import (
	"fmt"
	"time"
)

// View renders the current application state. The full three-region layout
// (header / chat / input) is implemented in Step 6; this placeholder exists so
// *Model satisfies tea.Model while the subscription wiring lands first.
func (m *Model) View() string {
	return ""
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
