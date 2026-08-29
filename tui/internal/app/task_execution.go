package app

import (
	"fmt"
	"strconv"
	"strings"

	"codea/tui/internal/runtime"
)

// TaskExecutionState is Codea-owned observable execution state for one root
// user turn. Verification fields are reserved for Task 30 so the shape stays
// stable when verification/continuation is added.
type TaskExecutionState struct {
	RootTurnID        string
	PlanSeen          bool
	ActiveStep        string
	CompletedSteps    int
	TotalSteps        int
	MutationSeen      bool
	VerifyAttempts    int
	VerifyPassed      bool
	AutoContinuation int
}

func (m *Model) resetTaskExecution(turnID string) {
	m.taskExecution = TaskExecutionState{RootTurnID: strings.TrimSpace(turnID)}
}

func (m *Model) observeTaskExecutionEvent(ev runtime.Event) {
	if m.activeTurnID == "" {
		return
	}
	if m.taskExecution.RootTurnID != m.activeTurnID {
		m.resetTaskExecution(m.activeTurnID)
	}
	if ev.SessionID != "" && m.sessionID != "" && runtime.SessionID(ev.SessionID) != m.sessionID {
		return
	}
	if ev.MessageID != "" && ev.MessageID != m.taskExecution.RootTurnID {
		return
	}
	if ev.Tool == nil {
		return
	}

	tool := strings.TrimSpace(ev.Tool.Name)
	if ev.Type == eventTypeToolCalled && mutationExecutionTool(tool) {
		m.taskExecution.MutationSeen = true
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
