package app

import (
	"fmt"
	"strings"
	"time"
)

var workingSpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

func (m *Model) approvalWaiting() bool {
	if m.activeTurnID == "" {
		return false
	}
	working, ok := m.executionTrace.Entry("turn:" + m.activeTurnID + ":working")
	return ok && working.Status == TraceWaiting
}

func (m *Model) executionStatusText() string {
	if m.approvalWaiting() {
		return "Permission required"
	}
	if m.reasoningActive {
		return "Thinking…"
	}
	entries := m.executionTrace.Entries()
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		if entry.TurnID != "" && m.activeTurnID != "" && entry.TurnID != m.activeTurnID {
			continue
		}
		if entry.Category == TraceTool && entry.Status == TraceRunning {
			return "Running tools…"
		}
	}
	return "Working…"
}

func (m *Model) spinnerGlyph() string {
	if len(workingSpinnerFrames) == 0 {
		return "◌"
	}
	idx := m.spinnerFrame % len(workingSpinnerFrames)
	if idx < 0 {
		idx = 0
	}
	return workingSpinnerFrames[idx]
}

func (m *Model) selectedModelLabel() string {
	if m.sessionID == "" {
		return "runtime-default"
	}
	if ref, ok := m.sessionModels[m.sessionID]; ok {
		provider := strings.TrimSpace(ref.ProviderID)
		model := strings.TrimSpace(ref.ModelID)
		if provider != "" && model != "" {
			return provider + "/" + model
		}
		if model != "" {
			return model
		}
	}
	return "runtime-default"
}

func (m *Model) latestTerminalWorking() (ExecutionTraceEntry, bool) {
	entries := m.executionTrace.Entries()
	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		if entry.Category != TraceWorking {
			continue
		}
		switch entry.Status {
		case TraceSuccess, TraceFailed, TraceDenied, TraceUnverified:
			return entry, true
		}
	}
	return ExecutionTraceEntry{}, false
}

func (m *Model) renderCompletionSummary() string {
	working, ok := m.latestTerminalWorking()
	if !ok {
		return ""
	}
	duration := working.Duration
	if duration <= 0 && !working.StartedAt.IsZero() && !working.FinishedAt.IsZero() && !working.FinishedAt.Before(working.StartedAt) {
		duration = working.FinishedAt.Sub(working.StartedAt)
	}
	prefix := "✓ Completed"
	if working.Status == TraceFailed {
		prefix = "✗ Failed"
	} else if working.Status == TraceDenied {
		prefix = "✗ Denied"
	} else if working.Status == TraceUnverified {
		prefix = "⚠ Unverified"
	} else if working.Status == TraceSuccess && working.TurnID == m.taskExecution.RootTurnID && m.taskExecution.MutationSeen && m.taskExecution.VerifyPassed && m.taskExecution.LastVerificationResult == "pass" {
		prefix = "✓ Verified"
	}
	if duration > 0 {
		prefix += " in " + formatDuration(duration)
	}

	metrics := m.traceActivityMetrics(working.TurnID)
	parts := make([]string, 0, 3)
	if metrics.tools > 0 {
		parts = append(parts, pluralMetric(metrics.tools, "tool call", "tool calls"))
	}
	if metrics.skills > 0 {
		parts = append(parts, pluralMetric(metrics.skills, "skill", "skills"))
	}
	if metrics.plugins > 0 {
		parts = append(parts, pluralMetric(metrics.plugins, "plugin", "plugins"))
	}
	if len(parts) == 0 {
		return prefix
	}
	return prefix + "\n  " + strings.Join(parts, " · ")
}

type traceMetrics struct {
	tools     int
	skills    int
	plugins   int
	subagents int
}

func (m *Model) traceActivityMetrics(turnID string) traceMetrics {
	var metrics traceMetrics
	for _, entry := range m.executionTrace.Entries() {
		if turnID != "" && entry.TurnID != turnID {
			continue
		}
		switch entry.Category {
		case TraceTool:
			metrics.tools++
		case TraceSkill:
			metrics.skills++
		case TracePlugin:
			metrics.plugins++
		case TraceSubagent:
			metrics.subagents++
		}
	}
	return metrics
}

func (m *Model) renderFocusActivitySummary() string {
	turnID := m.activeTurnID
	if turnID == "" {
		if terminal, ok := m.latestTerminalWorking(); ok {
			turnID = terminal.TurnID
		}
	}
	metrics := m.traceActivityMetrics(turnID)
	parts := make([]string, 0, 4)
	if metrics.tools > 0 {
		parts = append(parts, pluralMetric(metrics.tools, "tool call", "tool calls"))
	}
	if metrics.skills > 0 {
		parts = append(parts, pluralMetric(metrics.skills, "skill", "skills"))
	}
	if metrics.plugins > 0 {
		parts = append(parts, pluralMetric(metrics.plugins, "plugin", "plugins"))
	}
	if metrics.subagents > 0 {
		parts = append(parts, pluralMetric(metrics.subagents, "subagent", "subagents"))
	}
	if len(parts) == 0 {
		return ""
	}
	return "Activity · " + strings.Join(parts, " · ")
}

func pluralMetric(count int, singular, plural string) string {
	label := plural
	if count == 1 {
		label = singular
	}
	return fmt.Sprintf("%d %s", count, label)
}

// stableDurationText is kept separate from wall-clock animation so completion
// rendering remains deterministic from trace timestamps.
func stableDurationText(d time.Duration) string {
	if d <= 0 {
		return ""
	}
	return formatDuration(d)
}
