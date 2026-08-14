package app

import "codea/tui/internal/runtime"

// Codea domain event types consumed directly by the TUI beyond what the
// reasoning processor already handles. These string values are Codea
// semantics, not OpenCode vendor DTOs.
const (
	eventTypeStepFinished runtime.EventType = "step.finished"
	eventTypeSessionError runtime.EventType = "session.error"
	eventTypeRuntimeError runtime.EventType = "runtime.error"

	eventTypeToolCalled  runtime.EventType = "tool.called"
	eventTypeToolSuccess runtime.EventType = "tool.success"
	eventTypeToolFailed  runtime.EventType = "tool.failed"

	eventTypeApprovalRequested runtime.EventType = "approval.requested"
)
