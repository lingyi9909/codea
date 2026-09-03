package app

import (
	"fmt"
	"strings"

	"codea/tui/internal/components"
	"codea/tui/internal/reasoning"
	"codea/tui/internal/runtime"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
)

// Init starts the runtime subscription and the merge-refresh ticker. When a
// Skill manager is available, it also refreshes the metadata-only loaded Skill
// IDs used by Task 20 metrics; the result does not open the Skills page.
func (m *Model) Init() tea.Cmd {
	cmds := []tea.Cmd{SubscribeEvents(m.runtimeClient), TickCmd()}
	if m.skills != nil {
		cmds = append(cmds, ListSkillsCmd(m.skills))
	}
	return tea.Batch(cmds...)
}

// Update handles Bubble Tea messages: subscription lifecycle, key input,
// prompt submission, and streaming runtime event processing.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if handled, cmd := m.handleRuntimeWorkspaceMessage(msg); handled {
		return m, cmd
	}

	switch msg := msg.(type) {
	case subscribedMsg:
		m.eventCh = msg.ch
		m.runtimeStatus = runtime.RuntimeHealthy
		m.markDirty()
		return m, waitForEvent(msg.ch)

	case runtimeEventMsg:
		if m.processRuntimeEvent(msg.ev) {
			m.markDirty()
		}
		controlCmd := m.takeVerificationContinuationCmd()
		if m.eventCh != nil && controlCmd != nil {
			return m, tea.Batch(waitForEvent(m.eventCh), controlCmd)
		}
		if m.eventCh != nil {
			return m, waitForEvent(m.eventCh)
		}
		if controlCmd != nil {
			return m, controlCmd
		}
		return m, nil

	case promptResultMsg:
		if msg.err != nil {
			m.finishActiveTurnTrace(TraceFailed)
			m.finishStreamingWithOutcome(MetricStatusFailed, "prompt_error", false)
			m.markDirty()
			return m, nil
		}
		if msg.sessionID != "" {
			m.sessionID = msg.sessionID
		}
		return m, nil

	case sessionCreatedMsg:
		if msg.err != nil {
			m.pendingPrompt = nil
			m.finishActiveTurnTrace(TraceFailed)
			m.finishStreamingWithOutcome(MetricStatusFailed, "session_create_error", false)
			m.markDirty()
			return m, nil
		}
		m.sessionID = msg.sessionID
		m.bindActiveTraceSession(msg.sessionID)
		req := m.pendingPrompt
		m.pendingPrompt = nil
		if req == nil {
			return m, nil
		}
		return m, PromptCmd(m.runtimeClient, m.sessionID, *req)

	case listSessionsResultMsg:
		if msg.err != nil {
			m.sessionNotice = "Failed to load sessions: " + msg.err.Error()
			m.markDirty()
			return m, nil
		}
		m.sessionPanel.Open(sessionItems(msg.sessions))
		m.sessionPanel.SetActive(m.sessionID)
		m.sessionNotice = ""
		m.markDirty()
		return m, nil

	case listAgentsResultMsg:
		if msg.err != nil {
			m.appendInfo("Failed to load agents: " + msg.err.Error())
			return m, nil
		}
		lines := []string{"Agents:"}
		for _, agent := range msg.agents {
			label := "- " + agent.Name
			if strings.TrimSpace(agent.Mode) != "" {
				label += " (" + agent.Mode + ")"
			}
			lines = append(lines, label)
		}
		if len(msg.agents) == 0 {
			lines = append(lines, "- none")
		}
		m.appendInfo(strings.Join(lines, "\n"))
		return m, nil

	case runtimeHealthResultMsg:
		if msg.err != nil {
			m.appendInfo("Doctor quick check: FAIL\n" + msg.err.Error())
			return m, nil
		}
		result := "FAIL"
		if msg.health.Healthy {
			result = "PASS"
		}
		m.appendInfo(fmt.Sprintf("Doctor quick check: %s\nRuntime version: %s", result, msg.health.Version))
		return m, nil

	case cancelResponseResultMsg:
		if msg.sessionID != m.sessionID {
			return m, nil
		}
		if msg.err != nil {
			m.appendInfo("Cancel failed: " + msg.err.Error())
			return m, nil
		}
		if m.isStreaming {
			m.finishActiveTurnTrace(TraceFailed)
			m.finishStreamingWithOutcome(MetricStatusFailed, "cancelled", false)
		}
		m.appendInfo("Cancelled current response.")
		return m, nil

	case loadHistoryResultMsg:
		if m.pendingResumeID == "" || msg.sessionID != m.pendingResumeID {
			return m, nil
		}
		m.pendingResumeID = ""
		if msg.err != nil {
			m.sessionNotice = "Failed to load session history: " + msg.err.Error()
			m.markDirty()
			return m, nil
		}
		m.resumeSession(msg.sessionID, msg.messages)
		m.sessionPanel.Close()
		m.markDirty()
		return m, nil

	case approvalResultMsg:
		if m.permission.Request == nil || runtime.ApprovalID(m.permission.Request.ID) != msg.approvalID {
			return m, nil
		}
		m.approvalPending = false
		if msg.err != nil {
			m.approvalErr = msg.err.Error()
			m.markDirty()
			return m, nil
		}
		if m.activeApprovalTraceKey != "" {
			status := TraceSuccess
			if m.pendingApprovalDecision == runtime.ApprovalReject {
				status = TraceDenied
			}
			m.executionTrace.setStatus(m.activeApprovalTraceKey, status, true)
		}
		if m.activeTurnID != "" {
			m.executionTrace.setStatus("turn:"+m.activeTurnID+":working", TraceRunning, false)
		}
		m.activeApprovalTraceKey = ""
		m.pendingApprovalDecision = ""
		m.permission = components.PermissionModel{}
		m.approvalErr = ""
		m.markDirty()
		return m, nil

	case listSkillsResultMsg:
		return m, m.handleSkillListResult(msg)

	case setSkillResultMsg:
		return m, m.handleSkillSetResult(msg)

	case subscribeErrMsg:
		m.runtimeStatus = runtime.RuntimeCrashed
		m.finishActiveTurnTrace(TraceFailed)
		m.finishStreamingWithOutcome(MetricStatusFailed, "subscription_error", false)
		m.markDirty()
		return m, nil

	case eventStreamClosedMsg:
		m.finishActiveTurnTrace(TraceFailed)
		m.completeTaskMetric(MetricStatusFailed, "event_stream_closed", false)
		if m.runtimeStatus != runtime.RuntimeCrashed {
			m.runtimeStatus = runtime.RuntimeStopped
			m.markDirty()
		}
		return m, nil

	case tickMsg:
		if m.flushStreaming() {
			m.markDirty()
		}
		return m, TickCmd()

	case tea.KeyMsg:
		m.markDirty()
		return m, m.handleKey(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.markDirty()
		return m, nil
	}

	return m, nil
}

// handleKey routes keypresses. Quit always works; otherwise modal state takes
// priority over chat keys so typing/Enter/shortcuts cannot leak through.
func (m *Model) handleKey(msg tea.KeyMsg) tea.Cmd {
	if key.Matches(msg, m.keys.Quit) {
		m.completeTaskMetric(MetricStatusFailed, "user_quit", false)
		return tea.Quit
	}
	if m.permission.Visible() {
		return m.handleApprovalKey(msg)
	}
	if m.feedback.Visible() {
		eventID := m.feedback.EventID
		choice, handled := m.feedback.HandleKey(msg)
		if handled && choice != FeedbackSkip && m.metrics != nil {
			_ = m.metrics.RecordFeedback(eventID, choice)
		}
		return nil
	}
	if m.modelPicker.Visible {
		return m.handleModelPickerKey(msg)
	}
	if m.commandPalette.Visible {
		return m.handleCommandPaletteKey(msg)
	}
	if m.sessionPanel.Visible {
		return m.handleSessionKey(msg)
	}
	if m.currentPage == PageSkills {
		return m.handleSkillKey(msg)
	}
	switch {
	case key.Matches(msg, m.keys.Submit):
		return m.submit()
	case key.Matches(msg, m.keys.Newline):
		m.input += "\n"
		m.refreshCommandPalette()
		return nil
	case key.Matches(msg, m.keys.Sessions):
		return m.toggleSessions()
	case key.Matches(msg, m.keys.Skills):
		return m.toggleSkills()
	case key.Matches(msg, m.keys.ClearScreen):
		m.clearChat()
		return nil
	case key.Matches(msg, m.keys.ToggleThink):
		m.reasoningExpanded = !m.reasoningExpanded
		return nil
	default:
		m.handleTyping(msg)
		return nil
	}
}

func (m *Model) handleApprovalKey(msg tea.KeyMsg) tea.Cmd {
	if m.approvalPending {
		return nil
	}
	switch {
	case key.Matches(msg, m.keys.AllowOnce):
		return m.replyApproval(runtime.ApprovalOnce)
	case key.Matches(msg, m.keys.AllowAlways):
		return m.replyApproval(runtime.ApprovalAlways)
	case key.Matches(msg, m.keys.Reject), key.Matches(msg, m.keys.Esc):
		return m.replyApproval(runtime.ApprovalReject)
	}
	return nil
}

func (m *Model) replyApproval(decision runtime.ApprovalDecision) tea.Cmd {
	if m.permission.Request == nil {
		return nil
	}
	m.approvalPending = true
	m.pendingApprovalDecision = decision
	return ReplyApprovalCmd(m.runtimeClient, runtime.ApprovalID(m.permission.Request.ID), runtime.ApprovalReply{Decision: decision})
}

func (m *Model) handleSessionKey(msg tea.KeyMsg) tea.Cmd {
	switch {
	case key.Matches(msg, m.keys.Up):
		m.sessionPanel.MoveUp()
		m.sessionNotice = ""
		return nil
	case key.Matches(msg, m.keys.Down):
		m.sessionPanel.MoveDown()
		m.sessionNotice = ""
		return nil
	case key.Matches(msg, m.keys.Submit):
		return m.resumeSelectedSession()
	case key.Matches(msg, m.keys.Esc), key.Matches(msg, m.keys.Sessions):
		m.sessionPanel.Close()
		m.sessionNotice = ""
		return nil
	}
	return nil
}

func (m *Model) toggleSessions() tea.Cmd {
	if m.sessionPanel.Visible {
		m.sessionPanel.Close()
		m.sessionNotice = ""
		return nil
	}
	m.sessionPanel.Visible = true
	m.sessionPanel.Items = nil
	m.sessionPanel.Cursor = 0
	m.sessionNotice = ""
	return ListSessionsCmd(m.runtimeClient)
}

func (m *Model) resumeSelectedSession() tea.Cmd {
	item, ok := m.sessionPanel.Selected()
	if !ok || item.Active {
		m.sessionPanel.Close()
		m.sessionNotice = ""
		return nil
	}
	if m.isStreaming {
		m.sessionNotice = "Finish or cancel the current response before switching sessions."
		return nil
	}
	m.pendingResumeID = item.ID
	m.sessionNotice = ""
	return LoadSessionHistoryCmd(m.runtimeClient, item.ID)
}

func (m *Model) resumeSession(id runtime.SessionID, history []runtime.Message) {
	m.sessionID = id
	m.isStreaming = false
	m.pendingPrompt = nil
	m.modelPicker.Close()
	m.streamBuf.Reset()
	m.reasoningBuf.Reset()
	m.reasoningActive = false
	m.reasoningContent = ""
	m.reasoningDuration = 0
	m.reasoningExpanded = false
	m.tools = make([]ToolActivity, 0)
	m.executionTrace.Reset()
	m.activeTurnID = ""
	m.activeApprovalTraceKey = ""
	m.pendingApprovalDecision = ""
	m.proc.Reset()
	m.messages = historyToChatMessages(history)
	m.sessionNotice = ""
	m.currentAgent = "general"
}

func historyToChatMessages(history []runtime.Message) []ChatMessage {
	out := make([]ChatMessage, 0, len(history))
	for _, msg := range history {
		out = append(out, ChatMessage{
			Role:     messageRole(msg.Role),
			Content:  msg.Content,
			Finished: true,
		})
	}
	return out
}

func messageRole(role string) Role {
	switch role {
	case "user":
		return RoleUser
	case "assistant":
		return RoleAssistant
	default:
		return RoleInfo
	}
}

func sessionItems(sessions []runtime.Session) []components.SessionItem {
	items := make([]components.SessionItem, len(sessions))
	for i, s := range sessions {
		items[i] = components.SessionItem{
			ID:        runtime.SessionID(s.ID),
			Title:     s.Title,
			UpdatedAt: s.UpdatedAt,
		}
	}
	return items
}

func (m *Model) handleTyping(msg tea.KeyMsg) {
	switch msg.Type {
	case tea.KeyBackspace:
		m.input = deleteLastRune(m.input)
	case tea.KeyRunes:
		m.input += string(msg.Runes)
	case tea.KeySpace:
		m.input += " "
	}
	m.refreshCommandPalette()
}

func (m *Model) submit() tea.Cmd {
	if strings.TrimSpace(m.input) == "" {
		return nil
	}
	raw := m.input
	if strings.HasPrefix(raw, "/") {
		return m.submitCommand(raw)
	}
	if m.isStreaming {
		return nil
	}
	return m.startPrompt(raw, raw, "general")
}

func (m *Model) processRuntimeEvent(ev runtime.Event) bool {
	if !m.acceptsEvent(ev) {
		return false
	}
	if ev.Type == eventTypeStepFinished {
		currentRoot := strings.TrimSpace(m.taskExecution.RootTurnID)
		if currentRoot == "" || currentRoot != strings.TrimSpace(m.activeTurnID) || strings.TrimSpace(m.eventRootTurnID(ev)) != currentRoot {
			return false
		}
	}
	m.traceRuntimeEvent(ev)
	dirty := false
	switch ev.Type {
	case eventTypeStepFinished:
		dirty = m.applyReasoningEvents(m.proc.Flush()) || dirty
		m.queueVerificationStepFinished(ev)
		dirty = true
	case eventTypeSessionError:
		dirty = m.applyReasoningEvents(m.proc.Process(ev)) || dirty
		m.finishStreamingWithOutcome(MetricStatusFailed, "session_error", false)
		dirty = true
	case eventTypeRuntimeError:
		dirty = m.applyReasoningEvents(m.proc.Process(ev)) || dirty
		m.finishStreamingWithOutcome(MetricStatusFailed, "runtime_error", false)
		dirty = true
	case eventTypeToolCalled:
		m.addTool(ev)
		dirty = true
	case eventTypeToolSuccess:
		m.updateTool(ev, ToolSuccess)
		dirty = true
	case eventTypeToolFailed:
		m.updateTool(ev, ToolFailed)
		dirty = true
	case eventTypeApprovalRequested:
		if ev.Approval != nil {
			m.permission = components.NewPermissionModel(ev.Approval)
			m.approvalErr = ""
			m.approvalPending = false
			dirty = true
		}
	default:
		dirty = m.applyReasoningEvents(m.proc.Process(ev)) || dirty
	}
	return dirty
}

func (m *Model) acceptsEvent(ev runtime.Event) bool {
	if ev.SessionID == "" {
		return true
	}
	return ev.SessionID == string(m.sessionID)
}

func (m *Model) applyReasoningEvents(events []reasoning.Event) bool {
	dirty := false
	for _, pe := range events {
		switch pe.Kind {
		case reasoning.EventAnswerDelta:
			m.streamBuf.WriteString(pe.Content)
		case reasoning.EventReasoningStart:
			m.reasoningActive = true
			m.reasoningContent = ""
			m.reasoningBuf.Reset()
			dirty = true
		case reasoning.EventReasoningDelta:
			m.reasoningBuf.WriteString(pe.Content)
		case reasoning.EventReasoningEnd:
			m.reasoningActive = false
			m.reasoningDuration = pe.Duration
			m.reasoningExpanded = false
			dirty = true
		}
	}
	return dirty
}

func (m *Model) flushStreaming() bool {
	dirty := false
	if m.streamBuf.Len() > 0 {
		m.appendAnswer(m.streamBuf.String())
		m.streamBuf.Reset()
		dirty = true
	}
	if m.reasoningBuf.Len() > 0 {
		m.reasoningContent += m.reasoningBuf.String()
		m.reasoningBuf.Reset()
		dirty = true
	}
	return dirty
}

func (m *Model) appendAnswer(content string) {
	if content == "" {
		return
	}
	for i := len(m.messages) - 1; i >= 0; i-- {
		if m.messages[i].Role == RoleAssistant && !m.messages[i].Finished {
			m.messages[i].Content += content
			return
		}
	}
	m.messages = append(m.messages, ChatMessage{Role: RoleAssistant, Content: content})
}

func (m *Model) finishStreaming() {
	m.finishStreamingWithOutcome(MetricStatusCompleted, "", true)
}

func (m *Model) finishStreamingWithOutcome(status MetricStatus, errorCategory string, requestFeedback bool) {
	m.flushStreaming()
	m.isStreaming = false
	for i := len(m.messages) - 1; i >= 0; i-- {
		if m.messages[i].Role == RoleAssistant && !m.messages[i].Finished {
			m.messages[i].Finished = true
			break
		}
	}
	m.completeTaskMetric(status, errorCategory, requestFeedback)
}

func deleteLastRune(s string) string {
	r := []rune(s)
	if len(r) == 0 {
		return s
	}
	return string(r[:len(r)-1])
}

func (m *Model) addTool(ev runtime.Event) {
	if ev.Tool == nil {
		return
	}
	for i := range m.tools {
		if m.tools[i].CallID == ev.Tool.CallID {
			if ev.Tool.Name != "" {
				m.tools[i].Name = ev.Tool.Name
			}
			m.tools[i].Status = ToolRunning
			return
		}
	}
	m.tools = append(m.tools, ToolActivity{Name: ev.Tool.Name, CallID: ev.Tool.CallID, Status: ToolRunning})
}

func (m *Model) updateTool(ev runtime.Event, status ToolStatus) {
	if ev.Tool == nil {
		return
	}
	for i := range m.tools {
		if m.tools[i].CallID == ev.Tool.CallID {
			m.tools[i].Status = status
			return
		}
	}
}

func (m *Model) clearChat() {
	m.messages = make([]ChatMessage, 0)
	m.tools = make([]ToolActivity, 0)
	m.executionTrace.Reset()
	m.activeTurnID = ""
	m.activeApprovalTraceKey = ""
	m.pendingApprovalDecision = ""
	m.reasoningActive = false
	m.reasoningContent = ""
	m.reasoningExpanded = false
	m.reasoningDuration = 0
	m.streamBuf.Reset()
	m.reasoningBuf.Reset()
}
