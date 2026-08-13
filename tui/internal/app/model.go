// Package app hosts the Bubble Tea top-level Model for the Codea TUI.
//
// Dependency rule: this package depends only on the Codea runtime domain, the
// reasoning processor, the theme package, and the Bubble Tea stack. It must
// never import the OpenCode vendor layer or the supervisor — those are wired
// together in cmd/codea.
package app

import (
	"time"

	"codea/tui/internal/reasoning"
	"codea/tui/internal/runtime"
)

// Role identifies the author of a chat message.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleInfo      Role = "info"
)

// ChatMessage is one conversation turn. Tool activity is tracked separately
// (see ToolActivity), not folded into Content.
type ChatMessage struct {
	Role     Role
	Content  string
	Finished bool
}

// ToolStatus is the lifecycle state of a tool invocation for read-only display.
type ToolStatus string

const (
	ToolRunning ToolStatus = "running"
	ToolSuccess ToolStatus = "success"
	ToolFailed  ToolStatus = "failed"
)

// ToolActivity is a read-only view of a tool call in progress or completed.
type ToolActivity struct {
	Name   string
	CallID string
	Status ToolStatus
}

// Model is the Bubble Tea application state. All mutation happens inside
// Update (single goroutine), so no mutex is required.
type Model struct {
	currentPage Page
	width       int
	height      int

	runtimeClient runtime.AgentRuntime
	runtimeStatus runtime.RuntimeStatus

	keys KeyMap

	messages []ChatMessage
	input    string

	isStreaming bool
	sessionID   runtime.SessionID
	msgCounter  int

	proc              *reasoning.Processor
	reasoningActive   bool
	reasoningContent  string
	reasoningDuration time.Duration
	reasoningExpanded bool

	tools []ToolActivity

	eventCh <-chan runtime.Event
}

// NewModel constructs the Task 7 application model around the given Runtime.
func NewModel(client runtime.AgentRuntime) *Model {
	return &Model{
		currentPage:   PageChat,
		runtimeStatus: runtime.RuntimeStopped,
		runtimeClient: client,
		keys:          DefaultKeyMap(),
		messages:      make([]ChatMessage, 0),
		proc:          reasoning.NewProcessor(),
		tools:         make([]ToolActivity, 0),
	}
}
