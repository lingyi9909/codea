package app

import (
	"fmt"
	"strconv"
	"strings"

	"codea/tui/internal/runtime"
)

// TaskExecutionState is Codea-owned observable execution state for one root
// user turn. Verification state is machine-owned: prose never sets these fields.
type TaskExecutionState struct {
	RootTurnID              string
	PlanSeen                bool
	ActiveStep              string
	CompletedSteps          int
	TotalSteps              int
	MutationSeen            bool
	VerifyAttempts          int
	VerifyPassed            bool
	LastVerificationResult  string
	LastVerificationProfile string
	AutoContinuation        int
	messageRoots            map[string]string
	verificationCalls       map[string]struct{}
	verificationCallEpoch   map[string]int
	mutationEpoch           int
}

func (m *Model) resetTaskExecution(turnID string) {
	root := strings.TrimSpace(turnID)
	m.taskExecution = TaskExecutionState{
		RootTurnID:            root,
		messageRoots:          make(map[string]string),
		verificationCalls:     make(map[string]struct{}),
		verificationCallEpoch: make(map[string]int),
	}
	if root != "" {
		m.taskExecution.messageRoots[root] = root
	}
}

func (m *Model) recordMessageRoot(messageID, rootTurnID string) {
	messageID = strings.TrimSpace(messageID)
	rootTurnID = strings.TrimSpace(rootTurnID)
	if messageID == "" || rootTurnID == "" {
		return
	}
	if m.taskExecution.messageRoots == nil {
		m.taskExecution.messageRoots = make(map[string]string)
	}
	m.taskExecution.messageRoots[messageID] = rootTurnID
}

func (m *Model) rootTurnForMessage(messageID string) string {
	messageID = strings.TrimSpace(messageID)
	if messageID == "" {
		return ""
	}
	return m.taskExecution.messageRoots[messageID]
}

func (m *Model) observeMessageRoot(ev runtime.Event) {
	if ev.Type != "message.updated" || strings.TrimSpace(ev.MessageID) == "" {
		return
	}
	if ev.SessionID != "" && m.sessionID != "" && runtime.SessionID(ev.SessionID) != m.sessionID {
		return
	}
	if strings.TrimSpace(ev.MessageRole) != "assistant" {
		return
	}
	parent := strings.TrimSpace(ev.ParentMessageID)
	if parent == "" {
		return
	}
	root := m.rootTurnForMessage(parent)
	if root == "" && parent == m.activeTurnID {
		root = parent
	}
	if root != "" {
		m.recordMessageRoot(ev.MessageID, root)
	}
}

func (m *Model) eventRootTurnID(ev runtime.Event) string {
	if strings.TrimSpace(ev.MessageID) == "" {
		return m.activeTurnID
	}
	return m.rootTurnForMessage(ev.MessageID)
}

func (m *Model) observeTaskExecutionEvent(ev runtime.Event) {
	m.observeMessageRoot(ev)
	if m.activeTurnID == "" {
		return
	}
	if m.taskExecution.RootTurnID != m.activeTurnID {
		m.resetTaskExecution(m.activeTurnID)
	}
	if ev.SessionID != "" && m.sessionID != "" && runtime.SessionID(ev.SessionID) != m.sessionID {
		return
	}
	if ev.MessageID != "" {
		root := m.eventRootTurnID(ev)
		if root == "" || root != m.taskExecution.RootTurnID {
			return
		}
	}
	if ev.Tool == nil {
		return
	}

	tool := strings.TrimSpace(ev.Tool.Name)
	if ev.Type == eventTypeToolCalled && mutationExecutionTool(tool) {
		markMutation(&m.taskExecution)
	}
	if tool == "verify_project" {
		observeVerificationToolEvent(&m.taskExecution, ev)
		return
	}
	if ev.Type != eventTypeToolSuccess {
		return
	}
	if tool != "task_plan" && tool != "task_step" && tool != "task_status" {
		return
	}
	applyTaskPlanMetadata(&m.taskExecution, ev.Tool.Metadata)
}

func mutationExecutionTool(tool string) bool {
	switch strings.TrimSpace(tool) {
	case "write", "edit", "bash", "write_test_file", "write_document", "run_project_test":
		return true
	default:
		return false
	}
}

func markMutation(state *TaskExecutionState) {
	if state == nil {
		return
	}
	state.MutationSeen = true
	state.mutationEpoch++
	// A successful verification proves only the mutation state that existed when
	// that verify_project call started. Any later mutation makes it stale.
	state.VerifyPassed = false
	state.LastVerificationResult = ""
	state.LastVerificationProfile = ""
}

func observeVerificationToolEvent(state *TaskExecutionState, ev runtime.Event) {
	if state == nil || ev.Tool == nil {
		return
	}
	callID := strings.TrimSpace(ev.Tool.CallID)
	if state.verificationCalls == nil {
		state.verificationCalls = make(map[string]struct{})
	}
	if state.verificationCallEpoch == nil {
		state.verificationCallEpoch = make(map[string]int)
	}

	if ev.Type == eventTypeToolCalled {
		if callID == "" {
			return
		}
		if _, seen := state.verificationCalls[callID]; seen {
			return
		}
		state.verificationCalls[callID] = struct{}{}
		state.verificationCallEpoch[callID] = state.mutationEpoch
		state.VerifyAttempts++
		return
	}

	if callID == "" {
		return
	}
	callEpoch, started := state.verificationCallEpoch[callID]
	if !started || callEpoch != state.mutationEpoch {
		// Replayed/stale completion or a verification that began before a later
		// mutation cannot satisfy the current root task.
		return
	}

	if ev.Type == eventTypeToolFailed {
		state.VerifyPassed = false
		state.LastVerificationResult = "error"
		state.LastVerificationProfile = ""
		return
	}
	if ev.Type != eventTypeToolSuccess || ev.Tool.Metadata["codeaVerification"] != "true" {
		return
	}

	result := strings.TrimSpace(ev.Tool.Metadata["codeaVerificationResult"])
	profile := strings.TrimSpace(ev.Tool.Metadata["codeaVerificationProfile"])
	if !validVerificationResult(result) || !validVerificationProfile(profile) {
		return
	}
	state.LastVerificationResult = result
	state.LastVerificationProfile = profile
	state.VerifyPassed = result == "pass"
}

func validVerificationResult(result string) bool {
	switch result {
	case "pass", "fail", "timeout", "not_configured", "error":
		return true
	default:
		return false
	}
}

func validVerificationProfile(profile string) bool {
	switch profile {
	case "maven", "gradle", "go", "unknown":
		return true
	default:
		return false
	}
}

func applyTaskPlanMetadata(state *TaskExecutionState, metadata map[string]string) {
	if state == nil || metadata["codeaTaskPlan"] != "true" {
		return
	}
	total, errTotal := strconv.Atoi(metadata["codeaPlanTotal"])
	completed, errCompleted := strconv.Atoi(metadata["codeaPlanCompleted"])
	if errTotal != nil || errCompleted != nil || total < 3 || total > 7 || completed < 0 || completed > total {
		return
	}
	active := strings.TrimSpace(metadata["codeaPlanActive"])
	if len(active) > 100 {
		return
	}
	state.PlanSeen = true
	state.TotalSteps = total
	state.CompletedSteps = completed
	state.ActiveStep = active
}

func (m *Model) renderTaskProgress() string {
	if m.activeTurnID == "" {
		return ""
	}
	state := m.taskExecution
	if !state.PlanSeen || state.TotalSteps < 3 || state.TotalSteps > 7 {
		return ""
	}
	if state.RootTurnID != m.activeTurnID {
		return ""
	}
	base := fmt.Sprintf("Plan · %d/%d", state.CompletedSteps, state.TotalSteps)
	if m.viewMode == ViewFocus {
		return base
	}
	base += " steps"
	if state.ActiveStep != "" {
		base += " · " + state.ActiveStep
	}
	return base
}
