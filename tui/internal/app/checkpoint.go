package app

import (
	"context"
	"fmt"
	"strings"

	"codea/tui/internal/checkpoint"

	tea "github.com/charmbracelet/bubbletea"
)

// CheckpointService is the narrow Application dependency for Task 31. The
// composition root binds it to the already-resolved project directory.
type CheckpointService interface {
	Create(context.Context, checkpoint.CreateRequest) (checkpoint.Checkpoint, error)
	List(context.Context) ([]checkpoint.Checkpoint, error)
	Restore(context.Context, string) (checkpoint.RestoreResult, error)
}

type checkpointBaselineResultMsg struct {
	intent repoPromptIntent
	value  checkpoint.Checkpoint
	err    error
}

type checkpointCreateResultMsg struct {
	kind  checkpoint.Kind
	value checkpoint.Checkpoint
	err   error
}

type checkpointListResultMsg struct {
	items []checkpoint.Checkpoint
	err   error
}

type checkpointRestoreResultMsg struct {
	value checkpoint.RestoreResult
	err   error
}

func BaselineCheckpointCmd(service CheckpointService, intent repoPromptIntent) tea.Cmd {
	return func() tea.Msg {
		cp, err := service.Create(context.Background(), checkpoint.CreateRequest{
			TaskID: intent.request.MessageID,
			TurnID: intent.request.MessageID,
			Label:  "agent turn baseline",
			Kind:   checkpoint.KindBaseline,
		})
		return checkpointBaselineResultMsg{intent: intent, value: cp, err: err}
	}
}

func CreateCheckpointCmd(service CheckpointService, req checkpoint.CreateRequest) tea.Cmd {
	return func() tea.Msg {
		cp, err := service.Create(context.Background(), req)
		return checkpointCreateResultMsg{kind: req.Kind, value: cp, err: err}
	}
}

func ListCheckpointsCmd(service CheckpointService) tea.Cmd {
	return func() tea.Msg {
		items, err := service.List(context.Background())
		return checkpointListResultMsg{items: items, err: err}
	}
}

func RestoreCheckpointCmd(service CheckpointService, id string) tea.Cmd {
	return func() tea.Msg {
		result, err := service.Restore(context.Background(), id)
		return checkpointRestoreResultMsg{value: result, err: err}
	}
}

func (m *Model) SetCheckpointService(service CheckpointService) {
	m.checkpointService = service
	m.checkpointUnavailable = ""
	if recovery, ok := service.(interface{ RecoveryGuidance() string }); ok {
		if guidance := strings.TrimSpace(recovery.RecoveryGuidance()); guidance != "" {
			m.appendInfo(guidance)
		}
	}
}

func (m *Model) SetCheckpointUnavailable(err error) {
	m.checkpointService = nil
	m.checkpointUnavailable = ""
	if err != nil {
		m.checkpointUnavailable = err.Error()
	}
}

func (m *Model) continuePromptIntent(intent repoPromptIntent) tea.Cmd {
	if m.repoContextService != nil {
		return RepoContextCmd(m.repoContextService, intent)
	}
	if m.sessionID == "" {
		req := intent.request
		m.pendingPrompt = &req
		return CreateSessionCmd(m.runtimeClient, strings.TrimSpace(intent.displayText))
	}
	return PromptCmd(m.runtimeClient, m.sessionID, intent.request)
}

func (m *Model) beginBaselineCheckpoint(intent repoPromptIntent) tea.Cmd {
	if m.checkpointService == nil {
		if detail := strings.TrimSpace(m.checkpointUnavailable); detail != "" {
			m.appendInfo("Checkpoint unavailable; continuing without baseline protection: " + detail)
		}
		return m.continuePromptIntent(intent)
	}
	m.checkpointInFlight = true
	return BaselineCheckpointCmd(m.checkpointService, intent)
}

func (m *Model) queueFinalCheckpoint() {
	if m.checkpointService == nil || m.checkpointInFlight || m.pendingFinalCheckpoint != nil {
		return
	}
	root := strings.TrimSpace(m.taskExecution.RootTurnID)
	if root == "" {
		root = strings.TrimSpace(m.activeTurnID)
	}
	req := checkpoint.CreateRequest{TaskID: root, TurnID: root, Label: "verified task final", Kind: checkpoint.KindFinal}
	m.pendingFinalCheckpoint = &req
}

func (m *Model) takePendingCheckpointCmd() tea.Cmd {
	if m.pendingFinalCheckpoint == nil || m.checkpointService == nil || m.checkpointInFlight {
		return nil
	}
	req := *m.pendingFinalCheckpoint
	m.pendingFinalCheckpoint = nil
	m.checkpointInFlight = true
	return CreateCheckpointCmd(m.checkpointService, req)
}

func (m *Model) handleCheckpointMessage(msg tea.Msg) (bool, tea.Cmd) {
	switch msg := msg.(type) {
	case checkpointBaselineResultMsg:
		m.checkpointInFlight = false
		if msg.err != nil {
			m.appendInfo("Checkpoint unavailable; continuing without baseline protection: " + msg.err.Error())
		} else {
			m.lastBaselineCheckpoint = msg.value.ID
		}
		return true, m.continuePromptIntent(msg.intent)

	case checkpointCreateResultMsg:
		m.checkpointInFlight = false
		if msg.err != nil {
			if msg.kind == checkpoint.KindFinal {
				m.appendInfo("Final checkpoint failed; verification remains accepted: " + msg.err.Error())
			} else {
				m.appendInfo("Checkpoint failed: " + msg.err.Error())
			}
			return true, m.takePendingCheckpointCmd()
		}
		if msg.kind == checkpoint.KindFinal {
			m.appendInfo("Final checkpoint created: " + msg.value.ID)
		} else {
			m.appendInfo("Checkpoint created: " + msg.value.ID)
		}
		return true, m.takePendingCheckpointCmd()

	case checkpointListResultMsg:
		m.checkpointInFlight = false
		if msg.err != nil {
			m.appendInfo("Checkpoints unavailable: " + msg.err.Error())
			return true, nil
		}
		lines := []string{"Checkpoints:"}
		for _, cp := range msg.items {
			line := fmt.Sprintf("- %s · %s", cp.ID, cp.Kind)
			if strings.TrimSpace(cp.Label) != "" {
				line += " · " + cp.Label
			}
			lines = append(lines, line)
		}
		if len(msg.items) == 0 {
			lines = append(lines, "- none")
		}
		m.appendInfo(strings.Join(lines, "\n"))
		return true, nil

	case checkpointRestoreResultMsg:
		m.checkpointInFlight = false
		// A restore can fail after partially mutating the workspace. Invalidate
		// Repo Context for every completed restore attempt so a stale map is
		// never reused after either success or interruption.
		if m.repoContextService != nil {
			m.repoContextService.Invalidate()
		}
		if msg.err != nil {
			m.appendInfo("Restore failed: " + msg.err.Error())
			return true, nil
		}
		m.appendInfo(fmt.Sprintf("Restored %s · safety %s · %d files changed", msg.value.Target.ID, msg.value.Safety.ID, msg.value.FilesChanged))
		return true, nil
	}
	return false, nil
}
