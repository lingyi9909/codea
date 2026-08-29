package app

import (
	"fmt"
	"strconv"
	"strings"

	"codea/tui/internal/runtime"

	tea "github.com/charmbracelet/bubbletea"
)

const maxVerificationContinuations = 2

type ControlPromptKind string

const (
	ControlVerifyMissing ControlPromptKind = "verify_missing"
	ControlVerifyFailed  ControlPromptKind = "verify_failed"
)

const missingVerificationControlPrompt = "Codea verification gate: project mutation was observed but no fresh successful verification exists for this task. Use the existing plan, run verify_project, and do not claim completion until machine verification passes. If verification cannot be configured, report that limitation truthfully."
const failedVerificationControlPrompt = "Codea verification gate: the latest verify_project result failed. Inspect the verification evidence already available in this session, update the current plan step as needed, make the smallest justified fix, rerun verify_project, and stop if the bounded repair attempt cannot produce PASS."

func verificationStepTriggerKey(ev runtime.Event) string {
	if id := strings.TrimSpace(ev.PartID); id != "" {
		return "part:" + id
	}
	if id := strings.TrimSpace(ev.ID); id != "" {
		return "event:" + id
	}
	if id := strings.TrimSpace(ev.MessageID); id != "" {
		return "message:" + id + ":" + strconv.FormatInt(ev.Sequence, 10)
	}
	return "sequence:" + strconv.FormatInt(ev.Sequence, 10)
}

func (m *Model) rootAgentForVerification() string {
	if m.activeTurnID != "" {
		if entry, ok := m.executionTrace.Entry("turn:" + m.activeTurnID + ":agent"); ok {
			if agent := strings.TrimSpace(entry.Title); agent != "" {
				return agent
			}
		}
	}
	return m.activeAgent("")
}

func (m *Model) nextVerificationContinuation(ev runtime.Event) *runtime.PromptRequest {
	key := verificationStepTriggerKey(ev)
	if m.verificationContinuationTriggers == nil {
		m.verificationContinuationTriggers = make(map[string]struct{})
	}
	if _, replay := m.verificationContinuationTriggers[key]; replay {
		return nil
	}
	m.verificationContinuationTriggers[key] = struct{}{}

	decision := verificationDecision(m.taskExecution)
	if decision == VerifyNotRequired || decision == VerifyAccepted {
		m.finishStepWithVerification()
		return nil
	}
	if m.taskExecution.AutoContinuation >= maxVerificationContinuations {
		m.finishStepWithVerification()
		return nil
	}

	m.taskExecution.AutoContinuation++
	attempt := m.taskExecution.AutoContinuation
	root := strings.TrimSpace(m.taskExecution.RootTurnID)
	if root == "" {
		root = strings.TrimSpace(m.activeTurnID)
	}
	messageID := fmt.Sprintf("codea-verification-%s-%d", root, attempt)
	text := missingVerificationControlPrompt
	if decision == VerifyFailed {
		text = failedVerificationControlPrompt
	}

	req := runtime.PromptRequest{
		MessageID: messageID,
		Agent:     m.rootAgentForVerification(),
		Parts: []runtime.PromptPart{runtime.TextPart{
			Text:      text,
			Synthetic: true,
			Metadata: map[string]any{
				"codea.kind":     "verification-control",
				"codea.rootTurn": root,
				"codea.attempt":  attempt,
			},
		}},
	}
	if selected, ok := m.sessionModels[m.sessionID]; ok {
		model := selected
		req.Model = &model
	}
	m.recordMessageRoot(messageID, root)
	return &req
}

// handleVerificationStepFinished is the direct, testable control path. The
// Bubble Tea integration queues the same request and returns it as a Cmd from
// Update, so Runtime.Prompt is never invoked synchronously inside Update.
func (m *Model) handleVerificationStepFinished(ev runtime.Event) tea.Cmd {
	req := m.nextVerificationContinuation(ev)
	if req == nil {
		return nil
	}
	return PromptCmd(m.runtimeClient, m.sessionID, *req)
}

func (m *Model) queueVerificationStepFinished(ev runtime.Event) {
	if req := m.nextVerificationContinuation(ev); req != nil {
		m.pendingVerificationPrompt = req
	}
}

func (m *Model) takeVerificationContinuationCmd() tea.Cmd {
	if m.pendingVerificationPrompt == nil {
		return nil
	}
	req := *m.pendingVerificationPrompt
	m.pendingVerificationPrompt = nil
	return PromptCmd(m.runtimeClient, m.sessionID, req)
}
