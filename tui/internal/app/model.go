// Package app hosts the Bubble Tea top-level Model for the Codea TUI.
//
// Dependency rule: this package depends only on Codea-owned application/domain
// packages and the Bubble Tea stack. It must never import the OpenCode vendor
// layer or the supervisor — those are wired together in cmd/codea.
package app

import (
	"strings"
	"time"

	"codea/tui/internal/checkpoint"
	"codea/tui/internal/command"
	"codea/tui/internal/components"
	"codea/tui/internal/doctor"
	"codea/tui/internal/modelprofile"
	"codea/tui/internal/reasoning"
	"codea/tui/internal/runtime"
)

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleInfo      Role = "info"
)

type ChatMessage struct {
	Role     Role
	Content  string
	Finished bool
	TurnID   string
	Agent    string
	Model    string
}

type ToolStatus string

const (
	ToolRunning ToolStatus = "running"
	ToolSuccess ToolStatus = "success"
	ToolFailed  ToolStatus = "failed"
)

type ToolActivity struct {
	Name   string
	CallID string
	Status ToolStatus
}

type ViewMode string

const (
	ViewNormal  ViewMode = "normal"
	ViewVerbose ViewMode = "verbose"
	ViewFocus   ViewMode = "focus"
)

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

	pendingPrompt *runtime.PromptRequest

	pendingVerificationPrompt        *runtime.PromptRequest
	verificationContinuationTriggers map[string]struct{}

	checkpointService      CheckpointService
	checkpointUnavailable  string
	checkpointInFlight     bool
	pendingFinalCheckpoint *checkpoint.CreateRequest
	lastBaselineCheckpoint string

	pendingResumeID runtime.SessionID

	proc              *reasoning.Processor
	reasoningActive   bool
	reasoningContent  string
	reasoningDuration time.Duration
	reasoningExpanded bool

	tools []ToolActivity

	executionTrace          executionTrace
	taskExecution           TaskExecutionState
	viewMode                ViewMode
	activeTurnID            string
	activeApprovalTraceKey  string
	pendingApprovalDecision runtime.ApprovalDecision
	spinnerFrame            int

	commandRegistry *command.Registry
	commandPalette  commandPaletteModel

	modelPicker        modelPickerModel
	agentPicker        agentPickerModel
	sessionModels      map[runtime.SessionID]runtime.ModelRef
	workspaceInfo      WorkspaceInfo
	currentAgent       string
	doctorService      *doctor.Service
	repoContextService RepoContextService

	// Task 32 model qualification is local application state. Profiles contain
	// only bounded capability metadata; raw provider output never enters them.
	modelProfileStore *modelprofile.Store
	modelCheck        ModelCheckState
	runtimeModels     []runtime.Model

	sessionPanel components.SessionModel
	sessionNotice string
	permission components.PermissionModel
	approvalErr string
	approvalPending bool

	skills skillManager
	skillPanel components.SkillModel
	skillNotice string

	metrics        *MetricsCollector
	feedback       FeedbackModel
	activeMetricID string
	loadedSkillIDs []string

	eventCh <-chan runtime.Event

	streamBuf    strings.Builder
	reasoningBuf strings.Builder

	rendered string
	dirty    bool
}

func (m *Model) markDirty() { m.dirty = true }

func NewModel(client runtime.AgentRuntime) *Model {
	return &Model{
		currentPage:     PageChat,
		runtimeStatus:   runtime.RuntimeStopped,
		runtimeClient:   client,
		keys:            DefaultKeyMap(),
		messages:        make([]ChatMessage, 0),
		proc:            reasoning.NewProcessor(),
		tools:           make([]ToolActivity, 0),
		executionTrace:  newExecutionTrace(),
		viewMode:        ViewNormal,
		commandRegistry: defaultCommandRegistry(),
		sessionModels:   make(map[runtime.SessionID]runtime.ModelRef),
		currentAgent:    "general",
		loadedSkillIDs:  make([]string, 0),
		dirty:           true,
	}
}

func (m *Model) SetSkillManager(mgr skillManager) { m.skills = mgr }
