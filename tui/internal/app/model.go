// Package app hosts the Bubble Tea top-level Model for the Codea TUI.
//
// Dependency rule: this package depends only on Codea-owned application/domain
// packages and the Bubble Tea stack. It must never import the OpenCode vendor
// layer or the supervisor — those are wired together in cmd/codea.
package app

import (
	"strings"
	"time"

	"codea/tui/internal/command"
	"codea/tui/internal/components"
	"codea/tui/internal/doctor"
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

	// pendingPrompt is the first prompt awaiting a session to be created; it is
	// sent once the session is established so the current-session filter is in
	// effect before any of that session's events arrive.
	pendingPrompt *runtime.PromptRequest

	// pendingResumeID is the session whose history is being loaded during a
	// resume. It guards against a stale load result applying to the wrong session.
	pendingResumeID runtime.SessionID

	proc              *reasoning.Processor
	reasoningActive   bool
	reasoningContent  string
	reasoningDuration time.Duration
	reasoningExpanded bool

	tools []ToolActivity

	// commandRegistry owns terminal-independent parsing/execution definitions;
	// commandPalette is only presentation/navigation state.
	commandRegistry *command.Registry
	commandPalette  commandPaletteModel

	// Task 23 runtime workspace state. Model choice is session-scoped and kept
	// entirely in Codea-owned ModelRef values.
	modelPicker   modelPickerModel
	sessionModels map[runtime.SessionID]runtime.ModelRef
	workspaceInfo WorkspaceInfo
	currentAgent  string
	doctorService *doctor.Service

	// sessionPanel is the session list/resume overlay. It owns cursor and
	// visibility; the Application feeds it Codea-domain session items.
	sessionPanel components.SessionModel

	// sessionNotice is a transient panel message (e.g. streaming blocks resume).
	sessionNotice string

	// permission is the tool-approval modal. It consumes only the Codea-domain
	// runtime.ApprovalRequest, never vendor permission DTOs.
	permission components.PermissionModel

	// approvalErr surfaces a failed ReplyApproval without silently closing the
	// modal; the user can retry, reject, or close.
	approvalErr string

	// approvalPending is true while a ReplyApproval for the currently shown
	// request is in flight. While pending, further allow/reject keys are
	// swallowed so a single approval cannot be replied to twice.
	approvalPending bool

	// skills is the skill manager driving the skills page. It is injected by the
	// composition root; nil means the skills page is unavailable.
	skills skillManager

	// skillPanel is the skills-page presentation component (cursor + display).
	skillPanel components.SkillModel

	// skillNotice is a transient skills-page message (load/toggle failure or a
	// count of skills that failed to load).
	skillNotice string

	// Task 20 pilot telemetry is metadata-only and optional. loadedSkillIDs is
	// populated from the Skill snapshot; activeMetricID links one in-flight task
	// to its metadata event. feedback is a skippable post-task prompt.
	metrics        *MetricsCollector
	feedback       FeedbackModel
	activeMetricID string
	loadedSkillIDs []string

	eventCh <-chan runtime.Event

	// streamBuf and reasoningBuf coalesce high-frequency streaming deltas so a
	// token burst does not trigger one full render per token.
	streamBuf    strings.Builder
	reasoningBuf strings.Builder

	// rendered is the cached full View output; dirty marks it stale.
	rendered string
	dirty    bool
}

// markDirty invalidates the cached View output.
func (m *Model) markDirty() { m.dirty = true }

// NewModel constructs the application model around the given Runtime.
func NewModel(client runtime.AgentRuntime) *Model {
	return &Model{
		currentPage:     PageChat,
		runtimeStatus:   runtime.RuntimeStopped,
		runtimeClient:   client,
		keys:            DefaultKeyMap(),
		messages:        make([]ChatMessage, 0),
		proc:            reasoning.NewProcessor(),
		tools:           make([]ToolActivity, 0),
		commandRegistry: defaultCommandRegistry(),
		sessionModels:   make(map[runtime.SessionID]runtime.ModelRef),
		currentAgent:    "general",
		loadedSkillIDs:  make([]string, 0),
		dirty:           true,
	}
}

// SetSkillManager injects the skill manager used by the skills page. A nil
// manager leaves the page unavailable (the page still opens but shows a notice).
func (m *Model) SetSkillManager(mgr skillManager) { m.skills = mgr }
